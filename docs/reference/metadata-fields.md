---
title: Metadata fields
description: Every field on the Metadata struct and how Piko renders it.
nav:
  sidebar:
    section: "reference"
    subsection: "runtime"
    order: 20
---

# Metadata fields

`piko.Metadata` is the struct returned as the second value from a page's `Render` function. Its fields control `<title>`, `<meta>`, social-sharing tags, canonical URLs, caching directives, and redirects. This page enumerates every field. For task recipes see the how-to guides on [title and OG tags](../how-to/metadata-seo/title-and-og.md) and [structured data](../how-to/metadata-seo/structured-data.md).

## Schema

```go
type Metadata struct {
    Title          string
    Description    string
    Keywords       string
    Language       string
    CanonicalURL   string
    RobotsRule     string
    Status         int
    StatusText     string
    ClientRedirect string
    ServerRedirect string
    RedirectStatus int
    LastModified   *time.Time
    CacheKey       string
    OGTags         []OGTag
    MetaTags       []MetaTag
    AlternateLinks []AlternateLink
}
```

## Field reference

| Field | Type | Renders |
|---|---|---|
| `Title` | `string` | `<title>` element |
| `Description` | `string` | `<meta name="description">` |
| `Keywords` | `string` | `<meta name="keywords">` |
| `Language` | `string` | `lang` attribute on `<html>` |
| `CanonicalURL` | `string` | `<link rel="canonical" href="...">` |
| `RobotsRule` | `string` | `<meta name="robots">` (for example `"noindex,nofollow"`) |
| `Status` | `int` | HTTP status code (default `200`) |
| `StatusText` | `string` | HTTP status text |
| `ClientRedirect` | `string` | URL for browser redirect (changes visible URL) |
| `ServerRedirect` | `string` | URL for internal rewrite (preserves visible URL) |
| `RedirectStatus` | `int` | Redirect status (`301`, `302`, `303`, `307`, `308`) |
| `LastModified` | `*time.Time` | `Last-Modified` header and Schema.org datetime |
| `CacheKey` | `string` | Overrides the automatic cache key used by the render cache |
| `OGTags` | `[]OGTag` | Open Graph tags (`<meta property="og:..." content="...">`) |
| `MetaTags` | `[]MetaTag` | Generic `<meta name="..." content="...">` |
| `AlternateLinks` | `[]AlternateLink` | `<link rel="alternate" hreflang="...">` |

### `OGTag`

```go
type OGTag struct {
    Property string
    Content  string
}
```

Common properties: `og:title`, `og:description`, `og:type`, `og:url`, `og:image`, `og:site_name`.

### `MetaTag`

```go
type MetaTag struct {
    Name    string
    Content string
}
```

### `AlternateLink`

```go
type AlternateLink struct {
    Hreflang string
    Href     string
}
```

## Status handling

| `Status` | Effect |
|---|---|
| `0` or `200` | Standard response. |
| `201`, `204`, etc. | Success variants. |
| `301`, `302`, `307`, `308` | Redirect. Pair with `ClientRedirect` or `ServerRedirect`. |
| `404` | Not found. Error page handler renders the response body. |
| `410` | Gone. |
| `500` | Server error. |

See the [errors reference](errors.md) for error-type-to-status mapping.

## Redirects

`ClientRedirect` sends an HTTP redirect response, so the browser's address bar changes. `ServerRedirect` rewrites the response internally, so the browser's address bar does not change.

Set `RedirectStatus` alongside. If unspecified, Piko defaults to `302` for `ClientRedirect` and `200` for `ServerRedirect`.

## See also

- [Title and OG tags how-to](../how-to/metadata-seo/title-and-og.md).
- [Structured data how-to](../how-to/metadata-seo/structured-data.md).
- [Routing rules reference](routing-rules.md) for how `Status` interacts with routing.
- [Errors reference](errors.md).

**Used in.** Every page with custom metadata, including scenarios [001](../showcase/001-hello-world.md) through [020](../showcase/020-m3e-recipe-app.md).
