package auth

import "github.com/ghostfolio/ghostfolio/apps/api-go/internal/model"

// Permission constants matching libs/common/src/lib/permissions.ts
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

// rolePermissions maps each role to its set of permissions.
// Source of truth: libs/common/src/lib/permissions.ts getPermissions()
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
		AccessAssistant:  true,
		AccessHoldingsChart: true,
		CreateUserAccount: true,
		ReadAiPrompt:      true,
		ReadWatchlist:     true,
	},
	model.RoleInactive: {},
}

// HasPermission checks if the given role has the specified permission.
func HasPermission(role model.Role, perm string) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[perm]
}
