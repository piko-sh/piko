---
title: "002: Contact form"
description: Form submission with server-side actions and validation
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 30
---

# 002: Contact form

A contact form that submits data to a server-side action, validates the input, and displays the result, all without a full page reload.

## What this demonstrates

A server action is a Go struct that Piko exposes as a callable endpoint. Every action struct must embed `piko.ActionMetadata`. HTML `name` attributes map to Go struct fields through `json` tags. Piko processes `validate` tags before calling `Call`. If validation fails, the action never runs.

The template uses `p-on:submit.prevent` to intercept form submission and call an action instead. The built-in `$form` variable serialises form fields for the action, so the template never has to read form values manually. Actions return a struct and an error, just like `Render`.

## Project structure

```text
src/
  pages/
    contact.pk                   The page: form template + Go script + CSS
  actions/
    contact/
      submit.go                  The server action that processes the form
```

## How it works

The form uses `p-on:submit.prevent="action.contact.Submit($form)"` to intercept submission. `$form` serialises all named fields automatically.

The action defines three types. `SubmitAction` embeds `piko.ActionMetadata` for request context. `SubmitInput` is the input struct with `json` tags for field mapping and `validate` tags for rules. `SubmitResponse` holds the data that Piko returns to the client as JSON.

The action name follows convention. `action.contact.Submit` comes from the package directory (`contact/`) and the struct name minus the `Action` suffix.

```go
type SubmitInput struct {
    Name    string `json:"name"    validate:"required"`
    Email   string `json:"email"   validate:"required,email"`
    Message string `json:"message" validate:"required,min=10"`
}
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/002_contact_form/src/
go mod tidy
air
```

## See also

- [Server actions reference](../reference/server-actions.md).
- [How to forms](../how-to/actions/forms.md).
