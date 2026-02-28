package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ghostfolio/ghostfolio/apps/api-go/internal/repository"
	"github.com/jmoiron/sqlx"
)

// LogoService fetches favicon images for symbols or URLs via Google's favicon
// service.
type LogoService struct {
	db     *sqlx.DB
	client *http.Client
}

func NewLogoService(db *sqlx.DB) *LogoService {
	return &LogoService{
		db: db,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LogoResult holds the binary image data and its MIME type.
type LogoResult struct {
	Data        []byte
	ContentType string
}

// GetLogoByURL fetches the favicon for a given website URL.
func (s *LogoService) GetLogoByURL(ctx context.Context, siteURL string) (*LogoResult, error) {
	if siteURL == "" {
		return nil, fmt.Errorf("url is required")
	}
	return s.fetchFavicon(ctx, siteURL)
}

// GetLogoBySymbol looks up the website URL for a symbol from its SymbolProfile,
// then fetches the favicon. Returns nil if the symbol has no URL.
func (s *LogoService) GetLogoBySymbol(ctx context.Context, dataSource string, symbol string) (*LogoResult, error) {
	repo := repository.NewSymbolProfileRepository(s.db)
	profile, err := repo.FindByDataSourceAndSymbol(ctx, dataSource, symbol)
	if err != nil {
		return nil, nil // symbol not found
	}
	if profile.Url == nil || *profile.Url == "" {
		return nil, nil // no URL for this symbol
	}
	return s.fetchFavicon(ctx, *profile.Url)
}

// fetchFavicon retrieves a favicon from Google's favicon service.
func (s *LogoService) fetchFavicon(ctx context.Context, siteURL string) (*LogoResult, error) {
	faviconURL := "https://t0.gstatic.com/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=" +
		url.QueryEscape(siteURL) + "&size=64"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, faviconURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create favicon request: %w", err)
	}
	req.Header.Set("User-Agent", "request")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch favicon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("favicon service returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read favicon body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png"
	}

	return &LogoResult{Data: data, ContentType: contentType}, nil
}
