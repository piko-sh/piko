---
title: "019: M3E components"
description: Material Design 3 Essentials component library showcase
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 170
---

# 019: M3E components

A full showcase of the M3E (Material Design 3 Essentials) component library. Over 40 components are demonstrated across six categories, each with its own page showing variants, states, and configuration options.

## What this demonstrates

- **`components.M3E()`**: loading the Material Design 3 Essentials component set; follows Material Design 3 specifications
- **`components.M2()`**: loading the Material Design 2 base layer; `components.M3E()` adds the M3 components on top
- **Component categories**: actions, selection, navigation, containment, communication, foundations
- **Variants and states**: filled, outlined, tonal; enabled, disabled, selected; size variants (`xs`, `s`, `m`, `l`, `xl`) and shape variants
- **Slot-based composition**: icon slots, content slots, header/footer slots
- Form components (`checkbox`, `radio`, `text-field`, `select`) support `name`/`value` attributes for form submission

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
