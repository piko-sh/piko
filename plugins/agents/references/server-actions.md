# Server Actions

Use this guide when handling form submissions, CRUD operations, or any server-side action triggered from the client.

## Overview

Actions provide type-safe request handling without writing HTTP handlers. The flow:

1. Template binds a form/button to an action with `action.` prefix
2. Piko serialises the data and sends it to the server
3. The dispatch layer validates, converts types, and calls your `Call` method
4. You return a response and optional client-side helpers (toast, redirect, etc.)

## Basic form + action

### Template

```piko
<template>
  <form p-on:submit.prevent="action.contact_submit()">
    <label for="name">Name</label>
    <input id="name" name="name" type="text" required />

    <label for="email">Email</label>
    <input id="email" name="email" type="email" required />

    <label for="message">Message</label>
    <textarea id="message" name="message" rows="5" required></textarea>

    <button type="submit">Send</button>
  </form>
</template>
```

Key: `p-on:submit.prevent="action.contact_submit()"` prevents default form submission and sends via AJAX. Input `name` attributes map to `Call` method parameters.

### Action file

Create `actions/contact_submit.go`:

```go
package actions

import (
    "fmt"
    "piko.sh/piko"
)

type ContactResponse struct {
    OK bool `json:"ok"`
}

type ContactSubmitAction struct {
    piko.ActionMetadata
}

func (a ContactSubmitAction) Call(name string, email string, message string) (ContactResponse, error) {
    err := sendEmail(email, name, message)
    if err != nil {
        a.Response().AddHelper("showToast", "Could not send message.", "error")
        return ContactResponse{}, fmt.Errorf("sending email: %w", err)
    }

    a.Response().AddHelper("showToast", "Message sent!", "success")
    return ContactResponse{OK: true}, nil
}
```

### Action registration

Actions are **auto-registered** by the build system. After creating an action file, run `go run ./cmd/generator all` - the generator scans your `actions/` directory and creates `init()` functions that register each action automatically. You do not need to write any registration code.

## Action structure

Every action must:

1. Embed `piko.ActionMetadata` - provides `a.Ctx()`, `a.Request()`, `a.Response()`
2. Implement a `Call` method - parameters match form input `name` attributes
3. Return a typed response struct and an error

```go
type MyAction struct {
    piko.ActionMetadata
}

func (a MyAction) Call(username string, email string, age int) (MyResponse, error) {
    // Parameters populated automatically from form data or JSON body
}
```

## Parameter mapping

Form input `name` attributes map to `Call` parameters by name:

```html
<input name="username" />  →  username string
<input name="email" />     →  email string
<input name="age" />       →  age int
```

For JSON requests, property names map directly. The dispatch layer handles type conversion.

## Validation

### Automatic (dispatch layer)

Missing required parameters or type conversion failures return HTTP 400 with field-level errors automatically.

### Custom (in Call)

Return a `piko.ValidationField` error for field-level validation failures:

```go
func (a BookingAction) Call(start_date time.Time, end_date time.Time) (BookingResponse, error) {
    if start_date.After(end_date) {
        return BookingResponse{}, piko.ValidationField("start_date", "Must be before end date")
    }
    // ...
}
```

For multiple field errors at once, use `piko.NewValidationError`:

```go
return BookingResponse{}, piko.NewValidationError(map[string]string{
    "start_date": "Must be before end date",
    "end_date":   "Must be after start date",
})
```

## Response helpers

Trigger client-side actions after the server responds:

### showToast

```go
a.Response().AddHelper("showToast", "Message text", "success")  // green
a.Response().AddHelper("showToast", "Error occurred", "error")    // red
a.Response().AddHelper("showToast", "Warning!", "warning")        // yellow
a.Response().AddHelper("showToast", "Info message", "info")       // blue
a.Response().AddHelper("showToast", "Quick!", "success", "3000")  // custom duration (ms)
```

### closeModal

```go
a.Response().AddHelper("closeModal")                    // Close current modal
a.Response().AddHelper("closeModal", "edit-customer")   // Close by name
```

### reloadPartial

```go
a.Response().AddHelper("reloadPartial", "[data-partial=\"user-list\"]")
a.Response().AddHelper("reloadPartial", "#product-table")
```

### redirect

```go
a.Response().AddHelper("redirect", "/dashboard")          // Standard redirect
a.Response().AddHelper("redirect", "/login", "true")      // Replace history entry
```

### Combining helpers

```go
a.Response().AddHelper("showToast", "Saved!", "success")
a.Response().AddHelper("closeModal")
a.Response().AddHelper("reloadPartial", "[partial_name=\"customer-table\"]")
```

## Setting cookies

```go
a.Response().SetCookie(&http.Cookie{
    Name:     "session_token",
    Value:    session.Token,
    Path:     "/",
    HttpOnly: true,
    Secure:   true,
    SameSite: http.SameSiteStrictMode,
})
```

## Optional parameters

Use pointer types for optional parameters:

```go
func (a ProductUpsertAction) Call(id *int64, name string, price float64) (ProductResponse, error) {
    if id != nil {
        // Update existing
    } else {
        // Create new
    }
}
```

## CSRF protection

Automatic - no configuration needed:
- `piko.ActionMetadata` handles token generation and validation
- Forms submitted via `p-on:submit.prevent` with `action.` prefix include CSRF tokens
- Failed validation returns HTTP 403

## Triggering actions from buttons

Actions don't require forms. Bind to any event:

```piko
<button p-on:click="action.delete_product(state.ProductID)">Delete</button>
```

Pass template expressions as arguments:

