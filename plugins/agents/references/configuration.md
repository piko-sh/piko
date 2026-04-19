# Configuration

Use this guide when configuring a Piko application: bootstrap options, secret resolution, deployment, Docker/Kubernetes.

## Configuration model

Piko configures itself through Go code in `cmd/main/main.go`. No config files or env vars are read except `PIKO_LOG_LEVEL` (consulted once in `piko.New` to raise the bootstrap logger without rebuilding). Every other knob is a `With*` functional option to `piko.New`.

```go
ssr := piko.New(
    piko.WithPort(8080),
    piko.WithCSRFSecret(secretBytes),
    piko.WithLogLevel("info"),
    piko.WithRateLimit(piko.RateLimitConfig{Enabled: ptr(true)}),
)
```

This matches Nuxt, Next.js, Astro, SvelteKit, Remix etc. - code-based config, not YAML.

## The single environment-variable carve-out

`PIKO_LOG_LEVEL` raises the bootstrap logger before options apply. `piko.WithLogLevel(...)` always overrides it.

```bash
PIKO_LOG_LEVEL=debug ./my-app prod
```

Everything else - port, secrets, DB URL, OTLP endpoint - is your `func main`'s problem. Read from any source (env vars, secret managers, mounted files, koanf, viper) and pass to the relevant `With*` option.

## Common option groups

### Network

```go
piko.New(
    piko.WithPort(8080),
    piko.WithPublicDomain("example.com"),
    piko.WithForceHTTPS(true),
    piko.WithRequestTimeout(60*time.Second),
    piko.WithMaxConcurrentRequests(2000),
    piko.WithAutoNextPort(false),
)
```

### Logger

```go
piko.New(
    piko.WithLogLevel("info"), // trace, internal, debug, info, notice, warn, error
)
```

For wholesale replacement use `piko.WithLogger(logger_dto.Config{...})`. Switch the output adapter in `func main`:

```go
func main() {
    if mode == piko.RunModeProd {
        logger.AddJSONOutput()
    } else {
        logger.AddPrettyOutput()
    }
}
```

### Database

```go
piko.New(
    piko.WithDatabaseDriver("postgres"),
    piko.WithPostgresURL(os.Getenv("DATABASE_URL")),
    piko.WithPostgresMaxConns(10),
    piko.WithPostgresMinConns(2),
)
```

D1 (Cloudflare):

```go
piko.New(
    piko.WithDatabaseDriver("d1"),
    piko.WithD1APIToken(os.Getenv("D1_TOKEN")),
    piko.WithD1AccountID(os.Getenv("D1_ACCOUNT")),
    piko.WithD1DatabaseID(os.Getenv("D1_DATABASE")),
)
```

### Security

Pointer-typed scalar fields are wrapped with Go 1.26 `new(value)`:

```go
piko.New(
    piko.WithCSRFSecret([]byte(os.Getenv("CSRF_SECRET"))),
    piko.WithEncryptionKey(os.Getenv("ENCRYPTION_KEY")),
    piko.WithSecurityHeaders(piko.SecurityHeadersConfig{
        XFrameOptions: new("DENY"),
    }),
    piko.WithRateLimit(piko.RateLimitConfig{
        Enabled: new(true),
    }),
    piko.WithCookieSecurity(piko.CookieSecurityConfig{
        ForceHTTPOnly:      new(true),
        ForceSecureOnHTTPS: new(true),
        DefaultSameSite:    new("Lax"),
    }),
    piko.WithTrustedProxies("10.0.0.0/8"),
)
```

For CSP construct the policy in code:

```go
piko.New(
    piko.WithStrictCSP(),
    // or piko.WithPikoDefaultCSP(), piko.WithRelaxedCSP(), piko.WithAPICSP()
    // or piko.WithCSPString("default-src 'self'")
    // or piko.WithCSP(func(b *piko.CSPBuilder) { ... })
)
```

### OpenTelemetry

```go
piko.New(
    piko.WithOTLP(piko.OtlpConfig{
        Enabled:  new(true),
        Endpoint: new("otel-collector:4317"),
        Protocol: new("grpc"),
        Headers: map[string]string{
            "Authorization": "Bearer " + os.Getenv("OTLP_TOKEN"),
        },
    }),
)
```

