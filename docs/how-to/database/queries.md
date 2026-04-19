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

Piko generates type-safe Go methods from annotated SQL. You write the SQL, and the generator produces the Go. This guide covers the annotation syntax and the generation workflow. See the [querier reference](../../reference/querier.md) for the API surface and [Scenario 022](../../showcase/022-database-sqlite.md) for a runnable example.

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

Each query needs two headers, `name` and `command`.

```sql
-- piko.name: ListTasks
-- piko.command: many
SELECT id, title, completed, created_at
FROM tasks
ORDER BY created_at DESC;

-- piko.name: GetTask
-- piko.command: one
SELECT id, title, completed, created_at
FROM tasks
WHERE id = ?;

-- piko.name: CreateTask
-- piko.command: one
INSERT INTO tasks (title, created_at) VALUES (?, ?)
RETURNING id, title, completed, created_at;

-- piko.name: ToggleComplete
-- piko.command: exec
UPDATE tasks
SET completed = CASE WHEN completed = 0 THEN 1 ELSE 0 END
WHERE id = ?;

-- piko.name: DeleteTask
-- piko.command: exec
DELETE FROM tasks WHERE id = ?;
```

| Header | Purpose |
|---|---|
| `-- piko.name: Name` | Method name on the `Queries` struct. PascalCase. Must be unique within the whole `queries/` folder. |
| `-- piko.command: kind` | `one` (single row), `many` (slice of rows), `exec` (no rows returned). |

## Run the generator

```bash
piko generate
```

The generator walks `db/queries/*.sql`, reads your migration schema for type information, and emits Go under `db/generated/`. Re-run after every SQL change.

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
    database := piko.GetDatabase("primary")
    queries := generated.New(database)

    tasks, err := queries.ListTasks(a.Ctx())
    if err != nil {
        return ListResponse{}, piko.Errorf("could not load tasks", "list tasks: %w", err)
    }

    return ListResponse{Tasks: tasks}, nil
}
```

`piko.GetDatabase(name)` returns the `*sql.DB` registered under that name with `WithDatabase`.

## Parameters

Positional placeholders (`?` for SQLite and MySQL, `$1/$2/...` for Postgres) become Go parameters:

- **One placeholder** becomes a single typed argument: `DeleteTask(ctx, p1 int32) error`.
- **Multiple placeholders** become a typed params struct: `CreateTask(ctx, params CreateTaskParams) (CreateTaskRow, error)`.

The Go types come from the engine's type inference over the schema. A column declared `TEXT NOT NULL` infers `string`, while `INTEGER NOT NULL` infers `int32` on SQLite (or the appropriate size for the target dialect).

### Named parameters

Engines that support named parameters (`:name`, `@name`) let you override the generated field names:

```sql
-- piko.name: UpdateTask
-- piko.command: exec
UPDATE tasks SET title = :title, completed = :completed WHERE id = :id;
```

The generator emits `UpdateTaskParams` with fields `Title`, `Completed`, `Id`.

## Write queries

INSERT that returns the row:

```sql
-- piko.name: CreateAuthor
-- piko.command: one
INSERT INTO authors (name, email) VALUES (?, ?)
RETURNING id, name, email, created_at;
```

Use `one` when the INSERT has a `RETURNING` clause. Use `exec` when it does not.

UPDATE that returns nothing:

```sql
-- piko.name: SetAuthorEmail
-- piko.command: exec
UPDATE authors SET email = ? WHERE id = ?;
```

DELETE:

```sql
-- piko.name: DeleteAuthor
-- piko.command: exec
DELETE FROM authors WHERE id = ?;
```

## Complex read queries

The generator handles joins, CTEs, window functions, subqueries, and analytics-style queries. The `piko.command: many` header tells it to return a slice, and the selected columns become the fields on the generated row type.

```sql
-- piko.name: GetTopProductsByRevenue
-- piko.command: many
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

The generated methods route based on command kind. `many` and `one` (without `RETURNING`) go to the reader. `exec` and `one` with `RETURNING` go to the writer.

## Do not let queries leak into template logic

> **Note:** Querier methods belong in `Call` and the packages it imports, not in template expressions or partial `Render` functions. The render path expects data already shaped for display, and pushing I/O into it breaks the caching, testing, and partial-refresh assumptions the framework makes.

Call the generated methods from action `Call` methods or from packages the action imports. Do not call them from template expressions or partial `Render` functions. Templates expect data already shaped for display, not database I/O.

## See also

- [Querier reference](../../reference/querier.md).
- [How to migrations](migrations.md).
- [How to swapping database engines](swapping-engines.md).
- [Server actions reference](../../reference/server-actions.md) for calling queries from actions.
- Scenarios: [022 SQLite](../../showcase/022-database-sqlite.md), [024 PostgreSQL](../../showcase/024-database-postgres.md) for analytics-style queries.
