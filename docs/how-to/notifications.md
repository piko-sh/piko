---
title: How to send out-of-band notifications
description: Use the notification service to deliver alerts, transactional pings, and system messages through pluggable destination providers.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 100
---

# How to send out-of-band notifications

The notification service delivers system messages such as alerts, pings, deploy hooks, and customer-state updates to one or more destinations. Application code composes a notification with a fluent builder and the service fans it out to registered providers.

Piko exposes the service shell, the `ProviderPort` interface, the dispatcher, and the `NotificationBuilder` through `wdk/notification`. Only the `stdout` provider auto-registers as a development fallback when you configure no other provider. Supply your own adapter to deliver to Slack, PagerDuty, Discord, Microsoft Teams, Google Chat, ntfy, or a generic webhook. Use the provider-name constants `notification.ProviderSlack`, `ProviderDiscord`, `ProviderTeams`, `ProviderPagerDuty`, `ProviderGoogleChat`, `ProviderNtfy`, `ProviderWebhook`, and `ProviderStdout` when registering so routing rules and dashboards that switch on names keep working. See the [notification API reference](../reference/notification-api.md) for the full type surface.

## Implement a provider

A provider satisfies `notification.ProviderPort`. The contract is:

```go
type ProviderPort interface {
    Send(ctx context.Context, params *notification.SendParams) error
    SendBulk(ctx context.Context, notifications []*notification.SendParams) error
    SupportsBulkSending() bool
    GetCapabilities() notification.ProviderCapabilities
    Close(ctx context.Context) error
}
```

You supply the provider name at registration time via `WithNotificationProvider(name, provider)`. It is not part of the interface.

A minimal Slack provider:

```go
package slacknotify

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"

    "piko.sh/piko/wdk/notification"
)

type Provider struct {
    webhookURL string
    httpClient *http.Client
}

func New(webhookURL string) *Provider {
    return &Provider{webhookURL: webhookURL, httpClient: http.DefaultClient}
}

func (p *Provider) Send(ctx context.Context, params *notification.SendParams) error {
    body, _ := json.Marshal(map[string]string{
        "text": fmt.Sprintf("*%s*\n%s", params.Content.Title, params.Content.Message),
    })
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.webhookURL, bytes.NewReader(body))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")
    resp, err := p.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        return fmt.Errorf("slack: status %d", resp.StatusCode)
    }
    return nil
}

func (p *Provider) SendBulk(ctx context.Context, notifications []*notification.SendParams) error {
    for _, params := range notifications {
        if err := p.Send(ctx, params); err != nil {
            return err
        }
    }
    return nil
}

func (p *Provider) SupportsBulkSending() bool { return false }

func (p *Provider) GetCapabilities() notification.ProviderCapabilities {
    return notification.ProviderCapabilities{}
}

func (p *Provider) Close(_ context.Context) error { return nil }
```

## Register at bootstrap

```go
ssr := piko.New(
    piko.WithNotificationProvider("slack", slacknotify.New(os.Getenv("SLACK_WEBHOOK_URL"))),
    piko.WithNotificationProvider("pagerduty", pdnotify.New(os.Getenv("PD_TOKEN"))),
)
```

`WithNotificationProvider(name, provider)` adds a provider to the service registry. The first registered provider becomes the default, used when a notification does not pick one explicitly. Use `WithDefaultNotificationProvider(name)` to override.

## Build and send

The fluent builder composes a notification:

```go
import "piko.sh/piko/wdk/notification"

func raiseDeployAlert(ctx context.Context, version string) error {
    svc, err := notification.GetDefaultService()
    if err != nil {
        return err
    }
    return svc.NewNotification().
        Title("Deploy in progress").
        Message(fmt.Sprintf("Rolling out version %s", version)).
        Field("version", version).
        Field("environment", "production").
        Priority(notification.PriorityHigh).
        Provider("slack").
        Do(ctx)
}
```

Builder methods. Chain freely and terminate with `Do(ctx)`.

| Method | Purpose |
|---|---|
| `Title(s)` | Notification title. |
| `Message(s)` | Body. |
| `Field(key, value)` | Add a structured field (provider decides how to render). |
| `Fields(map[string]string)` | Add multiple fields at once. |
| `Image(url)` | Attach an image URL. |
| `Priority(p)` | `PriorityLow`, `PriorityNormal`, `PriorityHigh`, `PriorityCritical`. |
| `Type(t)` | Provider-specific message type. |
| `Source(s)` / `Environment(s)` / `Service(s)` | Routing hints. |
| `TraceID(s)` | Tie the notification to a trace. |
| `Provider(name)` | Route to one specific provider. |
| `ToProviders(names...)` | Multi-cast to a list of providers. |
| `ProviderOption(key, value)` | Per-provider extra. |
| `ProviderOptions(map)` | Multiple extras at once. |
| `Do(ctx) error` | Send. Terminus. |

## Send to multiple providers at once

```go
svc, _ := notification.GetDefaultService()
svc.NewNotification().
    Title("Database failover").
    Message("Primary DB is down; standby promoted").
    Priority(notification.PriorityCritical).
    ToProviders("slack", "pagerduty").
    Do(ctx)
```

## Bulk-send

For batched messages (digests, alert flushes), call the service directly:

```go
service, err := notification.GetDefaultService()
if err != nil {
    return err
}
err = service.SendBulk(ctx, []*notification.SendParams{
    {Content: notification.NotificationContent{Title: "Item 1", Message: "..."}, Priority: notification.PriorityLow},
    {Content: notification.NotificationContent{Title: "Item 2", Message: "..."}, Priority: notification.PriorityLow},
})
```

`SendBulkWithProvider(ctx, "slack", []*SendParams{...})` targets one provider. `SendBulk` uses the default. `SendToProviders(ctx, params, []string{"slack", "pagerduty"})` multicasts a single notification to a list of providers.

## Reliability

The dispatcher applies retry with exponential backoff per provider. Providers that fail past their retry budget surface in the dead-letter store. For critical alerts, multi-cast with `ToProviders(...)` so a single provider failure does not lose the message.

## See also

- [Notification API reference](../reference/notification-api.md) for the type surface.
- [How to background tasks](background-tasks.md) for sending notifications inside an executor.
- [How to configure the watchdog](observability/configure-watchdog.md) for wiring the watchdog to a notification provider.
