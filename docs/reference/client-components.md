---
title: Client components
description: The .pkc file format, reactive state model, lifecycle hooks, and supported template directives.
nav:
  sidebar:
    section: "reference"
    subsection: "file-formats"
    order: 20
---

# Client components

A `.pkc` file is a client-side reactive component that compiles to a native Web Component with shadow DOM encapsulation. This page documents the file format and the runtime surface. For task recipes see the [reactivity how-to](../how-to/client-components/reactivity.md) and [events how-to](../how-to/client-components/events.md).

<p align="center">
  <img src="../diagrams/pkc-component-lifecycle.svg"
       alt="Four-state machine. Created: the custom element upgrades and the shadow DOM attaches. Mounted: onConnected runs after the first render. Running: state mutations tracked by the reactive proxy cause local re-renders, with onBeforeRender, onAfterRender, and onUpdated firing per render. Unmounted: onDisconnected runs before the element leaves the DOM."
       width="760"/>
</p>

## File structure

A `.pkc` file has three sections: `<template>`, `<script lang="ts">`, and `<style>`.

```pkc
<template name="pp-counter">
    <div>
        <p>Count: {{ state.count }}</p>
        <button p-on:click="increment">Increment</button>
    </div>
</template>

<script lang="ts">
    const state = {
        count: 0 as number,
    };

    function increment() {
        state.count++;
    }
</script>

<style>
    div { padding: 1rem; border: 1px solid #eee; }
</style>
```

## Template attributes

| Attribute | Values | Purpose |
|---|---|---|
| `name` | kebab-case custom element name | Required. Defines the registered tag (for example `pp-counter`). Must contain a hyphen per the Web Components spec. |
| `enable` | `"form"` | Opts into form association; the component exposes itself as a form control. |

Convention. Use the `pp-` prefix for project components to avoid collisions with built-in and third-party custom elements.

## Script attributes

| Attribute | Values | Purpose |
|---|---|---|
| `lang` | `"ts"` | Required. The script is TypeScript. |

## Reactive state

Author a `const state = { ... }` object inside `<script>`. Property writes trigger a re-render.

Type annotations use TypeScript's `as` cast:

```typescript
const state = {
    count: 0 as number,
    name: 'Guest' as string,
    isActive: false as boolean,
    items: [] as string[],
    user: null as { name: string; email: string } | null,
};
```

Arrays and objects inside `state` are reactive. Push, splice, and property assignment all trigger updates.

Variables declared outside `state` are not reactive.

> **Note:** The `const state = { ... }` object is the only reactive scope. A `let count = 0` outside that object compiles, mutates, and never re-renders.

## Props

A `const props` object inside `<script>` declares the component's props. Parent templates pass values via attributes.

```typescript
const props = {
    label: '' as string,
    variant: 'primary' as 'primary' | 'secondary',
};
```

The component treats props as read-only.

## Directives

`.pkc` templates support the directives described below. See the [directives reference](directives.md) for full syntax.

| Directive | Behaviour |
|---|---|
| `p-if`, `p-else-if`, `p-else` | Conditional rendering. |
| `p-show` | Toggle CSS `display` without removing from the DOM. |
| `p-for` | Iterate; pair with `p-key` for stable identity. |
| `p-on:<event>` | Attach an event listener. |
| `p-model` | Two-way binding on form inputs. |
| `p-bind:<attr>` or `:<attr>` | Attribute binding. |
| `p-text` | Text content. |
| `p-html` | Raw HTML (unsafe). |
| `p-class` | Dynamic class list. |
| `p-style` | Dynamic inline style. |
| `p-ref` | Create a ref to a child element. |

## Lifecycle hooks

Export named functions to receive lifecycle callbacks. Every hook is optional. The compiler detects them by name and wires them to the component's lifecycle manager.

| Hook | Fires |
|---|---|
| `onConnected()` | After the custom element connects to the DOM. Once per mount cycle. |
| `onDisconnected()` | When the element is removed from the DOM. Resets so a re-mount fires `onConnected` again. |
| `onBeforeRender()` | Immediately before each re-render. |
| `onAfterRender()` | Immediately after each re-render finishes. |
| `onUpdated(changedProperties: Set<string>)` | After a re-render commits when one or more reactive properties changed; receives the set of changed property paths. |

## Events

Dispatch custom events from within the component:

```typescript
piko.dispatch('pp-counter:change', { count: state.count });
```

Listen to events with `p-on:<event>` in the template or `piko.bus.on(...)` globally.

## Shadow DOM and styling

The compiler scopes `<style>` to the shadow root by default. Styles do not leak out, and outer page styles do not leak in.

| Selector | Scope |
|---|---|
| `div { ... }` | Only matches inside the shadow root. |
| `:host { ... }` | The component's host element. |
| `:host(.active) { ... }` | The host element when it carries the matching class. |
| `::slotted(<selector>) { ... }` | Matches elements projected through slots. |

## Form association

Add `enable="form"` to the `<template>` to participate in forms. The component then responds to:

- `form.elements[name]` lookup.
- Validation (`checkValidity()`, `reportValidity()`).
- `formdata` submission.
- `reset` events.

Inside the component, call `piko.setFormValue(value)` to set the control's submitted value.

## Registration

Register components at bootstrap:

```go
ssr := piko.New(
    piko.WithComponents(
        components.PikoComponent{
            Tag:   "pp-counter",
            Asset: "components/pp-counter.pkc",
        },
    ),
)
```

The CLI scans `components/` by default. Piko registers external components installed via Go modules by their full import path.

## See also

- [How to reactivity](../how-to/client-components/reactivity.md).
- [How to events](../how-to/client-components/events.md).
- [How to event bus](../how-to/client-components/event-bus.md) for cross-component messaging.
- [Directives reference](directives.md).
- [Scenario 003: reactive counter](../showcase/003-reactive-counter.md), [Scenario 007: todo app](../showcase/007-todo-app.md), [Scenario 009: form wizard](../showcase/009-form-wizard.md).

Integration tests: [`tests/integration/pkc_serving/`](https://github.com/piko-sh/piko/tree/master/tests/integration/pkc_serving), [`tests/integration/asset_pipeline/`](https://github.com/piko-sh/piko/tree/master/tests/integration/asset_pipeline).
