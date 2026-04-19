---
title: How to add reactive state to a client component
description: Declare state, mutate it, and let Piko re-render the component.
nav:
  sidebar:
    section: "how-to"
    subsection: "client-components"
    order: 10
---

# How to add reactive state to a client component

A `.pkc` component re-renders automatically when its `state` object changes. This guide covers the patterns for declaring state, mutating it, and working with nested values. See the [client components reference](../../reference/client-components.md) for the full file format.

## Declare state

Inside `<script lang="ts">`, assign `const state = { ... }`:

```typescript
const state = {
    count: 0 as number,
    message: 'Hello' as string,
    items: [] as string[],
    user: null as { name: string } | null,
};
```

Type annotations use TypeScript's `as` cast. Every property assignment is reactive.

## Mutate primitive values

Writes to simple properties trigger a re-render:

```typescript
function increment() {
    state.count++;
}

function setMessage(text: string) {
    state.message = text;
}
```

The template updates immediately.

## Mutate arrays

Arrays are reactive. `push`, `splice`, `pop`, and direct index assignment all trigger re-renders:

```typescript
function addItem(text: string) {
    state.items.push(text);
}

function removeItem(index: number) {
    state.items.splice(index, 1);
}

function replaceItem(index: number, text: string) {
    state.items[index] = text;
}
```

When rendering an array with `p-for`, always set `p-key` to a stable identifier:

```piko
<li p-for="(idx, item) in state.items" p-key="item.id">
    {{ item.text }}
</li>
```

Without a key, Piko falls back to index-based diffing and may re-use DOM nodes incorrectly after reorders.

## Mutate objects

Property writes on nested objects are reactive:

```typescript
state.user = { name: 'Alice' };
state.user.name = 'Bob';
```

Replacing the whole object (first line above) is reactive, as is updating a single property (second line).

## Non-reactive variables

Any variable declared outside `state` does not trigger re-renders. Use this for transient helpers that should not appear in the UI:

```typescript
let lastKeystroke = 0;

function handleKeyup() {
    lastKeystroke = Date.now();
    // Nothing re-renders.
}
```

Promote a value to `state` when you want the template to reflect it.

## Computed values

There is no dedicated `computed` primitive. Recompute derived values inside the render path by using template expressions:

```html
<template>
    <p>Items: {{ state.items.length }}</p>
    <p>Total: {{ state.items.reduce((acc, item) => acc + item.price, 0) }}</p>
</template>
```

For heavier computations, store the derived value in `state` and update it when inputs change.

## Lifecycle hooks

Export functions named after the hook to run code at each lifecycle moment. The compiler picks them up by name:

```typescript
function onConnected() {
    console.log('mounted');
}

function onDisconnected() {
    console.log('unmounted');
}

function onUpdated(changedProperties: Set<string>) {
    console.log('changed:', [...changedProperties]);
}
```

`onBeforeRender()` and `onAfterRender()` round out the hook set; both run on every render. See [client components reference](../../reference/client-components.md) for the full lifecycle table.

## See also

- [Client components reference](../../reference/client-components.md).
- [How to events](events.md) for handling clicks and form input.
- [How to event bus](event-bus.md) for cross-component messaging.
- [Scenario 003: reactive counter](../../showcase/003-reactive-counter.md) and [Scenario 007: todo app](../../showcase/007-todo-app.md) for runnable examples.
