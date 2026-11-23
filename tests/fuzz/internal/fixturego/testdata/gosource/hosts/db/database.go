package main

import (
	"context"
	"database/sql"
	"os"

	_ "modernc.org/sqlite"
	"piko.sh/piko"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
)

func init() {
	workflowDatabase = openWorkflowDatabase
}

func openWorkflowDatabase() (piko.Option, func(), error) {
	conn, err := sql.Open("sqlite", os.Getenv("PIKO_FUZZ_WORKFLOW_DB"))
	if err != nil {
		return nil, nil, err
	}
	closeDB := func() { _ = conn.Close() }
	conn.SetMaxOpenConns(1)
	engine := db_engine_sqlite.SQLite()
	migrator, err := db.NewMigrator(engine, conn, os.DirFS("db"), "migrations")
	if err == nil {
		_, err = migrator.Up(context.Background())
	}
	if err != nil {
		closeDB()
		return nil, nil, err
	}
	return piko.WithDatabase("tasks", &db.DatabaseRegistration{DB: conn, EngineConfig: engine}), closeDB, nil
}
