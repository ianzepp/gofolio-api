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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/ianzepp/gofolio-api/internal/queue"
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
				displayID := id
				if _, parseErr := uuid.Parse(id); parseErr == nil {
					displayID = strings.ToUpper(id)
				}
				JSON(w, http.StatusNotFound, map[string]any{
					"error":      "Not Found",
					"message":    fmt.Sprintf("User with ID %s not found", displayID),
					"statusCode": http.StatusNotFound,
				})
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
func SyncDemoUser(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = deps
		_ = r
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// POST /admin/gather
func GatherData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := enqueueGatherJob(deps, r, queue.DataGatheringQueueName, queue.JobGatherHistoricalMarketData, map[string]any{
			"mode": "default",
		}); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// POST /admin/gather/:dataSource/:symbol
func GatherDataForSymbol(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")
		dateRange := r.URL.Query().Get("range")

		if err := enqueueGatherJob(deps, r, queue.DataGatheringQueueName, queue.JobGatherHistoricalMarketData, map[string]any{
			"dataSource": dataSource,
			"symbol":     symbol,
			"range":      dateRange,
		}); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// POST /admin/gather/:dataSource/:symbol/:dateString
func GatherDataForDate(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")
		dateString := r.PathValue("dateString")

		if err := enqueueGatherJob(deps, r, queue.DataGatheringQueueName, queue.JobGatherHistoricalMarketData, map[string]any{
			"dataSource": dataSource,
			"symbol":     symbol,
			"dateString": dateString,
		}); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// POST /admin/gather/max
func GatherMax(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := enqueueGatherJob(deps, r, queue.DataGatheringQueueName, queue.JobGatherHistoricalMarketData, map[string]any{
			"mode": "max",
		}); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// POST /admin/gather/profile-data
func GatherProfileData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := enqueueGatherJob(deps, r, queue.DataGatheringQueueName, queue.JobGatherAssetProfile, map[string]any{
			"mode": "all",
		}); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// POST /admin/gather/profile-data/:dataSource/:symbol
func GatherProfileDataForSymbol(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		if err := enqueueGatherJob(deps, r, queue.DataGatheringQueueName, queue.JobGatherAssetProfile, map[string]any{
			"dataSource": dataSource,
			"symbol":     symbol,
		}); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// POST /admin/market-data/:dataSource/:symbol/test
func TestMarketData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")
		var body map[string]any
		if !Bind(w, r, &body) {
			return
		}

		if err := enqueueGatherJob(deps, r, queue.DataGatheringQueueName, queue.JobGatherHistoricalMarketData, map[string]any{
			"mode":       "test",
			"dataSource": dataSource,
			"symbol":     symbol,
			"payload":    body,
		}); err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
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
		_ = deps
		_ = r
		Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
	}
}

// PUT /admin/settings/:key
func UpdateAdminSetting(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.AdminService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		key := r.PathValue("key")
		var body struct {
			Value string `json:"value"`
		}
		if !Bind(w, r, &body) {
			return
		}

		result, err := deps.AdminService.PutSetting(r.Context(), key, body.Value)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		JSON(w, http.StatusOK, result)
	}
}
