---
title: How to add custom error pages
description: Register 404, 500, range, and catch-all error pages using the ! prefix convention.
nav:
  sidebar:
    section: "how-to"
    subsection: "errors"
    order: 35
---

# How to add custom error pages

Add a `.pk` file whose name starts with `!` to serve a custom response for an error status code. See the [errors reference](../reference/errors.md) for the full error-type-to-status mapping and the [routing rules reference](../reference/routing-rules.md) for the file-naming rules this guide relies on.

## The `!` prefix convention

Error pages are `.pk` files whose name starts with `!`. There are three formats:

| Format | Example | Description |
|--------|---------|-------------|
| `!NNN.pk` | `!404.pk` | Exact status code |
| `!NNN-NNN.pk` | `!400-499.pk` | Status code range |
| `!error.pk` | `!error.pk` | Catch-all (any error code) |

```text
pages/
├── index.pk          -> /
├── !404.pk           -> Custom "Not Found" page
├── !500.pk           -> Custom "Internal Server Error" page
├── !400-499.pk       -> Handles any 4xx client error
├── !error.pk         -> Catch-all for any error code
├── app/
│   ├── dashboard.pk  -> /app/dashboard
│   └── !404.pk       -> Custom 404 for /app/* routes
└── admin/
    └── !403.pk       -> Custom "Forbidden" for /admin/* routes
```

Error pages are **not** routable. They do not create URL routes, and Piko only renders them when an error occurs.

> **Note**: Piko uses the `!` prefix because it is valid on every platform (Linux, macOS, Windows), signals "not a normal page" visually, and does not conflict with `_` (private partials) or `{param}` (dynamic routes). Only the three formats above are valid. Any other `!`-prefixed `.pk` file causes a build error.

## Creating a custom 404 page

Create `pages/!404.pk`:

```piko
<template>
  <div class="error-page">
    <h1>Page Not Found</h1>
    <p>The page you are looking for does not exist.</p>
    <a href="/">Go home</a>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Message string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        Message: "The page you are looking for does not exist.",
    }, piko.Metadata{Title: "Page Not Found"}, nil
}
</script>

<style>
.error-page { text-align: center; padding: 4rem 2rem; }
.error-page h1 { font-size: 3rem; margin-bottom: 1rem; }
.error-page a { color: #3b82f6; text-decoration: underline; }
</style>
```

When a visitor navigates to a URL that does not match any route, Piko renders this page with HTTP status 404.

## Accessing error context

Error pages can access details about the error that triggered them using `piko.GetErrorContext()`:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    statusCode := 404
    message := "The page you are looking for does not exist."
    path := r.URL().Path

    if errCtx := piko.GetErrorContext(r); errCtx != nil {
        statusCode = errCtx.StatusCode
        path = errCtx.OriginalPath
        if errCtx.Message != "" {
            message = errCtx.Message
        }
    }

    return Response{
        StatusCode: statusCode,
        Message:    message,
        Path:       path,
    }, piko.Metadata{Title: "Error"}, nil
}
```

The `ErrorPageContext` struct contains:

| Field | Type | Description |
|-------|------|-------------|
| `StatusCode` | `int` | The HTTP status code (for example 404, 500) |
| `Message` | `string` | A human-readable error message |
| `OriginalPath` | `string` | The URL path that caused the error |

> **Note**: `piko.GetErrorContext()` returns `nil` when Piko does not render the page as an error page. This lets you write error pages that also work as regular pages during development.

## Resolution priority

When an error occurs, Piko resolves error pages using a three-tier priority system. Within each tier, the most specific scope (deepest directory) wins:

1. **Exact match**: `!404.pk` matches only status 404
2. **Range match**: `!400-499.pk` matches any status in that range
3. **Catch-all**: `!error.pk` matches any error status code

For example, if a 404 occurs at `/app/missing` and these error pages exist:

```text
pages/
├── !error.pk         -> 3rd priority (catch-all)
├── !400-499.pk       -> 2nd priority (range)
├── !404.pk           -> 1st priority (exact match - wins)
└── app/
    └── !404.pk       -> 1st priority + more specific scope - wins over root !404.pk