Or per-field: `piko.WithOTLPEnabled(true)`, `piko.WithOTLPEndpoint("...")`, `piko.WithOTLPProtocol("grpc")`, `piko.WithOTLPHeaders(...)`, `piko.WithOTLPTraceSampleRate(0.1)`, `piko.WithOTLPInsecureTLS(false)`.

### Health probes

```go
piko.New(
    piko.WithHealthEnabled(true),
    piko.WithHealthProbePort(9090),
    piko.WithHealthBindAddress("0.0.0.0"), // use 0.0.0.0 in Docker/Kubernetes
    piko.WithHealthLivePath("/live"),
    piko.WithHealthReadyPath("/ready"),
    piko.WithHealthMetricsPath("/metrics"),
    piko.WithHealthMetricsEnabled(true),
    piko.WithHealthCheckTimeout(5*time.Second),
)
```

### TLS

```go
piko.New(
    piko.WithTLS(
        piko.WithTLSCertFile("/certs/server.pem"),
        piko.WithTLSKeyFile("/certs/server.key"),
        piko.WithTLSMinVersion("1.3"),
        piko.WithTLSHotReload(true),
    ),
)
```

### Storage and presigning

All scalar fields on `StoragePresignConfig` are pointer-typed:

```go
piko.New(
    piko.WithStoragePublicBaseURL("https://cdn.example.com"),
    piko.WithStoragePresign(piko.StoragePresignConfig{
        Secret:         new(os.Getenv("STORAGE_PRESIGN_SECRET")),
        DefaultExpiry:  new(15 * time.Minute),
        MaxExpiry:      new(time.Hour),
        DefaultMaxSize: new(int64(10 << 20)),
        MaxMaxSize:     new(int64(100 << 20)),
    }),
)
```

Or use the per-field setters: `piko.WithStoragePresignSecret(...)`, `piko.WithStoragePresignDefaultExpiry(...)`, `piko.WithStoragePresignMaxExpiry(...)`, `piko.WithStoragePresignDefaultMaxSize(...)`, `piko.WithStoragePresignMaxMaxSize(...)`.

### Source paths and serve paths

Override the defaults if your project layout differs from the scaffold:

```go
piko.New(
    piko.WithBaseDir("./src"),
    piko.WithPagesSourceDir("pages"),
    piko.WithComponentsSourceDir("components"),
    piko.WithI18nSourceDir("locales"),
    piko.WithAssetsSourceDir("assets"),
    piko.WithBaseServePath("/app"),
    piko.WithActionServePath("/_piko/actions"),
)
```

### Website metadata (theme, fonts, favicons, locales)

Passed via `WithWebsiteConfig`:

```go
piko.New(
    piko.WithWebsiteConfig(piko.WebsiteConfig{
        Name: "My Site",
        Theme: map[string]string{
            "colour-primary": "#6F47EB",
            "font-family":    "Inter, sans-serif",
        },
        Fonts: []piko.FontDefinition{
            {
                Type:    "google",
                URL:     "https://fonts.googleapis.com/css2?family=Inter:wght@400;500&display=swap",
                Instant: true,
            },
        },
        Favicons: []piko.FaviconDefinition{
            {Rel: "icon", Href: "/favicon.ico", Type: "image/x-icon"},
        },
        I18n: piko.I18nConfig{
            DefaultLocale: "en",
            Strategy:      "prefix_except_default",
            Locales:       []string{"en", "fr", "de"},
        },
    }),
)
```

If designers need to edit these without Go, unmarshal your own JSON file in `func main` and feed it into `WithWebsiteConfig`. Piko never auto-loads any file.

### i18n strategies

| Strategy | Example routes |
|----------|----------------|
| `prefix` | `/en/about`, `/fr/about` |
| `prefix_except_default` | `/about`, `/fr/about` |
| `query-only` | `/about` (locale from query param) |

Empty `Locales` slice disables i18n routing.

## Run modes

| Mode | Command | Use case |
|------|---------|----------|
| `prod` | `./app prod` | Production (AST caching, no hot-reload) |
| `dev` | `./app dev` | Development (file watching, hot-reload) |
| `dev-i` | `./app dev-i` | Interpreted mode (Yaegi interpreter) |

First CLI argument. Controls watcher and AST caching automatically. Use `piko.WithWatchMode(...)` only to override the default.

## WDK config kit (optional)

