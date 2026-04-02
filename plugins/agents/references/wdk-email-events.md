# WDK Email and Events

Use this guide when sending emails, handling event-driven messaging, or building pub/sub patterns.

## Email

The email package provides multi-provider email delivery with fluent builders, templated emails, batching, retry, and dead letter queues.

### Supported providers

| Provider | Package | Use case |
|----------|---------|----------|
| SMTP | `email_provider_smtp` | Standard email servers |
| SendGrid | `email_provider_sendgrid` | SendGrid API |
| AWS SES | `email_provider_ses` | Amazon SES |
| Postmark | `email_provider_postmark` | Postmark API |
| Stdout | `email_provider_stdout` | Development (prints to console) |
| Disk | `email_provider_disk` | Development (writes to files) |
| Mock | `email_provider_mock` | Unit testing |

### Setup

```go
import (
    "piko.sh/piko/wdk/email"
    "piko.sh/piko/wdk/email/email_provider_smtp"
)

provider, err := email_provider_smtp.NewSMTPProvider(ctx, email_provider_smtp.SMTPProviderArgs{
    Host:      "smtp.example.com",
    Port:      587,
    Username:  "user@example.com",
    Password:  "secret",
    FromEmail: "noreply@example.com",
})

svc := email.NewService("smtp")
svc.RegisterProvider("smtp", provider)
svc.SetDefaultProvider("smtp")
```

### Sending emails

```go
import "piko.sh/piko/wdk/email"

// Using the bootstrapped service
builder, err := email.NewEmailBuilderFromDefault()

err = builder.
    To("user@example.com").
    Subject("Welcome").
    BodyHTML("<h1>Welcome!</h1>").
    BodyPlain("Welcome!").
    Do(ctx)
```

### Templated emails (type-safe props)

```go
type WelcomeProps struct {
    Username string
    TrialDays int
}

builder, err := email.NewTemplatedEmailBuilderFromDefault[WelcomeProps]()

err = builder.
    To("user@example.com").
    Subject("Welcome").
    Props(WelcomeProps{Username: "Alice", TrialDays: 14}).
    BodyTemplate("emails/welcome.pk").
    Do(ctx)
```

### Attachments

```go
builder.Attachment("report.pdf", "application/pdf", pdfBytes)
```

### Bulk sending

```go
emails := []*email.SendParams{
    {To: []string{"alice@example.com"}, Subject: "Newsletter", BodyHTML: "<p>Update...</p>"},
    {To: []string{"bob@example.com"}, Subject: "Newsletter", BodyHTML: "<p>Update...</p>"},
}
err := svc.SendBulk(ctx, log, emails)
```

### Dispatcher configuration

The dispatcher handles batching, retry, and dead letter queues:

```go
config := email.DispatcherConfig{
    BatchSize:     10,
    FlushInterval: 30 * time.Second,
    MaxRetries:    3,
    DeadLetterQueue: true,
}
```

### Testing with mock

```go
mockProvider := email_provider_mock.NewMockEmailProvider()
// ... send emails ...
calls := mockProvider.GetSendCalls()
```

## Events

The events package provides pub/sub messaging built on [Watermill](https://watermill.io/). Piko and your application share the same router, publisher, and subscriber.

**Important**: Import Watermill types directly - the events package does not wrap them:

```go
import "github.com/ThreeDotsLabs/watermill/message"
```

### Supported providers

| Provider | Package | Use case |
|----------|---------|----------|
| GoChannel | `events_provider_gochannel` | Development, single instance (default) |
| NATS JetStream | `events_provider_nats` | Production, distributed, durable |

### Publishing messages

```go
import (
    "github.com/ThreeDotsLabs/watermill"
    "github.com/ThreeDotsLabs/watermill/message"
    "piko.sh/piko/wdk/events"
)

pub, err := events.GetPublisher()
msg := message.NewMessage(watermill.NewUUID(), []byte(`{"order_id": "123"}`))
err = pub.Publish("orders.created", msg)
```

### Subscribing to messages

```go
router, _ := events.GetRouter()
sub, _ := events.GetSubscriber()

router.AddNoPublisherHandler(
    "order-processor",
    "orders.created",
    sub,
    func(msg *message.Message) error {
        // nil = Ack, error = Nack (redelivery)
        return processOrder(msg)
    },
)
```

### NATS JetStream setup

```go
import "piko.sh/piko/wdk/events/events_provider_nats"

cfg := events_provider_nats.DefaultConfig()
cfg.URL = "nats://nats-server:4222"

provider, err := events_provider_nats.NewNATSProvider(cfg)
provider.Start(ctx)

app := piko.New(piko.WithEventsProvider(provider))
```

Key NATS features:
- At-least-once delivery (exactly-once with `TrackMessageID: true`)
- Durable subscriptions survive restarts
- Queue groups for competing consumers

### Handler patterns

```go
// Transient errors: Nack for retry
// Permanent errors: log and Ack (prevent infinite redelivery)
func handleOrder(msg *message.Message) error {
    var order Order
    if err := json.Unmarshal(msg.Payload, &order); err != nil {
        log.Error("Invalid payload", "error", err)
        return nil  // Ack malformed message
    }
    if err := processOrder(order); err != nil {
        if isTransient(err) {
            return err  // Nack for retry
        }
        log.Error("Permanent failure", "error", err)
        return nil  // Ack permanent failure
    }
    return nil
}
```

## LLM mistake checklist

- Forgetting to call `svc.SetDefaultProvider()` after registering
- Using `email.NewEmailBuilder(svc)` without first creating a service (use `email.NewEmailBuilderFromDefault()` when bootstrapped)
- Wrapping Watermill types instead of importing them directly
- Returning `nil` for transient errors (message won't be retried)
- Returning `error` for permanently malformed messages (causes infinite redelivery)
- Forgetting `provider.Start(ctx)` before using the NATS provider
- Not calling `provider.Close()` on shutdown

## Related

- `references/server-actions.md` - calling email/events from actions
- `references/wdk-data.md` - persistence and caching
