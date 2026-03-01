package db

import _ "embed"

//go:embed migrations/000001_initial_schema.up.sql
var initialSchemaMigration []byte

//go:embed migrations/000002_seed.up.sql
var seedMigration []byte

// InitialSchemaMigrationSQL returns the embedded bootstrap schema SQL.
func InitialSchemaMigrationSQL() []byte {
	return initialSchemaMigration
}

// SeedMigrationSQL returns the embedded seed data SQL.
func SeedMigrationSQL() []byte {
	return seedMigration
}
