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

Fluent methods: `.Provider(name)`, `.Recipient(...)`, `.Title(...)`, `.Message(...)`, `.Priority(...)`, `.Type(...)`, `.Do(ctx)`.

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
| Provider names | `ProviderSlack`, `ProviderDiscord`, `ProviderPagerDuty`, `ProviderTeams`, `ProviderGoogleChat`, `ProviderNtfy`, `ProviderWebhook`, `ProviderStdout` |

## Providers

| Sub-package | Backend |
|---|---|
| `notification_provider_slack` | Slack webhooks and Web API. |
| `notification_provider_discord` | Discord webhooks. |
| `notification_provider_pagerduty` | PagerDuty events. |
| `notification_provider_teams` | Microsoft Teams. |
| `notification_provider_googlechat` | Google Chat. |
| `notification_provider_ntfy` | ntfy.sh. |
| `notification_provider_webhook` | Generic HTTP webhook. |
| `notification_provider_stdout` | Development logger. |

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithNotificationProvider(name, provider)` | Registers a provider. |
| `piko.WithDefaultNotificationProvider(name)` | Marks the default. |

## See also

- [How to notifications](../how-to/notifications.md) for task recipes.
