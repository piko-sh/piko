---
title: How to swap database engines
description: Change between SQLite, PostgreSQL, MySQL, MariaDB, CockroachDB, and DuckDB without touching application code.
nav:
  sidebar:
    section: "how-to"
    subsection: "database"
    order: 30
---

# How to swap database engines

Piko supports SQLite, PostgreSQL, MySQL, MariaDB, CockroachDB, and DuckDB through swappable engine configs. A project picks the target at bootstrap, and the generator, migrator, and queries adapt to the chosen dialect. This guide covers what changes between engines and how to migrate a project from one to another. See the [querier reference](../../reference/querier.md) for the engine-config API.

## What stays the same

- `db/queries/*.sql` files with `piko.name` and `piko.command` annotations.
- Generated `Queries` struct and method signatures (names and return types are dialect-neutral).
- Call sites in actions and partials: `queries.ListTasks(ctx)` looks identical regardless of engine.
- The `MigrationService` API.

## What changes

| Area | Effect of changing engine |
|---|---|
| SQL dialect in queries and migrations | Minor syntactic differences: placeholder tokens, data types, RETURNING support, UPSERT syntax. |
| Engine config import | Swap `db_engine_sqlite.SQLite()` for the target's `Engine()` call. |
| Dialect passed to `NewMigrationExecutor` | Swap `db.SQLiteDialect()` for the target's dialect. |
| Driver import | Swap `modernc.org/sqlite` for the target's driver (`github.com/jackc/pgx/v5/stdlib`, `github.com/go-sql-driver/mysql`, etc.). |
| DSN format | Driver-specific. |

## Engine catalogue

| Engine | Import | Constructor | Dialect | Driver (typical) |
|---|---|---|---|---|
| SQLite | `piko.sh/piko/wdk/db/db_engine_sqlite` | `db_engine_sqlite.SQLite()` | `db.SQLiteDialect()` | `modernc.org/sqlite` (pure Go) or `github.com/mattn/go-sqlite3` (CGO) |
| PostgreSQL | `piko.sh/piko/wdk/db/db_engine_postgres` | `db_engine_postgres.Postgres()` | `db.PostgresDialect()` | `github.com/jackc/pgx/v5/stdlib` |
| PostgreSQL (PgBouncer) | `piko.sh/piko/wdk/db/db_engine_postgres` | `db_engine_postgres.PostgresPgBouncer()` | `db.PostgresPgBouncerDialect()` | `github.com/jackc/pgx/v5/stdlib` |
| MySQL | `piko.sh/piko/wdk/db/db_engine_mysql` | `db_engine_mysql.MySQL()` | `db.MySQLDialect()` or `db.MySQLDialectWithDSN(dsn)` | `github.com/go-sql-driver/mysql` |
| MariaDB | `piko.sh/piko/wdk/db/db_engine_mariadb` | `db_engine_mariadb.MariaDB()` | `db.MySQLDialect()` | `github.com/go-sql-driver/mysql` |
| CockroachDB | `piko.sh/piko/wdk/db/db_engine_cockroachdb` | `db_engine_cockroachdb.CockroachDB()` | `db.PostgresDialect()` | `github.com/jackc/pgx/v5/stdlib` |
| DuckDB | `piko.sh/piko/wdk/db/db_engine_duckdb` | `db_engine_duckdb.DuckDB()` | None - codegen-only engine; the `EngineConfig.MigrationDialect` is zero-valued, so do not point a `MigrationService` at a DuckDB connection. | `github.com/marcboeker/go-duckdb` |

Use `db_engine_postgres.PostgresPgBouncer()` when your application talks to PostgreSQL through PgBouncer in transaction-pooling mode. The pairing avoids advisory locks (which do not survive across pooled queries) and switches the migration runner to a table-based lock instead. `db_engine_mariadb.MariaDB()` ships as a MySQL variant - it returns the same `*MySQLEngine` underneath, with extra MariaDB-only function definitions registered.

## Example: swap SQLite to PostgreSQL

Start from a SQLite setup:

```go
import (
    _ "modernc.org/sqlite"

    "piko.sh/piko"
    "piko.sh/piko/wdk/db"
    "piko.sh/piko/wdk/db/db_engine_sqlite"
)

database, _ := sql.Open("sqlite", "file:./data/app.db")

executor := db.NewMigrationExecutor(database, db.SQLiteDialect())

ssr := piko.New(
    piko.WithDatabase("primary", &db.DatabaseRegistration{
        DB:           database,
        EngineConfig: db_engine_sqlite.SQLite(),
    }),
)
```

Switch to PostgreSQL by changing four lines:

