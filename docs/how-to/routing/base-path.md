---
title: How to serve from a URL prefix
description: Set baseServePath so every page, asset, and action route mounts under a shared prefix.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 760
---

# How to serve from a URL prefix

Mount the whole application under a path prefix when reverse-proxying behind a path-based router (`/app`, `/admin`, `/v2`). The framework rewrites every page, asset, and action route to include the prefix. For the routing primitives see [routing rules reference](../../reference/routing-rules.md).

## Set the prefix via environment variable

```bash
export PIKO_BASE_SERVE_PATH="/app"
```

The framework reads the variable at startup. Pages inside the binary stay unchanged.

## Or set it in code

```go
cfg := piko.DefaultServerConfig()
cfg.Paths.BaseServePath = "/app"
```

Use this form when configuration lives outside environment variables (for example, when deriving the prefix from a tenant identifier at startup).

## Effect on registered routes

```text
pages/index.pk           ->  /app/
pages/about.pk           ->  /app/about
pages/blog/{slug}.pk     ->  /app/blog/:slug
```

Action endpoints, asset URLs, and locale prefixes all compose with `BaseServePath`. An i18n route like `/fr/about` becomes `/app/fr/about`.

## Match the reverse proxy configuration

The prefix must match what the reverse proxy strips before forwarding. If the proxy preserves the prefix (forwards `/app/about` as-is), set `BaseServePath` to `/app`. If the proxy strips the prefix (forwards `/about`), leave `BaseServePath` unset.

A mismatch produces 404 responses on every page because the router looks for `/app/about` while the proxy forwards `/about`.

## See also

- [Routing rules reference](../../reference/routing-rules.md) for path matching.
- [How to enable i18n routing for a page](i18n-page-opt-in.md) for the locale-prefix interaction.
- [Bootstrap options reference](../../reference/bootstrap-options.md) for other server-config knobs.
