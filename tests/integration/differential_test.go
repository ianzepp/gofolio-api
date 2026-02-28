package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	Name            string            `json:"name"`
	Method          string            `json:"method"`
	Path            string            `json:"path"`
	Mode            compareMode       `json:"mode"`
	Headers         map[string]string `json:"headers"`
	Auth            string            `json:"auth"` // "", "jwt", "api-key"
	Body            json.RawMessage   `json:"body"`
	IgnoreJSONPaths []string          `json:"ignoreJsonPaths"`
}

type response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

func TestDifferentialAPI(t *testing.T) {
	legacyBaseURL := strings.TrimRight(os.Getenv("LEGACY_BASE_URL"), "/")
	goBaseURL := strings.TrimRight(os.Getenv("GO_BASE_URL"), "/")

	if legacyBaseURL == "" || goBaseURL == "" {
		t.Skip("set LEGACY_BASE_URL and GO_BASE_URL to run differential integration tests")
	}

	cases, err := loadCases()
	if err != nil {
		t.Fatalf("load cases: %v", err)
	}

	client := &http.Client{Timeout: 20 * time.Second}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
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
	fixturePath := filepath.Join("testdata", "cases.json")
	b, err := os.ReadFile(fixturePath)
	if err != nil {
		return nil, err
	}

	var cases []testCase
	if err := json.Unmarshal(b, &cases); err != nil {
		return nil, err
	}

	for i := range cases {
		if cases[i].Mode == "" {
			cases[i].Mode = compareStatus
		}
		if cases[i].Method == "" || cases[i].Path == "" || cases[i].Name == "" {
			return nil, fmt.Errorf("invalid case at index %d: name/method/path required", i)
		}
	}

	return cases, nil
}

func doRequest(client *http.Client, baseURL string, tc testCase) (response, error) {
	var bodyReader io.Reader
	if len(tc.Body) > 0 {
		bodyReader = bytes.NewReader(tc.Body)
	}

	req, err := http.NewRequest(tc.Method, baseURL+tc.Path, bodyReader)
	if err != nil {
		return response{}, err
	}

	if len(tc.Body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range tc.Headers {
		req.Header.Set(k, v)
	}

	switch tc.Auth {
	case "jwt":
		token := os.Getenv("JWT_TOKEN")
		if token == "" {
			return response{}, fmt.Errorf("case %q requires JWT_TOKEN", tc.Name)
		}
		req.Header.Set("Authorization", "Bearer "+token)
	case "api-key":
		key := os.Getenv("API_KEY")
		if key == "" {
			return response{}, fmt.Errorf("case %q requires API_KEY", tc.Name)
		}
		req.Header.Set("Authorization", "Bearer "+key)
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
