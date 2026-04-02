package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"piko.sh/piko"
	"piko.sh/piko/wdk/db"
	"piko.sh/piko/wdk/db/db_engine_mysql"
	"piko.sh/piko/wdk/logger"

	blogdb "testmodule/db"
	_ "testmodule/dist"
)

func main() {
	logger.AddPrettyOutput()

	ctx := context.Background()

	// Start a MySQL container so the user does not need any external setup.
	fmt.Println("[mysql-blog] Starting MySQL container...")
	container, dsn := startMySQL(ctx)
	defer func() {
		fmt.Println("[mysql-blog] Stopping MySQL container...")
		_ = container.Terminate(ctx)
	}()

	// Open the MySQL connection with retry, since the container may need a
	// moment after reporting ready before it actually accepts connections.
	database, err := connectWithRetry("mysql", dsn, 10, 2*time.Second)
	if err != nil {
		panic(fmt.Sprintf("opening MySQL connection: %v", err))
	}
	fmt.Println("[mysql-blog] Connected to MySQL")

	// Run migrations.
	migrationDialect := db.MySQLDialect()
	migrator := db.NewMigrationService(
		db.NewMigrationExecutor(database, migrationDialect), db.NewFSFileReader(blogdb.Migrations), "migrations",
	)

	if _, err := migrator.Up(ctx); err != nil {
		panic(fmt.Sprintf("running migrations: %v", err))
	}
	fmt.Println("[mysql-blog] Migrations applied")

	// Apply seed data if the database has not been seeded yet.
	seeder := db.NewSeedService(
		db.NewSeedExecutor(database, migrationDialect), db.NewFSFileReader(blogdb.Seeds), "seeds",
	)

	if applied, err := seeder.Apply(ctx); err != nil {
		panic(fmt.Sprintf("applying seeds: %v", err))
	} else if applied > 0 {
		fmt.Printf("[mysql-blog] %d seed(s) applied\n", applied)
	}

	command := piko.RunModeDev
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	ssr := piko.New(
		piko.WithDatabase("blog", &db.DatabaseRegistration{
			DB:           database,
			EngineConfig: db_engine_mysql.MySQL(),
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

// startMySQL starts a MySQL 8 container and returns the container handle
// alongside a DSN suitable for sql.Open.
func startMySQL(ctx context.Context) (testcontainers.Container, string) {
	request := testcontainers.ContainerRequest{
		Image:        "mysql:8",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "password",
			"MYSQL_DATABASE":      "blog",
		},
		// MySQL logs "ready for connections" twice: once during bootstrap and
		// once when fully ready. Wait for the second occurrence.
		WaitingFor: wait.ForLog("ready for connections").
			WithOccurrence(2).
			WithStartupTimeout(120 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: request,
		Started:          true,
	})
	if err != nil {
		panic(fmt.Sprintf("starting MySQL container: %v", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		panic(fmt.Sprintf("getting container host: %v", err))
	}

	port, err := container.MappedPort(ctx, "3306/tcp")
	if err != nil {
		panic(fmt.Sprintf("getting container port: %v", err))
	}

	dsn := fmt.Sprintf("root:password@tcp(%s:%s)/blog?parseTime=true", host, port.Port())
	fmt.Printf("[mysql-blog] MySQL ready at %s:%s\n", host, port.Port())

	return container, dsn
}

// connectWithRetry attempts to open and ping a database connection, retrying
// on failure. MySQL containers sometimes need a moment after the wait
// condition passes before they truly accept connections.
func connectWithRetry(driverName, dsn string, maxAttempts int, delay time.Duration) (*sql.DB, error) {
	var database *sql.DB
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		database, err = sql.Open(driverName, dsn)
		if err != nil {
			fmt.Printf("[mysql-blog] Connection attempt %d/%d failed (open): %v\n", attempt, maxAttempts, err)
			time.Sleep(delay)
			continue
		}

		if pingErr := database.Ping(); pingErr != nil {
			_ = database.Close()
			fmt.Printf("[mysql-blog] Connection attempt %d/%d failed (ping): %v\n", attempt, maxAttempts, pingErr)
			time.Sleep(delay)
			continue
		}

		return database, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", maxAttempts, err)
}
