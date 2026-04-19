# PK File Format

Use this guide when creating or modifying `.pk` files - pages, partials, or email templates.

## File structure

A `.pk` file has up to four sections, all optional, in any order:

```piko
<template>
  <h1>{{ state.Title }}</h1>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Title string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{Title: "Hello"}, piko.Metadata{}, nil
}
</script>

<style>
h1 { color: blue; }
</style>

<i18n lang="json">
{
  "en": { "greeting": "Hello" },
  "fr": { "greeting": "Bonjour" }
}
</i18n>
```

## Script section

The `<script type="application/x-go">` block contains server-side Go code.

### Required elements

- **Package declaration**: `package main` (or directory name)
- **Response struct**: Holds data for the template, accessed via `state`
- **Render function**: Entry point returning `(Response, piko.Metadata, error)`

```go
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

type Response struct {
    Title string
    Posts []Post
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    posts, err := domain.GetRecentPosts(10)
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    return Response{
        Title: "Blog",
        Posts: posts,
    }, piko.Metadata{
        Title:       "My Blog",
        Description: "Latest blog posts",
    }, nil
}
```

### piko.NoResponse and piko.NoProps

Use `piko.NoResponse` when the template needs no server data. Use `piko.NoProps` when the component accepts no props:

```go
func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "About"}, nil
}
```

### RequestData methods

| Method | Description |
|--------|-------------|
| `Context()` | Request context |
| `Method()` | HTTP method (GET, POST, etc.) |
| `Host()` | Request host |
| `URL()` | Parsed URL (defensive copy) |
| `Locale()` | Current locale |
| `DefaultLocale()` | Fallback locale |
| `PathParam(key)` | URL path parameter |
| `PathParams()` | All path parameters (`map[string]string`) |
| `QueryParam(key)` | First query parameter value |
| `QueryParamValues(key)` | All values for a query param |
| `FormValue(key)` | First form field value |
| `FormValues(key)` | All values for a form field |
| `T(key, fallback...)` | Global translation lookup |
| `LT(key, fallback...)` | Local (component-scoped) translation |

### Metadata

```go
piko.Metadata{
    Title:          "Page Title",
    Description:    "Meta description",
    Language:       "en",
    CanonicalURL:   "https://example.com/page",
    Status:         200,
    StatusText:     "Not Found",      // Custom text for non-200 status
    ClientRedirect: "/new-location",  // HTTP redirect
    ServerRedirect: "/internal-page", // Internal rewrite
    RedirectStatus: 301,              // Default: 302 (valid: 301, 302, 303, 307)
}
```

### Optional functions

```go
func Middlewares() []string {
    return []string{"auth", "csrf"}
}

func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: 3600}
}

func SupportedLocales() []string {
    return []string{"en", "fr", "de"}
}
```

### Page caching (CachePolicy)

Control server-side response caching per page or partial. When enabled, Piko caches the rendered HTML and serves pre-compressed variants (Brotli/Gzip) automatically.

```go
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled:        true,
        Static:         true,
        MaxAgeSeconds:  3600,
        Key:            "",
        MustRevalidate: false,
        NoStore:        false,
        OnRender:       false,
    }
}
```

| Field | Type | Purpose |
|-------|------|---------|
| `Enabled` | `bool` | Master switch - if `false`, response is never cached |
| `Static` | `bool` | Enable full-page HTML caching. Only for pages that don't vary per user |
| `MaxAgeSeconds` | `int` | Cache TTL in seconds. Also sets `Cache-Control: public, max-age=N` |
| `Key` | `string` | Optional custom key combined with URL for cache key. Use for per-user caching or A/B testing |
| `MustRevalidate` | `bool` | Adds `must-revalidate` to `Cache-Control` |
| `NoStore` | `bool` | Prevents caching entirely. Adds `no-store` to `Cache-Control` |
| `OnRender` | `bool` | Pre-render at build time with long TTL |

**How it works**:

- **Cache hit**: Serves pre-rendered, pre-compressed HTML directly. Response includes `X-Cache-Status: HIT`
- **Cache miss**: Renders the page, caches it, generates Brotli/Gzip variants in the background. Response includes `X-Cache-Status: MISS`
- **ETag validation**: Clients can send `If-None-Match` to get `304 Not Modified` responses
- **Singleflight**: Multiple concurrent requests for the same uncached page only render once - others wait for the result
- **Cache keys**: Generated from URL + locale + custom `Key` field

