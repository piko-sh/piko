---
title: How to serve pages from a URL prefix
description: Set baseServePath so every page route mounts under a shared prefix.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 760
---

# How to serve pages from a URL prefix

Mount page routes under a path prefix when reverse-proxying behind a path-based router (`/app`, `/admin`, `/v2`). `BaseServePath` rewrites only the page routes. Action endpoints, partial endpoints, and asset URLs use their own dedicated prefixes (`ActionServePath`, `PartialServePath`, `LibServePath`). For the routing primitives see [routing rules reference](../../reference/routing-rules.md).

## Set the prefix in code

Pass the prefix through `piko.WithBaseServePath` in `func main`:

```go
ssr := piko.New(
    piko.WithBaseServePath("/app"),
)
```

If you need the prefix to come from an environment variable or another runtime source, read it yourself and pass the value to the option:

```go
prefix := os.Getenv("APP_BASE_PATH")
ssr := piko.New(
    piko.WithBaseServePath(prefix),
)
```

## Effect on registered routes

```text
pages/index.pk           ->  /app/
pages/about.pk           ->  /app/about
pages/blog/{slug}.pk     ->  /app/blog/{slug}
```

`BaseServePath` only applies to page routes. Configure action, partial, and asset URLs separately through `WithActionServePath`, `WithPartialServePath`, and `WithLibServePath`. They do not inherit the page prefix.

Locale prefixes sit in front of the base prefix. A page that resolves to `/about` under default locale becomes `/fr/app/about` under the `fr` locale. The locale segment comes first, then the base path, then the page route.

## Match the reverse proxy configuration

The prefix must match what the reverse proxy strips before forwarding. If the proxy preserves the prefix (forwards `/app/about` as-is), set `BaseServePath` to `/app`. If the proxy strips the prefix (forwards `/about`), leave `BaseServePath` unset.

A mismatch produces 404 responses on every page because the router looks for `/app/about` while the proxy forwards `/about`.

## See also

- [Routing rules reference](../../reference/routing-rules.md) for path matching.
- [About routing](../../explanation/about-routing.md) for the routing model that the prefix layers on.
- [How to enable i18n routing for a page](i18n-page-opt-in.md) for the locale-prefix interaction.
- [Bootstrap options reference](../../reference/bootstrap-options.md) for the full `With*` surface.
