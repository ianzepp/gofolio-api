package service

import (
	"context"
	"encoding/json"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BenchmarkService struct {
	pool *pgxpool.Pool
}

func NewBenchmarkService(pool *pgxpool.Pool) *BenchmarkService {
	return &BenchmarkService{pool: pool}
}

type BenchmarkItem struct {
	DataSource    string  `json:"dataSource"`
	MarketChange  float64 `json:"marketChange"`
	Name          *string `json:"name"`
	Performances  any     `json:"performances"`
	Symbol        string  `json:"symbol"`
	Trend50d      string  `json:"trend50d"`
	Trend200d     string  `json:"trend200d"`
	AllTimeHigh   any     `json:"allTimeHigh,omitempty"`
}

type BenchmarkResponse struct {
	Benchmarks []BenchmarkItem `json:"benchmarks"`
}

func (s *BenchmarkService) GetBenchmarks(ctx context.Context) (BenchmarkResponse, error) {
	propRepo := repository.NewPropertyRepository(s.pool)
	spRepo := repository.NewSymbolProfileRepository(s.pool)

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
			DataSource:   string(p.DataSource),
			MarketChange: 0,
			Name:         p.Name,
			Performances: map[string]any{},
			Symbol:       p.Symbol,
			Trend50d:     "NEUTRAL",
			Trend200d:    "NEUTRAL",
		})
	}

	return BenchmarkResponse{Benchmarks: items}, nil
}
