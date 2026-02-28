package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/import/import.controller.ts

Routes + request/response shape:
- POST /api/v1/import (HandleCreateImport)
  Request: body importData: ImportDataDto
  Auth/permission: jwt / permissions.createOrder
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/import/dividends/{dataSource}/{symbol} (HandleGetImportDividends)
  Request: path dataSource: DataSource (arg: 'dataSource'); symbol: string (arg: 'symbol')
  Auth/permission: jwt / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleCreateImport(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if !Bind(w, r, &body) {
		return
	}
	_ = body
	JSON(w, http.StatusCreated, map[string]any{})
}

func HandleGetImportDividends(w http.ResponseWriter, r *http.Request) {
	dataSource := r.PathValue("dataSource")
	symbol := r.PathValue("symbol")
	_ = dataSource
	_ = symbol
	JSON(w, http.StatusOK, map[string]any{"activities": []any{}})
}
