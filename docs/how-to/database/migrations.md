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

The prefix is the version key. The migration runner parses `int64` from the leading digits of the filename, so any pattern that starts with digits works. Use zero-padded integers (`001`, `002`) or full timestamps (`20260115`, `20260115143000`). Anything after the digits - including a `T` separator or descriptive name - is decorative and never participates in version comparison. Pick one scheme and stick with it.

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

Writing the down file is optional but recommended. Without one, rolling that version back returns a `db.NoDownMigrationError`.

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

`migrator.Up` returns the count of migrations applied during the call. Log the count, or call `migrator.Status` afterwards for a per-file audit trail.

## Choose the right dialect

Swap `db.SQLiteDialect()` for your target database:

| Dialect | Lock style | Notes |
|---|---|---|
| `db.SQLiteDialect()` | No-op (`NoOpLock`) | SQLite has no advisory locks. The dialect installs a no-op lock and relies on SQLite's own single-writer serialisation; the migration runner does not add anything on top. |
| `db.PostgresDialect()` | Advisory lock | Standard Postgres deployments. |
| `db.PostgresPgBouncerDialect()` | Table-based lock | For Postgres behind PgBouncer in transaction mode, where advisory locks do not persist across queries. |
| `db.MySQLDialect()` | `GET_LOCK()` | Works for MySQL and MariaDB. |
| `db.MySQLDialectWithDSN(dsn)` | `GET_LOCK()` | Same as `MySQLDialect` but detects `multiStatements=true` and disables statement splitting when the driver handles it natively. |

## Roll forward and back

The migration service exposes four directional methods plus `Status`:

| Method | Signature | Use |
|---|---|---|
| `Up` | `Up(ctx) (int, error)` | Apply every pending migration. |
| `UpTo` | `UpTo(ctx, targetVersion int64) (int, error)` | Apply pending migrations up to and including the target version. |
| `Down` | `Down(ctx, steps int) (int, error)` | Roll back the last `steps` applied migrations. |
| `DownTo` | `DownTo(ctx, targetVersion int64) (int, error)` | Roll back applied migrations down to (but not including) the target version. |

```go
migrator.Down(ctx, 1)         // Undo the most recent migration.
migrator.DownTo(ctx, int64(1)) // Roll back everything applied after version 1.
migrator.UpTo(ctx, int64(2))   // Apply pending migrations up to version 2.
```

The integer argument to `DownTo` and `UpTo` is the numeric prefix parsed from the migration filename (`001_initial.up.sql` -> `1`). Both rollback methods require a `.down.sql` for every version they touch.

## Inspect status

```go
status, err := migrator.Status(ctx)
for _, m := range status {
    fmt.Printf("%d %s applied=%v\n", m.Version, m.Filename, m.Applied)
}
```

`MigrationStatus` exposes `Name`, `Filename`, `Version` (`int64`), `Applied`, `AppliedAt`, `ChecksumMatch`, `HasDownMigration`, `Dirty`, and `LastStatement`. `LastStatement` is a `*int`. It is `nil` for legacy applied rows that the runner recorded before per-statement tracking landed. Otherwise it points at the index of the statement that was running when a previous attempt failed. Use this in a health probe or a CLI helper to surface migration drift without querying the database directly.

For a checksum-only audit that does not touch any pending migrations, call `migrator.Validate(ctx)`. It walks the applied rows, recomputes each file's checksum, and returns an error if any have drifted - useful as a CI gate.

## Skip the transaction wrapper

By default the executor wraps each migration in `BEGIN`/`COMMIT`. Some statements (`CREATE INDEX CONCURRENTLY` on PostgreSQL, `VACUUM` on SQLite, certain DDL on MySQL) refuse to run inside a transaction. Add the directive `-- piko:no-transaction` as the first line of the migration file to opt that file out of transaction wrapping:

```sql
-- piko:no-transaction
CREATE INDEX CONCURRENTLY tasks_completed_idx ON tasks (completed);
```