```go
import (
    _ "github.com/jackc/pgx/v5/stdlib"

    "piko.sh/piko"
    "piko.sh/piko/wdk/db"
    "piko.sh/piko/wdk/db/db_engine_postgres"
)

database, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

executor := db.NewMigrationExecutor(database, db.PostgresDialect())

ssr := piko.New(
    piko.WithDatabase("primary", &db.DatabaseRegistration{
        DB:           database,
        EngineConfig: db_engine_postgres.Postgres(),
    }),
)
```

Everything else in the codebase (action handlers, generated queries, partials) stays untouched.

## Adjust the SQL

The generator reads your `migrations/*.sql` and `queries/*.sql` files as-is. If you wrote them for SQLite and now target Postgres, six syntax differences matter:

| Feature | SQLite | PostgreSQL | MySQL |
|---|---|---|---|
| Parameter placeholder | `?` | `$1`, `$2`, `$N` | `?` |
| Auto-increment primary key | `INTEGER PRIMARY KEY AUTOINCREMENT` | `SERIAL` or `GENERATED ALWAYS AS IDENTITY` | `INT AUTO_INCREMENT PRIMARY KEY` |
| Boolean | Stored as INTEGER | `BOOLEAN` | `TINYINT(1)` or `BOOLEAN` |
| Timestamp | Stored as INTEGER (epoch) or TEXT | `TIMESTAMPTZ` | `DATETIME` or `TIMESTAMP` |
| UPSERT | `ON CONFLICT ... DO UPDATE` | `ON CONFLICT ... DO UPDATE` | `ON DUPLICATE KEY UPDATE` |
| RETURNING | Supported (SQLite 3.35+) | Supported | Not supported; use separate SELECT |
| LIMIT with offset | `LIMIT n OFFSET m` | `LIMIT n OFFSET m` | `LIMIT m, n` |

When queries or migrations diverge, keep per-dialect variants in side-by-side files (for example, `users.postgres.sql` and `users.sqlite.sql`) or keep a single dialect-neutral subset where possible.

## Run migrations against the new engine fresh

Migrations tracked against SQLite do not translate automatically to Postgres. Two paths:

1. **Fresh database**: apply every migration from the start on the new engine. Appropriate during development or for projects that carry no production data yet.
2. **Data migration**: export data from the source, recreate the schema on the target, import the data, then mark the equivalent migrations as applied on the target without rerunning them (using a one-off script that writes the appropriate rows to the migration-tracking table).

For production migrations, option 2 is the realistic path. Write it as a one-off tool outside the main binary and run it under a maintenance window.

## Test against the target engine

Dialect differences surface as runtime errors, not compile errors. Exercise the target engine in integration tests:

- Use testcontainers-go to spin up a real PostgreSQL/MySQL/MariaDB container per test run. Scenarios [023 MySQL](../../../examples/scenarios/023_database_mysql/) and [024 PostgreSQL](../../../examples/scenarios/024_database_postgres/) demonstrate the pattern.
- For SQLite, point `sql.Open` at an in-memory database (`file::memory:?cache=shared`) for fast tests.

Never let a query reach production that has not run against the target engine at least once.

## Multiple databases in one project

Call `piko.WithDatabase(name, ...)` multiple times with different names:

```go
piko.WithDatabase("primary", &db.DatabaseRegistration{
    DB:           primaryDB,
    EngineConfig: db_engine_postgres.Postgres(),
}),
piko.WithDatabase("analytics", &db.DatabaseRegistration{
    DB:           duckDB,
    EngineConfig: db_engine_duckdb.DuckDB(),
}),
```

By default every database in a project shares the single `db/generated/` output directory. The generator produces one `Queries` struct per project, and each call site resolves the right `*sql.DB` by name via `db.GetDatabaseConnection("analytics")` (from `piko.sh/piko/wdk/db`). Override `GeneratedOutputDirectory` on a `DatabaseRegistration` if you want a database to emit into its own subdirectory instead.

## Back Piko's internal services with SQL

Reserved names let Piko's registry and orchestrator persist to SQL instead of the default in-memory backends:

```go
piko.WithDatabase(db.DatabaseNameRegistry, &db.DatabaseRegistration{
    DB:           postgresDB,
    EngineConfig: db_engine_postgres.Postgres(),
}),
piko.WithDatabase(db.DatabaseNameOrchestrator, &db.DatabaseRegistration{
    DB:           postgresDB,
    EngineConfig: db_engine_postgres.Postgres(),
}),
```

Use this for production deployments where a process restart should not lose in-flight orchestrator tasks.

## See also

- [Querier reference](../../reference/querier.md).
- [How to migrations](migrations.md).
- [How to writing queries](queries.md).
- Scenarios: [022 SQLite](../../../examples/scenarios/022_database_sqlite/), [023 MySQL](../../../examples/scenarios/023_database_mysql/), [024 PostgreSQL](../../../examples/scenarios/024_database_postgres/), [025 DuckDB](../../../examples/scenarios/025_database_duckdb/).
