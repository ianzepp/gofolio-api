package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/provider"
	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

// DataProviderService orchestrates calls to external data providers (Yahoo Finance,
// Manual, etc.). It is the single integration point between the service layer and
// the provider abstraction, hiding fan-out, batching, concurrency, and DB-write
// concerns from callers.
//
// TRADE-OFF: Providers are registered at startup as a flat map keyed by DataSource.
// This makes dispatch O(1) and avoids reflection, at the cost of requiring an
// explicit registration step in the composition root.
type DataProviderService struct {
	pool      *pgxpool.Pool
	providers map[model.DataSource]provider.DataProvider
}

// NewDataProviderService builds a DataProviderService from a slice of providers,
// indexing them by their DataSource name for O(1) dispatch. Duplicate DataSource
// registrations are silently overwritten — last writer wins.
func NewDataProviderService(pool *pgxpool.Pool, providers []provider.DataProvider) *DataProviderService {
	m := make(map[model.DataSource]provider.DataProvider, len(providers))
	for _, p := range providers {
		m[p.GetName()] = p
	}
	return &DataProviderService{pool: pool, providers: m}
}

// GetQuotes fetches live quotes for a set of asset identifiers concurrently,
// then persists each result to the MarketData table so that historical queries
// can read from the DB without hitting the external provider again.
//
// PHASE 1: Group identifiers by DataSource to minimise round-trips.
// PHASE 2: Split each source's symbols into provider-sized batches and dispatch
//
//	concurrent requests via errgroup.
//
// PHASE 3: Upsert successful quotes into MarketData with INTRADAY state so that
//
//	today's price is available for historical range queries.
//
// EDGE: Items whose provider is not registered are silently skipped rather than
// returning an error, since a single missing integration should not block all
// other quotes in the same request.
func (s *DataProviderService) GetQuotes(ctx context.Context, items []provider.AssetProfileIdentifier) (map[string]provider.DataProviderResponse, error) {
	// Group by data source
	grouped := make(map[model.DataSource][]string)
	for _, item := range items {
		grouped[item.DataSource] = append(grouped[item.DataSource], item.Symbol)
	}

	result := make(map[string]provider.DataProviderResponse)
	g, gCtx := errgroup.WithContext(ctx)

	type batchResult struct {
		quotes map[string]provider.DataProviderResponse
	}

	var mu = make(chan batchResult, len(items))

	for ds, symbols := range grouped {
		p, ok := s.providers[ds]
		if !ok {
			continue
		}

		maxPerReq := p.GetMaxSymbolsPerRequest()
		batches := batchSymbols(symbols, maxPerReq)

		for _, batch := range batches {
			batch := batch
			p := p
			g.Go(func() error {
				quotes, err := p.GetQuotes(gCtx, batch)
				if err != nil {
					return err
				}
				mu <- batchResult{quotes: quotes}
				return nil
			})
		}
	}

	// Wait in a separate goroutine so we can collect results
	errCh := make(chan error, 1)
	go func() {
		errCh <- g.Wait()
		close(mu)
	}()

	for br := range mu {
		for k, v := range br.quotes {
			result[k] = v
		}
	}

	if err := <-errCh; err != nil {
		return result, fmt.Errorf("get quotes: %w", err)
	}

	// Upsert to MarketData
	if s.pool != nil {
		repo := repository.NewMarketDataRepository(s.pool)
		today := time.Now().UTC().Truncate(24 * time.Hour)
		for _, item := range items {
			if q, ok := result[item.Symbol]; ok && q.MarketPrice > 0 {
				_ = repo.UpsertMarketPrice(ctx, string(item.DataSource), item.Symbol, today, q.MarketPrice, model.MarketDataStateIntraday)
			}
		}
	}

	return result, nil
}

// GetHistorical fetches historical price data for a symbol from the local
// MarketData table. This is the DB-first path — it assumes that a prior
// quote-fetch or data-import job has already populated the table for the
// requested range. Callers that need live provider data should use
// GetHistoricalRaw instead.
func (s *DataProviderService) GetHistorical(ctx context.Context, dataSource model.DataSource, symbol string, from, to time.Time) (map[string]provider.HistoricalResponse, error) {
	if s.pool == nil {
		return nil, fmt.Errorf("database not available")
	}
	repo := repository.NewMarketDataRepository(s.pool)
	data, err := repo.FindBySymbolAndDateRange(ctx, string(dataSource), symbol, from, to)
	if err != nil {
		return nil, fmt.Errorf("historical: %w", err)
	}

	result := make(map[string]provider.HistoricalResponse, len(data))
	for _, md := range data {
		date := md.Date.Format("2006-01-02")
		result[date] = provider.HistoricalResponse{MarketPrice: md.MarketPrice}
	}
	return result, nil
}