```piko
<button p-on:click="action.toggle_status(item.ID, !item.IsActive)">Toggle</button>
```

## SSE streaming (Server-Sent Events)

Actions can stream real-time progress updates to the client via SSE. Useful for long-running operations, live notifications, or progress bars.

### Server side

```go
type ProgressAction struct {
    piko.ActionMetadata
}

func (a *ProgressAction) Call(task_id string) (ProgressResponse, error) {
    return ProgressResponse{Status: "completed"}, nil
}

func (a *ProgressAction) StreamProgress(stream *piko.SSEStream) error {
    for i := 1; i <= 3; i++ {
        if err := stream.Send("progress", map[string]any{
            "step":    i,
            "percent": i * 33,
        }); err != nil {
            return err
        }
        time.Sleep(100 * time.Millisecond)
    }
    return stream.SendComplete(map[string]string{"status": "done"})
}
```

| Method | Purpose |
|--------|---------|
| `stream.Send(eventType, data)` | Send a progress event |
| `stream.SendComplete(finalData)` | Send completion event with final response |

### Client side

```typescript
const result = await action.stream.Progress({ taskId: 'test-1' })
    .withOnProgress((data, eventType) => {
        console.log(eventType, data.percent + '%');
    })
    .call();

console.log('Final:', result);
```

Native `EventSource` API also works for GET-based SSE endpoints.

## Advanced action patterns

### Action builder API

Actions called from client-side TypeScript support a fluent builder for advanced behaviours:

```typescript
action.my_action.Submit({ name: 'Alice' })
    .withDebounce(300)
    .withLoading('#submit-btn')
    .withOnSuccess((resp) => { /* handle success */ })
    .withOnError(() => { /* handle error */ })
    .build();
```

| Builder method | Purpose |
|----------------|---------|
| `.withDebounce(ms)` | Wait `ms` after last call before executing (search inputs, auto-save) |
| `.withLoading(target)` | Add loading state to element (CSS selector, HTMLElement, or boolean) |
| `.withOptimistic(callback)` | Execute callback immediately before server responds |
| `.withOnSuccess(callback)` | Handle successful response |
| `.withOnError(callback)` | Handle error (use with `.withOptimistic` for rollback) |
| `.withOnProgress(callback)` | Handle SSE progress events |
| `.call()` | Execute and return a Promise (for `await`) |
| `.build()` | Create action descriptor (for event handler return) |

### Debouncing

Prevent excessive server calls during rapid interactions (e.g. search-as-you-type):

```typescript
function onSearchInput() {
    return action.search.Query({ q: state.query })
        .withDebounce(300)
        .withOnSuccess((resp) => { state.results = resp.items; })
        .build();
}
```

### Optimistic updates

Update UI immediately while waiting for server confirmation, with rollback on failure:

```typescript
function likePost() {
    const original = state.like_count;

    return action.toggle.Like({ current_count: original })
        .withOptimistic(() => { state.like_count = original + 1; })
        .withOnSuccess((resp) => { state.like_count = resp.count; })
        .withOnError(() => { state.like_count = original; })
        .build();
}
```

### Loading states

Show visual feedback during action execution:

```typescript
function submitForm() {
    return action.form.Submit({ name: state.name })
        .withLoading('#submit-btn')
        .withOnSuccess(() => { /* done */ })
        .build();
}
```

### Action chaining

Execute actions sequentially where later actions depend on earlier results:

```typescript
function startWorkflow() {
    return action.step.One({})
        .withOnSuccess((resp) => {
            return action.step.Two({ prev_id: resp.id })
                .withOnSuccess(() => { /* both done */ })
                .build();
        })
        .build();
}
```

### Batch operations

Execute multiple independent actions in parallel:

```typescript
async function runBatch() {
    const [a, b] = await Promise.all([
        action.math.Add({ a: 10, b: 20 }).call(),
        action.math.Multiply({ a: 6, b: 7 }).call(),
    ]);
    console.log(a.result, b.result);
}
```

## Testing actions

```go
func TestContactSubmit(t *testing.T) {
    ctx := context.Background()
    entry := piko.ActionHandlerEntry{
        Name:   "ContactSubmit",
        Method: http.MethodPost,
        Create: func() any { return &actions.ContactSubmitAction{} },
        Invoke: func(ctx context.Context, action any, arguments map[string]any) (any, error) {
            return action.(*actions.ContactSubmitAction).Call(
                arguments["name"].(string),
                arguments["email"].(string),
                arguments["message"].(string),
            )
        },
    }
    tester := piko.NewActionTester(t, entry)

    result := tester.Invoke(ctx, map[string]any{
        "name":    "Alice",
        "email":   "alice@example.com",
        "message": "Hello",
    })
    result.AssertSuccess()
}
```

## LLM mistake checklist

- Forgetting to embed `piko.ActionMetadata` in the action struct
- Forgetting to run `go run ./cmd/generator all` after creating a new action
- Mismatching form input `name` attributes with `Call` parameter names
- Using `action.MyAction()` in templates (use snake_case: `action.my_action()`)
- Using `a.Response().AddFieldError()` (does not exist - return `piko.ValidationField()` as an error instead)
- Using `a.Response().AddCookie()` (the method is `SetCookie`, not `AddCookie`)
- Returning an error without adding a user-facing helper (user sees nothing)
- Forgetting `.prevent` on form submit (`p-on:submit.prevent`)

## Related

- `references/pk-file-format.md` - page templates with forms
- `references/template-syntax.md` - `p-on` directive and event modifiers
- `references/examples.md` - CRUD example
