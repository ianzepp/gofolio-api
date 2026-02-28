package handler

import "net/http"

func HandleGetDataProviderAssetProfile(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleGetDataProviderDividends(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{"dividends": []any{}})
}

func HandleGetDataProviderHistorical(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{"historicalData": []any{}})
}

func HandleDataProviderLookup(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{"items": []any{}})
}

func HandleGetDataProviderQuotes(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{"quotes": []any{}})
}

func HandleGetDataProviderStatus(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{})
}
