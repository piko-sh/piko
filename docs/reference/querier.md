---
title: Querier and migrations API
description: Generator annotations, generated types, migration service, dialects, and engine configs.
nav:
  sidebar:
    section: "reference"
    subsection: "runtime"
    order: 80
---

# Querier and migrations API

Piko's querier system generates type-safe Go code from plain `.sql` files. A project writes SQL, annotates each query with a small comment header, and runs the generator. Piko emits a `Queries` struct whose methods match the annotated queries exactly. The migration service runs schema changes with advisory locking, checksum verification, and dialect-specific SQL.

This page documents the generator annotations, the generated types, and the `piko.sh/piko/wdk/db` facade. For task recipes see the how-to guides on [migrations](../how-to/database/migrations.md), [writing queries](../how-to/database/queries.md), and [swapping database engines](../how-to/database/swapping-engines.md). Source of truth: [`wdk/db/`](https://github.com/piko-sh/piko/tree/master/wdk/db) and [`cmd/generate_dal/`](https://github.com/piko-sh/piko/tree/master/cmd/generate_dal).

## Project layout

Every database in a project lives in its own directory under `db/`:

```
db/
  schema.go              Go package that embeds the migration FS.
  migrations/
    001_initial.up.sql   First forward migration.
    001_initial.down.sql Optional rollback.
    002_add_index.up.sql
  queries/
    *.sql                Annotated query files.
  generated/             Generator output; do not edit by hand.
    querier.go
    models.go
    prepared.go
    <name>.sql.go
```

The generator discovers `migrations/` and `queries/`, reads every `.sql` file, and writes type-safe Go under `generated/`. Re-run after any change:

```bash
piko generate
```

## Query annotations

Every query in `queries/*.sql` starts with two comment headers:

```sql
-- piko.name: ListTasks
-- piko.command: many
SELECT id, title, completed, created_at
FROM tasks
ORDER BY created_at DESC;
```

| Header | Purpose |
|---|---|
| `-- piko.name: Name` | Method name on the `Queries` struct. PascalCase by convention. |
| `-- piko.command: kind` | What the generator emits for this query. See below. |

### Command kinds

| Kind | Generated signature | Return shape |
|---|---|---|
| `one` | `(Queries).<Name>(ctx, params) (<Name>Row, error)` | Single row; `sql.ErrNoRows` on miss. |
| `many` | `(Queries).<Name>(ctx, params) ([]<Name>Row, error)` | Slice of rows. |
| `exec` | `(Queries).<Name>(ctx, params) error` | No rows returned; used for INSERT/UPDATE/DELETE without `RETURNING`. |

## Parameter types

Positional parameters in SQL (`?` for SQLite/MySQL, `$1/$2` for Postgres) map to Go in one of two shapes.

**Single parameter** (one `?` or `$1`):

```sql
-- piko.name: DeleteTask
-- piko.command: exec
DELETE FROM tasks WHERE id = ?;
```

Generator emits:

```go
func (q *Queries) DeleteTask(ctx context.Context, p1 int32) error
```

**Multiple parameters** (two or more):

```sql
-- piko.name: CreateTask
-- piko.command: one
INSERT INTO tasks (title, created_at) VALUES (?, ?) RETURNING id, title, created_at;
```

Generator emits a params struct:

```go
type CreateTaskParams struct {
    P1 string
    P2 int32
}

func (q *Queries) CreateTask(ctx context.Context, params CreateTaskParams) (CreateTaskRow, error)
```

The types on the params struct come from the engine's type inference over the SQL expression. Override them with type hints if the inference is too loose (see the [writing queries how-to](../how-to/database/queries.md)).

## Generated types

The generator emits four files under `generated/`:

### `querier.go`

Defines the `DBTX` interface (satisfied by `*sql.DB`, `*sql.Tx`, and `*sql.Conn`) and the `Queries` struct:

```go
type Queries struct { ... }

func New(db DBTX) *Queries
func NewWithReplica(writer DBTX, reader DBTX) *Queries
func (q *Queries) WithTx(tx *sql.Tx) *Queries
func (q *Queries) RunInTx(ctx context.Context, db *sql.DB, fn func(*Queries) error) error
```

`NewWithReplica` routes read queries (command kind `one`, `many`) to the reader and write queries (command kind `exec`, plus `one` queries with `RETURNING` on writes) to the writer.

### `models.go`

Go structs for every table defined by migrations. Field names come from column names using Go capitalisation.

### `prepared.go`

Prepared-statement cache (used automatically).

### `<name>.sql.go`

One file per `queries/<name>.sql`. Each contains the typed methods for that file's queries.

## Migration service

The migration service applies `.up.sql` files in order, records a checksum per applied file, and refuses to run when the on-disk file has changed since the migrator applied it.

### `db.NewMigrationService(executor, fileReader, directory, opts...) MigrationService`

Creates a migration service. Typical wiring:

```go
import "piko.sh/piko/wdk/db"

executor := db.NewMigrationExecutor(database, db.SQLiteDialect())
reader := db.NewFSFileReader(embeddedFS)
migrator := db.NewMigrationService(executor, reader, "migrations")

if _, err := migrator.Up(ctx); err != nil {
    panic(err)
}
```

### `MigrationService` methods

| Method | Purpose |
|---|---|
| `Up(ctx) ([]AppliedMigration, error)` | Applies every unapplied up migration. |
| `UpTo(ctx, version) ([]AppliedMigration, error)` | Applies up to the named version. |
| `Down(ctx) (AppliedMigration, error)` | Rolls back the most recent migration. |
| `DownTo(ctx, version) ([]AppliedMigration, error)` | Rolls back to the named version. |
| `Status(ctx) ([]MigrationStatus, error)` | Lists every migration and whether the migrator has applied it. |

### `NewFSFileReader(fs embed.FS) FileReaderPort`

Wraps a `go:embed` filesystem so migrations ship inside the binary.

### `DialectConfig`

One per supported database dialect. Dialects set the SQL for locking, checksum storage, and version tracking.

| Function | Dialect |
|---|---|
| `db.SQLiteDialect()` | SQLite. |
| `db.PostgresDialect()` | PostgreSQL with advisory locks. |
| `db.PostgresPgBouncerDialect()` | PostgreSQL behind PgBouncer (table-based locking). |
| `db.MySQLDialect()` | MySQL and MariaDB. |
| `db.MySQLDialectWithDSN(dsn)` | MySQL/MariaDB with detection of `multiStatements=true`. |

### Migration errors

| Error | Meaning |
|---|---|
| `db.ChecksumMismatchError` | Someone edited an applied migration's file after the migrator applied it. |
| `db.DownChecksumMismatchError` | A down file's checksum does not match the value the migrator recorded when the up ran. |
| `db.MigrationExecutionError` | The SQL itself failed. |
| `db.LockAcquisitionError` | Could not obtain the advisory lock. |
| `db.MissingMigrationFileError` | Database records a version the disk does not have. |
| `db.NoDownMigrationError` | Rollback requested for a version with no `.down.sql`. |
| `db.ErrLockNotAcquired` | Sentinel returned when a non-blocking lock attempt fails. |

## Engine configs

`piko.WithDatabase(name, registration)` accepts a `*db.DatabaseRegistration`:

```go
piko.WithDatabase("primary", &db.DatabaseRegistration{
    DB:           sqlDB,
    EngineConfig: db_engine_postgres.Postgres(),
})
```

Available engine configs:

| Import | Helper | Dialect |
|---|---|---|
| `wdk/db/db_engine_sqlite` | `SQLite()` | SQLite. |
| `wdk/db/db_engine_postgres` | `Postgres()`, `PostgresPgBouncer()` | PostgreSQL. |
| `wdk/db/db_engine_mysql` | `MySQL()` | MySQL. |
| `wdk/db/db_engine_mariadb` | `MariaDB()` | MariaDB. |
| `wdk/db/db_engine_cockroachdb` | `CockroachDB()` | CockroachDB. |
| `wdk/db/db_engine_duckdb` | `DuckDB()` | DuckDB. |

Each engine config gives the generator type inference and the migrator the correct dialect. Swap the import to change engine. No other application code changes.

### Reserved database names

| Constant | Role |
|---|---|
| `db.DatabaseNameRegistry` | Backs Piko's internal registry (instead of the default in-memory store). |
| `db.DatabaseNameOrchestrator` | Backs the orchestrator queue. |

Register a SQL database under either name to persist that subsystem across restarts.

## Code-generation config

The generator accepts a `DatabaseConfig` that tunes type inference and adds custom functions:

```go
type DatabaseConfig = querier_dto.DatabaseConfig
type TypeOverride = querier_dto.TypeOverride
type CustomFunction = querier_dto.CustomFunctionConfig
```

Declare `TypeOverride` entries to force a SQL type to a specific Go type, or `CustomFunction` entries to register stored functions the engine would not otherwise know about.

## See also

- [How to migrations](../how-to/database/migrations.md).
- [How to writing queries](../how-to/database/queries.md).
- [How to swapping database engines](../how-to/database/swapping-engines.md).
- Scenarios: [022 SQLite](../showcase/022-database-sqlite.md), [023 MySQL](../showcase/023-database-mysql.md), [024 PostgreSQL](../showcase/024-database-postgres.md), [025 DuckDB](../showcase/025-database-duckdb.md).

Integration tests: [`tests/integration/querier_sqlite`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_sqlite), [`querier_postgres`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_postgres), [`querier_mariadb`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_mariadb), [`querier_cockroachdb`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_cockroachdb), [`querier_duckdb`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_duckdb).
