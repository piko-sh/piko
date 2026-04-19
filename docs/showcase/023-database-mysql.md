---
title: "023: MySQL / MariaDB"
description: A Piko project backed by MySQL (or MariaDB) with the generated querier.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 430
---

# 023: MySQL / MariaDB

A Piko project using MySQL (or MariaDB) as its database, with type-safe queries generated from `.sql` files.

## What this demonstrates

- The `WithDatabase` bootstrap option with the MySQL driver.
- Generator annotations specific to MySQL dialect.
- Connection-pool tuning through the registration struct.
- A `LifecycleComponent` that runs migrations.

## Project structure

```text
src/
  main.go               Bootstrap with WithDatabase("mysql", ...).
  querier/
    *.sql
  migrations/
    *.sql
  pages/
    products/
      index.pk
  actions/
    products/
      upsert.go
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/023_database_mysql/src/
docker compose up -d mariadb     # Starts MariaDB locally.
go mod tidy
piko generate
air
```

## See also

- [Bootstrap options reference: Database](../reference/bootstrap-options.md#database).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/023_database_mysql).
