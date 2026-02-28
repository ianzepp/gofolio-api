package handler

import "net/http"

func HandleGetBenchmarks(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, []any{})
}

func HandleCreateBenchmark(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleDeleteBenchmark(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource // TODO: pass to service
	_ = symbol     // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleGetBenchmarkDetail(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	startDateString := r.PathValue("startDateString")
	filters := ParseFilters(r)
	_ = dataSource      // TODO: pass to service
	_ = symbol          // TODO: pass to service
	_ = startDateString // TODO: pass to service
	_ = filters         // TODO: pass to service
	JSON(w, http.StatusOK, map[string]any{})
}
