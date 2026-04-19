---
title: How to wire frontend analytics (GA4 and GTM)
description: Enable the analytics frontend module for Google Analytics 4 or Google Tag Manager, adjust the Content Security Policy so the third-party scripts load, and keep your first-party deployment compliant.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 40
---

# How to wire frontend analytics (GA4 and GTM)

This guide integrates Google Analytics 4 directly, swaps it for Google Tag Manager, and makes the Content Security Policy (CSP) accept whichever one ships. For the field-level surface of `piko.AnalyticsConfig` see [frontend analytics reference](../reference/frontend-analytics.md). For backend analytics collectors (Plausible, custom webhooks) see [how to analytics](analytics.md).

## Pick GA4 direct or GTM

The two integrations share Piko's frontend module. Their CSP requirements are distinct.

| Integration | When to pick | CSP cost |
|---|---|---|
| `GA4` direct (`TrackingIDs`) | The project only needs Google Analytics 4. No ad tech, no custom tags, no Facebook Pixel. | Small. Add `googletagmanager.com` to `script-src` and the GA4 endpoints to `connect-src`. |
| GTM container (`GTMContainerID`) | Any mix of GA4, Facebook Pixel, Google Ads, custom event tags, or marketing tooling managed by a non-developer. | Large. GTM injects inline event handlers and uses `eval`, so the policy needs `unsafe-inline`, `unsafe-eval`, and a wide host allowlist. |

If the answer is "just GA4", integrate direct. If the answer is "the marketing team has a GTM account", use GTM and accept the CSP trade-off. Mixing both is legal but rarely useful.

## Integrate GA4 on its own

### Enable the module at bootstrap

```go
ssr := piko.New(
    piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{
        TrackingIDs: []string{"G-XXXXXXXXXX"},
        DebugMode:   false,
    }),
)
```

Replace `G-XXXXXXXXXX` with your GA4 Measurement ID. Multiple IDs in the slice fire the same events into multiple properties (useful during a migration between GA properties).

### Extend the CSP for GA4

GA4's loader lives at `www.googletagmanager.com/gtag/js`, and its event endpoints sit under `*.google-analytics.com` and `*.analytics.google.com`. Extend Piko's defaults to allow them:

```go
piko.WithCSP(func(b *piko.CSPBuilder) {
    b.WithPikoDefaults().
        ScriptSrc(piko.CSPSelf, piko.CSPHost("https://www.googletagmanager.com")).
        ConnectSrc(piko.CSPSelf,
            piko.CSPHost("https://www.google-analytics.com"),
            piko.CSPHost("https://*.google-analytics.com"),
            piko.CSPHost("https://*.analytics.google.com"),
            piko.CSPHost("https://*.googletagmanager.com"),
        )
}),
```

`WithPikoDefaults()` keeps Piko's hardened base policy (no `unsafe-inline`, no `unsafe-eval`, restricted `default-src`). The `ScriptSrc` and `ConnectSrc` calls layer the GA4 hosts on top. The combination lets GA4 load and send events while leaving every other surface locked down.

### Verify GA4 events are firing

Enable debug mode temporarily:

```go
piko.AnalyticsConfig{
    TrackingIDs: []string{"G-XXXXXXXXXX"},
    DebugMode:   true,
}
```

Open the Google Analytics DebugView (`Configure` then `DebugView` in the GA4 console). Navigate to a page, click something that Piko instruments automatically (an action, a form submit). Events appear in DebugView within seconds. Switch `DebugMode: false` before deploying to production. Debug mode doubles the event volume the browser ships.

## Integrate through GTM

### Enable the module at bootstrap

```go
ssr := piko.New(
    piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{
        GTMContainerID: "GTM-XXXXXXX",
        DebugMode:      false,
    }),
)
```

Piko's frontend runtime injects the GTM loader and pushes `dataLayer` events for every page view and action invocation. The tags a marketer configures in the GTM console (GA4, Facebook Pixel, Google Ads, custom events) then fire based on those events.

