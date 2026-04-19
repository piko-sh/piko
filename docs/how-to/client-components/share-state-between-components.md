---
title: How to share state between client components
description: Use HTML attribute writes to drive one PKC's state from another component or from a PK page script.
nav:
  sidebar:
    section: "how-to"
    subsection: "client-components"
    order: 40
---

# How to share state between client components

The runtime binds a PKC's reactive `state` bidirectionally to its host element's HTML attributes. To send state to another component, set the attribute on it. The receiving component's reactive state updates and the component re-renders. No event bus, no message envelope, no JS handle to keep in sync. See [client components reference](../../reference/client-components.md#state-and-html-attribute-binding) for the binding mechanics and [about reactivity](../../explanation/about-reactivity.md#what-this-enables-pkc-to-pkc-communication) for the rationale.

## When to use attribute writes versus the event bus

| You want | Reach for |
|---|---|
| Push a piece of state to another component | `setAttribute` / `toggleAttribute` (this guide) |
| Fan out a one-shot message with a payload | [`piko.bus`](event-bus.md) or [`piko.event.dispatch`](events.md) |
| Bubble a DOM event through ancestors | [`piko.event.dispatch`](events.md) |
| Notify multiple unrelated components of a system change | [`piko.bus`](event-bus.md) |

If the message you want to send maps cleanly to a state field, use an attribute write. Otherwise use a bus or DOM event.

## Drive a sibling component

Inside one PKC, target the sibling by ref and set its attribute:

```pkc
<template name="pp-search-toolbar">
    <input p-model="state.query" placeholder="Search" p-on:input="emit" />
</template>

<script lang="ts">
    const state = {
        query: '' as string,
    };

    function emit() {
        const results = document.querySelector('pp-search-results') as HTMLElement | null;
        results?.setAttribute('query', state.query);
    }
</script>
```

The `pp-search-results` component reads `query` as state and re-renders without any subscription:

```pkc
<template name="pp-search-results">
    <ul>
        <li p-for="item in filtered()" p-key="item.id">{{ item.name }}</li>
    </ul>
</template>

<script lang="ts">
    const state = {
        query: '' as string,
        all_items: [] as { id: number; name: string }[],
    };

    function filtered() {
        return state.all_items.filter(item => item.name.includes(state.query));
    }
</script>
```

## Drive a child component

A parent PKC writes child attributes the same way. Use `pkc.querySelector` (which is just `this.querySelector`) to reach the child:

```typescript
function highlight(id: string) {
    pkc.querySelectorAll<HTMLElement>('pp-card').forEach((card) => {
        card.toggleAttribute('selected', card.getAttribute('card_id') === id);
    });
}
```

`pkc.querySelector` and `pkc.querySelectorAll` reach the *light DOM* of the host - the elements the caller slots in. They do **not** see nodes rendered from the component's own template (those live inside the shadow root). The example above assumes the caller slots `<pp-card>` elements as children of the parent. For nodes inside the parent's template use `pkc.shadowRoot?.querySelector(...)` instead.

`toggleAttribute(name, force?)` is the shortcut for boolean state. Without the second argument it flips presence. With `true`/`false` it forces presence or absence.

## Drive a PKC from a `.pk` page script

PK script blocks have access to `pk.refs` (per-page DOM refs) and standard DOM APIs. Set the attribute on a PKC the same way:

```pk
<template>
    <pp-counter p-ref="counter" value="0"></pp-counter>
    <button p-on:click="reset">Reset</button>
</template>

<script lang="ts">
    function reset() {
        pk.refs.counter?.setAttribute('value', '0');
    }
</script>
```

The PKC's reactive state updates as if internal code had set it.

## Drive a PKC from a server action's response

A server action's `onSuccess` callback runs in the browser. Use it to set attributes on PKCs after the server commits:

```typescript
async function save() {
    const result = await action.posts.Update({ id: state.id, body: state.body }).call();
    pk.refs.toast?.setAttribute('message', `Saved at ${result.saved_at}`);
    pk.refs.toast?.toggleAttribute('visible', true);
}
```

## Read state back from a component

Reading is the inverse: `getAttribute` on the element returns the current attribute string, which is the rendered representation of state:

```typescript
const current = (document.querySelector('pp-search-toolbar') as HTMLElement | null)?.getAttribute('query');
```

For a typed read, prefer the component's own state object via the binding's other direction. Set the attribute and trust the round-trip, or expose a public getter on the component.

## Patterns to avoid

- **Reaching into another component's `state` object directly.** A PKC's `state` is private to the component. Cross-component reads happen through attribute reads or events, not through `otherComponent.state.x`.
- **Bus-broadcasting state changes.** If the message is "field X is now Y", an attribute write is more direct than a bus event plus a subscription that calls `setAttribute`.
- **Polymorphic attribute names.** Pick stable attribute names. The state field name and the attribute name are the same in snake_case (the recommended convention), so `state.is_loading` is `is_loading` on the element. Renaming one breaks the binding.

## See also

- [Client components reference](../../reference/client-components.md#state-and-html-attribute-binding) for the full attribute-binding mechanics.
- [About reactivity](../../explanation/about-reactivity.md#what-this-enables-pkc-to-pkc-communication) for the design rationale.
- [How to react to slotted children](watch-slotted-children.md) for parent-to-slotted-children attribute writes.
- [How to use the event bus](event-bus.md) for cases where attribute writes do not fit.
