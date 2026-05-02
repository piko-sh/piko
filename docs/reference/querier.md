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

The project's scaffolded generator (`cmd/generator/main.go`) reads every `.sql` file under `migrations/` and `queries/` and writes type-safe Go under `generated/`. The output is stale until the generator runs again. For the command to run it, see the how-to guides on [writing queries](../how-to/database/queries.md) and [migrations](../how-to/database/migrations.md).

## Query annotations

Every query in `queries/*.sql` starts with a single `piko.query(...)` header
that carries the query name plus the command kind:

```sql
-- piko.query(ListTasks, many)
SELECT id, title, completed, created_at
FROM tasks
ORDER BY created_at DESC;
```

### Grammar

The header is a function-call-with-keyword-arguments grammar. Each
directive accepts zero or more positional arguments followed by zero
or more `key: value` keyword arguments:

```
-- piko.query(<positionals>, <key>: <value>, ...)
-- $N as piko.<role>(<positionals>, <key>: <value>, ...)
-- piko.<role>(<positionals>, <key>: <value>, ...)
```

Required arguments are positional. Optional arguments are keyword
arguments. **You can also name every positional**, so all
three of these are equivalent:

```sql
-- piko.query(GetUser, one)                     -- positional
-- piko.query(name: GetUser, command: one)      -- keyword-as-positional
-- piko.query(GetUser, command: one)            -- mixed
```

The three forms are interchangeable. The generator accepts a positional
argument and its named keyword-as-positional equivalent for any required
argument.

### Top-level header: `piko.query(...)`

Multi-line layout works when the call gets long. The parser
continues across comment-prefixed lines while the paren depth stays
open:

```sql
-- piko.query(
--   SearchOrders,
--   many,
--   dynamic: runtime,
--   group_by: orders.id,
-- )
```

**Positionals** (in order):

| Position | Name | Type | Purpose |
|---|---|---|---|
| 1 | `name` | identifier (required) | Go method name on `Queries`. PascalCase. |
| 2 | `command` | enum (required) | Execution pattern. See command kinds below. |

**Keyword arguments**:

| Key | Type | Purpose |
|---|---|---|
| `dynamic` | `runtime` | Emit a fluent runtime builder instead of a static method. |
| `readonly` | `true`/`false` | Override automatic read/write detection. |
| `nullable` | `true`/`false` | Override automatic nullability propagation across the output columns. |
| `group_by` | qualified column (for example `orders.id`) | Declare the grouping column that collapses repeated parent rows in an embed join. |

### Command kinds

| Kind | Generated signature | Return shape |
|---|---|---|
| `one` | `(Queries).<Name>(ctx, params) (<Name>Row, error)` | Single row; `sql.ErrNoRows` on miss. |
| `many` | `(Queries).<Name>(ctx, params) ([]<Name>Row, error)` | Slice of rows. |
| `exec` | `(Queries).<Name>(ctx, params) error` | No rows returned; used for INSERT/UPDATE/DELETE without `RETURNING`. |
| `asyncexec` | `(Queries).<Name>(ctx, params) error` | Same wire signature as `exec`; the generated method carries a doc comment documenting that the call returns when the server accepts the queued mutation, not when it has finished applying. Used for engines whose mutation semantics are server-side asynchronous (ClickHouse `ALTER UPDATE` / `ALTER DELETE`). |
| `execrows` | `(Queries).<Name>(ctx, params) (int64, error)` | Returns the number of rows affected by the statement. |
| `execresult` | `(Queries).<Name>(ctx, params) (sql.Result, error)` | Returns the raw `sql.Result` so callers can read both `LastInsertId` and `RowsAffected`. |
| `stream` | `(Queries).<Name>(ctx, params, fn func(<Name>Row) error) error` | Streams rows to the callback without buffering the full result set. |
| `batch` | `(Queries).<Name>(ctx, items []<Name>Params) (*<Name>BatchResults, error)` | Pipelines one statement per item through the underlying driver's batch API (Postgres only); the result type carries `Close()` and `Exec(fn)`. |
| `copyfrom` | `(Queries).<Name>(ctx, items []<Name>Params) (int64, error)` | Streams rows via the Postgres `COPY FROM STDIN` protocol. Returns the row count. |

