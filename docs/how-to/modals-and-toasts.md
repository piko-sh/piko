---
title: How to configure modals and toasts
description: Enable the Modals and Toasts frontend modules, tune their behaviour, and trigger them from server actions.
nav:
  sidebar:
    section: "how-to"
    subsection: "frontend"
    order: 10
---

# How to configure modals and toasts

Piko ships three built-in frontend modules (`ModuleAnalytics`, `ModuleModals`, `ModuleToasts`). A project opts into the modules it uses and configures them at bootstrap. Actions can then trigger modal and toast behaviour by adding response helpers. This guide covers the modal and toast modules. See the [server actions reference](../reference/server-actions.md) for the helper mechanism and the [bootstrap options reference](../reference/bootstrap-options.md) for the surrounding API.

## Enable the modules

Pass each module into `WithFrontendModule`. Some modules accept a typed config value:

```go
ssr := piko.New(
    piko.WithFrontendModule(piko.ModuleModals, piko.ModalsConfig{
        DisableCloseOnEscape:   false,
        DisableCloseOnBackdrop: false,
    }),
    piko.WithFrontendModule(piko.ModuleToasts, piko.ToastsConfig{
        Position:        "top-right",
        DefaultDuration: 5000,
        MaxVisible:      5,
    }),
)
```

`WithFrontendModule` accepts an optional typed config as the second argument. Omit it to take the module's defaults.

## Modals configuration

`piko.ModalsConfig`:

| Field | Default | Behaviour |
|---|---|---|
| `DisableCloseOnEscape` | `false` | When `false`, pressing Escape closes the modal. Set to `true` to require an explicit close. |
| `DisableCloseOnBackdrop` | `false` | When `false`, clicking the backdrop closes the modal. Set to `true` to force the user to use a Close button. |

## Toasts configuration

`piko.ToastsConfig`:

| Field | Default | Purpose |
|---|---|---|
| `Position` | `"bottom-right"` | One of `"top-right"`, `"top-left"`, `"bottom-right"`, `"bottom-left"`, `"top-centre"`, `"bottom-centre"`. |
| `DefaultDuration` | `5000` | Display time in milliseconds. Set per-toast with the fourth argument to `showToast`. |
| `MaxVisible` | `5` | Maximum number of toasts visible at once; older ones dismiss. |

## Trigger from an action

Use `a.Response().AddHelper(...)` to queue client-side calls that the frontend runtime interprets after the action returns:

```go
func (a *CustomerSaveAction) Call(input CustomerSaveInput) (CustomerSaveResponse, error) {
    customer, err := saveCustomer(a.Ctx(), input)
    if err != nil {
        a.Response().AddHelper("showToast", "Could not save customer", "error")
        return CustomerSaveResponse{}, piko.NewError("save failed", err)
    }

    a.Response().AddHelper("showToast", "Customer saved", "success", 3000)
    a.Response().AddHelper("closeModal")
    a.Response().AddHelper("reloadPartial", "customer-list")

    return CustomerSaveResponse{ID: customer.ID}, nil
}
```

### `showToast` arguments

| Position | Type | Purpose |
|---|---|---|
| 1 | `string` | Message body. |
| 2 | `string` | Severity: `"success"`, `"error"`, `"warning"`, `"info"`. |
| 3 | `number` (optional) | Duration in milliseconds; overrides `DefaultDuration` for this toast. |

### `closeModal` arguments

Takes no arguments. Closes the top-most modal.

### `reloadPartial` arguments

| Position | Type | Purpose |
|---|---|---|
| 1 | `string` | Partial alias or CSS selector. |

The runtime finds the matching partial element in the DOM, refetches it from the server, and swaps it in without a full-page reload.

## Trigger from client TypeScript

Inside a PKC component or a PK `<script lang="ts">` block, call the runtime helpers directly:

```typescript
piko.showToast("Saved", "success");
piko.closeModal();
piko.reloadPartial("customer-list");
```

`piko.showModal(selector)` opens a modal by target id or selector. The modal's element must exist in the current DOM (typically rendered conditionally behind a `p-if`).

## Style modals and toasts

Both modules ship default styles that match Piko's design tokens. Override them with CSS custom properties at the document root:

```css
:root {
    --piko-toast-background: #1f2937;
    --piko-toast-color: #f9fafb;
    --piko-toast-border-radius: 8px;

    --piko-modal-backdrop: rgba(0, 0, 0, 0.6);
    --piko-modal-border-radius: 12px;
}
```

The exact variable names ship with the modules. Inspect the rendered DOM to see which variables the current version uses.

## See also

- [Server actions reference](../reference/server-actions.md) for `Response().AddHelper` and the full helper list.
- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithFrontendModule` and `WithCustomFrontendModule`.
- [How to forms](actions/forms.md) for a form flow that triggers toasts after submission.
