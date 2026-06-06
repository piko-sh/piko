---
title: How to use ClickHouse with Piko
description: Codegen and migrations for ClickHouse via the piko querier, covering type wrappers, named parameters, MergeTree engine clauses, ARRAY JOIN, and FINAL.
nav:
  sidebar:
    section: "how-to"
    subsection: "database"
    order: 35
---

# How to use ClickHouse with Piko

The Piko querier supports ClickHouse as a first-class engine.
Piko parses ClickHouse SQL, builds a structured catalogue from your
migrations, and generates type-safe Go code from your queries. The
workflow matches the postgres, sqlite, and mysql engines,
and Piko handles ClickHouse-specific shapes automatically.

This guide covers the things that diverge from SQL-standard
engines. Those areas are type wrappers, `{name:Type}` named parameters, MergeTree
table-engine clauses, ARRAY JOIN, FINAL, and migration semantics.

## Quick start

Wire ClickHouse into your application config:

```go
import (
    "piko.sh/piko/wdk/db"
    "piko.sh/piko/wdk/db/db_engine_clickhouse"
)

func dbConfig() db.EngineConfig {
    return db_engine_clickhouse.ClickHouse()
}
```

`ClickHouse()` returns a fully configured `db.EngineConfig` with:

- The driver name `clickhouse` (registered by the
  `github.com/ClickHouse/clickhouse-go/v2` package).
- The ClickHouse engine adapter (parser, type system, function
  catalogue).
- `migration_sql.ClickHouseDialect()` for migration execution, which
  runs each statement separately (ClickHouse has no DDL transactions)
  and uses no-op locking (ClickHouse has no advisory locks).

## Named parameters: `{name:Type}`

ClickHouse's native protocol uses brace-delimited parameters with an
embedded type tag. Piko keeps the `{name:Type}` brace form (it does
not rewrite to positional placeholders). Piko does rewrite each
parameter name to a `p_`-prefixed wire name, so reserved keywords
like `limit` and `offset` cannot reach the server as parameter names.

```sql
-- piko.query(GetUser, one)
SELECT id, email FROM users WHERE id = {user_id:UInt64};
```

Piko sends `{user_id:UInt64}` to the server as `{p_user_id:UInt64}`
and the generated Go method binds the value via
`clickhouse.Named("p_user_id", ...)` behind the scenes:

```go
user, err := queries.GetUser(ctx, 42)
```

The same `{name:Type}` token can appear multiple times in one query.
All occurrences bind to the single Go parameter:

```sql
-- piko.query(FindLinked, many)
SELECT id FROM nodes WHERE id = {nid:UInt64} OR parent_id = {nid:UInt64};
```

## Type wrappers

ClickHouse encodes wrappers inside the type name. Piko strips them
during normalisation and surfaces the resulting Go type:

| Type declaration | Generated Go field |
|---|---|
| `UInt64` | `uint64` |
| `Int32` | `int32` |
| `Nullable(String)` | `*string` |
| `Array(String)` | `[]string` |
| `Array(Nullable(UInt32))` | `[]*uint32` |
| `LowCardinality(String)` | `string` (storage hint only) |
| `Tuple(String, UInt64)` | named struct with `_1` / `_2` fields |
| `Tuple(name String, age UInt64)` | named struct with declared field names |
| `Map(String, UInt32)` | `map[string]uint32` |
| `Nested(name String, age UInt64)` | `[]struct { Name string; Age uint64 }` (sugar for `Array(Tuple(...))`) |
| `DateTime64(6, 'UTC')` | `time.Time` |
| `Decimal(18, 4)` | high-precision decimal type |
| `Enum8('a' = 1, 'b' = 2)` | string-backed enum |
| `UUID` | `github.com/google/uuid.UUID` or `string` |

`Nullable(T)` always becomes `*T` in the row struct. Piko strips `LowCardinality(T)`
because it is a storage-side compression hint with no
client-side impact.

## CREATE TABLE engine clauses

ClickHouse's `CREATE TABLE` carries query-planning metadata after the
column list. Piko parses the whole declaration:

```sql
CREATE TABLE events (
    id UInt64,
    ts DateTime,
    payload String
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(ts)
ORDER BY (ts, id)
SETTINGS index_granularity = 8192
TTL ts + INTERVAL 90 DAY;
```

Piko captures the `ENGINE`, `PARTITION BY`, `ORDER BY`, `SAMPLE BY`, `SETTINGS`, and
`TTL` clauses verbatim into the catalogue mutation's
`EngineSpecific` map. Downstream consumers (PREWHERE eligibility,
FINAL semantics) can inspect them without re-parsing the SQL. When
your migration declares no explicit `PRIMARY KEY`, Piko infers it from the
`ORDER BY` column list to match ClickHouse's own behaviour.

Piko accepts any engine declaration. The standard MergeTree family
(MergeTree, ReplacingMergeTree, SummingMergeTree, and the other
collapsing and aggregating variants) plus the Log, Memory, and
Distributed engines carry semantic meaning to Piko. Any other engine
stays catalogue-only. See the ClickHouse documentation for the full
engine list.

## SELECT extensions

ClickHouse-specific SELECT clauses parse cleanly. Code generation
emits them in the SQL passed to the driver:

```sql
-- piko.query(LatestVersion, one)
SELECT id, val FROM versioned FINAL WHERE id = {id:UInt64};

-- piko.query(EventsByTag, many)
SELECT id, tag FROM events ARRAY JOIN tags AS tag WHERE tag = {filter_tag:String};

-- piko.query(SampledStats, many)
SELECT category, count(*) AS c FROM events SAMPLE 0.1 GROUP BY category;

-- piko.query(RegionalFiltered, many)
SELECT id FROM events PREWHERE region = {region:String} WHERE ts > {start:DateTime};

-- piko.query(LatestPerCategory, many)
SELECT id, cat, ts FROM events ORDER BY cat, ts DESC LIMIT 3 BY cat;
```

