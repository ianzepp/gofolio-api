package handler

import "net/http"

func HandleGetSymbol(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleGetSymbolByDate(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	dateString := r.PathValue("dateString")
	_ = dataSource
	_ = symbol
	_ = dateString
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleSymbolLookup(w http.ResponseWriter, r *http.Request) {
	filters := ParseFilters(r)
	_ = filters
	JSON(w, http.StatusOK, map[string]any{"items": []any{}})
}
