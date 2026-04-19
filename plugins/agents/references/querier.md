# Querier: Migrations and Type-Safe Queries

Piko generates type-safe Go from annotated `.sql` files. Authors write SQL; the generator emits a `Queries` struct whose methods match the annotated queries exactly. The migration service applies schema changes with advisory locking and per-file checksums.

For full API surface and type catalogue see [`/docs/reference/querier.md`](../../../docs/reference/querier.md). For task recipes see [`/docs/how-to/database/migrations.md`](../../../docs/how-to/database/migrations.md), [`/docs/how-to/database/queries.md`](../../../docs/how-to/database/queries.md), and [`/docs/how-to/database/swapping-engines.md`](../../../docs/how-to/database/swapping-engines.md).

## Project layout

```
db/
  schema.go            //go:embed migrations/*.sql -> var Migrations embed.FS
  migrations/
    001_initial.up.sql
    001_initial.down.sql
    002_add_index.up.sql
  queries/
    tasks.sql
    users.sql
  seeds/               (optional)
    001_default_roles.sql
  generated/           (do not hand-edit)
    querier.go models.go prepared.go <name>.sql.go
```

Re-run after any SQL change: `go run ./cmd/generator/main.go all`.

## Migration filenames

Numeric prefix is the version (`int64` parsed from leading digits). Anything after the digits is decorative. Use zero-padded (`001_initial.up.sql`) or timestamped (`20260115143000_initial.up.sql`) - pick one and stick with it. `.down.sql` is optional but recommended; without one, rollback returns `db.NoDownMigrationError`.

```sql
-- 001_initial.up.sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    completed INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);
```

| Directive | Effect |
|---|---|
| `-- piko:no-transaction` (first line) | Skip `BEGIN`/`COMMIT` wrapping. Required for `CREATE INDEX CONCURRENTLY`, `VACUUM`. |

## Query annotations

Every query needs `piko.name` and `piko.command` headers:

```sql
-- piko.name: ListTasks
-- piko.command: many
SELECT id, title, completed, created_at
FROM tasks
ORDER BY created_at DESC;

-- piko.name: CreateTask
-- piko.command: one
INSERT INTO tasks (title, created_at) VALUES (?, ?)
RETURNING id, title, completed, created_at;

-- piko.name: DeleteTask
-- piko.command: exec
DELETE FROM tasks WHERE id = ?;
```

### Command kinds

| Kind | Generated signature | Use |
|---|---|---|
| `one` | `(ctx, params) (<Name>Row, error)` | Single row; `sql.ErrNoRows` on miss. `INSERT ... RETURNING` counts as one. |
| `many` | `(ctx, params) ([]<Name>Row, error)` | Zero or more rows. |
| `exec` | `(ctx, params) error` | Mutation, no result inspection. |
| `execrows` | `(ctx, params) (int64, error)` | Need affected-row count. |
| `execresult` | `(ctx, params) (sql.Result, error)` | Need `LastInsertId` plus `RowsAffected`. |
| `stream` | `(ctx, params, fn func(<Name>Row) error) error` | Large result sets without buffering. |
| `batch` | Batch handle with `Exec`/`Query`/`QueryRow` callbacks | Multiple parameter sets in one round-trip (driver-dependent). |
| `copyfrom` | `(ctx, rows) (int64, error)` | Postgres `COPY FROM` bulk insert. |

### Placeholders

- SQLite, MySQL, MariaDB, DuckDB: `?` (positional)
- PostgreSQL, CockroachDB: `$1`, `$2`

One placeholder -> single typed argument: `func DeleteTask(ctx, p1 int32) error`.
Multiple placeholders -> typed params struct:

```go
type CreateTaskParams struct { P1 string; P2 int32 }
func (q *Queries) CreateTask(ctx context.Context, params CreateTaskParams) (CreateTaskRow, error)
```

### Naming parameters with directives

Bind positional placeholders to named struct fields:

```sql
-- piko.name: FindUser
-- piko.command: one
-- $1 as piko.param(userID)
-- $2 as piko.optional(email)
SELECT id, name FROM users
WHERE id = $1 AND ($2 IS NULL OR email = $2);
```

| Directive | Effect |
|---|---|
| `piko.param(<name>)` | Required field on params struct. |
| `piko.optional(<name>)` | Nullable field; callers may omit. |
| `piko.limit(<name>)` / `piko.offset(<name>)` | Pagination semantics. |
| `piko.sortable(<name>)` | Sort key validation. |
| `piko.slice(<name>)` | `IN (?1)` slice expansion. |

Same syntax works with `?1`/`?2` for SQLite/MySQL. See [`/docs/reference/querier.md`](../../../docs/reference/querier.md) for the full directive surface.

## Generated `Queries`

```go
import generated "myapp/db/generated"

queries := generated.New(database)               // *sql.DB or any DBTX
queries := generated.NewWithReplica(writer, reader)

err := queries.RunInTx(ctx, database, func(q *generated.Queries) error {
    if err := q.DebitAccount(ctx, p1); err != nil { return err }
    return q.CreditAccount(ctx, p2)
})

txQueries := queries.WithTx(tx)                  // manual transaction
```

`NewWithReplica` routes `one`/`many` (without `RETURNING`) to the reader; `exec` and `RETURNING` writes go to the writer.

