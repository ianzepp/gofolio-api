package handler

import "net/http"

// HandleFlushCache handles POST /cache/flush.
// In the Go server there is no Redis cache to flush — this is a no-op that
// maintains API contract compatibility with the NestJS server.
// Auth: admin-only (accessAdminControl), enforced at the routing layer.
func HandleFlushCache(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
