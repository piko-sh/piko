---
title: "019: M3E components"
description: Material Design 3 Essentials component library showcase
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 170
---

# 019: M3E components

A full showcase of the M3E (Material Design 3 Essentials) component library. The page catalogues over 40 components across six categories, each with its own page covering variants, states, and configuration options.

## What this demonstrates

`components.M3E()` loads the Material Design 3 Essentials component set, which follows the Material Design 3 specifications. `components.M2()` loads the Material Design 2 base layer, and `components.M3E()` adds the M3 components on top. The component set covers six categories, including actions, selection, navigation, containment, communication, and foundations. Variants include filled, outlined, and tonal styles, alongside enabled, disabled, and selected states, plus size variants (`xs`, `s`, `m`, `l`, `xl`) and shape variants. Slot-based composition provides icon slots, content slots, and header and footer slots. Form components (`checkbox`, `radio`, `text-field`, `select`) accept `name` and `value` attributes for form submission.

## Project structure

```text
src/
  cmd/main/
    main.go                           Registers M2 and M3E component sets
  pages/
    index.pk                          Component catalogue index
    actions/                          Button, FAB, toolbar, segmented button, etc.
    selection/                        Checkbox, radio, switch, slider, text field, etc.
    navigation/                       Tabs, navigation drawer, top app bar, etc.
    containment/                      Card, dialog, data table, carousel, etc.
    communication/                    Badge, progress, snackbar, tooltip, etc.
    foundations/                      Divider, elevation, icon, ripple
  partials/
    layout.pk                         Shared layout with sidebar navigation
    nav.pk                            Navigation partial
```

## How it works

The entry point loads both M2 and M3E component sets:

```go
ssr := piko.New(
    piko.WithComponents(components.M2()...),
    piko.WithComponents(components.M3E()...),
)
```

Each page demonstrates one component with its variants. For example, the button page shows all five styles:

```html
<m3e-button variant="elevated">Elevated</m3e-button>
<m3e-button variant="filled">Filled</m3e-button>
<m3e-button variant="filled-tonal">Tonal</m3e-button>
<m3e-button variant="outlined">Outlined</m3e-button>
<m3e-button variant="text">Text</m3e-button>
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/019_m3e_components/src/
go mod tidy
air
```

## See also

- [M3E component library reference](../reference/m3e-components.md).
