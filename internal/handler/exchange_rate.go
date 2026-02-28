package handler

import "net/http"

// GetExchangeRate returns the exchange rate for a currency-pair symbol on a
// given date. The symbol format is "USDEUR" and dateString is "2024-01-15".
// Returns 404 when no rate is found.
func GetExchangeRate(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.PathValue("symbol")
		dateString := r.PathValue("dateString")

		resp, err := deps.ExchangeRateService.GetExchangeRate(r.Context(), symbol, dateString)
		if err != nil {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if resp == nil {
			Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}

		JSON(w, http.StatusOK, resp)
	}
}