Resolve registered databases at runtime:

```go
database, err := db.GetDatabaseConnection("primary")  // *sql.DB
reader, err  := db.GetDatabaseReader("primary")       // DBTX
writer, err  := db.GetDatabaseWriter("primary")       // DBTX
```

Call from action `Call` methods only - never from template expressions or partial `Render`.

## Migration service

```go
import (
    "piko.sh/piko/wdk/db"
    "piko.sh/piko/wdk/db/db_engine_sqlite"
    mydb "myapp/db"
)

executor := db.NewMigrationExecutor(database, db.SQLiteDialect())
reader   := db.NewFSFileReader(mydb.Migrations)            // io/fs.FS
migrator := db.NewMigrationService(executor, reader, "migrations",
    db.WithNonBlockingLock(),                              // optional
)

if _, err := migrator.Up(ctx); errors.Is(err, db.ErrLockNotAcquired) {
    // another process holds the lock
}
```

### Methods

| Method | Returns | Use |
|---|---|---|
| `Up(ctx)` | `(int, error)` | Apply every pending migration. |
| `UpTo(ctx, version)` | `(int, error)` | Apply up to and including `version`. |
| `Down(ctx, steps)` | `(int, error)` | Roll back the last `steps`. |
| `DownTo(ctx, version)` | `(int, error)` | Roll back down to (excluding) `version`. |
| `Status(ctx)` | `([]MigrationStatus, error)` | Per-file audit (Applied, ChecksumMatch, Dirty, LastStatement). |
| `Validate(ctx)` | `error` | Checksum-only audit, no execution. CI gate. |

### Options

`WithNonBlockingLock()`, `WithBeforeMigration(hook)`, `WithAfterMigration(hook)`, `WithBeforeRun(hook)`, `WithAfterRun(hook)`.

### Dialects

`db.SQLiteDialect()`, `db.PostgresDialect()`, `db.PostgresPgBouncerDialect()`, `db.MySQLDialect()`, `db.MySQLDialectWithDSN(dsn)`.

### Errors

`db.ChecksumMismatchError`, `db.DownChecksumMismatchError`, `db.MigrationExecutionError`, `db.LockAcquisitionError`, `db.MissingMigrationFileError`, `db.NoDownMigrationError`, `db.ErrLockNotAcquired` (sentinel - use `errors.Is`).

Recovery from interrupted runs is automatic: dirty rows resume from the next statement on the next `Up`. Recovery from checksum mismatch: revert the edit, add a new migration, or update the recorded checksum manually after review.

## Seed service

For reference data (countries, default roles, demo content). Same shape as migrations; recorded in `piko_seeds`. Files live under `db/seeds/`. Make seeds idempotent (`INSERT ... ON CONFLICT DO NOTHING`).

```go
seedExecutor := db.NewSeedExecutor(database, db.SQLiteDialect())
seeds := db.NewSeedService(seedExecutor, db.NewFSFileReader(mydb.Seeds), "seeds")
if err := seeds.Apply(ctx); err != nil { panic(err) }
```

Registered via `WithDatabase` (with `SeedFS` on the registration) and resolved with `db.GetSeedService(name)`.

## Engine registration

```go
piko.WithDatabase("primary", &db.DatabaseRegistration{
    DB:           sqlDB,
    EngineConfig: db_engine_postgres.Postgres(),
    MigrationFS:  mydb.Migrations,
    SeedFS:       mydb.Seeds,
})
```

Engine factories: `db_engine_sqlite.SQLite()`, `db_engine_postgres.Postgres()` / `PostgresPgBouncer()`, `db_engine_mysql.MySQL()`, `db_engine_mariadb.MariaDB()`, `db_engine_cockroachdb.CockroachDB()`, `db_engine_duckdb.DuckDB()`.

Reserved names: `db.DatabaseNameRegistry` (framework registry), `db.DatabaseNameOrchestrator` (orchestrator queue) - register a SQL database under either to persist that subsystem.

## LLM mistake checklist

- Forgetting `-- piko.name:` and `-- piko.command:` directives (queries do not generate)
- Mixing placeholder styles (`?` vs `$1`) - dialect picks one
- Missing `.up.sql`/`.down.sql` pair (rollback returns `db.NoDownMigrationError`)
- Editing an applied migration on disk (returns `db.ChecksumMismatchError`; only safe rollback is to write a new migration)
- Wrapping `CREATE INDEX CONCURRENTLY` / `VACUUM` in a transaction (use `-- piko:no-transaction` on the first line)
- Forgetting to run `go run ./cmd/generator/main.go all` after changing query files
- Calling `migrator.Up` after `piko.Run` (run migrations before the server starts accepting traffic)
- Importing the SQLite cgo driver without CGO enabled (use `db_driver_sqlite_nocgo`)

## Related

- `references/wdk-data.md` - wider WDK data surface (storage, cache, persistence overview)
- `/docs/reference/querier.md` - full method/type catalogue
- `/docs/how-to/database/queries.md` - writing queries in depth
- `/docs/how-to/database/migrations.md` - migrations in depth
- `/examples/scenarios/022_database_sqlite/`, `023_database_mysql/`, `024_database_postgres/`, `025_database_duckdb/` - working scenarios
