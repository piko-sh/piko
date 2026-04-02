---
title: Email templates
description: Creating transactional email templates
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 110
---

# Email templates

Create responsive, cross-client compatible email templates using PML (Piko Mail Language) syntax. PML components are a specialised subset of PK templates designed for email client compatibility: they transform to email-safe HTML with automatic CSS inlining and Outlook VML fallbacks. For the full list of PML components, see the PML tag reference.

## Email template structure

Email templates use PML components that are designed specifically for email client compatibility. The primary layout components are:

- `<pml-container>` - Groups multiple rows with a shared background
- `<pml-row>` - Creates horizontal sections
- `<pml-col>` - Column within a row for content placement
- `<pml-button>` - Email-safe call-to-action buttons
- `<pml-img>` - Responsive images with CID embedding
- `<pml-p>` - Styled paragraphs

### Basic email template

```piko
<!-- emails/welcome.pk -->
<template>
  <pml-row background-color="#f8f9fa" padding="20px 0">
    <pml-col>
      <pml-img
        src="assets/logo.png"
        profile="email-logo"
        alt="Logo"
        width="200px"
        align="center"
      />
    </pml-col>
  </pml-row>

  <pml-row padding="30px 20px">
    <pml-col>
      <pml-p font-size="24px" color="#333" align="center">
        Welcome {{ state.UserName }}!
      </pml-p>

      <pml-p padding="20px 0">
        Thanks for joining us. We're excited to have you on board.
      </pml-p>

      <pml-button
        href="{{ state.LoginURL }}"
        background-color="#007bff"
        color="#ffffff"
        border-radius="5px"
      >
        Get Started
      </pml-button>
    </pml-col>
  </pml-row>
</template>

<script lang="go">
package emails

import "piko.sh/piko"

type Props struct {
    UserName string `prop:"user_name"`
    LoginURL string `prop:"login_url"`
}

type Response struct {
    UserName string
    LoginURL string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        UserName: props.UserName,
        LoginURL: props.LoginURL,
    }, piko.Metadata{}, nil
}
</script>
```

## Using pml-container for grouped sections

Use `<pml-container>` when you need multiple rows to share a common background:

```piko
<template>
  <pml-container background-color="#eeeeee" padding="20px 10px">
    <pml-row background-color="#ffffff" padding="15px">
      <pml-col>
        <pml-p>First section inside the container.</pml-p>
      </pml-col>
    </pml-row>

    <pml-row background-color="#fafafa" padding="15px">
      <pml-col>
        <pml-p>Second section, stacked vertically below the first.</pml-p>
      </pml-col>
    </pml-row>
  </pml-container>
</template>
```

## Sending emails

Piko provides a fluent builder API for sending emails. There are two approaches:

### Simple email builder

For emails with inline HTML/plain text content:

```go
import "piko.sh/piko/wdk/email"

err := email.NewEmail().
    To("user@example.com").
    Subject("Welcome!").
    BodyHTML("<h1>Hello!</h1><p>Welcome to our service.</p>").
    BodyPlain("Hello! Welcome to our service.").
    Send(ctx)
```

### Templated email builder

For type-safe emails using .pk files:

```go
import "piko.sh/piko/wdk/email"

type WelcomeProps struct {
    UserName string
    LoginURL string
}

err := email.NewTemplatedEmail[WelcomeProps]().
    To("user@example.com").
    Subject("Welcome!").
    WithProps(WelcomeProps{
        UserName: "John",
        LoginURL: "https://app.com/login",
    }).
    WithRequest(r).
    BodyTemplate("emails/welcome.pk").
    Send(ctx)
```

### Builder methods

Both builders support these common methods:

| Method | Description |
|--------|-------------|
| `To(addresses ...string)` | Add recipient email addresses |
| `Cc(addresses ...string)` | Add CC recipients |
| `Bcc(addresses ...string)` | Add BCC recipients |
| `From(address string)` | Set sender address |
| `Subject(subject string)` | Set email subject |
| `WithAttachment(filename, mimeType string, content []byte)` | Add file attachment |
| `WithProvider(name string)` | Use a specific email provider |
| `Immediate()` | Bypass dispatcher queue, send immediately |
| `Send(ctx context.Context)` | Send the email |

The templated builder adds:

| Method | Description |
|--------|-------------|
| `WithProps(props T)` | Set type-safe template properties |
| `WithRequest(r *http.Request)` | Set HTTP request context |
| `BodyTemplate(path string)` | Set the template file path |

## Multi-column layouts

Create side-by-side columns for product grids or comparison layouts:

```piko
<template>
  <pml-row padding="20px 0">
    <pml-col width="50%">
      <pml-img src="assets/product1.jpg" width="280px" alt="Product 1" />
      <pml-p font-weight="bold">Product One</pml-p>
      <pml-p color="#666">$29.99</pml-p>
    </pml-col>

    <pml-col width="50%">
      <pml-img src="assets/product2.jpg" width="280px" alt="Product 2" />
      <pml-p font-weight="bold">Product Two</pml-p>
      <pml-p color="#666">$39.99</pml-p>
    </pml-col>
  </pml-row>
</template>
```

## Image handling

Email images are automatically processed and embedded:

### CID embedding

Local images are embedded using Content-ID (CID) for reliable display without external requests:

```piko
<pml-img
  src="assets/logo.png"
  profile="email-logo"
  width="200px"
  alt="Company Logo"
/>
<!-- Renders as: <img src="cid:asset_abc123" ... /> -->
```

### High-DPI support

Use the `densities` attribute for crisp images on Retina displays:

```piko
<pml-img
  src="assets/banner.jpg"
  profile="email-hero"
  densities="x1 x2"
  width="600px"
  alt="Banner"
/>
```

### Responsive images

Make images scale on mobile devices:

```piko
<pml-img
  src="assets/hero.jpg"
  width="600px"
  fluid-on-mobile="true"
  alt="Hero image"
/>
```

## Styling with CSS

Add a `<style>` block for CSS that gets inlined automatically:

```piko
<template>
  <pml-row>
    <pml-col>
      <table class="quote-details">
        <tr>
          <td><strong>Quote #:</strong></td>
          <td>{{ state.QuoteID }}</td>
        </tr>
      </table>
    </pml-col>
  </pml-row>
</template>

<style>
table.quote-details {
  width: 100%;
  border-collapse: collapse;
  margin: 20px 0;
}

table.quote-details td {
  padding: 10px;
  border-bottom: 1px solid #e5e7eb;
}
</style>
```
