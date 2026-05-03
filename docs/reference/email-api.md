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

Piko's email service sends transactional email through one of ten provider backends. Callers compose emails directly or render them from type-safe templates, and the service can dispatch them in the background with batching, retry, and a dead-letter queue. For task recipes see the [email templates how-to](../how-to/email-templates.md). Source file: [`wdk/email/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/email/facade.go).

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

### Shared builder methods

Both `EmailBuilder` and `TemplatedEmailBuilder[PropsT]` expose a fluent interface terminated by `.Do(ctx)`.

| Method | Purpose |
|---|---|
| `.To(addresses ...string)` | Append recipient addresses. |
| `.Cc(addresses ...string)` | Append CC addresses. |
| `.Bcc(addresses ...string)` | Append blind carbon copy (BCC) recipients. |
| `.From(address string)` | Set the sender address. |
| `.Subject(subject string)` | Set the subject line. |
| `.Attachment(filename, mimeType string, content []byte)` | Attach a file to the message. |
| `.Provider(name string)` | Override the provider for this send only. |
| `.Immediate()` | Bypass the dispatcher queue and send straight away. |
| `.ProviderOption(key string, value any)` | Pass a provider-specific option to the adapter. |
| `.Do(ctx context.Context) error` | Validate and send (or queue) the email. |
| `.Build() email_dto.SendParams` | Return a deep copy of the configured parameters. |
| `.Clone()` | Return an independent copy of the builder. |

### `EmailBuilder` body methods

| Method | Purpose |
|---|---|
| `.BodyHTML(html string)` | Set the HTML body. |
| `.BodyPlain(plain string)` | Set the plain-text body. |

### `TemplatedEmailBuilder[PropsT]` template methods

| Method | Purpose |
|---|---|
| `.BodyTemplate(templatePath string)` | Use a PK email template at the given path; rendering happens inside `.Do`. |
| `.Props(props PropsT)` | Provide the strongly typed template props. |
| `.Request(request *http.Request)` | Supply the HTTP request used to resolve template paths. |
| `.PremailerOptions(opts premailer.Options)` | Override the default CSS-inlining settings. |
| `.BodyPlain(plain string)` | Provide a custom plain-text body that overrides the template's generated text. |

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
| `email_provider_ses` | AWS Simple Email Service. |
| `email_provider_sendgrid` | SendGrid API. |
| `email_provider_mailgun` | Mailgun API. |
| `email_provider_postmark` | Postmark API. |
| `email_provider_mailchimp_transactional` | Mailchimp Transactional (formerly Mandrill). |
| `email_provider_gmail` | Gmail API. |
| `email_provider_disk` | Writes messages to disk; useful for snapshot tests. |
| `email_provider_stdout` | Logs messages to stdout; useful in dev. |
| `email_provider_mock` | In-memory test double. |

## Bootstrap options

| Option | Purpose |
|---|---|
| `piko.WithEmailService(service)` | Replaces the bootstrap-built service with a pre-constructed one. |
| `piko.WithEmailProvider(name, provider)` | Registers a provider under a name. |
| `piko.WithDefaultEmailProvider(name)` | Marks a registered provider as default. |
| `piko.WithEmailDispatcher(cfg)` | Enables the background dispatcher. |
| `piko.WithEmailDeadLetterQueue(queue)` | Persists undeliverable emails. |

## See also

- [How to email templates](../how-to/email-templates.md) for composing PK email templates.
- [About email rendering](../explanation/about-email-rendering.md) for why emails reuse the PK substrate.
- [PML components reference](pml-components.md) for the email tag vocabulary.
- [Premailer reference](premailer.md) for the CSS-inlining and validation pipeline.
- [Scenario 026: email contact form](../../examples/scenarios/026_email_contact/) for a runnable example.