### Build the CSP from scratch

GTM requires three things Piko's defaults refuse:

1. **`script-src 'unsafe-inline'`.** GTM's bootstrap writes inline `<script>` elements that hold the tag configuration.
2. **`script-src 'unsafe-eval'`.** GTM built-in tags use `eval` or `new Function(...)` internally.
3. **`script-src-attr 'unsafe-inline'`.** GTM binds inline event handlers (`onclick`, `onload`) to elements that it adds to the DOM at runtime.

Because these conflict with `WithPikoDefaults`, build the policy from scratch. The following shape covers a typical GA4-through-GTM plus Facebook Pixel plus Google Ads deployment:

```go
piko.WithCSP(func(b *piko.CSPBuilder) {
    b.DefaultSrc(piko.CSPSelf).
        StyleSrc(
            piko.CSPSelf,
            piko.CSPUnsafeInline,
            piko.CSPHost("https://fonts.googleapis.com"),
        ).
        ScriptSrc(
            piko.CSPSelf,
            piko.CSPUnsafeInline,
            piko.CSPUnsafeEval,
            piko.CSPHost("https://www.googletagmanager.com"),
            piko.CSPHost("https://tagmanager.google.com"),
            piko.CSPHost("https://connect.facebook.net"),
            piko.CSPHost("https://googleads.g.doubleclick.net"),
            piko.CSPHost("https://www.googleadservices.com"),
        ).
        ScriptSrcAttr(piko.CSPUnsafeInline).
        FontSrc(
            piko.CSPSelf,
            piko.CSPHost("https://fonts.gstatic.com"),
            piko.CSPData,
        ).
        ImgSrc(piko.CSPSelf, piko.CSPData, piko.CSPBlob, piko.CSPHTTPS).
        ConnectSrc(
            piko.CSPSelf,
            piko.CSPHost("https://www.google-analytics.com"),
            piko.CSPHost("https://*.google-analytics.com"),
            piko.CSPHost("https://*.analytics.google.com"),
            piko.CSPHost("https://*.googletagmanager.com"),
            piko.CSPHost("https://www.google.com"),
            piko.CSPHost("https://www.facebook.com"),
            piko.CSPHost("https://*.facebook.com"),
        ).
        FrameSrc(
            piko.CSPSelf,
            piko.CSPHost("https://td.doubleclick.net"),
        )
}),
```

Reasoning for each directive:

- `StyleSrc` with `unsafe-inline` because GTM injects `<style>` tags for some tag templates, and Google Fonts hosts the font CSS.
- `ScriptSrc` with `unsafe-inline`, `unsafe-eval`, and the GTM / Google Ads / Facebook hosts.
- `ScriptSrcAttr` with `unsafe-inline` because GTM wires inline event handlers.
- `FontSrc` accepts Google Fonts plus `data:` URIs (Google inlines small fonts).
- `ImgSrc` is permissive because tracking pixels come from a wide and changing set of domains.
- `ConnectSrc` covers analytics beacons (GA4, Ads, Facebook).
- `FrameSrc` allows the conversion-linker iframes Google injects for cross-domain tracking.

Treat this block as a starting template. Every time a marketer adds a new tag in the GTM console, check the browser's console for CSP violations and extend the appropriate directive.

### Use server-side GTM to reduce third-party exposure

Server-side GTM (sGTM) runs the tag container on a server you control, typically at a first-party subdomain such as `sgtm.example.com`. The browser ships events to your domain. Your sGTM container then forwards them to GA4, Facebook, and other destinations.

Benefits:

- **Fewer third-party hosts in CSP.** Only the sGTM subdomain needs to appear in `script-src` and `connect-src`.
- **Harder to block.** Ad blockers and privacy tools trained on `googletagmanager.com` do not recognise your subdomain.
- **More control over data.** You can enrich or filter events before they leave your infrastructure.

CSP for sGTM:

