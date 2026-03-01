package handler

import (
	"database/sql"
	"errors"
	"net/http"
)

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

func CreateBenchmark(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.BenchmarkService == nil {
			Error(w, http.StatusInternalServerError, "Service unavailable")
			return
		}

		var body struct {
			DataSource string `json:"dataSource"`
			Symbol     string `json:"symbol"`
		}
		if !Bind(w, r, &body) {
			return
		}

		if err := deps.BenchmarkService.AddBenchmark(r.Context(), body.DataSource, body.Symbol); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusNotFound, "Not Found")
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusCreated, map[string]any{})
	}
}

func DeleteBenchmark(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		if deps.BenchmarkService == nil {
			Error(w, http.StatusInternalServerError, "Service unavailable")
			return
		}

		if err := deps.BenchmarkService.DeleteBenchmark(r.Context(), dataSource, symbol); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				Error(w, http.StatusNotFound, "Not Found")
				return
			}
			Error(w, http.StatusInternalServerError, err.Error())
			return
		}

		JSON(w, http.StatusOK, map[string]any{})
	}
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
