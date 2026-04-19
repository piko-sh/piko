---
title: Cache API
description: Cache service, typed cache instances, search, and provider registration.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 130
---

# Cache API

Piko's cache provides in-memory, distributed, and multilevel caching with optional compression, encryption, and full-text search. Each cache instance carries generic key and value parameters and sits inside a namespace. For the design rationale see [about caching](../explanation/about-caching.md). For task recipes see [how to cache](../how-to/cache.md). Source file: [`wdk/cache/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/cache/facade.go).

## Service

| Function | Returns |
|---|---|
| `cache.NewService(defaultProviderName string) Service` | Constructs a new service. |
| `cache.GetDefaultService() (Service, error)` | Returns the service the bootstrap built. |
| `cache.RegisterProviderFactory(name, factory)` | Registers a domain-specific factory. Use in `init()`. |

## Cache construction

```go
func CreateNamespace[K comparable, V any](ctx context.Context, service Service, providerName, namespace string, options Options[K, V]) (Cache[K, V], error)
func NewCache[K comparable, V any](service Service, options Options[K, V]) (Cache[K, V], error)
func NewCacheFromDefault[K comparable, V any](options Options[K, V]) (Cache[K, V], error)
func NewCacheBuilder[K comparable, V any](service Service) (*Builder[K, V], error)
func NewCacheBuilderFromDefault[K comparable, V any]() (*Builder[K, V], error)
```

`CreateNamespace` is the recommended entry point. It maps one namespace per value type, which isolates entries and keeps key spaces from colliding. `NewCacheBuilderFromDefault` is the fluent shortcut when Piko is already bootstrapped.

> **Note:** Treat one namespace as one value type. Sharing a namespace across `User` and `Product` causes their key spaces to overlap, so a `User{ID:42}` collides with a `Product{ID:42}`.

### Builder fluent API

The builder returned from `NewCacheBuilder` chains configuration before `Build(ctx)`. Method names have **no `With` prefix**:

```go
myCache, err := builder.
    Provider("otter").
    Namespace("products").
    MaximumSize(10000).
    Compression().
    Build(ctx)
```

Builder method groups:

| Group | Methods |
|---|---|
| Provider / source | `Provider(name)`, `Namespace(ns)`, `FactoryBlueprint(name)`, `MultiLevel(l1, l2)`, `L1Options(any)`, `L2Options(any)`, `L2CircuitBreaker(maxFailures, openTimeout)`, `Options(any)` |
| Capacity | `MaximumSize(int)`, `MaximumWeight(uint64)`, `InitialCapacity(int)`, `Weigher(func(K, V) uint32)` |
| Transformers | `Transformer(name, configs...)`, `Compression()`, `Encryption()`, `EncryptionWithService(any)` |
| Encoders | `Encoder(AnyEncoder)`, `TypedEncoder(EncoderPort[V])`, `DefaultEncoder(AnyEncoder)` |
| Time | `Expiration(time.Duration)`, `WriteExpiration(time.Duration)`, `AccessExpiration(time.Duration)`, `ExpiryCalculator(...)`, `RefreshCalculator(...)` |
| Hooks | `OnDeletion(func(DeletionEvent[K, V]))`, `OnAtomicDeletion(func(DeletionEvent[K, V]))` |
| Stats / observability | `StatsRecorder(StatsRecorder)`, `Logger(Logger)` |
| Other | `Executor(func(operation func()))`, `Clock(Clock)`, `Searchable(*SearchSchema)` |

`Build(ctx)` is the terminus.

`Expiration` is a convenience for `WriteExpiration` (a fixed TTL applied on creation and updates only). `AccessExpiration` sets a sliding TTL that resets on every access (read, write, compute).

## The `Cache` interface

