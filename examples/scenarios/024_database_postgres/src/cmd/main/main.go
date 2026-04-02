package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_postgres"
	"piko.sh/piko/wdk/logger"

	analyticsdb "testmodule/db"
	_ "testmodule/dist"
)

func main() {
	logger.AddPrettyOutput()

	ctx := context.Background()

	// Start a PostgreSQL container so the user does not need any external setup.
	fmt.Println("[postgres-analytics] Starting PostgreSQL container...")
	container, primary_dsn := startPostgres(ctx)
	defer func() {
		fmt.Println("[postgres-analytics] Stopping PostgreSQL container...")
		_ = container.Terminate(ctx)
	}()

	// In production the replica would point to a separate read-only server.
	// Here both the primary and replica use the same container for simplicity.
	replica_dsn := primary_dsn

	// Open the primary connection and run migrations before starting the app.
	database, err := sql.Open("pgx", primary_dsn)
	if err != nil {
		panic(fmt.Sprintf("opening PostgreSQL connection: %v", err))
	}

	executor := db.NewMigrationExecutor(database, db.PostgresDialect())
	file_reader := db.NewFSFileReader(analyticsdb.Migrations)
	migrator := db.NewMigrationService(executor, file_reader, "migrations")

	if _, err := migrator.Up(ctx); err != nil {
		panic(fmt.Sprintf("running migrations: %v", err))
	}
	fmt.Println("[postgres-analytics] Migrations applied")

	// Apply seed data if the database has not been seeded yet.
	seeder := db.NewSeedService(
		db.NewSeedExecutor(database, db.PostgresDialect()), db.NewFSFileReader(analyticsdb.Seeds), "seeds",
	)

	if applied, err := seeder.Apply(ctx); err != nil {
		panic(fmt.Sprintf("applying seeds: %v", err))
	} else if applied > 0 {
		fmt.Printf("[postgres-analytics] %d seed(s) applied\n", applied)
	}

	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	ssr := piko.New(
		piko.WithDatabase("analytics", &db.DatabaseRegistration{
			DB:           database,
			DriverName:   "pgx",
			EngineConfig: db_engine_postgres.Postgres(),
			Replicas: []db.Replica{
				{DSN: replica_dsn, Weight: 1},
			},
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

// startPostgres starts a PostgreSQL 16 container and returns the container
// handle alongside a DSN suitable for sql.Open with the pgx driver.
func startPostgres(ctx context.Context) (testcontainers.Container, string) {
	request := testcontainers.ContainerRequest{
		Image:        "postgres:16",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "analytics",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		panic(fmt.Sprintf("starting PostgreSQL container: %v", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		panic(fmt.Sprintf("getting container host: %v", err))
	}

	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		panic(fmt.Sprintf("getting container port: %v", err))
	}

	dsn := fmt.Sprintf("postgres://postgres:password@%s:%s/analytics?sslmode=disable", host, port.Port())
	fmt.Printf("[postgres-analytics] PostgreSQL ready at %s:%s\n", host, port.Port())

	return container, dsn
}

