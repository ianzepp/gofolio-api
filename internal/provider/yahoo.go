// Package provider — yahoo.go implements a DataProvider backed by Yahoo Finance.
//
// # Authentication / crumb protocol
//
// Yahoo Finance's v7 quote and v10 quoteSummary endpoints require a "crumb"
// anti-CSRF token that is tied to a session cookie. The flow is:
//  1. Fetch https://finance.yahoo.com to obtain session cookies.
//  2. Exchange those cookies for a crumb string via /v1/test/getcrumb.
//  3. Append crumb=<value> to all subsequent API requests alongside the cookie.
//
// The crumb/cookie pair is cached for crumbCacheDur (1 hour) behind a mutex so
// concurrent callers share a single session rather than each triggering a full
// re-handshake. On HTTP 401 the crumb is invalidated and re-fetched once before
// returning an error.
//
// # Symbol conversion
//
// Yahoo Finance uses different symbol conventions from Ghostfolio:
//   - Forex pairs: USDCHF (6-char uppercase) → USDCHF=X
//   - Crypto:      BTCUSD (not 6-char) → BTC-USD
//
// convertToYahooSymbol / convertFromYahooSymbol handle the round-trip.
//
// # Fallback strategy
//
// GetQuotes tries the v7 batch endpoint first (supports up to yahooMaxPerReq
// symbols per call) and falls back to individual quoteSummary calls if the batch
// endpoint fails (e.g. rate limit or crumb rejection). This avoids a total
// blackout when the batch endpoint is temporarily unhealthy.
//
// # Close price preference
//
// Historical data prefers adjusted-close (adjclose) over raw close when both are
// present in the chart response. This matches standard finance practice where
// splits and dividends are reflected in the price series.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ianzepp/gofolio-api/internal/model"
)

const (
	yahooBaseURL   = "https://query2.finance.yahoo.com"
	yahooMaxPerReq = 50 // batch limit for the v7 quote endpoint
	crumbCacheDur  = 1 * time.Hour
	// WHY: Yahoo Finance rejects requests without a browser-like UA string.
	yahooUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
)

// YahooFinanceProvider fetches market data from Yahoo Finance. It is safe for
// concurrent use; the crumb/cookie session is protected by an internal mutex.
type YahooFinanceProvider struct {
	client *http.Client

	mu          sync.Mutex
	crumb       string
	cookie      string
	crumbExpiry time.Time
}

// NewYahooFinanceProvider creates a new Yahoo Finance provider.
func NewYahooFinanceProvider() *YahooFinanceProvider {
	return &YahooFinanceProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *YahooFinanceProvider) CanHandle(_ string) bool { return true }

func (p *YahooFinanceProvider) GetName() model.DataSource { return model.DataSourceYahoo }

func (p *YahooFinanceProvider) GetTestSymbol() string { return "AAPL" }

func (p *YahooFinanceProvider) GetMaxSymbolsPerRequest() int { return yahooMaxPerReq }

func (p *YahooFinanceProvider) GetDataProviderInfo() DataProviderInfo {
	return DataProviderInfo{
		IsPremium: false,
		Name:      "Yahoo Finance",
		URL:       "https://finance.yahoo.com",
	}
}

// GetQuotes fetches live quotes for the given symbols. Symbols are converted to
// Yahoo format before the request and converted back in the response map.
//
// PHASE 1: Try the v7 batch endpoint (requires crumb).
// PHASE 2: On failure, fall back to individual quoteSummary calls per symbol.
//
// EDGE: Symbols that fail in fallback mode are silently skipped rather than
// failing the entire batch, to maximise the partial result returned to callers.
func (p *YahooFinanceProvider) GetQuotes(ctx context.Context, symbols []string) (map[string]DataProviderResponse, error) {
	if len(symbols) == 0 {
		return map[string]DataProviderResponse{}, nil
	}

	yahooSymbols := make([]string, len(symbols))
	for i, s := range symbols {
		yahooSymbols[i] = convertToYahooSymbol(s)
	}

	result := make(map[string]DataProviderResponse)

	// Try v7 quote endpoint first
	data, err := p.fetchQuotesV7(ctx, yahooSymbols)
	if err == nil {
		for _, q := range data {
			sym := convertFromYahooSymbol(q.Symbol)
			result[sym] = DataProviderResponse{
				Currency:    q.Currency,
				DataSource:  string(model.DataSourceYahoo),
				MarketPrice: q.RegularMarketPrice,
				MarketState: mapMarketState(q.MarketState),
				Name:        q.ShortName,
			}
		}
		return result, nil
	}

	// Fallback: fetch individually via quoteSummary
	for _, sym := range symbols {
		ySym := convertToYahooSymbol(sym)
		q, qErr := p.fetchQuoteSummary(ctx, ySym)
		if qErr != nil {
			continue
		}
		result[sym] = DataProviderResponse{
			Currency:    q.Currency,
			DataSource:  string(model.DataSourceYahoo),
			MarketPrice: q.RegularMarketPrice,
			MarketState: mapMarketState(q.MarketState),
			Name:        q.ShortName,
		}
	}
	return result, nil
}

