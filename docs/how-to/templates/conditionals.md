---
title: How to conditionally render elements
description: Show, hide, and chain elements based on template expressions.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 10
---

# How to conditionally render elements

Piko templates have three directives for conditional rendering: `p-if`, `p-else-if`, and `p-else` (removes from the DOM), plus `p-show` (hides via CSS). This guide covers when to use each. See the [directives reference](../../reference/directives.md) for the full syntax.

## Render based on a boolean

`p-if` removes the element from the DOM when the expression is false:

```piko
<template>
  <p p-if="state.IsLoggedIn">Welcome back</p>
  <p p-if="!state.IsLoggedIn">Please log in</p>
</template>
```

For expressions more complex than negation, use comparison and logical operators:

```piko
<p p-if="state.Count > 0">You have items</p>
<p p-if="state.Status == 'active'">Status is active</p>
<p p-if="state.IsLoggedIn && state.IsPremium">Premium member</p>
<p p-if="state.Count == 0 || state.IsEmpty">No items</p>
```

## Chain alternatives with `p-else-if` and `p-else`

Chain conditions for mutually exclusive rendering. Only the first matching branch renders:

```piko
<template>
  <div class="status-indicator">
    <p p-if="state.Status == 'ok'" class="text-green">Everything is running smoothly</p>
    <p p-else-if="state.Status == 'warning'" class="text-yellow">Warning: check system logs</p>
    <p p-else-if="state.Status == 'error'" class="text-red">Error: system malfunction</p>
    <p p-else class="text-grey">Status unknown</p>
  </div>
</template>
```

`p-else-if` and `p-else` must immediately follow a `p-if` or another `p-else-if` at the same nesting level.

## Toggle visibility with `p-show`

`p-show` toggles CSS `display` instead of removing the element. The element stays in the DOM.

```html
<div p-show="state.IsExpanded" class="details-panel">
  Detailed content
</div>
```

Use `p-show` when the state toggles frequently: keeping the element mounted avoids DOM churn. Use `p-if` when the element is expensive to render and Piko displays it rarely.

## Branch on enumerated values

For finite enums, a `p-if` chain reads cleanly:

```piko
<div p-if="state.Role == 'admin'">
  <h2>Admin dashboard</h2>
</div>
<div p-else-if="state.Role == 'moderator'">
  <h2>Moderator panel</h2>
</div>
<div p-else-if="state.Role == 'user'">
  <h2>User dashboard</h2>
</div>
<div p-else>
  <h2>Guest view</h2>
</div>
```

For five or more branches, consider rendering a partial by name instead:

```piko
<piko:partial :is="'role-' + state.Role" />
```

(The partial names must match: `role-admin`, `role-moderator`, etc.)

## See also

- [Directives reference](../../reference/directives.md) for the full syntax.
- [How to loops](loops.md).
- [Template syntax reference](../../reference/template-syntax.md) for operators and expressions.
