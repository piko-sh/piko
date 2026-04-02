// Package db provides embedded MySQL migration and seed files for the blog
// database. Pass Migrations as the MigrationFS field and Seeds as the SeedFS
// field in a DatabaseRegistration to apply the schema and demo data
// automatically, or use them directly with the respective services.
package db

import "embed"

// Migrations contains the MySQL migration files for the blog database.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Seeds contains the MySQL seed files for populating demo data.
//
//go:embed seeds/*.sql
var Seeds embed.FS
