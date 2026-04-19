---
title: How to set a cache policy on a page
description: Cache rendered pages, key per-user variants, and disable caching on sensitive pages.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 740
---

# How to set a cache policy on a page

Pages attach a cache policy by exporting `CachePolicy` from the Go script block. The policy controls server-side AST caching, optional full-page HTML caching, and the `Cache-Control` header on the response. For the rationale behind the cache layers see [about caching](../../explanation/about-caching.md). For every field on the policy struct see [metadata-fields](../../reference/metadata-fields.md).

## Cache a static marketing page aggressively

To cache the rendered HTML for an hour and serve hits without re-running `Render`:

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

`Static: true` skips `Render` entirely on a hit. Use it only when the output does not vary per user.

## Partition the cache key with `policy.Key`

Piko mixes `CachePolicy.Key` into the cache artefact ID alongside the URL and locale. Use it to partition the cache between deploys, A/B variants, or any other namespace that the URL alone does not capture:

```go
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled:       true,
        MaxAgeSeconds: 300,
        Key:           "v2",
    }
}
```

`Key` stays fixed at build time. The user-facing `CachePolicy()` function takes no arguments, and the codegen wrapper that adapts it to the runtime contract discards the request. There is no supported path for per-request cache keys (per-user, per-cookie, per-header) at the page level.

For pages that genuinely vary per user, do not enable full-page caching. Leave `Enabled: false` and let the AST cache (which keys on the rendered AST shape, not the request) speed up render without storing the user-specific HTML:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    user := auth.UserFromContext(r.Context())
    return Response{User: user}, piko.Metadata{Title: "My Profile"}, nil
}

func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled: false,
        NoStore: true,
    }
}
```

Use `CachePolicy.Key` for partitioning. The production response path does not consume the `Metadata.CacheKey` field. See [metadata fields](../../reference/metadata-fields.md) for which fields each path reads.

## Disable caching on a sensitive page

To bypass server-side caching entirely:

```go
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled: false,
        NoStore: true,
    }
}
```

`NoStore: true` short-circuits server-side caching so the page renders fresh on every request. It does not emit `Cache-Control: no-store` automatically. Add a header middleware if your downstream proxies must honour that directive.

## See also

- [Metadata fields reference](../../reference/metadata-fields.md) for every field on `CachePolicy` (and the `Metadata` struct it embeds in).
- [Routing rules reference](../../reference/routing-rules.md) for path matching.
- [About caching](../../explanation/about-caching.md) for tiered caching rationale.
- [Cache API reference](../../reference/cache-api.md) for the underlying cache primitives.
- [How to cache action responses](../actions/caching.md) for the action-level equivalent.
