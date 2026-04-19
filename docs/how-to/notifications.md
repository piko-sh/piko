---
title: How to send notifications
description: Register a notification provider and dispatch alerts to Slack, Discord, email, SMS, or a webhook.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 100
---

# How to send notifications

Piko's notification service is a generic fan-out of messages to one or more providers. Use it for operational alerts (Slack on deploys, PagerDuty on errors) or user-facing messages (SMS on password resets, push on new mentions). This guide covers the common setup. See the [bootstrap options reference](../reference/bootstrap-options.md) for the surrounding API.

## Register a notification provider

Pass a provider at bootstrap:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/adapters/notification/slack"
    "piko.sh/piko/adapters/notification/webhook"
)

ssr := piko.New(
    piko.WithNotificationProvider("slack", slack.NewProvider(slack.Config{
        WebhookURL: slackWebhookURL,
        DefaultChannel: "#ops",
    })),
    piko.WithNotificationProvider("pagerduty", webhook.NewProvider(webhook.Config{
        URL: "https://events.pagerduty.com/...",
    })),
    piko.WithDefaultNotificationProvider("slack"),
)
```

`WithDefaultNotificationProvider` picks which provider handles un-routed messages. Explicitly target a specific provider when the message has a specific destination.

## Send from an action

Use `piko.GetNotificationService()` inside an action or a background task:

```go
package orders

import (
    "piko.sh/piko"
    "piko.sh/piko/internal/notification/notification_dto"
)

type FulfilInput struct {
    OrderID int64 `json:"orderID" validate:"required"`
}

type FulfilAction struct {
    piko.ActionMetadata
}

func (a *FulfilAction) Call(input FulfilInput) (piko.NoResponse, error) {
    order, err := fulfilOrder(a.Ctx(), input.OrderID)
    if err != nil {
        return piko.NoResponse{}, piko.NewError("could not fulfil", err)
    }

    piko.GetNotificationService().Send(a.Ctx(), &notification_dto.Message{
        Subject: "Order fulfilled",
        Body:    fmt.Sprintf("Order %d dispatched to %s", order.ID, order.Customer),
        Tags:    []string{"order-fulfilment"},
    })

    return piko.NoResponse{}, nil
}
```

The default provider handles the message. To target a specific provider, pass its name:

```go
piko.GetNotificationService().SendWithProvider(a.Ctx(), "pagerduty", &notification_dto.Message{
    Subject: "Order processor degraded",
    Body:    err.Error(),
    Severity: notification_dto.SeverityError,
})
```

## Message fields

| Field | Purpose |
|---|---|
| `Subject` | Short headline shown in dense channels (email subject, Slack preview). |
| `Body` | Full message body; providers interpret this as markdown where supported. |
| `Recipient` | Provider-specific recipient (email address, phone number, Slack channel). Leave empty for default. |
| `Tags` | Labels for routing and filtering. Providers can use these to pick a channel. |
| `Severity` | `Info`, `Warn`, `Error`, `Critical`. Providers map to their native severity. |
| `Metadata` | Map of custom key-value pairs; provider-specific. |

## Choose a provider implementation

Typical provider adapters that ship or that an application might write:

| Provider | Description |
|---|---|
| Webhook | POSTs JSON to an arbitrary URL. Works for PagerDuty, Opsgenie, custom dashboards. |
| Slack | POSTs to a Slack incoming webhook with channel routing. |
| Discord | POSTs to a Discord webhook. |
| Email | Wraps the email service (re-uses `piko.WithEmailProvider` under the hood). |
| SMS | Ships through a provider like Twilio or AWS SNS. |

For any provider outside the adapter library, implement the `notification_domain.NotificationProviderPort` interface and register it.

## Do not block the request

Notification sends are typically fire-and-forget. Sending synchronously inside an action adds latency for every request. Use one of three patterns:

1. **Spawn a goroutine inside the action**: acceptable for best-effort notifications where a failed send is not fatal.
2. **Use the orchestrator service** (see [how to background tasks](background-tasks.md)) for reliable queued delivery with retry.
3. **Batch-send from a lifecycle component** that collects notifications and flushes on a short interval.

## Failure behaviour

`Send` returns an error, and `SendWithProvider` does too. The framework does not retry. For reliability, wrap the call in the orchestrator.

## See also

- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithNotificationProvider` and `WithDefaultNotificationProvider`.
- [How to background tasks](background-tasks.md) for reliable asynchronous sends.
- [How to email templates](email-templates.md) for email-specific patterns.
