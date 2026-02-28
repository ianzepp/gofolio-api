package handler

import "net/http"

// GetLogo fetches a favicon for the URL provided in the ?url= query parameter.
func GetLogo(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		siteURL := r.URL.Query().Get("url")
		if siteURL == "" {
			Error(w, http.StatusBadRequest, "url query parameter is required")
			return
		}

		result, err := deps.LogoService.GetLogoByURL(r.Context(), siteURL)
		if err != nil || result == nil {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", result.ContentType)
		w.WriteHeader(http.StatusOK)
		w.Write(result.Data)
	}
}

// GetLogoBySymbol fetches a favicon for a symbol by looking up its website URL
// from the SymbolProfile table.
func GetLogoBySymbol(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		result, err := deps.LogoService.GetLogoBySymbol(r.Context(), dataSource, symbol)
		if err != nil || result == nil {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", result.ContentType)
		w.WriteHeader(http.StatusOK)
		w.Write(result.Data)
	}
}
