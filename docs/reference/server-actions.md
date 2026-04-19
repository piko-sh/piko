---
title: Server actions
description: Action types, dispatch surface, response helpers, cookie helpers, and error semantics.
nav:
  sidebar:
    section: "reference"
    subsection: "runtime"
    order: 30
---

# Server actions

An action is a Go struct that handles a client-triggered operation such as a form submission or an RPC call. The struct embeds `piko.ActionMetadata` and implements a `Call` method with typed parameters and a typed return value. The generator registers every action struct it finds and mounts it at a conventional URL. This page describes the types and surface. For task recipes see the how-to guides on [forms](../how-to/actions/forms.md) and [streaming with SSE](../how-to/actions/streaming-with-sse.md).

[About the action protocol](../explanation/about-the-action-protocol.md) walks through the dispatch sequence, including CSRF, parameter binding, rate limiting, and the fork between `Call` and `StreamProgress`.

## Action struct

An action is any exported struct that embeds `piko.ActionMetadata`:

```go
package actions

import "piko.sh/piko"

type ContactSubmitAction struct {
    piko.ActionMetadata
}

func (a ContactSubmitAction) Call(name string, email string, message string) (ContactResponse, error) {
    return ContactResponse{OK: true}, nil
}
```

The generated dispatch code binds incoming fields to `Call` arguments, validates them against the parameter types, and returns the response value as JSON.

> **Note:** The method name `Call` carries meaning. The generator looks for it by exact name; an action with `Handle` or `Run` instead compiles but never mounts.

## `ActionMetadata`

Embedded field. Exposes three methods that operate on the current request.

| Method | Type | Purpose |
|---|---|---|
| `a.Ctx()` | `context.Context` | Request context. Cancellation propagates to any downstream IO. |
| `a.Request()` | `piko.RequestMetadata` | URL, method, headers, query parameters, form data, session, cookies. |
| `a.Response()` | `piko.ResponseBuilder` | Set cookies, add client-side helpers, attach field errors. |

The dispatch layer validates CSRF tokens before `Call` runs.

## `Call` signature

Two shapes are valid and interchangeable:

**Individual parameters**:

```go
func (a *CustomerUpsertAction) Call(id *int64, name string, email string) (UpsertResponse, error) {
    return UpsertResponse{}, nil
}
```

**Input struct** (idiomatic for actions with three or more fields or struct-level validation):

```go
type SignupInput struct {
    Username string  `json:"username" validate:"required,min=3"`
    Email    string  `json:"email"    validate:"required,email"`
    Password string  `json:"password" validate:"required,min=10"`
    Age      int     `json:"age"`
    Website  *string `json:"website"`
}

func (a *SignupAction) Call(input SignupInput) (SignupResponse, error) {
    return SignupResponse{UserID: 42}, nil
}
```

The two are equivalent from the dispatch layer's perspective. Whether the generator sees `Call(name string, email string)` or `Call(input SignupInput)`, it emits a TypeScript call shape on the client that mirrors the Go parameters. For guidance on when each form fits best, see [about the action protocol](../explanation/about-the-action-protocol.md).

**Direct parameter mapping.** The generator maps every Go parameter to a TS/JS argument position by one-for-one correspondence. For this reason `Call` signatures cannot accept `context.Context`. The embedded `piko.ActionMetadata` exposes the request context via `a.Ctx()`. Including it in the parameter list would break the direct mapping.

> **Note:** Unlike an `http.Handler`, a `Call` signature does not accept `context.Context`. The parameter list maps one-for-one onto client arguments, and a context parameter would shift every index. Reach the request context through `a.Ctx()` on the embedded `ActionMetadata`.

The dispatch layer parses the request body and maps fields to parameters or struct fields by their JSON tag (or by Go field name, for unnamed parameters). It applies `validate` rules and rejects requests whose fields fail conversion or validation before calling the method body.

Pointer parameters (`*int64`, `*string`) mark a value as optional. A value type with a zero value is still considered set.

## Typed responses

The first return value is the action's own response struct, serialised to JSON. There is no generic `ActionResponse` wrapper.

```go
type SignupResponse struct {
    UserID int    `json:"user_id"`
    Name   string `json:"name"`
}
```

