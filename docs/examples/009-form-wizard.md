---
title: "009: Form wizard"
description: Multi-step form with conditional rendering and client-side validation
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 100
---

# 009: Form wizard

A multi-step signup wizard that collects user information across three steps, with client-side validation at each stage. Built entirely as a PKC component.

## What this demonstrates

- **`p-if` / `p-else-if` / `p-else` chains**: controlling which wizard step is rendered; must be on consecutive sibling elements with no gaps
- **`p-model` on varied input types**: text, email, select, and checkbox; selects the correct DOM property and event based on input type
- **Lifecycle hooks**: `onConnected`, `onDisconnected`, `onUpdated` are plain functions with reserved names
- **`p-class` dynamic classes**: step indicator dots with `active` and `done` states
- **Client-side validation**: checking fields before advancing, with error feedback driven by reactive state
- State mutations are the sole mechanism for updating the UI; never manipulate the DOM directly in PKC

## Project structure

```text
src/
  components/
    pp-signup-wizard.pkc          The wizard component
  pages/
    index.pk                      Host page mounting <pp-signup-wizard>
```

## How it works

The wizard is driven entirely by `state.step`. Changing it triggers the `p-if`/`p-else-if`/`p-else` chain to swap DOM elements:

```html
<div p-if="state.step === 1" class="step step-1">...</div>
<div p-else-if="state.step === 2" class="step step-2">...</div>
<div p-else class="step step-3">...</div>
```

`p-model` binds different input types automatically:
- **Text/email**: listens to `input` event, reads `.value`
- **Select**: listens to `change` event, reads `.value`
- **Checkbox**: listens to `change` event, reads `.checked` (boolean)

Validation sets `state.error` to show a message; clearing it hides the error via `p-if`.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/009_form_wizard/src/
go mod tidy
air
```
