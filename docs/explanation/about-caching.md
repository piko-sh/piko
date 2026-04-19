---
title: About caching
description: Stampede protection, tiered caching, compute-patterns, and why tag-based invalidation sits alongside key-based.
nav:
  sidebar:
    section: "explanation"
    subsection: "operations"
    order: 60
---

# About caching

Caching looks simple from the outside. Store a value under a key. Return it faster next time. The internals of a good cache library have more weight than that. Stampede protection, tiered storage, atomic updates, group invalidation. Piko's cache API borrows its shape from `maypok86/otter/v2` and adds the Piko-level hooks that multi-instance production deployments need. This page explains the reasoning behind each piece.

## Stampede protection through loaders

A cache miss that triggers an expensive load is a footgun. If ten concurrent requests hit the same missing key, a naive cache runs the loader ten times. Database gets hammered. The service cascades into failure under what should have been a modest load.

Piko's `Get(ctx, key, loader)` serialises concurrent loads for the same key. The first caller runs the loader. Every other caller waiting for that key blocks on the first loader's result. When the loader finishes, all callers receive the value. Only one database query ran for that process.

The serialisation comes from the underlying Otter L1 cache. It applies cleanly to single-process deployments and to multilevel deployments where every instance has its own L1. It is not a distributed lock. An L2-only deployment that bypasses the local Otter layer (talking straight to Redis or Valkey) gets per-instance coalescing at best. It may still see one loader run per instance under contention. Distributed coalescing requires either keeping the multilevel wrapper in front of L2 or using a provider-side feature such as a Redis lock.

This is also why application code should prefer `Get` (the loader-taking form) over the naive `GetIfPresent` / populate pattern. The latter works but requires the caller to re-implement the serialisation logic. The former gets it for free wherever the Otter layer is present.

## Compute patterns as atomic updates

Sometimes the question is not "fetch this value" but "update this value". The cache should not be a source of race conditions in that case. The compute family of methods gives atomic get-or-compute semantics:

```go
cache.Compute(ctx, key, func(current V, found bool) (V, ComputeAction) {
    current.Counter++
    return current, cache.ComputeActionSet
})
```

The callback runs under a lock for that key. Concurrent `Compute` calls for the same key serialise. The cache itself becomes a coordination primitive, not just a store. This pattern fits rate limiters, counters, and any value that read-modify-writes from multiple goroutines.

`ComputeIfAbsent`, `ComputeIfPresent`, and `ComputeWithTTL` are variants for the cases where you want the callback to fire only on a specific presence state, or where the new entry needs its own TTL.

`ComputeAction` lets the callback decide what to do after computing. The three options store the new value, delete the entry, or make no change. The semantics stay explicit. A caller that produces a zero value does not accidentally store zeros all over the cache. It returns `ComputeActionNoop`.

## Tiered caching for multi-instance deployments

Single-process caches work fine for one process. Multi-instance deployments face a choice. A purely distributed cache (Redis, Valkey) gives every instance the same view but adds a network hop to every read. A purely local cache is fast but each instance gets its own misses, and a value loaded by instance A is not visible to instance B.

Piko's `cache_provider_multilevel` is the compromise. Level 1 is the local Otter cache in each instance. Level 2 is the shared Redis or Valkey cache. A read checks L1 first. If L1 misses, the read checks L2. If L2 has the value, the cache populates L1 and returns the value. If L2 misses, the loader runs and both levels populate on write.

The trade-off is staleness. An instance that has a value cached in L1 does not observe an L2 invalidation. Multi-level caches get their staleness bound from the L1 TTL. Short L1 TTLs give near-fresh reads at the cost of more L2 round-trips. Long L1 TTLs give faster reads with a larger staleness window. Pick the bound that matches the application's tolerance.

For data that must be strictly consistent across instances, use L2 directly without the multilevel wrapper. For data where a minute or two of staleness is acceptable (product catalogue, configuration), multilevel wins.

## Transformer chains

Transformers run on the value bytes during write and reverse during read. Each transformer is a pair of functions that agree on a transformation direction. A compression transformer writes zstd-compressed bytes and reads them back by decompressing. An encryption transformer encrypts on write and decrypts on read.