// GetHistorical fetches daily historical prices via the v8 chart endpoint.
// Results are keyed by YYYY-MM-DD strings and zero-price entries are skipped
// (Yahoo returns null close prices for non-trading days in some markets).
func (p *YahooFinanceProvider) GetHistorical(ctx context.Context, symbol string, from, to time.Time) (map[string]HistoricalResponse, error) {
	ySym := convertToYahooSymbol(symbol)
	u := fmt.Sprintf("%s/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d",
		yahooBaseURL, url.PathEscape(ySym), from.Unix(), to.Unix())

	body, err := p.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("yahoo historical: %w", err)
	}

	var resp yahooChartResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("yahoo historical parse: %w", err)
	}

	result := make(map[string]HistoricalResponse)
	if len(resp.Chart.Result) == 0 {
		return result, nil
	}

	r := resp.Chart.Result[0]
	closes := getClosePrices(r)
	for i, ts := range r.Timestamp {
		if i >= len(closes) {
			break
		}
		if closes[i] == 0 {
			continue
		}
		date := time.Unix(ts, 0).UTC().Format("2006-01-02")
		result[date] = HistoricalResponse{MarketPrice: closes[i]}
	}
	return result, nil
}

// GetDividends fetches dividend payment events via the v8 chart endpoint using
// the events=div query flag. The dividend amount is returned as MarketPrice in
// HistoricalResponse to use the same wire type as price history.
func (p *YahooFinanceProvider) GetDividends(ctx context.Context, symbol string, from, to time.Time) (map[string]HistoricalResponse, error) {
	ySym := convertToYahooSymbol(symbol)
	u := fmt.Sprintf("%s/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d&events=div",
		yahooBaseURL, url.PathEscape(ySym), from.Unix(), to.Unix())

	body, err := p.doGet(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("yahoo dividends: %w", err)
	}

	var resp yahooChartResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("yahoo dividends parse: %w", err)
	}

	result := make(map[string]HistoricalResponse)
	if len(resp.Chart.Result) == 0 {
		return result, nil
	}

	r := resp.Chart.Result[0]
	if r.Events == nil || r.Events.Dividends == nil {
		return result, nil
	}

	for _, div := range r.Events.Dividends {
		date := time.Unix(div.Date, 0).UTC().Format("2006-01-02")
		result[date] = HistoricalResponse{MarketPrice: div.Amount}
	}
	return result, nil
}

// GetAssetProfile fetches minimal profile info (name, currency, asset class)
// from the quoteSummary price module. Only the price module is requested to
// minimise response size — full fundamental data is not needed for profiling.
func (p *YahooFinanceProvider) GetAssetProfile(ctx context.Context, symbol string) (*model.SymbolProfile, error) {
	ySym := convertToYahooSymbol(symbol)
	q, err := p.fetchQuoteSummary(ctx, ySym)
	if err != nil {
		return nil, fmt.Errorf("yahoo asset profile: %w", err)
	}

	name := q.ShortName
	profile := &model.SymbolProfile{
		DataSource: model.DataSourceYahoo,
		Symbol:     symbol,
		Currency:   q.Currency,
		Name:       &name,
	}

	ac, asc := mapYahooQuoteType(q.QuoteType)
	if ac != "" {
		assetClass := model.AssetClass(ac)
		profile.AssetClass = &assetClass
	}
	if asc != "" {
		assetSubClass := model.AssetSubClass(asc)
		profile.AssetSubClass = &assetSubClass
	}

	return profile, nil
}

