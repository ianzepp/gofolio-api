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
	"encoding/json"
	"net/http"
	"os"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/google/uuid"
)

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

		if err := deps.UserService.DeleteUser(r.Context(), userID); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

func DeleteUserById(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		id := r.PathValue("id")
		if err := deps.UserService.DeleteUser(r.Context(), id); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

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

		// Generate new access token
		accessToken := uuid.New().String()
		// TODO: persist the new access token to the user record
		JSON(w, http.StatusCreated, map[string]any{
			"accessToken": accessToken,
		})
	}
}

func CreateUserAccessTokenById(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.UserService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		_ = r.PathValue("id")
		accessToken := uuid.New().String()
		// TODO: persist the new access token to the target user record
		JSON(w, http.StatusCreated, map[string]any{
			"accessToken": accessToken,
		})
	}
}

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