### Dynamic runtime builder

`dynamic: runtime` emits a fluent builder instead of a fixed method, so callers compose the
filter, ordering, and pagination at runtime:

```go
rows, err := queries.SearchPosts(ctx).
    Where("category", "=", "tech").
    Where("views", ">=", 100).
    OrderBy("published", "DESC").
    Limit(20).
    Offset(40).
    All(ctx) // also .One(ctx) and .Count(ctx)
```

When the terminal runs, Piko validates every dynamic input against a closed allow-list.
The builder rejects column names, operators, and ORDER BY directions outside their
allow-list with a sentinel error. The sentinel is `piko: unknown column in runtime query
filter` plus its operator and direction equivalents. The builder never interpolates them
into SQL, and an invalid input makes the terminal return that error instead of panicking.
Piko always sends bound values as driver parameters, so a value can never alter the
statement's structure.

**Dynamic queries can only filter and order over the data they select.** The column allow-list
is the query's SELECT projection. A column becomes filterable and orderable only when it
appears in the SELECT list, whether directly, via `*`, or through a CTE that projects it. A
column that exists on a FROM or JOIN table but stays absent from the projection falls outside
the allow-list. The builder rejects `.Where("password_hash", "LIKE", "a%")` or
`.OrderBy("password_hash", "ASC")` on a query that does not select `password_hash`. To filter
or order by a column, select it. Computed expressions and aggregates, which have no underlying
source column, are likewise not filterable or orderable as bare identifiers.

### Parameter directives

Parameter placeholders in the SQL (`$1`/`$2` for Postgres, `?` for
SQLite/MySQL, `:name`/`@name` for named-parameter engines) become
typed Go parameters via a per-placeholder directive line:

```sql
-- piko.query(SearchOrders, many)
-- $1 as piko.param(status, optional: true)
-- $2 as piko.param(ids, type: int, kind: slice)
-- piko.sortable(order_by, [name, total, placed_at])
-- $3 as piko.param(page_size, default: 10, max: 100)
-- $4 as piko.param(skip)
SELECT id, status, total FROM orders
WHERE ($1::text IS NULL OR status = $1) AND id = ANY($2)
LIMIT $3 OFFSET $4;
```

Each role:

#### `piko.param(name, type:, nullable:, default:)`

The default binding. Use it when you want to give the placeholder a
Go-friendly name and optionally override its inferred SQL type or
nullability.

```sql
-- $1 as piko.param(user_id, type: int8, nullable: false)
```

#### `piko.param` qualities: `optional:` and `kind: slice`

`piko.param` carries keyword qualities that shape the input instead of acting as separate
directives.

`optional: true` makes the parameter nullable. When the caller passes nil, the generator emits
a runtime predicate that drops the matching WHERE clause (the `$1 IS NULL OR column = $1` idiom
collapses to this). It is mutually exclusive with `default:`.

```sql
-- $1 as piko.param(status, optional: true)
SELECT id, status FROM orders WHERE status = $1;
```

`kind: slice` expands a Go slice into IN-list placeholders at call time. The element type can be
overridden with `type:` when inference is too loose.

```sql
-- $1 as piko.param(ids, type: int, kind: slice)
SELECT * FROM users WHERE id IN ($1);
```

#### `piko.sortable(name, [columns])`

A validated dynamic ORDER BY. The second positional is a bracketed
list of allowed column names. Passing any value outside the list at
call time fails fast. `piko.sortable` is a standalone header. It
binds no placeholder, so it carries no `$N as` prefix, and the SELECT
carries no `ORDER BY` clause. The generator appends the `ORDER BY`
from the validated column enum.

```sql
-- piko.sortable(order_by, [name, price, created_at])
SELECT id, name, price FROM products;
```

You can also supply the `columns` list as a keyword argument
(`columns: [name, price, created_at]`) for long lists where
breaking the call across lines reads better.

#### LIMIT / OFFSET parameters

