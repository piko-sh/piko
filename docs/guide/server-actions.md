---
title: Server actions
description: Complete reference for server actions in Piko
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 70
---

# Server actions

Server actions handle form submissions and client-triggered operations in Piko. They provide type-safe, validated request handling with typed responses and client-side helper invocations.

## Action interface

Every action embeds `piko.ActionMetadata` and implements a `Call` method with typed parameters and a typed return value:

```go
type ContactSubmitAction struct {
    piko.ActionMetadata
}

func (a ContactSubmitAction) Call(name string, email string, message string) (ContactResponse, error) {
    // Parameters are populated and validated by the generated dispatch code.
    // Process the action using name, email, message directly.
    return ContactResponse{OK: true}, nil
}
```

In v2, there is no separate `piko.Action` interface to satisfy. The generated dispatch code handles parameter binding, validation, and routing automatically.

## Basic action structure

```go
package actions

import (
    "piko.sh/piko"
)

type ContactResponse struct {
    OK bool `json:"ok"`
}

type ContactSubmitAction struct {
    piko.ActionMetadata
}

func (a ContactSubmitAction) Call(name string, email string, message string) (ContactResponse, error) {
    // name, email, and message are already validated and populated
    // by the generated dispatch code.

    a.Response().AddHelper("showToast", "Thank you for your message!", "success")

    return ContactResponse{OK: true}, nil
}
```

## ActionMetadata

Every action struct must embed `piko.ActionMetadata`. This provides access to the request context, session data, and response helpers:

```go
type ActionMetadata struct {
    // Internal fields managed by the dispatch layer.
}
```

The embedded metadata gives your action these methods:

- `a.Ctx()` returns the request context (`context.Context`)
- `a.Request()` returns session and request metadata
- `a.Response()` returns a response builder for adding helpers, cookies, and errors

CSRF tokens are handled automatically by the dispatch layer. You do not need to manage them manually.

## Typed parameters

In v2, action parameters are defined directly on the `Call` method signature rather than as struct fields with JSON tags:

```go
// v2: Parameters are method arguments
func (a SignupAction) Call(
    username string,
    email string,
    password string,
    age int,
    website string,
    country string,
) (SignupResponse, error) {
    // All parameters are populated by the generated dispatch code.
    // Validation is handled through the dispatch layer.
    return SignupResponse{UserID: 42}, nil
}
```

The generated dispatch code maps form fields and JSON properties to the `Call` parameters by name, performs type conversion, and validates inputs before your action code runs.

## Typed responses

Instead of returning a generic `piko.ActionResponse` with a status code, actions return your own typed response struct:

```go
type SignupResponse struct {
    UserID int    `json:"user_id"`
    Name   string `json:"name"`
}

func (a SignupAction) Call(username string, email string) (SignupResponse, error) {
    user, err := createUser(username, email)
    if err != nil {
        return SignupResponse{}, err
    }

    return SignupResponse{
        UserID: user.ID,
        Name:   user.Name,
    }, nil
}
```

## Response helpers

Use `a.Response()` to add client-side helpers, set cookies, and attach field-level errors:

```go
func (a CustomerSaveAction) Call(name string, email string) (CustomerResponse, error) {
    customer, err := saveCustomer(name, email)
    if err != nil {
        return CustomerResponse{}, err
    }

    a.Response().AddHelper("showToast", "Customer saved!", "success")
    a.Response().AddHelper("closeModal")
    a.Response().AddHelper("reloadPartial", "customer-list")

    return CustomerResponse{ID: customer.ID}, nil
}
```

### Validation errors

Return a `piko.ValidationField` error for field-level validation, or `piko.NewValidationError` for multiple fields:

```go
func (a SaveAction) Call(email string) (SaveResponse, error) {
    if isEmailTaken(email) {
        return SaveResponse{}, piko.ValidationField("email", "This email is already registered")
    }

    // Continue processing...
    return SaveResponse{OK: true}, nil
}
```

## Helpers

Helpers trigger client-side actions after the server response. They require the corresponding frontend modules to be enabled.

### showToast

Display a notification message. Requires `piko.ModuleToasts` to be enabled.

