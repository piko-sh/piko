---
title: "011: Instant messaging"
description: Real-time chat with SSE streaming, auto-reconnection, and event replay
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 120
---

# 011: Instant messaging

A real-time chat application where multiple browser tabs exchange messages via Piko's SSE streaming. Messages flow through a server-side hub: `chat.Send` broadcasts to all connected `chat.Listen` streams. If a connection drops, `withRetryStream()` reconnects and replays missed messages using event IDs.

## What this demonstrates

- **Server-mediated real-time messaging**: `chat.Send` and `chat.Listen` actions working through a shared in-memory hub
- **Long-lived SSE streams**: `chat.Listen` keeps the connection open indefinitely; the stream never calls `SendComplete`; it runs until the client disconnects
- **`withRetryStream()` for auto-reconnection**: configurable reconnection with exponential backoff
- **Event ID resumption**: `stream.EnableEventIDs()` attaches IDs; on reconnect the client sends `Last-Event-ID` and the server replays missed messages from a ring buffer
- **Heartbeat keep-alive**: server sends pings every 30 seconds to prevent proxy timeouts
- `Subscribe()` before reading history ensures no messages are missed between the history read and the subscription
- Non-blocking fan-out: `Broadcast` uses `select` with `default` to skip slow subscribers

## Project structure

```text
src/
  actions/
    chat/
      hub.go                          Chat hub singleton with Subscribe/Broadcast/History
      send.go                         POST action: broadcast a message
      listen.go                       SSE streaming action: subscribe and forward
  pages/
    index.pk                          Chat UI with login, message feed, send controls
```

## How it works

The hub manages subscribers and a ring buffer of the last 100 messages. The `Listen` action subscribes and forwards indefinitely:

```go
func (a *ListenAction) StreamProgress(stream *piko.SSEStream) error {
    stream.EnableEventIDs()
    msgCh, unsubscribe := hub.Subscribe()
    defer unsubscribe()
    // Replay history, then forward live messages
    for {
        select {
        case <-stream.Done(): return nil
        case msg := <-msgCh: stream.Send("chat", msg)
        case <-heartbeat.C: stream.SendHeartbeat()
        }
    }
}
```

The client connects with auto-reconnection:

```typescript
await action.chat.Listen({})
    .withOnProgress((data, eventType) => { /* append message */ })
    .withRetryStream({ maxReconnects: Infinity, baseDelay: 2000, backoff: 'exponential' })
    .call();
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/011_live_notifications/src/
go mod tidy
air
```
