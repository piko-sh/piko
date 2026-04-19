---
title: config.json reference
description: Fields and sections Piko reads from the project config.json file.
nav:
  sidebar:
    section: "reference"
    subsection: "bootstrap"
    order: 60
---

# `config.json` reference

A Piko project keeps runtime configuration in `config.json` at the project root. The generator reads it at build time, and the server reads it at startup. Fields set programmatically through bootstrap options (for example `piko.WithPort`, `piko.WithWebsiteConfig`) override the values in `config.json`.

This page catalogues the top-level fields. The authoritative schema lives in the `bootstrap` package. When a field accepts a provider-specific sub-config, consult the provider's adapter for its schema.

## Top-level fields

```json
{
  "name": "My site",
  "port": 8080,
  "bindAddress": "0.0.0.0",
  "paths": { ... },
  "network": { ... },
  "build": { ... },
  "database": { ... },
  "security": { ... },
  "storage": { ... },
  "healthProbe": { ... },
  "logger": { ... },
  "otlp": { ... },
  "seo": { ... },
  "assets": { ... },
  "i18n": { ... }
}
```

Every field is optional. Piko applies defaults for anything unset.

### `name`

`string` -- human-readable project name used in banners and the dev widget.

### `port`

`int` -- HTTP port, default `8080`.

### `bindAddress`

`string` -- bind address, default `"0.0.0.0"`.

### `paths`

Override default project-relative paths:

| Sub-field | Purpose |
|---|---|
| `pages` | Location of `.pk` pages (default `pages`). |
| `components` | Location of `.pkc` components (default `components`). |
| `partials` | Location of shared partials (default `partials`). |
| `actions` | Location of action packages (default `actions`). |
| `content` | Location of collection content (default `content`). |
| `i18n` | Location of locale JSON files (default `i18n`). |
| `assets` | Location of static assets (default `assets`). |
| `dist` | Output directory for generated code (default `dist`). |

### `network`

| Sub-field | Purpose |
|---|---|
| `trustedProxies` | Array of CIDRs whose `X-Forwarded-*` headers Piko honours. |
| `cloudflareEnabled` | Trust `CF-Connecting-IP` when true. |

### `build`

| Sub-field | Purpose |
|---|---|
| `mode` | `"dev"`, `"dev-i"`, or `"prod"`. |
| `generatorProfiling` | Enables profiling output from the generator. |

### `database`

A map of named database registrations:

```json
"database": {
  "primary": {
    "driver": "postgres",
    "dsn": "postgres://..."
  }
}
```

Fields inside each registration depend on the driver. See the querier adapter for the specific driver.

### `security`

| Sub-field | Purpose |
|---|---|
| `csrfSecret` | Secret used to sign CSRF tokens (hex or base64). |
| `csrfTokenMaxAge` | Token lifetime in seconds. |
| `rateLimitEnabled` | Enables the built-in rate limiter. |
| `csp` | Inline CSP string (use bootstrap options for programmatic builders). |
| `crossOriginResourcePolicy` | `Cross-Origin-Resource-Policy` header value. |

### `storage`

Default storage provider settings:

| Sub-field | Purpose |
|---|---|
| `defaultProvider` | Name of the default provider (for example `"s3"`). |
| `presignBaseURL` | Base URL used when generating presigned download URLs. |
| `publicBaseURL` | Base URL used for public asset links. |

Adapters carry provider-specific credentials, not `config.json`.

### `healthProbe`

| Sub-field | Default | Purpose |
|---|---|---|
| `enabled` | `true` | Run the health server. |
| `port` | `"9090"` | Health server port. |
| `bindAddress` | `"127.0.0.1"` | Bind address for the health server. |
| `livePath` | `"/live"` | Liveness endpoint path. |
| `readyPath` | `"/ready"` | Readiness endpoint path. |
| `checkTimeoutSeconds` | `5` | Per-probe timeout. |

### `logger`

| Sub-field | Purpose |
|---|---|
| `level` | `"debug"`, `"info"`, `"warn"`, `"error"`. |
| `format` | `"json"` or `"text"`. |
| `pretty` | Pretty-print logs in dev. |

### `otlp`

OpenTelemetry exporter settings. Piko reads this section when `WithMonitoring` runs with OTLP configured.

| Sub-field | Purpose |
|---|---|
| `endpoint` | OTLP collector endpoint. |
| `headers` | Map of headers sent with OTLP requests. |
| `protocol` | `"grpc"` or `"http"`. |

### `seo`

| Sub-field | Purpose |
|---|---|
| `baseURL` | Canonical site URL used in sitemap and robots.txt. |
| `robotsTxt` | Raw robots.txt body (overrides the generated one). |
| `sitemapEnabled` | Generate `sitemap.xml`. |

### `assets`

| Sub-field | Purpose |
|---|---|
| `breakpoints` | Responsive breakpoint widths (for example `[320, 640, 960, 1280, 1920]`). |
| `densities` | Pixel-density multipliers (for example `[1, 2, 3]`). |
| `formats` | Output formats in preference order (for example `["avif", "webp", "jpeg"]`). |
| `quality` | Default output quality (0-100). |
| `lqipEnabled` | Generate a Low-Quality Image Placeholder (`LQIP`). |
| `lqipSize` | `LQIP` pixel size (kept small for inline data URIs). |

### `i18n`

| Sub-field | Purpose |
|---|---|
| `defaultLocale` | Fallback locale. |
| `strategy` | `"prefix"`, `"prefix_except_default"`, or `"domain"`. |
| `locales` | Array of supported locale codes. |

## Programmatic override

Bootstrap can supply or override any field via `piko.WithWebsiteConfig(&piko.WebsiteConfig{...})`. The option expects the same schema as `config.json`. When both are present, the bootstrap value wins.

## See also

- [Bootstrap options reference](bootstrap-options.md) for programmatic overrides and provider registration.
- [i18n API reference](i18n-api.md) for the `i18n` section in context.
- [Health API reference](health-api.md) for the `healthProbe` section.
