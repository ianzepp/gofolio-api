package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

type compareMode string

const (
	compareStatus compareMode = "status"
	compareJSON   compareMode = "json"
)

type testCase struct {
	Name                     string            `json:"name"`
	Method                   string            `json:"method"`
	Path                     string            `json:"path"`
	Mode                     compareMode       `json:"mode"`
	Headers                  map[string]string `json:"headers"`
	Auth                     string            `json:"auth"` // "", "jwt", "api-key"
	RequiredEnv              []string          `json:"requiredEnv"`
	Body                     json.RawMessage   `json:"body"`
	IgnoreJSONPaths          []string          `json:"ignoreJsonPaths"`
	FlexibleArrayPath        string            `json:"flexibleArrayPath"`
	FlexibleArrayLengthDelta int               `json:"flexibleArrayLengthDelta"`
	FlexibleArrayItemShape   bool              `json:"flexibleArrayEnforceItemShape"`
}

type response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

func TestDifferentialAPI(t *testing.T) {
	legacyBaseURL := strings.TrimRight(os.Getenv("LEGACY_BASE_URL"), "/")
	goBaseURL := strings.TrimRight(os.Getenv("GO_BASE_URL"), "/")

	if legacyBaseURL == "" {
		legacyBaseURL = "http://localhost:3333"
	}
	if goBaseURL == "" {
		goBaseURL = "http://localhost:3334"
	}

	// Legacy can occasionally take longer than 20s under cold-cache/seeded load.
	client := &http.Client{Timeout: 60 * time.Second}
	if !isReachable(client, legacyBaseURL+"/api/v1/health") || !isReachable(client, goBaseURL+"/api/v1/health") {
		t.Skipf("differential APIs not reachable (legacy=%s go=%s)", legacyBaseURL, goBaseURL)
	}

	cases, err := loadCases()
	if err != nil {
		t.Fatalf("load cases: %v", err)
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			if missing, ok := missingRequiredEnv(tc); ok {
				t.Skipf("skipping case %q: missing env %s", tc.Name, missing)
			}

			legacyResp, err := doRequest(client, legacyBaseURL, tc)
			if err != nil {
				t.Fatalf("legacy request failed: %v", err)
			}

			goResp, err := doRequest(client, goBaseURL, tc)
			if err != nil {
				t.Fatalf("go request failed: %v", err)
			}

			if legacyResp.StatusCode != goResp.StatusCode {
				t.Fatalf(
					"status mismatch for %s %s: legacy=%d go=%d",
					tc.Method,
					tc.Path,
					legacyResp.StatusCode,
					goResp.StatusCode,
				)
			}

			if tc.Mode == compareJSON {
				assertJSONEqual(t, tc, legacyResp.Body, goResp.Body)
			}
		})
	}
}

func loadCases() ([]testCase, error) {
	files, err := filepath.Glob(filepath.Join("testdata", "*.json"))
	if err != nil {
		return nil, fmt.Errorf("glob testdata: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no JSON fixture files found in testdata/")
	}

	var all []testCase
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", f, err)
		}

		var cases []testCase
		if err := json.Unmarshal(b, &cases); err != nil {
			return nil, fmt.Errorf("parse %s: %w", f, err)
		}

		for i := range cases {
			if cases[i].Mode == "" {
				cases[i].Mode = compareStatus
			}
			if cases[i].Method == "" || cases[i].Path == "" || cases[i].Name == "" {
				return nil, fmt.Errorf("invalid case at index %d in %s: name/method/path required", i, f)
			}
		}

		all = append(all, cases...)
	}

	return all, nil
}

