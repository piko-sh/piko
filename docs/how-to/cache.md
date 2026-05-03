---
title: How to cache data
description: Register a cache provider, create a typed cache, swap adapters, tag entries for group invalidation, run searches, and chain compression with encryption.
nav:
  sidebar:
    section: "how-to"
    subsection: "services"
    order: 50
---

# How to cache data

This guide shows how to register a cache backend, create a typed cache, read through it, invalidate entries, search indexed values, and chain transformers. For the full API surface see [cache API reference](../reference/cache-api.md). For the rationale see [about caching](../explanation/about-caching.md).

## Register a provider at bootstrap

The Otter provider is an in-memory cache that uses the TinyLFU eviction policy. Good default for single-instance servers:

```go
package main

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/cache/cache_provider_otter"
)

func main() {
    ssr := piko.New(
        piko.WithCacheProvider("memory", cache_provider_otter.NewOtterProvider()),
        piko.WithDefaultCacheProvider("memory"),
    )
    ssr.Run()
}
```

`cache_provider_otter.NewOtterProvider()` takes no arguments. Set per-namespace size and eviction limits through `cache.Options[K, V]` when calling `cache.CreateNamespace`.

## Swap in a distributed adapter

For multi-instance deployments, move to Redis or Valkey:

```go
import (
    "piko.sh/piko/wdk/cache"
    "piko.sh/piko/wdk/cache/cache_provider_redis"
)

registry := cache.NewEncodingRegistry(nil)

redisProvider, err := cache_provider_redis.NewRedisProvider(cache_provider_redis.Config{
    Address:  os.Getenv("REDIS_URL"),
    Password: os.Getenv("REDIS_PASSWORD"),
    Registry: registry,
})
if err != nil {
    log.Fatal(err)
}

ssr := piko.New(
    piko.WithCacheProvider("redis", redisProvider),
    piko.WithDefaultCacheProvider("redis"),
)
```

`NewRedisProvider` pings the server during construction and returns an error if unreachable. The `Config.Registry` field is mandatory: it is the `EncodingRegistry` Redis uses to encode and decode cache values. Redis cluster (`cache_provider_redis_cluster`), Valkey (`cache_provider_valkey`), and Valkey cluster (`cache_provider_valkey_cluster`) follow the same shape.

## Chain a local level-1 cache with a distributed level-2 cache

`cache_provider_multilevel` wraps two cache instances of the same key/value types. Reads check L1 first and fall through to L2 on miss. Writes populate both. The adapter is generic over `[K, V]` so the L1 and L2 caches must already share that type:

```go
import (
    "piko.sh/piko/wdk/cache"
    "piko.sh/piko/wdk/cache/cache_provider_multilevel"
)

tiered := cache_provider_multilevel.NewMultiLevelAdapter[string, Customer](
    ctx,
    "customer-tiered",
    l1Cache,
    l2Cache,
    cache_provider_multilevel.Config{
        MaxConsecutiveFailures: 5,
        OpenStateTimeout:       30 * time.Second,
    },
)

_ = tiered
```

`l1Cache` and `l2Cache` are `cache.ProviderPort[K, V]` instances. The circuit-breaker `Config` protects L2 from cascading failures by short-circuiting after consecutive errors. Local L1 keeps hot reads fast. L2 shares state across instances.

## Create a typed cache

Each value type lives in its own namespace. Declare the cache once at package scope so callers share the instance:

```go
package customer

import (
    "context"

    "piko.sh/piko/wdk/cache"
)

type Customer struct {
    ID    int64
    Name  string
    Email string
}

var customerCache cache.Cache[string, Customer]

func InitCache(ctx context.Context) error {
    service, err := cache.GetDefaultService()
    if err != nil {
        return err
    }

    customerCache, err = cache.CreateNamespace[string, Customer](
        ctx, service, "memory", "customer",
        cache.Options[string, Customer]{MaximumSize: 10000},
    )
    return err
}
```

## Read through the cache

Use the loader-taking form of `Get` when a miss should fetch the value from the source:

```go
func GetCustomer(ctx context.Context, id string) (Customer, error) {
    return customerCache.Get(ctx, id, func(ctx context.Context, k string) (Customer, error) {
        return loadCustomerFromDB(ctx, k)
    })
}
```

The loader runs at most once concurrently per key. Parallel callers for the same key block on the first loader. This prevents stampedes when a popular key expires. Use `GetIfPresent(ctx, key)` when a miss should NOT trigger a load.

## Compute atomically

For read-modify-write cases, use `Compute`:

```go
customerCache.Compute(ctx, id, func(current Customer, found bool) (Customer, cache.ComputeAction) {
    if !found {
        return Customer{}, cache.ComputeActionNoop
    }
    current.Name = strings.ToUpper(current.Name)
    return current, cache.ComputeActionSet
})
```

