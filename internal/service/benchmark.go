package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ianzepp/gofolio-api/internal/model"
	"github.com/ianzepp/gofolio-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

// BenchmarkService returns performance summaries for the market indices or
// assets that an admin has designated as benchmarks. Benchmarks give users
// a reference point (e.g. S&P 500) against which to compare their portfolio.
type BenchmarkService struct {
	db *sqlx.DB
}

// NewBenchmarkService constructs a BenchmarkService backed by the given pool.
func NewBenchmarkService(db *sqlx.DB) *BenchmarkService {
	return &BenchmarkService{db: db}
}

// BenchmarkItem is a single benchmark entry in the GET /api/v1/benchmark response.
// MarketCondition, MarketChange, Trend50d, and Trend200d are currently stubbed to
// neutral/zero values — full computation requires historical market data ingestion
// that is not yet implemented in the Go port. The stubs maintain JSON parity with
// the NestJS response shape so that the frontend can render without modification.
type BenchmarkItem struct {
	DataSource      string  `json:"dataSource"`
	MarketCondition string  `json:"marketCondition"`
	MarketChange    float64 `json:"marketChange"`
	Name            *string `json:"name"`
	Performances    any     `json:"performances"`
	Symbol          string  `json:"symbol"`
	Trend50d        string  `json:"trend50d"`
	Trend200d       string  `json:"trend200d"`
	AllTimeHigh     any     `json:"allTimeHigh,omitempty"`
}

// BenchmarkResponse is the top-level wrapper for the benchmark list endpoint.
type BenchmarkResponse struct {
	Benchmarks []BenchmarkItem `json:"benchmarks"`
}

// GetBenchmarks loads the admin-configured benchmark symbols from the BENCHMARKS
// Property row, resolves their SymbolProfiles, and returns a list of stub items.
// Errors at any stage (missing property, bad JSON, unknown profiles) return an
// empty list rather than an error, because a misconfigured benchmark setting
// should not degrade the rest of the application.
func (s *BenchmarkService) GetBenchmarks(ctx context.Context) (BenchmarkResponse, error) {
	propRepo := repository.NewPropertyRepository(s.db)
	spRepo := repository.NewSymbolProfileRepository(s.db)

	benchProp, err := propRepo.FindByKey(ctx, "BENCHMARKS")
	if err != nil {
		return BenchmarkResponse{Benchmarks: []BenchmarkItem{}}, nil
	}

	var benchProps []BenchmarkProperty
	if err := json.Unmarshal([]byte(benchProp.Value), &benchProps); err != nil || len(benchProps) == 0 {
		return BenchmarkResponse{Benchmarks: []BenchmarkItem{}}, nil
	}

	ids := make([]string, len(benchProps))
	for i, bp := range benchProps {
		ids[i] = bp.SymbolProfileId
	}

	profiles, err := spRepo.FindByIDs(ctx, ids)
	if err != nil {
		return BenchmarkResponse{Benchmarks: []BenchmarkItem{}}, nil
	}

	items := make([]BenchmarkItem, 0, len(profiles))
	for _, p := range profiles {
		items = append(items, BenchmarkItem{
			DataSource:      string(p.DataSource),
			MarketCondition: "NEUTRAL_MARKET",
			MarketChange:    0,
			Name:            p.Name,
			Performances:    map[string]any{},
			Symbol:          p.Symbol,
			Trend50d:        "NEUTRAL",
			Trend200d:       "NEUTRAL",
		})
	}

	return BenchmarkResponse{Benchmarks: items}, nil
}

// AddBenchmark appends a new benchmark to the BENCHMARKS Property JSON array.
func (s *BenchmarkService) AddBenchmark(ctx context.Context, dataSource, symbol string) error {
	spRepo := repository.NewSymbolProfileRepository(s.db)
	propRepo := repository.NewPropertyRepository(s.db)

	profile, err := spRepo.FindByDataSourceAndSymbol(ctx, dataSource, symbol)
	if err != nil {
		return fmt.Errorf("symbol profile not found: %w", err)
	}

	benchProps, err := s.loadBenchmarkProps(ctx, propRepo)
	if err != nil {
		benchProps = []BenchmarkProperty{}
	}

	// Check for duplicate
	for _, bp := range benchProps {
		if bp.SymbolProfileId == profile.ID {
			return nil // already exists
		}
	}

	benchProps = append(benchProps, BenchmarkProperty{
		SymbolProfileId: profile.ID,
		EnableSharing:   false,
	})

	return s.saveBenchmarkProps(ctx, propRepo, benchProps)
}

// DeleteBenchmark removes a benchmark from the BENCHMARKS Property JSON array.
func (s *BenchmarkService) DeleteBenchmark(ctx context.Context, dataSource, symbol string) error {
	spRepo := repository.NewSymbolProfileRepository(s.db)
	propRepo := repository.NewPropertyRepository(s.db)

	profile, err := spRepo.FindByDataSourceAndSymbol(ctx, dataSource, symbol)
	if err != nil {
		return fmt.Errorf("symbol profile not found: %w", err)
	}

	benchProps, err := s.loadBenchmarkProps(ctx, propRepo)
	if err != nil {
		return nil // nothing to delete
	}

	filtered := make([]BenchmarkProperty, 0, len(benchProps))
	for _, bp := range benchProps {
		if bp.SymbolProfileId != profile.ID {
			filtered = append(filtered, bp)
		}
	}

	return s.saveBenchmarkProps(ctx, propRepo, filtered)
}

func (s *BenchmarkService) loadBenchmarkProps(ctx context.Context, propRepo *repository.PropertyRepository) ([]BenchmarkProperty, error) {
	prop, err := propRepo.FindByKey(ctx, "BENCHMARKS")
	if err != nil {
		return nil, err
	}
	var items []BenchmarkProperty
	if err := json.Unmarshal([]byte(prop.Value), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *BenchmarkService) saveBenchmarkProps(ctx context.Context, propRepo *repository.PropertyRepository, items []BenchmarkProperty) error {
	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal benchmarks: %w", err)
	}
	_, err = propRepo.Upsert(ctx, model.Property{Key: "BENCHMARKS", Value: string(data)})
	return err
}
