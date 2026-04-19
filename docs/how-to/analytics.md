---
title: How to wire backend analytics
description: Register a collector, emit custom events, attach revenue and properties to automatic events.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 30
---

# How to wire backend analytics

Piko emits backend analytics events automatically. Each page load produces one `EventPageView`, and each action invocation produces one `EventAction`. A project registers one or more collectors to forward those events to a destination (Google Analytics, Plausible, Mixpanel, stdout, a webhook). This guide walks through the setup. See the [analytics API reference](../reference/analytics-api.md) for the full API.

## Register a collector at bootstrap

Pass collectors into [`WithBackendAnalytics`](../reference/bootstrap-options.md#analytics):

```go
import (
    "log"

    "piko.sh/piko"
    "piko.sh/piko/wdk/analytics/analytics_collector_stdout"
    ga4 "piko.sh/piko/wdk/analytics/analytics_collector_ga4"
)

ga4Collector, err := ga4.NewCollector(
    os.Getenv("GA4_MEASUREMENT_ID"),
    os.Getenv("GA4_API_SECRET"),
)
if err != nil {
    log.Fatal(err)
}

ssr := piko.New(
    piko.WithBackendAnalytics(
        analytics_collector_stdout.NewCollector(),
        ga4Collector,
    ),
)
```

`ga4.NewCollector` validates the measurement ID and API secret eagerly and returns an error if either is empty. The stdout collector cannot fail and exposes a single-return constructor.

Every collector receives every event through its `Collect` method. Piko does not retry on failure. A collector that requires retry logic must implement it (the in-tree GA4 collector is the reference).

## Implement a collector

A collector satisfies `piko.AnalyticsCollector` (alias for `analytics_domain.Collector`). Implement all five methods. The simplest collector treats most of them as no-ops.

```go
package stdout

import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    "piko.sh/piko"
)

type Collector struct{}

func NewCollector() *Collector { return &Collector{} }

func (c *Collector) Start(ctx context.Context) {}

func (c *Collector) Collect(ctx context.Context, event *piko.AnalyticsEvent) error {
    raw, err := json.Marshal(event)
    if err != nil {
        return err
    }
    fmt.Fprintln(os.Stdout, string(raw))
    return nil
}

func (c *Collector) Flush(ctx context.Context) error { return nil }

func (c *Collector) Close(ctx context.Context) error { return nil }

func (c *Collector) Name() string { return "stdout" }
```

`Collect` runs asynchronously in a worker pool. A slow collector does not block the request. Collectors that need to batch should accumulate in `Collect` and send in `Flush`. `Start` runs once at bootstrap (use it for background loops), `Flush` runs on graceful shutdown plus optional periodic timers, and `Close` runs once after the final flush.

## Emit a custom event

Call `piko.TrackAnalyticsEvent` from any action or render function:

```go
func (a CheckoutCompleteAction) Call(orderID int64) (Response, error) {
    order, err := orders.Get(a.Ctx(), orderID)
    if err != nil {
        return Response{}, err
    }

    piko.TrackAnalyticsEvent(a.Ctx(), &piko.AnalyticsEvent{
        Type:      piko.EventCustom,
        EventName: "checkout.completed",
        Properties: map[string]string{
            "order_id": fmt.Sprint(order.ID),
            "plan":     order.Plan,
        },
    })

    return Response{OK: true}, nil
}
```

Piko enriches the event from the request context. It populates `ClientIP`, `Locale`, `Hostname`, `UserID` (if authenticated), and the matched route pattern automatically.

## Promote the automatic event to a custom one

Instead of emitting a second event, change the name of the automatic one for the current request:

```go
func (a SearchAction) Call(query string) (Response, error) {
    piko.SetAnalyticsEventName(a.Ctx(), "search.submitted")
    piko.AddAnalyticsProperty(a.Ctx(), "query_length", fmt.Sprint(len(query)))

    results, err := search.Query(a.Ctx(), query)
    if err != nil {
        return Response{}, err
    }

    return Response{Results: results}, nil
}
```

The analytics middleware promotes the automatic pageview event to a custom event with the given name once the handler returns.

## Attach revenue

```go
piko.SetAnalyticsRevenue(a.Ctx(), maths.GBP(29, 99))
```

Piko attaches the value to the automatic event for the current request.

## Property limits

`AddAnalyticsProperty` silently drops entries past the cap of 64 per event. Keep property sets small because they carry per-event overhead in every collector.

## Ordering and context

Piko enriches custom events emitted from actions using the current request. Events emitted from background goroutines (outside a request context) ship without enrichment. Set `ClientIP`, `UserID`, and other fields manually if the destination needs them.

## Batch events in a high-volume collector

A collector that posts every event synchronously becomes a bottleneck when traffic is high. A production collector batches events and flushes periodically. The GA4 collector in `wdk/analytics/analytics_collector_ga4` is the reference pattern. Key pieces:

```go
package custom

import (
    "context"
    "sync"
    "time"

    "piko.sh/piko"
)

type Collector struct {
    endpoint string
    client   httpClient
    buf      []*piko.AnalyticsEvent
    mu       sync.Mutex
    flush    chan struct{}
    done     chan struct{}
}

func NewCollector(endpoint string) *Collector {
    c := &Collector{
        endpoint: endpoint,
        client:   defaultClient,
        buf:      make([]*piko.AnalyticsEvent, 0, 128),
        flush:    make(chan struct{}, 1),
        done:     make(chan struct{}),
    }
    go c.loop()
    return c
}

func (c *Collector) Start(ctx context.Context) {
    go c.loop()
}

func (c *Collector) Collect(ctx context.Context, event *piko.AnalyticsEvent) error {
    c.mu.Lock()
    c.buf = append(c.buf, event)
    full := len(c.buf) >= 128
    c.mu.Unlock()

    if full {
        select {
        case c.flush <- struct{}{}:
        default:
        }
    }
    return nil
}

func (c *Collector) Flush(ctx context.Context) error {
    c.flushNow()
    return nil
}

func (c *Collector) Close(ctx context.Context) error {
    close(c.done)
    return nil
}

func (c *Collector) Name() string { return "custom-batch" }

func (c *Collector) loop() {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-c.done:
            c.flushNow()
            return
        case <-c.flush:
            c.flushNow()
        case <-ticker.C:
            c.flushNow()
        }
    }
}
```

`flushNow` takes the buffered events, posts them in one request, and handles retry or circuit-breaker logic. The details depend on the target:

- Retry with exponential backoff for 5xx responses. Cap attempts.
- Open a circuit breaker after N consecutive failures so the collector stops attempting for a cooldown window.
- Drop the oldest events when the buffer fills beyond a hard limit; analytics data is not business-critical.
- Piko calls `Flush` during graceful shutdown, so any buffered events ship before `Close` releases resources.

See [`wdk/analytics/analytics_collector_ga4`](https://github.com/piko-sh/piko/tree/master/wdk/analytics/analytics_collector_ga4) for the full batching, retry, and circuit-breaker implementation.

## See also

- [Analytics API reference](../reference/analytics-api.md) for the full surface.
- [About analytics](../explanation/about-analytics.md) for the collector-interface rationale, the frontend-vs-backend split, and privacy trade-offs.
- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithBackendAnalytics`.
- [Scenario 028: analytics ecommerce](../../examples/scenarios/028_analytics_ecommerce/) for a full walkthrough.
