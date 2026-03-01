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
// bootstrap the initial schema, then runs the seed SQL to populate minimum
// viable data (admin user, Property rows).
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
		// Still run seed (idempotent) to ensure seed data exists even if
		// the schema was created in a previous run that failed before seeding.
		if err := runSeed(ctx, d); err != nil {
			return err
		}
		return nil
	}

	log.Printf(`schema missing ("User" table not found), running initial migration`)

	if _, err := d.ExecContext(ctx, string(migrationSQL)); err != nil {
		return fmt.Errorf("run migration: %w", err)
	}

	log.Printf("initial migration complete")

	if err := runSeed(ctx, d); err != nil {
		return err
	}

	return nil
}

// runSeed executes the embedded seed SQL. All statements use
// ON CONFLICT DO NOTHING so this is safe to call on every startup.
func runSeed(ctx context.Context, d *sqlx.DB) error {
	seedSQL := SeedMigrationSQL()
	if len(seedSQL) == 0 {
		return nil
	}

	log.Printf("running seed data migration")
	if _, err := d.ExecContext(ctx, string(seedSQL)); err != nil {
		return fmt.Errorf("run seed: %w", err)
	}
	log.Printf("seed data migration complete")
	return nil
}
