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
        piko.WithCacheProvider("memory", cache_provider_otter.New(1 * 1024 * 1024 * 1024)),
        piko.WithDefaultCacheProvider("memory"),
    )
    ssr.Run()
}
```

`cache_provider_otter.New` takes a maximum size in bytes.

## Swap in a distributed adapter

For multi-instance deployments, move to Redis or Valkey:

```go
import (
    "piko.sh/piko/wdk/cache/cache_provider_redis"
)

redisProvider := cache_provider_redis.New(cache_provider_redis.Config{
    Address:  os.Getenv("REDIS_URL"),
    Password: os.Getenv("REDIS_PASSWORD"),
})

ssr := piko.New(
    piko.WithCacheProvider("redis", redisProvider),
    piko.WithDefaultCacheProvider("redis"),
)
```

Redis cluster (`cache_provider_redis_cluster`), Valkey (`cache_provider_valkey`), and Valkey cluster (`cache_provider_valkey_cluster`) follow the same shape.

## Chain a local level-1 cache with a distributed level-2 cache

`cache_provider_multilevel` wraps two providers. Reads check L1 first and fall through to L2 on miss. Writes populate both:

```go
import (
    "piko.sh/piko/wdk/cache/cache_provider_multilevel"
    "piko.sh/piko/wdk/cache/cache_provider_otter"
    "piko.sh/piko/wdk/cache/cache_provider_redis"
)

l1 := cache_provider_otter.New(128 * 1024 * 1024)
l2 := cache_provider_redis.New(cache_provider_redis.Config{Address: os.Getenv("REDIS_URL")})

tiered := cache_provider_multilevel.New(l1, l2)

ssr := piko.New(
    piko.WithCacheProvider("tiered", tiered),
    piko.WithDefaultCacheProvider("tiered"),
)
```

Local L1 keeps hot reads fast. L2 shares state across instances.

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

Or set a default TTL on the cache with `Options.DefaultTTL` at creation. For per-entry refresh windows (refresh before expiry), set `Options.RefreshAfterWrite`.

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

Query with filter constructors:

```go
results, err := customerCache.Search(ctx, cache.SearchOptions{
    Filters: []cache.Filter{
        cache.Eq("region", "EU"),
        cache.Gt("createdAt", startTimestamp),
    },
    SortBy:    "createdAt",
    SortOrder: cache.SortDesc,
    Limit:     20,
})
```

Full-text search on `TextField`s uses `SearchOptions.Query`. Providers without search return `ErrSearchNotSupported`.

## Compress and encrypt cached values

Transformers apply to values on write and reverse on read. Chain compression first, then encryption:

```go
import (
    "piko.sh/piko/wdk/cache/cache_transformer_zstd"
    "piko.sh/piko/wdk/cache/cache_transformer_crypto"
)

crypto := cache_transformer_crypto.New(cache_transformer_crypto.Config{
    Key: []byte(os.Getenv("CACHE_ENCRYPTION_KEY")),
})

opts := cache.Options[string, Customer]{
    MaximumSize: 10000,
    Transformers: []cache.TransformerPort{
        cache_transformer_zstd.New(),
        crypto,
    },
}
```

Compression reduces memory and network cost for cached HTML or JSON. Encryption adds protection when caching personally identifying data in shared infrastructure.

## Invalidate a single entry

```go
customerCache.Invalidate(ctx, id)
```

To clear everything: `customerCache.InvalidateAll(ctx)`. There's no batched single-key invalidation; loop over `Invalidate` or use tags for grouped removal.

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
- [Scenario 016: cached API](../showcase/016-cached-api.md) for an action-level cache example.
