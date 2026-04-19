---
title: "022: SQLite database"
description: A Piko project backed by SQLite with the generated querier.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 420
---

# 022: SQLite database

A Piko project using SQLite as its database, with type-safe queries generated from `.sql` files by `piko generate`.

## What this demonstrates

- The `WithDatabase` bootstrap option with the SQLite driver.
- Type-safe query functions generated from `querier/*.sql` at build time.
- A lifecycle component that runs migrations on startup.
- Action code calling the generated querier.

## Project structure

```text
src/
  main.go               Bootstrap with WithDatabase("sqlite", ...).
  querier/
    *.sql               SQL files with annotations for the generator.
  migrations/
    001_initial.sql     Schema.
  pages/
    todos/
      index.pk          Todo list page.
  actions/
    todos/
      create.go         Uses the generated querier.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/022_database_sqlite/src/
go mod tidy
piko generate
air
```

## See also

- [Bootstrap options reference: Database](../reference/bootstrap-options.md#database).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/022_database_sqlite).
