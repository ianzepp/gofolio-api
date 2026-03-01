package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

func TestEscapeLike(t *testing.T) {
	got := escapeLike(`abc_%\test`)
	want := `abc\_\%\\test`
	if got != want {
		t.Fatalf("escapeLike mismatch, got %q want %q", got, want)
	}
}

func TestSymbolProfileRepository_SearchByQuery_HappyPath(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewSymbolProfileRepository(sqlxDB)
	ctx := context.Background()

	rows := sqlmock.NewRows([]string{
		"id", "assetClass", "assetSubClass", "comment", "countries", "createdAt",
		"currency", "cusip", "dataSource", "figi", "figiComposite", "figiShareClass",
		"holdings", "isActive", "isin", "name", "scraperConfiguration", "sectors",
		"symbol", "symbolMapping", "updatedAt", "url", "userId",
	}).AddRow(
		"sym-1", nil, nil, nil,
		[]byte("[]"), time.Now(), "USD", nil, "YAHOO", nil, nil, nil,
		[]byte("[]"), true, strPtr("US1234567890"), strPtr("ACME Corp"), []byte("{}"), []byte("[]"),
		"ACME", []byte("{}"), time.Now(), strPtr("https://acme.test"), strPtr("user-1"),
	)

	queryRegex := regexp.MustCompile(`SELECT(.|\n)+FROM "SymbolProfile"(.|\n)+LIMIT 50`)
	mock.ExpectQuery(queryRegex.String()).
		WithArgs("YAHOO", "abc\\%%").
		WillReturnRows(rows)

	items, err := repo.SearchByQuery(ctx, "YAHOO", "abc%")
	if err != nil {
		t.Fatalf("SearchByQuery returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 result, got %d", len(items))
	}
	if items[0].Symbol != "ACME" {
		t.Fatalf("expected symbol ACME, got %s", items[0].Symbol)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("mock expectations not met: %v", err)
	}
}

func strPtr(s string) *string { return &s }
