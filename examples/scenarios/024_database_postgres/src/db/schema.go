// Package db provides embedded PostgreSQL migration and seed files for the
// analytics database. Pass Migrations as the MigrationFS field and Seeds as the
// SeedFS field in a DatabaseRegistration to apply the schema and demo data
// automatically, or use them directly with the respective services.
package db

import "embed"

// Migrations contains the PostgreSQL migration files for the analytics database.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Seeds contains the PostgreSQL seed files for populating demo data.
//
//go:embed seeds/*.sql
var Seeds embed.FS
