// Package handler — admin.go implements handlers for /api/v1/admin/*.
//
// The admin surface covers platform-wide dashboards, market-data management,
// and user administration. Most data-gather and profile-data mutation routes
// are stubs that return empty success responses — they exist to satisfy the
// API contract while the background job system is ported from NestJS.
//
// The four non-stub handlers (GetAdmin, GetAdminMarketData, GetAdminUsers,
// GetAdminUser) are guarded at routing time by the accessAdminControl permission
// and delegate to AdminService for all DB access, keeping the handler layer thin.
package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
)

// GetAdmin handles GET /api/v1/admin and returns the admin dashboard summary.
// If AdminService is not wired (e.g. in test environments) a safe empty response
// is returned rather than a service-unavailable error, because the dashboard is
// non-critical and some deployments may intentionally omit the admin service.
func GetAdmin(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.AdminService == nil {
			JSON(w, http.StatusOK, map[string]any{
				"exchangeRates":    []any{},
				"settings":         map[string]any{},
				"transactionCount": 0,
				"userCount":        0,
				"version":          "",
			})
			return
		}

		dashboard, err := deps.AdminService.GetDashboard(r.Context())
		if err != nil {
			Error(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		JSON(w, http.StatusOK, dashboard)
	}
}

// GetAdminMarketData handles GET /api/v1/admin/market-data. Pagination and
// filter parameters are parsed via the shared ParsePagination/ParseFilters
// helpers and forwarded to AdminService for DB-level filtering.
func GetAdminMarketData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.AdminService == nil {
			JSON(w, http.StatusOK, map[string]any{"marketData": []any{}})
			return
		}

		pg := ParsePagination(r)
		filters := ParseFilters(r)

		resp, err := deps.AdminService.GetMarketData(r.Context(), pg.Skip, pg.Take, filters.Query, pg.SortColumn, pg.SortDirection)
		if err != nil {
			Error(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		JSON(w, http.StatusOK, resp)
	}
}

// GetAdminUsers handles GET /api/v1/admin/user and returns a paginated user list.
func GetAdminUsers(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.AdminService == nil {
			JSON(w, http.StatusOK, []any{})
			return
		}

		pg := ParsePagination(r)

		resp, err := deps.AdminService.GetUsers(r.Context(), pg.Skip, pg.Take)
		if err != nil {
			Error(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		JSON(w, http.StatusOK, resp)
	}
}

// GetAdminUser handles GET /api/v1/admin/user/{id} and returns the full admin
// view of a single user, including role, settings, and activity counts.
// Returns 404 when the user does not exist, unlike other admin endpoints that
// return empty objects — the NestJS original propagates the Prisma not-found error.
func GetAdminUser(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		if deps.AdminService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		user, err := deps.AdminService.GetUser(r.Context(), id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusNotFound, "User not found")
				return
			}
			Error(w, http.StatusInternalServerError, "Internal server error")
			return
		}

		JSON(w, http.StatusOK, user)
	}
}

// --- Stubs that remain unchanged ---

// GET /admin/demo-user/sync
func HandleSyncDemoUser(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather
func HandleGatherData(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/:dataSource/:symbol
func HandleGatherDataForSymbol(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	dateRange := r.URL.Query().Get("range")
	_ = dataSource
	_ = symbol
	_ = dateRange
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/:dataSource/:symbol/:dateString
func HandleGatherDataForDate(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	dateString := r.PathValue("dateString")
	_ = dataSource
	_ = symbol
	_ = dateString
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/max
func HandleGatherMax(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/profile-data
func HandleGatherProfileData(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/profile-data/:dataSource/:symbol
func HandleGatherProfileDataForSymbol(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/market-data/:dataSource/:symbol/test
func HandleTestMarketData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}

// DELETE /admin/profile-data/:dataSource/:symbol
func DeleteProfileData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		if deps.AdminService == nil {
			Error(w, http.StatusInternalServerError, "Service unavailable")
			return
		}

		if err := deps.AdminService.DeleteProfileData(r.Context(), dataSource, symbol); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, map[string]any{})
	}
}

// PATCH /admin/profile-data/:dataSource/:symbol
func UpdateProfileData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		var body map[string]any
		if !Bind(w, r, &body) {
			return
		}

		if deps.AdminService == nil {
			Error(w, http.StatusInternalServerError, "Service unavailable")
			return
		}

		if err := deps.AdminService.UpdateProfileData(r.Context(), dataSource, symbol, body); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, map[string]any{})
	}
}

// POST /admin/profile-data/:dataSource/:symbol
func CreateProfileData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		var body struct {
			Currency string `json:"currency"`
		}
		if !Bind(w, r, &body) {
			return
		}

		if deps.AdminService == nil {
			Error(w, http.StatusInternalServerError, "Service unavailable")
			return
		}

		currency := body.Currency
		if currency == "" {
			currency = "USD"
			if deps.UserService != nil {
				userID := auth.UserID(r)
				if userID != "" {
					settings, err := deps.UserService.GetSettings(r.Context(), userID)
					if err == nil && settings.Settings != nil {
						settingsMap := map[string]any{}
						if err := json.Unmarshal([]byte(*settings.Settings), &settingsMap); err == nil {
							if baseCurrency, ok := settingsMap["baseCurrency"].(string); ok && baseCurrency != "" {
								currency = baseCurrency
							}
						}
					}
				}
			}
		}

		if err := deps.AdminService.CreateProfileData(r.Context(), dataSource, symbol, currency); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusCreated, map[string]any{})
	}
}

// PUT /admin/settings/:key
func HandleUpdateAdminSetting(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	_ = key
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}
