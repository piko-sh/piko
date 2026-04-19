---
title: "025: DuckDB (analytics)"
description: A Piko project backed by DuckDB for embedded analytics queries.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 450
---

# 025: DuckDB (analytics)

A Piko project that uses DuckDB for in-process analytical queries. DuckDB is well suited to dashboards that run aggregate queries over millions of rows without a separate database process.

## What this demonstrates

- The `WithDatabase` bootstrap option with the DuckDB driver.
- Columnar-store query patterns (aggregates, window functions).
- Loading Parquet files into DuckDB at startup via a lifecycle component.
- A dashboard page that runs multiple aggregate queries and renders charts.

## Project structure

```text
src/
  main.go               Bootstrap with WithDatabase("duckdb", ...).
  querier/
    *.sql               Aggregate analytics queries.
  data/
    events.parquet      Source data loaded into DuckDB at startup.
  pages/
    dashboard/
      index.pk          Dashboard page.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/025_database_duckdb/src/
go mod tidy
piko generate
air
```

## See also

- [Bootstrap options reference: Database](../reference/bootstrap-options.md#database).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/025_database_duckdb).
