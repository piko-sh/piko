---
title: How to mount custom HTTP handlers
description: Register raw http.Handler routes on the chi router for feeds, webhooks, OAuth callbacks, and other paths that do not fit the file-based model.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 80
---

# How to mount custom HTTP handlers

Some paths do not fit a PK page (which renders HTML), a server action (which expects typed JSON), or a partial (which expects a fragment). XML feeds, OAuth callbacks, third-party webhooks, well-known files, machine-to-machine endpoints, and reverse-proxy paths all want a raw `http.Handler`. Piko exposes its underlying chi router for exactly this case. See [about routing](../../explanation/about-routing.md#mounting-raw-http-handlers) for when this is the right tool.

## The router

`SSRServer.AppRouter` is a public exported `*chi.Mux` field. The router is [chi](https://github.com/go-chi/chi). The full chi API is available without any wrapper.

```go
import "piko.sh/piko"

func main() {
    ssr := piko.New(
        piko.WithWebsiteConfig(piko.WebsiteConfig{Name: "Example"}),
    )

    ssr.AppRouter.Get("/feed.xml", feedHandler)

    if err := ssr.Run(piko.RunModeProd); err != nil {
        log.Fatal(err)
    }
}
```

Register handlers before calling `ssr.Run`. Routes added after the server starts are subject to chi's standard goroutine-safety rules. They may not pick up middleware applied at startup. Treat the bootstrap phase as the registration window.

## Serve an XML feed

```go
func feedHandler(w http.ResponseWriter, r *http.Request) {
    items := database.GetItemsWithContext(r.Context())

    body, err := feed.Build(items)
    if err != nil {
        http.Error(w, "feed unavailable", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/xml; charset=utf-8")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
    w.Write(body)
}
```

`r.Context()` carries the per-request context the rest of the runtime uses. Pass it through to your data layer.

## Accept a webhook

Use `Post` for endpoints that receive payloads from external services:

```go
ssr.AppRouter.Post("/webhooks/stripe", func(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "read body", http.StatusBadRequest)
        return
    }
    defer r.Body.Close()

    if err := stripeClient.VerifySignature(r.Header.Get("Stripe-Signature"), body); err != nil {
        http.Error(w, "invalid signature", http.StatusUnauthorized)
        return
    }

    if err := stripeClient.HandleEvent(r.Context(), body); err != nil {
        http.Error(w, "process event", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
})
```

## Use URL parameters

chi's path parameters work directly:

```go
ssr.AppRouter.Get("/api/v1/properties/{id}", func(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    // ...
})
```

## Group routes with shared middleware

Use `Route` for path-prefixed groups, or `Group` plus `Use` for inline middleware:

```go
ssr.AppRouter.Route("/api/v1", func(r chi.Router) {
    r.Use(authMiddleware)
    r.Use(rateLimitMiddleware)

    r.Get("/me", meHandler)
    r.Get("/properties", listProperties)
    r.Post("/properties", createProperty)
})
```

The middleware chain composes on top of any global middleware Piko already applies (CSRF, logging, request ID).

## Mount a third-party handler

Use `Mount` to delegate a path prefix to another router or handler:

```go
ssr.AppRouter.Mount("/admin", adminApp.Router())
ssr.AppRouter.Mount("/debug", middleware.Profiler())
```

## Serve a static file

For one-off static responses (`robots.txt`, `humans.txt`, `.well-known/*`):

```go
ssr.AppRouter.Get("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain; charset=utf-8")
    w.Write(robotsBody)
})
```

For a directory of static assets, use `http.FileServer` mounted with `chi`:

```go
ssr.AppRouter.Mount("/.well-known/", http.StripPrefix("/.well-known/", http.FileServer(http.Dir("./well-known"))))
```

## See also

- [About routing](../../explanation/about-routing.md#mounting-raw-http-handlers) for when to use this and when not to.
- [How to apply page middleware](page-middleware.md) for middleware on file-based pages.
- [chi documentation](https://github.com/go-chi/chi) for the full router API.
