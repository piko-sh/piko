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

> **Note:** In `.pk` (server-rendered) pages and partials, the `state` object stays read-only and reflects the server `Render` function's output. Visibility toggles driven from the page need a server round-trip (query parameter, partial reload, or action). For client-only toggles, lift the markup into a `.pkc` component where `state` reacts.

## Hide rarely shown content with `p-if`

In a `.pk` page, `state.X` stays read-only and reflects the server `Render` function. Drive a toggle by routing the user's choice through a query parameter and reloading the page (or a partial that wraps the section).

```piko
<!-- pages/settings.pk -->
<template>
    <div class="settings">
        <a p-if="!state.AdvancedOpen" href="?advanced=1">Show advanced</a>
        <a p-if="state.AdvancedOpen" href="?advanced=0">Hide advanced</a>

        <section p-if="state.AdvancedOpen">
            <piko:partial is="advanced_chart" :server.stats="state.Stats" />
        </section>
    </div>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    advanced_chart "myapp/partials/advanced_chart.pk"
)

type Props struct {
    Advanced bool `query:"advanced"`
}

type Response struct {
    AdvancedOpen bool
    Stats        Stats
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{AdvancedOpen: props.Advanced, Stats: loadStats()}, piko.Metadata{}, nil
}
</script>
```

The expensive subtree only mounts when the user opens advanced settings. Hidden settings never enter the DOM, and the partial composition keeps the chart's CSS, props, and slots isolated.

## Toggle interactive content with `p-show`

When a section toggles often and the user's local state must survive, lift the toggle into a `.pkc` client component. PKC files have reactive `state.X` mutation, which `.pk` pages do not.

`p-on:` in a PKC binds to a method reference, not an inline statement. The compiler emits `this.$$ctx.METHOD_NAME.call(this, ...)`, so the value must be a function name (or a function call expression). Inline assignments like `p-on:click="state.tab = 'profile'"` do not compile. Define a small method per action and reference it:

```piko
<!-- components/tabs.pkc -->
<template>
    <div class="tabs">
        <button p-on:click="showProfile">Profile</button>
        <button p-on:click="showSecurity">Security</button>

        <div p-show="state.tab == 'profile'">
            <slot name="profile"></slot>
        </div>
        <div p-show="state.tab == 'security'">
            <slot name="security"></slot>
        </div>
    </div>
</template>

<script lang="ts">
const state = {
    tab: 'profile'
};

function showProfile() {
    state.tab = 'profile';
}

function showSecurity() {
    state.tab = 'security';
}
</script>
```

The page invokes the component and provides the form fields as slot content. Both tab panels stay mounted, so input contents survive when the user switches tabs and back. Switching tabs does not pay the cost of remounting form controls or losing focus.

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
