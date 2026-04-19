---
title: How to cache action responses
description: Implement Cacheable to set a TTL, vary by header, and build custom cache keys for action responses.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 730
---

# How to cache action responses

Actions implement `Cacheable` to opt into response caching. The framework keys cached entries by action name and arguments by default, with hooks to vary by request header or build a custom key. For the action structure see [server actions reference](../../reference/server-actions.md). For caching across the framework see [about caching](../../explanation/about-caching.md).

## Add a CacheConfig receiver

Implement `Cacheable` by adding a `CacheConfig()` receiver that returns a `*piko.CacheConfig`:

```go
type Cacheable interface {
    CacheConfig() *piko.CacheConfig
}

type CacheConfig struct {
    // TTL is the cache time-to-live duration.
    TTL time.Duration

    // VaryHeaders lists headers that affect the cache key.
    // Different header values produce different cache entries.
    VaryHeaders []string

    // KeyFunc is an optional function to generate custom cache keys.
    // If nil, the default key combines the action name and arguments.
    KeyFunc func(*http.Request) string
}
```

## Cache a list endpoint for five minutes

```go
package product

import (
    "fmt"
    "net/http"
    "time"

    "piko.sh/piko"
    "myapp/pkg/dal"
)

type ListResponse struct {
    Products []dal.Product `json:"products"`
}

type ListAction struct {
    piko.ActionMetadata
}

func (a ListAction) Method() piko.HTTPMethod {
    return piko.MethodGet
}

func (a ListAction) CacheConfig() *piko.CacheConfig {
    return &piko.CacheConfig{
        TTL:         5 * time.Minute,
        VaryHeaders: []string{"Accept-Language"},
        KeyFunc: func(r *http.Request) string {
            return r.URL.Path + "?" + r.URL.RawQuery
        },
    }
}

func (a ListAction) Call(category string) (ListResponse, error) {
    products, err := dal.ListProducts(a.Ctx(), category)
    if err != nil {
        return ListResponse{}, fmt.Errorf("listing products: %w", err)
    }

    return ListResponse{Products: products}, nil
}
```

Each distinct `Accept-Language` value produces its own cache entry. The query string is part of the key so paginated requests do not collide.

## Disable caching on a destructive action

Pair `Cacheable` with `MethodOverridable` to ensure mutating endpoints stay uncached, even if a route mistakenly matches a cacheable shape:

```go
func (a DeleteAction) CacheConfig() *piko.CacheConfig {
    return &piko.CacheConfig{TTL: 0}
}
```

A zero `TTL` short-circuits the cache lookup.

## Compose with other interfaces

`Cacheable` composes with `MethodOverridable`, `RateLimitable`, and `ResourceLimitable`. See [How to override an action's HTTP method](method-override.md), [How to rate-limit an action](rate-limiting.md), and [How to set resource limits on an action](resource-limits.md).

## See also

- [Server actions reference](../../reference/server-actions.md) for the action surface.
- [Cache API reference](../../reference/cache-api.md) for the underlying cache primitives.
- [About caching](../../explanation/about-caching.md) for stampede protection and tiered caching design.
- [How to cache data](../cache.md) for direct cache use outside the action layer.
