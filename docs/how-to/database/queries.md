---
title: How to write type-safe database queries
description: Author SQL query files, annotate them for the generator, and call the generated Go methods from actions.
nav:
  sidebar:
    section: "how-to"
    subsection: "database"
    order: 20
---

# How to write type-safe database queries

Piko generates type-safe Go methods from annotated SQL. You write the SQL, and the generator produces the Go. This guide covers the annotation syntax and the generation workflow. See the [querier reference](../../reference/querier.md) for the API surface and [Scenario 022](../../../examples/scenarios/022_database_sqlite/) for a runnable example.

## Place queries under `db/queries/`

Group related queries per file:

```
db/
  queries/
    tasks.sql
    users.sql
    comments.sql
```

Each file produces one `<name>.sql.go` under `db/generated/`. The file name is cosmetic. The query name from the annotation is what matters.

## Annotate every query

Each query needs a single `piko.query(...)` header. The first two
arguments are positional: `name` (PascalCase Go method name) and
`command` (execution kind). Additional flags like `dynamic`,
`readonly`, `nullable`, `group_by` are optional keyword arguments.

```sql
-- piko.query(ListTasks, many)
SELECT id, title, completed, created_at
FROM tasks
ORDER BY created_at DESC;

-- piko.query(GetTask, one)
SELECT id, title, completed, created_at
FROM tasks
WHERE id = ?;

-- piko.query(CreateTask, one)
INSERT INTO tasks (title, created_at) VALUES (?, ?)
RETURNING id, title, completed, created_at;

-- piko.query(ToggleComplete, exec)
UPDATE tasks
SET completed = CASE WHEN completed = 0 THEN 1 ELSE 0 END
WHERE id = ?;

-- piko.query(DeleteTask, exec)
DELETE FROM tasks WHERE id = ?;
```

You may also name every positional. The parser accepts the following
two equivalent forms:

```sql
-- piko.query(GetUser, one)                      -- positional
-- piko.query(name: GetUser, command: one)       -- keyword-as-positional
```

Use whichever form reads clearest at the call site. For one-line
queries the positional form is shorter. For long multi-keyword
calls, names anchor the reader.

| Position / Key | Purpose |
|---|---|
| 1 (positional) `name` | Method name on the `Queries` struct. PascalCase. Must be unique within the whole `queries/` folder. |
| 2 (positional) `command` | Execution pattern. One of nine kinds. |

The commands you reach for most are `one`, `many`, `exec`, `execrows`, and `execresult`:

| Kind | Returns | Use |
|---|---|---|
| `one` | A single row struct, or `sql.ErrNoRows`. | `SELECT ... LIMIT 1`, `INSERT ... RETURNING`, or any single-row read. |
| `many` | `[]Row`. | `SELECT` returning zero or more rows. |
| `exec` | `error` only. | `INSERT`, `UPDATE`, `DELETE` where you do not need to inspect the result. |
| `execrows` | `(int64, error)` - rows affected. | Mutations where the caller needs the affected-row count. |
| `execresult` | `(sql.Result, error)`. | Mutations where the caller needs `LastInsertId()` or the full `sql.Result`. |

The remaining four (`batch`, `stream`, `copyfrom`, `asyncexec`) cover batched execution, streamed iteration, and bulk COPY inserts. `asyncexec` covers fire-and-forget mutations whose completion the server reports asynchronously, for example ClickHouse `ALTER UPDATE` and `DELETE`. See the [querier reference](../../reference/querier.md) for the full enumeration of all nine kinds and their exact return shapes.

## Run the generator

Inside your generated (scaffolded) project, run the generator that the scaffold created at `cmd/generator/main.go`:

```bash
go run ./cmd/generator/main.go all
```

This command lives in your project, not in the piko repository itself. The scaffolded generator walks `db/queries/*.sql`, reads your migration schema for type information, and emits Go under `db/generated/`. Re-run after every SQL change. Air-watch (the dev server) runs this on every save automatically.

Do not edit files under `generated/` by hand. The next run overwrites them.

## Call the generated methods

Construct the `Queries` struct from the registered database and call methods directly:

