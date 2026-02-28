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
	// Legacy parity: JS date-fns format() uses local timezone semantics.
	return t.In(time.Local).Format(DateFormat)
}

// ParseDate parses a YYYY-MM-DD string using local timezone semantics.
func ParseDate(s string) time.Time {
	t, _ := time.ParseInLocation(DateFormat, s, time.Local)
	return t
}

// StartOfDay returns local midnight for the given date.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// EndOfDay returns local 23:59:59.999999999 for the given date.
func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, time.Local)
}

// IsInvestmentActivity returns true for BUY and SELL activities.
func IsInvestmentActivity(orderType string) bool {
	return orderType == "BUY" || orderType == "SELL"
}
