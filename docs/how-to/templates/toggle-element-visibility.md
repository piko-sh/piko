---
title: How to toggle element visibility
description: Pick between p-if and p-show based on DOM presence, render cost, and preserved state.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 740
---

# How to toggle element visibility

Piko provides two directives for conditional visibility: `p-if` removes the element from the DOM, `p-show` hides it with CSS. The right choice depends on how often the element toggles, the cost of rendering, and whether the element holds state that must survive a hide. For the full directive surface see [directives reference](../../reference/directives.md).

## Pick the right directive

| Directive | DOM presence when false | Pick when |
|---|---|---|
| `p-if` | Element removed | Content rarely changes, rendering is expensive, or state on the hidden element should reset |
| `p-show` | Element present, hidden via CSS | Content toggles frequently, rendering is cheap, or input state must survive the hide |

`p-show` falsy elements receive `display: none !important;`, so any selector that overrides `display` does not break the hide.

## Hide rarely shown content with `p-if`

```html
<template>
    <div class="settings">
        <button p-on:click="state.AdvancedOpen = !state.AdvancedOpen">
            Toggle advanced
        </button>
        <section p-if="state.AdvancedOpen">
            <!-- Heavy markup that costs DOM time to render. -->
            <ExpensiveChart :data="state.Stats" />
        </section>
    </div>
</template>
```

The expensive subtree only mounts when the user opens advanced settings. Hidden settings never enter the DOM.

## Toggle interactive content with `p-show`

```html
<template>
    <div class="tabs">
        <button p-on:click="state.Tab = 'profile'">Profile</button>
        <button p-on:click="state.Tab = 'security'">Security</button>

        <div p-show="state.Tab == 'profile'">
            <input p-model="state.Name" />
        </div>
        <div p-show="state.Tab == 'security'">
            <input p-model="state.NewPassword" type="password" />
        </div>
    </div>
</template>
```

Both tab panels stay mounted, so input contents survive when the user switches tabs and back. Switching tabs does not pay the cost of remounting form controls or losing focus.

## Combine with `p-else` for branches

`p-if` pairs with `p-else-if` and `p-else` so a single directive group expresses a branch:

```html
<template>
    <div p-if="state.Status == 'loading'">Loading...</div>
    <div p-else-if="state.Status == 'error'">Failed: {{ state.Error }}</div>
    <div p-else>Ready</div>
</template>
```

`p-show` does not branch. Mutually exclusive `p-show` blocks need separate expressions on each branch.

## See also

- [Directives reference](../../reference/directives.md) for the full directive list and binding semantics.
- [Template syntax reference](../../reference/template-syntax.md) for the expression grammar inside directive values.
- [How to control partial refresh behaviour](partial-refresh.md) when toggling content needs to refresh from the server instead of branching locally.
