package calculator

import "time"

const DateFormat = "2006-01-02"

// GetFactor returns +1 for BUY, -1 for SELL, 0 for everything else.
func GetFactor(orderType string) int {
	switch orderType {
	case "BUY":
		return 1
	case "SELL":
		return -1
	default:
		return 0
	}
}

// FormatDate formats a time as YYYY-MM-DD.
func FormatDate(t time.Time) string {
	return t.Format(DateFormat)
}

// ParseDate parses a YYYY-MM-DD string to a time.Time in UTC.
func ParseDate(s string) time.Time {
	t, _ := time.Parse(DateFormat, s)
	return t
}

// StartOfDay returns midnight UTC for the given date.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// EndOfDay returns 23:59:59.999999999 UTC for the given date.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.UTC)
}

// IsInvestmentActivity returns true for BUY and SELL activities.
func IsInvestmentActivity(orderType string) bool {
	return orderType == "BUY" || orderType == "SELL"
}