The second return value, `error`, signals failure. See [Error handling](#error-handling).

## `Response()` helpers

`a.Response()` returns a builder that chains client-side effects onto the response:

| Builder method | Arguments | Effect |
|---|---|---|
| `AddHelper(name, args...)` | helper name and its arguments | Emits a helper call the frontend runtime dispatches |
| `SetCookie(cookie)` | `*http.Cookie` | Attaches a `Set-Cookie` header |

### Built-in helpers

| Helper | Arguments | Effect | Requires |
|---|---|---|---|
| `showToast` | `message, level, [durationMs]` | Displays a toast notification (`success`, `error`, `warning`, `info`) | `piko.ModuleToasts` |
| `closeModal` | none | Closes the current modal | `piko.ModuleModals` |
| `reloadPartial` | `alias or selector` | Refetches and re-renders a partial | `piko.ModuleModals` |
| `redirect` | `url` | Client-side navigation | always enabled |

Enable modules at bootstrap:

```go
ssr := piko.New(
    piko.WithFrontendModule(piko.ModuleToasts),
    piko.WithFrontendModule(piko.ModuleModals),
)
```

## Cookie helpers

| Function | Description |
|---|---|
| `piko.SessionCookie(name, value, expires)` | Secure session cookie: `HttpOnly`, `Secure`, `SameSite=Lax`. |
| `piko.SessionCookieInsecure(name, value, expires)` | Session cookie without the `Secure` flag, for local development over HTTP. |
| `piko.ClearCookie(name)` | Expires the named cookie. |
| `piko.ClearCookieInsecure(name)` | Clear-cookie variant without the `Secure` flag. |
| `piko.Cookie(name, value, maxAge, ...opts)` | Customisable cookie with functional options. |

### Cookie options

The `piko.Cookie` constructor accepts functional options:

| Option | Effect |
|---|---|
| `piko.WithPath(path)` | Sets the cookie path. |
| `piko.WithDomain(domain)` | Sets the cookie domain. |
| `piko.WithInsecure()` | Drops the `Secure` flag. |
| `piko.WithJavaScriptAccess()` | Drops the `HttpOnly` flag. |
| `piko.WithSameSiteStrict()` | Sets `SameSite=Strict`. |
| `piko.WithSameSiteNone()` | Sets `SameSite=None` (implies `Secure`). |

## Error handling

`Call` returns an `error`. The dispatch layer treats different error shapes differently.

| Error shape | HTTP status | Behaviour |
|---|---|---|
| `nil` | 200 | Response body is the typed response, JSON-encoded. |
| `piko.ValidationField(field, message)` | 400 | Field-level validation error rendered into the form. |
| `piko.NewValidationError(fields)` | 400 | Multi-field validation error. |
| Any other `error` | 500 | Piko logs the error with full detail, and the client sees a generic failure message. |

Internal errors are never exposed to the client verbatim. Wrap them with `fmt.Errorf` to preserve context for logs.

## Discovery and routing

The generator scans `actions/**/*.go` at build time and registers every exported struct that embeds `piko.ActionMetadata`. The routing convention is:

```
action package     URL
-----------------  -------------------------
actions/customer   /actions/customer.Upsert
actions/auth       /actions/auth.Login
```

The package name becomes the URL segment and the action struct name (without the `Action` suffix) becomes the method name. Run `piko generate` after adding or renaming an action.

## See also

- [How to forms](../how-to/actions/forms.md) - end-to-end form-submission recipes.
- [How to streaming with SSE](../how-to/actions/streaming-with-sse.md) - long-lived streaming responses.
- [How to override an action's HTTP method](../how-to/actions/method-override.md) - `GET`, `PUT`, `PATCH`, `DELETE`, `HEAD`, `OPTIONS`.
- [How to cache action responses](../how-to/actions/caching.md) - TTL, vary-by-header, custom keys.
- [How to rate-limit an action](../how-to/actions/rate-limiting.md) - per-IP, per-user, per-session.
- [How to set resource limits on an action](../how-to/actions/resource-limits.md) - body-size, timeout, concurrency, SSE duration.
- [How to testing](../how-to/testing.md) - unit-testing actions with `pikotest`.

**Used in**: [Scenario 002: contact form](../showcase/002-contact-form.md), [Scenario 010: progress tracker](../showcase/010-progress-tracker.md), [Scenario 016: cached API](../showcase/016-cached-api.md), [Scenario 017: rate-limited API](../showcase/017-rate-limited-api.md).

Integration tests: [`tests/integration/e2e_browser/actions_test.go`](https://github.com/piko-sh/piko/blob/master/tests/integration/e2e_browser/actions_test.go), [`tests/integration/cache_rendering/`](https://github.com/piko-sh/piko/tree/master/tests/integration/cache_rendering).
