// Package handler — health.go
//
// Implements the three public health endpoints ported from
// apps/api/src/app/health/health.controller.ts.
//
// All three routes are unauthenticated so that load balancers and orchestrators
// (Kubernetes liveness/readiness probes, uptime monitors) can poll them without
// a JWT. The responses are intentionally minimal — just enough for automated
// checks to detect server availability without exposing internal state.

package handler

import "net/http"

// knownDataEnhancers is the fixed set of data-enrichment integrations
// recognised by this server. A request for an unrecognised enhancer returns
// 404 rather than an empty "ok" to match NestJS behavior and avoid
// silently masking misconfigured probe targets.
var knownDataEnhancers = map[string]struct{}{
	"YAHOO":        {},
	"OPENFIGI":     {},
	"TRACKINSIGHT": {},
}

// HandleGetHealth responds to GET /api/v1/health with a minimal liveness
// payload. It does not check database or cache connectivity — those concerns
// belong to readiness probes, not liveness. Keeping this handler dependency-
// free means it stays available even when the DB pool is nil.
func HandleGetHealth(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// HandleGetHealthDataEnhancer responds to GET /api/v1/health/data-enhancer/{name}.
// It validates the name against the known-enhancer set and returns 404 for
// unrecognised values. The NestJS controller returns 404 via NotFoundException
// for unknown enhancers; this mirrors that behavior so probe configs do not
// need to change when traffic is routed to this server.
func HandleGetHealthDataEnhancer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if _, ok := knownDataEnhancers[name]; !ok {
		Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"name":   name,
		"status": "ok",
	})
}

// HandleGetHealthDataProvider responds to GET /api/v1/health/data-provider/{dataSource}.
// Unlike the data-enhancer endpoint it accepts any dataSource value without
// validation — the NestJS controller does not reject unknown data sources here,
// so this handler matches that permissive behavior to preserve parity.
//
// NOTE: A future implementation should verify provider connectivity and return
// a non-ok status when the provider is unreachable, matching the richer health
// check behavior in the NestJS HealthIndicator pattern.
func HandleGetHealthDataProvider(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	JSON(w, http.StatusOK, map[string]any{
		"dataSource": dataSource,
		"status":     "ok",
	})
}