For a structured loader for your application's own config (not Piko's), import `wdk/config`. Independent utility - no Piko dependency. Swap for koanf, viper, envconfig, or your own loader.

```go
import "piko.sh/piko/wdk/config"

type AppConfig struct {
    Host     string `json:"host" env:"APP_HOST" flag:"host" default:"localhost"`
    Port     int    `json:"port" env:"APP_PORT" flag:"port" default:"8080" validate:"min=1,max=65535"`
}

cfg := &AppConfig{}
loadCtx, err := config.Load(ctx, cfg, config.LoaderOptions{
    FilePaths:  []string{"config.json", "config.yaml"},
    FlagPrefix: "app",
})
```

### Struct tags

| Tag | Purpose | Example |
|-----|---------|---------|
| `json` | JSON file field name | `json:"databaseUrl"` |
| `yaml` | YAML file field name | `yaml:"database_url"` |
| `env` | Environment variable | `env:"DATABASE_URL"` |
| `flag` | CLI flag name | `flag:"dbUrl"` |
| `default` | Default value | `default:"localhost"` |
| `validate` | Validation rules | `validate:"required,url"` |

### Secret placeholders

Resolvers ship for `env:`, `file:`, `base64:`, AWS Secrets Manager, GCP Secret Manager, Azure Key Vault, Kubernetes Secrets, HashiCorp Vault. Register the ones you need and resolve placeholders before passing to Piko options:

```go
vaultResolver, err := config_resolver_vault.NewResolver()
if err != nil {
    return err
}

cfg := AppConfig{
    DatabaseURL: "vault:secret/data/app#database_url",
}
if _, err := config.Load(ctx, &cfg, config.LoaderOptions{
    Resolvers: []config.Resolver{vaultResolver},
}); err != nil {
    return err
}

piko.New(piko.WithPostgresURL(cfg.DatabaseURL))
```

For process-wide registration use `config_resolver_vault.Register()` (or `config.RegisterResolver(vaultResolver)`) and pass `UseGlobalResolvers: true` in `LoaderOptions`. The dependency-free resolvers (`env:`, `file:`, `base64:`) are always registered automatically by `config.Load`.

### Lazy secrets in actions

For values you do not want in memory until used:

```go
type Config struct {
    APIKey piko.Secret[string]
}

handle, err := cfg.APIKey.Acquire(ctx)
defer handle.Close()
apiKey := handle.Value()
```

## Docker deployment

```dockerfile
FROM golang:1.26 as build
WORKDIR /app
COPY . .
RUN go build -o bin/generator cmd/generator/main.go && bin/generator all
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/app cmd/main/main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=build /app/bin/app /app/app
COPY --from=build /app/.piko /app/.piko
COPY --from=build /app/.out /app/.out
CMD ["/app/app", "prod"]
```

Pass deploy-time values (database URLs, secrets) through container env vars or mounted secrets and read them in `func main`.

## Kubernetes

```yaml
livenessProbe:
  httpGet:
    path: /live
    port: 9090
readinessProbe:
  httpGet:
    path: /ready
    port: 9090
```

Pair with `piko.WithHealthBindAddress("0.0.0.0")` so the probe is reachable from outside the pod.

## Graceful shutdown

Components are stopped in reverse registration order (LIFO). Each gets a proportional share of the 30-second timeout budget.

```go
ssr := piko.New()
ssr.RegisterLifecycle(databaseComponent)  // stopped last
ssr.RegisterLifecycle(cacheComponent)     // stopped second
ssr.RegisterLifecycle(workerComponent)    // stopped first
```

## LLM mistake checklist

- Putting configuration in YAML or JSON files (Piko reads neither)
- Reading `PIKO_PORT`, `PIKO_DATABASE_URL`, etc. (these env vars do not exist; only `PIKO_LOG_LEVEL` does)
- Forgetting that secrets must reach Piko via your own `func main` and pass through `With*` options
- Setting health probe `bindAddress` to `127.0.0.1` in Docker/Kubernetes (use `0.0.0.0`)
- Forgetting to run `go run ./cmd/generator/main.go all` after changing pages/partials
- Using `dev` mode in production (file watching overhead)
- Not configuring `WithTrustedProxies(...)` when rate limiting behind a reverse proxy

## Related

- `references/project-structure.md` - directory layout
- `references/wdk-security.md` - security headers and rate limiting
- `references/cli-reference.md` - CLI commands