```go
// Success toast (green)
a.Response().AddHelper("showToast", "Saved successfully!", "success")

// Error toast (red)
a.Response().AddHelper("showToast", "Something went wrong", "error")

// Warning toast (yellow)
a.Response().AddHelper("showToast", "Please review your input", "warning")

// Info toast (blue)
a.Response().AddHelper("showToast", "New updates available", "info")

// With custom duration (milliseconds)
a.Response().AddHelper("showToast", "Quick message", "info", 2000)
```

### closeModal

Close the current modal. Requires `piko.ModuleModals` to be enabled.

```go
a.Response().AddHelper("closeModal")
```

### reloadPartial

Reload a specific partial component. Requires `piko.ModuleModals` to be enabled.

```go
// By partial alias
a.Response().AddHelper("reloadPartial", "customer-list")

// By CSS selector
a.Response().AddHelper("reloadPartial", "[data-partial=\"notifications\"]")
```

### redirect

Navigate to a different page:

```go
a.Response().AddHelper("redirect", "/dashboard")
a.Response().AddHelper("redirect", "/customers/123")
```

### Enabling frontend modules

Enable helpers in your application setup:

```go
ssr := piko.New(
    piko.WithFrontendModule(piko.ModuleToasts),
    piko.WithFrontendModule(piko.ModuleModals),
)
```

## Cookie management

Piko provides helper functions for creating secure cookies in action responses.

### Session cookies

```go
import "time"

func (a LoginAction) Call(email string, password string) (LoginResponse, error) {
    user, err := authenticateUser(email, password)
    if err != nil {
        return LoginResponse{}, err
    }

    session, err := createSession(user.ID)
    if err != nil {
        return LoginResponse{}, err
    }

    a.Response().SetCookie(piko.SessionCookie("pp_session", session.ID, session.ExpiresAt))
    a.Response().AddHelper("redirect", "/dashboard")

    return LoginResponse{UserID: user.ID}, nil
}
```

### Clearing cookies (logout)

```go
func (a LogoutAction) Call() (LogoutResponse, error) {
    a.Response().SetCookie(piko.ClearCookie("pp_session"))
    a.Response().AddHelper("redirect", "/")

    return LogoutResponse{}, nil
}
```

### Cookie helper functions

| Function | Description |
|----------|-------------|
| `piko.SessionCookie(name, value, expires)` | Secure session cookie (HttpOnly, Secure, SameSite=Lax) |
| `piko.SessionCookieInsecure(name, value, expires)` | Session cookie for local dev (no Secure flag) |
| `piko.ClearCookie(name)` | Delete a cookie |
| `piko.ClearCookieInsecure(name)` | Delete cookie for local dev |
| `piko.Cookie(name, value, maxAge, ...opts)` | Customisable cookie with functional options |

### Cookie options

```go
piko.Cookie("preferences", value, 365*24*time.Hour,
    piko.WithPath("/settings"),
    piko.WithDomain("example.com"),
    piko.WithSameSiteStrict(),
)
```

Available options:
- `piko.WithPath(path)` - Set cookie path
- `piko.WithDomain(domain)` - Set cookie domain
- `piko.WithInsecure()` - Allow HTTP (dev only)
- `piko.WithJavaScriptAccess()` - Remove HttpOnly flag
- `piko.WithSameSiteStrict()` - Use SameSite=Strict
- `piko.WithSameSiteNone()` - Use SameSite=None (requires Secure)

## Error handling

### Validation errors

In v2, the generated dispatch code validates parameters before calling your action. Failed validation returns a 400 Bad Request with field-level errors. For business logic validation, return a `piko.ValidationField` or `piko.NewValidationError` error from `Call`.

### Application errors

Return errors from `Call` to signal failures. The dispatch layer logs internal errors and returns an appropriate response to the client:

```go
func (a GetCustomerAction) Call(id int64) (CustomerResponse, error) {
    customer, err := findCustomer(id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            a.Response().AddHelper("showToast", "Customer not found", "error")
            return CustomerResponse{}, nil
        }

        // Internal errors: returned error is logged, not exposed to the client
        return CustomerResponse{}, fmt.Errorf("finding customer %d: %w", id, err)
    }

    return CustomerResponse{
        ID:    customer.ID,
        Name:  customer.Name,
        Email: customer.Email,
    }, nil
}
```

