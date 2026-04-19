---
title: Health API
description: Interfaces, types, and endpoints for liveness and readiness probes.
nav:
  sidebar:
    section: "reference"
    subsection: "bootstrap"
    order: 50
---

# Health API

Piko exposes `/live` and `/ready` endpoints on a separate port (default 9090, bound to localhost). Each endpoint aggregates every registered health probe and returns an overall state. This page documents the interfaces and built-in probes. Source of truth: [`health.go`](https://github.com/piko-sh/piko/blob/master/health.go).

## Endpoints

| Endpoint | Purpose | Failure consequence |
|---|---|---|
| `/live` | Is the application running (not deadlocked)? | Orchestrator typically restarts the container. |
| `/ready` | Is the application ready to serve traffic? | Orchestrator typically withholds traffic until ready. |

Both run on a separate health HTTP server. The default configuration binds to `127.0.0.1:9090`. Expose on `0.0.0.0` for container orchestrators only when necessary, because the health endpoint reveals internal service state.

## Interface

### `HealthProbe`

```go
type HealthProbe interface {
    Name() string
    Check(ctx context.Context, checkType HealthCheckType) HealthStatus
}
```

- `Name()` returns the probe's display name (appears in the aggregated output).
- `Check(ctx, checkType)` runs the probe. Return quickly for liveness; readiness may do a thorough connection check.

## Types

### `HealthCheckType`

| Constant | Meaning |
|---|---|
| `HealthCheckLiveness` | Is the service alive? |
| `HealthCheckReadiness` | Is the service ready to accept traffic? |

### `HealthState`

| Constant | Meaning |
|---|---|
| `HealthStateHealthy` | Component is working normally. |
| `HealthStateDegraded` | Component is working but with reduced performance or limited features. |
| `HealthStateUnhealthy` | Component is not working. |

### `HealthStatus`

```go
type HealthStatus struct {
    Name         string
    State        HealthState
    Message      string
    Timestamp    time.Time
    Duration     string
    Dependencies []*HealthStatus
}
```

`Dependencies` lets a probe report the health of nested sub-components (for example, a database probe that wraps separate replica probes).

## Registration

```go
ssr := piko.New(
    piko.WithCustomHealthProbe(&RedisProbe{client: redisClient}),
)
```

See the [bootstrap options reference](bootstrap-options.md#health) for TLS options on the health endpoint (`WithHealthTLS`).

## Built-in probes

Piko registers probes for its internal services automatically:

- `RegistryService` (artefact storage and metadata)
- `OrchestratorService` (task queue and workers)
- `CollectionService` (content providers)
- `RenderService` (template rendering pipeline)
- `StorageService` (file and blob storage)
- `EmailService` (email delivery)
- `CryptoService` (encryption operations)
- `CacheService` (cache backends)

Piko also registers any component that implements [`LifecycleComponent`](lifecycle-api.md) as a health probe. Its default probe reports the component as healthy for liveness and degraded for readiness, because it did not supply a readiness check. Implement [`LifecycleHealthProbe`](lifecycle-api.md) for fine-grained readiness.

## Response format

A healthy response:

```json
{
  "status": "healthy",
  "probes": [
    { "name": "RegistryService", "state": "healthy", "duration": "2ms" },
    { "name": "ApplicationRedisCache", "state": "healthy", "duration": "1ms" }
  ]
}
```

Any probe in `unhealthy` state promotes the overall status to `unhealthy` and returns HTTP 503.

## Configuration

In `piko.yaml`:

```yaml
healthProbe:
  enabled: true
  port: "9090"
  bindAddress: "127.0.0.1"
  livePath: "/live"
  readyPath: "/ready"
  checkTimeoutSeconds: 5
```

Expose externally only when the orchestrator needs it:

```yaml
healthProbe:
  bindAddress: "0.0.0.0"
```

## See also

- [How to health checks](../how-to/health-checks.md) for implementing probes.
- [Lifecycle API reference](lifecycle-api.md) for `LifecycleWithHealth`.
- [Bootstrap options reference](bootstrap-options.md) for `WithCustomHealthProbe` and `WithHealthTLS`.
- Source: [`health.go`](https://github.com/piko-sh/piko/blob/master/health.go).
