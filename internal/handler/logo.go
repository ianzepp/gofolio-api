package handler

import "net/http"

func HandleGetLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
}

func HandleGetLogoBySymbol(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
}