The generator types a `piko.param` placed in a `LIMIT` or `OFFSET` clause as a non-negative
integer from its clause position. No special directive applies. Add the `default:` and `max:`
qualities to clamp the value at call time. A default fills a zero value, and max caps it.

```sql
-- $1 as piko.param(page_size, default: 10, max: 100)
-- $2 as piko.param(page_offset)
SELECT id, name FROM products LIMIT $1 OFFSET $2;
```

### Header directive: `piko.embed`

Groups projection columns into a nested Go struct. The required
positional is the source table or view. The `from:` keyword names
the FROM-clause alias whose columns belong to this embed group:

```sql
-- piko.query(GetBookWithAuthor, one)
-- piko.embed(authors, from: a)
SELECT b.id, b.title, a.id, a.name
FROM books b
INNER JOIN authors a ON a.id = b.author_id
WHERE b.id = $1;
```

The generated row type carries `Authors AuthorsRow` (or whatever
name you set via `as:`).

| Position / Key | Type | Purpose |
|---|---|---|
| 1 (positional) | `table` | Source table or view whose columns the embed group holds. |
| `from:` | identifier | FROM-clause alias whose columns belong to this embed group. |
| `as:` | identifier | Override the Go field name (defaults to the table name in Go-cased form). |

### Type override: `piko.column`

Override the inferred SQL type, the custom Go destination type, or
the nullability of one output column. Valid both in query headers
(per-query override) and in migration files (catalogue-wide override
that propagates through every passthrough query).

**Per-query override**, which pins a derived column's type when the
analyser cannot infer it precisely:

```sql
-- piko.query(GetUserEmailLower, one)
-- piko.column(email_lower, type: text)
SELECT id, LOWER(email) AS email_lower FROM users WHERE id = $1;
```

**Custom Go destination type**, which maps a column to a user-defined
Go type (the generated file imports the package automatically):

```sql
-- piko.query(GetUserUUID, one)
-- piko.column(id, go_type: "github.com/google/uuid.UUID")
SELECT id, email FROM users WHERE id = $1;
```

The generated row struct uses `uuid.UUID` instead of the default
string and the file gains `import "github.com/google/uuid"`. Nullable
columns become `*uuid.UUID` automatically.

**Migration-level (schema-wide) override**, declared once above the
`CREATE TABLE`. Every passthrough query that selects the column
inherits the override:

```sql
-- piko.column(users.id, go_type: "github.com/google/uuid.UUID")
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL
);
```

```sql
-- piko.query(GetUser, one)
SELECT id, email FROM users WHERE id = $1;   -- ID field is uuid.UUID

-- piko.query(GetUserAsString, one)
SELECT id::text AS id_str FROM users;        -- ID_str field is string
```

The override propagates only through *direct passthrough
projections*. Any cast (`::type`, `CAST(...)`), function call, or
arithmetic operation drops the override and falls back to the
analyser's inferred type. Re-apply per-query with another
`piko.column` line when you need the override on a derived column.

**Precedence**:

1. Query-level `piko.column` directive wins over migration-level.
2. Migration-level `piko.column` wins over inferred type.
3. `type:` and `go_type:` are mutually exclusive on one directive
   (the generator raises a Q038 diagnostic when a directive declares both).
4. `nullable:` works alongside either `type:` or `go_type:`.

| Position / Key | Type | Purpose |
|---|---|---|
| 1 (positional) | `name` | Output column name (query header) or qualified `table.column` (migration). |
| `type:` | SQL type | Override the analyser's inferred SQL type; mapped through the engine's type registry. Mutually exclusive with `go_type:`. |
| `go_type:` | quoted string | Explicit Go destination type with import path (for example `"github.com/google/uuid.UUID"`). The generator imports the package automatically. Mutually exclusive with `type:`. |
| `nullable:` | `true`/`false` | Override inferred nullability. Nullable + custom `go_type:` produces `*Type` wrapping. |

### Migration directive: `piko.migration`

In a migration file (`*.up.sql`), `piko.migration(readonly: ...)` overrides the read-only
detection on the `CREATE FUNCTION` immediately following it. Useful when you have a function
that is logically read-only but the SQL engine cannot tell (Postgres `STABLE`/`IMMUTABLE` vs
`VOLATILE`, MySQL `MODIFIES SQL DATA`, etc.).

