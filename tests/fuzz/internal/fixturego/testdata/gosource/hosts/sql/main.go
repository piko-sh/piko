package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
)

func main() {
	result, err := db.Generate(context.Background(), db_engine_sqlite.SQLite(), db.GenerateRequest{
		Files: os.DirFS("db"), PackageName: "generated", MigrationDirectory: "migrations",
		QueryDirectory: "queries", Driver: db.DriverDatabaseSQL,
	})
	if err != nil {
		panic(err)
	}
	for _, d := range result.Diagnostics {
		if d.Severity == db.SeverityError {
			panic(fmt.Sprintf("Seagull: %v", d))
		}
	}
	if len(result.Files) == 0 {
		panic("Seagull emitted no accessors")
	}
	if err := os.MkdirAll("db/generated", 0700); err != nil {
		panic(err)
	}
	for _, file := range result.Files {
		if filepath.Base(file.Name) != file.Name {
			panic("invalid generated filename")
		}
		if err := os.WriteFile(filepath.Join("db/generated", file.Name), file.Content, 0600); err != nil {
			panic(err)
		}
	}
}