Transformers chain. `Options.TransformConfig.EnabledTransformers` is an ordered list of registered transformer names. Writes run forward through the list, and reads run backward. A natural chain puts compression first, then encryption. Small data ciphertext stays small. Reversing the order (encryption first, then compression) is pointless because encrypted output is incompressible.

Chaining keeps each transformer small. `cache_transformer_zstd` does one job (zstd). `cache_transformer_crypto` does one job (AES-GCM). A custom transformer for domain-specific serialisation (for example, FlatBuffers) joins the same chain without either built-in transformer knowing about it.

## Why tag-based invalidation exists alongside key-based

The cache stores (key, value) pairs. Invalidating by key is the natural primitive. But real applications often need to invalidate a group of entries together. When a tenant switches plans, every cached entry for that tenant goes stale. Iterating the cache to find those entries is slow, and tracking which keys belong to which tenants in application code is error-prone.

Tags solve this. An entry carries a list of tags on write:

```go
cache.Set(ctx, key, value, "tenant:42", "region:eu")
```

`InvalidateByTags(ctx, "tenant:42")` removes every entry carrying the tag in one call (it accepts multiple tags variadically and returns the count removed). The cache maintains the tag-to-keys index internally. Applications do not track keys. They describe the group the entry belongs to and invalidate by group.

Tags are not a replacement for keys. A value still has exactly one key. Tags are a secondary index that makes group invalidation efficient.

## Why search sits inside the cache

Most caches are pure key-value. Piko's cache supports an optional search index because some use cases conflate the two. A product catalogue cache benefits from a free-text query over `title` alongside the ability to fetch a single product by ID. Splitting the two (a cache for by-ID reads and a separate search index) duplicates the data and the invalidation logic.

Providers that ship a search index (Redis with RediSearch, some Valkey builds) expose it through Piko's `Search(ctx, query, opts)` method. The query string is positional and `*SearchOptions` carries pagination, sort, and structured filters. A sibling `Query(ctx, opts)` method handles structured filtering without a free-text term. Providers without search return `ErrSearchNotSupported`. Callers declare the schema at cache creation, and the provider indexes on write. Typed constructors (`Eq`, `Gt`, `Between`, `SortDesc`) build filters and sort orders instead of raw strings, keeping the query vocabulary small and verifiable at compile time.

## Why typed namespaces

A cache for `[]byte` values is awkward in an application that also caches `User`, `Product`, and `Order` types. Each caller serialises and deserialises, and every miss path repeats the same boilerplate.

Generic types let Piko give each domain its own typed cache:

```go
userCache    cache.Cache[string, User]
productCache cache.Cache[string, Product]
```

A namespace isolates the key space so user ID `42` does not collide with product ID `42`. The generic parameters make every call type-safe. `productCache.Get(ctx, "42")` returns a `Product`, not `any`.

The trade-off is that domain packages usually initialise their caches at startup. A cache does not get created inline at the call site. The cost is worth it. The alternative reintroduces the type-checking burden in every caller.

## Stats and observation

Every cache instance tracks hits, misses, evictions, loads, and size. `Stats()` returns a snapshot. A production deployment plugs a `StatsRecorder` into `Options.StatsRecorder` to stream the counters into the metrics pipeline. Cache hit ratio matters more than cache size. A cache with 100 percent hit ratio at 10 MB does more useful work than a cache with 30 percent hit ratio at 10 GB.

Eviction events carry a cause (`CauseOverflow`, `CauseExpiration`, `CauseInvalidation`, `CauseReplacement`). Sustained `CauseOverflow` evictions mean the cache is too small. Sustained `CauseExpiration` evictions on values that should be long-lived mean the TTL is too tight.

## See also

- [Cache API reference](../reference/cache-api.md) for the full surface.
- [How to cache](../how-to/cache.md) for adapter swapping, tag-based invalidation, search, and transformer recipes.
- [Scenario 016: cached API](../../examples/scenarios/016_cached_api/) for a runnable example.
- [About the hexagonal architecture](about-the-hexagonal-architecture.md) for how the provider pattern fits the rest of Piko.
