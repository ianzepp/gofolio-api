package service

import (
	"context"
	"encoding/json"
	"os"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InfoService struct {
	pool *pgxpool.Pool
}

func NewInfoService(pool *pgxpool.Pool) *InfoService {
	return &InfoService{pool: pool}
}

type BenchmarkProperty struct {
	SymbolProfileId string `json:"symbolProfileId"`
	EnableSharing   bool   `json:"enableSharing"`
}

type BenchmarkAssetProfile struct {
	DataSource string  `json:"dataSource"`
	ID         string  `json:"id"`
	Name       *string `json:"name"`
	Symbol     string  `json:"symbol"`
}

type InfoResponse struct {
	BaseCurrency      string                  `json:"baseCurrency"`
	Benchmarks        []BenchmarkAssetProfile `json:"benchmarks"`
	Currencies        []string                `json:"currencies"`
	DemoAuthToken     string                  `json:"demoAuthToken"`
	GlobalPermissions []string                `json:"globalPermissions"`
	IsReadOnlyMode    bool                    `json:"isReadOnlyMode,omitempty"`
	Statistics        any                     `json:"statistics"`
}

func (s *InfoService) GetInfo(ctx context.Context) (InfoResponse, error) {
	propRepo := repository.NewPropertyRepository(s.pool)
	spRepo := repository.NewSymbolProfileRepository(s.pool)

	// Build global permissions from env flags
	globalPermissions := []string{}
	if envBool("ENABLE_FEATURE_AUTH_TOKEN") {
		globalPermissions = append(globalPermissions, "enableAuthToken")
	}
	if envBool("ENABLE_FEATURE_STATISTICS") {
		globalPermissions = append(globalPermissions, "enableStatistics")
	}
	if envBool("ENABLE_FEATURE_SUBSCRIPTION") {
		globalPermissions = append(globalPermissions, "enableSubscription")
	}
	if envBool("ENABLE_FEATURE_SYSTEM_MESSAGE") {
		globalPermissions = append(globalPermissions, "enableSystemMessage")
	}
	if envBool("ENABLE_FEATURE_AUTH_GOOGLE") {
		globalPermissions = append(globalPermissions, "enableAuthGoogle")
	}
	if envBool("ENABLE_FEATURE_AUTH_OIDC") {
		globalPermissions = append(globalPermissions, "enableAuthOidc")
	}
	if envBool("ENABLE_FEATURE_FEAR_AND_GREED_INDEX") {
		globalPermissions = append(globalPermissions, "enableFearAndGreedIndex")
	}

	// Check if user signup is enabled via Property table
	isSignupEnabled := true // default
	prop, err := propRepo.FindByKey(ctx, "IS_USER_SIGNUP_ENABLED")
	if err == nil && prop.Value == "false" {
		isSignupEnabled = false
	}
	if isSignupEnabled {
		globalPermissions = append(globalPermissions, "createUserAccount")
	}

	// Load benchmarks from Property table
	benchmarks := []BenchmarkAssetProfile{}
	benchProp, err := propRepo.FindByKey(ctx, "BENCHMARKS")
	if err == nil {
		var benchProps []BenchmarkProperty
		if json.Unmarshal([]byte(benchProp.Value), &benchProps) == nil && len(benchProps) > 0 {
			ids := make([]string, len(benchProps))
			for i, bp := range benchProps {
				ids[i] = bp.SymbolProfileId
			}
			profiles, err := spRepo.FindByIDs(ctx, ids)
			if err == nil {
				for _, p := range profiles {
					benchmarks = append(benchmarks, BenchmarkAssetProfile{
						DataSource: string(p.DataSource),
						ID:         p.ID,
						Name:       p.Name,
						Symbol:     p.Symbol,
					})
				}
			}
		}
	}

	// Load demo auth token from Property table
	demoAuthToken := ""
	demoProp, err := propRepo.FindByKey(ctx, "DEMO_USER_ID")
	if err == nil && demoProp.Value != "" {
		// The legacy API generates a JWT for the demo user.
		// For now, return empty — the demo token requires JWT signing which
		// is handled at the handler/auth layer.
		_ = demoProp.Value
	}

	// Get currencies from SymbolProfile table (distinct currencies)
	currencies := []string{"USD"} // fallback

	return InfoResponse{
		BaseCurrency:      "USD",
		Benchmarks:        benchmarks,
		Currencies:        currencies,
		DemoAuthToken:     demoAuthToken,
		GlobalPermissions: globalPermissions,
		Statistics:        nil,
	}, nil
}

func envBool(key string) bool {
	v := os.Getenv(key)
	return v == "true" || v == "1"
}
