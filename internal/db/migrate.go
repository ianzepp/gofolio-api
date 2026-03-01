package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

// MigrateIfSchemaMissing checks whether the required sentinel table ("User")
// exists in the public schema. If missing, it executes the provided SQL to
// bootstrap the initial schema.
//
// This is a one-shot bootstrap for fresh databases and does not handle
// incremental migrations.
func MigrateIfSchemaMissing(ctx context.Context, d *sqlx.DB, migrationSQL []byte) error {
	var userTable sql.NullString
	err := d.QueryRowContext(ctx,
		`SELECT to_regclass('public."User"')`,
	).Scan(&userTable)
	if err != nil {
		return fmt.Errorf("check schema: %w", err)
	}

	if userTable.Valid {
		log.Printf(`schema already present ("User" table exists), skipping migration`)
		return nil
	}

	log.Printf(`schema missing ("User" table not found), running initial migration`)

	if _, err := d.ExecContext(ctx, string(migrationSQL)); err != nil {
		return fmt.Errorf("run migration: %w", err)
	}

	log.Printf("initial migration complete")
	return nil
}
