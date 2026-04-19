---
title: "003: Reactive counter"
description: Client-side counter with reactive state and Shadow DOM
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 40
---

# 003: Reactive counter

A client-side counter built with PKC (Piko client components). The component's state lives in the browser, updates happen without server round-trips, and the DOM re-renders automatically when state changes.

## What this demonstrates

Piko's reactivity proxy tracks properties on the `state` object. Updates are fine-grained, so only DOM nodes referencing the changed property update. Event handlers bind to DOM events through `p-on:click`. The `p-class="{ className: condition }"` directive toggles CSS classes conditionally.

Shadow DOM isolates component styles and markup from the host page, so no CSS leaks in or out. The `name` attribute in a `.pkc` script maps to a custom element tag name, which must contain a hyphen. A `.pk` page mounts a PKC component using its custom element tag. PKC components run client-side only, with no Go execution on the server.

## Project structure

```text
src/
  components/
    pp-counter.pkc            The counter component - template + TypeScript + CSS
  pages/
    index.pk                  Host page that mounts <pp-counter>
```

## How it works

The `.pkc` file has the same three-section structure as `.pk`, but runs client-side.

In `<script lang="ts" name="pp-counter">`, the `name` attribute determines the custom element tag. The `pp-` prefix is a Piko convention. Declare the `state` object as a `const` at the top level, and Piko wraps it in a reactive proxy at runtime. Every property write triggers a re-render of dependent expressions. The `p-class="{ negative: state.count < 0 }"` directive adds the `negative` class when the count goes below zero.

The host page includes `<pp-counter></pp-counter>`. Piko detects the tag at compile time, bundles the JavaScript, and registers the element.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/003_reactive_counter/src/
go mod tidy
air
```

## See also

- [Client components reference](../reference/client-components.md).
- [How to reactivity](../how-to/client-components/reactivity.md).