```go
package tasks

import (
    "piko.sh/piko"

    generated "myapp/db/generated"
)

type ListAction struct {
    piko.ActionMetadata
}

type ListResponse struct {
    Tasks []generated.ListTasksRow `json:"tasks"`
}

func (a *ListAction) Call(_ piko.NoInput) (ListResponse, error) {
    database, err := db.GetDatabaseConnection("primary")
    if err != nil {
        return ListResponse{}, piko.Errorf("could not open database", "get database: %w", err)
    }
    queries := generated.New(database)

    tasks, err := queries.ListTasks(a.Ctx())
    if err != nil {
        return ListResponse{}, piko.Errorf("could not load tasks", "list tasks: %w", err)
    }

    return ListResponse{Tasks: tasks}, nil
}
```

`db.GetDatabaseConnection(name)` (from `piko.sh/piko/wdk/db`) returns the `*sql.DB` registered under that name with `WithDatabase`. Use `db.GetDatabaseReader(name)` and `db.GetDatabaseWriter(name)` when you configure read/write splits.

## Parameters

Positional placeholders (`?` for SQLite and MySQL, `$1/$2/...` for Postgres) become Go parameters:

- **One placeholder** becomes a single typed argument named after the column it binds to: `DeleteTask(ctx, id int32) error`. Without a `piko.param` directive, Piko infers the name from the bound column.
- **Multiple placeholders** become a typed params struct: `CreateTask(ctx, params CreateTaskParams) (CreateTaskRow, error)`.

The Go types come from the engine's type inference over the schema. A column declared `TEXT NOT NULL` infers `string`, while `INTEGER NOT NULL` infers `int32` on SQLite (or the appropriate size for the target dialect).

### Naming parameters with directives

Piko supports `:name` placeholders, but you still bind each to a Go
field name with a `piko.param` directive line above it. The generator
does not auto-derive the struct field from the bare `:name`. Name
parameters by adding a directive line above each placeholder you want
to bind to a Go field name:

```sql
-- piko.query(FindUser, one)
-- $1 as piko.param(user_id)
-- $2 as piko.param(email, optional: true)
SELECT id, name FROM users
WHERE id = $1 AND ($2 IS NULL OR email = $2);
```

`piko.param(<name>)` produces a required field. Adding `optional: true`
produces a nullable field that callers may omit. When a caller omits
it, the generator emits a runtime predicate that elides the matching
WHERE clause. The generator emits `FindUserParams{UserID: ..., Email: ...}`
from the directives. The same syntax works with `?1`/`?2` for SQLite
and MySQL placeholders.

`piko.param` is the one parameter directive. Its keyword arguments cover the common cases:

- `optional: true` makes a nullable field. The generator elides the WHERE predicate at call time when the caller passes nil.
- `kind: slice` expands a Go slice into IN-list placeholders.
- `default:` and `max:` set a default and an inclusive cap for a numeric (pagination) parameter, for example in a LIMIT or OFFSET clause.
- `type:` and `nullable:` override the inferred SQL type and nullability.

See the [querier reference](../../reference/querier.md) for the full keyword surface. To validate a dynamic `ORDER BY` against a closed column allow-list, use `piko.sortable` instead (shown below).

Example combining `piko.param` and `piko.sortable`:

```sql
-- piko.query(BrowseProducts, many)
-- $1 as piko.param(category, optional: true)
-- piko.sortable(order_by, columns: [name, price, created_at])
SELECT id, name, price, category FROM products
WHERE ($1 IS NULL OR category = $1)
```

`piko.sortable` is a standalone header. It binds no placeholder, so it
takes no `$N as` prefix, and you do not write an `ORDER BY` clause in
the SQL body. The generated builder appends the validated `ORDER BY`
against the closed column allow-list. The caller selects the sort
column and direction at runtime through the generated sortable input
field. Leave off any trailing semicolon on a sortable query, because
the generator concatenates the appended `ORDER BY` onto the base SQL.

See the [querier reference](../../reference/querier.md) for the full
directive surface, including `piko.embed` for nested struct
projections and `piko.migration(readonly:)` for migration-side function overrides.

### Custom Go types per column

Sometimes the analyser cannot infer a column's Go type precisely. You
may also want to map a column to a custom Go type in a different
package, such as `uuid.UUID` or your own domain newtype. For both
cases, use `piko.column`:

