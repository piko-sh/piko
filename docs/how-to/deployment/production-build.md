---
title: How to build for production
description: Build the generator and binary, configure production mode, and run the compiled application.
nav:
  sidebar:
    section: "how-to"
    subsection: "deployment"
    order: 10
---

# How to build for production

This guide walks through building a Piko application for production. Steps cover compiling assets, building the server binary, choosing a run mode, and handing off to a process manager. See the [CLI reference](../../reference/cli.md) for every `piko` subcommand and the [bootstrap options reference](../../reference/bootstrap-options.md) for every `With*` option.

## Project layout

A scaffolded Piko project has two entry points under `cmd/`:

```text
my-app/
├── cmd/
│   ├── generator/
│   │   └── main.go    # Asset/manifest generator
│   └── main/
│       └── main.go    # Server entry point
├── actions/
├── components/
├── pages/
└── partials/
```

`cmd/generator/main.go` produces the template and asset manifests. `cmd/main/main.go` is the HTTP server.

## Step 1: Generate assets

Build and run the generator first. The generator compiles every `.pk` template, processes assets, and emits a manifest the server reads at startup.

```bash
go build -o bin/generator cmd/generator/main.go
bin/generator all
```

`all` invokes `piko.GenerateModeAll`, which covers template compilation, SQL-querier generation, and asset processing. Other modes are available if you need finer control:

| Command | Mode constant | Effect |
|---|---|---|
| `bin/generator manifest` | `piko.GenerateModeManifest` | Compile templates and emit the manifest only. |
| `bin/generator sql` | `piko.GenerateModeSQL` | Run the querier generator only. |
| `bin/generator assets` | `piko.GenerateModeAssets` | Asset processing only. |
| `bin/generator all` | `piko.GenerateModeAll` | Everything. |

## Step 2: Build the server binary

Build a statically linked, stripped binary:

```bash
CGO_ENABLED=0 go build \
  -ldflags="-s -w" \
  -o bin/app \
  cmd/main/main.go
```

- `CGO_ENABLED=0` produces a static binary with no C dependencies. Use this unless you are linking a CGO-only SQLite driver.
- `-ldflags="-s -w"` strips debug symbols and DWARF information for a smaller binary.

## Step 3: Configure the application

Configure the framework in `func main` with `With*` options. Branch on a build flag, environment variable, or the run-mode argument if you need different settings between environments.

```go
ssr := piko.New(
    piko.WithPublicDomain("yourdomain.com"),
    piko.WithForceHTTPS(true),
    piko.WithAutoNextPort(false),
)
```

The first command-line argument to `ssr.Run` carries the run mode (`dev`, `prod`) and controls watch mode automatically. Use `piko.WithWatchMode(...)` if you need to override that default.

## Step 4: Run the binary

Piko applications accept the run mode as the first command-line argument:

```bash
./bin/app prod
```

| Run mode | Constant | Behaviour |
|---|---|---|
| `dev` | `piko.RunModeDev` | Hot reload, file watching, verbose dev output. |
| `dev-i` | `piko.RunModeDevInterpreted` | Yaegi interpreter: no recompilation, slower runtime. |
| `prod` | `piko.RunModeProd` | Compiled templates, AST caching, production HTTP stack. |

The scaffolded `cmd/main/main.go` reads the argument and passes it to `ssr.Run`:

```go
package main

import (
    "os"

    "piko.sh/piko"
    "piko.sh/piko/wdk/logger"
)

func main() {
    logger.AddPrettyOutput()

    command := piko.RunModeDev
    if len(os.Args) > 1 {
        command = os.Args[1]
    }

    ssr := piko.New()
    if err := ssr.Run(command); err != nil {
        panic(err)
    }
}
```

The example above is the minimal shape. The wizard-generated scaffold goes one step further and constructs the SSR via `internal.NewServer(command)`, a thin project-local wrapper under `internal/piko.go` that holds the shared option set. Both `cmd/main` and `cmd/generator` configure the framework the same way. If you scaffold with `piko new`, edit `internal/piko.go` to adjust framework options instead of scattering `piko.With*` calls across each binary.

Switch `AddPrettyOutput()` to `AddJSONOutput()` for structured logs in production.