Return `ComputeActionDelete` to evict. `ComputeActionNoop` leaves the entry untouched. For "compute only when absent" or "compute only when present" use `ComputeIfAbsent` or `ComputeIfPresent`. For per-call TTL control use `ComputeWithTTL`.

## Tag entries for group invalidation

`Set` accepts a variadic list of tags. `InvalidateByTags` (variadic plural) removes every entry carrying any of the tags and returns the count removed:

```go
customerCache.Set(ctx, customer.ID, customer, "tenant:"+customer.TenantID, "region:"+customer.Region)
```

When a tenant changes subscription tier, invalidate every cached entry for that tenant in one call:

```go
removed, _ := customerCache.InvalidateByTags(ctx, "tenant:"+tenantID)
```

This avoids scanning the whole cache and avoids tracking key lists in application code.

## Set a TTL

```go
customerCache.SetWithTTL(ctx, id, customer, 10*time.Minute, "tenant:"+tenantID)
```

Set per-entry TTLs through `SetWithTTL` or `ComputeWithTTL`. Set a Redis-level default through `cache_provider_redis.Config.DefaultTTL`. For background refresh, set `Options.RefreshCalculator`. See [cache API reference](../reference/cache-api.md) for the full options surface.

## Search cached values

Some providers expose a search index. Declare the schema at cache creation:

```go
schema := cache.NewSearchSchema(
    cache.TextField("name"),
    cache.TagField("region"),
    cache.SortableNumericField("createdAt"),
)

opts := cache.Options[string, Customer]{
    MaximumSize:  10000,
    SearchSchema: schema,
}
```

For full-text queries call `Search(ctx, query, opts)`. The query string is positional. Pass `nil` for opts to accept the defaults:

```go
results, err := customerCache.Search(ctx, "alice", &cache.SearchOptions{
    Filters: []cache.Filter{
        cache.Eq("region", "EU"),
    },
    SortBy:    "createdAt",
    SortOrder: cache.SortDesc,
    Limit:     20,
})
```

For structured filtering without a text query, call `Query(ctx, opts)`:

```go
results, err := customerCache.Query(ctx, &cache.QueryOptions{
    Filters: []cache.Filter{
        cache.Eq("region", "EU"),
        cache.Gt("createdAt", startTimestamp),
    },
    SortBy:    "createdAt",
    SortOrder: cache.SortDesc,
    Limit:     20,
})
```

Providers without search return `ErrSearchNotSupported`.

## Compress and encrypt cached values

Use the cache builder's `Compression()` and `Encryption()` helpers. Side-effect imports register the `zstd` and `crypto-service` blueprints:

```go
import (
    _ "piko.sh/piko/wdk/cache/cache_transformer_crypto"
    _ "piko.sh/piko/wdk/cache/cache_transformer_zstd"
    "piko.sh/piko/wdk/cache"
)

builder, err := cache.NewCacheBuilderFromDefault[string, Customer]()
if err != nil {
    return err
}

customerCache, err := builder.
    Provider("memory").
    Namespace("customer").
    MaximumSize(10000).
    Compression().
    Encryption().
    Build(ctx)
```

For explicit zstd configuration, pass a config to `Transformer("zstd", config)`:

```go
import (
    "piko.sh/piko/wdk/cache"
    "piko.sh/piko/wdk/cache/cache_transformer_zstd"
)

zstdConfig := cache_transformer_zstd.DefaultConfig()

customerCache, err := builder.
    Provider("memory").
    Namespace("customer").
    MaximumSize(10000).
    Transformer("zstd", zstdConfig).
    Transformer("crypto-service").
    Build(ctx)
```

`Encryption()` resolves the crypto service registered through `WithCryptoService`. For an explicit service call `EncryptionWithService(service)`. See [about caching](../explanation/about-caching.md) for the transformer chain rationale.

## Invalidate a single entry

```go
customerCache.Invalidate(ctx, id)
```

To clear everything, call `customerCache.InvalidateAll(ctx)`. There is no batched single-key invalidation. Loop over `Invalidate` or use tags for grouped removal.

## Observe

`customerCache.Stats()` returns hits, misses, evictions, loads, and current size. Feed these into the bootstrap logger, or plug a `cache.StatsRecorder` into the builder via `.StatsRecorder(recorder)` for a custom sink.

Hook deletion events for audit using the builder's `OnDeletion` (or `OnAtomicDeletion` for stricter ordering):

```go
myCache, err := builder.
    Provider("otter").
    Namespace("customers").
    OnDeletion(func(event cache.DeletionEvent[string, Customer]) {
        metrics.Inc("cache.eviction", "cause", string(event.Cause))
    }).
    Build(ctx)
```

## See also

- [Cache API reference](../reference/cache-api.md) for every method, constructor, and constant.
- [About caching](../explanation/about-caching.md) for stampede protection, tiered caching, and transformer chain rationale.
- [Scenario 016: cached API](../../examples/scenarios/016_cached_api/) for an action-level cache example.
