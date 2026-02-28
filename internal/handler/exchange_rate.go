package handler

import (
	"errors"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
)

// GetExchangeRate returns the exchange rate for a currency-pair symbol on a
// given date. The symbol format is "USDEUR" and dateString is "2024-01-15".
// Returns 404 when no rate is found.
func GetExchangeRate(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.ExchangeRateService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		symbol := r.PathValue("symbol")
		dateString := r.PathValue("dateString")

		resp, err := deps.ExchangeRateService.GetExchangeRate(r.Context(), symbol, dateString)
		if err != nil {
			if errors.Is(err, service.ErrInvalidExchangeRateDate) {
				Error(w, http.StatusBadRequest, err.Error())
				return
			}
			Error(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}
		if resp == nil {
			Error(w, http.StatusNotFound, http.StatusText(http.StatusNotFound))
			return
		}

		JSON(w, http.StatusOK, resp)
	}
}
