package handler

import "net/http"

func GetInfo(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.InfoService == nil {
			// Fallback stub when DB is unavailable
			JSON(w, http.StatusOK, map[string]any{
				"baseCurrency":      "USD",
				"benchmarks":        []any{},
				"currencies":        []string{"USD"},
				"demoAuthToken":     "",
				"globalPermissions": []string{},
				"statistics":        nil,
			})
			return
		}

		info, err := deps.InfoService.GetInfo(r.Context())
		if err != nil {
			JSON(w, http.StatusOK, map[string]any{
				"baseCurrency":      "USD",
				"benchmarks":        []any{},
				"currencies":        []string{"USD"},
				"demoAuthToken":     "",
				"globalPermissions": []string{},
				"statistics":        nil,
			})
			return
		}

		JSON(w, http.StatusOK, info)
	}
}