`Cache[K, V]` mirrors the [maypok86/otter v2](https://github.com/maypok86/otter) API. Methods grouped by purpose:

### Read

| Method | Purpose |
|---|---|
| `GetIfPresent(ctx, key) (V, bool, error)` | Returns the value if present and unexpired; `(zero, false, nil)` on miss. No loader call. |
| `Get(ctx, key, loader) (V, error)` | Returns the value, calling `loader` on miss. Provides built-in stampede protection. |
| `GetEntry(ctx, key) (Entry[K, V], bool, error)` | Returns an entry snapshot with metadata; resets the access timer. |
| `ProbeEntry(ctx, key) (Entry[K, V], bool, error)` | Same as `GetEntry` but does NOT reset the access timer. Useful for monitoring. |
| `All() iter.Seq2[K, V]` | Iterator over every key-value pair. |
| `Keys() iter.Seq[K]` | Iterator over keys. |
| `Values() iter.Seq[V]` | Iterator over values. |

### Write

| Method | Purpose |
|---|---|
| `Set(ctx, key, value, tags...) error` | Stores with optional tag groups. |
| `SetWithTTL(ctx, key, value, ttl, tags...) error` | Per-entry TTL override. |
| `Compute(ctx, key, fn) (V, bool, error)` | Atomic update. `fn` receives `(oldValue, found)` and returns `(newValue, ComputeAction)`. |
| `ComputeIfAbsent(ctx, key, fn) (V, bool, error)` | Compute only when the key is missing. |
| `ComputeIfPresent(ctx, key, fn) (V, bool, error)` | Compute only when the key exists. |
| `ComputeWithTTL(ctx, key, fn) (V, bool, error)` | Compute with per-call TTL via `ComputeResult[V]`. |
| `BulkGet(ctx, keys, bulkLoader) (map[K]V, error)` | Batch retrieval; `bulkLoader` fills misses. |
| `BulkSet(ctx, items, tags...) error` | Batch store. On Redis/Valkey this uses pipeline and the multi-set (MSET) command. |

### Invalidate

| Method | Purpose |
|---|---|
| `Invalidate(ctx, key) error` | Remove a single entry. |
| `InvalidateByTags(ctx, tags...) (int, error)` | Remove every entry tagged with any of `tags`; returns the count removed. Variadic. |
| `InvalidateAll(ctx) error` | Remove every entry. |

### Refresh

| Method | Purpose |
|---|---|
| `Refresh(ctx, key, loader) <-chan LoadResult[V]` | Asynchronously reload one key. The cache serves the old value until the refresh completes. |
| `BulkRefresh(ctx, keys, bulkLoader)` | Fire-and-forget batched refresh. |

### Time

| Method | Purpose |
|---|---|
| `SetExpiresAfter(ctx, key, duration) error` | Update the expiry on an existing entry. |
| `SetRefreshableAfter(ctx, key, duration) error` | Update the refresh window on an existing entry. |

### Capacity / stats / lifecycle

| Method | Purpose |
|---|---|
| `EstimatedSize() int` | Approximate entry count. |
| `WeightedSize() uint64` | Total weight (or the same as `EstimatedSize` when weights are off). |
| `GetMaximum() uint64` / `SetMaximum(size)` | Read or change the cache's max capacity. |
| `Stats() Stats` | Snapshot counters (hits, misses, evictions, loads, load failures, size). |
| `Close(ctx) error` | Release resources. |

### Search

`Cache[K, V]` also exposes search when you configure the cache with a `SearchSchema`:

| Method | Purpose |
|---|---|
| `Search(ctx, query, *SearchOptions) (SearchResult[K, V], error)` | Full-text search across `TEXT` fields. |
| `Query(ctx, *QueryOptions) (SearchResult[K, V], error)` | Structured filter / sort / pagination without full-text. |
| `SupportsSearch() bool` | Capability check. |
| `GetSchema() *SearchSchema` | Returns the configured schema, or `nil`. |

`Search` and `Query` return `ErrSearchNotSupported` on providers without search support.

### Transactional providers

Providers that support multi-operation rollback (data access layer (DAL) transactions) implement the `Transactional[K, V]` interface in addition to `Cache[K, V]`:

```go
type Transactional[K, V] interface {
    BeginTransaction(ctx context.Context) TransactionCache[K, V]
}
```

`TransactionCache[K, V]` embeds the cache plus `Commit(ctx) error` and `Rollback(ctx) error`. Providers that do not implement `Transactional` still participate in transactions via Piko's generic journal-based fallback.

### Deletion-event hook

Use the builder's `OnDeletion(func(DeletionEvent[K, V]))` (or `OnAtomicDeletion(...)` for stricter ordering guarantees) to observe cache evictions, replacements, and explicit invalidations. The event's `Cause` field is one of the `Cause*` constants below.

## Options and builders

| Type | Purpose |
|---|---|
| `Options[K, V]` | Settings struct (provider, namespace, max size, TTL, encoder, transformers, stats recorder). |
| `Builder[K, V]` | Fluent configurator, returns a `Cache[K, V]` from `Build(ctx)`. |
| `Entry[K, V]` | Immutable snapshot with key, value, TTL, and metadata. |
| `DeletionEvent[K, V]` | Payload for `OnDeletion` / `OnAtomicDeletion` callbacks. |
| `Loader[K, V]` | `Load(ctx, key) (V, error)`. Called on miss by `Get(ctx, key, loader)`. |
| `LoaderFunc[K, V]` | Function adapter implementing `Loader`. |
| `BulkLoader[K, V]` | Batch variant of `Loader`. |
| `BulkLoaderFunc[K, V]` | Function adapter implementing `BulkLoader`. |
| `LoadResult[V]` | Async load/refresh outcome. |
| `ComputeResult[V]` | Compute outcome with optional TTL override. |
| `ExpiryCalculator[K, V]` | Custom TTL resolver per entry. |
| `RefreshCalculator[K, V]` | Custom refresh window per entry. |
| `Clock` | Injectable time source for tests. |
| `Logger` | Logger port. |
| `Stats` | Snapshot (hits, misses, evictions, loads, load failures, size). |
| `StatsRecorder` | Sink for stats events. |

## Search

Callers declare searchable fields with `SearchSchema` and `FieldSchema`, attach the schema to a cache (via builder option or provider config), and query with `SearchOptions` or `QueryOptions`.

```go
schema := cache.NewSearchSchema(
    cache.TextField("title"),
    cache.TagField("category"),
    cache.SortableNumericField("price"),
)
```

### Field constructors

| Function | Type | Purpose |
|---|---|---|
| `TextField(name)` | `TEXT` | Tokenised full-text. |
| `TagField(name)` | `TAG` | Exact match. |
| `NumericField(name)` | `NUMERIC` | Range queries. |
| `SortableNumericField(name)` | `NUMERIC` sortable | Range queries + sort. |
| `SortableTextField(name)` | `TEXT` sortable | Full-text + sort. |
| `GeoField(name)` | `GEO` | Coordinate-based queries. |
| `VectorField(name, dimension)` | `VECTOR` | Cosine similarity. |
| `VectorFieldWithMetric(name, dimension, metric)` | `VECTOR` | Custom distance metric. |
| `NewSearchSchema(fields...)` | (constructor) | Bundles fields. |
| `NewSearchSchemaWithAnalyser(analyser, fields...)` | (constructor) | Bundles fields with a `TextAnalyseFunc` for stemming, stop words, normalisation. |

### Field type constants

`FieldTypeText`, `FieldTypeTag`, `FieldTypeNumeric`, `FieldTypeGeo`, `FieldTypeVector`.

### Filter constructors

| Function | Operation |
|---|---|
| `Eq(field, value)` | Field equals value. |
| `Ne(field, value)` | Field does not equal value. |
| `Gt(field, value)` | Greater than. |
| `Ge(field, value)` | Greater than or equal. |
| `Lt(field, value)` | Less than. |
| `Le(field, value)` | Less than or equal. |
| `In(field, values...)` | Field value in set. |
| `Between(field, min, max)` | Inclusive range. |
| `Prefix(field, prefix)` | Prefix match on TAG field. |

### Filter op constants

`FilterOpEq`, `FilterOpNe`, `FilterOpGt`, `FilterOpGe`, `FilterOpLt`, `FilterOpLe`, `FilterOpIn`, `FilterOpBetween`, `FilterOpPrefix`.

### Sort order

`SortAsc`, `SortDesc`.

### Search types

| Type | Purpose |
|---|---|
| `SearchOptions` | Full-text search configuration. |
| `QueryOptions` | Structured query without full-text. |
| `SearchResult[K, V]` | Result set with total count and hits. |
| `SearchHit[K, V]` | Single hit (key, value, score, matched fields). |
| `Filter` | Filter condition (built via the filter constructors). |
| `TextAnalyseFunc` | Text analyser for stemming and tokenisation. |

`ErrSearchNotSupported` surfaces when a provider does not ship search.

## Transformers

Transformers apply compression or encryption to cached values. They chain, so the cache can compress a value then encrypt it before storage.

| Constant | Meaning |
|---|---|
| `TransformerCompression` | Compression transformer. |
| `TransformerEncryption` | Encryption transformer. |
| `TransformerCustom` | User-supplied transformer. |

| Type | Purpose |
|---|---|
| `TransformerPort` | Interface (`Transform`, `Reverse`). |
| `TransformerType` | Enum of the constants above. |
| `TransformConfig` | Attachment point on `Options` / builder. |

Shipped implementations:

| Sub-package | Role |
|---|---|
| `cache_transformer_zstd` | Zstandard compression. |
| `cache_transformer_crypto` | AES-GCM encryption. |

Chain order: `Compression().Encryption()` compresses first, then encrypts.

## Deletion causes and compute actions

| Constant | Meaning |
|---|---|
| `CauseInvalidation` | Explicit delete or tag invalidation. |
| `CauseReplacement` | Value overwritten by a later `Set`. |
| `CauseOverflow` | Evicted because the cache was full. |
| `CauseExpiration` | TTL elapsed. |
| `ComputeActionSet` | `Compute` (or one of its variants) returned a value to store. |
| `ComputeActionDelete` | Handler returned a delete signal. |
| `ComputeActionNoop` | Handler signalled no change. |

## Providers

| Sub-package | Role |
|---|---|
| `cache_provider_otter` | In-memory, generational, uses the TinyLFU eviction policy. Default. |
| `cache_provider_redis` | Redis via `go-redis`. |
| `cache_provider_redis_cluster` | Redis Cluster. |
| `cache_provider_valkey` | Valkey. |
| `cache_provider_valkey_cluster` | Valkey Cluster. |
| `cache_provider_multilevel` | L1 plus L2 composition. |
| `cache_provider_mock` | In-memory test double. |
| `cache_encoder_gob`, `cache_encoder_json` | Value encoders. |
| `cache_transformer_crypto`, `cache_transformer_zstd` | Stream transformers. |
| `cache_linguistics` | Text analyser used by search. |

## Encoding

Some providers (Redis, Valkey) need values serialised. The encoding registry accepts a default encoder and type-specific overrides:

```go
registry := cache.NewEncodingRegistry(defaultEncoder)
```

Types: `EncoderPort[V]`, `AnyEncoder`, `EncodingRegistry`.

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithCacheProvider(name, provider)` | Registers a provider under a name. |
| `piko.WithDefaultCacheProvider(name)` | Marks a registered provider as default. |
| `piko.WithCacheService(service)` | Registers a fully configured service. |

## Errors

A `Loader` returns `ErrNotFound` when a value is missing from the data source. Cache instances return `ErrSearchNotSupported` on providers without search.

## See also

- [About caching](../explanation/about-caching.md) for stampede protection, tiered caching, and transformer chain rationale.
- [How to cache](../how-to/cache.md) for adapter swapping, tag-based invalidation, search, and transformer recipes.
- [Scenario 016: cached API](../../examples/scenarios/016_cached_api/) for an action-level cache example.
