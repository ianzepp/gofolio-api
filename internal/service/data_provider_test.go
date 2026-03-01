package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/provider"
)

type fakeProvider struct {
	name       model.DataSource
	maxBatch   int
	quotes     map[string]provider.DataProviderResponse
	quoteErr   error
	quoteCalls [][]string
	mu         sync.Mutex
}

func (f *fakeProvider) CanHandle(symbol string) bool { return true }

func (f *fakeProvider) GetName() model.DataSource { return f.name }

func (f *fakeProvider) GetTestSymbol() string { return "TEST" }

func (f *fakeProvider) GetMaxSymbolsPerRequest() int { return f.maxBatch }

func (f *fakeProvider) GetDataProviderInfo() provider.DataProviderInfo {
	return provider.DataProviderInfo{Name: string(f.name)}
}

func (f *fakeProvider) GetQuotes(ctx context.Context, symbols []string) (map[string]provider.DataProviderResponse, error) {
	f.mu.Lock()
	f.quoteCalls = append(f.quoteCalls, append([]string(nil), symbols...))
	f.mu.Unlock()
	if f.quoteErr != nil {
		return nil, f.quoteErr
	}
	result := make(map[string]provider.DataProviderResponse, len(symbols))
	for _, s := range symbols {
		if q, ok := f.quotes[s]; ok {
			result[s] = q
		}
	}
	return result, nil
}

func (f *fakeProvider) GetHistorical(ctx context.Context, symbol string, from, to time.Time) (map[string]provider.HistoricalResponse, error) {
	return map[string]provider.HistoricalResponse{}, nil
}

func (f *fakeProvider) GetDividends(ctx context.Context, symbol string, from, to time.Time) (map[string]provider.HistoricalResponse, error) {
	return map[string]provider.HistoricalResponse{}, nil
}

func (f *fakeProvider) GetAssetProfile(ctx context.Context, symbol string) (*model.SymbolProfile, error) {
	return nil, nil
}

func (f *fakeProvider) Search(ctx context.Context, query string, includeIndices bool) (provider.LookupResponse, error) {
	return provider.LookupResponse{}, nil
}

func (f *fakeProvider) GetHistoricalRaw(ctx context.Context, symbol string, from, to time.Time) (map[string]provider.HistoricalResponse, error) {
	return map[string]provider.HistoricalResponse{}, nil
}

func TestDataProviderService_GetQuotes_HappyPath(t *testing.T) {
	yahoo := &fakeProvider{
		name:     model.DataSourceYahoo,
		maxBatch: 1,
		quotes: map[string]provider.DataProviderResponse{
			"AAPL": {Symbol: "AAPL", MarketPrice: 150, DataSource: string(model.DataSourceYahoo)},
			"MSFT": {Symbol: "MSFT", MarketPrice: 300, DataSource: string(model.DataSourceYahoo)},
		},
	}
	manual := &fakeProvider{
		name:   model.DataSourceManual,
		quotes: map[string]provider.DataProviderResponse{"CASH": {Symbol: "CASH", MarketPrice: 1, DataSource: string(model.DataSourceManual)}},
	}

	svc := NewDataProviderService(nil, []provider.DataProvider{yahoo, manual})
	ctx := context.Background()
	items := []provider.AssetProfileIdentifier{
		{DataSource: model.DataSourceYahoo, Symbol: "AAPL"},
		{DataSource: model.DataSourceYahoo, Symbol: "MSFT"},
		{DataSource: model.DataSourceManual, Symbol: "CASH"},
	}

	result, err := svc.GetQuotes(ctx, items)
	if err != nil {
		t.Fatalf("GetQuotes returned error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 quotes, got %d", len(result))
	}
	if result["AAPL"].MarketPrice != 150 {
		t.Errorf("expected AAPL price 150, got %v", result["AAPL"].MarketPrice)
	}
	if result["MSFT"].MarketPrice != 300 {
		t.Errorf("expected MSFT price 300, got %v", result["MSFT"].MarketPrice)
	}
	if result["CASH"].MarketPrice != 1 {
		t.Errorf("expected CASH price 1, got %v", result["CASH"].MarketPrice)
	}

	if len(yahoo.quoteCalls) != 2 {
		t.Fatalf("expected yahoo to be called twice due to batching, got %d", len(yahoo.quoteCalls))
	}
	if len(manual.quoteCalls) != 1 {
		t.Fatalf("expected manual to be called once, got %d", len(manual.quoteCalls))
	}
}

func TestDataProviderService_GetQuotes_SkipsMissingProviders(t *testing.T) {
	svc := NewDataProviderService(nil, []provider.DataProvider{
		&fakeProvider{
			name: model.DataSourceManual,
			quotes: map[string]provider.DataProviderResponse{
				"CASH": {Symbol: "CASH", MarketPrice: 1},
			},
		},
	})
	items := []provider.AssetProfileIdentifier{
		{DataSource: model.DataSourceYahoo, Symbol: "AAPL"},
		{DataSource: model.DataSourceManual, Symbol: "CASH"},
	}
	result, err := svc.GetQuotes(context.Background(), items)
	if err != nil {
		t.Fatalf("GetQuotes returned error: %v", err)
	}
	if len(result) != 1 || result["CASH"].MarketPrice != 1 {
		t.Fatalf("expected only CASH quote, got %v", result)
	}
}

func TestDataProviderService_GetQuotes_PropagatesProviderError(t *testing.T) {
	sentinel := errors.New("provider boom")
	yahoo := &fakeProvider{
		name:     model.DataSourceYahoo,
		maxBatch: 1,
		quoteErr: sentinel,
	}
	svc := NewDataProviderService(nil, []provider.DataProvider{yahoo})
	_, err := svc.GetQuotes(context.Background(), []provider.AssetProfileIdentifier{{DataSource: model.DataSourceYahoo, Symbol: "AAPL"}})
	if err == nil || !errors.Is(err, sentinel) {
		t.Fatalf("expected provider error, got %v", err)
	}
}

func TestBatchSymbolsNoLimit(t *testing.T) {
	symbols := []string{"AAPL", "MSFT"}
	batches := batchSymbols(symbols, 0)
	if len(batches) != 1 || len(batches[0]) != 2 {
		t.Fatalf("expected single batch with original symbols, got %v", batches)
	}
}

func TestBatchSymbolsWithLimit(t *testing.T) {
	symbols := []string{"AAPL", "MSFT", "GOOG"}
	batches := batchSymbols(symbols, 2)
	if len(batches) != 2 {
		t.Fatalf("expected two batches, got %d", len(batches))
	}
	if len(batches[0]) != 2 || len(batches[1]) != 1 {
		t.Fatalf("unexpected batch sizes: %v", batches)
	}
}
