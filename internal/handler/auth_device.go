package handler

import (
	"net/http"

	"github.com/ianzepp/gofolio-api/internal/auth"
)

// DeleteAuthDevice handles DELETE /api/v1/auth-device/{id}.
// Removes a WebAuthn device registration by ID.
// Auth: jwt + deleteAuthDevice permission, enforced at the routing layer.
func DeleteAuthDevice(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.DB == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")

		// Legacy parity: delete is keyed by device id only.
		_, err := deps.DB.ExecContext(r.Context(),
			`DELETE FROM "AuthDevice" WHERE id = $1`,
			id)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, map[string]any{})
	}
}
