// permissions.go defines the role-based access control model for the Go API
// server. It is a direct port of the TypeScript permission system located in
// libs/common/src/lib/permissions.ts (getPermissions() function).
//
// Architecture: permissions are assigned per-role at startup as static maps.
// This is intentionally simple — there are no dynamic permission grants,
// group memberships, or resource-level ACLs. The trade-off is that adding or
// revoking a permission for an individual user requires a role change, not a
// targeted permission update. This matches the NestJS implementation.
//
// Roles form a strict hierarchy in terms of breadth: ADMIN > USER > DEMO > INACTIVE.
// DEMO users can create a new account but cannot perform any portfolio operations.
// INACTIVE users have no permissions and cannot access any protected endpoints.
//
// IMPORTANT: When the NestJS permissions.ts changes (e.g., a new feature is
// gated behind a permission), this file must be updated in lockstep. Divergence
// here will cause the Go server to enforce different access rules than NestJS.
package auth

import "github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"

// Permission constants matching libs/common/src/lib/permissions.ts.
// Each constant is the string literal used as a key in the rolePermissions map
// and expected by middleware callers (e.g., RequirePermission(cfg, CreateOrder, …)).
// String values must match exactly — they are compared directly against the
// TypeScript source and are part of the public API contract between client and server.
const (
	AccessAdminControl                = "accessAdminControl"
	AccessAssistant                   = "accessAssistant"
	AccessHoldingsChart               = "accessHoldingsChart"
	CreateAccess                      = "createAccess"
	CreateAccount                     = "createAccount"
	CreateAccountBalance              = "createAccountBalance"
	CreateApiKey                      = "createApiKey"
	CreateMarketData                  = "createMarketData"
	CreateMarketDataOfOwnAssetProfile = "createMarketDataOfOwnAssetProfile"
	CreateOrder                       = "createOrder"
	CreateOwnTag                      = "createOwnTag"
	CreatePlatform                    = "createPlatform"
	CreateTag                         = "createTag"
	CreateUserAccount                 = "createUserAccount"
	CreateWatchlistItem               = "createWatchlistItem"
	DeleteAccess                      = "deleteAccess"
	DeleteAccount                     = "deleteAccount"
	DeleteAccountBalance              = "deleteAccountBalance"
	DeleteAuthDevice                  = "deleteAuthDevice"
	DeleteOrder                       = "deleteOrder"
	DeleteOwnUser                     = "deleteOwnUser"
	DeletePlatform                    = "deletePlatform"
	DeleteTag                         = "deleteTag"
	DeleteUser                        = "deleteUser"
	DeleteWatchlistItem               = "deleteWatchlistItem"
	EnableAuthGoogle                  = "enableAuthGoogle"
	EnableAuthOidc                    = "enableAuthOidc"
	EnableAuthToken                   = "enableAuthToken"
	EnableDataProviderGhostfolio      = "enableDataProviderGhostfolio"
	EnableFearAndGreedIndex           = "enableFearAndGreedIndex"
	EnableImport                      = "enableImport"
	EnableBlog                        = "enableBlog"
	EnableStatistics                  = "enableStatistics"
	EnableSubscription                = "enableSubscription"
	EnableSubscriptionInterstitial    = "enableSubscriptionInterstitial"
	EnableSystemMessage               = "enableSystemMessage"
	ImpersonateAllUsers               = "impersonateAllUsers"
	ReadAiPrompt                      = "readAiPrompt"
	ReadMarketData                    = "readMarketData"
	ReadMarketDataOfMarkets           = "readMarketDataOfMarkets"
	ReadMarketDataOfOwnAssetProfile   = "readMarketDataOfOwnAssetProfile"
	ReadPlatforms                     = "readPlatforms"
	ReadPlatformsWithAccountCount     = "readPlatformsWithAccountCount"
	ReadTags                          = "readTags"
	ReadWatchlist                     = "readWatchlist"
	ReportDataGlitch                  = "reportDataGlitch"
	SyncDemoUserAccount               = "syncDemoUserAccount"
	ToggleReadOnlyMode                = "toggleReadOnlyMode"
	UpdateAccount                     = "updateAccount"
	UpdateAccess                      = "updateAccess"
	UpdateAuthDevice                  = "updateAuthDevice"
	UpdateMarketData                  = "updateMarketData"
	UpdateMarketDataOfOwnAssetProfile = "updateMarketDataOfOwnAssetProfile"
	UpdateOrder                       = "updateOrder"
	UpdateOwnAccessToken              = "updateOwnAccessToken"
	UpdatePlatform                    = "updatePlatform"
	UpdateTag                         = "updateTag"
	UpdateUserSettings                = "updateUserSettings"
	UpdateViewMode                    = "updateViewMode"
)

