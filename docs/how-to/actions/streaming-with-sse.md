---
title: How to stream action responses with SSE
description: Write a server action that streams progress updates to the client using server-sent events.
nav:
  sidebar:
    section: "how-to"
    subsection: "actions"
    order: 20
---

# How to stream action responses with SSE

Long-running actions can stream progress to the client via Server-Sent Events (SSE) instead of forcing the browser to wait for one large response. This guide shows how to build a streaming action. See the [server-actions reference](../../reference/server-actions.md) for the full API and [Scenario 010: progress tracker](../../showcase/010-progress-tracker.md) for the runnable example.

<p align="center">
  <img src="../../diagrams/sse-stream-sequence.svg"
       alt="A sequence diagram with Browser on the left and Server on the right, time flowing top to bottom. Browser invokes the action call which becomes a POST request with Accept text event-stream. The server's dispatcher picks StreamProgress because the Accept header matched. Activation bars on each lifeline mark when each side is active. The server emits progress events at percent 10, 40, 70 inside a Send loop, and the browser runs onProgress for each event to update state. The server then emits a final complete event carrying the typed response, the browser runs onComplete, and the stream closes. A footer notes that without the Accept header the dispatcher calls Call instead and returns a single JSON body."
       width="800"/>
</p>

## Implement `SSECapable`

A streaming action implements the `piko.SSECapable` interface in addition to its normal `Call` method:

```go
package task

import (
    "fmt"
    "time"

    "piko.sh/piko"
)

type ProcessInput struct {
    JobID string `json:"jobID" validate:"required"`
}

type ProcessResponse struct {
    JobID string `json:"job_id"`
}

type ProcessAction struct {
    piko.ActionMetadata
}

func (a *ProcessAction) Call(input ProcessInput) (ProcessResponse, error) {
    return ProcessResponse{JobID: input.JobID}, nil
}

func (a *ProcessAction) StreamProgress(stream *piko.SSEStream) error {
    for i := 0; i <= 100; i += 10 {
        event := map[string]any{
            "progress": i,
            "message":  fmt.Sprintf("Processing step %d", i),
        }
        if err := stream.Send("progress", event); err != nil {
            return err
        }
        time.Sleep(500 * time.Millisecond)
    }

    return stream.SendComplete(ProcessResponse{JobID: "done"})
}
```

The runtime selects `StreamProgress` when the client requests `Accept: text/event-stream`. Otherwise `Call` runs as a normal action.

## Consume the stream from the browser

Call the action with `.withOnProgress(callback).call()`. The callback receives each streamed event:

```html
<template>
  <button p-on:click="handleClick">Start</button>
  <progress :value="state.Progress" max="100"></progress>
</template>

<script lang="ts">
async function handleClick(): Promise<void> {
    const result = await action.task.Process({ jobID: "abc-123" })
        .withOnProgress((data: unknown, eventType: string) => {
            const event = data as { done: number; total: number; label: string };
            state.Progress = (event.done / event.total) * 100;
        })
        .call();

    // `result` is the typed response returned from Call() (or SendComplete in StreamProgress).
    console.log("Finished", result.job_id);
}
</script>
```

The callback receives `(data, eventType)`: `data` is the payload for a single event, and `eventType` is the event name passed to `stream.Send`. The final value from `.call()` is the typed response the action returned (usually via `stream.SendComplete`).

## Resume after a dropped connection

SSE clients can resume from the last received event ID. Send an event ID on each message:

```go
func (a ProcessAction) StreamProgress(stream *piko.SSEStream) error {
    for i, step := range steps {
        event := map[string]any{
            "step": step.Name,
            "done": i + 1,
        }
        if err := stream.SendWithID("progress", fmt.Sprintf("%d", i+1), event); err != nil {
            return err
        }
    }
    return stream.SendComplete(nil)
}
```

The client reconnects automatically and includes a `Last-Event-ID` header. Use `a.Request().Header("Last-Event-ID")` in the action to skip already-sent steps.

## Close on client disconnect

`a.Ctx().Done()` fires when the client closes the stream. Check it between iterations:

```go
for i := 0; i < len(steps); i++ {
    select {
    case <-a.Ctx().Done():
        return a.Ctx().Err()
    default:
    }
    // ... process step i
}
```

## Handling errors mid-stream

Return a non-nil error from `StreamProgress` to emit an `error` event and close the stream:

```go
if err := runStep(step); err != nil {
    return fmt.Errorf("step %d: %w", i, err)
}
```

The frontend runtime surfaces the error as a thrown exception from the `await` on `.call()`.

## See also

- [Server actions reference](../../reference/server-actions.md) for the full API including `SSECapable` and `SSEStream`.
- [How to forms](forms.md) for non-streaming actions.
- [Scenario 010: progress tracker](../../showcase/010-progress-tracker.md) for a runnable walkthrough.
- [Scenario 011: instant messaging](../../showcase/011-instant-messaging.md) for a chat-style stream with event-ID resumption.

Integration test: [`tests/integration/e2e_browser/`](https://github.com/piko-sh/piko/tree/master/tests/integration/e2e_browser) exercises SSE actions end-to-end.