JOIN variants: `INNER`, `LEFT`, `RIGHT`, `FULL`, `CROSS`, plus
ClickHouse's `ASOF`, `SEMI`, and `ANTI` variants. `ARRAY JOIN` and
`LEFT ARRAY JOIN` both parse. The analyser treats array-joined
columns as scalar projections in the result.

`GROUP BY` accepts the standard `WITH ROLLUP`, `WITH CUBE`, and
`WITH TOTALS` extensions.

## Migration semantics

ClickHouse has no DDL transactions and no advisory locks. The Piko
migration dialect (`migration_sql.ClickHouseDialect()`) handles both
constraints:

- **Per-statement execution.** Piko splits each migration's body on
  top-level semicolons and executes one statement at a time.
- **Append-only history.** Each up-migration records a single
  completed row on success. It does not use per-statement dirty or
  `last_statement` resumable tracking, so a failed migration does
  not record partial progress to resume from.
- **No-op locking.** ClickHouse migration locking is a no-op. There
  is no `piko_migration_lock` table and no in-database serialisation
  between concurrent migrators. Coordinate concurrent migrators
  externally (CI lock or deployment gate) for production
  multi-replica setups.
- **History table engine.** The `piko_migrations` and `piko_seeds`
  tables use `ENGINE = ReplacingMergeTree(applied_at) ORDER BY
  version`, which collapses duplicate version rows on merge. Piko
  reads the history with `FINAL` to see the current applied set.

Piko maps the `Bool` type to ClickHouse's native `Bool` (UInt8
underneath). Earlier ClickHouse versions used `UInt8` directly for
boolean flags. Piko accepts both.

## Asynchronous mutations

`ALTER TABLE ... UPDATE` and `ALTER TABLE ... DELETE` run
fire-and-forget on the ClickHouse server. The server accepts the statement,
queues a background mutation in `system.mutations`, and sends the
client no rows-affected count. Use the `asyncexec` command
to surface that lifecycle in the generated method:

```sql
-- piko.query(PurgeOld, asyncexec)
ALTER TABLE events DELETE WHERE ts < {cutoff:DateTime};
```

The generated method has signature `PurgeOld(ctx, cutoff) error` and
carries a doc comment documenting that a nil return means the server
accepted the queued mutation, not that the mutation has finished
applying. Inspect `system.mutations` for completion status.

Using `exec` on the same statement still works because the underlying
driver call is identical. The analyser emits a Hint diagnostic
(`Q040`) recommending `asyncexec`. That diagnostic keeps a
synchronous-looking call signature from quietly hiding the
fire-and-forget semantics.

## User-defined functions

`CREATE FUNCTION name AS (params) -> expr` registers a lambda in the
catalogue. Piko parses the lambda body structurally and infers the return
type from the body expression:

```sql
CREATE FUNCTION times_two AS x -> x * 2;
```

The body's `x * 2` resolves through the type system. The literal
side anchors the supertype so calls to `times_two(...)` carry a
typed return value instead of degrading to `any`. Polymorphic
bodies whose parameters the type system cannot resolve against the call site,
for example `(x, y) -> x + y`, degrade to `Unknown`. The
parameter types are not knowable without a call-site narrowing.
Override per call site with a `piko.column` directive when the
analyser cannot pick a type. Define the polymorphic function in a
migration:

```sql
CREATE FUNCTION add_xy AS (x, y) -> x + y;
```

Then anchor its return type at the call site with `piko.column`:

```sql
-- piko.query(Compute, one)
-- piko.column(result, go_type: int32)
SELECT add_xy({a:Int32}, {b:Int32}) AS result;
```

Piko still accepts the legacy form `CREATE FUNCTION name` without `AS lambda`.
The catalogue records the name and downstream calls
receive `Unknown` for the return type.

## Complex types

ClickHouse's variant, dynamic, and aggregate-state types map to
specific Go types by default:

| ClickHouse type | Default Go type |
|---|---|
| `Variant(A, B, ...)` | `encoding/json.RawMessage` |
| `Dynamic` | `encoding/json.RawMessage` |
| `AggregateFunction(...)` | `[]byte` |
| `SimpleAggregateFunction(...)` | `[]byte` |

The variant and dynamic mappings preserve the active member's wire
representation as JSON so callers can decode the tagged payload at
runtime. The aggregate-state mapping preserves the opaque binary
blob so callers can re-feed it into `*Merge*` aggregations. Both mappings
ignore nullability since the wire shape is identical.

Override the default per call site with `piko.column(go_type: ...)`
when a concrete decoded type is more ergonomic:

```sql
-- piko.query(LoadEvent, one)
-- piko.column(payload, go_type: "myapp.EventPayload")
SELECT id, payload FROM events WHERE id = {id:UInt64};
```

## Not supported yet

These ClickHouse features are not in scope for v1:

- Cluster-wide DDL (`ON CLUSTER cluster_name`) parses through the
  clause as opaque metadata but offers no cluster-aware codegen.

## See also

- [Querier reference](../../reference/querier.md), the full
  directive surface and command kinds.
- [Engine swapping guide](swapping-engines.md) for moving between
  engines.
- ClickHouse documentation: https://clickhouse.com/docs
- `clickhouse-go/v2` driver: https://github.com/ClickHouse/clickhouse-go
