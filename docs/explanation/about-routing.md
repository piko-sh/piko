---
title: About routing
description: Why Piko derives routes from the filesystem, and the tradeoffs of that choice.
nav:
  sidebar:
    section: "explanation"
    subsection: "architecture"
    order: 40
---

# About routing

Piko derives routes from files in the `pages/` directory. A file at `pages/blog/{slug}.pk` serves `/blog/{slug}`, a file at `pages/{category}/index.pk` serves `/{category}`, and a file at `pages/docs/{slug}*.pk` serves `/docs/{slug}*` (a catch-all). No routing table, no registration code, no decorators.

## File-based routing is a trade

Code-based routing (Express-style `app.get("/blog/:slug", handler)`, or Gin's chi-tree registration) puts the URL and the handler together in source, where they can reference any shared state. File-based routing puts them apart. The URL lives in the filesystem, and the handler lives inside the file it names. Piko picks the second trade for three reasons.

The first reason is visibility. A project's route list should be visible without running the program. `ls pages/blog/` is faster than grepping a registration tree. For a docs site, marketing site, or product catalogue, this matters more than the flexibility of programmatic registration.

The second reason is hierarchy. The filesystem already encodes it. A page at `pages/admin/users/{id}.pk` is therefore under `/admin/users`, and therefore a child of the pages in `pages/admin/`. An existing mental model (directory structure) carries the route structure for free, without a separate nested-router abstraction.

The third reason is determinism. Tests, deploys, and incident response benefit from URL-to-file determinism. If a request to `/blog/hello-world` returns 500, the responsible file is `pages/blog/{slug}.pk`. There is no middleware chain or nested-router search to reverse-engineer.

## Precedence

<p align="center">
  <img src="../diagrams/routing-precedence.svg"
       alt="Three-tier precedence ladder. Tier 1 matches literal paths like pages/about.pk. Tier 2 matches dynamic segments like pages/blog/{slug}.pk. Tier 3 matches catch-alls like pages/api/{path}*.pk where the asterisk after the parameter is the variadic marker. A request falls through the tiers until one matches."
       width="600"/>
</p>

When two files could serve the same URL, Piko picks the more-specific one. Static segments win over parameters, more segments win over fewer, and exact file-name matches win over both dynamic and catch-all. The full [routing rules reference](../reference/routing-rules.md) enumerates every tiebreaker. The design goal is that reading the filenames makes the precedence predictable.

The predictability is load-bearing. It lets `pages/api/{path}*.pk` act as a catch-all backstop that real static files can shadow once they exist. It lets `pages/{category}/index.pk` render a category landing page without interfering with `pages/about/index.pk`. It lets `pages/!404.pk` hook the 404 flow at the root and `pages/admin/!404.pk` override it for the admin section.

## The error-page convention

The `!` prefix is the routing system's one deliberate departure from "files name URLs." `pages/!404.pk` does not serve `/` or `/404`. The router turns to it when it needs an error page. `pages/admin/!404.pk` overrides the root error page for admin routes.

## Rendering happens server-side by default

A route in Piko is a page, and a page is a `Render` function that returns typed data. The template substitutes the data and emits HTML. There is no hydration step, no client-side router, no SPA-style route-to-component dispatch in the browser.

Client-side navigation layers on top. `<piko:a>` links intercept clicks, fetch the next page's HTML, and swap the body without a full reload. The server still owns the URL-to-HTML contract, and the browser is a consumer, not a participant. This is deliberate. See [About SSR](about-ssr.md) for the reasoning.

## When to reach past the filesystem

The file-based system covers most needs. When it does not, the primary escape hatch is page-level middleware. A `Middlewares()` function in any PK file returns a chain of `func(http.Handler) http.Handler` wrappers. This handles auth, logging, header manipulation, redirects, rewrites, and anything else that would live in code-based routing. Middleware composes on top of the file-based route the page already declares. It does not let the application mount arbitrary unrelated handlers.

Some paths do not fit the file-based or action shape: XML feeds, OAuth callbacks, third-party webhooks, well-known files, or custom proxies. For those, Piko exposes the underlying router directly. `SSRServer.AppRouter` is a public `*chi.Mux` field. The router is [chi](https://github.com/go-chi/chi). Any handler registered on it sits alongside the file-based pages and shares the same middleware stack:

```go
ssr := piko.New(...)

ssr.AppRouter.Get("/feed.xml", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/xml; charset=utf-8")
    w.Write(buildFeed(r.Context()))
})

if err := ssr.Run(piko.RunModeProd); err != nil {
    log.Fatal(err)
}
```

The full chi API is available: `Get`, `Post`, `Put`, `Delete`, `Patch`, `Handle`, `Mount`, `Route`, `With`, route groups, URL parameters (`{name}`), and chi middleware. Register handlers before calling `ssr.Run`. Patterns that collide with file-based pages take whichever route the file-based system mounted first, since chi's first-match wins.

The escape hatch should stay an escape hatch. The Piko model favours typed routes the compiler understands: PK pages for HTML, actions for typed RPC, partials for fragments. A handler mounted on `AppRouter` is opaque to the manifest, the LSP, and the type checker. It is plain Go that returns plain bytes. Reach for it when typing the response shape would be ceremony for ceremony's sake. Examples include an RSS feed, a sitemap, a robots file, or a webhook endpoint that accepts an opaque blob. Reach for actions or pages for everything else.

See [How to mount custom HTTP handlers](../how-to/routing/custom-handlers.md) for worked patterns.

## See also

- [Routing rules reference](../reference/routing-rules.md) for the full precedence and mapping rules.
- [How to static routes](../how-to/routing/basic-routes.md), [dynamic routes](../how-to/routing/dynamic-routes.md), [catch-all routes](../how-to/routing/catch-all-routes.md).
- [How to apply middleware to a page](../how-to/routing/page-middleware.md), [how to set a cache policy on a page](../how-to/routing/cache-policy.md), and [how to serve from a URL prefix](../how-to/routing/base-path.md).
- [How to mount custom HTTP handlers](../how-to/routing/custom-handlers.md) for paths that do not fit a PK page or action.
- [About SSR](about-ssr.md) for the rendering model routes plug into.
