// Package handler — data_provider.go implements the Ghostfolio data-provider
// proxy endpoints under /api/v1/data-providers/ghostfolio/*.
//
// These endpoints expose the Go server as a data-provider to external consumers
// (e.g. other Ghostfolio instances configured to use this one as a Ghostfolio
// data source). All routes delegate to DataProviderService which fans out to the
// registered provider implementations (Yahoo Finance, Manual, etc.).
//
// Status endpoint design note: GetDataProviderStatus hits three separate DB
// repositories (analytics, property, subscription) to assemble the quota/plan
// response. This mirrors the NestJS AdminController helper that does the same
// fan-out, and is intentionally left in the handler rather than a service because
// it aggregates cross-domain data with no business logic.
//
// Parity reference:
// apps/api/src/app/endpoints/data-providers/ghostfolio/ghostfolio.controller.ts
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

	"database/sql"
	"github.com/ianzepp/gofolio-api/internal/auth"
	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/provider"
	"github.com/ianzepp/gofolio-api/internal/repository"
)

// GetDataProviderAssetProfile handles GET /api/v1/data-providers/ghostfolio/asset-profile/{symbol}.
// The symbol is always looked up against the Yahoo Finance data source because
// this endpoint is part of the Ghostfolio data-provider protocol, which currently
// treats Yahoo as the default external source.
//
// EDGE: If no profile is found the handler returns an empty object rather than
// 404, matching the NestJS contract (the client handles missing profiles gracefully).
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

// GetDataProviderDividends handles GET /api/v1/data-providers/ghostfolio/dividends/{symbol}.
// The from/to query parameters default to the trailing 12-month window when omitted.
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

// GetDataProviderHistorical handles GET /api/v1/data-providers/ghostfolio/historical/{symbol}.
// Returns OHLCV-style historical data keyed by date string (YYYY-MM-DD) under
// the "historicalData" key. Date range defaults to trailing 12 months.
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

// DataProviderLookup handles GET /api/v1/data-providers/ghostfolio/lookup.
// Short queries (fewer than 2 non-space characters) return an empty item list
// immediately rather than hitting the upstream provider. This matches the NestJS
// minimum-length guard and avoids noise queries to Yahoo Finance.
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

// GetDataProviderQuotes handles GET /api/v1/data-providers/ghostfolio/quotes.
// Accepts a comma-separated "symbols" query parameter and returns a map of
// symbol → quote data. All symbols are assumed to be Yahoo Finance symbols;
// data-source routing is not performed at this endpoint.
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

// GetDataProviderStatus handles GET /api/v1/data-providers/ghostfolio/status and
// returns quota usage and subscription tier for the authenticated user.
//
// PHASE 1: Read daily request count from analytics repository.
// PHASE 2: Read configured daily request cap from the property store.
// PHASE 3: Determine subscription tier by checking for an active subscription row.
//
// WHY: The quota cap is stored as an admin-configurable property rather than a
// compile-time constant so operators can tune it without a redeploy. The
// subscription tier is derived at query-time (not cached) to reflect real-time
// plan changes.
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

		if deps.DB != nil {
			analyticsRepo := repository.NewAnalyticsRepository(deps.DB)
			if analytics, err := analyticsRepo.FindByUserID(r.Context(), authInfo.UserID); err == nil {
				dailyRequests = analytics.DataProviderGhostfolioDailyRequests
			}

			propRepo := repository.NewPropertyRepository(deps.DB)
			if prop, err := propRepo.FindByKey(r.Context(), "DATA_SOURCES_GHOSTFOLIO_DATA_PROVIDER_MAX_REQUESTS"); err == nil {
				if parsed, parseErr := strconv.Atoi(strings.TrimSpace(prop.Value)); parseErr == nil {
					dailyRequestsMax = parsed
				}
			}

			subscriptionRepo := repository.NewSubscriptionRepository(deps.DB)
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

// parseQueryTime reads a YYYY-MM-DD date from a query parameter and returns
// defaultVal if the parameter is absent or cannot be parsed.
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

// isNotFoundError reports whether err represents a "no rows" DB result,
// used to distinguish missing-record from genuine DB errors.
func isNotFoundError(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
