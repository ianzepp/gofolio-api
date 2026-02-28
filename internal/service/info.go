package service

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"

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
	if featureEnabled("ENABLE_FEATURE_AUTH_TOKEN", true) {
		globalPermissions = append(globalPermissions, "enableAuthToken")
	}
	if featureEnabled("ENABLE_FEATURE_STATISTICS", false) {
		globalPermissions = append(globalPermissions, "enableStatistics")
	}
	if featureEnabled("ENABLE_FEATURE_SUBSCRIPTION", false) {
		globalPermissions = append(globalPermissions, "enableSubscription")
	}
	if featureEnabled("ENABLE_FEATURE_SYSTEM_MESSAGE", false) {
		globalPermissions = append(globalPermissions, "enableSystemMessage")
	}
	if featureEnabled("ENABLE_FEATURE_AUTH_GOOGLE", false) {
		globalPermissions = append(globalPermissions, "enableAuthGoogle")
	}
	if featureEnabled("ENABLE_FEATURE_AUTH_OIDC", false) {
		globalPermissions = append(globalPermissions, "enableAuthOidc")
	}
	if featureEnabled("ENABLE_FEATURE_FEAR_AND_GREED_INDEX", false) {
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

	// Load currencies from Property table (legacy key: CURRENCIES), fallback to USD.
	currencies := []string{"USD", "USX"}
	if currenciesProp, err := propRepo.FindByKey(ctx, "CURRENCIES"); err == nil {
		var parsed []string
		if json.Unmarshal([]byte(currenciesProp.Value), &parsed) == nil && len(parsed) > 0 {
			combined := append([]string{}, currencies...)
			combined = append(combined, parsed...)
			currencies = uniqueSortedNonEmpty(combined)
		}
	}

	return InfoResponse{
		BaseCurrency:      "USD",
		Benchmarks:        benchmarks,
		Currencies:        currencies,
		DemoAuthToken:     demoAuthToken,
		GlobalPermissions: globalPermissions,
		Statistics:        nil,
	}, nil
}

func featureEnabled(key string, defaultValue bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func uniqueSortedNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	// Keep ordering deterministic for JSON parity checks.
	sort.Strings(out)
	return out
}
