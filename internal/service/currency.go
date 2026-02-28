package service

import (
	"strings"

	"golang.org/x/text/currency"
)

// defaultCurrencyCode is the base currency assumed when parsing symbol strings
// of the form "USDEUR", "USDGBP", etc. Stripping this prefix reveals the
// quote currency.
const defaultCurrencyCode = "USD"

// getCurrencyFromSymbol extracts the quote currency from a Ghostfolio-style
// currency pair symbol by removing the base currency prefix (USD). For example,
// "USDEUR" -> "EUR". This is a heuristic used when no SymbolProfile exists and
// the symbol must be interpreted as a currency pair.
func getCurrencyFromSymbol(symbol string) string {
	return strings.Replace(symbol, defaultCurrencyCode, "", 1)
}

// isCurrencyCode returns true if code is a recognised currency identifier.
// It first checks a hardcoded list of non-standard codes that appear in the
// legacy Ghostfolio codebase (USX for US cents, GBp for pence, etc.) before
// falling back to the CLDR ISO 4217 parser.
//
// WHY: The x/text/currency parser normalises to uppercase before lookup, so
// mixed-case codes like "GBp" (used for UK pence) would fail if passed through
// directly. The switch handles these before the normalised lookup.
func isCurrencyCode(code string) bool {
	if code == "" {
		return false
	}

	// Keep parity with common derived currencies from the legacy codebase.
	switch code {
	case "USX", "GBp", "ILA", "ZAc":
		return true
	}

	_, err := currency.ParseISO(strings.ToUpper(code))
	return err == nil
}
