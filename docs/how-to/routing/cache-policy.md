---
title: How to set a cache policy on a page
description: Define CachePolicy on a page to control AST caching, full-page HTML caching, build-time rendering, and Cache-Control headers.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 740
---

# How to set a cache policy on a page

Pages attach a cache policy by exporting `CachePolicy` from the Go script block. The policy controls server-side AST caching, optional full-page HTML caching, build-time pre-rendering, and the `Cache-Control` header on the response. For the rationale behind the cache layers see [about caching](../../explanation/about-caching.md).

## CachePolicy fields

```go
type CachePolicy struct {
    // Key is an optional identifier combined with the URL to form the cache key.
    // Use it for per-user caching, A/B testing, or locale-aware variants.
    Key string

    // MaxAgeSeconds sets the Cache-Control max-age and the server-side TTL.
    MaxAgeSeconds int

    // Enabled is the master switch for server-side AST caching.
    Enabled bool

    // OnRender pre-renders the AST at build time and treats output as static.
    OnRender bool

    // Static enables full-page HTML caching. Highest performance.
    // Only safe for pages that do not vary per-user.
    Static bool

    // MustRevalidate forces revalidation with the origin on every request.
    MustRevalidate bool

    // NoStore prevents the response from being cached anywhere.
    NoStore bool
}
```

## Cache a static marketing page aggressively

```piko
<script type="application/x-go">
package main

import "piko.sh/piko"

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "About Us"}, nil
}

func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled:       true,
        MaxAgeSeconds: 3600,
        Static:        true,
    }
}
</script>
```

`Static: true` caches the rendered HTML, so the request never enters the page's `Render` function on a hit. Use it only when the output does not vary per user.

## Cache per-user pages with a custom key

For pages that depend on the authenticated user, include the user ID in the cache key and disable full-page caching:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    user := auth.UserFromContext(r.Context())
    return Response{User: user}, piko.Metadata{
        Title:    "My Profile",
        CacheKey: fmt.Sprintf("user:%s", user.ID),
    }, nil
}

func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled:       true,
        MaxAgeSeconds: 300,
        Static:        false,
    }
}
```

The AST cache speeds up the render. The full-page HTML cache stays off because the body changes per user.

## Disable caching on a sensitive page

For pages that no intermediary should cache:

```go
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled: false,
        NoStore: true,
    }
}
```

`NoStore: true` adds `Cache-Control: no-store` to the response.

## See also

- [Routing rules reference](../../reference/routing-rules.md) for path matching.
- [About caching](../../explanation/about-caching.md) for tiered caching rationale.
- [Cache API reference](../../reference/cache-api.md) for the underlying cache primitives.
- [How to cache action responses](../actions/caching.md) for the action-level equivalent.
