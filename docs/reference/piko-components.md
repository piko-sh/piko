---
title: Built-in Piko components
description: The components Piko ships and how to register them.
nav:
  sidebar:
    section: "reference"
    subsection: "components"
    order: 10
---

# Built-in Piko components

Piko ships six categories of components under the `piko.sh/piko/components` package. Each category has an opt-in helper that returns a list of component definitions callers pass to `piko.WithComponents(...)` at bootstrap. Source of truth: [`components/components.go`](https://github.com/piko-sh/piko/blob/master/components/components.go).

## Categories

| Helper | Category | Components |
|---|---|---|
| `components.Piko()` | Core demo components | `piko-counter`, `piko-card` |
| `components.Example()` | Documentation examples | `example-greeting` |
| `components.M2()` | Material 2 data tables | `m2-data-table`, `m2-data-table-row`, `m2-data-table-cell`, `m2-data-table-header`, `m2-data-table-pagination` |
| `components.M3E()` | Material 3 Expressive library (full) | See [M3E components reference](m3e-components.md) |
| `components.Dev()` | Dev-mode overlay | `piko-dev-widget` (auto-enabled by `WithDevWidget()`) |
| `components.All()` | Every category except `Dev` | Equivalent to `Piko() + Example() + M2() + M3E()` |

## Registration

```go
import "piko.sh/piko/components"

ssr := piko.New(
    piko.WithComponents(components.Piko()...),
    piko.WithComponents(components.M3E()...),
)
```

Helpers return `[]piko.ComponentDefinition`. Spread them into `WithComponents` with `...`. Mix and match categories as needed. Unused components do not ship to the browser when CSS and JS tree-shaking are active.

## `piko-counter`

A minimal reactive counter demonstrating state and event handling.

```html
<piko-counter label="Visitors" start="0" step="1" />
```

| Prop | Type | Default | Purpose |
|---|---|---|---|
| `label` | `string` | `"Count"` | Display label. |
| `start` | `number` | `0` | Initial value. |
| `step` | `number` | `1` | Increment amount. |

Source: [`components/piko/piko-counter.pkc`](https://github.com/piko-sh/piko/blob/master/components/piko/piko-counter.pkc).

## `piko-card`

A simple card wrapper with a named `header`, default content slot, and optional `footer`.

```html
<piko-card>
    <h3 p-slot="header">Card title</h3>
    <p>Card body content.</p>
    <div p-slot="footer">Footer content</div>
</piko-card>
```

Source: [`components/piko/piko-card.pkc`](https://github.com/piko-sh/piko/blob/master/components/piko/piko-card.pkc).

## `example-greeting`

A documentation-example component. Useful as a reference when authoring your own components.

Source: [`components/example/example-greeting.pkc`](https://github.com/piko-sh/piko/blob/master/components/example/example-greeting.pkc).

## M2 data tables

A small set of data-table primitives sufficient to render tabular data with sorting and pagination.

| Tag | Purpose |
|---|---|
| `m2-data-table` | Table container. |
| `m2-data-table-header` | Column header cell with sort affordances. |
| `m2-data-table-row` | Row container. |
| `m2-data-table-cell` | Body cell. |
| `m2-data-table-pagination` | Pager footer. |

See [Scenario 006: data table](../showcase/006-data-table.md) for a runnable walkthrough. Source files live under [`components/m2/`](https://github.com/piko-sh/piko/tree/master/components/m2).

## M3E components

Material 3 Expressive library (46 components). See the [M3E components reference](m3e-components.md) for the full list.

## Piko dev widget

An in-browser dev overlay that surfaces request timing, hot-reload status, and warnings. `piko.WithDevWidget()` enables the overlay. Registration of `components.Dev()` happens automatically.

Source: [`components/dev/piko-dev-widget.pkc`](https://github.com/piko-sh/piko/blob/master/components/dev/piko-dev-widget.pkc).

## See also

- [M3E components reference](m3e-components.md) for the Material 3 set.
- [Client components reference](client-components.md) for writing your own PKC components.
- [Bootstrap options reference](bootstrap-options.md) for `WithComponents` and `WithDevWidget`.