The directive applies per file. Other migrations stay transactional.

## Lock acquisition mode

`db.NewMigrationService` blocks waiting for the dialect lock by default. Pass `db.WithNonBlockingLock()` to fail fast when another process is mid-migration:

```go
migrator := db.NewMigrationService(executor, reader, "migrations",
    db.WithNonBlockingLock(),
)

if _, err := migrator.Up(ctx); errors.Is(err, db.ErrLockNotAcquired) {
    // another process holds the lock - retry, exit, or surface the state
}
```

This is the right default for short-lived CLI invocations or sidecar containers, where waiting is worse than retrying later.

## Recover from interrupted runs (dirty migrations)

If a process dies mid-migration, the row in `piko_migrations` keeps `dirty = 1` and `last_statement` pointing at the index of the statement that was running when the run died. On the next call to `Up`, the service spots the dirty row, lines it up with the matching pending file, and resumes from the next statement. Previously committed work is not redone. No manual cleanup applies for the common case of "retry the same migration after a transient failure".

If the dirty record does not line up with the next pending file, the run fails fast with a domain-level dirty-migration error. This happens when the file is missing, when someone renumbered it, or when it sits behind other unrelated pending versions. The run refuses to apply anything else until the operator resolves the inconsistency.

## Recover from checksum mismatch

If you edit an applied migration file after it ran, `Up` returns `db.ChecksumMismatchError`. The service protects against silently losing track of schema evolution. Options to recover:

1. **Revert the edit.** Preferred. Keep history faithful and add a new migration for the change.
2. **Add a new migration that describes the edit.** Preserves auditability.
3. **Update the recorded checksum manually.** Only when you know the edit is byte-compatible (whitespace, comments) and you have reviewed the implications.

Never edit a migration file that has run in production without a conscious decision about which of the three paths applies.

Use `errors.As` to extract structured fields from typed migration errors and `errors.Is(err, db.ErrLockNotAcquired)` for the sentinel. See [Migration errors](../../reference/querier.md#migration-errors) for the full catalogue.

## Lifecycle hooks

The migration service accepts `MigrationServiceOption` values for lifecycle hooks:

- `BeforeMigrationHook`: runs before each migration's SQL.
- `AfterMigrationHook`: runs after each migration.
- `BeforeRunHook`: runs before any migration.
- `AfterRunHook`: runs after the full run.

Use hooks to log, emit metrics, or gate on environment checks.

## Apply seed data

For reference data and fixtures - country lists, default roles, demo content - use the seed service instead of smuggling `INSERT` statements into schema migrations. Seed files live under their own directory (typically `db/seeds/`) and follow the same versioned naming as migrations (`001_countries.sql`, `002_default_roles.sql`). The runner records applied versions in a `piko_seeds` table so it runs each file at most once.

```go
seedExecutor := db.NewSeedExecutor(database, db.SQLiteDialect())
seedReader := db.NewFSFileReader(mydb.Seeds)
seeds := db.NewSeedService(seedExecutor, seedReader, "seeds")

if err := seeds.Apply(ctx); err != nil {
    panic(err)
}
```

`SeedService` exposes `Apply`, `Status`, and the underlying `AppliedSeed` records via the same shape as `MigrationStatus`. When you register seeds through `WithDatabase` (with a `SeedFS` on the registration), the framework wires the service automatically and you can resolve it with `db.GetSeedService(name)`.

Seeds should be idempotent at the SQL level. Use `INSERT ... ON CONFLICT DO NOTHING` or equivalent so that re-running a seed file in a recovery scenario stays safe even after the history table loses its rows.

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
- Scenarios: [022 SQLite](../../../examples/scenarios/022_database_sqlite/), [023 MySQL](../../../examples/scenarios/023_database_mysql/), [024 PostgreSQL](../../../examples/scenarios/024_database_postgres/).
