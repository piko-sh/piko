---
title: How to write and run database migrations
description: Lay out migration files, run them on startup, write rollbacks, and handle checksum mismatches.
nav:
  sidebar:
    section: "how-to"
    subsection: "database"
    order: 10
---

# How to write and run database migrations

Piko's migration service applies SQL files in version order, records a checksum per applied file, and refuses to run if a previously applied file has changed on disk. This guide covers writing migrations, running them at startup, and recovering when a checksum goes wrong. See the [querier reference](../../reference/querier.md) for the API surface.

## Lay out migration files

Migrations live under `db/migrations/` as paired `.up.sql` and `.down.sql` files, numbered for ordering:

```
db/
  schema.go
  migrations/
    001_initial.up.sql
    001_initial.down.sql
    002_add_comments.up.sql
    002_add_comments.down.sql
```

The prefix is a sortable version key. Any sortable scheme works. Common choices are zero-padded integers (`001`, `002`) or timestamps (`20260115T1430_initial`). Pick one and stick with it.

A minimal pair:

**`001_initial.up.sql`**:

```sql
CREATE TABLE tasks (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    title      TEXT NOT NULL,
    completed  INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);

CREATE INDEX tasks_created_at_idx ON tasks (created_at DESC);
```

**`001_initial.down.sql`**:

```sql
DROP INDEX tasks_created_at_idx;
DROP TABLE tasks;
```

Writing the down file is optional but recommended. Without one, `migrator.Down(...)` for that version returns `NoDownMigrationError`.

## Embed migrations in the binary

Expose the migration folder from a package in `db/`. This lets the binary ship with its own schema.

**`db/schema.go`**:

```go
// Package db provides embedded database migrations.
package db

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS
```

Build-time errors surface if a migration file contains malformed SQL or the embed directive is wrong.

## Run migrations at startup

Apply `Up` before the server starts accepting traffic. A typical `main.go`:

```go
package main

import (
    "context"
    "database/sql"
    "os"

    _ "modernc.org/sqlite"

    "piko.sh/piko"
    "piko.sh/piko/wdk/db"
    "piko.sh/piko/wdk/db/db_engine_sqlite"

    mydb "myapp/db"
)

func main() {
    database, err := sql.Open("sqlite", "file:./data/app.db")
    if err != nil {
        panic(err)
    }

    executor := db.NewMigrationExecutor(database, db.SQLiteDialect())
    reader := db.NewFSFileReader(mydb.Migrations)
    migrator := db.NewMigrationService(executor, reader, "migrations")

    if _, err := migrator.Up(context.Background()); err != nil {
        panic(err)
    }

    ssr := piko.New(
        piko.WithDatabase("primary", &db.DatabaseRegistration{
            DB:           database,
            EngineConfig: db_engine_sqlite.SQLite(),
        }),
    )

    if err := ssr.Run(piko.RunModeDev); err != nil {
        panic(err)
    }
}
```

`migrator.Up` returns the list of applied migrations. Log it for an audit trail if your project needs one.

## Choose the right dialect

Swap `db.SQLiteDialect()` for your target database:

| Dialect | Lock style | Notes |
|---|---|---|
| `db.SQLiteDialect()` | Single-writer | SQLite has no advisory locks; the migration service uses transaction-level serialisation. |
| `db.PostgresDialect()` | Advisory lock | Standard Postgres deployments. |
| `db.PostgresPgBouncerDialect()` | Table-based lock | For Postgres behind PgBouncer in transaction mode, where advisory locks do not persist across queries. |
| `db.MySQLDialect()` | `GET_LOCK()` | Works for MySQL and MariaDB. |
| `db.MySQLDialectWithDSN(dsn)` | `GET_LOCK()` | Same as `MySQLDialect` but detects `multiStatements=true` and disables statement splitting when the driver handles it natively. |

## Roll back a migration

`Down` rolls back the most recent migration. `DownTo` rolls back to a specific version.

```go
migrator.Down(ctx)                     // Undo the most recent one.
migrator.DownTo(ctx, "001_initial")    // Roll back everything after 001.
```

Both methods require a `.down.sql` for every version they touch.

## Inspect status

```go
status, err := migrator.Status(ctx)
for _, m := range status {
    fmt.Printf("%s applied=%v\n", m.Version, m.Applied)
}
```

Use this in a health probe or a CLI helper to surface migration drift without querying the database directly.

## Recover from checksum mismatch

If you edit an applied migration file after it ran, `Up` returns `db.ChecksumMismatchError`. The service protects against silently losing track of schema evolution. Options to recover:

1. **Revert the edit.** Preferred. Keep history faithful and add a new migration for the change.
2. **Add a new migration that describes the edit.** Preserves auditability.
3. **Update the recorded checksum manually.** Only when you know the edit is byte-compatible (whitespace, comments) and you have reviewed the implications.

Never edit a migration file that has run in production without a conscious decision about which of the three paths applies.

## Lifecycle hooks

The migration service accepts `MigrationServiceOption` values for lifecycle hooks:

- `BeforeMigrationHook`: runs before each migration's SQL.
- `AfterMigrationHook`: runs after each migration.
- `BeforeRunHook`: runs before any migration.
- `AfterRunHook`: runs after the full run.

Use hooks to log, emit metrics, or gate on environment checks.

## Combine with a Lifecycle component

For services that need migrations and graceful shutdown wired together, use a [lifecycle component](../lifecycle.md):

```go
type DatabaseComponent struct {
    db       *sql.DB
    migrator db.MigrationService
}

func (c *DatabaseComponent) OnStart(ctx context.Context) error {
    _, err := c.migrator.Up(ctx)
    return err
}
```

This pattern groups database startup with its health probe (see the [health checks how-to](../health-checks.md)) and cleans up on shutdown.

## See also

- [Querier reference](../../reference/querier.md).
- [How to writing queries](queries.md).
- [How to swapping database engines](swapping-engines.md).
- [How to lifecycle](../lifecycle.md) for running migrations as a managed lifecycle component.
- Scenarios: [022 SQLite](../../showcase/022-database-sqlite.md), [023 MySQL](../../showcase/023-database-mysql.md), [024 PostgreSQL](../../showcase/024-database-postgres.md).
