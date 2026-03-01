package db

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
)

// MigrateIfSchemaMissing checks whether the required sentinel table ("User")
// exists in the public schema. If missing, it executes the provided SQL to
// bootstrap the initial schema, then runs the seed SQL to populate minimum
// viable non-sensitive data.
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
		return ensureBootstrapAdmin(ctx, d)
	}

	log.Printf(`schema missing ("User" table not found), running initial migration`)

	if _, err := d.ExecContext(ctx, string(migrationSQL)); err != nil {
		return fmt.Errorf("run migration: %w", err)
	}

	log.Printf("initial migration complete")

	if err := runSeed(ctx, d); err != nil {
		return err
	}

	return ensureBootstrapAdmin(ctx, d)
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

const defaultBootstrapAdminID = "00000000-0000-0000-0000-000000000001"
const bootstrapAdminAdvisoryLockID int64 = 381012781

// ensureBootstrapAdmin creates an ADMIN user if one does not exist.
// Creation is driven by BOOTSTRAP_ADMIN_TOKEN and uses ACCESS_TOKEN_SALT
// hashing, matching the anonymous-auth token flow.
func ensureBootstrapAdmin(ctx context.Context, d *sqlx.DB) error {
	hasAdmin, err := adminExists(ctx, d)
	if err != nil {
		return err
	}
	if hasAdmin {
		return nil
	}

	token := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_TOKEN"))
	if token == "" {
		if isProductionEnvironment() {
			return fmt.Errorf("missing BOOTSTRAP_ADMIN_TOKEN in production while no admin user exists")
		}
		log.Printf("bootstrap admin token not set; skipping admin bootstrap in non-production")
		return nil
	}

	salt := os.Getenv("ACCESS_TOKEN_SALT")
	if salt == "" {
		return fmt.Errorf("missing ACCESS_TOKEN_SALT while BOOTSTRAP_ADMIN_TOKEN is set")
	}

	adminID := strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_ID"))
	if adminID == "" {
		adminID = defaultBootstrapAdminID
	}

	hashed := hashAccessToken(token, salt)

	tx, err := d.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bootstrap admin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, bootstrapAdminAdvisoryLockID); err != nil {
		return fmt.Errorf("acquire bootstrap admin lock: %w", err)
	}

	hasAdmin, err = adminExists(ctx, tx)
	if err != nil {
		return err
	}
	if hasAdmin {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit bootstrap admin tx: %w", err)
		}
		return nil
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO "User" (id, "accessToken", provider, role, "createdAt", "updatedAt")
		 VALUES ($1, $2, 'ANONYMOUS', 'ADMIN', NOW(), NOW())
		 ON CONFLICT (id) DO NOTHING`,
		adminID, hashed,
	); err != nil {
		return fmt.Errorf("insert bootstrap admin user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap admin tx: %w", err)
	}

	log.Printf("bootstrap admin ensured (id=%s)", adminID)
	return nil
}

func adminExists(ctx context.Context, q sqlx.ExtContext) (bool, error) {
	var exists bool
	if err := sqlx.GetContext(ctx, q, &exists, `SELECT EXISTS (SELECT 1 FROM "User" WHERE role = 'ADMIN')`); err != nil {
		return false, fmt.Errorf("check admin existence: %w", err)
	}
	return exists, nil
}

func hashAccessToken(token, salt string) string {
	m := hmac.New(sha512.New, []byte(salt))
	m.Write([]byte(token))
	return hex.EncodeToString(m.Sum(nil))
}

func isProductionEnvironment() bool {
	if strings.TrimSpace(os.Getenv("RAILWAY_ENVIRONMENT")) != "" {
		return true
	}
	for _, key := range []string{"APP_ENV", "GO_ENV", "NODE_ENV", "ENVIRONMENT"} {
		if strings.EqualFold(strings.TrimSpace(os.Getenv(key)), "production") {
			return true
		}
	}
	return false
}