```go
piko.WithCSP(func(b *piko.CSPBuilder) {
    b.DefaultSrc(piko.CSPSelf).
        ScriptSrc(
            piko.CSPSelf,
            piko.CSPUnsafeInline,
            piko.CSPUnsafeEval,
            piko.CSPHost("https://sgtm.example.com"),
        ).
        ScriptSrcAttr(piko.CSPUnsafeInline).
        ConnectSrc(
            piko.CSPSelf,
            piko.CSPHost("https://sgtm.example.com"),
        ).
        FrameSrc(
            piko.CSPSelf,
            piko.CSPHost("https://sgtm.example.com"),
        )
}),
```

Point the `GTMContainerID` at the sGTM container, and configure your GTM admin to route the loader and beacons through the custom subdomain. See [Google's sGTM documentation](https://developers.google.com/tag-platform/tag-manager/server-side) for the DNS and deployment side.

## Common CSP pitfalls

Three mistakes come up often. All of them surface as console errors in the browser's devtools.

- **`Refused to execute inline script`.** GTM needs `script-src 'unsafe-inline'` and `script-src-attr 'unsafe-inline'`. Piko's defaults exclude both, so a plain `WithPikoDefaults()` plus host allowlists does not work for GTM. Build the policy from scratch.
- **`Refused to connect to ...`.** A tag configured in GTM tries to send a beacon to a domain not in `connect-src`. The browser tells you which host. Add it and redeploy.
- **`Refused to load the image ...`.** Tracking pixels come from varied hosts. The `ImgSrc(piko.CSPSelf, piko.CSPData, piko.CSPBlob, piko.CSPHTTPS)` block covers most cases by allowing any HTTPS image. Lock this down further if the project's policy demands it.

After every change, ship to a staging environment, open devtools, and refresh the page. CSP violations print in red and the affected feature silently fails. Every red line is a lead on what to allow.

## Privacy controls

`piko.AnalyticsConfig` exposes two fields that shape what leaves the browser:

```go
piko.AnalyticsConfig{
    TrackingIDs:     []string{"G-XXXXXXXXXX"},
    AnonymiseIP:     true,   // strips the last octet of IPv4 addresses before sending
    DisablePageView: false,  // suppresses the automatic page_view event on every navigation
}
```

`AnonymiseIP: true` is a GDPR-friendly default. `DisablePageView: true` helps when a single-page-app navigates between client-side routes faster than the page-view event needs to fire. The application then emits a single custom event per logical screen.

For the full field surface, including `DebugMode` and the list of automatic events (`page_view`, `piko_action`, `piko_error`), see [frontend analytics reference](../reference/frontend-analytics.md).

## Emit custom events

From any PK or PKC template, call the runtime helper:

```typescript
piko.analytics.track("checkout_started", {
    cart_value: 4999,
    currency: "GBP",
});
```

The event reaches GA4 (direct) or lands in the `dataLayer` (GTM), where marketing tags pick it up.

From a Go action, the backend collector path is a better fit:

```go
piko.TrackAnalyticsEvent(a.Ctx(), &piko.AnalyticsEvent{
    Type: piko.EventCustom,
    Name: "checkout_completed",
})
```

The two channels cover different cases. See [about analytics](../explanation/about-analytics.md) for the frontend-versus-backend split and when to reach for each.

## See also

- [Frontend analytics reference](../reference/frontend-analytics.md) for every field on `piko.AnalyticsConfig`.
- [Bootstrap options reference](../reference/bootstrap-options.md#csp) for every CSP helper, including `WithStrictCSP`, `WithRelaxedCSP`, `WithAPICSP`, and `WithCSPString`.
- [How to analytics](analytics.md) for backend collectors (Plausible, webhooks, custom).
- [About analytics](../explanation/about-analytics.md) for the design rationale behind the split.
- [Scenario 028: analytics ecommerce](../showcase/028-analytics-ecommerce.md) for a runnable GA4 example.
