---
title: "003: Reactive counter"
description: Client-side counter with reactive state and Shadow DOM
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 40
---

# 003: Reactive counter

A client-side counter built with PKC (Piko Client Components). The component's state lives in the browser, updates happen instantly without server round-trips, and the DOM re-renders automatically when state changes.

## What this demonstrates

- **Reactive state**: a `state` object whose properties are tracked by Piko's reactivity proxy; updates are fine-grained. Only DOM nodes referencing the changed property update
- **Event handling**: binding DOM events with `p-on:click`
- **Conditional CSS classes**: `p-class="{ className: condition }"`
- **Shadow DOM encapsulation**: component styles and markup are isolated from the host page; no CSS leaks in or out
- **Custom elements**: the `name` attribute in a `.pkc` script maps to an HTML tag name (must contain a hyphen)
- **Host pages**: a `.pk` page mounts a PKC component using its custom element tag
- PKC components are **client-side only**: no Go execution on the server

## Project structure

```text
src/
  components/
    pp-counter.pkc            The counter component - template + TypeScript + CSS
  pages/
    index.pk                  Host page that mounts <pp-counter>
```

## How it works

The `.pkc` file has the same three-section structure as `.pk`, but runs client-side:

- **`<script lang="ts" name="pp-counter">`**: the `name` attribute determines the custom element tag. The `pp-` prefix is a Piko convention.
- **`state`**: declared as a `const` at the top level; Piko wraps it in a reactive proxy at runtime. Every property write triggers a re-render of dependent expressions.
- **`p-class="{ negative: state.count < 0 }"`**: conditionally adds the `negative` class when the count goes below zero.

The host page simply includes `<pp-counter></pp-counter>`. Piko detects the tag at compile time, bundles the JavaScript, and registers the element.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/003_reactive_counter/src/
go mod tidy
air
```
