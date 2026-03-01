// Package handler implements HTTP handler functions for the Ghostfolio Go API.
// Each handler follows a closure pattern: a constructor (e.g. GetUser) receives a
// *Deps dependency bundle and returns an http.HandlerFunc. This decouples routing
// from service wiring and makes handlers trivially unit-testable.
//
// This file covers the /api/v1/user family of endpoints, ported from the NestJS
// UserController. The most significant behavioral difference from a standard REST
// API is the access-token model: every user has a UUID access token that is stored
// in the database as an HMAC-SHA512 hash (salted with ACCESS_TOKEN_SALT). This
// lets the client prove identity when rotating or deleting credentials without
// having the server store the plaintext token.
//
// Parity reference: apps/api/src/app/user/user.controller.ts
//
// Routes implemented:
//   - GET    /api/v1/user              GetUser
//   - POST   /api/v1/user              CreateUser (anonymous signup)
//   - DELETE /api/v1/user              DeleteUser (self-delete with token proof)
//   - DELETE /api/v1/user/{id}         DeleteUserById (admin delete)
//   - POST   /api/v1/user/access-token CreateUserAccessToken (rotate own token)
//   - POST   /api/v1/user/{id}/access-token CreateUserAccessTokenById (admin rotate)
//   - PUT    /api/v1/user/setting      UpdateUserSetting
package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/user/user.controller.ts

Routes + request/response shape:
- GET /api/v1/user (GetUser)
  Auth/permission: jwt / -
  Response: User { id, accounts, access, settings, permissions, subscription, tags, ... }
- POST /api/v1/user (CreateUser / signup)
  Auth/permission: public / -
  Response: { accessToken, authToken, role }
- DELETE /api/v1/user (DeleteUser)
  Auth/permission: jwt / permissions.deleteOwnUser
  Response: User
- DELETE /api/v1/user/{id} (DeleteUserById)
  Auth/permission: jwt / permissions.deleteUser
  Response: User
- POST /api/v1/user/{id}/access-token (CreateUserAccessTokenById)
  Auth/permission: jwt / permissions.accessAdminControl
  Response: { accessToken }
- POST /api/v1/user/access-token (CreateUserAccessToken)
  Auth/permission: jwt / permissions.updateOwnAccessToken
  Response: { accessToken }
- PUT /api/v1/user/setting (UpdateUserSetting)
  Auth/permission: jwt / -
  Response: settings object
*/

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"encoding/hex"
	"errors"
	"net/http"
	"os"

	"github.com/ianzepp/gofolio-api/internal/auth"
	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/google/uuid"
	"database/sql"
)

// GetUser handles GET /api/v1/user and assembles the rich user object expected by
// the Ghostfolio frontend. The response aggregates data from four different
// services (user, settings, accounts, tags) because the NestJS original performs
// the same fan-out in-controller rather than in a dedicated service layer.
//
// WHY: The client uses a single GET /user call on startup to bootstrap the entire
// UI state (permissions, settings, account list, tags). Keeping that aggregation
// in the handler mirrors the NestJS contract exactly and avoids a new service API.
func GetUser(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		user, err := deps.UserService.GetUser(r.Context(), userID)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Build permissions list from role.
		// Note: Authenticate middleware only sets UserID, not the full User object.
		// We already loaded the user above, so use that directly.
		permissions := auth.PermissionsForRole(user.Role)

		// Load user settings
		var settingsMap map[string]any
		settings, err := deps.UserService.GetSettings(r.Context(), userID)
		if err == nil && settings.Settings != nil {
			_ = json.Unmarshal([]byte(*settings.Settings), &settingsMap)
		}
		if settingsMap == nil {
			settingsMap = map[string]any{}
		}

		// Load accounts
		accounts := []model.Account{}
		if deps.AccountService != nil {
			accts, err := deps.AccountService.GetAccounts(r.Context(), userID)
			if err == nil && accts != nil {
				accounts = accts
			}
		}

		// Load tags
		tags := []model.Tag{}
		if deps.TagService != nil {
			t, err := deps.TagService.GetTagsByUserID(r.Context(), userID)
			if err == nil && t != nil {
				tags = t
			}
		}

		// Legacy parity: GET /user returns a rich User object
		JSON(w, http.StatusOK, map[string]any{
			"id":          user.ID,
			"access":      []any{},
			"accounts":    accounts,
			"permissions": permissions,
			"settings":    settingsMap,
			"subscription": map[string]any{
				"type": "Basic",
			},
			"tags": tags,
		})
	}
}

// CreateUser handles POST /api/v1/user and creates an anonymous user account.
// On success it issues both a plaintext access token (stored hashed in the DB)
// and a signed JWT so the caller can immediately authenticate subsequent requests.
//
// WHY: Ghostfolio supports access-token-based anonymous sign-up with no email or
// password. The plaintext token is returned once here and never again — the caller
// must store it. The JWT is the short-lived session credential.
func CreateUser(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := uuid.New().String()
		accessToken := uuid.New().String()

		user := model.User{
			ID:          id,
			AccessToken: &accessToken,
			Provider:    model.ProviderAnonymous,
			Role:        model.RoleUser,
		}

		created, err := deps.UserService.CreateUser(r.Context(), user)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Sign JWT for the new user
		jwtSecret := os.Getenv("JWT_SECRET_KEY")
		authToken, err := auth.SignJWT(created.ID, jwtSecret)
		if err != nil {
			Error(w, http.StatusInternalServerError, "failed to sign auth token")
			return
		}

		JSON(w, http.StatusCreated, map[string]any{
			"accessToken": accessToken,
			"authToken":   authToken,
			"role":        created.Role,
		})
	}
}

