---
title: How to handle form submissions with actions
description: Write a server action that validates form input, flashes toasts, and updates partials.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 10
---

# How to handle form submissions with actions

This guide walks through writing a server action for a form that accepts validated input, emits client-side feedback, and updates related partials. See the [server-actions reference](../../reference/server-actions.md) for the full API.

## Create the action file

Place action files in sub-packages under `actions/`. The generator maps the package name to the URL segment and the struct name to the action method.

```
actions/
    customer/
        upsert.go      # UpsertAction -> /_piko/actions/customer.Upsert
        delete.go      # DeleteAction -> /_piko/actions/customer.Delete
```

The action name is `<package>.<StructName minus the trailing "Action">`. `UpsertAction` in package `customer` becomes `customer.Upsert`. If you name the struct `CustomerUpsertAction` the action name is `customer.CustomerUpsert`. The generator strips only the trailing `Action` suffix.

Create `actions/customer/upsert.go`:

```go
package customer

import (
    "fmt"

    "piko.sh/piko"
)

type UpsertInput struct {
    ID    *int64 `json:"id"`
    Name  string `json:"name"  validate:"required"`
    Email string `json:"email" validate:"required,email"`
}

type UpsertResponse struct {
    ID     int64  `json:"id"`
    Action string `json:"action"`
}

type UpsertAction struct {
    piko.ActionMetadata
}

func (a *UpsertAction) Call(input UpsertInput) (UpsertResponse, error) {
    var customer *Customer
    var err error

    if input.ID == nil {
        customer, err = createCustomer(input.Name, input.Email)
    } else {
        customer, err = updateCustomer(*input.ID, input.Name, input.Email)
    }
    if err != nil {
        return UpsertResponse{}, fmt.Errorf("saving customer: %w", err)
    }

    verb := "created"
    if input.ID != nil {
        verb = "updated"
    }

    a.Response().AddHelper("showToast", "Customer "+verb, "success")
    a.Response().AddHelper("closeModal")
    a.Response().AddHelper("reloadPartial", "#customer-list")

    return UpsertResponse{ID: customer.ID, Action: verb}, nil
}
```

