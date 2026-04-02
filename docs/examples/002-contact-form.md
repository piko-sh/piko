---
title: "002: Contact form"
description: Form submission with server-side actions and validation
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 30
---

# 002: Contact form

A contact form that submits data to a server-side action, validates the input, and displays the result, all without a full page reload.

## What this demonstrates

- **Server actions**: defining a Go struct that Piko exposes as a callable endpoint; `piko.ActionMetadata` must be embedded in every action struct
- **Form data mapping**: HTML `name` attributes map to Go struct fields via `json` tags
- **Server-side validation**: `validate` tags are processed before `Call` is invoked; if validation fails, the action never executes
- **`p-on:submit.prevent`**: intercepting form submission and calling an action instead
- **`$form`**: the built-in variable that serialises form fields for the action; handles serialisation automatically. No manual form value reading needed
- Actions return a struct and an error, just like `Render`

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

The action defines three types:

- **`SubmitAction`**: embeds `piko.ActionMetadata` for request context
- **`SubmitInput`**: input struct with `json` tags for field mapping and `validate` tags for rules
- **`SubmitResponse`**: data returned to the client as JSON

The action is named by convention: `action.contact.Submit` comes from the package directory (`contact/`) and the struct name minus the `Action` suffix.

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
