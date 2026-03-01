package handler

import "net/http"

// FlushCache handles POST /cache/flush.
// When Redis is configured, this clears cache namespaces used by Ghostfolio.
// Without Redis, it degrades to a no-op for local/dev parity.
// Auth: admin-only (accessAdminControl), enforced at the routing layer.
func FlushCache(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps != nil && deps.CacheService != nil {
			if err := deps.CacheService.Flush(r.Context()); err != nil {
				Error(w, http.StatusInternalServerError, err.Error())
				return
			}
		}

		// NestJS @Post handlers default to 201 when no explicit HttpCode is set.
		w.WriteHeader(http.StatusCreated)
	}
}
