---
title: How to set resource limits on an action
description: Implement ResourceLimitable to cap request body size, response size, timeout, concurrency, memory, and SSE duration on a server action.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 750
---

# How to set resource limits on an action

Actions implement `ResourceLimitable` to bound their own resource footprint. The limits protect the rest of the application from a single endpoint's bad day. Common cases include a runaway upload, a slow downstream call, a goroutine leak under load, and an SSE client that never disconnects. For the action structure see [server actions reference](../../reference/server-actions.md).

## Add a ResourceLimits receiver

Implement `ResourceLimitable` by adding a `ResourceLimits()` receiver that returns a `*piko.ResourceLimits`:

```go
type ResourceLimitable interface {
    ResourceLimits() *piko.ResourceLimits
}

type ResourceLimits struct {
    MaxRequestBodySize   int64         // Maximum request body size in bytes (0 = framework default).
    MaxResponseSize      int64         // Maximum response size in bytes (0 = no limit).
    Timeout              time.Duration // Hard execution timeout (0 = framework default).
    SlowThreshold        time.Duration // Logs the action as slow once exceeded (0 = framework default).
    MaxConcurrent        int           // Maximum concurrent executions (0 = no limit).
    MaxMemoryUsage       int64         // Advisory per-call memory hint in bytes (0 = no limit).
    MaxSSEDuration       time.Duration // Hard cap on SSE connection lifetime.
    SSEHeartbeatInterval time.Duration // Interval between SSE heartbeat messages.
}
```

Zero leaves the framework default in place for fields that have one, and disables the limit for fields that do not.

## Bound a file upload

```go
package upload

import (
    "fmt"
    "time"

    "piko.sh/piko"
)

type FileResponse struct {
    Filename string `json:"filename"`
    Size     int64  `json:"size"`
}

type FileAction struct {
    piko.ActionMetadata
}

func (a FileAction) ResourceLimits() *piko.ResourceLimits {
    return &piko.ResourceLimits{
        MaxRequestBodySize: 50 * 1024 * 1024, // 50 MB
        Timeout:            2 * time.Minute,
        MaxConcurrent:      5,
    }
}

func (a FileAction) Call(file piko.FileUpload) (FileResponse, error) {
    data, err := file.ReadAll()
    if err != nil {
        return FileResponse{}, fmt.Errorf("reading upload: %w", err)
    }

    header := file.Header()

    if err := storage.Save(a.Ctx(), header.Filename, data); err != nil {
        return FileResponse{}, fmt.Errorf("saving file: %w", err)
    }

    a.Response().AddHelper("showToast", "File uploaded.", "success")

    return FileResponse{Filename: header.Filename, Size: header.Size}, nil
}
```

The 50 MB body cap rejects oversize requests before any handler code runs. The two-minute timeout bounds the worst case. `MaxConcurrent: 5` keeps a flood of uploads from saturating the disk or network.

## Tame a long-running SSE stream

For actions that stream Server-Sent Events, set both `MaxSSEDuration` and `SSEHeartbeatInterval`:

```go
func (a NotificationStream) ResourceLimits() *piko.ResourceLimits {
    return &piko.ResourceLimits{
        MaxSSEDuration:       30 * time.Minute,
        SSEHeartbeatInterval: 15 * time.Second,
    }
}
```

The heartbeat stops intermediate proxies from closing idle connections, and the duration cap forces a clean reconnect cycle so memory does not grow forever.

## Compose with other interfaces

`ResourceLimitable` composes with `MethodOverridable`, `Cacheable`, and `RateLimitable`. See [How to override an action's HTTP method](method-override.md), [How to cache action responses](caching.md), and [How to rate-limit an action](rate-limiting.md).

## See also

- [Server actions reference](../../reference/server-actions.md) for the action surface.
- [How to stream with SSE](streaming-with-sse.md) for the streaming protocol.
- [How to profile a Piko application](../profiling.md) when limits start tripping under load.
