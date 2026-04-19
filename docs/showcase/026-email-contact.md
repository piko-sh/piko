---
title: "026: Email contact form"
description: Send transactional emails through an email provider when a visitor submits the contact form.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 460
---

# 026: Email contact form

A contact form whose server action sends an email through the configured email provider and flashes a success toast on the client.

## What this demonstrates

- `WithEmailProvider` with a concrete provider (SMTP for local dev).
- An email template as a PK file with a Go `Render` function that returns the body.
- A server action that calls `piko.GetEmailService().Send(...)`.
- The email dispatcher queue, which retries failed sends.

## Project structure

```text
src/
  cmd/main/main.go      Bootstrap with the email provider.
  pages/
    index.pk            The contact form page.
  emails/
    confirmation.pk     Template sent to the submitter.
    notification.pk     Template sent to the site owner.
  actions/
    contact/
      send.go           Validates input, renders the templates, sends both emails.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/026_email_contact/src/
go mod tidy
air
```

Submit the form on the index page. The scenario prints captured email output to the console, or delivers through whichever provider the scenario uses.

## See also

- [How to email templates](../how-to/email-templates.md).
- [Bootstrap options reference: Email](../reference/bootstrap-options.md#email).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/026_email_contact).
