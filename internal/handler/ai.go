package handler

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/ianzepp/gofolio-api/internal/auth"
	"github.com/ianzepp/gofolio-api/internal/service"
)

// GetAiPrompt handles GET /api/v1/ai/prompt/{mode}.
// It builds a markdown table of the user's portfolio holdings sorted by
// allocation, optionally wrapped in an analysis prompt template.
// Auth: jwt + readAiPrompt permission, enforced at the routing layer.
func GetAiPrompt(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.PortfolioService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		userID := auth.UserID(r)
		if userID == "" {
			Error(w, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
			return
		}

		mode := r.PathValue("mode")
		filters := ParseFilters(r)

		pf := &service.PortfolioFilter{
			AccountIDs:   filters.Accounts,
			AssetClasses: filters.AssetClasses,
			DataSource:   filters.DataSource,
			Symbol:       filters.Symbol,
			Tags:         filters.Tags,
		}

		holdings, err := deps.PortfolioService.GetHoldings(r.Context(), userID, "max", pf)
		if err != nil {
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Sort by allocation descending
		sorted := make([]service.PortfolioHolding, len(holdings.Holdings))
		copy(sorted, holdings.Holdings)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].AllocationInPercentage > sorted[j].AllocationInPercentage
		})

		// Build markdown table
		var sb strings.Builder
		sb.WriteString("| Name | Symbol | Currency | Asset Class | Asset Sub Class | Allocation in Percentage |\n")
		sb.WriteString("| --- | --- | --- | --- | --- | --- |\n")
		for _, h := range sorted {
			assetClass := ""
			if h.AssetClass != nil {
				assetClass = *h.AssetClass
			}
			assetSubClass := ""
			if h.AssetSubClass != nil {
				assetSubClass = *h.AssetSubClass
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %.3f%% |\n",
				h.Name, h.Symbol, h.Currency, assetClass, assetSubClass,
				h.AllocationInPercentage*100))
		}

		prompt := sb.String()

		if mode != "portfolio" {
			prompt = fmt.Sprintf(`As a financial advisor, analyze the following investment portfolio in detail. Please provide your analysis in English.

%s

Please provide the following sections:
1. **Overview**: A brief summary of the portfolio composition
2. **Risk Assessment**: Evaluate the diversification and risk level
3. **Advantages**: What are the strengths of this portfolio?
4. **Disadvantages**: What are the weaknesses or risks?
5. **Target Group**: What type of investor is this portfolio suitable for?
6. **Optimization Ideas**: Specific suggestions for improvement
7. **Conclusion**: A brief overall assessment`, prompt)
		}

		JSON(w, http.StatusOK, map[string]any{
			"prompt": prompt,
		})
	}
}
