---
title: How to monitor a production deployment
description: Configure structured logging, OpenTelemetry export, health probes, Sentry, and notification alerts.
nav:
  sidebar:
    section: "how-to"
    subsection: "deployment"
    order: 40
---

# How to monitor a production deployment

Piko ships structured logging, OpenTelemetry hooks, health probes, Prometheus metrics, Sentry integration, and a notification service out of the tree. This guide wires them into a typical production setup. See the [logger reference](../../reference/logger-api.md), [health API reference](../../reference/health-api.md), and [notification reference](../../reference/notification-api.md) for the authoritative surface.

## Switch logging to JSON in production

The development default is a pretty, human-readable console logger. Production should emit structured JSON so log aggregators (Datadog, Splunk, Grafana Loki) can index the fields.

```go
package main

import "piko.sh/piko/wdk/logger"

func main() {
    if os.Getenv("PIKO_ENV") == "prod" {
        logger.AddJSONOutput()
    } else {
        logger.AddPrettyOutput()
    }

    log := logger.GetLogger("myapp")
    log.Info("Server started", logger.String("env", os.Getenv("PIKO_ENV")))
    // ...
}
```

Piko's logger uses a seven-level scheme with framework-internal levels kept separate from user application levels:

| Level | Numeric | Zone | Typical use |
|---|---|---|---|
| `TRACE` | -8 | Framework | Loop-level detail, variable dumps. |
| `INTERNAL` | -6 | Framework | Service registration, cache ops, adapter lifecycle. |
| `DEBUG` | -4 | User | Request/response detail, state dumps. |
| `INFO` | 0 | User | Normal operations. Production default. |
| `NOTICE` | 2 | Shared | Startup, shutdown, other lifecycle events. |
| `WARN` | 4 | Shared | Recoverable issues, deprecations. |
| `ERROR` | 8 | Shared | Failures that need attention. |

Set the level with `PIKO_LOG_LEVEL`. Both names (`info`, `debug`) and numeric values (`0`, `-4`) work.

## Add contextual attributes

Attach per-request attributes with `With(...)`:

```go
log := logger.GetLogger("myapp/handlers").With(
    logger.String("request_id", getRequestID(r)),
    logger.String("method", r.Method),
    logger.String("path", r.URL.Path),
)

log.Info("Request received")
// ... process ...
log.Info("Request completed",
    logger.Int("status_code", 200),
    logger.Duration("duration", time.Since(start)),
)
```

Every log line in that handler now carries `request_id`, `method`, and `path` automatically.

## Export traces and metrics via OpenTelemetry

Configure OTLP in `piko.yaml`:

```yaml
otlp:
  enabled: true
  endpoint: "otel-collector:4317"
  protocol: "grpc"
  headers:
    Authorization: "Bearer ${OTLP_TOKEN}"
  tls:
    insecure: false
```

Environment variable equivalents:

```
PIKO_OTLP_ENABLED=true
PIKO_OTLP_ENDPOINT=otel-collector:4317
PIKO_OTLP_PROTOCOL=grpc
PIKO_OTLP_TLS_INSECURE=false
```

Wrap units of work in spans so traces correlate with log lines:

```go
func (s *OrderService) ProcessOrder(ctx context.Context, orderID string) error {
    log := logger.GetLogger("myapp/orders")

    return log.RunInSpan(ctx, "ProcessOrder", func(spanCtx context.Context, spanLog logger.Logger) error {
        spanLog.Info("Processing order")

        if err := s.validateOrder(spanCtx, orderID); err != nil {
            return fmt.Errorf("validation failed: %w", err)
        }
        if err := s.chargePayment(spanCtx, orderID); err != nil {
            return fmt.Errorf("payment failed: %w", err)
        }
        spanLog.Info("Order completed")
        return nil
    }, logger.String("order_id", orderID))
}
```

`RunInSpan` creates the span, passes a span-scoped logger into the closure, ends the span on return, and records any returned error.

## Wire health probes to Kubernetes

```yaml
healthProbe:
  enabled: true
  port: "9090"
  bindAddress: "0.0.0.0"
  livePath: "/live"
  readyPath: "/ready"
  metricsPath: "/metrics"
  metricsEnabled: true
  checkTimeoutSeconds: 5
```

The probe server runs on its own port, so application-level issues do not affect it.

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: myapp
          ports:
            - containerPort: 8080
              name: http
            - containerPort: 9090
              name: health
          livenessProbe:
            httpGet:
              path: /live
              port: health
            initialDelaySeconds: 5
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: health
            initialDelaySeconds: 5
            periodSeconds: 5
