package service

import (
	"strings"

	"golang.org/x/text/currency"
)

const defaultCurrencyCode = "USD"

func getCurrencyFromSymbol(symbol string) string {
	return strings.Replace(symbol, defaultCurrencyCode, "", 1)
}

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
