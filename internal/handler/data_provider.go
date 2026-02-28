package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/data-providers/ghostfolio/ghostfolio.controller.ts

Routes + request/response shape:
- GET /api/v1/data-providers/ghostfolio/asset-profile/{symbol} (GetDataProviderAssetProfile)
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
- GET /api/v1/data-providers/ghostfolio/dividends/{symbol} (GetDataProviderDividends)
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
- GET /api/v1/data-providers/ghostfolio/historical/{symbol} (GetDataProviderHistorical)
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
- GET /api/v1/data-providers/ghostfolio/lookup (DataProviderLookup)
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
- GET /api/v1/data-providers/ghostfolio/quotes (GetDataProviderQuotes)
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
- GET /api/v1/data-providers/ghostfolio/status (GetDataProviderStatus)
  Auth/permission: api-key / permissions.enableDataProviderGhostfolio
*/

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/auth"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/provider"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5"
)

// GetDataProviderAssetProfile returns a handler for GET /api/v1/data-providers/ghostfolio/asset-profile/{symbol}.
func GetDataProviderAssetProfile(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DataProviderService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		symbol := r.PathValue("symbol")
		items := []provider.AssetProfileIdentifier{
			{DataSource: model.DataSourceYahoo, Symbol: symbol},
		}

		profiles, err := deps.DataProviderService.GetAssetProfiles(r.Context(), items)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		if p, ok := profiles[symbol]; ok && p != nil {
			JSON(w, http.StatusOK, p)
			return
		}
		JSON(w, http.StatusOK, map[string]any{})
	}
}

// GetDataProviderDividends returns a handler for GET /api/v1/data-providers/ghostfolio/dividends/{symbol}.
func GetDataProviderDividends(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DataProviderService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		symbol := r.PathValue("symbol")
		from := parseQueryTime(r, "from", time.Now().AddDate(-1, 0, 0))
		to := parseQueryTime(r, "to", time.Now())

		dividends, err := deps.DataProviderService.GetDividends(r.Context(), model.DataSourceYahoo, symbol, from, to)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, map[string]any{"dividends": dividends})
	}
}

// GetDataProviderHistorical returns a handler for GET /api/v1/data-providers/ghostfolio/historical/{symbol}.
func GetDataProviderHistorical(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DataProviderService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		symbol := r.PathValue("symbol")
		from := parseQueryTime(r, "from", time.Now().AddDate(-1, 0, 0))
		to := parseQueryTime(r, "to", time.Now())

		items := []provider.AssetProfileIdentifier{
			{DataSource: model.DataSourceYahoo, Symbol: symbol},
		}

		historical, err := deps.DataProviderService.GetHistoricalRaw(r.Context(), items, from, to)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		data := map[string]provider.HistoricalResponse{}
		if h, ok := historical[symbol]; ok {
			data = h
		}

		JSON(w, http.StatusOK, map[string]any{"historicalData": data})
	}
}

// DataProviderLookup returns a handler for GET /api/v1/data-providers/ghostfolio/lookup.
func DataProviderLookup(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DataProviderService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		query := r.URL.Query().Get("query")
		if query == "" || len(strings.TrimSpace(query)) < 2 {
			JSON(w, http.StatusOK, map[string]any{"items": []any{}})
			return
		}

		includeIndices := r.URL.Query().Get("includeIndices") == "true"

		result, err := deps.DataProviderService.Search(r.Context(), query, includeIndices)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, result)
	}
}

// GetDataProviderQuotes returns a handler for GET /api/v1/data-providers/ghostfolio/quotes.
func GetDataProviderQuotes(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DataProviderService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		symbolsParam := r.URL.Query().Get("symbols")
		if symbolsParam == "" {
			JSON(w, http.StatusOK, map[string]any{"quotes": map[string]any{}})
			return
		}

		symbols := strings.Split(symbolsParam, ",")
		items := make([]provider.AssetProfileIdentifier, len(symbols))
		for i, s := range symbols {
			items[i] = provider.AssetProfileIdentifier{
				DataSource: model.DataSourceYahoo,
				Symbol:     strings.TrimSpace(s),
			}
		}

		quotes, err := deps.DataProviderService.GetQuotes(r.Context(), items)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, map[string]any{"quotes": quotes})
	}
}

// GetDataProviderStatus returns a handler for GET /api/v1/data-providers/ghostfolio/status.
func GetDataProviderStatus(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DataProviderService == nil {
			Error(w, http.StatusServiceUnavailable, "Service unavailable")
			return
		}

		authInfo := auth.GetAuthInfo(r.Context())
		if authInfo == nil || authInfo.UserID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		dailyRequests := 0
		dailyRequestsMax := 0
		subscription := map[string]any{"type": "Basic"}

		if deps.Pool != nil {
			analyticsRepo := repository.NewAnalyticsRepository(deps.Pool)
			if analytics, err := analyticsRepo.FindByUserID(r.Context(), authInfo.UserID); err == nil {
				dailyRequests = analytics.DataProviderGhostfolioDailyRequests
			}

			propRepo := repository.NewPropertyRepository(deps.Pool)
			if prop, err := propRepo.FindByKey(r.Context(), "DATA_SOURCES_GHOSTFOLIO_DATA_PROVIDER_MAX_REQUESTS"); err == nil {
				if parsed, parseErr := strconv.Atoi(strings.TrimSpace(prop.Value)); parseErr == nil {
					dailyRequestsMax = parsed
				}
			}

			subscriptionRepo := repository.NewSubscriptionRepository(deps.Pool)
			if _, err := subscriptionRepo.FindActiveByUserID(r.Context(), authInfo.UserID); err == nil {
				subscription = map[string]any{"type": "Premium"}
			} else if err != nil && !isNotFoundError(err) {
				Error(w, http.StatusInternalServerError, err.Error())
				return
			}
		}

		JSON(w, http.StatusOK, map[string]any{
			"dailyRequests":    dailyRequests,
			"dailyRequestsMax": dailyRequestsMax,
			"subscription":     subscription,
		})
	}
}

func parseQueryTime(r *http.Request, key string, defaultVal time.Time) time.Time {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	t, err := time.Parse("2006-01-02", val)
	if err != nil {
		return defaultVal
	}
	return t
}

func isNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