```

The endpoints return JSON:

```json
{
  "name": "application",
  "state": "HEALTHY",
  "timestamp": "2026-04-21T10:00:00Z",
  "duration": "1.234ms",
  "dependencies": [
    {"name": "database", "state": "HEALTHY", "duration": "0.5ms"}
  ]
}
```

State values: `HEALTHY`, `DEGRADED`, `UNHEALTHY`. Liveness returns 200 when healthy, 503 when unhealthy. Readiness returns 200 for healthy or degraded, 503 for unhealthy.

## Register a custom health probe

Implement `Check` for each external dependency:

```go
import (
    "context"
    "database/sql"
    "time"

    "piko.sh/piko/internal/healthprobe/healthprobe_dto"
)

type DatabaseProbe struct {
    db *sql.DB
}

func (p *DatabaseProbe) Name() string { return "database" }

func (p *DatabaseProbe) Check(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
    start := time.Now()
    if err := p.db.PingContext(ctx); err != nil {
        return healthprobe_dto.Status{
            Name:      p.Name(),
            State:     healthprobe_dto.StateUnhealthy,
            Message:   "database ping failed: " + err.Error(),
            Timestamp: time.Now(),
            Duration:  time.Since(start).String(),
        }
    }
    return healthprobe_dto.Status{
        Name:      p.Name(),
        State:     healthprobe_dto.StateHealthy,
        Timestamp: time.Now(),
        Duration:  time.Since(start).String(),
    }
}
```

Register it as a `LifecycleHealthProbe` during bootstrap. See the [health checks how-to](../health-checks.md) for the full registration pattern.

## Scrape Prometheus metrics

`metricsEnabled: true` serves OpenTelemetry metrics at `http://<host>:9090/metrics` in Prometheus exposition format. Point your scraper at that endpoint:

```yaml
scrape_configs:
  - job_name: myapp
    static_configs:
      - targets: ["myapp.default.svc.cluster.local:9090"]
```

Piko emits HTTP request counters, latencies, process stats, and any custom metrics registered through the metrics exporter.

## Enable Sentry error reporting

Import the Sentry integration and configure it at startup:

```go
import (
    "piko.sh/piko/wdk/logger"
    sentry "piko.sh/piko/wdk/logger/logger_integration_sentry"
)

func main() {
    sentry.Enable(sentry.Config{
        DSN:              os.Getenv("SENTRY_DSN"),
        Environment:      os.Getenv("PIKO_ENV"),
        Release:          buildVersion,
        TracesSampleRate: 0.1,
        SampleRate:       1.0,
    })
    logger.AddJSONOutput()
    // ...
}
```

Or enable via YAML with a blank import:

```go
import _ "piko.sh/piko/wdk/logger/logger_integration_sentry"
```

```yaml
logger:
  integrations:
    - type: sentry
      enabled: true
      sentry:
        dsn: "${SENTRY_DSN}"
        environment: production
        release: "myapp@1.0.0"
        tracesSampleRate: 0.1
        sampleRate: 1.0
        eventLevel: error
        breadcrumbLevel: info
        enableTracing: true
```

`log.Error(...)` calls automatically become Sentry events.

## Send alerts to a chat channel

Use the notification service for high-priority incidents. Register a provider at bootstrap:

```go
piko.New(
    piko.WithNotificationProvider(
        "slack",
        notification_provider_slack.New(notification_provider_slack.Config{
            WebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
        }),
    ),
    piko.WithDefaultNotificationProvider("slack"),
)
```

Dispatch notifications from code:

```go
builder, err := notification.NewNotificationBuilderFromDefault()
if err != nil {
    return err
}
return builder.
    Title("Payment provider down").
    Message("Stripe returned 503 for the last 10 minutes.").
    Priority(notification.PriorityHigh).
    Do(ctx)
```

See the [notification reference](../../reference/notification-api.md) for every provider (Slack, Discord, PagerDuty, Teams, Google Chat, ntfy.sh, Webhook, Stdout).

## Forward log errors to notifications

Wire the logger and notifier together so `ERROR` entries become alerts automatically:

```go
import (
    "piko.sh/piko/internal/logger/logger_adapters/integrations"
    "piko.sh/piko/wdk/notification"
)

notifyService, _ := notification.GetDefaultService()
adapter := integrations.NewNotificationServiceAdapter(notifyService)
// Register `adapter` with the logger during startup.
```

## Uptime monitoring

Point an external uptime service at one or both of:

- `https://<public-domain>/` (end-to-end path through the application).
- `http://<host>:9090/ready` (readiness probe, cheaper, does not exercise the page stack).

Typical alert conditions include response time above 5 seconds, non-200 status, or downtime longer than one minute.

## See also

- [How to health checks](../health-checks.md) for custom probes.
- [Logger API reference](../../reference/logger-api.md).
- [Health API reference](../../reference/health-api.md).
- [Notification API reference](../../reference/notification-api.md).
- [How to profiling](../profiling.md) for CPU and allocation analysis.
- [How to troubleshooting deployment](troubleshooting.md).
