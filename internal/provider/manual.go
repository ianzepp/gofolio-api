// Package provider — manual.go implements a DataProvider backed by the local
// Postgres database rather than an external API.
//
// The ManualProvider serves symbols whose DataSource is model.DataSourceManual.
// These are user-created assets (e.g. private equity, real estate, custom funds)
// where market data must be entered manually or scraped from a user-configured URL.
//
// # Scraper integration
//
// When a symbol profile has a ScraperConfiguration JSON blob, GetQuotes attempts
// to fetch a live price from the configured URL using a dot-path selector over
// the JSON response. Only "instant" mode is supported (direct JSON GET); HTML
// scraping is deferred. If the scraper fetch fails for any reason the last stored
// DB price is used as the fallback — the error is silently swallowed to avoid
// breaking the entire quote batch for a misconfigured scraper.
//
// # Dividend data
//
// ManualProvider.GetDividends always returns empty because manual symbols do not
// have structured dividend event data in the DB schema. Users who want dividend
// tracking for manual symbols must record them as Dividend-type orders.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ManualProvider serves market data from the local database for MANUAL data
// source symbols. It is safe for concurrent use — each method creates its own
// repository instances scoped to the request context.
type ManualProvider struct {
	pool   *pgxpool.Pool
	client *http.Client
}

