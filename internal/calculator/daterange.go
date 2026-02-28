package calculator

import (
	"strconv"
	"time"
)

// GetIntervalFromDateRange returns start and end dates for a named date range.
// Ported from libs/common/src/lib/calculation-helper.ts.
func GetIntervalFromDateRange(dateRange string, portfolioStart ...time.Time) (start, end time.Time) {
	pStart := time.Unix(0, 0).In(time.Local)
	if len(portfolioStart) > 0 {
		pStart = portfolioStart[0]
	}

	now := time.Now()
	today := StartOfDay(now)
	endDate := EndOfDay(now)
	startDate := pStart
	legacyResetUTC := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch dateRange {
	case "1d":
		candidate := today.AddDate(0, 0, -1)
		startDate = maxTime(startDate, candidate)
	case "mtd":
		som := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		candidate := som.AddDate(0, 0, -1)
		startDate = maxTime(startDate, candidate)
	case "wtd":
		// weekStartsOn: Monday (1)
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday
		}
		monday := today.AddDate(0, 0, -(weekday - 1))
		candidate := monday.AddDate(0, 0, -1)
		startDate = maxTime(startDate, candidate)
	case "ytd":
		soy := time.Date(today.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		candidate := soy.AddDate(0, 0, -1)
		startDate = maxTime(startDate, candidate)
	case "1y":
		// Legacy parity: resetHours(new Date()) in NestJS builds a UTC-midnight
		// Date and then applies subYears(), which can shift the formatted local day.
		candidate := legacyResetUTC.AddDate(-1, 0, 0)
		startDate = maxTime(startDate, candidate)
	case "5y":
		// Same legacy resetHours() behavior as 1y.
		candidate := legacyResetUTC.AddDate(-5, 0, 0)
		startDate = maxTime(startDate, candidate)
	case "max":
		// startDate stays as portfolioStart
	default:
		// Calendar year string: "2024", "2023", etc.
		if year, err := strconv.Atoi(dateRange); err == nil {
			endDate = EndOfDay(time.Date(year, 12, 31, 0, 0, 0, 0, time.Local))
			candidate := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
			startDate = maxTime(startDate, candidate)
		}
	}

	return startDate, endDate
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
