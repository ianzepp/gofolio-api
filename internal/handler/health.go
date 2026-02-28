package handler

import "net/http"

func HandleGetHealth(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func HandleGetHealthDataEnhancer(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	JSON(w, http.StatusOK, map[string]any{
		"name":   name,
		"status": "ok",
	})
}

func HandleGetHealthDataProvider(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	JSON(w, http.StatusOK, map[string]any{
		"dataSource": dataSource,
		"status":     "ok",
	})
}
