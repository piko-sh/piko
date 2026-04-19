---
title: Email API
description: Email service, builders, dispatcher, and provider registration.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 120
---

# Email API

Piko's email service sends transactional email through one of six provider backends. Callers compose emails directly or render them from type-safe templates, and the service can dispatch them in the background with batching, retry, and a dead-letter queue. For task recipes see the [email templates how-to](../how-to/email-templates.md). Source file: [`wdk/email/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/email/facade.go).

## Service

| Function | Returns |
|---|---|
| `email.NewService(defaultProviderName string, opts ...ServiceOption) Service` | Constructs a new service. |
| `email.GetDefaultService() (Service, error)` | Returns the service the bootstrap built. |

## Builders

```go
func NewEmailBuilder(service Service) (*EmailBuilder, error)
func NewEmailBuilderFromDefault() (*EmailBuilder, error)
func NewTemplatedEmailBuilder[PropsT any](service Service) (*TemplatedEmailBuilder[PropsT], error)
func NewTemplatedEmailBuilderFromDefault[PropsT any]() (*TemplatedEmailBuilder[PropsT], error)
```

The templated builder is generic on a `PropsT` struct that mirrors the template's prop shape, so Piko compiles the body from a PK email template with typed inputs.

Fluent calls on either builder include `.To(...)`, `.Cc(...)`, `.Bcc(...)`, `.From(...)`, `.Subject(...)`, `.Body(...)`, `.BodyHTML(...)`, `.Attach(...)`, `.Header(key, value)`, `.Priority(...)`. Terminate with `.Do(ctx)`.

## Types

| Type | Purpose |
|---|---|
| `Service` | Entry point. Manages providers. |
| `ProviderPort` | Interface a provider implements. |
| `EmailBuilder` | Fluent composer for plain or HTML emails. |
| `TemplatedEmailBuilder[PropsT]` | Fluent composer for templated emails. |
| `SendParams` | Complete parameter struct accepted by the service. |
| `Attachment` | File attachment (filename, MIME type, bytes). |
| `MultiError` | Error type that aggregates failures from bulk sends. |

## Background dispatcher

| Type | Purpose |
|---|---|
| `DispatcherConfig` | Batching, retry, DLQ, and concurrency settings. |
| `DispatcherStats` | Runtime counters and latency data. |
| `DeadLetterEntry` | An email the dispatcher failed to deliver. |

## Providers

| Sub-package | Backend |
|---|---|
| `email_provider_smtp` | SMTP server, supports STARTTLS and TLS. |
| `email_provider_ses` | AWS `Simple Email Service`. |
| `email_provider_sendgrid` | SendGrid API. |
| `email_provider_mailgun` | Mailgun API. |
| `email_provider_postmark` | Postmark API. |
| `email_provider_resend` | Resend API. |
| `email_provider_mock` | In-memory test double. |

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithEmailProvider(name, provider)` | Registers a provider under a name. |
| `piko.WithDefaultEmailProvider(name)` | Marks a registered provider as default. |
| `piko.WithEmailDispatcher(cfg)` | Enables the background dispatcher. |
| `piko.WithEmailDeadLetterQueue(queue)` | Persists undeliverable emails. |

## See also

- [How to email templates](../how-to/email-templates.md) for composing PK email templates.
- [Scenario 026: email contact form](../showcase/026-email-contact.md) for a runnable example.
