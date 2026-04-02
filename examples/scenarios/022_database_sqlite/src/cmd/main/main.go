package main

import (
	"context"
	"database/sql"
	"os"

	_ "modernc.org/sqlite"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
	"piko.sh/piko/wdk/logger"

	taskdb "testmodule/db"
	_ "testmodule/dist"
)

func main() {
	logger.AddPrettyOutput()

	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	// Open SQLite database with WAL mode.
	if mkdirErr := os.MkdirAll("./data", 0o755); mkdirErr != nil {
		panic("creating data directory: " + mkdirErr.Error())
	}

	database, err := sql.Open("sqlite", "file:./data/tasks.db")
	if err != nil {
		panic(err)
	}
	database.SetMaxOpenConns(1)

	if _, err := database.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000"); err != nil {
		panic(err)
	}

	// Run migrations.
	executor := db.NewMigrationExecutor(database, db.SQLiteDialect())
	file_reader := db.NewFSFileReader(taskdb.Migrations)
	migrator := db.NewMigrationService(executor, file_reader, "migrations")

	if _, err := migrator.Up(context.Background()); err != nil {
		panic(err)
	}

	ssr := piko.New(
		piko.WithDatabase("tasks", &db.DatabaseRegistration{
			DB:           database,
			EngineConfig: db_engine_sqlite.SQLite(),
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
