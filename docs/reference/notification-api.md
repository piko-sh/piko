---
title: Notification API
description: Notification service, builder, dispatcher, and provider registration.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 160
---

# Notification API

Piko's notification service sends messages to chat platforms, incident tools, and ad-hoc webhooks. Callers send notifications synchronously or queue them through a dispatcher that retries with backoff and routes undeliverable messages to a dead-letter queue. For task recipes see the [notifications how-to](../how-to/notifications.md). Source of truth: [`wdk/notification/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/notification/facade.go).

## Service

| Function | Returns |
|---|---|
| `notification.NewService() Service` | Constructs a new service. |
| `notification.GetDefaultService() (Service, error)` | Returns the bootstrap-configured service. |

## Builder

```go
func NewNotificationBuilder(service Service) (*NotificationBuilder, error)
func NewNotificationBuilderFromDefault() (*NotificationBuilder, error)
```

Fluent methods: `.Title(...)`, `.Message(...)`, `.Field(key, value)`, `.Fields(map)`, `.Image(url)`, `.Priority(...)`, `.Type(...)`, `.Source(...)`, `.Environment(...)`, `.Service(...)`, `.TraceID(...)`, `.Provider(name)`, `.ToProviders(names...)`, `.ProviderOption(key, value)`, `.ProviderOptions(map)`, `.Do(ctx)`.

## Types

| Type | Purpose |
|---|---|
| `Service` | Manages providers. |
| `ProviderPort` | Interface a provider implements. |
| `DispatcherPort` | Interface for async dispatchers. |
| `SendParams` | Full parameter struct. |
| `NotificationContent` | Core content: title, body, attachments. |
| `ProviderCapabilities` | Declares feature support per provider. |
| `DispatcherConfig` | Backoff, concurrency, DLQ. |
| `DispatcherStats` | Live dispatcher counters. |
| `RetryConfig` | Backoff parameters. |
| `DeadLetterEntry` | A message the dispatcher failed to deliver. |

## Constants

| Group | Values |
|---|---|
| Priority | `PriorityLow`, `PriorityNormal`, `PriorityHigh`, `PriorityCritical` |
| Type | `TypePlain`, `TypeRich`, `TypeTemplated` |
| Provider name | `ProviderSlack`, `ProviderDiscord`, `ProviderPagerDuty`, `ProviderTeams`, `ProviderGoogleChat`, `ProviderNtfy`, `ProviderWebhook`, `ProviderStdout` |

## Providers

The `wdk/notification` package exposes the service facade, dispatcher, builder, and `ProviderPort` interface. The provider implementations themselves live in the Piko-internal package `internal/notification/notification_adapters/driver_providers` and are not directly importable from user code. Instead, callers register their own `ProviderPort` implementations via `WithNotificationProvider`.

Piko auto-registers a built-in **stdout** provider as the default when callers configure no custom provider. Piko exports the provider name constant `ProviderStdout` (along with `ProviderSlack`, `ProviderDiscord`, `ProviderPagerDuty`, `ProviderTeams`, `ProviderGoogleChat`, `ProviderNtfy`, `ProviderWebhook`) for convenience when selecting a registered provider through the builder's `.Provider(name)` method or `WithDefaultNotificationProvider`.

To send to other destinations (Slack webhooks, PagerDuty events, internal HTTP endpoints, and so on), implement `ProviderPort` in your application and register it with the service. Keep the destination logic in your application code, and share it across services as needed.

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithNotificationProvider(name, provider)` | Registers a provider. |
| `piko.WithDefaultNotificationProvider(name)` | Marks the default. |

## See also

- [How to notifications](../how-to/notifications.md) for task recipes.
