---
title: "028: Analytics ecommerce"
description: Track pageviews, add-to-cart events, and revenue through a custom analytics collector.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 480
---

# 028: Analytics ecommerce

A small e-commerce site wired to a custom analytics collector. Piko tracks every pageview automatically. Add-to-cart and checkout events attach revenue data and custom properties.

## What this demonstrates

- `WithBackendAnalytics` with a custom collector implementation.
- `piko.SetAnalyticsRevenue` to attach monetary values to pageviews.
- `piko.AddAnalyticsProperty` for per-event custom properties.
- `piko.SetAnalyticsEventName` to promote a pageview to a named custom event.
- `piko.TrackAnalyticsEvent` for fully custom events emitted from actions.

## Project structure

```text
src/
  cmd/main/main.go      Bootstrap wiring the frontend analytics module and backend collectors.
  pages/
    index.pk            A single shop page with products and an add-to-cart action.
  actions/
    cart/
      purchase.go       Records a purchase and emits the analytics event.
  partials/             Shared layout and product-card partials.
  components/           Client components that drive the analytics events.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/028_analytics_ecommerce/src/
go mod tidy
air
```

The configured backend collectors receive every event. Swap them for a real collector (GA4, Plausible, Mixpanel) in production.

## See also

- [How to analytics](../how-to/analytics.md).
- [Analytics API reference](../reference/analytics-api.md).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/028_analytics_ecommerce).
