package handler

import "net/http"

func GetBenchmarks(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.BenchmarkService == nil {
			JSON(w, http.StatusOK, map[string]any{"benchmarks": []any{}})
			return
		}

		resp, err := deps.BenchmarkService.GetBenchmarks(r.Context())
		if err != nil {
			JSON(w, http.StatusOK, map[string]any{"benchmarks": []any{}})
			return
		}

		JSON(w, http.StatusOK, resp)
	}
}

func HandleCreateBenchmark(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleDeleteBenchmark(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{})
}

func HandleGetBenchmarkDetail(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	startDateString := r.PathValue("startDateString")
	filters := ParseFilters(r)
	_ = dataSource
	_ = symbol
	_ = startDateString
	_ = filters
	JSON(w, http.StatusOK, map[string]any{})
}
