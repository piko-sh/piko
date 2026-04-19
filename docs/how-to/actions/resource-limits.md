---
title: How to set resource limits on an action
description: Cap request body size, execution timeout, slow-action threshold, and SSE connection duration on a server action.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 750
---

# How to set resource limits on an action

Implement `ResourceLimitable` on an action when you want to bound its request body size, execution time, or SSE connection lifetime. For the action surface see [server-actions](../../reference/server-actions.md).

## Limit upload size and timeout

To cap a file upload at 50 MB and abort after two minutes:

```go
func (a FileAction) ResourceLimits() *piko.ResourceLimits {
    return &piko.ResourceLimits{
        MaxRequestBodySize: 50 * 1024 * 1024,
        Timeout:            2 * time.Minute,
    }
}
```

The body cap rejects oversize requests before the handler runs. The timeout binds via `context.WithTimeoutCause` so downstream IO observes the cancellation.

## Log slow actions without aborting them

To record a slow-action log entry when a request exceeds a threshold but still let it finish, set `SlowThreshold`:

```go
func (a FileAction) ResourceLimits() *piko.ResourceLimits {
    return &piko.ResourceLimits{
        Timeout:       2 * time.Minute,
        SlowThreshold: 30 * time.Second,
    }
}
```

## Cap an SSE stream's lifetime

To force a clean reconnect every thirty minutes, set `MaxSSEDuration`:

```go
func (a NotificationStream) ResourceLimits() *piko.ResourceLimits {
    return &piko.ResourceLimits{
        MaxSSEDuration: 30 * time.Minute,
    }
}
```

The runtime does not emit periodic heartbeats by itself. Call `stream.SendHeartbeat()` from inside `StreamProgress` if proxies in front of your server close idle connections. See [streaming with SSE](streaming-with-sse.md) for the streaming protocol implementation.

## See also

- [Server actions reference](../../reference/server-actions.md) for the full action surface.
- [About the action protocol](../../explanation/about-the-action-protocol.md) for the dispatch pipeline that enforces these limits.
- [How to cache action responses](caching.md), [How to rate-limit an action](rate-limiting.md), and [How to override an action's HTTP method](method-override.md) for the interfaces that compose with `ResourceLimitable`.
- [How to stream with SSE](streaming-with-sse.md) for streaming responses.
