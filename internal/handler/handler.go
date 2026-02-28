package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// --- Response helpers ---

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("handler: failed to encode JSON response: %v", err)
	}
}

// Error writes a JSON error response matching NestJS format:
// { "statusCode": 403, "message": "Forbidden" }
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]any{
		"statusCode": status,
		"message":    message,
	})
}

// NoContent writes a 204 response with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// --- Request helpers ---

// Bind decodes a JSON request body into the target struct.
// Returns false and writes a 400 error if decoding fails.
// Use this instead of ReadBody — it takes both w and r.
func Bind(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Body == nil {
		Error(w, http.StatusBadRequest, "Request body is required")
		return false
	}

	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		Error(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return false
	}

	return true
}

// --- Query parameter helpers ---

// Filters represents the common filter query parameters used across many endpoints.
type Filters struct {
	Accounts     []string
	AssetClasses []string
	DataSource   string
	Symbol       string
	Tags         []string
	Range        string // "1d", "1y", "5y", "max", "mtd", "wtd", "ytd"
	Query        string // free text search
	HoldingType  string
}

// ParseFilters extracts the common filter query parameters from a request.
func ParseFilters(r *http.Request) Filters {
	q := r.URL.Query()
	return Filters{
		Accounts:     splitCSV(q.Get("accounts")),
		AssetClasses: splitCSV(q.Get("assetClasses")),
		DataSource:   q.Get("dataSource"),
		Symbol:       q.Get("symbol"),
		Tags:         splitCSV(q.Get("tags")),
		Range:        q.Get("range"),
		Query:        q.Get("query"),
		HoldingType:  q.Get("holdingType"),
	}
}

// Pagination represents skip/take/sort query parameters.
type Pagination struct {
	Skip          int
	Take          int
	SortColumn    string
	SortDirection string // "asc" or "desc"
}

// ParsePagination extracts pagination query parameters from a request.
func ParsePagination(r *http.Request) Pagination {
	q := r.URL.Query()
	return Pagination{
		Skip:          queryInt(q.Get("skip"), 0),
		Take:          queryInt(q.Get("take"), 50),
		SortColumn:    q.Get("sortColumn"),
		SortDirection: q.Get("sortDirection"),
	}
}

// Impersonation returns the impersonation user ID from the request header, if present.
func Impersonation(r *http.Request) string {
	return r.Header.Get("Impersonation-Id")
}

// --- Internal utilities ---

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func queryInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}

	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
