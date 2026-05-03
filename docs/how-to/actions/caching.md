---
title: How to cache action responses
description: Set a TTL, vary by header, and disable caching on destructive actions.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 730
---

# How to cache action responses

Implement `Cacheable` on an action when you want responses memoised by argument values and headers. Piko keys cached entries by action name, parsed arguments, and the headers you list. For the action surface see [server-actions](../../reference/server-actions.md). For the wider caching design see [about caching](../../explanation/about-caching.md).

## Cache responses for a fixed TTL

To cache a list endpoint for five minutes, varying by `Accept-Language`:

```go
type ListInput struct {
    Category string `json:"category"`
}

func (a ListAction) CacheConfig() *piko.CacheConfig {
    return &piko.CacheConfig{
        VaryHeaders: []string{"Accept-Language"},
        TTL:         5 * time.Minute,
    }
}

func (a ListAction) Call(input ListInput) (ListResponse, error) {
    products, err := dal.ListProducts(a.Ctx(), input.Category)
    if err != nil {
        return ListResponse{}, fmt.Errorf("listing products: %w", err)
    }
    return ListResponse{Products: products}, nil
}
```

Each distinct value of `input.Category` and each distinct `Accept-Language` produces a separate cache entry. Make sure every variable the response depends on is a `Call` parameter. The framework exposes query parameters on `a.Request().QueryParams` but does not bind them into the arguments map, so they do not contribute to the cache key.

For GET-method actions, read query strings via `a.Request().QueryParams` inside `Call`. Query values do not contribute to the cache key. See [How to override an action's HTTP method](method-override.md).

## Disable caching on a destructive action

To short-circuit the cache lookup on a mutating action that shares a base type with cacheable siblings, return a zero TTL:

```go
func (a DeleteAction) CacheConfig() *piko.CacheConfig {
    return &piko.CacheConfig{TTL: 0}
}
```

## See also

- [Server actions reference](../../reference/server-actions.md) for the action surface.
- [Cache API reference](../../reference/cache-api.md) for the underlying cache primitives.
- [About caching](../../explanation/about-caching.md) for stampede protection and tiered caching.
- [How to cache data](../cache.md) for direct cache use outside the action layer.