// Search performs a symbol search via the v1/finance/search endpoint.
// Results are filtered to the allowed asset types (EQUITY, ETF, FUTURE,
// MUTUALFUND, CRYPTOCURRENCY) and optionally INDEX. Other types (e.g. OPTION,
// CURRENCY) are excluded because they are not supported by Ghostfolio's
// portfolio model.
//
// NOTE: The Currency field in LookupItem is populated with the Yahoo exchange
// code (e.g. "NMS"), not a currency string, because the search endpoint does not
// return currency. Callers must resolve currency separately if needed.
func (p *YahooFinanceProvider) Search(ctx context.Context, query string, includeIndices bool) (LookupResponse, error) {
	u := fmt.Sprintf("%s/v1/finance/search?q=%s&quotesCount=10&newsCount=0",
		yahooBaseURL, url.QueryEscape(query))

	body, err := p.doGet(ctx, u)
	if err != nil {
		return LookupResponse{}, fmt.Errorf("yahoo search: %w", err)
	}

	var resp yahooSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return LookupResponse{}, fmt.Errorf("yahoo search parse: %w", err)
	}

	allowedTypes := map[string]bool{
		"EQUITY":         true,
		"ETF":            true,
		"FUTURE":         true,
		"MUTUALFUND":     true,
		"CRYPTOCURRENCY": true,
	}
	if includeIndices {
		allowedTypes["INDEX"] = true
	}

	items := make([]LookupItem, 0, len(resp.Quotes))
	for _, q := range resp.Quotes {
		if !allowedTypes[q.QuoteType] {
			continue
		}
		sym := convertFromYahooSymbol(q.Symbol)
		ac, asc := mapYahooQuoteType(q.QuoteType)
		name := q.ShortName
		if name == "" {
			name = q.LongName
		}
		items = append(items, LookupItem{
			AssetClass:    ac,
			AssetSubClass: asc,
			Currency:      q.Exchange, // Yahoo search returns exchange, not currency
			DataSource:    model.DataSourceYahoo,
			Name:          name,
			Symbol:        sym,
		})
	}

	return LookupResponse{Items: items}, nil
}

// --- Crumb/cookie management ---

// ensureCrumb returns a valid (crumb, cookie) pair for Yahoo Finance API access,
// re-fetching from Yahoo if the cached pair has expired or been invalidated.
// The mutex ensures only one goroutine performs the two-step handshake at a time;
// others wait and reuse the result.
func (p *YahooFinanceProvider) ensureCrumb(ctx context.Context) (string, string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.crumb != "" && time.Now().Before(p.crumbExpiry) {
		return p.crumb, p.cookie, nil
	}

	// Step 1: get a cookie from finance.yahoo.com
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://finance.yahoo.com", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", yahooUserAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("yahoo cookie fetch: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	var cookies []string
	for _, c := range resp.Cookies() {
		cookies = append(cookies, c.Name+"="+c.Value)
	}
	cookieStr := strings.Join(cookies, "; ")

	// Step 2: get crumb
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, yahooBaseURL+"/v1/test/getcrumb", nil)
	if err != nil {
		return "", "", err
	}
	req2.Header.Set("User-Agent", yahooUserAgent)
	req2.Header.Set("Cookie", cookieStr)

	resp2, err := p.client.Do(req2)
	if err != nil {
		return "", "", fmt.Errorf("yahoo crumb fetch: %w", err)
	}
	defer resp2.Body.Close()
	crumbBytes, _ := io.ReadAll(resp2.Body)
	crumb := strings.TrimSpace(string(crumbBytes))
	if crumb == "" {
		return "", "", fmt.Errorf("yahoo: empty crumb")
	}

	p.crumb = crumb
	p.cookie = cookieStr
	p.crumbExpiry = time.Now().Add(crumbCacheDur)
	return crumb, cookieStr, nil
}

// --- HTTP helpers ---

// doGet performs an unauthenticated GET request. Used for endpoints that do not
// require the crumb/cookie session (e.g. v8 chart).
func (p *YahooFinanceProvider) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", yahooUserAgent)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("yahoo: HTTP %d for %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}

