---
title: How to add a custom health probe
description: Implement HealthProbe for an application-specific check and register it with the framework.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 40
---

# How to add a custom health probe

A custom health probe monitors an application-specific dependency (a Redis cache, an external API, a feature flag service) and contributes its state to Piko's `/live` and `/ready` endpoints. This guide shows how to write and register one. See the [health API reference](../reference/health-api.md) for the surface.

## Implement the interface

A probe implements `piko.HealthProbe`:

```go
package probes

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "piko.sh/piko"
)

type RedisProbe struct {
    client *redis.Client
}

func NewRedisProbe(client *redis.Client) *RedisProbe {
    return &RedisProbe{client: client}
}

func (p *RedisProbe) Name() string {
    return "ApplicationRedis"
}

func (p *RedisProbe) Check(ctx context.Context, checkType piko.HealthCheckType) piko.HealthStatus {
    start := time.Now()

    err := p.client.Ping(ctx).Err()
    state := piko.HealthStateHealthy
    message := "redis reachable"
    if err != nil {
        state = piko.HealthStateUnhealthy
        message = fmt.Sprintf("redis ping failed: %v", err)
    }

    return piko.HealthStatus{
        Name:      p.Name(),
        State:     state,
        Message:   message,
        Timestamp: time.Now(),
        Duration:  time.Since(start).String(),
    }
}
```

The `checkType` argument distinguishes liveness checks from readiness checks. A liveness check is fast and answers "is the process alive?". A readiness check is thorough and answers "can this component serve traffic right now?". Return quickly for liveness because a ping is usually sufficient. For readiness, an end-to-end operation such as write-then-read, schema check, or auth-token refresh works well.

## Distinguish liveness from readiness

```go
func (p *RedisProbe) Check(ctx context.Context, checkType piko.HealthCheckType) piko.HealthStatus {
    if checkType == piko.HealthCheckLiveness {
        return liveness(ctx, p)
    }
    return readiness(ctx, p)
}
```

Liveness should never fail for transient issues. If it returns `Unhealthy`, the orchestrator restarts the process. Readiness can fail more eagerly. A `Degraded` or `Unhealthy` result tells the orchestrator to stop routing traffic until the probe recovers.

## Register the probe

Pass it at bootstrap:

```go
redisClient := redis.NewClient(&redis.Options{Addr: os.Getenv("REDIS_ADDR")})

ssr := piko.New(
    piko.WithCustomHealthProbe(probes.NewRedisProbe(redisClient)),
)
```

The probe contributes to both `/live` and `/ready` automatically.

## Combine with lifecycle

A component that needs both managed startup/shutdown and health probing can implement both interfaces. See the [lifecycle how-to](lifecycle.md) for the combined pattern.

## Choose a state value

| State | Meaning | Behaviour |
|---|---|---|
| `HealthStateHealthy` | Component is working normally. | Endpoint returns 200. |
| `HealthStateDegraded` | Component is working with reduced performance or limited features. | Endpoint returns 200 but surfaces the state in the response body. |
| `HealthStateUnhealthy` | Component is not working. | Endpoint returns 503. |

A single `Unhealthy` probe marks the overall endpoint as unhealthy. `Degraded` probes do not. The endpoint stays 200.

## Timeouts and context

The `ctx` argument carries a deadline (default 5 seconds, configurable in `piko.yaml`). Honour it. The framework cancels any probe that blocks past the deadline, and the health endpoint reports it as unhealthy with a timeout message.

## Expose the endpoint

The health server defaults to `127.0.0.1:9090`. To expose it to an orchestrator, update `piko.yaml`:

```yaml
healthProbe:
  bindAddress: "0.0.0.0"
```

The endpoint reveals internal service state. Do not expose it on a public network without the reverse-proxy controls appropriate for your environment.

## See also

- [Health API reference](../reference/health-api.md).
- [How to lifecycle](lifecycle.md) for `LifecycleWithHealth`.
- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithHealthTLS`.
