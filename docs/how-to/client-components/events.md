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

Client components respond to DOM events via the `p-on` directive and dispatch custom events via `piko.dispatch`. This guide covers both. See the [client components reference](../../reference/client-components.md) for the full directive list.

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

Use event modifiers to filter keys:

```html
<input type="text" p-on:keyup.enter="submit" />
```

Supported key modifiers: `enter`, `escape`, `space`, `tab`, `delete`, `up`, `down`, `left`, `right`.

## Dispatch a custom event

Emit events other components can listen to:

```typescript
function submit() {
    piko.dispatch('pp-form:submit', {
        name: state.name,
        agreed: state.agreed,
    });
}
```

By convention, custom event names use the component tag as a namespace: `pp-form:submit`, `pp-counter:changed`.

## Listen to a custom event

Subscribe in another component's `onConnected` and tear down in `onDisconnected`:

```typescript
function onConnected() {
    piko.bus.on('pp-form:submit', handleSubmit);
}

function onDisconnected() {
    piko.bus.off('pp-form:submit', handleSubmit);
}

function handleSubmit(payload: { name: string; agreed: boolean }) {
    console.log(payload);
}
```

`piko.bus.on` registers a listener, and `piko.bus.off` removes it. Always pair them inside `onConnected` and `onDisconnected` to prevent leaks.

> **Note:** The bus shape (`piko.bus.on/off` vs the real `pk.hooks` / event-system entry points) is a separate Phase-1 sweep. Treat the API names in this snippet as a placeholder; refer to [How to event bus](event-bus.md) for the canonical shape after that sweep lands.

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
- [How to reactivity](reactivity.md).
- [How to event bus](event-bus.md) for cross-component communication.
- [Scenario 003: reactive counter](../../showcase/003-reactive-counter.md).
- [Scenario 008: event bus chat](../../showcase/008-event-bus-chat.md) for inter-component events.
