---
title: How to handle events in a client component
description: Attach click, input, and keyboard handlers; bind form inputs; dispatch custom events.
nav:
  sidebar:
    section: "how-to"
    subsection: "client-components"
    order: 20
---

# How to handle events in a client component

Client components respond to DOM events via the `p-on` directive and dispatch custom events via `piko.event.dispatch` (DOM-bubbling) or `piko.bus.emit` (global pub/sub). This guide covers both. See the [client components reference](../../reference/client-components.md) for the full directive list.

## Attach a click handler

Use `p-on:click` to bind a function. The function name must match an exported function in the `<script>`:

```pkc
<template name="pp-button">
    <button p-on:click="handleClick">Click me</button>
</template>

<script lang="ts">
    function handleClick() {
        console.log('clicked');
    }
</script>
```

## Read the event object

Accept an `event` argument in the handler:

```typescript
function handleClick(event: MouseEvent) {
    event.preventDefault();
    console.log(event.clientX, event.clientY);
}
```

## Pass custom arguments

Pass arguments inline:

```piko
<template>
    <ul>
        <li p-for="item in state.items" p-key="item.id">
            <button p-on:click="remove(item.id)">Remove</button>
        </li>
    </ul>
</template>

<script lang="ts">
    function remove(id: number) {
        state.items = state.items.filter(item => item.id !== id);
    }
</script>
```

The `event` argument is still available as the last parameter:

```html
<button p-on:click="handle(item.id, $event)">Handle</button>
```

```typescript
function handle(id: number, event: MouseEvent) {
    console.log(id, event.clientX);
}
```

## Bind form inputs with `p-model`

`p-model` sets up two-way binding on a form control:

```pkc
<template>
    <input type="text" p-model="state.name" />
    <p>Hello, {{ state.name }}</p>
</template>

<script lang="ts">
    const state = {
        name: '' as string,
    };
</script>
```

`p-model` works on `<input>`, `<textarea>`, `<select>`, and checkbox/radio inputs. The component re-renders on each input event.

For checkboxes:

```html
<input type="checkbox" p-model="state.agreed" />
```

For radio groups:

```html
<input type="radio" value="small" p-model="state.size" />
<input type="radio" value="medium" p-model="state.size" />
```

## Handle keyboard events

The supported event modifiers are `prevent`, `stop`, `once`, `self`, `passive`, and `capture`. There are no key-name modifiers - filter keys inside the handler:

```html
<input type="text" p-on:keyup="onKey" />
```

```typescript
function onKey(event: KeyboardEvent) {
    if (event.key === 'Enter') {
        submit();
    }
}
```

## Dispatch a custom event

Emit events other components can listen to. Two surfaces, picked by who needs to hear them:

```typescript
function submit() {
    // 1. DOM-bubbling event - an ancestor listens with p-event:pp-form:submit
    piko.event.dispatch(this, 'pp-form:submit', {
        name: state.name,
        agreed: state.agreed,
    });

    // 2. Global pub/sub - listeners can be anywhere, even outside the DOM tree
    piko.bus.emit('pp-form:submit', { name: state.name, agreed: state.agreed });
}
```

`piko.event.dispatch(target, name, detail?, options?)` takes a target (an element, a CSS selector, or a `p-ref` name) and produces a real `CustomEvent` that bubbles through the DOM. `piko.bus.emit` decouples from the DOM and reaches every subscriber.

By convention, custom event names use the component tag as a namespace: `pp-form:submit`, `pp-counter:changed`.

Note that `this` inside `submit` is the component instance only because the compiler emits the handler call as `this.$$ctx.fn.call(this, e)`. If you forward the call from another helper that does not preserve `this`, pass `pkc` (or the explicit element) as the target instead.

## Listen to a custom event in the template

`p-event:` is the framework's purpose-built directive for catching custom events emitted by a child PKC. It mirrors `p-on:`, but targets namespaced custom event names. The compiler emits the binding as a `pe:event-name` VDOM prop:

```pkc
<template name="parent-component">
    <pp-form p-event:pp-form:submit="handleSubmit($event)"></pp-form>
</template>

<script lang="ts">
    function handleSubmit(event: CustomEvent<{ name: string; agreed: boolean }>) {
        console.log(event.detail);
    }
</script>
```

The directive listens on the host element of the child and receives the same `CustomEvent` that `piko.event.dispatch` produces. Modifiers (`prevent`, `stop`, `once`, `self`, `passive`, `capture`) work the same as `p-on:`.

Prefer `p-event:` over `p-on:` for custom events. `p-on:` works for plain DOM event names but conflates names that contain a `:` with the modifier delimiter, and the framework's emitted VDOM uses `pe:` specifically for custom events.

## Listen to a custom event from JavaScript

`piko.bus.on(name, handler)` returns a `() => void` unsubscribe function. Capture it on connect and call it on disconnect:

```typescript
let offSubmit: (() => void) | undefined;

pkc.onConnected(() => {
    offSubmit = piko.bus.on('pp-form:submit', handleSubmit);
});

pkc.onDisconnected(() => {
    offSubmit?.();
});

function handleSubmit(payload: { name: string; agreed: boolean }) {
    console.log(payload);
}
```

`piko.bus.off(name)` removes **every** listener for `name`, not the specific handler - the function-return form above is the only safe way to detach a single subscriber. `piko.bus.once(name, handler)` is the fire-once variant and likewise returns an unsubscribe function.

## Prevent event bubbling or default behaviour

Use built-in modifiers:

```html
<button p-on:click.stop="handle">Stop propagation</button>
<a p-on:click.prevent="handle">Prevent default</a>
<button p-on:click.once="handle">Fire once</button>
```

Stack modifiers: `p-on:click.stop.prevent`.

## See also

- [Client components reference](../../reference/client-components.md) for the full directive list.
- [About reactivity](../../explanation/about-reactivity.md) for where event handlers fit in the PK/PKC split.
- [How to reactivity](reactivity.md).
- [How to share state between components](share-state-between-components.md) for state-shaped cross-component communication via attribute writes.
- [How to react to slotted children](watch-slotted-children.md).
- [How to event bus](event-bus.md) for cross-component messaging that does not map to a state field.
