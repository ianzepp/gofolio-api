package handler

import (
	"errors"
	"net/http"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/service"
)

// GetLogo fetches a favicon for the URL provided in the ?url= query parameter.
func GetLogo(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps == nil || deps.LogoService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		siteURL := r.URL.Query().Get("url")
		result, err := deps.LogoService.GetLogoByURL(r.Context(), siteURL)
		if err != nil || result == nil {
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
		if deps == nil || deps.LogoService == nil {
			Error(w, http.StatusServiceUnavailable, http.StatusText(http.StatusServiceUnavailable))
			return
		}

		dataSource := r.PathValue("dataSource")
		symbol := r.PathValue("symbol")

		result, err := deps.LogoService.GetLogoBySymbol(r.Context(), dataSource, symbol)
		if errors.Is(err, service.ErrLogoSymbolNotFound) || result == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", result.ContentType)
		w.WriteHeader(http.StatusOK)
		w.Write(result.Data)
	}
}
