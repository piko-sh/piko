---
title: About configuration
description: Why Piko configures the framework through Go code, the build-time config.json that the manifest builder consumes, and where deploy-time secrets fit in.
nav:
  sidebar:
    section: "explanation"
    subsection: "operations"
    order: 40
---

# Configuration philosophy

Piko configures itself almost entirely through Go code in your `func main`.
At runtime the framework reads no application configuration files. The
build-time pipeline is the one exception. When generating the manifest, the
generator looks for a `config.json` in the project root and folds its
`WebsiteConfig` (theme, fonts, locales, i18n strategy) into the embedded
manifest. Passing `WithWebsiteConfig` in `piko.New` overrides that file
entirely. If neither is present, the framework starts with an empty
website config.

The framework also consults a small set of environment variables when
their associated options are not set in code. `piko.New` reads
`PIKO_LOG_LEVEL` once so you can raise the bootstrap logger to `debug`
without recompiling. `PIKO_PORT`, `PIKO_I18N_SOURCE_DIR`,
`PIKO_ACTION_SERVE_PATH`, and others provide defaults for
the matching `With*` options. An explicit option in your `main` always
wins over the environment.

This page explains why and how to configure the framework. It also covers
where to put deploy-time secrets. Finally, it points to the utility Piko
ships for file-based configuration in *your application* (which Piko
itself does not consume).

## Configure the framework

Every framework knob is a `With*` functional option passed to `piko.New`.
The same options work in dev and prod, and deploy-time values come from your
own `func main`.

```go
ssr := piko.New(
    piko.WithPort(8080),
    piko.WithAutoNextPort(true),
    piko.WithCSRFSecret(secretKey),
    piko.WithLogLevel("info"),
    piko.WithRateLimit(piko.RateLimitConfig{
        Enabled:        new(true),
        HeadersEnabled: new(true),
    }),
    piko.WithCSSReset(piko.WithCSSResetComplete()),
    piko.WithDevWidget(),
)
```

Group `With*` options exist for tightly coupled fields:

- `WithSecurityHeaders(piko.SecurityHeadersConfig{...})`
- `WithCookieSecurity(piko.CookieSecurityConfig{...})`
- `WithRateLimit(piko.RateLimitConfig{...})`
- `WithSandbox(piko.SandboxConfig{...})`
- `WithReporting(piko.ReportingConfig{...})`
- `WithCaptcha(piko.CaptchaOptions{...})`
- `WithAWSKMS(piko.AWSKMSConfig{...})`
- `WithGCPKMS(piko.GCPKMSConfig{...})`
- `WithStoragePresign(piko.StoragePresignConfig{...})`
- `WithOTLP(piko.OtlpConfig{...})`
- `WithLogger(logger_dto.Config{...})`

Per-field options exist for the rest. See [bootstrap
options](../reference/bootstrap-options.md) for the full surface.

## Why code, not files

Piko is a framework you compile a binary against. The compiler bakes routes,
generators, asset pipelines, build modes, and all framework wiring in at
`go build` time. Treating those values as runtime YAML created a second mental model
that competed with functional options for the same surface and silently
drifted from it. Removing the file-loaded `ServerConfig` collapses
configuration to one shape, the one the type system already enforces.

This matches the SSR-framework convention: Nuxt, Next.js, Astro, SvelteKit,
Remix, Vite, and Tailwind all configure themselves through executable
TypeScript or JavaScript modules, not through YAML. Next.js explicitly
[deprecated and
removed](https://nextjs.org/docs/app/guides/upgrading/version-16) its file
based runtime config in favour of env-var access from application code.

## The `PIKO_LOG_LEVEL` exception

`piko.New` reads `PIKO_LOG_LEVEL` once, before any options apply, so you
can raise log verbosity in production without rebuilding the binary. An
explicit `WithLogLevel(...)` option in your `main` overrides it.

```bash
PIKO_LOG_LEVEL=debug ./my-app prod
```

Other `PIKO_*` variables (`PIKO_PORT`, `PIKO_I18N_SOURCE_DIR`,
`PIKO_ACTION_SERVE_PATH`, and similar) provide defaults for the matching
`With*` options when those options are not set in code. They are
deliberately limited. Anything secret, environment-specific, or composed
from multiple sources is your `func main`'s problem.

## Deploy-time values: bring your own loader

Piko has no opinion on how you load secrets and per-environment values. Pass
them to the relevant `With*` option from any source you trust.

```go
ssr := piko.New(
    piko.WithCSRFSecret([]byte(os.Getenv("CSRF_SECRET"))),
    piko.WithPostgresURL(os.Getenv("DATABASE_URL")),
    piko.WithStoragePresignSecret(os.Getenv("STORAGE_PRESIGN_SECRET")),
)
```

For more structure, use a small loader of your choice
([koanf](https://github.com/knadh/koanf), [viper](https://github.com/spf13/viper),
[envconfig](https://github.com/kelseyhightower/envconfig), or your own).
Piko ships a generic loader and resolver kit at
[`piko.sh/piko/wdk/config`](../reference/wdk-config.md) that you may use:
it has no dependency on Piko's internals and you can keep it or swap it.

## Secret resolvers

To expand placeholder strings such as `"aws-secret:my/key"` in your own
configuration, use the `wdk/config` package. It provides resolvers for AWS
Secrets Manager, GCP Secret Manager, Azure Key Vault, Kubernetes Secrets,
and HashiCorp Vault. They satisfy the generic `Resolver`
interface. You can register them with the global registry or compose them
into a custom `Loader`. See [secrets and
resolvers](../how-to/deployment/secrets-resolvers.md).

## Theme, fonts, favicons, locales

Pass the website-level metadata (theme colours, favicons, fonts, supported
locales, i18n strategy) explicitly via `WithWebsiteConfig`:

```go
ssr := piko.New(
    piko.WithWebsiteConfig(piko.WebsiteConfig{
        Name: "My Site",
        I18n: piko.I18nConfig{
            DefaultLocale: "en",
            Locales:       []string{"en", "fr", "de"},
        },
        // theme, fonts, favicons, ...
    }),
)
```

If you prefer keeping these as JSON, drop a `config.json` next to the
project root with the same shape as `WebsiteConfig`. The build-time
manifest generator reads it and embeds the result. The runtime then
serves the embedded data with no further file access. Designers can edit
`config.json` and rebuild without touching Go code. Setting
`WithWebsiteConfig` in `func main` overrides the file entirely. If you
want both editing paths, unmarshal your own JSON in `main` and pass the
result to `WithWebsiteConfig`.
