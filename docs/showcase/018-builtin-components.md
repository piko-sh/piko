---
title: "018: Built-in components"
description: Loading Piko's embedded counter and card components
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 160
---

# 018: Built-in components

A minimal page demonstrating Piko's built-in component library. The `piko-counter` and `piko-card` components ship with the framework. The `components.Piko()` helper loads them from an embedded filesystem.

## What this demonstrates

Calling `components.Piko()` returns the built-in components as a slice, ready to spread into `WithComponents`. Piko embeds the set in the binary, with no external dependencies. `piko.WithComponents()` registers component sets at startup. The `piko-counter` element is an interactive counter custom element. The `piko-card` element exposes named slots for `header`, the default slot, and `footer`. Named slots such as `slot="header"` and `slot="footer"` project content into the component's layout.

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

## See also

- [Built-in Piko components reference](../reference/piko-components.md).
