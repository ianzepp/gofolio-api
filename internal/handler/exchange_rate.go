package handler

import "net/http"

func HandleGetExchangeRate(w http.ResponseWriter, r *http.Request) {
	symbol := r.PathValue("symbol")
	dateString := r.PathValue("dateString")
	_ = symbol
	_ = dateString
	JSON(w, http.StatusOK, map[string]any{})
}
