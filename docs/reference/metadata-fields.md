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
    TwitterCards   []MetaTag
    StructuredData []string
    AlternateLinks []map[string]string
}
```

`InternalMetadata` embeds `Metadata` and also embeds `CachePolicy`. Pages set the cache fields (`Key`, `MaxAgeSeconds`, `Enabled`, `OnRender`, `Static`, `MustRevalidate`, `NoStore`) on the same returned struct. The renderer flattens them into the same JSON document at render time.

## Field reference

| Field | Type | Renders |
|---|---|---|
| `Title` | `string` | `<title>` element |
| `Description` | `string` | `<meta name="description">` |
| `Keywords` | `string` | `<meta name="keywords">` |
| `Language` | `string` | `lang` attribute on `<html>` |
| `CanonicalURL` | `string` | `<link rel="canonical" href="...">` |
| `RobotsRule` | `string` | `<meta name="robots">` (for example `"noindex,nofollow"`) |
| `Status` | `int` | Not consulted by the response writer. See [Status handling](#status-handling). |
| `StatusText` | `string` | Not consulted by the response writer. See [Status handling](#status-handling). |
| `ClientRedirect` | `string` | URL for browser redirect (changes visible URL) |
| `ServerRedirect` | `string` | URL for internal rewrite (preserves visible URL) |
| `RedirectStatus` | `int` | Redirect status for `ClientRedirect`. Accepts `301` (permanent), `302` (temporary, default), `303` (see other), or `307` (temporary, preserve method). |
| `LastModified` | `*time.Time` | `<lastmod>` entry in `sitemap.xml` and Schema.org `dateModified`. Falls back to the source file's modification time when `nil`. Does not set the `Last-Modified` HTTP header. |
| `CacheKey` | `string` | Overrides the automatic cache key used by the render cache |
| `OGTags` | `[]OGTag` | Open Graph tags (`<meta property="og:..." content="...">`) |
| `MetaTags` | `[]MetaTag` | Generic `<meta name="..." content="...">` |
| `TwitterCards` | `[]MetaTag` | Twitter Card meta tags (`twitter:card`, `twitter:site`, `twitter:title`, etc.). Reuses `MetaTag` (`Name`/`Content`). |
| `StructuredData` | `[]string` | Raw JSON-LD blocks rendered as `<script type="application/ld+json">`. The renderer drops invalid JSON entries with a warning log. |
| `AlternateLinks` | `[]map[string]string` | `<link rel="alternate">` entries. Each map carries the link attributes (typically `hreflang` and `href`, optionally `media` or `type`). |

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

### `AlternateLinks` entries

Each entry is a `map[string]string` whose keys are the rendered attribute names. The renderer does not fix a schema, so any combination works:

```go
metadata.AlternateLinks = []map[string]string{
    {"hreflang": "en-GB", "href": "https://example.com/en-gb/"},
    {"hreflang": "fr",    "href": "https://example.com/fr/"},
    {"media": "only screen and (max-width: 640px)", "href": "https://m.example.com/"},
}
```

The unrelated `seo_dto.AlternateLink` struct (used by sitemap generation) is not the type used here.

## Status handling

Callers write `Metadata.Status` and `Metadata.StatusText`, but the response writer does not read them. Successful renders unconditionally write `200 OK`. Non-200 status codes come from typed errors returned by `Render`, not from these fields. The pikotest harness still inspects `Metadata.Status` (so unit-test assertions on the field continue to behave as written), but production responses ignore it.

To return a non-200 status from a page, return a typed error from `Render` (for example `&piko.NotFoundError{Message: "..."}` or `&piko.GenericPageError{Status: 503, Message: "..."}`). See the [errors reference](errors.md) for the full error-type-to-status mapping. For redirects, set `ClientRedirect` (or `ServerRedirect`) and `RedirectStatus` instead of `Status`.

## Redirects

`ClientRedirect` sends an HTTP redirect response, so the browser's address bar changes. `ServerRedirect` rewrites the response in-process, so the browser's address bar does not change.

When both `ServerRedirect` and `ClientRedirect` carry values, `ServerRedirect` takes precedence.

Set `RedirectStatus` alongside `ClientRedirect`. Piko defaults to `302` (Found) when you omit `RedirectStatus` or supply an unsupported value, and ignores `RedirectStatus` entirely when `ClientRedirect` is empty. `ServerRedirect` is an internal rewrite, so the inner page's response writer controls the status code. `RedirectStatus` is not consulted.

## See also

- [Title and OG tags how-to](../how-to/metadata-seo/title-and-og.md).
- [Structured data how-to](../how-to/metadata-seo/structured-data.md).
- [Routing rules reference](routing-rules.md) for status codes returned via typed errors.
- [Errors reference](errors.md).

**Used in.** Every page with custom metadata, including scenarios [001](../../examples/scenarios/001_hello_world/) through [020](../../examples/scenarios/020_m3e_recipe_app/).
