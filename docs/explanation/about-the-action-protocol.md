---
title: About the action protocol
description: "How server actions work under the hood: RPC over HTTP with typed requests, typed responses, and SSE streaming."
nav:
  sidebar:
    section: "explanation"
    subsection: "architecture"
    order: 30
---

# About the action protocol

Piko's server actions are the only way a client component talks to the server. Every form submission, every button click that touches the database, every progress-bar update rides the action protocol. This page explains the shape of that protocol and the reasoning behind it.

## The surface

<p align="center">
  <img src="../diagrams/action-protocol-sequence.svg"
       alt="Browser on the left posts to the Piko dispatch layer in the centre, which performs CSRF, binding, cache, and rate-limit checks before inspecting the Accept header. The dispatch then forks horizontally: the default branch goes to the Call method (top right) which returns a JSON body; the text/event-stream branch goes to StreamProgress (bottom right) which returns an SSE stream of progress events plus a complete event. Two return paths along the bottom of the diagram carry the JSON response and the SSE events back to the browser."
       width="800"/>
</p>

A server action is a Go struct that embeds `piko.ActionMetadata` and implements a `Call` method. The generator discovers the struct, mounts it under a conventional path (`/_piko/actions/<package>.<name>`), and wires the dispatch code that binds an HTTP request to the `Call` method's parameters. The `_piko` prefix keeps Piko-internal routes namespaced so they do not collide with application URLs.

From the browser, the generated TypeScript surface is `action.<namespace>.<function>({...}).call()`:

```typescript
const result = await action.customer.upsert({ name: 'Alice', email: 'a@example.com' }).call();
```

The browser posts JSON to `/_piko/actions/customer.Upsert`. When the call carries file uploads, the runtime switches to `multipart/form-data` instead. The dispatch layer parses the body, validates against the Go parameter types, runs `Call`, and returns the response as JSON. There is no `application/x-www-form-urlencoded` branch.

## Why RPC, not REST

Most Go web frameworks expose a `http.Handler` shape: request in, response out. Actions could have been that. Piko chose RPC instead for three reasons.

First, **types flow both ways**. An action's `Call` method declares typed parameters and a typed return value. The generator emits TypeScript shims that mirror those types on the client. A rename of a field in the Go response struct updates the TypeScript interface, and a consumer that relied on the old field name fails to compile. That round-trip would require a lot of manual work in a REST-handler shape.

Second, **dispatch is mechanical, not handcrafted**. Piko handles parameter binding, CSRF validation, rate limiting, caching, and response shaping. Application code writes only the `Call` body. Contrast that with hand-writing per-field decoders, a validator, a response writer, and a JSON encoder. Every action would be ten lines of boilerplate, repeated everywhere, each copy subtly different.

Third, **the URL convention is discoverable**. `/_piko/actions/customer.Upsert` reads plainly. The `customer` package has an `Upsert` action. No route registration file. No list of URLs to maintain. The file system (`actions/customer/upsert.go`) determines the URL.

## The tradeoffs of RPC

RPC actions are not a good fit for third-party API consumption. An RPC endpoint exposes Go-shaped operations, not resource-oriented URLs that a generic HTTP client would recognise. If a project needs to expose a public API, it should use standard REST handlers alongside actions, not instead of them.

RPC actions tie the client and server to the same generator. A project that wants to call its actions from a mobile app in a different language has to replicate the RPC shape by hand or use OpenAPI schemas generated separately. For in-house web frontends this is a non-issue. For broader API ecosystems it is a real cost.

## Errors are first-class

An action returns `(Response, error)`. The error shape determines the HTTP status:

- `nil` gives 200 with the response body.
- `piko.ValidationField(field, message)` gives 422 with the field error attached; the client renders it inline next to the matching input.
- `piko.NewValidationError(fields)` gives 422 with multiple field errors at once.
- Any other `error` gives 500; the dispatch layer logs the full detail and the client sees a generic failure message.

This shape forces application code to distinguish user-correctable validation from internal failures. Piko's [safe-error type](../reference/errors.md) formalises the same pattern for render functions.

## SSE for long-running actions

