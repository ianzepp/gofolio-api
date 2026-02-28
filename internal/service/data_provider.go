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

// DataProviderService orchestrates calls to data providers.
type DataProviderService struct {
	pool      *pgxpool.Pool
	providers map[model.DataSource]provider.DataProvider
}

// NewDataProviderService creates a new data provider orchestrator.
func NewDataProviderService(pool *pgxpool.Pool, providers []provider.DataProvider) *DataProviderService {
	m := make(map[model.DataSource]provider.DataProvider, len(providers))
	for _, p := range providers {
		m[p.GetName()] = p
	}
	return &DataProviderService{pool: pool, providers: m}
}

// GetQuotes fetches live quotes for items, grouped by data source.
// It upserts the results into the MarketData table with INTRADAY state.
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

// GetHistorical fetches historical data from the database (DB-first path).
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

// GetHistoricalRaw fetches historical data directly from providers (bypasses DB).
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

// GetDividends fetches dividend data from a provider.
func (s *DataProviderService) GetDividends(ctx context.Context, dataSource model.DataSource, symbol string, from, to time.Time) (map[string]provider.HistoricalResponse, error) {
	p, ok := s.providers[dataSource]
	if !ok {
		return nil, fmt.Errorf("no provider for data source %s", dataSource)
	}
	return p.GetDividends(ctx, symbol, from, to)
}

// Search fans out a query to all providers and merges results sorted by name.
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

// GetAssetProfiles fetches asset profiles from providers.
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

// GetProvider returns the provider for a data source, if registered.
func (s *DataProviderService) GetProvider(ds model.DataSource) (provider.DataProvider, bool) {
	p, ok := s.providers[ds]
	return p, ok
}

// batchSymbols splits symbols into batches of maxSize. If maxSize <= 0, returns one batch.
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
