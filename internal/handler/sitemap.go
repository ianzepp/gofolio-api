package handler

/*
Stub parity reference:
Original NestJS controller(s):
- apps/api/src/app/endpoints/sitemap/sitemap.controller.ts

Routes + request/response shape:
- GET /sitemap.xml (HandleGetSitemap)
  Request: no explicit request params
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
- GET /api/v1/sitemap.xml (HandleGetSitemap)
  Request: no explicit request params
  Auth/permission: public / -
  Response: currently stubbed (placeholder; implement to match Nest behavior).
*/

import "net/http"

func HandleGetSitemap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
</urlset>`))
}
