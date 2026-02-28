package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/benchmarks/benchmarks.controller.ts

Routes + request/response shape:
- GET /api/v1/benchmarks (HandleGetBenchmarks)
  Request: no explicit request params
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- POST /api/v1/benchmarks (HandleCreateBenchmark)
  Request: no explicit request params
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- DELETE /api/v1/benchmarks/{dataSource}/{symbol} (HandleDeleteBenchmark)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / permissions.accessAdminControl
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/benchmarks/{dataSource}/{symbol}/{startDateString} (HandleGetBenchmarkDetail)
  Request: path dataSource: DataSource (arg: 'dataSource'); startDateString: string (arg: 'startDateString'); symbol: string (arg: 'symbol'); query dateRange: DateRange = 'max' (arg: 'range')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

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