// DeleteUser handles DELETE /api/v1/user (self-delete). The caller must supply
// their plaintext access token in the request body; the handler hashes it and
// cross-checks against the DB record to confirm identity before deleting.
//
// WHY: Requiring token proof prevents CSRF-style deletions where an attacker
// has a valid JWT but not the access token. This matches the NestJS contract.
func DeleteUser(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body struct {
			AccessToken string `json:"accessToken"`
		}
		if !Bind(w, r, &body) {
			return
		}

		hashed, err := hashAccessToken(body.AccessToken)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		userByToken, err := deps.UserService.GetUserByAccessToken(r.Context(), hashed)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if userByToken.ID != userID {
			Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}

		if err := deps.UserService.DeleteUser(r.Context(), userID); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// DeleteUserById handles DELETE /api/v1/user/{id} for admin-initiated deletions.
// Admins cannot delete themselves via this route — self-deletion must go through
// DeleteUser which requires access-token proof.
//
// WHY: Separating admin-delete from self-delete preserves the two-factor proof
// requirement for self-deletion while giving admins an unconditional delete path.
func DeleteUserById(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		currentUserID := auth.UserID(r)
		if currentUserID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		id := r.PathValue("id")
		if id == currentUserID {
			// Legacy parity: delete-by-id cannot target self.
			Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}
		if err := deps.UserService.DeleteUser(r.Context(), id); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// CreateUserAccessToken handles POST /api/v1/user/access-token, allowing a user
// to rotate their own access token. The caller must present the current plaintext
// token in the request body to prove possession before a new one is generated.
//
// WHY: This proof-of-possession requirement prevents an attacker with a stolen
// JWT from silently rotating the victim's access token.
func CreateUserAccessToken(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body struct {
			AccessToken string `json:"accessToken"`
		}
		if !Bind(w, r, &body) {
			return
		}

		currentHashed, err := hashAccessToken(body.AccessToken)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		userByToken, err := deps.UserService.GetUserByAccessToken(r.Context(), currentHashed)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if userByToken.ID != userID {
			Error(w, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}

		// Generate new access token
		accessToken := uuid.New().String()
		hashed, err := hashAccessToken(accessToken)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := deps.UserService.UpdateAccessToken(r.Context(), userID, hashed); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusCreated, map[string]any{
			"accessToken": accessToken,
		})
	}
}

// CreateUserAccessTokenById handles POST /api/v1/user/{id}/access-token for
// admin-initiated token rotation. Unlike the self-serve endpoint, no current
// token proof is required — access is gate-kept by the accessAdminControl
// permission at the routing level.
func CreateUserAccessTokenById(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		accessToken := uuid.New().String()
		hashed, err := hashAccessToken(accessToken)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := deps.UserService.UpdateAccessToken(r.Context(), id, hashed); err != nil {
			// Legacy parity: prisma update on missing user propagates internal server error.
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusCreated, map[string]any{
			"accessToken": accessToken,
		})
	}
}

// hashAccessToken produces a deterministic HMAC-SHA512 hex digest of the
// plaintext access token using ACCESS_TOKEN_SALT as the key. Storing only
// the hash means a DB breach does not expose usable tokens.
//
// EDGE: Returns an error if ACCESS_TOKEN_SALT is empty — callers must handle
// this as a configuration failure rather than an auth error.
func hashAccessToken(accessToken string) (string, error) {
	salt := os.Getenv("ACCESS_TOKEN_SALT")
	if salt == "" {
		return "", errors.New("ACCESS_TOKEN_SALT is required")
	}
	mac := hmac.New(sha512.New, []byte(salt))
	_, err := mac.Write([]byte(accessToken))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// UpdateUserSetting handles PUT /api/v1/user/setting and performs a
// partial-update (merge) of the caller's settings JSON blob. Null values in the
// request body delete the corresponding key; non-null values upsert it.
//
// WHY: Settings are stored as opaque JSON in the DB. Merging rather than
// replacing avoids clients needing to round-trip the full settings object on
// every partial update, and matches the NestJS behavior.
func UpdateUserSetting(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		var body map[string]any
		if !Bind(w, r, &body) {
			return
		}

		// Merge incoming settings with existing
		existing, _ := deps.UserService.GetSettings(r.Context(), userID)
		var merged map[string]any
		if existing.Settings != nil {
			_ = json.Unmarshal([]byte(*existing.Settings), &merged)
		}
		if merged == nil {
			merged = map[string]any{}
		}
		for k, v := range body {
			if v == nil {
				delete(merged, k)
			} else {
				merged[k] = v
			}
		}

		settingsJSON, err := json.Marshal(merged)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		settingsStr := string(settingsJSON)

		_, err = deps.UserService.UpdateSettings(r.Context(), model.Settings{
			UserId:   userID,
			Settings: &settingsStr,
		})
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, merged)
	}
}
