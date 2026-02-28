package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/admin/admin.controller.ts

Routes + request/response shape:
- GET /api/v1/admin (HandleGetAdmin)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/admin/demo-user/sync (HandleSyncDemoUser)
  Request: no explicit request params
  Auth/permission: jwt / permissions.syncDemoUserAccount
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/admin/gather (HandleGatherData)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/admin/gather/{dataSource}/{symbol} (HandleGatherDataForSymbol)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol'); query dateRange: DateRange (arg: 'range')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/admin/gather/{dataSource}/{symbol}/{dateString} (HandleGatherDataForDate)
  Request: path dataSource: DataSource (arg: 'dataSource'); dateString: string (arg: 'dateString'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/admin/gather/max (HandleGatherMax)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/admin/gather/profile-data (HandleGatherProfileData)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/admin/gather/profile-data/{dataSource}/{symbol} (HandleGatherProfileDataForSymbol)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/admin/market-data (HandleGetAdminMarketData)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/admin/market-data/{dataSource}/{symbol}/test (HandleTestMarketData)
  Request: body data: { scraperConfiguration: ScraperConfiguration }
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- DELETE /api/v1/admin/profile-data/{dataSource}/{symbol} (HandleDeleteProfileData)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- PATCH /api/v1/admin/profile-data/{dataSource}/{symbol} (HandleUpdateProfileData)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol'); body assetProfile: UpdateAssetProfileDto
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/admin/profile-data/{dataSource}/{symbol} (HandleCreateProfileData)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- PUT /api/v1/admin/settings/{key} (HandleUpdateAdminSetting)
  Request: path key: string (arg: 'key'); body data: UpdatePropertyDto
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/admin/user (HandleGetAdminUsers)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/admin/user/{id} (HandleGetAdminUser)
  Request: path id: string (arg: 'id')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

// GET /admin
func HandleGetAdmin(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{
		"exchangeRates":    []any{},
		"settings":         map[string]any{},
		"transactionCount": 0,
		"userCount":        0,
		"version":          "",
	})
}

// GET /admin/demo-user/sync
func HandleSyncDemoUser(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather
func HandleGatherData(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/:dataSource/:symbol
func HandleGatherDataForSymbol(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	dateRange := r.URL.Query().Get("range")
	_ = dataSource
	_ = symbol
	_ = dateRange
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/:dataSource/:symbol/:dateString
func HandleGatherDataForDate(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	dateString := r.PathValue("dateString")
	_ = dataSource
	_ = symbol
	_ = dateString
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/max
func HandleGatherMax(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/profile-data
func HandleGatherProfileData(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/gather/profile-data/:dataSource/:symbol
func HandleGatherProfileDataForSymbol(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /admin/market-data
func HandleGetAdminMarketData(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	_ = filters
	JSON(w, http.StatusOK, map[string]any{"marketData": []any{}})
}

// POST /admin/market-data/:dataSource/:symbol/test
func HandleTestMarketData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}

// DELETE /admin/profile-data/:dataSource/:symbol
func HandleDeleteProfileData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

// PATCH /admin/profile-data/:dataSource/:symbol
func HandleUpdateProfileData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}

// POST /admin/profile-data/:dataSource/:symbol
func HandleCreateProfileData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusCreated, map[string]any{})
}

// PUT /admin/settings/:key
func HandleUpdateAdminSetting(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	_ = key
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /admin/user
func HandleGetAdminUsers(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}

// GET /admin/user/:id
func HandleGetAdminUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	_ = id
	JSON(w, http.StatusOK, map[string]any{})
}
