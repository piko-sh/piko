---
title: "010: Progress tracker"
description: Streaming progress updates via server-sent events (SSE)
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 110
---

# 010: Progress tracker

A streaming progress tracker where a server action pushes real-time updates to the browser using Server-Sent Events (SSE). The client renders a live progress bar and event log as events arrive.

## What this demonstrates

The `StreamProgress` method is a convention that opts an action into SSE streaming. Piko routes SSE requests to it automatically when the method is present. Inside the method, `stream.Send(eventType, data)` sends a named SSE event with JSON data and can run any number of times. The `stream.SendComplete(data)` call sends a final event and closes the stream. Calling `Send` after that returns an error.

The action builder API, `action.task.process(input).withOnProgress(callback).call()`, opens an SSE connection and routes events to the callback. A progress bar updates incrementally as events arrive, with no polling. `Call` remains mandatory as a fallback for non-streaming clients. A `.pk` page can hold both a `<script type="application/x-go">` section for the server and a `<script lang="ts">` section for the browser.

## Project structure

```text
src/
  actions/
    task/
      process.go                    Server action: streams 5 progress events via SSE
  pages/
    index.pk                        Page with progress bar and event log
```

## How it works

The action implements both `Call` (standard fallback) and `StreamProgress` (SSE entry point):

```go
func (a *ProcessAction) StreamProgress(stream *piko.SSEStream) error {
    for i := 1; i <= 5; i++ {
        stream.Send("progress", map[string]any{
            "step": i, "total": 5, "percent": i * 20,
        })
        time.Sleep(200 * time.Millisecond)
    }
    return stream.SendComplete(map[string]string{"status": "done"})
}
```

The client opens the stream with the action builder:

```typescript
const result = await action.task.process({ taskName })
    .withOnProgress((data, eventType) => { /* update UI */ })
    .call();
```

The `withOnProgress` callback fires for each `stream.Send()`. When `stream.SendComplete()` fires, the `.call()` promise resolves.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/010_progress_tracker/src/
go mod tidy
air
```

## See also

- [Server actions reference](../reference/server-actions.md).
- [How to streaming with SSE](../how-to/actions/streaming-with-sse.md).
