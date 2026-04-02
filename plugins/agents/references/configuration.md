# Configuration

Use this guide when setting up piko.yaml, config.json, environment variables, secret resolution, or deployment configuration.

## Configuration files

Piko uses two configuration files:

| File | Purpose |
|------|---------|
| `config.json` | Website config: theme variables, fonts, favicons, i18n settings |
| `piko.yaml` | Server config: network, database, security, logging, observability |

## config.json

Placed in the project root. Defines user-facing settings.

```json
{
  "name": "My Website",
  "theme": {
    "colour-primary": "#6F47EB",
    "font-family": "Inter, sans-serif"
  },
  "fonts": [
    {
      "type": "google",
      "url": "https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap",
      "instant": true
    }
  ],
  "favicons": [
    {"rel": "icon", "href": "/favicon.ico", "type": "image/x-icon"}
  ],
  "i18n": {
    "defaultLocale": "en",
    "strategy": "prefix_except_default",
    "locales": ["en", "fr", "de"]
  }
}
```

### Theme variables

All `theme` keys become CSS custom properties prefixed with `--g-`:

```css
:root {
  --g-colour-primary: #6F47EB;
  --g-font-family: Inter, sans-serif;
}
```

### i18n strategies

| Strategy | Example routes |
|----------|----------------|
| `prefix` | `/en/about`, `/fr/about` |
| `prefix_except_default` | `/about`, `/fr/about` |
| `query-only` | `/about` (locale from query param) |
| `disabled` | Single locale only |

## piko.yaml

Server configuration with multi-source precedence:

1. Programmatic defaults
2. Struct tag defaults
3. Configuration files (`piko.yaml`, `piko-{env}.yaml`, `piko.local.yaml`)
4. `.env` files
5. Environment variables (`PIKO_` prefix)
6. Command-line flags
7. Secret resolvers
8. Programmatic overrides (`piko.WithXxx(...)`) - highest priority

### Network

```yaml
network:
  port: "8080"
  publicDomain: "example.com"
  forceHttps: true
  requestTimeoutSeconds: 60
```

### Database

```yaml
database:
  driver: postgres    # default: sqlite; also supports postgres, d1
  postgres:
    url: "${PIKO_DATABASE_POSTGRES_URL}"
    maxConns: 10
    minConns: 2
```

### Security

```yaml
security:
  headers:
    enabled: true
    xFrameOptions: DENY
    contentSecurityPolicy: "default-src 'self'"
  rateLimit:
    enabled: true
    storage: memory
    global:
      requestsPerMinute: 1000
  cookies:
    forceHttpOnly: true
    forceSecureOnHttps: true
    defaultSameSite: Lax
  cryptoProvider: local_aes_gcm
  encryptionKey: "${PIKO_ENCRYPTION_KEY}"
```

### Logging

```yaml
logger:
  level: info       # trace, internal, debug, info, notice, warn, error
  addSource: true
  outputs:
    - type: stdout
      format: json       # default: pretty
      level: info
  notifications:
    - name: slack-alerts
      type: slack
      minLevel: error
      slack:
        webhookUrl: "${SLACK_WEBHOOK_URL}"
```

Supported notification targets: Slack, Discord, PagerDuty, Microsoft Teams, Google Chat, Ntfy, email, generic webhooks.

### OpenTelemetry

```yaml
otlp:
  enabled: true
  protocol: http          # default: http (also supports grpc)
  endpoint: localhost:4317
```

### Health probes

```yaml
healthProbe:
  enabled: true
  port: "9090"
  bindAddress: "127.0.0.1"   # use 0.0.0.0 for Docker/Kubernetes
  livePath: "/live"
  readyPath: "/ready"
```

### Build optimisation

> **Note**: Compiler settings are currently hardcoded constants and cannot be configured via YAML. The section below shows the internal defaults for reference.

```yaml
compiler:
  logLevel: warn
  verifyGeneratedCode: false
  cssTreeShaking: true
  cssTreeShakingSafelist:
    - open
    - active
```

## Run modes

| Mode | Command | Use case |
|------|---------|----------|
| `prod` | `./app prod` | Production (AST caching, no hot-reload) |
| `dev` | `./app dev` | Development (file watching, hot-reload) |
| `dev-i` | `./app dev-i` | Interpreted mode (Yaegi interpreter) |

## WDK config package

For custom configuration structs with multi-source loading:

```go
import "piko.sh/piko/wdk/config"

type AppConfig struct {
    Host     string `json:"host" env:"APP_HOST" flag:"host" default:"localhost"`
    Port     int    `json:"port" env:"APP_PORT" flag:"port" default:"8080" validate:"min=1,max=65535"`
}

cfg := &AppConfig{}
loadCtx, err := config.Load(ctx, cfg, config.LoaderOptions{
    FilePaths: []string{"config.json", "config.yaml"},
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

### Lazy secrets

For values that should not be in memory until needed:

```go
type Config struct {
    APIKey config_domain.Secret[string] `env:"API_KEY"`
}

handle, err := cfg.APIKey.Acquire(ctx)
defer handle.Release()
apiKey := handle.Value()
```

## Docker deployment

```dockerfile
FROM golang:1.24 as build
WORKDIR /app
COPY . .
RUN go build -o bin/generator cmd/generator/main.go && bin/generator all
RUN go build -ldflags="-s -w" -o bin/app cmd/main/main.go

FROM golang:1.24 as runtime
COPY --from=build /app/bin/app /app/
COPY --from=build /app/piko.yaml /app/
COPY --from=build /app/config.json /app/
COPY --from=build /app/.piko /app/.piko
COPY --from=build /app/.out /app/.out
CMD ["/app/app", "prod"]
```

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

## Graceful shutdown

Components are stopped in reverse registration order (LIFO). Each gets a proportional share of the 30-second timeout budget.

```go
ssr := piko.New()
ssr.RegisterLifecycle(databaseComponent)  // stopped last
ssr.RegisterLifecycle(cacheComponent)     // stopped second
ssr.RegisterLifecycle(workerComponent)    // stopped first
```

## LLM mistake checklist

- Using `config.json` for server settings (use `piko.yaml`)
- Using `piko.yaml` for theme variables (use `config.json`)
- Forgetting `PIKO_` prefix for environment variables
- Setting health probe `bindAddress` to `127.0.0.1` in Docker/Kubernetes (use `0.0.0.0`)
- Forgetting to run `go run ./cmd/generator/main.go all` after changing pages/partials
- Using `dev` mode in production (file watching overhead)
- Not configuring `trustedProxies` when rate limiting behind a reverse proxy

## Related

- `references/project-structure.md` - directory layout
- `references/wdk-security.md` - security headers and rate limiting
- `references/cli-reference.md` - CLI commands
