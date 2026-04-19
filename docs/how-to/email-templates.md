---
title: How to write email templates
description: Compose transactional emails with PML, send them through the email service, and test the output.
nav:
  sidebar:
    section: "how-to"
    subsection: "services"
    order: 110
---

# How to write email templates

This guide shows how to author responsive email templates using the Piko mail language (PML), render them with typed props, and send them through the email service. See the [PML components reference](../reference/pml-components.md) for every tag and the [premailer reference](../reference/premailer.md) for the CSS-inlining pipeline.

## Project layout

Email templates live under `emails/` at the project root (or inside `src/emails/` if you use a `src/` layout):

```
emails/
  welcome.pk
  password-reset.pk
  order-confirmation.pk
  layout.pk           # shared header/footer
```

Each `.pk` file is a typed template with its own props struct. The generator compiles them into Go functions the same way it compiles page templates.

## Basic template structure

```piko
<template>
  <pml-container>
    <pml-row>
      <pml-col>
        <pml-img
          src="assets/logo.png"
          width="200px"
          alt="Company logo"
          align="center"
        />
      </pml-col>
    </pml-row>

    <pml-row padding="30px 20px">
      <pml-col>
        <pml-p font-size="20px" align="center">
          Welcome, {{ props.Name }}
        </pml-p>

        <pml-p padding="20px 0">
          Thanks for signing up. Click the button below to activate your account.
        </pml-p>

        <pml-button
          href="{{ props.ActivationURL }}"
          background-color="#6F47EB"
          color="#ffffff"
        >
          Activate account
        </pml-button>
      </pml-col>
    </pml-row>
  </pml-container>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    Name          string `prop:"name"`
    ActivationURL string `prop:"activation_url"`
}

type Response struct {
    Name          string
    ActivationURL string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        Name:          props.Name,
        ActivationURL: props.ActivationURL,
    }, piko.Metadata{
        Title: "Welcome",
    }, nil
}
</script>

<style>
  .button-hover:hover { background-color: #5936C7 !important; }
  @media only screen and (max-width: 480px) {
    .mobile-hide { display: none !important; }
  }
</style>
```

The premailer processes the `<style>` block. It writes inlineable rules onto the matched elements, and it places pseudo-classes plus `@media` queries in a `<style>` block at the bottom of the `<body>`, where Gmail preserves them.

## Send from an action

```go
package actions

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/email"
)

type WelcomeProps struct {
    Name          string
    ActivationURL string
}

type SendWelcomeAction struct {
    piko.ActionMetadata
}

func (a *SendWelcomeAction) Call(userID int64) error {
    user, err := loadUser(a.Ctx(), userID)
    if err != nil {
        return err
    }

    builder, err := email.NewTemplatedEmailBuilderFromDefault[WelcomeProps]()
    if err != nil {
        return err
    }

    return builder.
        To(user.Email).
        Subject("Welcome to MyApp").
        Props(WelcomeProps{
            Name:          user.Name,
            ActivationURL: "https://myapp.example.com/activate/" + user.ActivationToken,
        }).
        BodyTemplate("emails/welcome.pk").
        Do(a.Ctx())
}
```

The `[WelcomeProps]` type parameter ties the builder to the same struct the template declares, so the compiler catches any props-shape mismatch.

## Register an email provider

Configure at bootstrap:

```go
package main

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/email/email_provider_smtp"
)

func main() {
    ssr := piko.New(
        piko.WithEmailProvider("smtp", email_provider_smtp.New(email_provider_smtp.Config{
            Host:     "smtp.example.com",
            Port:     587,
            Username: "noreply@example.com",
            Password: mustSecret("SMTP_PASSWORD"),
        })),
        piko.WithDefaultEmailProvider("smtp"),
    )
    ssr.Run()
}
```

Swap the provider for SES, SendGrid, Mailgun, Postmark, or Resend by changing the registration. See the [email API reference](../reference/email-api.md) for all providers.

## Grouping sections with a shared background

Use `pml-container` when a block of rows needs one background colour without double padding:

```piko
<pml-container background-color="#f5f5f5" padding="20px">
  <pml-row>
    <pml-col>
      <pml-p font-weight="bold">Your order</pml-p>
    </pml-col>
  </pml-row>
  <pml-row>
    <pml-col>
      <pml-p>Details here.</pml-p>
    </pml-col>
  </pml-row>
</pml-container>
```

## Multi-column layouts

Columns split the row evenly when you omit `width`:

```piko
<pml-row padding="20px">
  <pml-col>
    <pml-img src="assets/product-1.jpg" alt="Product 1" />
    <pml-p font-weight="bold">Product One</pml-p>
    <pml-p color="#666">£29.99</pml-p>
  </pml-col>
  <pml-col>
    <pml-img src="assets/product-2.jpg" alt="Product 2" />
    <pml-p font-weight="bold">Product Two</pml-p>
    <pml-p color="#666">£39.99</pml-p>
  </pml-col>
</pml-row>
```

Columns stack vertically below the mobile breakpoint (480 px by default). To keep them side-by-side on mobile, wrap them in `<pml-no-stack>`.

