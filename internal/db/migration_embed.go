package db

import _ "embed"

//go:embed migrations/000001_initial_schema.up.sql
var initialSchemaMigration []byte

// InitialSchemaMigrationSQL returns the embedded bootstrap schema SQL.
func InitialSchemaMigrationSQL() []byte {
	return initialSchemaMigration
}