```sql
-- piko.migration(readonly: true)
CREATE FUNCTION calculate_score(item_id INT) RETURNS INT ...

-- piko.migration(readonly: false)
CREATE FUNCTION reset_counter() RETURNS VOID ...
```

`piko.migration` also accepts `no_transaction: true` above a statement that cannot run inside a
transaction. The generator ignores `piko.migration` in a query file and ignores `piko.query`
in a migration file (each raises a Q044 warning).

For a query's read-only flag, use the keyword on the query header instead:

```sql
-- piko.query(BareReadOnly, many, readonly: true)
SELECT id, name FROM items;
```

## Parameter types

Positional parameters in SQL (`?` for SQLite/MySQL, `$1/$2` for Postgres) map to Go in one of two shapes.

**Single parameter** (one `?` or `$1`):

```sql
-- piko.query(DeleteTask, exec)
DELETE FROM tasks WHERE id = ?;
```

Generator emits:

```go
func (q *Queries) DeleteTask(ctx context.Context, p1 int32) error
```

**Multiple parameters** (two or more):

```sql
-- piko.query(CreateTask, one)
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

`NewWithReplica` routes read-only queries to the reader and writing queries to the writer, based on the analysed query's read/write nature. A query that only reads goes to the reader. Any statement that modifies data, including a writing CTE or a call to a data-modifying function, goes to the writer. The `readonly:` keyword on the query header overrides the automatic detection.

### `models.go`

Go structs for every table defined by migrations. Field names come from column names using Go capitalisation.

### `prepared.go`

Prepared-statement cache (used automatically).

### `<name>.sql.go`

One file per `queries/<name>.sql`. Each contains the typed methods for that file's queries.

## Diagnostics

The generator emits diagnostics with stable `Q`-prefixed codes so
LSP plugins, IDE integrations, and CI scripts can match against
them. The generator reserves numbers and never re-uses them. It retains
a gap (`Q021`) for forward-compatibility.

| Code | Severity | Meaning |
|---|---|---|
| `Q001` | error | Unknown column. The column reference does not resolve in any table or alias in scope. |
| `Q002` | error | Ambiguous column. The column name matches more than one table in the current scope. |
| `Q003` | error | Unknown table. The referenced table, CTE, or table-valued function is not in the catalogue. |
| `Q004` | error | Expression type error. Type inference failed on an expression. |
| `Q005` | error | Unknown function. The function call has no matching overload for the argument count. |
| `Q006` | error | Duplicate query name. Two queries declare the same `name`. |
| `Q007` | error | Directive syntax error (lexer catch-all). Unexpected character, unterminated input, malformed token. |
| `Q008` | error | Missing required directive (legacy code; see `Q031` for the per-positional flavour). |
| `Q009` | error | Command/output mismatch. The declared `command` does not match the query's actual output (for example `exec` on a `SELECT`). |
| `Q010` | error | SQL parse failure during catalogue building. |
| `Q011` | error | `piko.sortable` references a column that is not in the query output. |
| `Q012` | warn  | A query file contains more than one SQL statement. The generator analyses only the last one. |
| `Q013` | error | A parameter assigns to a generated (non-writable) column. |
| `Q014` | error | `group_by` references a column that is not in the query output. |
| `Q015` | error | `group_by` is present but no `piko.embed` directive exists on non-key tables. |
| `Q016` | error | The query pairs `group_by` with a command other than `many`. |
| `Q017` | error | A `kind: slice` parameter cannot accompany `batch` or `copyfrom` commands. |
| `Q018` | error | A `kind: slice` parameter cannot accompany `dynamic: runtime`. |
| `Q019` | error | A `kind: slice` parameter cannot also be sortable. Sortable is a standalone directive, so this no longer occurs. |
| `Q020` | error | Compound query (UNION/INTERSECT/EXCEPT) branches have different column counts. |
| `Q022` | info  | A `dynamic: runtime` query's static SQL already contains a `WHERE`, so the builder appends with ` AND `. |
| `Q023` | info  | The generated `<Name>CountSQL` constant wraps the original `SELECT` because the query has `GROUP BY`, `DISTINCT`, or a window function. |
| `Q024` | warn  | A `dynamic: runtime` query could not be rewritten into a COUNT query, so the generated builder has no `.Count(ctx)`. The query still generates, but the count terminal is unavailable. |
| `Q025` | error | A `dynamic: runtime` query's static SQL ends with `;`. Strip the trailing semicolon. |
| `Q026` | error | Unknown directive. Carries a `Suggestion` field with the closest Levenshtein match. |
| `Q027` | error | Unknown keyword argument key for the directive. Carries a `Suggestion`. |
| `Q028` | error | Invalid keyword-argument value. Closed-enum failure carries a `Suggestion`; shape failures (non-integer, non-list) do not. |
| `Q029` | error | Duplicate keyword argument key in one directive call (also fires when a positional and a same-named keyword-as-positional fill the same slot). |
| `Q030` | error | Internal nil guard during type resolution. A defensive check that should not normally fire. |
| `Q031` | error | Missing required positional argument or keyword argument on a directive call. |
| `Q032` | error | Unclosed directive call. Multi-line paren depth never returned to zero. |
| `Q033` | error | Parameter-binding syntax error (expected `as` / `piko` / role name in a `$N as piko.X(...)` chain). Carries a `Suggestion` for near-miss typos. |
| `Q034` | error | Invalid list literal. The closing token of a bracketed list was not `]` (commonly a stray `)`). The generator rejects the malformed value. |
| `Q035` | error | Unterminated string literal. The generator reports it before multi-line continuation so it surfaces the real cause instead of a cascade into `Q032`. |
| `Q036` | error | `piko.column` directive in a query header references an output column name that is not in the SELECT projection. Carries a `Suggestion` of the closest actual output column name. |
| `Q037` | error | `piko.column` directive in a migration file references a `table.column` that does not exist in the catalogue. Carries a `Suggestion` of the closest known qualified column reference. |
| `Q038` | error | `piko.column` declared both `type:` and `go_type:` on the same directive. The two are mutually exclusive. The quick-fix is to delete the second one. |
| `Q039` | error | A single query block contained more directive-eligible logical lines than the parser processes. A defence-in-depth cap set well above any realistic directive header. |
| `Q040` | hint | A query using `exec` runs against an engine that surfaces asynchronous mutations and the underlying statement is one of those async forms (ClickHouse `ALTER UPDATE` / `ALTER DELETE`). Switching the command to `asyncexec` surfaces the fire-and-forget semantics so callers do not mistake a nil error for "mutation has completed". Hint-level so existing projects do not fail their CI on upgrade. |
| `Q041` | error  | A query declared with the `asyncexec` command landed on an engine whose mutation semantics are synchronous (for example postgres, sqlite, mysql). The command is meaningful only on engines that distinguish acceptance from completion. Pick `exec` / `execresult` / `execrows` instead. |
| `Q042` | warn  | A parameter directive (named or positional) names a parameter whose placeholder never appears in the query body. |
| `Q043` | warn  | A migration file mixes a non-transactional statement (a `piko.migration(no_transaction: true)` directive or an auto-detected statement such as `CREATE INDEX CONCURRENTLY`) with other statements, so the whole migration runs without a transaction and is not atomic. Move the non-transactional statement into its own migration file. |
| `Q044` | warn  | A directive sits in the wrong file: a `piko.migration` directive in a query file, or a `piko.query` header in a migration file. The parser ignores the directive. |
| `Q045` | warn  | An engine function resolver returned a resolution without declaring its data access, so the generator defaults the function to read-only. |

LSP plugins can use the `Suggestion` field on diagnostics that carry
one to power one-click quick-fixes.

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
| `Up(ctx) (int, error)` | Applies every unapplied up migration. Returns the count applied. |
| `UpTo(ctx, targetVersion int64) (int, error)` | Applies pending up migrations up to and including `targetVersion`. Returns the count applied. |
| `Down(ctx, steps int) (int, error)` | Rolls back the last `steps` applied migrations in reverse order. Returns the count rolled back. |
| `DownTo(ctx, targetVersion int64) (int, error)` | Rolls back applied migrations down to (but not including) `targetVersion`. Returns the count rolled back. |
| `Status(ctx) ([]MigrationStatus, error)` | Lists every migration and whether the migrator has applied it. |
| `Validate(ctx) error` | Checks that every recorded migration's checksum still matches its on-disk file without executing anything. Returns an error on the first mismatch or missing file. |

### Migration service options

Pass these as variadic `opts...` to `NewMigrationService`:

| Option | Purpose |
|---|---|
| `WithNonBlockingLock() MigrationServiceOption` | Use a non-blocking advisory lock attempt that returns `ErrLockNotAcquired` instead of waiting. |
| `WithBeforeMigration(hook BeforeMigrationHook) MigrationServiceOption` | Register a hook that runs before each individual migration. |
| `WithAfterMigration(hook AfterMigrationHook) MigrationServiceOption` | Register a hook that runs after each individual migration completes. |
| `WithBeforeRun(hook BeforeRunHook) MigrationServiceOption` | Register a hook that runs before the migration run begins. |
| `WithAfterRun(hook AfterRunHook) MigrationServiceOption` | Register a hook that runs after the migration run completes. |

The package exports the hook function types as `BeforeMigrationHook`, `AfterMigrationHook`, `BeforeRunHook`, and `AfterRunHook`.

### `NewFSFileReader(filesystem fs.FS) FileReaderPort`

Wraps any `io/fs.FS` (typically a `go:embed` filesystem, also accepts `os.DirFS`) so migrations ship inside the binary. Returns the `FileReaderPort` interface that `NewMigrationService` and `NewSeedService` accept.

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

## Seed service

The seed service applies SQL seed files in order, recording a checksum per applied file. Construction mirrors the migration service.

### `NewSeedService(executor SeedExecutorPort, fileReader FileReaderPort, directory string) SeedService`

Creates a seed service that reads seed files from `directory` via `fileReader` and applies them through `executor`.

### `NewSeedExecutor(database *sql.DB, dialect DialectConfig) SeedExecutorPort`

Constructs a SQL-based seed executor for the given database connection and dialect.

## Database and service helpers

These accessors look up registered databases and services by the name passed to `piko.WithDatabase`.

| Function | Purpose |
|---|---|
| `GetDatabaseConnection(name string) (*sql.DB, error)` | Returns the underlying `*sql.DB` for a registered database. |
| `GetDatabaseReader(name string) (DBTX, error)` | Returns the read-side `DBTX`, honouring replica routing if configured. |
| `GetDatabaseWriter(name string) (DBTX, error)` | Returns the write-side `DBTX`, honouring replica routing if configured. |
| `GetMigrationService(name string) (MigrationService, error)` | Returns the migration service registered for the named database. |
| `GetSeedService(name string) (SeedService, error)` | Returns the seed service registered for the named database. |

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
| `wdk/db/db_engine_timescaledb` | `TimescaleDB()` | TimescaleDB (postgres child engine; recognises CREATE HYPERTABLE, continuous aggregates, compression DDL, and ~70 time-series functions). |
| `wdk/db/db_engine_duckdb` | `DuckDB()` | DuckDB. |
| `wdk/db/db_engine_clickhouse` | `ClickHouse()` | ClickHouse (table-based migration lock; no transactions). |

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
- Scenarios: [022 SQLite](../../examples/scenarios/022_database_sqlite/), [023 MySQL](../../examples/scenarios/023_database_mysql/), [024 PostgreSQL](../../examples/scenarios/024_database_postgres/), [025 DuckDB](../../examples/scenarios/025_database_duckdb/).

Integration tests: [`tests/integration/querier_sqlite`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_sqlite), [`querier_postgres`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_postgres), [`querier_mariadb`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_mariadb), [`querier_cockroachdb`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_cockroachdb), [`querier_duckdb`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_duckdb), [`querier_clickhouse`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_clickhouse), [`querier_timescaledb`](https://github.com/piko-sh/piko/tree/master/tests/integration/querier_timescaledb).
