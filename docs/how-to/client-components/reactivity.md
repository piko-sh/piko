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

Use `snake_case` for state field names. The runtime binds each field bidirectionally to an HTML attribute of the same name on the host element. Writing `state.is_loading = true` reflects out as `is_loading="true"`, and external code calling `setAttribute('is_loading', 'true')` writes back to state. snake_case avoids the camelCase-to-kebab-case conversion the runtime applies otherwise. See [client components reference](../../reference/client-components.md#state-and-html-attribute-binding) for the binding mechanics and [how to share state between components](share-state-between-components.md) for cross-PKC patterns built on it.

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

Register lifecycle callbacks on the `pkc` alias (which the compiler binds to `this` for the component instance). The compiler does not auto-detect named functions - you must call `pkc.onX(callback)` explicitly:

```typescript
pkc.onConnected(() => {
    console.log('mounted');
});

pkc.onDisconnected(() => {
    console.log('unmounted');
});

pkc.onUpdated((changedProperties) => {
    console.log('changed:', Array.from(changedProperties).join(', '));
    if (changedProperties.has('name')) {
        state.updateCount++;
    }
});
```

`pkc.onUpdated` receives a `Set<string>` of changed property names. Use `.has(name)` and `Array.from(set)` to inspect it.

`pkc.onBeforeRender(callback)` and `pkc.onAfterRender(callback)` round out the per-render hooks. Both run on every render.

`pkc.onCleanup(callback)` registers a teardown function that runs after `onDisconnected`. The callback fires once when the component disconnects, then the cleanup queue clears. Use it to co-locate setup and teardown logic - register a cleanup from inside `onConnected` and Piko runs it at the end of the matching disconnect:

```typescript
pkc.onConnected(() => {
    const interval = window.setInterval(tick, 1000);
    pkc.onCleanup(() => window.clearInterval(interval));
});
```

See the [client components reference](../../reference/client-components.md) for the full lifecycle table.

## See also

- [Client components reference](../../reference/client-components.md).
- [About reactivity](../../explanation/about-reactivity.md) for the design rationale behind state and reruns.
- [How to events](events.md) for handling clicks and form input.
- [How to share state between components](share-state-between-components.md) for cross-PKC communication via attribute writes.
- [How to react to slotted children](watch-slotted-children.md) for slot-driven patterns.
- [How to event bus](event-bus.md) for cross-component messaging that does not map to a state field.