// NewManualProvider creates a new manual (DB-backed) provider.
func NewManualProvider(pool *pgxpool.Pool) *ManualProvider {
	return &ManualProvider{
		pool:   pool,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *ManualProvider) CanHandle(_ string) bool { return true }

func (p *ManualProvider) GetName() model.DataSource { return model.DataSourceManual }

func (p *ManualProvider) GetTestSymbol() string { return "MANUAL" }

func (p *ManualProvider) GetMaxSymbolsPerRequest() int { return 0 }

func (p *ManualProvider) GetDataProviderInfo() DataProviderInfo {
	return DataProviderInfo{
		IsPremium: false,
		Name:      "Manual",
		URL:       "",
	}
}

// GetQuotes fetches the latest stored market price for each symbol from the DB.
// If a symbol has a scraper configuration, tryScraperPrice is called and its
// result replaces the DB price. Symbols with no DB record and no valid scraper
// return a price of 0 rather than an error, matching the NestJS behaviour where
// manual symbols with no data show as zero-priced.
//
// MarketState is always "delayed" because manual data has no real-time feed.
func (p *ManualProvider) GetQuotes(ctx context.Context, symbols []string) (map[string]DataProviderResponse, error) {
	if len(symbols) == 0 {
		return map[string]DataProviderResponse{}, nil
	}

	repo := repository.NewMarketDataRepository(p.pool)
	spRepo := repository.NewSymbolProfileRepository(p.pool)

	result := make(map[string]DataProviderResponse)

	latest, err := repo.FindLatestBySymbols(ctx, string(model.DataSourceManual), symbols)
	if err != nil {
		return nil, fmt.Errorf("manual quotes: %w", err)
	}

	latestBySymbol := make(map[string]model.MarketData)
	for _, md := range latest {
		latestBySymbol[md.Symbol] = md
	}

	for _, sym := range symbols {
		price := 0.0
		if md, ok := latestBySymbol[sym]; ok {
			price = md.MarketPrice
		}

		// Check for scraper configuration with instant mode
		scraperPrice, scraperOk := p.tryScraperPrice(ctx, spRepo, sym)
		if scraperOk {
			price = scraperPrice
		}

		result[sym] = DataProviderResponse{
			DataSource:  string(model.DataSourceManual),
			MarketPrice: price,
			MarketState: "delayed",
		}
	}

	return result, nil
}

// GetHistorical returns historical data from the DB.
func (p *ManualProvider) GetHistorical(ctx context.Context, symbol string, from, to time.Time) (map[string]HistoricalResponse, error) {
	repo := repository.NewMarketDataRepository(p.pool)
	data, err := repo.FindBySymbolAndDateRange(ctx, string(model.DataSourceManual), symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("manual historical: %w", err)
	}

	result := make(map[string]HistoricalResponse)
	for _, md := range data {
		date := md.Date.Format("2006-01-02")
		result[date] = HistoricalResponse{MarketPrice: md.MarketPrice}
	}
	return result, nil
}

// GetDividends returns empty — manual provider has no dividend data.
func (p *ManualProvider) GetDividends(_ context.Context, _ string, _, _ time.Time) (map[string]HistoricalResponse, error) {
	return map[string]HistoricalResponse{}, nil
}

// GetAssetProfile queries the symbol profile from the DB.
func (p *ManualProvider) GetAssetProfile(ctx context.Context, symbol string) (*model.SymbolProfile, error) {
	repo := repository.NewSymbolProfileRepository(p.pool)
	sp, err := repo.FindByDataSourceAndSymbol(ctx, string(model.DataSourceManual), symbol)
	if err != nil {
		return nil, fmt.Errorf("manual asset profile: %w", err)
	}
	return &sp, nil
}

// Search performs an ILIKE search on symbol profiles with MANUAL data source.
func (p *ManualProvider) Search(ctx context.Context, query string, _ bool) (LookupResponse, error) {
	repo := repository.NewSymbolProfileRepository(p.pool)
	profiles, err := repo.SearchByQuery(ctx, string(model.DataSourceManual), query)
	if err != nil {
		return LookupResponse{}, fmt.Errorf("manual search: %w", err)
	}

	items := make([]LookupItem, 0, len(profiles))
	for _, sp := range profiles {
		name := ""
		if sp.Name != nil {
			name = *sp.Name
		}
		item := LookupItem{
			Currency:   sp.Currency,
			DataSource: model.DataSourceManual,
			Name:       name,
			Symbol:     sp.Symbol,
		}
		if sp.AssetClass != nil {
			item.AssetClass = string(*sp.AssetClass)
		}
		if sp.AssetSubClass != nil {
			item.AssetSubClass = string(*sp.AssetSubClass)
		}
		items = append(items, item)
	}

	return LookupResponse{Items: items}, nil
}

// tryScraperPrice attempts to fetch a live price by calling fetchJSONPrice if the
// symbol profile has a valid scraper configuration. Returns (0, false) on any
// failure — missing profile, missing/invalid config, network error, or selector
// error — so the caller falls back to the DB price silently.
//
// SAFETY: All failure modes are non-fatal. A misconfigured scraper must not
// break quote fetching for the rest of the portfolio.
func (p *ManualProvider) tryScraperPrice(ctx context.Context, spRepo *repository.SymbolProfileRepository, symbol string) (float64, bool) {
	sp, err := spRepo.FindByDataSourceAndSymbol(ctx, string(model.DataSourceManual), symbol)
	if err != nil {
		return 0, false
	}

	if len(sp.ScraperConfiguration) == 0 {
		return 0, false
	}

	var scraperCfg scraperConfig
	if err := json.Unmarshal(sp.ScraperConfiguration, &scraperCfg); err != nil {
		return 0, false
	}

	if scraperCfg.URL == "" || scraperCfg.Selector == "" {
		return 0, false
	}

	// Only support JSON mode for now (HTML deferred)
	if scraperCfg.Mode != "" && scraperCfg.Mode != "instant" {
		return 0, false
	}

	price, err := p.fetchJSONPrice(ctx, scraperCfg.URL, scraperCfg.Selector, scraperCfg.Locale)
	if err != nil {
		return 0, false
	}
	return price, true
}

type scraperConfig struct {
	Mode     string `json:"mode"`
	URL      string `json:"url"`
	Selector string `json:"selector"`
	Locale   string `json:"locale"`
}

// fetchJSONPrice GETs rawURL, decodes the JSON response, and extracts a numeric
// price by walking the dot-separated selector path through nested objects.
//
// Example: selector "data.price.last" walks data → price → last.
//
// The locale parameter is forwarded as Accept-Language so locale-sensitive APIs
// return consistently formatted numbers. Supported value types at the leaf node
// are float64, string (parsed as float64), and json.Number.
func (p *ManualProvider) fetchJSONPrice(ctx context.Context, rawURL, selector, locale string) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	if locale != "" {
		req.Header.Set("Accept-Language", locale)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return 0, fmt.Errorf("scraper: HTTP %d", resp.StatusCode)
	}

	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("scraper: JSON parse: %w", err)
	}

	// Walk the dot-path selector
	parts := strings.Split(selector, ".")
	current := data
	for _, part := range parts {
		switch obj := current.(type) {
		case map[string]any:
			current = obj[part]
		default:
			return 0, fmt.Errorf("scraper: cannot navigate selector %q at %q", selector, part)
		}
	}

	switch v := current.(type) {
	case float64:
		return v, nil
	case string:
		return strconv.ParseFloat(v, 64)
	case json.Number:
		return v.Float64()
	default:
		return 0, fmt.Errorf("scraper: unexpected type %T for selector %q", current, selector)
	}
}
