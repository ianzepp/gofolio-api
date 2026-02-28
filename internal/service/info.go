package service

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jmoiron/sqlx"
)

// InfoService assembles the public instance metadata returned by
// GET /api/v1/info. This endpoint is unauthenticated — it is used by the
// frontend to decide which features and auth methods to render before a user
// has logged in, so it must never return sensitive data.
type InfoService struct {
	db *sqlx.DB
}

// NewInfoService constructs an InfoService backed by the given pool.
func NewInfoService(db *sqlx.DB) *InfoService {
	return &InfoService{db: db}
}

// BenchmarkProperty is the stored shape of a single benchmark entry in the
// BENCHMARKS Property row. EnableSharing controls whether the benchmark
// appears in public portfolio views.
type BenchmarkProperty struct {
	SymbolProfileId string `json:"symbolProfileId"`
	EnableSharing   bool   `json:"enableSharing"`
}

// BenchmarkAssetProfile is the public-safe projection of a SymbolProfile used
// in the info response. It omits financial data and PII so that this endpoint
// can remain unauthenticated.
type BenchmarkAssetProfile struct {
	DataSource string  `json:"dataSource"`
	ID         string  `json:"id"`
	Name       *string `json:"name"`
	Symbol     string  `json:"symbol"`
}

// InfoResponse is the payload returned by GET /api/v1/info. GlobalPermissions
// drives feature-flag rendering on the client: the frontend inspects this list
// to conditionally show auth buttons, subscription flows, system messages, etc.
// Keeping permissions as a string slice (rather than a bitmask or map) preserves
// JSON parity with the NestJS implementation and keeps the contract additive.
type InfoResponse struct {
	BaseCurrency      string                  `json:"baseCurrency"`
	Benchmarks        []BenchmarkAssetProfile `json:"benchmarks"`
	Currencies        []string                `json:"currencies"`
	DemoAuthToken     string                  `json:"demoAuthToken"`
	GlobalPermissions []string                `json:"globalPermissions"`
	IsReadOnlyMode    bool                    `json:"isReadOnlyMode,omitempty"`
	Statistics        any                     `json:"statistics"`
}

// GetInfo assembles the public instance metadata. Feature flags are read from
// environment variables so that deployments can toggle capabilities without a
// database migration. Database-backed state (signup enabled, benchmarks,
// currencies) is layered on top.
//
// PHASE 1: Resolve feature flags from env to build GlobalPermissions.
// PHASE 2: Check IS_USER_SIGNUP_ENABLED property; append createUserAccount
//
//	if signup is permitted (default on when property is absent).
//
// PHASE 3: Load BENCHMARKS property and resolve SymbolProfile IDs to names.
// PHASE 4: Merge hardcoded base currencies with CURRENCIES property values,
//
//	deduplicating and sorting for deterministic JSON output.
//
// NOTE: Demo auth token generation is deferred to the handler/auth layer since
// it requires JWT signing keys that are not available in this service.
func (s *InfoService) GetInfo(ctx context.Context) (InfoResponse, error) {
	propRepo := repository.NewPropertyRepository(s.db)
	spRepo := repository.NewSymbolProfileRepository(s.db)

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

	// Load currencies from Property table (legacy key: CURRENCIES). Legacy base
	// list includes CHF alongside USD/USX.
	currencies := []string{"CHF", "USD", "USX"}
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

// featureEnabled reads an environment variable and interprets it as a boolean.
// An absent variable falls back to defaultValue, allowing both opt-in and
// opt-out feature flags. Accepted truthy values: "1", "true", "yes", "on".
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

// uniqueSortedNonEmpty deduplicates, trims, and sorts a string slice, omitting
// blanks. It is used to produce a stable, canonical currency list so that
// JSON parity checks do not fail due to ordering differences between runs.
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