`ID` is a pointer so the caller can omit it to mean "create". The `validate:"required"` tags are metadata used by code generation and tooling. They do not run automatically before `Call`. Validate inside `Call` and return `piko.ValidationField` / `piko.NewValidationError` for field errors (see [Return field-level validation errors](#return-field-level-validation-errors) below).

### Alternative: Individual `Call` parameters

For short calls, drop the struct: `Call(id *int64, name string, email string)`. Use the input struct for forms (TS type matches the form shape, tags document fields, adding a field is local). Use individual parameters for one or two arguments such as `Delete(id int64)`. `Call` cannot take a `context.Context`. Reach for the request context with `a.Ctx()` inside the body. See [about the action protocol](../../explanation/about-the-action-protocol.md) and [server actions reference](../../reference/server-actions.md) for the full discussion.

## Return field-level validation errors

For validation beyond what struct tags can express, return `piko.ValidationField` from `Call`. See [errors reference](../../reference/errors.md) for every error constructor and how each maps to an HTTP status.

```go
func (a *SignupAction) Call(input SignupInput) (SignupResponse, error) {
    if isEmailTaken(input.Email) {
        return SignupResponse{}, piko.ValidationField("email", "This email is already registered")
    }

    return SignupResponse{OK: true}, nil
}
```

For multiple fields at once, use `piko.NewValidationError`:

```go
return SignupResponse{}, piko.NewValidationError(map[string]string{
    "email":    "Invalid format",
    "password": "Must be at least 10 characters",
})
```

The dispatch layer returns HTTP 422 (Unprocessable Content) and the frontend maps each error to the matching form input.

## Regenerate after adding an action

Run your project's scaffolded generator so the dispatch code picks up the new struct. `piko new` creates a `cmd/generator/main.go` entry point inside each scaffolded project. Piko itself does not ship a top-level `generate` command.

```bash
go run ./cmd/generator/main.go all
```

Adjust the path to match wherever your project keeps its generator entry point. After regenerating, Piko mounts the action at `/_piko/actions/customer.Upsert` (POST).

## Submit from a PK template

Bind `p-on:submit.prevent` on the form to a handler function, and call the action from that handler. The `$form` expression passes the current form's data handle to the handler:

```html
<template>
  <form id="customer-form" p-on:submit.prevent="handleSubmit($event, $form)">
    <input type="hidden" name="id" value="{{ state.CustomerID }}" />
    <input type="text" name="name" required />
    <input type="email" name="email" required />
    <button type="submit">Save</button>
  </form>
</template>

<script lang="ts">
async function handleSubmit(event: SubmitEvent, form: FormDataHandle): Promise<void> {
    try {
        const data = await action.customer.Upsert(form).call();
        // `data` is the typed response returned from Call().
        console.log("saved", data.id);
    } catch (err) {
        console.error("save failed", err);
    }
}
</script>
```

The `action.customer.Upsert(form)` expression builds the call from the current form's fields. Calling `.call()` posts the form-encoded data to the action endpoint. Piko renders validation errors inline next to the matching field without a full-page reload.

## Enable toasts and modals

`redirect` ships in the frontend core and works without any extra wiring. `showToast` lives in the toasts module. `closeModal` and `reloadPartial` both live in the modals module. Enable them at bootstrap when you use those helpers:

```go
ssr := piko.New(
    piko.WithFrontendModule(piko.ModuleToasts),
    piko.WithFrontendModule(piko.ModuleModals),
)
```

Without those modules, calls to `showToast`, `closeModal`, and `reloadPartial` are silently ignored on the client.

Note that `reloadPartial` takes a CSS selector, not a bare ID. Pass `"#customer-list"` to target an element with `id="customer-list"`, or use any other selector such as `[data-table="products"]`.

## Authentication: set a session cookie

```go
func (a LoginAction) Call(email string, password string) (LoginResponse, error) {
    user, err := authenticateUser(email, password)
    if err != nil {
        a.Response().AddHelper("showToast", "Invalid email or password", "error")
        return LoginResponse{}, nil
    }

    session, err := createSession(user.ID)
    if err != nil {
        return LoginResponse{}, fmt.Errorf("creating session: %w", err)
    }

    a.Response().SetCookie(piko.SessionCookie("pp_session", session.ID, session.ExpiresAt))
    a.Response().AddHelper("redirect", "/dashboard")

    return LoginResponse{UserID: user.ID}, nil
}
```

For logout, clear the cookie:

```go
a.Response().SetCookie(piko.ClearCookie("pp_session"))
a.Response().AddHelper("redirect", "/")
```

During local development over HTTP, use `piko.SessionCookieInsecure` and `piko.ClearCookieInsecure` to avoid the browser rejecting cookies without the `Secure` flag.

## Handling application errors

Wrap internal errors with context so logs are informative. The dispatch layer returns a generic failure response to the client:

```go
func (a SaveAction) Call(email string, data string) (SaveResponse, error) {
    if err := saveToDatabase(data); err != nil {
        if isUniqueConstraintError(err) {
            a.Response().AddHelper("showToast", "Item already exists", "error")
            return SaveResponse{}, piko.ValidationField("email", "This email is already registered")
        }
        return SaveResponse{}, fmt.Errorf("saving to database: %w", err)
    }

    a.Response().AddHelper("showToast", "Saved", "success")
    return SaveResponse{OK: true}, nil
}
```

## See also

- [Server actions reference](../../reference/server-actions.md) for the full API.
- [How to streaming with SSE](streaming-with-sse.md) for long-running actions.
- [How to cache action responses](caching.md) for response caching.
- [How to rate-limit an action](rate-limiting.md) for per-IP and per-user limits.
- [How to set resource limits on an action](resource-limits.md) for body-size, timeout, and concurrency caps.
- [How to testing](../testing.md) for unit-testing actions.
- [Scenario 002: contact form](../../../examples/scenarios/002_contact_form/) for a runnable end-to-end example.
