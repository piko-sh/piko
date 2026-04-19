---
title: "008: Event bus chat"
description: Cross-component communication via Piko's event bus
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 90
---

# 008: Event bus chat

A chat interface demonstrating cross-component communication via Piko's event bus. A PKC component publishes messages with `piko.bus.emit()`, while a server-rendered partial subscribes with `piko.bus.on()` and updates the DOM in real time. The two communicate without direct references to each other.

## What this demonstrates

PKC components publish named events with payload data through `piko.bus.emit()`. Partials subscribe to events from their `<script lang="ts">` section with `piko.bus.on()`. The event bus suits independent components, not parent-child pairs, which should use props. Either side can change without breaking the other, so long as the event contract holds. The `piko.bus` object is available globally in PKC components and partial client-side scripts.

Partials can hold both a `<script type="application/x-go">` section for the server and a `<script lang="ts">` section for the browser, which lets server-rendered content become interactive. A single page can mix a `<piko:partial>` with a PKC custom element.

## Project structure

```text
src/
  components/
    pp-chat-input.pkc         PKC component - captures input, emits events
  partials/
    message-log.pk            Partial - listens for events, updates DOM
  pages/
    index.pk                  Host page composing both
```

## How it works

The PKC component captures input via `p-model` and publishes on send:

```ts
piko.bus.emit('chat-message', { text: state.message, sender: 'User', timestamp: Date.now() });
```

The partial's client-side script subscribes and manipulates the DOM:

```ts
piko.bus.on('chat-message', (data) => {
    // append message to the log
});
```

The host page composes both without coordinating them:

```piko
<piko:partial is="message_log"></piko:partial>
<pp-chat-input></pp-chat-input>
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/008_event_bus_chat/src/
go mod tidy
air
```

## See also

- [Client components reference](../reference/client-components.md).
- [How to event bus](../how-to/client-components/event-bus.md).