**Common patterns**:

```go
// Static marketing page - cache for 24 hours
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: 86400}
}

// Frequently updated page - short TTL
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{Enabled: true, Static: true, MaxAgeSeconds: 60}
}

// Per-user dashboard - no static caching
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{Enabled: false}
}

// Sensitive data - prevent all caching
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{NoStore: true}
}
```

## Template section

The `<template>` block contains HTML with Piko directives and `{{ state.Field }}` interpolation. See `references/template-syntax.md` for the full directive reference.


## Importing partials

Import `.pk` files to use as reusable components:

```go
import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
    header "myapp/partials/header.pk"
)
```

Invoke with `<piko:partial>` and the `is` attribute set to the import alias. Only `<piko:partial>` triggers partial invocation; an `is` attribute on any other tag is treated as a regular HTML attribute.

```piko
<piko:partial is="layout" :server.page_title="state.Title">
  <piko:partial is="header" :server.user="state.CurrentUser"></piko:partial>
  <main>{{ state.Content }}</main>
</piko:partial>
```

## Style section

```piko
<style>
h1 { color: var(--g-colour-primary); }
</style>
```

- All component types (pages, partials, emails) are scoped by default
- Use `<style global>` for unscoped styles
- Multiple `<style>` blocks allowed
- See `references/styling.md` for scoping details

## i18n section

```piko
<i18n lang="json">
{
  "en": { "greeting": "Hello" },
  "fr": { "greeting": "Bonjour" }
}
</i18n>
```

Access in templates with `{{ LT("greeting") }}` and in Go with `r.LT("greeting", "fallback")`.

## Props (partials only)

```go
type Props struct {
    Title     string `prop:"title"`
    IsPrimary bool   `prop:"is-primary" coerce:"true"`
    ItemCount int    `prop:"item-count" coerce:"true"`
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    // Use props.Title, props.IsPrimary, etc.
}
```

| Tag | Purpose |
|-----|---------|
| `prop:"name"` | Maps attribute to field (kebab-case) |
| `default:"value"` | Default if not provided |
| `factory:"FuncName"` | Factory function for complex defaults |
| `validate:"required"` | Mark as required |
| `coerce:"true"` | Type coercion (string to int, bool, etc.) |
| `query:"param"` | Bind to URL query parameter |

## Error pages

Error pages use the `!` prefix convention in `pages/`:

| Filename | Handles |
|----------|---------|
| `!404.pk` | Exact status code 404 |
| `!400-499.pk` | Any status in range 400-499 |
| `!error.pk` | Any error status code (catch-all) |

Priority: exact (`!404.pk`) > range (`!400-499.pk`) > catch-all (`!error.pk`). Deeper directories take precedence within each tier.

Use `piko.GetErrorContext(r)` to access `StatusCode`, `Message`, and `OriginalPath`. Returns `nil` when not rendering as an error page.

Only the three formats above are valid - any other `!`-prefixed `.pk` file causes a build error.

## LLM mistake checklist

- **Using `{{ }}` inside attributes** - `{{ }}` is ONLY for text content between tags. For dynamic attributes use `:` prefix: `:href="state.URL"`. Never write `href="{{ state.URL }}"` or `href={{ state.URL }}`
- Forgetting `type="application/x-go"` on the script tag
- Forgetting `package main` in the script block
- Using `props` in a page (pages use `piko.NoProps`)
- Accessing state without `state.` prefix in templates
- Using `v-if` or `v-for` instead of `p-if` / `p-for`
- Writing JavaScript in template expressions (Piko has its own expression language)
- Using `(item, index)` order in for loops (it's `(index, item)` - Go order)
- Note: `:prop` renders value as an HTML attribute too; `:server.prop` is server-only (use `:server.prop` when you don't want the prop exposed in rendered HTML)
- Returning `error` from Render without handling it in the template

## Related

- `references/template-syntax.md` - directives and expressions
- `references/partials-and-slots.md` - partial props and slots
- `references/server-actions.md` - form actions and response helpers
- `references/styling.md` - CSS scoping and themes
- `references/pk-javascript-interactive.md` - client-side TypeScript, lifecycle hooks, event handling, piko namespace
