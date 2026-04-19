---
title: "024: PostgreSQL"
description: A Piko project backed by PostgreSQL with the generated querier.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 440
---

# 024: PostgreSQL

A Piko project using PostgreSQL as its database, with type-safe queries generated from `.sql` files and connection-pool tuning through pgxpool.

## What this demonstrates

- The `WithDatabase` bootstrap option with the PostgreSQL driver.
- Generator annotations specific to PostgreSQL dialect (array types, JSONB).
- A startup lifecycle component that runs migrations.
- Row-level transactions inside an action.

## Project structure

```text
src/
  main.go               Bootstrap with WithDatabase("postgres", ...).
  querier/
    *.sql
  migrations/
    *.sql
  pages/
    articles/
      index.pk
      {slug}.pk
  actions/
    articles/
      publish.go
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/024_database_postgres/src/
docker compose up -d postgres     # Starts PostgreSQL locally.
go mod tidy
piko generate
air
```

## See also

- [Bootstrap options reference: Database](../reference/bootstrap-options.md#database).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/024_database_postgres).