## Hero banners

`pml-hero` layers content over a background image:

```piko
<pml-hero
  mode="fixed-height"
  height="300px"
  background-url="assets/hero.jpg"
  background-color="#000000"
  vertical-align="middle"
>
  <pml-p align="center" color="#ffffff" font-size="36px" font-weight="bold">
    Summer sale
  </pml-p>
  <pml-button href="https://example.com/shop" background-color="#ff4444">
    Shop now
  </pml-button>
</pml-hero>
```

The compiler generates an Outlook VML fallback automatically so the background appears in Outlook desktop.

## High-DPI images

Use `densities` for automatic retina support:

```piko
<pml-img
  src="assets/banner.jpg"
  densities="1x 2x"
  width="600px"
  alt="Banner"
/>
```

The compiler generates a `srcset` with the 1x and 2x variants. Apple Mail and recent Gmail render the 2x image on retina screens. Outlook falls back to the base `src`.

## CID-embedded images

Piko attaches image assets referenced with `src="assets/..."` to the outgoing email as CID-embedded parts automatically. During PML transformation the compiler rewrites the `<img>` tag's `src` to `cid:<generated-id>`. No client needs to fetch the image over HTTPS, which improves deliverability and privacy.

The compiler leaves external URLs (`src="https://..."`) unchanged.

## Preheader text

The preheader is the short preview text shown next to the subject line in most clients. Add a hidden `pml-p`:

```piko
<pml-container>
  <pml-row>
    <pml-col>
      <pml-p css-class="preheader" color="#ffffff" font-size="1px" line-height="1px">
        Your account is now active.
      </pml-p>
    </pml-col>
  </pml-row>
  <!-- rest of the email -->
</pml-container>

<style>
  .preheader { display: none !important; visibility: hidden; opacity: 0; overflow: hidden; }
</style>
```

## Append tracking parameters to links

Configure the premailer to append UTM parameters to every link:

```go
builder.
    WithLinkQueryParams(map[string]string{
        "utm_source":   "transactional",
        "utm_medium":   "email",
        "utm_campaign": "welcome",
    }).
    BodyTemplate("emails/welcome.pk").
    Do(a.Ctx())
```

The premailer walks every `<a href>` in the rendered template and appends the parameters, preserving any existing query string.

## Inline CSS variables with a theme

The build step resolves `var(--colour-primary)` inside the template's `<style>` block against a theme map:

```go
builder.
    WithTheme(map[string]string{
        "--colour-primary":   "#6F47EB",
        "--colour-secondary": "#0ea5e9",
    }).
    BodyTemplate("emails/welcome.pk").
    Do(a.Ctx())
```

Undefined variables surface as premailer diagnostics.

## Plain-text alternative

Every email should include a plain-text version. Compose it alongside the HTML:

```go
builder.
    To(user.Email).
    Subject("Welcome").
    BodyTemplate("emails/welcome.pk").
    BodyPlain("Welcome, " + user.Name + ".\n\nActivate at " + url).
    Do(a.Ctx())
```

Some providers derive plain text from the HTML automatically. Providing it explicitly gives you control over how quotes, links, and lists render.

## Attachments

```go
pdfBytes, err := renderInvoicePDF(a.Ctx(), invoiceID)
if err != nil {
    return err
}

builder.
    Attach(email.Attachment{
        Filename:    "invoice.pdf",
        ContentType: "application/pdf",
        Body:        pdfBytes,
    }).
    Do(a.Ctx())
```

## Preview during development

Add a `Preview` function to the template's script block to register scenarios for the CLI preview tool:

```go
func Preview() []piko.PreviewScenario {
    return []piko.PreviewScenario{
        {
            Name: "default",
            Props: Props{
                Name:          "Alice",
                ActivationURL: "https://example.com/activate/xyz",
            },
        },
        {
            Name: "long name",
            Props: Props{
                Name:          strings.Repeat("A", 64),
                ActivationURL: "https://example.com/activate/xyz",
            },
        },
    }
}
```

Run `piko preview` to open the template in the browser with live reload.

## Common gotchas

The premailer warns on `display: flex` and `display: grid`, so it rejects flexbox and grid. Use `pml-row`/`pml-col` instead.

Gmail strips `<style>` blocks in `<head>`. Piko places leftover rules in a `<style>` block inside `<body>` so they survive.

Outlook ignores `max-width` on tables, so the PML row wraps its table in a `<div style="max-width">` container.

Shorthand `margin` is unreliable in Yahoo and Outlook, so the premailer expands it to longhand properties.

The build step evaluates CSS variables, so any variable referenced at runtime does not exist in email clients.

## See also

- [PML components reference](../reference/pml-components.md) for every tag and attribute.
- [Premailer reference](../reference/premailer.md) for the CSS-inlining and validation pipeline.
- [Email API reference](../reference/email-api.md) for the service that sends rendered templates.
- [About email rendering](../explanation/about-email-rendering.md) for the design rationale.
- [Scenario 026: email contact form](../showcase/026-email-contact.md) for a runnable walk-through.
