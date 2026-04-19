---
title: How to apply middleware to a page
description: Define a Middlewares function on a page to attach authentication, logging, or request-shaping middleware to the route.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 730
---

# How to apply middleware to a page

Pages attach standard `func(http.Handler) http.Handler` middleware to their route by exporting a `Middlewares` function from the Go script block. Piko wraps the page handler in the returned slice. For the routing primitives see [routing rules reference](../../reference/routing-rules.md).

<p align="center">
  <img src="../../diagrams/page-middleware-chain.svg"
       alt="A request enters at the top and travels inward through three nested middlewares before reaching the page Render function. The first entry in the Middlewares slice is the outermost wrapper and is the first to see the request and the last to see the response. The third entry is the innermost wrapper, closest to the handler. The response leaves in reverse order. A code panel shows the matching slice declaration: RequestLogger at index 0, RequireAuth at index 1, RequireAdmin at index 2."
       width="600"/>
</p>

## Define Middlewares on the page

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
    user := auth.UserFromContext(r.Context())
    return Response{User: user}, piko.Metadata{Title: "Admin Dashboard"}, nil
}

// Middlewares returns HTTP middleware applied to this route.
// Order in the slice is application order; the first entry wraps the rest.
func Middlewares() []func(http.Handler) http.Handler {
    return []func(http.Handler) http.Handler{
        auth.RequireAuthenticated,
        auth.RequireAdmin,
    }
}
</script>
```

## Understand the execution order

> **Note:** Piko wraps from the last slice entry inward, so index `[0]` is the outermost layer and the last entry is closest to `Render`. Treat the first entry as the layer that sees requests first and responses last (auth, logging belong here); treat the last entry as the layer closest to the page logic.

Piko wraps the handler from the **last** entry to the **first**, so the first entry in the slice runs first on the request and last on the response:

```go
func Middlewares() []func(http.Handler) http.Handler {
    return []func(http.Handler) http.Handler{
        firstMiddleware,   // outermost: first to see the request, last to see the response
        secondMiddleware,
        thirdMiddleware,   // innermost: last to see the request, first to see the response
    }
}
```

This matches the standard Go HTTP middleware idiom (`m1(m2(m3(handler)))`).

## Common middleware patterns

Authentication gate:

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

Request logging:

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

## See also

- [Routing rules reference](../../reference/routing-rules.md) for path matching and priority.
- [About routing](../../explanation/about-routing.md) for where middleware sits in the request lifecycle.
- [How to set a cache policy on a page](cache-policy.md) for the matching cache hook.
- [How to security](../security.md) for the wider authentication and CSRF picture.