```sql
-- piko.query(GetUserUUID, one)
-- piko.column(id, go_type: "github.com/google/uuid.UUID")
SELECT id, email FROM users WHERE id = $1;
```

The generated row struct field uses `uuid.UUID` and the file imports
`github.com/google/uuid`. Nullable columns wrap as `*uuid.UUID`. Your
custom type must implement `sql.Scanner` for `database/sql` engines
(or the equivalent for `pgx`).

For schema-wide overrides, declare `piko.column` in the migration
file above the `CREATE TABLE`. Every query that selects the column
unchanged inherits the override. Casts and function calls drop it.

```sql
-- piko.column(users.id, go_type: "github.com/google/uuid.UUID")
CREATE TABLE users (id UUID PRIMARY KEY, ...);
```

## Write queries

INSERT that returns the row:

```sql
-- piko.query(CreateAuthor, one)
INSERT INTO authors (name, email) VALUES (?, ?)
RETURNING id, name, email, created_at;
```

Use `one` when the INSERT has a `RETURNING` clause. Use `exec` when it does not.

UPDATE that returns nothing:

```sql
-- piko.query(SetAuthorEmail, exec)
UPDATE authors SET email = ? WHERE id = ?;
```

DELETE:

```sql
-- piko.query(DeleteAuthor, exec)
DELETE FROM authors WHERE id = ?;
```

## Complex read queries

The generator handles joins, CTEs, window functions, subqueries, and analytics-style queries. The `command` argument (`many`) in the `piko.query(...)` header tells it to return a slice, and the selected columns become the fields on the generated row type.

```sql
-- piko.query(GetTopProductsByRevenue, many)
WITH ranked AS (
    SELECT
        p.id,
        p.name,
        p.category,
        SUM(o.total) AS revenue,
        ROW_NUMBER() OVER (PARTITION BY p.category ORDER BY SUM(o.total) DESC) AS category_rank
    FROM products p
    JOIN orders o ON o.product_id = p.id
    GROUP BY p.id, p.name, p.category
)
SELECT id, name, category, revenue, category_rank
FROM ranked
WHERE category_rank <= $1
ORDER BY revenue DESC;
```

The generator infers every column type from the query plan.

## Transactions

Wrap a set of queries in a transaction with `RunInTx`:

```go
queries := generated.New(database)

err := queries.RunInTx(ctx, database, func(q *generated.Queries) error {
    if err := q.DebitAccount(ctx, params1); err != nil {
        return err
    }
    if err := q.CreditAccount(ctx, params2); err != nil {
        return err
    }
    return nil
})
```

`RunInTx` opens a transaction, runs the closure, and commits on nil return or rolls back on error.

For manual control:

```go
tx, err := database.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback()

txQueries := queries.WithTx(tx)
if err := txQueries.Something(ctx, ...); err != nil {
    return err
}

return tx.Commit()
```

## Read replicas

Route reads to a replica and writes to the primary:

```go
queries := generated.NewWithReplica(writerDB, readerDB)
```

The generated methods route on the analysed read-only status of the query. Read-only queries go to the reader, and mutating queries go to the writer. Detection is automatic. You can override it with the `readonly:` keyword on `piko.query`. A `SELECT` that calls a data-modifying function routes to the writer despite being `command:one` or `command:many`.

## Do not let queries leak into template logic

> **Note:** Querier methods belong in `Call` and the packages it imports, not in template expressions or partial `Render` functions. The render path expects data already shaped for display, and pushing I/O into it breaks the caching, testing, and partial-refresh assumptions Piko makes.

Call the generated methods from action `Call` methods or from packages the action imports. Do not call them from template expressions or partial `Render` functions. Templates expect data already shaped for display, not database I/O.

## See also

- [Querier reference](../../reference/querier.md).
- [How to migrations](migrations.md).
- [How to swapping database engines](swapping-engines.md).
- [Server actions reference](../../reference/server-actions.md) for calling queries from actions.
- Scenarios: [022 SQLite](../../../examples/scenarios/022_database_sqlite/), [024 PostgreSQL](../../../examples/scenarios/024_database_postgres/) for analytics-style queries.