// rolePermissions maps each role to its granted permissions, represented as a
// boolean-valued map for O(1) lookup. The set is built at package init time and
// is never mutated; concurrent reads are safe without synchronization.
//
// Source of truth: libs/common/src/lib/permissions.ts getPermissions().
// Notable differences between roles:
//   - ADMIN gains CreateTag, DeleteTag, UpdateTag, DeleteUser, ImpersonateAllUsers,
//     ReadPlatformsWithAccountCount, and admin control panel access vs USER.
//   - USER has DeleteOwnUser and CreateApiKey (self-service); ADMIN does not need
//     these because admins manage other users, not their own account lifecycle.
//   - DEMO is deliberately limited to account creation + read-only views to support
//     the hosted demo instance without allowing data mutation.
//   - INACTIVE has no permissions; it is a soft-disable mechanism.
var rolePermissions = map[model.Role]map[string]bool{
	model.RoleAdmin: {
		AccessAdminControl:                true,
		AccessAssistant:                   true,
		AccessHoldingsChart:               true,
		CreateAccess:                      true,
		CreateAccount:                     true,
		CreateAccountBalance:              true,
		CreateMarketData:                  true,
		CreateMarketDataOfOwnAssetProfile: true,
		CreateOrder:                       true,
		CreateOwnTag:                      true,
		CreatePlatform:                    true,
		CreateTag:                         true,
		CreateWatchlistItem:               true,
		DeleteAccess:                      true,
		DeleteAccount:                     true,
		DeleteAccountBalance:              true,
		DeleteAuthDevice:                  true,
		DeleteOrder:                       true,
		DeletePlatform:                    true,
		DeleteTag:                         true,
		DeleteUser:                        true,
		DeleteWatchlistItem:               true,
		EnableDataProviderGhostfolio:      true,
		ReadAiPrompt:                      true,
		ReadMarketData:                    true,
		ReadMarketDataOfOwnAssetProfile:   true,
		ReadPlatforms:                     true,
		ReadPlatformsWithAccountCount:     true,
		ReadTags:                          true,
		ReadWatchlist:                     true,
		UpdateAccount:                     true,
		UpdateAccess:                      true,
		UpdateAuthDevice:                  true,
		UpdateMarketData:                  true,
		UpdateMarketDataOfOwnAssetProfile: true,
		UpdateOrder:                       true,
		UpdatePlatform:                    true,
		UpdateTag:                         true,
		UpdateUserSettings:                true,
		UpdateViewMode:                    true,
	},
	model.RoleUser: {
		AccessAssistant:                   true,
		AccessHoldingsChart:               true,
		CreateAccess:                      true,
		CreateAccount:                     true,
		CreateAccountBalance:              true,
		CreateMarketDataOfOwnAssetProfile: true,
		CreateOrder:                       true,
		CreateOwnTag:                      true,
		CreateWatchlistItem:               true,
		DeleteAccess:                      true,
		DeleteAccount:                     true,
		DeleteAccountBalance:              true,
		DeleteAuthDevice:                  true,
		DeleteOrder:                       true,
		DeleteWatchlistItem:               true,
		EnableDataProviderGhostfolio:      true,
		ReadAiPrompt:                      true,
		ReadMarketDataOfOwnAssetProfile:   true,
		ReadPlatforms:                     true,
		ReadWatchlist:                     true,
		UpdateAccount:                     true,
		UpdateAccess:                      true,
		UpdateAuthDevice:                  true,
		UpdateMarketDataOfOwnAssetProfile: true,
		UpdateOrder:                       true,
		UpdateUserSettings:                true,
		UpdateViewMode:                    true,
	},
	model.RoleDemo: {
		AccessAssistant:     true,
		AccessHoldingsChart: true,
		CreateUserAccount:   true,
		ReadAiPrompt:        true,
		ReadWatchlist:       true,
	},
	model.RoleInactive: {},
}

// PermissionsForRole returns the list of permission strings granted to the given
// role. The returned slice is unordered (map iteration order). Returns an empty
// (non-nil) slice for unknown roles so callers can safely range over the result.
//
// This is primarily used by the /user endpoint to expose a caller's effective
// permissions to the frontend, mirroring the NestJS UserService behavior.
func PermissionsForRole(role model.Role) []string {
	perms, ok := rolePermissions[role]
	if !ok {
		return []string{}
	}
	result := make([]string, 0, len(perms))
	for k := range perms {
		result = append(result, k)
	}
	return result
}

// HasPermission reports whether the given role has the specified permission.
// Returns false for unknown roles (treating them as INACTIVE). This is the
// hot path called on every protected request via RequirePermission, so it is
// intentionally a simple two-step map lookup with no allocations.
func HasPermission(role model.Role, perm string) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}
