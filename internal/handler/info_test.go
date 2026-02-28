package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGetInfo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	rr := httptest.NewRecorder()

	deps := &Deps{} // nil services trigger fallback stub
	GetInfo(deps)(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	requiredKeys := []string{
		"baseCurrency",
		"benchmarks",
		"currencies",
		"demoAuthToken",
		"globalPermissions",
		"statistics",
	}

	for _, key := range requiredKeys {
		if _, ok := body[key]; !ok {
			t.Fatalf("expected key %q in response body", key)
		}
	}

	if got, ok := body["baseCurrency"].(string); !ok || got != "USD" {
		t.Fatalf("expected baseCurrency=USD, got %#v", body["baseCurrency"])
	}
}

func TestInfoRoute(t *testing.T) {
	deps := &Deps{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/info", GetInfo(deps))

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "route serves GET",
			method:     http.MethodGet,
			path:       "/api/v1/info",
			wantStatus: http.StatusOK,
		},
		{
			name:       "method not allowed",
			method:     http.MethodPost,
			path:       "/api/v1/info",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rr := httptest.NewRecorder()

			mux.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rr.Code)
			}
		})
	}
}