### Error response pattern

```go
func (a SaveAction) Call(email string, data string) (SaveResponse, error) {
    err := saveToDatabase(data)
    if err != nil {
        if isUniqueConstraintError(err) {
            a.Response().AddHelper("showToast", "Item already exists", "error")
            return SaveResponse{}, piko.ValidationField("email", "This email is already registered")
        }

        // Unexpected errors
        return SaveResponse{}, fmt.Errorf("saving to database: %w", err)
    }

    a.Response().AddHelper("showToast", "Saved!", "success")
    return SaveResponse{OK: true}, nil
}
```

## Action discovery

Actions are discovered automatically by the generator. Place your action files in sub-packages under `actions/` and the build step registers them for you. There is no manual `AllActions()` map; simply run the generator after adding new actions.

```text
actions/
    customer/
        upsert.go     # UpsertAction -> action.customer.Upsert
        delete.go     # DeleteAction -> action.customer.Delete
    auth/
        login.go      # LoginAction  -> action.auth.Login
        logout.go     # LogoutAction -> action.auth.Logout
```

Actions are mounted at `/actions/{package}.{name}` by default.

## Common patterns

### CRUD actions

**Create/Update (Upsert):**

```go
type CustomerUpsertResponse struct {
    ID     int64  `json:"id"`
    Action string `json:"action"`
}

type CustomerUpsertAction struct {
    piko.ActionMetadata
}

func (a CustomerUpsertAction) Call(id *int64, name string, email string) (CustomerUpsertResponse, error) {
    var customer *Customer
    var err error

    if id == nil {
        customer, err = createCustomer(name, email)
    } else {
        customer, err = updateCustomer(*id, name, email)
    }

    if err != nil {
        return CustomerUpsertResponse{}, fmt.Errorf("saving customer: %w", err)
    }

    action := "created"
    if id != nil {
        action = "updated"
    }

    a.Response().AddHelper("showToast", "Customer "+action, "success")
    a.Response().AddHelper("closeModal")
    a.Response().AddHelper("reloadPartial", "customer-list")

    return CustomerUpsertResponse{ID: customer.ID, Action: action}, nil
}
```

**Delete:**

```go
type CustomerDeleteResponse struct {
    OK bool `json:"ok"`
}

type CustomerDeleteAction struct {
    piko.ActionMetadata
}

func (a CustomerDeleteAction) Call(id int64) (CustomerDeleteResponse, error) {
    err := deleteCustomer(id)
    if err != nil {
        return CustomerDeleteResponse{}, fmt.Errorf("deleting customer %d: %w", id, err)
    }

    a.Response().AddHelper("showToast", "Customer deleted", "success")
    a.Response().AddHelper("reloadPartial", "customer-list")

    return CustomerDeleteResponse{OK: true}, nil
}
```

### Authentication actions

**Login:**

```go
type LoginResponse struct {
    UserID int64 `json:"user_id"`
}

type LoginAction struct {
    piko.ActionMetadata
}

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
    a.Response().AddHelper("showToast", "Welcome back!", "success")
    a.Response().AddHelper("redirect", "/dashboard")

    return LoginResponse{UserID: user.ID}, nil
}
```

**Logout:**

```go
type LogoutResponse struct{}

type LogoutAction struct {
    piko.ActionMetadata
}

func (a LogoutAction) Call() (LogoutResponse, error) {
    // Optionally invalidate server-side session
    userID := a.Request().Session.UserID
    if userID != "" {
        _ = invalidateSession(userID)
    }

    a.Response().SetCookie(piko.ClearCookie("pp_session"))
    a.Response().AddHelper("redirect", "/")

    return LogoutResponse{}, nil
}
```
## Next steps

- [Advanced actions](/docs/guide/advanced-actions) → Middleware, rate limiting, caching
- [Testing actions](/docs/guide/testing) → Unit testing server actions
