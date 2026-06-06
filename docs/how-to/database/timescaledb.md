---
title: How to use TimescaleDB with Piko
description: Hypertables, continuous aggregates, compression policies, and the TimescaleDB function library through Piko's postgres-derived engine.
nav:
  sidebar:
    section: "how-to"
    subsection: "database"
    order: 40
---

# Using TimescaleDB with Piko

TimescaleDB is a postgres extension that adds time-series functionality (hypertables, continuous aggregates, compression policies, retention policies, and a rich function library including `time_bucket`, `first`, `last`, `locf`, `interpolate`, and the hyperfunctions toolkit).

Piko's TimescaleDB engine is a thin child of the postgres engine. It re-uses everything postgres provides (parser, type system, codegen, migration dialect) and adds:

- Over 120 distinct TimescaleDB function names (around 170 overload signatures): time_bucket, first/last, stats_agg, hyperloglog, policy and job management, hypertable helpers, metadata queries
- Around 21 opaque (unknown-category) type aliases (statssummary1d, counter_summary, candlestick, tdigest, hyperloglog, uddsketch, and the rest)
- A statement-extension that recognises four TimescaleDB-only DDL forms

This page shows the setup steps, then documents the four extra DDL shapes Piko understands plus the EngineSpecific metadata Piko attaches to each.

## Setup

Register the database the same way you would for any postgres-family engine, using `piko.WithDatabase` and the `db.DatabaseRegistration` shape:

```go
import (
    "database/sql"

    _ "github.com/jackc/pgx/v5/stdlib"

    "piko.sh/piko"
    "piko.sh/piko/wdk/db"
    "piko.sh/piko/wdk/db/db_engine_timescaledb"
)

database, err := sql.Open("pgx", "postgres://user:pass@localhost:5432/app")
if err != nil {
    // handle err
}

ssr := piko.New(
    piko.WithDatabase("primary", &db.DatabaseRegistration{
        DB:           database,
        EngineConfig: db_engine_timescaledb.TimescaleDB(),
    }),
)
```

The driver name remains `postgres` (TimescaleDB is wire-compatible) and Piko reuses the migration dialect unchanged.

## DDL forms Piko recognises

### CREATE HYPERTABLE (keyword form)

```sql
CREATE HYPERTABLE IF NOT EXISTS readings (
    ts          TIMESTAMPTZ NOT NULL,
    device_id   BIGINT,
    temperature DOUBLE PRECISION
) PARTITION BY RANGE (ts);
```

Produces a `MutationCreateTable` with `EngineSpecific["TIMESCALE_HYPERTABLE"] = "true"`. Schema-qualified names (`analytics.readings`) work the same way. If `IF NOT EXISTS` is present, Piko also sets `TIMESCALE_IF_NOT_EXISTS = "true"`. Piko captures trailing `PARTITION BY` / `WITH` clauses opaquely into `TIMESCALE_TRAILING`.

### create_hypertable() function call

```sql
SELECT create_hypertable('readings', 'ts', chunk_time_interval => INTERVAL '1 day');
```

Produces a `MutationCreateTable` annotation (not a schema change). `TIMESCALE_HYPERTABLE = "true"`, `TIMESCALE_TIME_COLUMN = "ts"`, and `TIMESCALE_ANNOTATE_ONLY = "true"` mark it as a hypertable promotion of an already-created table. Piko preserves extra named arguments (`chunk_time_interval => ...`, `space_partitions => ...`) in `TIMESCALE_CALL_EXTRAS`.

### CREATE MATERIALIZED VIEW with timescaledb.continuous

```sql
CREATE MATERIALIZED VIEW hourly_temps
WITH (timescaledb.continuous = true, timescaledb.materialized_only = true)
AS SELECT time_bucket('1 hour', ts) AS bucket, avg(temperature)
FROM readings
GROUP BY bucket
WITH NO DATA;
```

Produces a `MutationCreateView` with:

- `TIMESCALE_CONTINUOUS_AGGREGATE = "true"`
- `TIMESCALE_CONTINUOUS_FLAG = "true"` (from the reloption itself)
- `TIMESCALE_MATERIALIZED_ONLY = "true"`
- `TIMESCALE_WITH_DATA = "false"`
- `TIMESCALE_VIEW_BODY = <raw SELECT text>`

Regular `CREATE MATERIALIZED VIEW name AS SELECT ...` (without the `timescaledb.continuous` reloption) takes the built-in postgres path with no TimescaleDB metadata.

### ALTER TABLE / ALTER MATERIALIZED VIEW SET (timescaledb.*)

```sql
ALTER TABLE readings SET (
    timescaledb.compress = true,
    timescaledb.compress_segmentby = 'device_id',
    timescaledb.compress_orderby = 'ts DESC'
);
```

Produces a `MutationAlterTableAlterColumn` (postgres's existing mutation kind for ALTER TABLE SET) with:

- `TIMESCALE_COMPRESSION_ENABLED = "true"`
- `TIMESCALE_COMPRESSION_SEGMENTBY = "device_id"`
- `TIMESCALE_COMPRESSION_ORDERBY = "ts DESC"`

The `ALTER MATERIALIZED VIEW` form additionally sets `TIMESCALE_ALTER_MATERIALIZED = "true"`. Piko preserves unrecognised `timescaledb.*` keys in `TIMESCALE_RELOPTION_<KEY>` so it drops nothing silently.

`ALTER TABLE x SET SCHEMA y` takes the built-in postgres path (it is not a compression operation).

## Function library

The function catalogue automatically picks up `time_bucket`, `first`, `last`, `locf`, `interpolate`, `histogram`, `stats_agg`, `counter_agg`, `candlestick_agg`, `hyperloglog`, `tdigest`, `approx_percentile`, `add_compression_policy`, `add_retention_policy`, `add_continuous_aggregate_policy`, `create_hypertable`, `add_dimension`, `drop_chunks`, `show_chunks`, `compress_chunk`, `decompress_chunk`, `refresh_continuous_aggregate`, `approximate_row_count`, `hypertable_size`, plus their close relatives.

Piko registers the aggregate-state aliases (`statssummary1d`, `counter_summary`, `candlestick`, `tdigest`, and the rest) as opaque (unknown-category) types that codegen emits as `any`. Override the Go destination per column by naming the output column in a query header:

```sql
-- piko.query(GetCounterState, one)
-- piko.column(state, go_type: "github.com/your/pkg.YourType")
SELECT counter_agg(ts, value) AS state FROM readings WHERE device_id = $1;
```

## Limitations

- The hypertable column parser captures column names and inline constraints (NOT NULL, PRIMARY KEY, DEFAULT, REFERENCES) but treats column type text opaquely. Downstream type resolution depends on the postgres type normaliser running over the captured engine name.
- Piko captures continuous-aggregate SELECT bodies as opaque text in `TIMESCALE_VIEW_BODY` instead of fully analysing them. Codegen that needs typed columns for a continuous aggregate relies on re-tokenising the body.
- The TimescaleDB function overload registrations are conservative. Unusual `time_bucket` argument shapes (timezone-aware overloads) may fall through to the catalogue resolver's polymorphism handling.

## Extending the engine

The TimescaleDB engine builds on the postgres engine's extension hooks (statement extensions, post-parse hooks, and a curated parser context). Those hooks are contributor-facing and not needed to use TimescaleDB through Piko. To write your own child engine, read `wdk/db/db_engine_postgres/extension_context.go` for the extension surface and `wdk/db/db_engine_timescaledb/` for the reference consumer.
