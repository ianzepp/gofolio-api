package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/health/health.controller.ts

Routes + request/response shape:
- GET /api/v1/health (HandleGetHealth)
  Request: no explicit request params
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/health/data-enhancer/{name} (HandleGetHealthDataEnhancer)
  Request: path name: string (arg: 'name')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/health/data-provider/{dataSource} (HandleGetHealthDataProvider)
  Request: path dataSource: DataSource (arg: 'dataSource')
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

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
