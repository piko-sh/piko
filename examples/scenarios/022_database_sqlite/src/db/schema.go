// Package db provides embedded SQLite migration files for the task manager
// database. Pass Migrations as the MigrationFS field in a DatabaseRegistration
// to apply the schema automatically, or use it directly with a
// MigrationService.
package db

import "embed"

// Migrations contains the SQLite migration files for the tasks database.
//
//go:embed migrations/*.sql
var Migrations embed.FS
