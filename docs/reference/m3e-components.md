---
title: M3E component library
description: The Material 3 Expressive components that ship with Piko.
nav:
  sidebar:
    section: "reference"
    subsection: "components"
    order: 20
---

# M3E component library

Piko ships a Material 3 Expressive (M3E) component library as 46 PKC components under [`components/m3e/`](https://github.com/piko-sh/piko/tree/master/components/m3e). Register the whole set with `components.M3E()`.

## Registration

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/components"
)

ssr := piko.New(
    piko.WithComponents(components.M3E()...),
)
```

Once registered, the components are available as custom elements in any PK template. Tree-shaking removes unused components from the shipped bundle.

## Component list

Components grouped by role. Each tag appears in PK templates exactly as written. See the individual `.pkc` files for prop and slot details.

### Layout and structure

| Tag | Role |
|---|---|
| `m3e-card` | Surface container. |
| `m3e-divider` | Visual separator. |
| `m3e-elevation` | Applied elevation surface. |
| `m3e-list` | List container. |
| `m3e-list-item` | Item inside a list. |
| `m3e-toolbar` | Toolbar surface. |
| `m3e-top-app-bar` | Top application bar. |
| `m3e-bottom-app-bar` | Bottom application bar. |
| `m3e-side-sheet` | Side-mounted sheet. |
| `m3e-bottom-sheet` | Bottom-mounted sheet. |

### Navigation

| Tag | Role |
|---|---|
| `m3e-navigation-bar` | Bottom navigation bar. |
| `m3e-navigation-drawer` | Side navigation drawer. |
| `m3e-navigation-rail` | Side navigation rail. |
| `m3e-tabs` | Tab bar. |
| `m3e-tab` | Individual tab. |
| `m3e-menu` | Menu container. |
| `m3e-menu-item` | Menu item. |
| `m3e-search` | Search bar. |

### Buttons

| Tag | Role |
|---|---|
| `m3e-button` | Standard button. |
| `m3e-button-group` | Multi-button row. |
| `m3e-icon-button` | Icon-only button. |
| `m3e-fab` | Floating action button. |
| `m3e-extended-fab` | Extended FAB (icon + label). |
| `m3e-fab-menu` | FAB that expands into a menu. |
| `m3e-split-button` | Split-action button. |
| `m3e-segmented-button` | Segmented control. |

### Inputs

| Tag | Role |
|---|---|
| `m3e-text-field` | Text input. |
| `m3e-checkbox` | Checkbox. |
| `m3e-radio` | Single radio button. |
| `m3e-radio-group` | Group of radios. |
| `m3e-switch` | Toggle switch. |
| `m3e-slider` | Range slider. |
| `m3e-select` | Select/dropdown. |
| `m3e-date-picker` | Date picker. |
| `m3e-time-picker` | Time picker. |

### Feedback

| Tag | Role |
|---|---|
| `m3e-snackbar` | Transient message. |
| `m3e-dialog` | Modal window. |
| `m3e-tooltip` | Hover tooltip. |
| `m3e-progress` | Progress indicator. |
| `m3e-loading-indicator` | Loading spinner. |
| `m3e-ripple` | Ripple overlay for interactive surfaces. |

### Content

| Tag | Role |
|---|---|
| `m3e-carousel` | Horizontal carousel. |
| `m3e-badge` | Count or status badge. |
| `m3e-chip` | Single chip (filter, input, suggestion). |
| `m3e-chip-set` | Collection of chips. |
| `m3e-icon` | Inline icon. |

## Props and slots

Each component declares its reactive state and slots in its `.pkc` file. The general conventions:

- A `<script lang="ts">` block declares state as a `const state = { ... }` object with `as Type` annotations. The compiler emits a static `propTypes` getter from the annotations so the runtime can coerce attribute strings back to typed values when external code sets attributes on the host element. Primitive fields (`string`, `number`, `boolean`) auto-reflect to and from HTML attributes; non-primitive fields stay internal.
- Slots use the standard HTML `<slot>` (default) or `<slot name="...">` (named) - these are real Web Components.
- Callers project content into named slots with the standard HTML `slot="name"` attribute on the content element.

Open the specific `.pkc` file for the state table. Browse the source at [`components/m3e/`](https://github.com/piko-sh/piko/tree/master/components/m3e).

## Theming

M3E components read palette and typography values from CSS custom properties on the root element. Project-level CSS that defines the same custom properties overrides the defaults.

## Scenarios using M3E

- [Scenario 019: M3E components](../../examples/scenarios/019_m3e_components/) demonstrates each component in isolation.
- [Scenario 020: M3E recipe app](../../examples/scenarios/020_m3e_recipe_app/) uses M3E for a multi-page app with routing, LLM calls, and SSE streaming.

## See also

- [Built-in components reference](piko-components.md) for the other built-in categories.
- [Client components reference](client-components.md) for the PKC format M3E components use.
- [Scenario 019](../../examples/scenarios/019_m3e_components/) and [Scenario 020](../../examples/scenarios/020_m3e_recipe_app/) for runnable examples.
