package calculator

import (
	"testing"
	"time"
)

func TestGetIntervalFromDateRange_Max(t *testing.T) {
	portfolioStart := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	start, end := GetIntervalFromDateRange("max", portfolioStart)

	if !start.Equal(portfolioStart) {
		t.Errorf("max start = %v, want %v", start, portfolioStart)
	}
	// end should be end of today
	today := time.Now().UTC()
	if end.Year() != today.Year() || end.Month() != today.Month() || end.Day() != today.Day() {
		t.Errorf("max end date = %v, want today", end)
	}
}

func TestGetIntervalFromDateRange_1d(t *testing.T) {
	start, _ := GetIntervalFromDateRange("1d")
	today := time.Now().UTC()
	yesterday := today.AddDate(0, 0, -1)
	// Start should be yesterday or later
	if start.Year() != yesterday.Year() || start.Month() != yesterday.Month() || start.Day() != yesterday.Day() {
		t.Errorf("1d start = %v, want around %v", start, yesterday)
	}
}

func TestGetIntervalFromDateRange_Year(t *testing.T) {
	start, end := GetIntervalFromDateRange("2023")

	if start.Year() != 2023 || start.Month() != 1 || start.Day() != 1 {
		t.Errorf("2023 start = %v, want 2023-01-01", start)
	}
	if end.Year() != 2023 || end.Month() != 12 || end.Day() != 31 {
		t.Errorf("2023 end = %v, want 2023-12-31", end)
	}
}

func TestGetIntervalFromDateRange_Ytd(t *testing.T) {
	start, _ := GetIntervalFromDateRange("ytd")
	now := time.Now().UTC()
	// Start should be Dec 31 of previous year
	expectedStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	if start.Year() != expectedStart.Year() || start.Month() != expectedStart.Month() || start.Day() != expectedStart.Day() {
		t.Errorf("ytd start = %v, want %v", start, expectedStart)
	}
}

func TestGetIntervalFromDateRange_MaxWithEarlyPortfolioStart(t *testing.T) {
	// Portfolio start before epoch should still work
	portfolioStart := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	start, _ := GetIntervalFromDateRange("max", portfolioStart)
	if start.After(portfolioStart) {
		t.Errorf("max with epoch start: start %v should not be after portfolioStart %v", start, portfolioStart)
	}
}
