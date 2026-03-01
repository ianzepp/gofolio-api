package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var supportedLanguages = map[string]bool{
	"ca": true, "de": true, "en": true, "es": true, "fr": true,
	"it": true, "ko": true, "nl": true, "pl": true, "pt": true,
	"tr": true, "uk": true, "zh": true,
}

const defaultLanguage = "en"
const defaultTitle = "Ghostfolio"
const defaultDescription = "Ghostfolio is a personal finance dashboard to keep track of your net worth including cash, stocks, ETFs and cryptocurrencies across multiple platforms."
const defaultKeywords = "app, asset, cryptocurrency, dashboard, etf, finance, management, performance, portfolio, software, stock, trading, wealth, web3"
const defaultSlogan = "Open Source Wealth Management Software"

// StaticHandler serves the Angular frontend with language detection,
// HTML template interpolation, and SPA fallback. It replicates the behavior
// of NestJS ServeStaticModule + HtmlTemplateMiddleware.
type StaticHandler struct {
	clientDir    string
	indexHtmlMap map[string]string // languageCode -> index.html contents
	fileServer   http.Handler
}

// NewStaticHandler creates a handler that serves Angular static files from
// clientDir. It loads all per-language index.html files on startup for
// template interpolation.
func NewStaticHandler(clientDir string) *StaticHandler {
	h := &StaticHandler{
		clientDir:    clientDir,
		indexHtmlMap: make(map[string]string),
		fileServer:   http.FileServer(http.Dir(clientDir)),
	}

	for lang := range supportedLanguages {
		path := filepath.Join(clientDir, lang, "index.html")
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("static: could not load %s/index.html: %v", lang, err)
			continue
		}
		h.indexHtmlMap[lang] = string(data)
	}

	if len(h.indexHtmlMap) == 0 {
		log.Printf("static: no index.html templates loaded from %s (frontend may not be built)", clientDir)
	} else {
		log.Printf("static: loaded %d language templates from %s", len(h.indexHtmlMap), clientDir)
	}

	return h
}

func (h *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Do not serve SPA content for API paths. Keep API miss behavior aligned
	// with NestJS, where unknown /api routes return JSON 404.
	if strings.HasPrefix(path, "/api/") {
		JSON(w, http.StatusNotFound, map[string]any{
			"error":      "Not Found",
			"message":    "Cannot " + r.Method + " " + r.URL.RequestURI(),
			"statusCode": http.StatusNotFound,
		})
		return
	}

	// Root path: redirect to language-specific path
	if path == "/" || path == "" {
		lang := detectLanguage(r)
		http.Redirect(w, r, "/"+lang+"/", http.StatusMovedPermanently)
		return
	}

	// If it's a file request, serve directly from filesystem
	if isFileRequest(path) {
		h.fileServer.ServeHTTP(w, r)
		return
	}

	// SPA fallback: serve interpolated index.html for the detected language
	lang := extractLanguageFromPath(path)
	h.serveIndexHTML(w, r, lang, path)
}

func (h *StaticHandler) serveIndexHTML(w http.ResponseWriter, r *http.Request, lang, path string) {
	tmpl, ok := h.indexHtmlMap[lang]
	if !ok {
		tmpl, ok = h.indexHtmlMap[defaultLanguage]
		if !ok {
			http.Error(w, "frontend not available", http.StatusServiceUnavailable)
			return
		}
	}

	rootURL := os.Getenv("ROOT_URL")
	currentDate := time.Now().Format("2006-01-02")
	title := defaultTitle + " \u2013 " + defaultSlogan

	html := tmpl
	html = strings.ReplaceAll(html, "${languageCode}", lang)
	html = strings.ReplaceAll(html, "${title}", title)
	html = strings.ReplaceAll(html, "${description}", defaultDescription)
	html = strings.ReplaceAll(html, "${keywords}", defaultKeywords)
	html = strings.ReplaceAll(html, "${rootUrl}", rootURL)
	html = strings.ReplaceAll(html, "${currentDate}", currentDate)
	html = strings.ReplaceAll(html, "${featureGraphicPath}", "assets/cover.png")
	html = strings.ReplaceAll(html, "${path}", path)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// detectLanguage extracts the preferred language from the Accept-Language header.
func detectLanguage(r *http.Request) string {
	accept := r.Header.Get("Accept-Language")
	if accept == "" {
		return defaultLanguage
	}

	// Take first language code (before any comma or semicolon)
	code := accept
	if i := strings.IndexByte(code, ','); i >= 0 {
		code = code[:i]
	}
	if i := strings.IndexByte(code, ';'); i >= 0 {
		code = code[:i]
	}
	// Extract 2-letter code before hyphen
	if i := strings.IndexByte(code, '-'); i >= 0 {
		code = code[:i]
	}
	code = strings.TrimSpace(strings.ToLower(code))

	if supportedLanguages[code] {
		return code
	}
	return defaultLanguage
}

// extractLanguageFromPath gets the 2-letter language code from the URL path.
func extractLanguageFromPath(path string) string {
	// Path looks like /en/dashboard or /de/portfolio
	trimmed := strings.TrimPrefix(path, "/")
	if len(trimmed) >= 2 {
		code := trimmed[:2]
		if supportedLanguages[code] {
			return code
		}
	}
	return defaultLanguage
}

// isFileRequest returns true if the path looks like a request for a static file
// (has a file extension). Matches the NestJS HtmlTemplateMiddleware logic.
func isFileRequest(path string) bool {
	// Special cases matching NestJS behavior
	if path == "/assets/LICENSE" {
		return true
	}
	if strings.HasSuffix(path, "-de.fi") || strings.HasSuffix(path, "-markets.sh") {
		return false
	}
	if strings.Contains(path, "auth/ey") {
		return false
	}

	// Check for file extension: if the last segment after '.' differs from
	// the full filename, it has an extension
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return ext != ""
}
