package service

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func setupSQLMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	return sqlx.NewDb(db, "sqlmock"), mock, func() {
		_ = db.Close()
	}
}

func TestExchangeRateService_GetExchangeRate_YahooHappyPath(t *testing.T) {
	sqlxDB, mock, cleanup := setupSQLMock(t)
	defer cleanup()

	svc := NewExchangeRateService(sqlxDB)
	ctx := context.Background()

	query := `SELECT id, "createdAt", "dataSource", date,
		"marketPrice", state, symbol
		 FROM "MarketData"
		 WHERE "dataSource" = $1 AND symbol = $2 AND date <= $3
		 ORDER BY date DESC
		 LIMIT 1`
	rows := sqlmock.NewRows([]string{"id", "createdAt", "dataSource", "date", "marketPrice", "state", "symbol"}).
		AddRow("rate-id", time.Now(), "YAHOO", time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC), 1.2345, "READY", "USDEUR")
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs("YAHOO", "USDEUR", sqlmock.AnyArg()).
		WillReturnRows(rows)

	resp, err := svc.GetExchangeRate(ctx, "USDEUR", "2024-03-15")
	if err != nil {
		t.Fatalf("GetExchangeRate returned error: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.MarketPrice != 1.2345 {
		t.Fatalf("expected market price 1.2345, got %v", resp.MarketPrice)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestExchangeRateService_GetExchangeRatesByCurrency_HappyPath(t *testing.T) {
	sqlxDB, mock, cleanup := setupSQLMock(t)
	defer cleanup()

	svc := NewExchangeRateService(sqlxDB)
	ctx := context.Background()

	startDate := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 3, 12, 0, 0, 0, 0, time.UTC)

	queryRegex := regexp.MustCompile(`SELECT(.|\n)+FROM "MarketData"(.|\n)+ORDER BY date ASC`)
	rows := sqlmock.NewRows([]string{"id", "createdAt", "dataSource", "date", "marketPrice", "state", "symbol"}).
		AddRow("md-1", time.Now(), "YAHOO", startDate, 0.91, "READY", "USDEUR").
		AddRow("md-2", time.Now(), "YAHOO", endDate, 0.92, "READY", "USDEUR")
	mock.ExpectQuery(queryRegex.String()).
		WithArgs(startDate, endDate, "YAHOO", "USDEUR").
		WillReturnRows(rows)

	currencies := []string{"USD", "USD", "EUR"}
	rates, err := svc.GetExchangeRatesByCurrency(ctx, currencies, "EUR", startDate, endDate)
	if err != nil {
		t.Fatalf("GetExchangeRatesByCurrency returned error: %v", err)
	}

	usdPair := rates["USDEUR"]
	if len(usdPair) != 3 {
		t.Fatalf("expected 3 entries for USDEUR, got %d", len(usdPair))
	}
	if usdPair["2024-03-10"] != 0.91 {
		t.Errorf("expected 2024-03-10 rate 0.91, got %v", usdPair["2024-03-10"])
	}
	if usdPair["2024-03-11"] != 0.92 {
		t.Errorf("expected 2024-03-11 forward-filled rate 0.92, got %v", usdPair["2024-03-11"])
	}
	if usdPair["2024-03-12"] != 0.92 {
		t.Errorf("expected 2024-03-12 rate 0.92, got %v", usdPair["2024-03-12"])
	}

	eurPair := rates["EUREUR"]
	if len(eurPair) != 3 {
		t.Fatalf("expected 3 entries for EUREUR, got %d", len(eurPair))
	}
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if eurPair[dateStr] != 1.0 {
			t.Errorf("expected unity rate for %s, got %v", dateStr, eurPair[dateStr])
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func TestExchangeRateService_GetExchangeRate_InvalidDate(t *testing.T) {
	svc := NewExchangeRateService(nil)
	if _, err := svc.GetExchangeRate(context.Background(), "USDEUR", "15-03-2024"); err == nil || !errors.Is(err, ErrInvalidExchangeRateDate) {
		t.Fatalf("expected invalid date error, got %v", err)
	}
}

func TestExchangeRateService_GetExchangeRatesByCurrency_DefaultsToOneWithoutData(t *testing.T) {
	sqlxDB, mock, cleanup := setupSQLMock(t)
	defer cleanup()

	svc := NewExchangeRateService(sqlxDB)
	ctx := context.Background()
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	queryRegex := regexp.MustCompile(`SELECT(.|\n)+FROM "MarketData"(.|\n)+ORDER BY date ASC`)
	cols := []string{"id", "createdAt", "dataSource", "date", "marketPrice", "state", "symbol"}
	mock.ExpectQuery(queryRegex.String()).
		WithArgs(start, end, "YAHOO", "CHFEUR").
		WillReturnRows(sqlmock.NewRows(cols))
	mock.ExpectQuery(queryRegex.String()).
		WithArgs(start, end, "MANUAL", "CHFEUR").
		WillReturnRows(sqlmock.NewRows(cols))

	rates, err := svc.GetExchangeRatesByCurrency(ctx, []string{"CHF"}, "EUR", start, end)
	if err != nil {
		t.Fatalf("GetExchangeRatesByCurrency returned error: %v", err)
	}
	pair := rates["CHFEUR"]
	if len(pair) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(pair))
	}
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		if pair[dateStr] != 1.0 {
			t.Fatalf("expected unity rate for %s, got %v", dateStr, pair[dateStr])
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}
