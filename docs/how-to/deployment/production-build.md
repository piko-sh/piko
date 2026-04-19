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

This guide walks through building a Piko application for production. Steps cover compiling assets, building the server binary, choosing a run mode, and handing off to a process manager. See the [CLI reference](../../reference/cli.md) for every `piko` subcommand and the [`config.json` reference](../../reference/config-json-schema.md) for every configuration field.

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
├── partials/
├── piko.yaml
└── config.json
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

## Step 3: Configure the environment

Override development defaults in `piko-prod.yaml`:

```yaml
network:
  publicDomain: "yourdomain.com"
  forceHttps: true
  autoNextPort: false

build:
  watchMode: false
```

Use `piko.local.yaml` for per-machine overrides that you keep out of version control.

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

Switch `AddPrettyOutput()` to `AddJSONOutput()` for structured logs in production.

## Step 5: Expose the health probe

Piko runs two HTTP servers. One serves the application on port 8080 (default), the other serves the health probe on port 9090 (default). The default bind address for the health probe is `127.0.0.1`, which keeps it off the public network. Expose it to the orchestrator by binding to all interfaces:

```yaml
healthProbe:
  enabled: true
  port: "9090"
  bindAddress: "0.0.0.0"
  livePath: "/live"
  readyPath: "/ready"
  metricsPath: "/metrics"
  metricsEnabled: true
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

ENV PIKO_ENV="prod"

COPY --from=build /app/bin/app /app/app
COPY --from=build /app/piko.yaml /app
COPY --from=build /app/piko-prod.yaml /app
COPY --from=build /app/config.json /app
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
- Security headers: on by default, tunable under `security.headers` in `piko.yaml`.
- CSRF: on by default for actions.
- Secrets: loaded via environment variables, `.env`, or a secret resolver (see [environment config how-to](environment-config.md)).
- Rate limiting: configure `security.rateLimit` if exposed to the public internet.
- Assets: run the generator before building the binary.
- Logging: swap to JSON in production (`logger.AddJSONOutput()`).
- Health probe: accessible to the orchestrator with `bindAddress: "0.0.0.0"`.
- Metrics: `/metrics` on the health probe server when `metricsEnabled: true`.
- Process manager: configured to restart on failure.

## Scaling

Piko applications are stateless when sessions, file uploads, caches, and rate-limit counters all live in external stores. For horizontal scaling:

- Sessions: back by Redis, Valkey, or the database via the cache service.
- Uploads: use `storage_provider_s3`, `storage_provider_gcs`, or another external provider.
- Cache: use `cache_provider_redis`, `cache_provider_valkey`, or multilevel.
- Rate limiter: use a Redis backend so counters survive instance restarts and replicas share the same state.

For vertical scaling inside one instance, tune `GOMAXPROCS`, database connection pools, and the HTTP read/write timeouts via `piko.yaml`.

## See also

- [How to environment configuration](environment-config.md).
- [How to TLS](tls.md).
- [How to monitoring](monitoring.md).
- [How to troubleshooting deployment](troubleshooting.md).
- [CLI reference](../../reference/cli.md).
- [`config.json` reference](../../reference/config-json-schema.md).
