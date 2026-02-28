package handler

import "net/http"

func HandleGetInfo(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]any{
		"baseCurrency":      "USD",
		"benchmarks":        []any{},
		"currencies":        []string{"USD"},
		"demoAuthToken":     "",
		"globalPermissions": []string{},
		"statistics":        nil,
	})
}
