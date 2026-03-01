package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleGetHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	HandleGetHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if got := rr.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", got)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if body["status"] != "OK" {
		t.Fatalf("expected status=ok, got %q", body["status"])
	}
}

func TestHealthRoutes(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", HandleGetHealth)
	mux.HandleFunc("GET /api/v1/health/data-enhancer/{name}", HandleGetHealthDataEnhancer)
	mux.HandleFunc("GET /api/v1/health/data-provider/{dataSource}", HandleGetHealthDataProvider)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantKV     map[string]string
	}{
		{
			name:       "health ok",
			method:     http.MethodGet,
			path:       "/api/v1/health",
			wantStatus: http.StatusOK,
			wantKV:     map[string]string{"status": "OK"},
		},
		{
			name:       "health data enhancer known name",
			method:     http.MethodGet,
			path:       "/api/v1/health/data-enhancer/YAHOO",
			wantStatus: http.StatusOK,
			wantKV: map[string]string{
				"status": "ok",
				"name":   "YAHOO",
			},
		},
		{
			name:       "health data enhancer unknown name",
			method:     http.MethodGet,
			path:       "/api/v1/health/data-enhancer/yahoo-finance",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "health data provider path param",
			method:     http.MethodGet,
			path:       "/api/v1/health/data-provider/YAHOO",
			wantStatus: http.StatusOK,
			wantKV:     map[string]string{"status": "OK"},
		},
		{
			name:       "method not allowed",
			method:     http.MethodPost,
			path:       "/api/v1/health",
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

			if len(tc.wantKV) == 0 {
				return
			}

			var body map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode response body: %v", err)
			}

			for key, want := range tc.wantKV {
				if got := body[key]; got != want {
					t.Fatalf("expected %s=%q, got %q", key, want, got)
				}
			}
		})
	}
}
