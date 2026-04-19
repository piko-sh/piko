---
title: About analytics
description: The collector interface, the frontend-vs-backend split, enrichment, and privacy trade-offs.
nav:
  sidebar:
    section: "explanation"
    subsection: "operations"
    order: 50
---

# About analytics

Piko emits analytics automatically. Each page view and each action invocation becomes a structured event. Registered collectors forward the events to their destinations. The event shape, the frontend-backend split, and the automatic enrichment all reflect deliberate choices. This page explains those choices.

## Two channels, one event shape

Piko has two analytics channels. The backend channel runs server-side. Every page request produces one `EventPageView` through the middleware, and every action produces one `EventAction`. A backend collector receives the event in Go, enriches it, and forwards it to a destination such as Plausible, Mixpanel, or a custom warehouse. The frontend channel runs in the browser. A small client-side module tracks page views in a single-page-app sense (navigating between routes without a full reload), JavaScript errors, and custom events the application emits.

The two channels share the event shape but not the emission path. A page load produces one backend event on the server and one frontend event in the browser. Neither tries to deduplicate against the other. The server event is authoritative for request-level analytics (response times, error rates, authenticated user identity). The frontend event is authoritative for user-behaviour analytics (scroll depth, time on page, rage clicks, JavaScript errors).

Splitting the channels keeps each one honest about what it knows. A server-side collector cannot observe scroll depth because the server sees one request. A browser-side collector cannot reliably observe response time because its clock starts after the server has already sent bytes. Giving each channel its own collector interface avoids the pretence that one can do the other's job.

## The collector interface

A backend collector implements `AnalyticsCollector` (alias for `analytics_domain.Collector`). The interface gives a collector its own lifecycle so it can batch, flush on shutdown, and identify itself in logs:

```go
type AnalyticsCollector interface {
    Start(ctx context.Context)
    Collect(ctx context.Context, event *AnalyticsEvent) error
    Flush(ctx context.Context) error
    Close(ctx context.Context) error
    Name() string
}
```

The framework does not prescribe how collectors handle batching, retry, or credentials. The GA4 collector in-tree batches events in `Collect`, sends them in `Flush`, and applies exponential backoff with retry. The Plausible collector posts each event directly because Plausible's API is low-cost. The stdout collector writes JSON for developers. A custom collector picks whichever strategy matches its destination.

`Flush` runs during graceful shutdown after the server stops accepting traffic, so any in-flight buffer makes it out before the process exits. `Close` releases connections and goroutines after the final flush.

Frontend collectors follow the same shape in TypeScript, exposed as `piko.analytics.track(event)` on the client runtime.

## What automatic enrichment adds

A collector does not need to populate every field of `AnalyticsEvent`. The framework fills in the fields it knows about before the collector runs:

- `ClientIP` from the request, after proxy-aware parsing (respecting `X-Forwarded-For` when the server trusts it).
- `Locale` from the request context.
- `MatchedPattern` from the routing layer (so `/blog/hello-world` appears as `/blog/{slug}`).
- `Hostname` from the request `Host` header.
- `UserID` from an authenticated session, if the application has declared one.
- `Timestamp` as a monotonic reference.

Enrichment respects values the caller sets. If an action emits an event with an already-populated `UserID`, the enrichment does not overwrite it. The intent is that the bulk of events get their context for free, and callers who know more can add it explicitly.

An action can promote the automatic page-view event. `SetAnalyticsEventName(ctx, "search.submitted")` renames the page view that the middleware is about to emit. `AddAnalyticsProperty(ctx, key, value)` attaches additional properties before emission. Promotion avoids duplicate events when an action wants to say "this page view is actually a search-submission event".

## Why the property cap

`AddAnalyticsProperty` silently drops entries past 64 per event. The cap exists because most downstream analytics systems have a similar limit, and properties are not the right place for unbounded data. Exceeding 64 properties usually signals that the caller should structure data differently. Emit a custom event with a dedicated property set instead of piling everything onto the page view.

## Privacy trade-offs

Analytics and privacy pull in different directions. Piko makes two choices that lean toward privacy and leaves others to the application.

The first choice. IP anonymisation is opt-in at the collector level, not at the middleware level. The framework passes the full client IP to the collector. Each collector decides whether to trim it before shipping. GA4 has its own IP anonymisation setting. Plausible never receives the IP in the first place. A stdout collector logs whatever the framework passes in. Centralising the choice inside each collector keeps each destination's contract with the application explicit.

The second choice. The framework never generates a persistent user identifier. Page views are anonymous by default. A value for `UserID` only appears if the application sets one, typically from an authenticated session. This avoids accidental tracking of logged-out visitors across sessions through a synthetic identifier.

The choices Piko leaves to the application:

- Whether to set a cookie for user tracking. Piko does not add one.
- Whether to track user-agent strings. The collector receives the UA through the request but does not forward it unless the collector chooses to.
- Whether to comply with region-specific consent regimes (GDPR, `CCPA`). Compliance is application-level, not framework-level. A common pattern registers the collector conditionally, so declining consent skips the registration.

## Collector failure is non-fatal

A failed `Track` call does not fail the request. Analytics is observability, not business logic. A dropped event is acceptable. A dropped order is not. The framework logs collector errors and moves on. Collectors that need retry logic implement it internally. The GA4 adapter is the reference pattern.

Analytics volumes are therefore a floor, not a ceiling. A collector outage drops events. For destinations where that matters (for example, a financial audit log), the event type does not belong in analytics. It belongs in an ordinary service call.

## Ordering and context

Automatic enrichment depends on the request context. Events emitted from background goroutines outside a request context ship without enrichment. A background job that tracks "email sent" has to set `ClientIP` manually or leave it empty.

Order of emission within a request is not guaranteed to match order of `Track` invocation. The worker pool processes events asynchronously. Do not rely on timestamps with sub-millisecond precision to sequence events from the same request. If sequencing matters, set a monotonic counter explicitly.

## When to reach for analytics and when not to

Reach for analytics when the question is about aggregate user behaviour. Which pages get visits, which actions get invoked, how conversion funnels move. Analytics answers statistical questions about the population.

Reach for tracing (OpenTelemetry spans) when the question is about a specific request. Which call was slow, which downstream failed, what the payload looked like. Tracing answers mechanical questions about individual requests.

Reach for structured logging when the question is about an error or audit-worthy action. Who approved a purchase, which credential failed, what happened at 03:14. Logs answer forensic questions.

Using analytics for forensics fails because of the failure-mode trade-off above. A dropped event vanishes without trace. Using traces or logs for aggregate analysis works but costs far more than analytics to store and query.

## See also

- [Analytics API reference](../reference/analytics-api.md) for the backend surface.
- [Frontend analytics reference](../reference/frontend-analytics.md) for the browser-side module.
- [How to analytics](../how-to/analytics.md) for registering collectors, emitting custom events, and batching patterns.
- [Scenario 028: analytics ecommerce](../showcase/028-analytics-ecommerce.md) for a runnable example.
- [About the hexagonal architecture](about-the-hexagonal-architecture.md) for the collector interface in context.
