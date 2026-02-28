package handler

import "net/http"

func GetMarketData(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		if deps.MarketDataService == nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		resp, err := deps.MarketDataService.GetBySymbol(r.Context(), dataSource, symbol)
		if err != nil {
			JSON(w, http.StatusOK, map[string]any{})
			return
		}

		JSON(w, http.StatusOK, resp)
	}
}

func HandleCreateMarketData(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleGetMarkets(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}
