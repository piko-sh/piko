---
title: "008: Event bus chat"
description: Cross-component communication via Piko's event bus
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 90
---

# 008: Event bus chat

A chat interface demonstrating cross-component communication via Piko's event bus. A PKC component publishes messages with `piko.bus.emit()`, while a server-rendered partial subscribes with `piko.bus.on()` and updates the DOM in real time. The two communicate without direct references to each other.

## What this demonstrates

- **`piko.bus.emit()` in PKC components**: publishing named events with payload data
- **`piko.bus.on()` in partials**: subscribing to events from a partial's `<script lang="ts">` section
- **Decoupled communication**: the event bus is for independent components (not parent-child; use props for that); either component can be replaced as long as the event contract is maintained
- **`piko.bus`** is available globally in PKC components and partial client-side scripts
- **Partials with client-side TypeScript**: partials can have both `<script type="application/x-go">` (server) and `<script lang="ts">` (browser) sections, enabling server-rendered content to become interactive
- **Mixed composition**: combining a `<piko:partial>` with a PKC custom element on one page

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

```html
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
