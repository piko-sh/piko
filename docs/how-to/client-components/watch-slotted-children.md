---
title: How to react to slotted children
description: Attach a slot listener and drive slotted PKCs by writing their attributes.
nav:
  sidebar:
    section: "how-to"
    subsection: "client-components"
    order: 50
---

# How to react to slotted children

A PKC component receives projected content through `<slot>` elements. To react when slotted content changes (added, removed, reordered) and to drive the slotted elements from the parent, use the slot APIs on `pkc`. See [client components reference](../../reference/client-components.md#slots) for the API surface.

## Watch the default slot

`pkc.attachSlotListener(slot_name, callback)` registers a callback that fires once immediately with the current slotted elements, and again on every `slotchange`. Pass `''` for the default slot:

```pkc
<template name="pp-tab-strip">
    <div class="tabs">
        <slot></slot>
    </div>
</template>

<script lang="ts">
    const state = {
        active: '' as string,
    };

    pkc.onConnected(() => {
        pkc.attachSlotListener('', (elements) => {
            for (const tab of elements) {
                tab.toggleAttribute('selected', tab.getAttribute('value') === state.active);
            }
        });
    });
</script>
```

Each entry in `elements` is the actual element node assigned to the slot, in document order. The list is live and reflects the current state of the slot, not the state at the time of registration.

## Watch a named slot

Pass the slot's `name` attribute:

```pkc
<template name="pp-page-shell">
    <header><slot name="header"></slot></header>
    <main><slot></slot></main>
</template>

<script lang="ts">
    pkc.onConnected(() => {
        pkc.attachSlotListener('header', (elements) => {
            for (const item of elements) {
                item.setAttribute('density', state.density);
            }
        });
    });
</script>
```

## Drive slotted PKCs by writing their attributes

Slotted children are real DOM elements. If a slotted element is itself a PKC, setting an attribute on it writes through to its reactive state via the standard binding (see [share state between components](share-state-between-components.md)):

```typescript
pkc.attachSlotListener('', (elements) => {
    for (const card of elements) {
        card.setAttribute('theme', state.theme);
        card.toggleAttribute('compact', state.compact);
    }
});
```

The slotted PKC re-renders when its attribute changes. The parent does not need a JavaScript reference, an event subscription, or a shared store.

## React to a single slot change

To act only on additions or removals, compare the new list against the previous one:

```typescript
let previous: Element[] = [];

pkc.attachSlotListener('', (elements) => {
    const added = elements.filter((element) => !previous.includes(element));
    const removed = previous.filter((element) => !elements.includes(element));

    added.forEach((element) => element.setAttribute('theme', state.theme));
    removed.forEach((element) => element.removeAttribute('theme'));

    previous = elements;
});
```

The callback fires on every change including reorders, so retaining the previous list is the recommended pattern.

## Read slotted content on demand

`pkc.getSlottedElements(slot_name?)` returns the current assigned elements without registering a listener:

```typescript
function focusFirst() {
    const tabs = pkc.getSlottedElements();
    (tabs[0] as HTMLElement | undefined)?.focus();
}
```

`pkc.hasSlotContent(slot_name?)` returns a boolean for the empty/non-empty check:

```typescript
function showFallback() {
    return !pkc.hasSlotContent();
}
```

Both methods return `[]`/`false` before the shadow root attaches. Call them from `pkc.onConnected` or later.

## When the slot is itself reactive content

If your template renders the `<slot>` conditionally (`p-if`), the runtime can add or remove the slot element itself between renders. The listener attaches to the current slot element on registration. If the slot disappears and reappears across renders, re-attach the listener inside `pkc.onConnected` and let `attachSlotListener` re-bind. The runtime queues attaches when the shadow root is not yet available and replays them on attach (`flushPendingListeners`).

## See also

- [Client components reference](../../reference/client-components.md#slots) for the slot API surface.
- [How to share state between components](share-state-between-components.md) for the attribute-write pattern this guide builds on.
- [About reactivity](../../explanation/about-reactivity.md#what-this-enables-pkc-to-pkc-communication) for the design rationale.
