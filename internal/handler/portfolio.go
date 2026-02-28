package handler

import "net/http"

// GET /portfolio/details
func HandleGetPortfolioDetails(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/dividends
func HandleGetPortfolioDividends(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/holding/:dataSource/:symbol
func HandleGetPortfolioHolding(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	impersonationId := Impersonation(r)
	_ = dataSource
	_ = symbol
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// PUT /portfolio/holding/:dataSource/:symbol/tags
func HandleUpdatePortfolioHoldingTags(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = dataSource
	_ = symbol
	_ = body
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/holdings
func HandleGetPortfolioHoldings(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{"holdings": []any{}})
}

// GET /portfolio/investments
func HandleGetPortfolioInvestments(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/performance
func HandleGetPortfolioPerformance(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	impersonationId := Impersonation(r)
	_ = filters
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}

// GET /portfolio/report
func HandleGetPortfolioReport(w http.ResponseWriter, r *http.Request) {
	impersonationId := Impersonation(r)
	_ = impersonationId
	JSON(w, http.StatusOK, map[string]any{})
}
