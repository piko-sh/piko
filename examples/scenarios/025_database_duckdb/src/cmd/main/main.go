package main

import (
	"database/sql"
	"io/fs"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_duckdb"
	"piko.sh/piko/wdk/logger"

	salesdb "testmodule/db"
	_ "testmodule/dist"
)

func main() {
	logger.AddPrettyOutput()

	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if mkdirErr := os.MkdirAll("./data", 0o755); mkdirErr != nil {
		panic("creating data directory: " + mkdirErr.Error())
	}

	database, err := sql.Open("duckdb", "./data/sales.duckdb")
	if err != nil {
		panic(err)
	}

	// Apply schema and seed data directly since DuckDB has no migration
	// dialect in the querier system.
	schema, readErr := fs.ReadFile(salesdb.Migrations, "migrations/001_sales.up.sql")
	if readErr != nil {
		panic("reading schema: " + readErr.Error())
	}
	if _, execErr := database.Exec(string(schema)); execErr != nil {
		panic("applying schema: " + execErr.Error())
	}

	ssr := piko.New(
		piko.WithDatabase("sales", &db.DatabaseRegistration{
			DB:           database,
			EngineConfig: db_engine_duckdb.DuckDB(),
		}),
		piko.WithCSSReset(piko.WithCSSResetComplete()),
		piko.WithDevWidget(),
		piko.WithDevHotreload(),
		piko.WithMonitoring(),
	)
	if err := ssr.Run(command); err != nil {
		panic(err)
	}
}
