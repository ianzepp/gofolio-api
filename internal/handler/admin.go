package handler

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
