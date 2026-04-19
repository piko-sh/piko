---
title: Frontend analytics module
description: Configuration and automatic events emitted by the ModuleAnalytics frontend module.
nav:
  sidebar:
    section: "reference"
    subsection: "frontend"
    order: 10
---

# Frontend analytics module

`piko.ModuleAnalytics` is a built-in frontend module that adds Google Analytics 4 (GA4) and `Google Tag Manager` (GTM) support with zero application code. Once enabled, the module automatically tracks page views and routing events, and exposes hooks that PKC or PK scripts can call. This page documents the configuration and emitted events.

For backend analytics (server-side collectors, `TrackAnalyticsEvent`, action events), see the [analytics API reference](analytics-api.md). The two sides run independently. Enable both for end-to-end coverage.

## Enable the module

```go
ssr := piko.New(
    piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{
        TrackingIDs:     []string{"G-XXXXXXXXXX"},
        GTMContainerID:  "GTM-XXXXXXX",
        DebugMode:       false,
        AnonymiseIP:     true,
        DisablePageView: false,
    }),
)
```

## `piko.AnalyticsConfig`

| Field | Type | Default | Purpose |
|---|---|---|---|
| `TrackingIDs` | `[]string` | `nil` | GA4 measurement IDs (for example `"G-XXXXXXXXXX"`). Multiple IDs duplicate events to each property. |
| `GTMContainerID` | `string` | `""` | GTM container ID (for example `"GTM-XXXXXXX"`). When set, the GTM snippet loads and SPA navigation events push to `dataLayer`. |
| `DebugMode` | `bool` | `false` | Enables console logging for debugging. Leave off in production. |
| `AnonymiseIP` | `bool` | `false` | Enables GA4's IP anonymisation. Recommended for GDPR/UK GDPR compliance. |
| `DisablePageView` | `bool` | `false` | Disables automatic page-view tracking. Set to `true` if you want to send page views manually or track only custom events. |

The module needs at least one of `TrackingIDs` or `GTMContainerID`. Both can coexist. When neither carries a value, the module still loads but emits no network traffic.

## Automatic events

When the module runs with per-event tracking intact, the following events fire:

| Event | When it fires |
|---|---|
| `page_view` | On initial page load and on every SPA-style navigation (when a Piko navigation occurs without a full reload). |
| `piko_action` | After a server action returns a response, successful or failed. Properties include action name and status. |
| `piko_error` | When the runtime catches a client-side error. |

Both GA4 and GTM receive the events (GTM gets them via `dataLayer.push`).

## Manual tracking from PKC or PK scripts

The module exposes a global `piko.analytics` object with tracking helpers:

```typescript
piko.analytics.track("button_clicked", {
    button: "checkout",
    plan:   "annual",
});
```

For ecommerce events:

```typescript
piko.analytics.track("purchase", {
    transaction_id: order.id,
    value:          order.total,
    currency:       "GBP",
    items:          order.items,
});
```

GA4's standard event names and parameters all work. Refer to the [GA4 event reference](https://developers.google.com/analytics/devguides/collection/ga4/events) for the canonical list.

## Disable automatic page views

Set `DisablePageView: true` if you route through a custom system that prefers explicit page-view calls. Emit them yourself:

```typescript
piko.analytics.track("page_view", {
    page_path:  window.location.pathname,
    page_title: document.title,
});
```

## Interaction with GTM

When `GTMContainerID` carries a value, the module pushes events into the GTM `dataLayer` with the event name prefix `piko_`. GTM triggers can pattern-match those event names without the module caring which GTM container the page loads.

## Privacy

- `AnonymiseIP` is the right default for most jurisdictions. GA4 respects the flag when sending request data.
- The module does not fingerprint the browser beyond what GA4 itself collects.
- Cookie consent is the application's responsibility. The module does not gate itself; combine it with a cookie-consent solution that sets `window.gtag` consent flags as appropriate.

## See also

- [Analytics API reference](analytics-api.md) for backend analytics events and collectors.
- [How to analytics](../how-to/analytics.md) for backend collector implementation.
- [Scenario 028: analytics ecommerce](../showcase/028-analytics-ecommerce.md) for a runnable example.
- [Bootstrap options reference](bootstrap-options.md) for `WithFrontendModule`.
