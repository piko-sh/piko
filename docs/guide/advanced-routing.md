---
title: Advanced routing
description: Advanced routing patterns including i18n opt-in, middleware, caching, route priority tiebreakers, and base path configuration
nav:
  sidebar:
    section: "guide"
    subsection: "advanced"
    order: 710
---

# Advanced routing

This guide covers advanced routing patterns not found in the individual feature guides. You should already be familiar with [routing](/docs/guide/routing) and [i18n](/docs/guide/i18n).

For foundational topics covered in dedicated guides, see:

- **[Routing](/docs/guide/routing)** → File-based routing, dynamic and catch-all routes, basic route priority, redirects
- **[Internationalisation](/docs/guide/i18n)** → I18n configuration, routing strategies, locale detection, translations
- **[Metadata](/docs/guide/metadata)** → Redirects (`ClientRedirect`, `ServerRedirect`), status codes

## I18n page opt-in

Pages must opt into i18n routing by defining a `SupportedLocales` function. Without it, a page only gets a route for the default locale, even if i18n is configured globally.

See [i18n](/docs/guide/i18n) for configuration and routing strategies (including `prefix`, `prefix_except_default`, and `query-only`).

```go
// SupportedLocales enables i18n routing for this page.
// Routes will be generated for each locale listed here.
func SupportedLocales() []string {
    return []string{"en", "fr", "de"}
}
```

For a page at `pages/about.pk` with `prefix_except_default` strategy, this generates:
- English (default): `/about`
- French: `/fr/about`
- German: `/de/about`

## Page middleware

Apply HTTP middleware to specific pages for authentication, logging, or request transformation.

### Defining middleware

Add a `Middlewares` function to your page's script block:

**File**: `pages/admin/dashboard.pk`

```piko
<template>
  <piko:partial is="layout" :server.page_title="'Admin dashboard'">
    <h1>Admin dashboard</h1>
    <p>Welcome, {{ state.User.Name }}</p>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "net/http"

    "piko.sh/piko"
    "myapp/pkg/auth"
    layout "myapp/partials/layout.pk"
)

type Response struct {
    User auth.User
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // User is guaranteed to be authenticated and admin
    user := auth.UserFromContext(r.Context())

    return Response{User: user}, piko.Metadata{
        Title: "Admin Dashboard",
    }, nil
}

// Middlewares returns HTTP middleware applied to this route.
// Executed in reverse order: RequireAuthenticated runs first, then RequireAdmin.
func Middlewares() []func(http.Handler) http.Handler {
    return []func(http.Handler) http.Handler{
        auth.RequireAdmin,
        auth.RequireAuthenticated,
    }
}
</script>
```

### Middleware execution order

Middlewares are applied in **reverse order**. The last middleware in the slice wraps the innermost handler:

```go
func Middlewares() []func(http.Handler) http.Handler {
    return []func(http.Handler) http.Handler{
        thirdMiddleware,   // Runs third (innermost)
        secondMiddleware,  // Runs second
        firstMiddleware,   // Runs first (outermost)
    }
}
```

This follows the standard Go HTTP middleware convention where the first middleware in the chain handles the request first and the response last.

### Common middleware patterns

**Authentication**:
```go
func RequireAuthenticated(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, err := auth.GetUserFromSession(r)
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusFound)
            return
        }
        ctx := auth.WithUser(r.Context(), user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Logging**:
```go
func RequestLogger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        log.Info("Request completed",
            logger.String("path", r.URL.Path),
            logger.Duration("duration", time.Since(start)),
        )
    })
}
```

## Cache policies

Define caching behaviour for pages using the `CachePolicy` function.

### CachePolicy structure

```go
type CachePolicy struct {
    // Key is an optional identifier combined with the URL to create the cache key.
    // Use for per-user caching, A/B testing, or localisation.
    Key string

    // MaxAgeSeconds sets the Cache-Control max-age value and server-side cache TTL.
    MaxAgeSeconds int

    // Enabled is the master switch for server-side AST caching.
    Enabled bool

    // OnRender indicates whether the AST should be generated once at build time
    // and treated as static content with a long cache TTL.
    OnRender bool

    // Static enables full-page HTML caching (highest performance).
    // Only suitable for pages that do not vary per-user.
    Static bool

    // MustRevalidate forces revalidation with the origin server.
    MustRevalidate bool

    // NoStore prevents the response from being cached.
    NoStore bool
}
```

### Example: cacheable static page

**File**: `pages/about.pk`

```piko
<script type="application/x-go">
package main

import "piko.sh/piko"

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{
        Title: "About Us",
    }, nil
}

// CachePolicy enables aggressive caching for this static page.
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled:       true,
        MaxAgeSeconds: 3600,  // 1 hour
        Static:        true,  // Cache the full HTML output
    }
}
</script>
```

### Example: user-specific caching

For pages that vary by user, use a custom cache key:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    user := auth.UserFromContext(r.Context())

    return Response{User: user}, piko.Metadata{
        Title:    "My Profile",
        CacheKey: fmt.Sprintf("user:%s", user.ID),  // Per-user cache key
    }, nil
}

func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled:       true,
        MaxAgeSeconds: 300,  // 5 minutes
        Static:        false, // Don't cache full HTML (varies by user)
    }
}
```

### Example: no caching

For sensitive or frequently-changing pages:

```go
func CachePolicy() piko.CachePolicy {
    return piko.CachePolicy{
        Enabled: false,
        NoStore: true,
    }
}
```

## Route priority tiebreakers

For the basic priority order (static > dynamic > catch-all), see [routing](/docs/guide/routing). When routes have the same type, Piko applies additional tiebreakers:

1. **Path depth.** More specific paths (more segments) take priority.
2. **Static segments.** Routes with more static segments win.
3. **Alphabetical.** Tie-breaker for identical specificity.

### Nested dynamic routes

```text
pages/
├── {category}/
│   ├── index.pk           → /:category (dynamic)
│   └── {id}.pk            → /:category/:id (2 dynamic)
└── docs/
    └── {id}.pk            → /docs/:id (1 static + 1 dynamic)
```

**Priority**: `/docs/:id` wins over `/:category/:id` because it has more static segments (1 vs 0).

### Checking route priority

At build time, Piko logs the order in which routes are registered. Check your build output to verify priority:

```text
INFO  Registering page route pattern=/docs/:id
INFO  Registering page route pattern=/docs
INFO  Registering page route pattern=/:category/:id
INFO  Registering page route pattern=/:category
```

## Base path configuration

Configure a URL prefix for all routes using `baseServePath`:

**Environment variable**:
```bash
export PIKO_BASE_SERVE_PATH="/app"
```

**Or in server config**:
```go
cfg := piko.DefaultServerConfig()
cfg.Paths.BaseServePath = "/app"
```

**Result**:
- `pages/index.pk` → `/app/`
- `pages/about.pk` → `/app/about`
- `pages/blog/{slug}.pk` → `/app/blog/:slug`