An action that takes more than a second or two should stream progress instead of making the user wait. An action implements the optional `SSECapable` interface with a `StreamProgress(stream *piko.SSEStream) error` method. The dispatch layer picks `StreamProgress` up when the client requests `Accept: text/event-stream`. Otherwise it calls `Call` as normal.

```go
func (a *ProcessAction) StreamProgress(stream *piko.SSEStream) error {
    for i := 0; i <= 100; i += 10 {
        stream.Send("progress", map[string]any{"progress": i})
        time.Sleep(500 * time.Millisecond)
    }
    return stream.SendComplete(ProcessResponse{Status: "done"})
}
```

`StreamProgress` has no additional method arguments beyond the stream. The dispatch layer populates the action struct with the decoded inputs before either `Call` or `StreamProgress` runs.

The client receives an event stream with named events and a final `complete` event that carries the typed response. The SSE transport sits on top of the same RPC routing. The dispatch layer decides at runtime whether to run `Call` or `StreamProgress`.

## Caching and rate limiting as structural concerns

Actions implement optional interfaces to opt into caching and rate limiting:

- `Cacheable` with `CacheConfig() *CacheConfig` attaches a TTL (and optional vary headers or a custom key function). Repeated calls with the same arguments return cached responses.
- `RateLimitable` with `RateLimit() *RateLimit` attaches a token bucket. Calls over the budget return HTTP 429.

These are declarations, not code. Piko implements the behaviour. Application code says "this action caches for 60 seconds", and the cache layer handles the rest.

## Call shape: parameters or struct

`Call` can accept either individual parameters (`Call(id int64, name string)`) or a single input struct (`Call(input UpsertInput)`). The generator treats both the same way. It emits a TypeScript call shape that mirrors the Go parameters, and the dispatch layer binds the request body accordingly. Which form fits best depends on the shape of the action.

Individual parameters fit small, action-style calls that take one to three scalar arguments. `Delete(id int64)`, `ToggleFavourite(itemID int64)`, and `Rename(id int64, newName string)` read naturally without a struct definition, and the call site stays terse. For anything that reads like an imperative on a known entity, individual parameters match the shape of the operation.

An input struct fits form submissions and any call with four or more fields, optional fields, or validation rules. Keeping `validate` tags next to the data they guard makes validation visible at the type definition. The TypeScript type on the client matches the form's shape exactly, which helps when the client binds form data to the call. When an action evolves to take more fields, adding them to a struct is more sustainable than growing a parameter list.

Either form is correct. The choice reflects the action's shape, not a global convention.

## What the protocol does not do

### HTTP method override

Actions mount at POST by default. The seven `MethodGet`, `MethodHead`, `MethodPost`, `MethodPut`, `MethodDelete`, `MethodOptions`, and `MethodPatch` constants exist, and an action may declare a `Method() piko.HTTPMethod` receiver. At present the generator's `detectHTTPMethodOverride` only checks whether the receiver is *defined*. It ignores the return value, so every action with a `Method()` receiver still mounts at POST in the registry. Treat the verb as effectively fixed for the moment.

### Other absences

SSE-capable actions register an additional GET on the same path so the browser's `EventSource` can connect. Actions do not support path parameters. Parameters always travel in the JSON body, the query string, or the multipart form (for uploads). They do not expose the underlying `http.Request`, and application code receives only the decoded arguments. The narrowness is deliberate. Actions are narrower than `http.Handler`, and the narrowness is the point.

Actions do not return redirects. To navigate the browser after an action, the response uses the `redirect` helper (`a.Response().AddHelper("redirect", "/dashboard")`) which the frontend runtime interprets. The HTTP response itself is always 200 with a JSON body when successful.

## See also

- [Server actions reference](../reference/server-actions.md) for the full API.
- [How to forms](../how-to/actions/forms.md) for the form-submission recipe.
- [How to streaming with SSE](../how-to/actions/streaming-with-sse.md).
- [How to cache action responses](../how-to/actions/caching.md) and [how to rate-limit an action](../how-to/actions/rate-limiting.md).
- [About reactivity](about-reactivity.md) for where actions fit in the PK/PKC split.
