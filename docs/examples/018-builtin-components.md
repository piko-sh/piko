---
title: "018: Built-in components"
description: Loading Piko's embedded counter and card components
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 160
---

# 018: Built-in components

A minimal page demonstrating Piko's built-in component library. The `piko-counter` and `piko-card` components ship with the framework and are loaded from an embedded filesystem via `components.Piko()`.

## What this demonstrates

- **`components.Piko()`**: loading built-in components shipped with the framework; returns a slice to spread into `WithComponents`; embedded in the Piko binary, no external dependencies needed
- **`piko.WithComponents()`**: registering component sets at startup
- **`piko-counter`**: an interactive counter custom element
- **`piko-card`**: a card component with named slots (`header`, default, `footer`)
- **Slot-based composition**: named slots (`slot="header"`, `slot="footer"`) project content into the component's layout

## Project structure

```text
src/
  cmd/main/
    main.go                           Registers built-in components via components.Piko()
  pages/
    index.pk                          Page using piko-counter and piko-card
```

## How it works

The entry point loads the built-in component set:

```go
ssr := piko.New(
    piko.WithComponents(components.Piko()...),
)
```

The page uses the components directly in the template:

```html
<piko-counter></piko-counter>

<piko-card>
    <span slot="header">Hello from a built-in card</span>
    <p>Card content goes here.</p>
    <span slot="footer">Footer text</span>
</piko-card>
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/018_builtin_components/src/
go mod tidy
air
```