```

Piko picks `pages/app/!404.pk` because it is an exact match **and** has the most specific scope.

## Hierarchical scoping

Error pages are hierarchical. Piko resolves the most specific error page first, falling back to broader scopes:

```text
pages/
├── !404.pk               -> Handles 404s for all routes
├── app/
│   ├── !404.pk           -> Handles 404s for /app/* routes
│   └── settings/
│       └── !404.pk       -> Handles 404s for /app/settings/* routes
└── admin/
    └── !404.pk           -> Handles 404s for /admin/* routes
```

For a request to `/app/settings/nonexistent`:
1. Piko checks for `pages/app/settings/!404.pk`: **found**, renders it
2. If not found, checks `pages/app/!404.pk`
3. If not found, checks `pages/!404.pk`
4. If no custom error page exists, returns the default plain text response

This lets you create different error experiences for different sections of your application. For example, use a branded error page for the marketing site and a more functional one for the application dashboard.

## Catch-all error page

Create `pages/!error.pk` to handle **any** error status code with a single page:

```piko
<!-- pages/!error.pk -->
<template>
  <div class="error-page">
    <h1>{{ state.StatusCode }}</h1>
    <p>{{ state.Message }}</p>
    <a href="/">Go home</a>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    StatusCode int
    Message    string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    statusCode := 500
    message := "Something went wrong."

    if errCtx := piko.GetErrorContext(r); errCtx != nil {
        statusCode = errCtx.StatusCode
        if errCtx.Message != "" {
            message = errCtx.Message
        }
    }

    return Response{
        StatusCode: statusCode,
        Message:    message,
    }, piko.Metadata{Title: "Error"}, nil
}
</script>
```

This works well when you want a single error page instead of one per status code. Exact matches (`!404.pk`) and ranges (`!400-499.pk`) take priority over catch-all.

## Range error pages

Create `pages/!NNN-NNN.pk` to handle a range of status codes:

```text
pages/
├── !400-499.pk       -> All client errors (400, 401, 403, 404, 422, etc.)
├── !500-599.pk       -> All server errors
└── !404.pk           -> Exact 404 (takes priority over range)
```

Range pages are useful when you want the same error treatment for an entire class of errors. Both bounds are inclusive and must be valid HTTP status codes (100-599) with the lower bound first.

## Collection 404 pages

When using `p-collection` with a dynamic route (for example `pages/blog/{slug}.pk`), navigating to a slug that does not exist in the collection automatically triggers a 404 error, which renders the appropriate `!404.pk` error page.

This means you do not need to manually handle missing collection items. Piko detects the missing item and falls through to your error page hierarchy automatically.

## Typed errors in Render functions

When your page's Render function returns an error, Piko inspects the error type to determine the correct HTTP status code and render the matching error page.

### Built-in error helpers

Piko provides typed error helpers that map to specific HTTP status codes:

```go
import "piko.sh/piko"

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    user, err := getUser(r.PathParam("id"))
    if err != nil {
        // Returns 404 and renders !404.pk
        return Response{}, piko.Metadata{}, piko.NotFound("user", r.PathParam("id"))
    }

    if !user.CanViewDashboard {
        // Returns 403 and renders !403.pk
        return Response{}, piko.Metadata{}, piko.Forbidden("you do not have dashboard access")
    }

    return Response{User: user}, piko.Metadata{}, nil
}
```

| Helper | Status Code | Use Case |
|--------|-------------|----------|
| `piko.NotFound(resource, id)` | 404 | Resource does not exist |
| `piko.NotFoundResource(resource)` | 404 | Resource type not found (no specific ID) |
| `piko.Forbidden(message)` | 403 | Authenticated but lacks permission |
| `piko.Unauthorised(message)` | 401 | Authentication required |
| `piko.BadRequest(message)` | 400 | Malformed request |
| `piko.Conflict(message)` | 409 | State conflict |
| `piko.ConflictWithCode(message, code)` | 409 | Conflict with machine-readable code |
| `piko.NewValidationError(fields)` | 422 | Field validation failures |
| `piko.ValidationField(field, message)` | 422 | Single field validation failure |
| `piko.PageError(statusCode, message)` | *any* | Arbitrary status code |
| `piko.Teapot(message)` | 418 | Teapot response (RFC 2324) |

### Generic page errors

For status codes without a dedicated helper, use `piko.PageError()`:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    if isOverLimit(r.Context()) {
        return Response{}, piko.Metadata{}, piko.PageError(429, "too many requests")
    }
    return Response{}, piko.Metadata{}, nil
}
```

### Plain errors default to 500

Returning a plain `error` (or any error that does not implement `ActionError`) results in HTTP 500 and renders `!500.pk`:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    data, err := fetchData(r.Context())
    if err != nil {
        // Plain error -> 500, renders !500.pk
        return Response{}, piko.Metadata{}, fmt.Errorf("fetching data: %w", err)
    }
    return Response{Data: data}, piko.Metadata{}, nil
}
```

### Wrapped errors

Piko uses `errors.As()` to unwrap errors, so typed errors work even when wrapped:

```go
// This still resolves to a 404 because errors.As unwraps the chain
return Response{}, piko.Metadata{}, fmt.Errorf("loading profile: %w", piko.NotFound("user", id))
```

## Default behaviour

When no custom error page exists for a given status code and scope, Piko returns a plain text response:

| Status | Response |
|--------|----------|
| 404 | `404 page not found` |
| 500 | `Internal Server Error` |
| Other codes | The matching HTTP status with a plain text message |

Custom error pages are optional. Your application works fine without them, and they let you provide a better user experience.

## Scaffold default

When you create a new Piko project with `piko new`, the scaffold includes a `pages/!404.pk`. It provides a styled 404 page that uses `piko.GetErrorContext()` to display relevant details. You can customise it or delete it as needed.

## See also

- [Errors reference](../reference/errors.md) for the typed error constructors that trigger error pages.
- [Routing rules reference](../reference/routing-rules.md) for how the router finds the right `!` page.
- [Metadata reference](../reference/metadata-fields.md) for HTTP status codes and redirects.
