---
title: "009: Form wizard"
description: Multi-step form with conditional rendering and client-side validation
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 100
---

# 009: Form wizard

A multi-step signup wizard that collects user information across three steps, with client-side validation at each stage. Built entirely as a PKC component.

## What this demonstrates

A `p-if`/`p-else-if`/`p-else` chain controls which wizard step Piko renders. The directives must sit on consecutive sibling elements with no gaps. The `p-model` directive works across text, email, select, and checkbox inputs, and picks the correct DOM property and event for each input type. Lifecycle hooks (`onConnected`, `onDisconnected`, `onUpdated`) are plain functions with reserved names.

Dynamic `p-class` styling drives the step indicator dots through `active` and `done` states. Client-side validation checks fields before advancing and feeds errors back through reactive state. State mutations are the sole mechanism for updating the UI. Never manipulate the DOM directly in PKC.

## Project structure

```text
src/
  components/
    pp-signup-wizard.pkc          The wizard component
  pages/
    index.pk                      Host page mounting <pp-signup-wizard>
```

## How it works

`state.step` drives the wizard entirely. Changing it triggers the `p-if`/`p-else-if`/`p-else` chain to swap DOM elements:

```piko
<div p-if="state.step === 1" class="step step-1">...</div>
<div p-else-if="state.step === 2" class="step step-2">...</div>
<div p-else class="step step-3">...</div>
```

`p-model` binds different input types automatically. Text and email inputs listen for the `input` event and read `.value`. Select elements listen for `change` and read `.value`. Checkboxes listen for `change` and read `.checked` as a boolean.

Validation sets `state.error` to show a message. Clearing it hides the error through `p-if`.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/009_form_wizard/src/
go mod tidy
air
```

## See also

- [Client components reference](../reference/client-components.md).
- [How to forms](../how-to/actions/forms.md).
