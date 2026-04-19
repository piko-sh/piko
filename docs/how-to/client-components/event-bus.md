---
title: How to use the event bus for cross-component messaging
description: Emit and subscribe to events that cross component boundaries.
nav:
  sidebar:
    section: "how-to"
    subsection: "client-components"
    order: 30
---

# How to use the event bus for cross-component messaging

Components that do not share a parent-child relationship cannot communicate via props or custom DOM events. The global `piko.bus` provides a publish/subscribe channel that any component can use. See the [client components reference](../../reference/client-components.md) for the broader API.

<p align="center">
  <img src="../../diagrams/event-bus-pubsub.svg"
       alt="Component A calls piko.bus.emit with a named event and a payload. The bus delivers synchronously to every registered handler. Components B and C both subscribed to the same event name and each receive the payload. Naming conventions use tag-prefixed, feature-prefixed, and app-prefixed names."
       width="600"/>
</p>

## Emit an event

Any component can publish on the bus:

```typescript
function notifyUsers() {
    piko.bus.emit('chat:message', {
        user: state.currentUser,
        text: state.draft,
        at: new Date().toISOString(),
    });
}
```

The first argument is the event name. Use a prefix to avoid collisions. The second argument is the payload.

## Subscribe to an event

Subscribe inside `onConnected` and unsubscribe inside `onDisconnected`:

```typescript
function onConnected() {
    piko.bus.on('chat:message', handleMessage);
}

function onDisconnected() {
    piko.bus.off('chat:message', handleMessage);
}

function handleMessage(payload: { user: string; text: string; at: string }) {
    state.messages.push(payload);
}
```

Always pair subscription and unsubscription. A missing `off` call leaks the handler after Piko removes the component from the DOM.

## Subscribe once

Register a handler that fires exactly once:

```typescript
piko.bus.once('app:ready', () => {
    console.log('ready');
});
```

## Naming convention

Use a namespace prefix to keep event names readable and non-overlapping:

| Prefix | Use |
|---|---|
| `<component-tag>:` | Events from a specific component: `pp-counter:changed` |
| `app:` | Application-wide events: `app:ready`, `app:locale-change` |
| `chat:`, `auth:`, etc. | Feature-area events |

## See also

- [How to events](events.md) for intra-component event handling.
- [Client components reference](../../reference/client-components.md).
- [Scenario 008: event bus chat](../../showcase/008-event-bus-chat.md) for a runnable walkthrough.