## Step 5: Expose the health probe

Piko runs two HTTP servers. One serves the application on port 8080 (default), the other serves the health probe on port 9090 (default). The default bind address for the health probe is `127.0.0.1`, which keeps it off the public network. Expose it to the orchestrator by binding to all interfaces:

```go
ssr := piko.New(
    piko.WithHealthEnabled(true),
    piko.WithHealthProbePort(9090),
    piko.WithHealthBindAddress("0.0.0.0"),
    piko.WithHealthLivePath("/live"),
    piko.WithHealthReadyPath("/ready"),
    piko.WithHealthMetricsPath("/metrics"),
    piko.WithHealthMetricsEnabled(true),
)
```

## Step 6: Wire up a process manager

Piko does not supervise itself. Use a process manager that restarts the binary on crash.

### systemd

```ini
[Unit]
Description=MyApp (Piko)
After=network.target

[Service]
ExecStart=/opt/myapp/bin/app prod
WorkingDirectory=/opt/myapp
Restart=on-failure
User=myapp
EnvironmentFile=/etc/myapp/env

[Install]
WantedBy=multi-user.target
```

### Docker

A multi-stage Dockerfile:

```dockerfile
# Build stage
FROM golang:1.26 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build -o bin/generator cmd/generator/main.go
RUN bin/generator all
RUN go build -ldflags="-s -w" -o bin/app cmd/main/main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /app

COPY --from=build /app/bin/app /app/app
COPY --from=build /app/.piko /app/.piko
COPY --from=build /app/.out /app/.out

CMD ["/app/app", "prod"]
```

The `distroless/static:nonroot` base image keeps the container small and runs as a non-root user by default.

### Kubernetes

A minimal deployment manifest:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
        - name: myapp
          image: registry.example.com/myapp:latest
          ports:
            - containerPort: 8080
              name: http
            - containerPort: 9090
              name: health
          livenessProbe:
            httpGet:
              path: /live
              port: health
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /ready
              port: health
            initialDelaySeconds: 5
            periodSeconds: 10
          envFrom:
            - secretRef:
                name: myapp-secrets
```

## Production checklist

Before the first deploy:

- HTTPS: terminate TLS directly with `piko.WithTLS(...)` (see [TLS how-to](tls.md)) or front with a reverse proxy.
- Security headers: on by default, tunable through `piko.WithSecurityHeaders(...)`.
- CSRF: on by default for actions.
- Secrets: loaded into your `func main` from any source (env vars, secret manager, mounted files) and passed to the relevant `With*` option (see [secrets how-to](../secrets.md) and [secrets resolvers](secrets-resolvers.md)).
- Rate limiting: configure `piko.WithRateLimit(...)` if exposed to the public internet.
- Assets: run the generator before building the binary.
- Logging: swap to JSON in production (`logger.AddJSONOutput()`).
- Health probe: accessible to the orchestrator with `piko.WithHealthBindAddress("0.0.0.0")`.
- Metrics: `/metrics` on the health probe server when `piko.WithHealthMetricsEnabled(true)`.
- Process manager: configured to restart on failure.

## Scaling

Piko applications are stateless when sessions, file uploads, caches, and rate-limit counters all live in external stores. For horizontal scaling:

- Sessions: back by Redis, Valkey, or the database via the cache service.
- Uploads: use `storage_provider_s3`, `storage_provider_gcs`, or another external provider.
- Cache: use `cache_provider_redis`, `cache_provider_valkey`, or multilevel.
- Rate limiter: use a Redis backend so counters survive instance restarts and replicas share the same state.

For vertical scaling inside one instance, tune `GOMAXPROCS`, database connection pools (`piko.WithPostgresMaxConns`, `piko.WithPostgresMinConns`), and the HTTP request timeout (`piko.WithRequestTimeout`).

## See also

- [Configuration philosophy](../../explanation/about-configuration.md).
- [How to TLS](tls.md).
- [How to monitoring](monitoring.md).
- [How to troubleshooting deployment](troubleshooting.md).
- [CLI reference](../../reference/cli.md).
- [Bootstrap options reference](../../reference/bootstrap-options.md).
