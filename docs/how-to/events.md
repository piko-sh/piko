---
title: How to publish and subscribe to events
description: Use Piko's Watermill-based router to broadcast and handle events within a single process or across a bus.
nav:
  sidebar:
    section: "how-to"
    subsection: "services"
    order: 60
---

# How to publish and subscribe to events

This guide shows how to publish messages to Piko's event bus and how to register a handler that consumes them. See the [events reference](../reference/events-api.md) for the full API. The underlying library is [Watermill](https://watermill.io).

## Get the publisher and subscriber

```go
package main

import (
    "context"
    "encoding/json"

    "github.com/ThreeDotsLabs/watermill/message"
    "piko.sh/piko/wdk/events"
)

func publishCustomerCreated(ctx context.Context, customerID string) error {
    publisher, err := events.GetPublisher()
    if err != nil {
        return err
    }

    payload, err := json.Marshal(map[string]string{"id": customerID})
    if err != nil {
        return err
    }

    msg := message.NewMessage(customerID, payload)
    return publisher.Publish("customer.created", msg)
}
```

The publisher is the shared instance configured at bootstrap. The topic name is arbitrary. Pick a convention (`<bounded-context>.<event>`) and document it.

## Handle events on startup

Register handlers on Piko's router during bootstrap:

```go
package main

import (
    "context"
    "encoding/json"

    "github.com/ThreeDotsLabs/watermill/message"
    "piko.sh/piko"
    "piko.sh/piko/wdk/events"
)

func main() {
    ssr := piko.New()

    ctx := context.Background()
    router, err := events.GetRouter()
    if err != nil {
        panic(err)
    }

    subscriber, err := events.GetSubscriber()
    if err != nil {
        panic(err)
    }

    router.AddNoPublisherHandler(
        "send-welcome-email",
        "customer.created",
        subscriber,
        func(msg *message.Message) error {
            var payload struct {
                ID string `json:"id"`
            }
            if err := json.Unmarshal(msg.Payload, &payload); err != nil {
                return err
            }
            return sendWelcomeEmail(ctx, payload.ID)
        },
    )

    ssr.Run()
}
```

`AddNoPublisherHandler` is appropriate when the handler does not publish further events. Use `AddHandler` when it does, to get ack semantics right.

## Configure a non-default backend

The default provider is in-process GoChannel. For distributed messaging, register the NATS provider at bootstrap:

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/events/events_provider_nats"
)

natsProvider, err := events_provider_nats.NewNATSProvider(events_provider_nats.Config{
    URL: "nats://localhost:4222",
})
if err != nil {
    panic(err)
}

ssr := piko.New(
    piko.WithEventsProvider(natsProvider),
)
```

Two providers ship with Piko. The in-process `events_provider_gochannel` (the default) and `events_provider_nats` for distributed messaging. Both live under `wdk/events/events_provider_*`.

## See also

- [Events API reference](../reference/events-api.md).
- [How to background tasks](background-tasks.md) for queueing and retry patterns.
- [Watermill documentation](https://watermill.io) for handler middleware and routing options.
