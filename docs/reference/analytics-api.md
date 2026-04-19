---
title: Analytics API
description: Collector interface, event types, and helper functions for backend analytics.
nav:
  sidebar:
    section: "reference"
    subsection: "runtime"
    order: 60
---

# Analytics API

Piko emits backend analytics events automatically (page views, action invocations) and lets application code emit custom events. A project registers one or more collectors via the bootstrap option [`WithBackendAnalytics`](bootstrap-options.md#analytics) and Piko fan-outs each event to every collector. This page documents the API. Source of truth: [`analytics.go`](https://github.com/piko-sh/piko/blob/master/analytics.go).

## Interface

### `AnalyticsCollector`

Type alias for `analytics_domain.Collector`. A collector receives every analytics event and is responsible for forwarding it to its destination (Google Analytics, Plausible, Mixpanel, stdout, a webhook).

Implementers must satisfy five methods:

| Method | Purpose |
|---|---|
| `Start(ctx)` | Launch any background goroutines (flush loops, batch workers). Called once at bootstrap. |
| `Collect(ctx, *AnalyticsEvent) error` | Receive a single event. Implementations may buffer for batching. |
| `Flush(ctx) error` | Send any buffered events to the destination. Called during graceful shutdown and optionally on a timer. |
| `Close(ctx) error` | Release resources. Called once after `Flush` during shutdown. |
| `Name() string` | Human-readable identifier for logging and metrics. |

Application code does NOT call these directly. Use `piko.TrackAnalyticsEvent(ctx, event)` and the helpers below - Piko fans out to every registered collector through `Collect`.

## Types

### `AnalyticsEvent`

Type alias for `analytics_dto.Event`. Carries the data for a single event:

- `Type` (`AnalyticsEventType`): event kind.
- `EventName` (`string`): explicit name for custom events (separate from `ActionName` which identifies a server action).
- `ActionName` (`string`): server action name; populated automatically for `EventAction` events.
- `Timestamp` (`time.Time`): when the event occurred.
- `Request` (`*http.Request`): raw request, available for adapters that read additional headers.
- `ClientIP`, `Locale`, `Hostname`, `UserID`, `MatchedPattern`, `URL`, `Path`, `Method`, `UserAgent`, `Referrer` (`string`): enriched automatically from the request context when available.
- `Duration` (`time.Duration`): request handling time.
- `StatusCode` (`int`): HTTP response status code.
- `Properties` (`map[string]string`): custom key-value pairs (max 64).
- `Revenue` (`*maths.Money`): optional monetary value.

### `AnalyticsEventType`

Type alias for `analytics_dto.EventType`. Values:

| Constant | When fired |
|---|---|
| `EventPageView` | Automatic, once per page request. |
| `EventAction` | Automatic, once per action invocation. |
| `EventCustom` | Fired manually with `TrackAnalyticsEvent` or by promoting the current pageview via `SetAnalyticsEventName`. |

## Functions

### `TrackAnalyticsEvent(ctx, event)`

Sends a custom event to every registered collector. Piko enriches the event from the current request context (client IP, locale, matched route pattern, authenticated user ID) when one is available. Calling this outside a request, or when the project registers no collectors, is a no-op.

```go
event := &piko.AnalyticsEvent{
    Type:       piko.EventCustom,
    EventName:  "checkout.completed",
    Properties: map[string]string{"plan": "annual"},
}
piko.TrackAnalyticsEvent(ctx, event)
```

### `SetAnalyticsRevenue(ctx, revenue)`

Attaches revenue data to the current request's automatic event. No-op outside a request context.

```go
piko.SetAnalyticsRevenue(ctx, maths.NewMoneyFromMinorInt(2999, "GBP"))
```

### `AddAnalyticsProperty(ctx, key, value)`

Attaches a custom property to the current request's automatic event. Silently drops the property when the event already holds 64 properties. No-op outside a request context.

### `SetAnalyticsEventName(ctx, name)`

Promotes the automatic pageview event to a named custom event. No-op outside a request context.

```go
piko.SetAnalyticsEventName(ctx, "search.submitted")
```

## Enrichment

Every event sent through `TrackAnalyticsEvent` has its empty fields filled in from the request context:

| Field | Source |
|---|---|
| `ClientIP` | Resolved from the trusted-proxy chain. |
| `Locale` | Current request locale. |
| `Hostname` | Request host. |
| `MatchedPattern` | The matched route pattern (for example `/blog/{slug}`). |
| `UserID` | The authenticated user ID if an auth guard has populated one. |

Piko preserves any field that the caller sets explicitly.

## Registration

Collectors register at bootstrap:

```go
import (
    "log"

    "piko.sh/piko"
    "piko.sh/piko/wdk/analytics/analytics_collector_stdout"
    ga4 "piko.sh/piko/wdk/analytics/analytics_collector_ga4"
)

ga4Collector, err := ga4.NewCollector("G-MEASUREMENT", "api-secret", ga4.WithDebug(true))
if err != nil {
    log.Fatalf("ga4 collector: %v", err)
}

ssr := piko.New(
    piko.WithBackendAnalytics(
        analytics_collector_stdout.NewCollector(),
        ga4Collector,
    ),
)
```

Shipped collector packages:

| Package | Constructor | Use |
|---|---|---|
| `analytics_collector_stdout` | `NewCollector()` | Logs events to stdout. Useful in dev. |
| `analytics_collector_ga4` | `NewCollector(measurementID, apiSecret, opts...)` | Google Analytics 4. |
| `analytics_collector_plausible` | `NewCollector(domain, opts...)` | Plausible. |
| `analytics_collector_webhook` | `NewCollector(url, opts...)` | Generic webhook POST. |

## See also

- [How to analytics](../how-to/analytics.md) for collector implementation and custom-event patterns.
- [About analytics](../explanation/about-analytics.md) for the collector-interface rationale and frontend/backend split.
- [Bootstrap options reference](bootstrap-options.md) for `WithBackendAnalytics`.
- Source: [`analytics.go`](https://github.com/piko-sh/piko/blob/master/analytics.go).
