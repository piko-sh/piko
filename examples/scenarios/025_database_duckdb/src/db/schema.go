// Package db provides embedded DuckDB migration files for the sales analytics
// database.
package db

import "embed"

// Migrations contains the DuckDB migration files for the sales database.
//
//go:embed migrations/*.sql
var Migrations embed.FS