func doRequest(client *http.Client, baseURL string, tc testCase) (response, error) {
	var bodyReader io.Reader
	if len(tc.Body) > 0 {
		bodyReader = bytes.NewReader(expandEnv(tc.Body))
	}

	req, err := http.NewRequest(tc.Method, baseURL+expandEnvString(tc.Path), bodyReader)
	if err != nil {
		return response{}, err
	}

	if len(tc.Body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range tc.Headers {
		req.Header.Set(k, expandEnvString(v))
	}

	if err := applyAuth(req, tc.Auth, tc.Name); err != nil {
		return response{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return response{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response{}, err
	}

	return response{
		StatusCode: resp.StatusCode,
		Body:       body,
		Header:     resp.Header.Clone(),
	}, nil
}

func applyAuth(req *http.Request, authMode string, caseName string) error {
	if authMode == "" {
		return nil
	}

	switch {
	case authMode == "jwt":
		token := os.Getenv("JWT_TOKEN")
		if token == "" {
			return fmt.Errorf("case %q requires JWT_TOKEN", caseName)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	case strings.HasPrefix(authMode, "jwt:"):
		envName := strings.TrimSpace(strings.TrimPrefix(authMode, "jwt:"))
		if envName == "" {
			return fmt.Errorf("case %q has invalid auth mode %q", caseName, authMode)
		}
		token := os.Getenv(envName)
		if token == "" {
			return fmt.Errorf("case %q requires %s", caseName, envName)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	case authMode == "api-key":
		key := os.Getenv("API_KEY")
		if key == "" {
			return fmt.Errorf("case %q requires API_KEY", caseName)
		}
		req.Header.Set("Authorization", "Api-Key "+key)
		return nil
	case strings.HasPrefix(authMode, "api-key:"):
		envName := strings.TrimSpace(strings.TrimPrefix(authMode, "api-key:"))
		if envName == "" {
			return fmt.Errorf("case %q has invalid auth mode %q", caseName, authMode)
		}
		key := os.Getenv(envName)
		if key == "" {
			return fmt.Errorf("case %q requires %s", caseName, envName)
		}
		req.Header.Set("Authorization", "Api-Key "+key)
		return nil
	default:
		return fmt.Errorf("case %q has unsupported auth mode %q", caseName, authMode)
	}
}

func missingRequiredEnv(tc testCase) (string, bool) {
	for _, envName := range tc.RequiredEnv {
		name := strings.TrimSpace(envName)
		if name == "" {
			continue
		}
		if os.Getenv(name) == "" {
			return name, true
		}
	}

	if authEnvName := requiredEnvForAuth(tc.Auth); authEnvName != "" && os.Getenv(authEnvName) == "" {
		return authEnvName, true
	}

	return "", false
}

func requiredEnvForAuth(authMode string) string {
	if authMode == "" {
		return ""
	}

	switch {
	case authMode == "jwt":
		return "JWT_TOKEN"
	case strings.HasPrefix(authMode, "jwt:"):
		return strings.TrimSpace(strings.TrimPrefix(authMode, "jwt:"))
	case authMode == "api-key":
		return "API_KEY"
	case strings.HasPrefix(authMode, "api-key:"):
		return strings.TrimSpace(strings.TrimPrefix(authMode, "api-key:"))
	default:
		return ""
	}
}

func isReachable(client *http.Client, url string) bool {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return true
}

func expandEnvString(s string) string {
	if s == "" {
		return s
	}
	return os.ExpandEnv(s)
}

func expandEnv(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	return []byte(expandEnvString(string(b)))
}

func assertJSONEqual(t *testing.T, tc testCase, legacyBody, goBody []byte) {
	t.Helper()

	legacyValue, err := decodeJSON(legacyBody)
	if err != nil {
		t.Fatalf("legacy response is not valid JSON: %v\nbody=%s", err, string(legacyBody))
	}

	goValue, err := decodeJSON(goBody)
	if err != nil {
		t.Fatalf("go response is not valid JSON: %v\nbody=%s", err, string(goBody))
	}

	for _, path := range tc.IgnoreJSONPaths {
		parts := strings.Split(path, ".")
		legacyValue = removeJSONPath(legacyValue, parts)
		goValue = removeJSONPath(goValue, parts)
	}

	if tc.FlexibleArrayPath != "" {
		parts := strings.Split(tc.FlexibleArrayPath, ".")
		if err := assertFlexibleArrayEqual(legacyValue, goValue, parts, tc.FlexibleArrayLengthDelta, tc.FlexibleArrayItemShape); err != nil {
			t.Fatalf(
				"JSON mismatch for %s %s (flexible array rule)\nerror=%v",
				tc.Method,
				tc.Path,
				err,
			)
		}
		// Compare remaining JSON strictly after removing the flexible array path.
		legacyValue = removeJSONPath(legacyValue, parts)
		goValue = removeJSONPath(goValue, parts)
	}

	legacyCanonical, err := canonicalJSON(legacyValue)
	if err != nil {
		t.Fatalf("canonicalize legacy JSON: %v", err)
	}

	goCanonical, err := canonicalJSON(goValue)
	if err != nil {
		t.Fatalf("canonicalize go JSON: %v", err)
	}

	if !bytes.Equal(legacyCanonical, goCanonical) {
		t.Fatalf(
			"JSON mismatch for %s %s\nlegacy=%s\ngo=%s",
			tc.Method,
			tc.Path,
			legacyCanonical,
			goCanonical,
		)
	}
}

func decodeJSON(body []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()

	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}

	return v, nil
}

func canonicalJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func removeJSONPath(v any, parts []string) any {
	if len(parts) == 0 {
		return v
	}

	switch cur := v.(type) {
	case map[string]any:
		key := parts[0]
		if len(parts) == 1 {
			delete(cur, key)
			return cur
		}

		next, ok := cur[key]
		if !ok {
			return cur
		}
		cur[key] = removeJSONPath(next, parts[1:])
		return cur

	case []any:
		key := parts[0]
		if key == "*" {
			for i := range cur {
				cur[i] = removeJSONPath(cur[i], parts[1:])
			}
			return cur
		}

		index, err := strconv.Atoi(key)
		if err != nil || index < 0 || index >= len(cur) {
			return cur
		}

		cur[index] = removeJSONPath(cur[index], parts[1:])
		return cur

	default:
		return v
	}
}

func assertFlexibleArrayEqual(legacyRoot, goRoot any, pathParts []string, allowedDelta int, enforceItemShape bool) error {
	legacyTarget, ok := getJSONPath(legacyRoot, pathParts)
	if !ok {
		return fmt.Errorf("legacy missing path %q", strings.Join(pathParts, "."))
	}
	goTarget, ok := getJSONPath(goRoot, pathParts)
	if !ok {
		return fmt.Errorf("go missing path %q", strings.Join(pathParts, "."))
	}

	legacyArr, ok := legacyTarget.([]any)
	if !ok {
		return fmt.Errorf("legacy path %q is not an array", strings.Join(pathParts, "."))
	}
	goArr, ok := goTarget.([]any)
	if !ok {
		return fmt.Errorf("go path %q is not an array", strings.Join(pathParts, "."))
	}

	if absInt(len(legacyArr)-len(goArr)) > allowedDelta {
		return fmt.Errorf(
			"array length difference exceeds allowed delta at %q (legacy=%d go=%d allowed=%d)",
			strings.Join(pathParts, "."),
			len(legacyArr),
			len(goArr),
			allowedDelta,
		)
	}

	if enforceItemShape {
		legacyShapes := collectArrayItemShapes(legacyArr)
		goShapes := collectArrayItemShapes(goArr)
		if !equalStringSlices(legacyShapes, goShapes) {
			return fmt.Errorf(
				"array item shape mismatch at %q (legacyShapes=%v goShapes=%v)",
				strings.Join(pathParts, "."),
				legacyShapes,
				goShapes,
			)
		}
	}

	return nil
}

func getJSONPath(v any, parts []string) (any, bool) {
	cur := v
	for _, part := range parts {
		switch typed := cur.(type) {
		case map[string]any:
			next, ok := typed[part]
			if !ok {
				return nil, false
			}
			cur = next
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(typed) {
				return nil, false
			}
			cur = typed[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

func collectArrayItemShapes(items []any) []string {
	uniq := map[string]struct{}{}
	for _, item := range items {
		uniq[shapeSignature(item)] = struct{}{}
	}
	result := make([]string, 0, len(uniq))
	for sig := range uniq {
		result = append(result, sig)
	}
	sort.Strings(result)
	return result
}

func shapeSignature(v any) string {
	shape := buildShape(v)
	b, err := json.Marshal(shape)
	if err != nil {
		return fmt.Sprintf("shape-marshal-error:%T", v)
	}
	return string(b)
}

func buildShape(v any) any {
	switch cur := v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case string:
		return "string"
	case json.Number:
		return "number"
	case float64, float32, int, int64, int32, uint, uint64, uint32:
		return "number"
	case map[string]any:
		fields := make(map[string]any, len(cur))
		for k, val := range cur {
			fields[k] = buildShape(val)
		}
		return map[string]any{
			"type":   "object",
			"fields": fields,
		}
	case []any:
		itemShapes := collectArrayItemShapes(cur)
		return map[string]any{
			"type":  "array",
			"items": itemShapes,
		}
	default:
		return fmt.Sprintf("unknown:%T", v)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
