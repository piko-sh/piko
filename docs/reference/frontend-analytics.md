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

`piko.ModuleAnalytics` is a built-in frontend module that adds Google Analytics 4 (GA4) and Google Tag Manager (GTM) integration. When enabled, the module tracks page views and routing events automatically and exposes a tracking surface to PKC and PK scripts. This page documents the configuration and emitted events.

For backend analytics (server-side collectors, `TrackAnalyticsEvent`, action events), see the [analytics API reference](analytics-api.md). The two surfaces are independent.

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
| `DebugMode` | `bool` | `false` | Enables console logging of every dispatched event. |
| `AnonymiseIP` | `bool` | `false` | Enables GA4's IP anonymisation. |
| `DisablePageView` | `bool` | `false` | Suppresses automatic page-view tracking. |

The module loads with no network traffic when neither `TrackingIDs` nor `GTMContainerID` carries a value. Both can coexist.

## Automatic events

When the module runs with per-event tracking intact, the following events fire. GA4 receives the canonical event name via `gtag('event', ...)`. GTM receives a parallel `piko_`-prefixed event via `dataLayer.push`.

| GA4 event | GTM event | When it fires |
|---|---|---|
| `page_view` | `piko_page_view` | On initial page load and on every SPA-style navigation (when a Piko navigation occurs without a full reload). Suppressed when `DisablePageView` is true. |
| `navigation` | `piko_navigation` | On `navigation:complete` after a successful SPA navigation. Carries `navigation_trigger` and navigation duration. |
| `exception` | `piko_error` | On `navigation:error` when an SPA navigation fails, and on the runtime `error` hook when a client-side error reaches the catch handler. |
| `server_action` | `piko_action` | After a server action returns a response, successful or failed. Properties include action name, success flag, and duration. |
| `modal_view` | `piko_modal_view` | On `modal:open` when a modal becomes visible. Properties include modal ID. |

Custom events sent through `piko.analytics.track(name, params)` reach GA4 under `name` and GTM under `piko_<name>`.

## Manual tracking from PKC or PK scripts

The module installs a global `piko.analytics` object exposing `track(name, params)`.

```typescript
piko.analytics.track("button_clicked", {
    button: "checkout",
    plan:   "annual",
});

piko.analytics.track("purchase", {
    transaction_id: order.id,
    value:          order.total,
    currency:       "GBP",
    items:          order.items,
});
```

GA4 receives the canonical event name. GTM receives a parallel `piko_<name>` event. The [GA4 event reference](https://developers.google.com/analytics/devguides/collection/ga4/events) documents the full list of GA4 event names and parameters.

## Page-view suppression

`DisablePageView: true` suppresses the module's automatic `page_view` and `piko_page_view` emissions. Send manual page views through `piko.analytics.track`.

```typescript
piko.analytics.track("page_view", {
    page_path:  window.location.pathname,
    page_title: document.title,
});
```

## Interaction with GTM

When `GTMContainerID` carries a value, the module pushes events into the GTM `dataLayer` with the event name prefix `piko_`. The [Automatic events](#automatic-events) table documents the GA4-to-GTM name mapping.

## Privacy

`AnonymiseIP` toggles GA4's IP anonymisation flag. The module performs no browser fingerprinting beyond what GA4 itself collects. The module does not implement consent gating. Cookie-consent flags (`window.gtag` consent state) remain the application's responsibility.

## See also

- [Analytics API reference](analytics-api.md) for backend analytics events and collectors.
- [About analytics](../explanation/about-analytics.md) for the frontend/backend split rationale.
- [How to analytics](../how-to/analytics.md) for backend collector implementation.
- [Scenario 028: analytics ecommerce](../../examples/scenarios/028_analytics_ecommerce/) for a runnable example.
- [Bootstrap options reference](bootstrap-options.md) for `WithFrontendModule`.
