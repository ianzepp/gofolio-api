package calculator

import (
	"testing"
	"time"
)

func TestGetFactor(t *testing.T) {
	tests := []struct {
		orderType string
		want      int
	}{
		{"BUY", 1},
		{"SELL", -1},
		{"DIVIDEND", 0},
		{"FEE", 0},
		{"INTEREST", 0},
		{"LIABILITY", 0},
		{"ITEM", 0},
		{"LIQUIDITY", 0},
		{"", 0},
	}
	for _, tt := range tests {
		t.Run(tt.orderType, func(t *testing.T) {
			got := GetFactor(tt.orderType)
			if got != tt.want {
				t.Errorf("GetFactor(%q) = %d, want %d", tt.orderType, got, tt.want)
			}
		})
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	date := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	localDate := date.In(time.Local)
	expectedString := localDate.Format(DateFormat)
	s := FormatDate(date)
	if s != expectedString {
		t.Errorf("FormatDate = %q, want %q", s, expectedString)
	}
	parsed := ParseDate(s)
	expectedTime := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, time.Local)
	if !parsed.Equal(expectedTime) {
		t.Errorf("ParseDate(%q) = %v, want %v", s, parsed, expectedTime)
	}
}

func TestIsInvestmentActivity(t *testing.T) {
	if !IsInvestmentActivity("BUY") {
		t.Error("expected BUY to be investment activity")
	}
	if !IsInvestmentActivity("SELL") {
		t.Error("expected SELL to be investment activity")
	}
	if IsInvestmentActivity("DIVIDEND") {
		t.Error("expected DIVIDEND to not be investment activity")
	}
	if IsInvestmentActivity("FEE") {
		t.Error("expected FEE to not be investment activity")
	}
}

func TestStartOfDay(t *testing.T) {
	d := time.Date(2024, 6, 15, 14, 30, 45, 123, time.UTC)
	sod := StartOfDay(d)
	if sod.Hour() != 0 || sod.Minute() != 0 || sod.Second() != 0 {
		t.Errorf("StartOfDay = %v, want midnight", sod)
	}
}

func TestEndOfDay(t *testing.T) {
	d := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	eod := EndOfDay(d)
	if eod.Hour() != 23 || eod.Minute() != 59 || eod.Second() != 59 {
		t.Errorf("EndOfDay = %v, want 23:59:59", eod)
	}
}
