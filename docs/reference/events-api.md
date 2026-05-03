---
title: Events API
description: Watermill-based message bus accessors for publishing and subscribing to events.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 180
---

# Events API

Piko wraps [Watermill](https://watermill.io) for typed in-process or distributed event messaging. Two backends ship by default. In-memory GoChannel and NATS. The facade exposes shared router, publisher, and subscriber instances that Piko builds once and shares across the application. For task recipes see the [background tasks how-to](../how-to/background-tasks.md). Source file: [`wdk/events/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/events/facade.go).

## Accessors

| Function | Returns |
|---|---|
| `events.GetRouter() (*message.Router, error)` | The shared router. |
| `events.GetPublisher() (message.Publisher, error)` | The shared publisher. |
| `events.GetSubscriber() (message.Subscriber, error)` | The shared subscriber. |
| `events.GetProvider() (Provider, error)` | The underlying provider. |
| `events.IsRunning() bool` | Whether the router is running. |

The `message.*` types come from [`github.com/ThreeDotsLabs/watermill/message`](https://pkg.go.dev/github.com/ThreeDotsLabs/watermill/message). Piko does not re-export them.

## Types

| Type | Purpose |
|---|---|
| `Provider` | Backend-agnostic bus provider. |
| `ProviderConfig` | Shared settings. |
| `RouterConfig` | Router-specific settings. |

## Logger adapter

```go
events.NewWatermillLoggerAdapter(l logger.Logger) watermill.LoggerAdapter
```

Bridges Piko's logger into Watermill so messages land in the same structured-log pipeline as the rest of the application.

## Defaults

`events.DefaultProviderConfig()` returns the default config (in-memory GoChannel with back-pressure enabled).

## Providers

| Sub-package | Backend |
|---|---|
| `events_provider_gochannel` | In-memory (default). |
| `events_provider_nats` | NATS message broker. |

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithEventsProvider(provider)` | Replaces the default provider. |

## See also

- [How to background tasks](../how-to/background-tasks.md) uses the router for worker pools.
- [Watermill documentation](https://watermill.io) for handler signatures and middleware.