// doGetWithCrumb performs a crumb-authenticated GET. On HTTP 401 it invalidates
// the cached crumb, re-fetches it once, and retries the request. A second 401
// propagates as an error rather than triggering another retry to prevent loops.
func (p *YahooFinanceProvider) doGetWithCrumb(ctx context.Context, rawURL string) ([]byte, error) {
	crumb, cookie, err := p.ensureCrumb(ctx)
	if err != nil {
		return nil, err
	}

	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	fullURL := rawURL + sep + "crumb=" + url.QueryEscape(crumb)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", yahooUserAgent)
	req.Header.Set("Cookie", cookie)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		io.Copy(io.Discard, resp.Body)
		// Invalidate crumb and retry once
		p.mu.Lock()
		p.crumb = ""
		p.mu.Unlock()

		crumb, cookie, err = p.ensureCrumb(ctx)
		if err != nil {
			return nil, err
		}
		fullURL = rawURL + sep + "crumb=" + url.QueryEscape(crumb)
		req2, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
		if err != nil {
			return nil, err
		}
		req2.Header.Set("User-Agent", yahooUserAgent)
		req2.Header.Set("Cookie", cookie)
		resp2, err := p.client.Do(req2)
		if err != nil {
			return nil, err
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			io.Copy(io.Discard, resp2.Body)
			return nil, fmt.Errorf("yahoo: HTTP %d after crumb refresh", resp2.StatusCode)
		}
		return io.ReadAll(resp2.Body)
	}

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("yahoo: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// --- v7 quote ---

type yahooV7Response struct {
	QuoteResponse struct {
		Result []yahooQuote `json:"result"`
	} `json:"quoteResponse"`
}

type yahooQuote struct {
	Symbol             string  `json:"symbol"`
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	Currency           string  `json:"currency"`
	MarketState        string  `json:"marketState"`
	ShortName          string  `json:"shortName"`
	QuoteType          string  `json:"quoteType"`
}

// fetchQuotesV7 calls the v7 batch quote endpoint and returns raw quote structs.
// The caller is responsible for symbol conversion before passing symbols here.
func (p *YahooFinanceProvider) fetchQuotesV7(ctx context.Context, symbols []string) ([]yahooQuote, error) {
	u := fmt.Sprintf("%s/v7/finance/quote?symbols=%s",
		yahooBaseURL, url.QueryEscape(strings.Join(symbols, ",")))

	body, err := p.doGetWithCrumb(ctx, u)
	if err != nil {
		return nil, err
	}

	var resp yahooV7Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.QuoteResponse.Result, nil
}

// --- quoteSummary fallback ---

type yahooQuoteSummaryResponse struct {
	QuoteSummary struct {
		Result []struct {
			Price struct {
				Symbol             string        `json:"symbol"`
				RegularMarketPrice yahooRawValue `json:"regularMarketPrice"`
				Currency           string        `json:"currency"`
				MarketState        string        `json:"marketState"`
				ShortName          string        `json:"shortName"`
				QuoteType          string        `json:"quoteType"`
			} `json:"price"`
		} `json:"result"`
	} `json:"quoteSummary"`
}

type yahooRawValue struct {
	Raw float64 `json:"raw"`
}

// fetchQuoteSummary calls the v10 quoteSummary endpoint for a single symbol,
// requesting only the "price" module to minimise payload size. Returns an error
// when the result set is empty (Yahoo returns a 200 with empty results on unknown
// symbols rather than a 4xx).
func (p *YahooFinanceProvider) fetchQuoteSummary(ctx context.Context, symbol string) (*yahooQuote, error) {
	u := fmt.Sprintf("%s/v10/finance/quoteSummary/%s?modules=price",
		yahooBaseURL, url.PathEscape(symbol))

	body, err := p.doGetWithCrumb(ctx, u)
	if err != nil {
		return nil, err
	}

	var resp yahooQuoteSummaryResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	if len(resp.QuoteSummary.Result) == 0 {
		return nil, fmt.Errorf("yahoo quoteSummary: no results for %s", symbol)
	}

	price := resp.QuoteSummary.Result[0].Price
	return &yahooQuote{
		Symbol:             price.Symbol,
		RegularMarketPrice: price.RegularMarketPrice.Raw,
		Currency:           price.Currency,
		MarketState:        price.MarketState,
		ShortName:          price.ShortName,
		QuoteType:          price.QuoteType,
	}, nil
}

// --- Chart response types ---

type yahooChartResponse struct {
	Chart struct {
		Result []yahooChartResult `json:"result"`
	} `json:"chart"`
}

type yahooChartResult struct {
	Timestamp  []int64         `json:"timestamp"`
	Indicators yahooIndicators `json:"indicators"`
	Events     *yahooEvents    `json:"events,omitempty"`
}

type yahooIndicators struct {
	Quote []struct {
		Close []json.Number `json:"close"`
	} `json:"quote"`
	AdjClose []struct {
		AdjClose []json.Number `json:"adjclose"`
	} `json:"adjclose"`
}

type yahooEvents struct {
	Dividends map[string]yahooDividend `json:"dividends,omitempty"`
}

type yahooDividend struct {
	Amount float64 `json:"amount"`
	Date   int64   `json:"date"`
}

// getClosePrices extracts an ordered slice of daily closing prices from a chart
// result. Adjusted close (post-split/dividend) is preferred over raw close to
// produce consistent return calculations. Falls back to raw close when adjclose
// is absent (common for assets without split history).
func getClosePrices(r yahooChartResult) []float64 {
	// WHY: adjclose gives comparable prices across split dates; raw close does not.
	if len(r.Indicators.AdjClose) > 0 && len(r.Indicators.AdjClose[0].AdjClose) > 0 {
		nums := r.Indicators.AdjClose[0].AdjClose
		out := make([]float64, len(nums))
		for i, n := range nums {
			out[i], _ = n.Float64()
		}
		return out
	}
	if len(r.Indicators.Quote) > 0 {
		nums := r.Indicators.Quote[0].Close
		out := make([]float64, len(nums))
		for i, n := range nums {
			out[i], _ = n.Float64()
		}
		return out
	}
	return nil
}

// --- Search response types ---

type yahooSearchResponse struct {
	Quotes []struct {
		Symbol    string `json:"symbol"`
		ShortName string `json:"shortname"`
		LongName  string `json:"longname"`
		QuoteType string `json:"quoteType"`
		Exchange  string `json:"exchange"`
	} `json:"quotes"`
}

// --- Symbol conversion ---

// convertToYahooSymbol maps a Ghostfolio symbol to Yahoo Finance format.
// Ghostfolio stores forex as "USDCHF" and crypto as "BTCUSD"; Yahoo uses
// "USDCHF=X" and "BTC-USD" respectively.
func convertToYahooSymbol(symbol string) string {
	// Forex: USDCHF -> USDCHF=X
	if len(symbol) == 6 && isAllAlpha(symbol) && !strings.Contains(symbol, "-") {
		// Check if it looks like a forex pair (6 uppercase letters)
		upper := strings.ToUpper(symbol)
		if upper == symbol {
			return symbol + "=X"
		}
	}
	// Crypto: BTCUSD -> BTC-USD
	if strings.HasSuffix(symbol, "USD") && len(symbol) > 3 && len(symbol) <= 10 {
		base := symbol[:len(symbol)-3]
		if isAllAlpha(base) && strings.ToUpper(base) == base && strings.ToUpper(symbol) == symbol {
			// Only convert if it's all uppercase and looks like crypto
			// But not 6-char forex pairs (handled above)
			if len(symbol) != 6 {
				return base + "-USD"
			}
		}
	}
	return symbol
}

// convertFromYahooSymbol is the inverse of convertToYahooSymbol: it maps Yahoo
// Finance symbol format back to Ghostfolio's internal format.
func convertFromYahooSymbol(symbol string) string {
	// USDCHF=X -> USDCHF
	if strings.HasSuffix(symbol, "=X") {
		return strings.TrimSuffix(symbol, "=X")
	}
	// BTC-USD -> BTCUSD
	if strings.Contains(symbol, "-") {
		parts := strings.SplitN(symbol, "-", 2)
		if len(parts) == 2 {
			return parts[0] + parts[1]
		}
	}
	return symbol
}

func isAllAlpha(s string) bool {
	for _, c := range s {
		if c < 'A' || (c > 'Z' && c < 'a') || c > 'z' {
			return false
		}
	}
	return true
}

// --- Market state mapping ---

// mapMarketState converts Yahoo Finance's market state string ("REGULAR",
// "PRE", "POST", "CLOSED") to Ghostfolio's two-value vocabulary ("open" / "closed").
func mapMarketState(state string) string {
	if state == "REGULAR" {
		return "open"
	}
	return "closed"
}

// --- Quote type mapping ---

// mapYahooQuoteType translates a Yahoo Finance quoteType string to the
// Ghostfolio (assetClass, assetSubClass) pair. Unknown types return empty strings,
// which callers treat as "unclassified" rather than an error.
func mapYahooQuoteType(quoteType string) (assetClass, assetSubClass string) {
	switch quoteType {
	case "EQUITY":
		return string(model.AssetClassEquity), string(model.AssetSubClassStock)
	case "ETF":
		return string(model.AssetClassEquity), string(model.AssetSubClassETF)
	case "MUTUALFUND":
		return string(model.AssetClassEquity), string(model.AssetSubClassMutualFund)
	case "CRYPTOCURRENCY":
		return string(model.AssetClassLiquidity), string(model.AssetSubClassCryptocurrency)
	case "FUTURE":
		return string(model.AssetClassCommodity), string(model.AssetSubClassCommodity)
	default:
		return "", ""
	}
}