// GetHistoricalRaw fetches historical price data concurrently from the external
// providers, bypassing the local MarketData table entirely. It is used when
// freshness matters (e.g. the symbol/date endpoint) or when the DB cache has
// not yet been populated.
//
// TRADE-OFF: Going direct to providers is slower and consumes API quota, but
// guarantees the most up-to-date series for any requested date range.
func (s *DataProviderService) GetHistoricalRaw(ctx context.Context, items []provider.AssetProfileIdentifier, from, to time.Time) (map[string]map[string]provider.HistoricalResponse, error) {
	result := make(map[string]map[string]provider.HistoricalResponse)
	g, gCtx := errgroup.WithContext(ctx)

	type itemResult struct {
		symbol string
		data   map[string]provider.HistoricalResponse
	}
	ch := make(chan itemResult, len(items))

	for _, item := range items {
		item := item
		p, ok := s.providers[item.DataSource]
		if !ok {
			continue
		}
		g.Go(func() error {
			data, err := p.GetHistorical(gCtx, item.Symbol, from, to)
			if err != nil {
				return err
			}
			ch <- itemResult{symbol: item.Symbol, data: data}
			return nil
		})
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- g.Wait()
		close(ch)
	}()

	for ir := range ch {
		result[ir.symbol] = ir.data
	}

	if err := <-errCh; err != nil {
		return result, fmt.Errorf("historical raw: %w", err)
	}
	return result, nil
}

// GetDividends fetches dividend events for a symbol directly from the named
// provider. Unlike GetHistorical, dividends are not cached in the MarketData
// table, so this always reaches out to the external source.
func (s *DataProviderService) GetDividends(ctx context.Context, dataSource model.DataSource, symbol string, from, to time.Time) (map[string]provider.HistoricalResponse, error) {
	p, ok := s.providers[dataSource]
	if !ok {
		return nil, fmt.Errorf("no provider for data source %s", dataSource)
	}
	return p.GetDividends(ctx, symbol, from, to)
}

// Search fans out a symbol lookup query to every registered provider
// concurrently and merges their results into a single list sorted by name.
// Individual provider errors are non-fatal — a failing provider is skipped so
// that a single integration outage does not block the search feature entirely.
//
// WHY: Users should be able to add positions from any provider, and a global
// search makes provider selection transparent at the UI layer.
func (s *DataProviderService) Search(ctx context.Context, query string, includeIndices bool) (provider.LookupResponse, error) {
	g, gCtx := errgroup.WithContext(ctx)
	ch := make(chan []provider.LookupItem, len(s.providers))

	for _, p := range s.providers {
		p := p
		g.Go(func() error {
			resp, err := p.Search(gCtx, query, includeIndices)
			if err != nil {
				// Non-fatal: skip this provider
				return nil
			}
			ch <- resp.Items
			return nil
		})
	}

	go func() {
		g.Wait()
		close(ch)
	}()

	var allItems []provider.LookupItem
	for items := range ch {
		allItems = append(allItems, items...)
	}

	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Name < allItems[j].Name
	})

	return provider.LookupResponse{Items: allItems}, nil
}

// GetAssetProfiles fetches rich asset metadata (name, asset class, currency,
// sector weights, etc.) from providers concurrently. Failed lookups are
// non-fatal and simply absent from the returned map, allowing partial results
// to be used by callers that can tolerate missing profiles.
//
// WHY: Asset profiles are fetched on-demand during import or enrichment rather
// than eagerly, so a batch-concurrent fetch keeps latency manageable.
func (s *DataProviderService) GetAssetProfiles(ctx context.Context, items []provider.AssetProfileIdentifier) (map[string]*model.SymbolProfile, error) {
	result := make(map[string]*model.SymbolProfile)
	g, gCtx := errgroup.WithContext(ctx)

	type profileResult struct {
		symbol  string
		profile *model.SymbolProfile
	}
	ch := make(chan profileResult, len(items))

	for _, item := range items {
		item := item
		p, ok := s.providers[item.DataSource]
		if !ok {
			continue
		}
		g.Go(func() error {
			profile, err := p.GetAssetProfile(gCtx, item.Symbol)
			if err != nil {
				return nil // non-fatal
			}
			ch <- profileResult{symbol: item.Symbol, profile: profile}
			return nil
		})
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- g.Wait()
		close(ch)
	}()

	for pr := range ch {
		result[pr.symbol] = pr.profile
	}

	if err := <-errCh; err != nil {
		return result, fmt.Errorf("asset profiles: %w", err)
	}
	return result, nil
}

// GetProvider returns the registered DataProvider for a given DataSource, or
// false if none has been registered. Callers use this for conditional logic
// that depends on whether a particular integration is available at runtime.
func (s *DataProviderService) GetProvider(ds model.DataSource) (provider.DataProvider, bool) {
	p, ok := s.providers[ds]
	return p, ok
}

// batchSymbols partitions a symbol list into sub-slices of at most maxSize
// elements, which are then dispatched as separate API requests. This is
// necessary because providers enforce per-request symbol limits (e.g. Yahoo
// Finance allows up to ~1500 symbols per call). If maxSize <= 0 the entire
// slice is returned as a single batch, acting as "no limit".
func batchSymbols(symbols []string, maxSize int) [][]string {
	if maxSize <= 0 || len(symbols) <= maxSize {
		return [][]string{symbols}
	}
	var batches [][]string
	for i := 0; i < len(symbols); i += maxSize {
		end := i + maxSize
		if end > len(symbols) {
			end = len(symbols)
		}
		batches = append(batches, symbols[i:end])
	}
	return batches
}
