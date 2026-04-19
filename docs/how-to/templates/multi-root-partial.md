---
title: How to build a multi-root partial
description: Author a partial whose template has more than one root element, and understand how the framework inserts every root at the call site.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 760
---

# How to build a multi-root partial

Piko partials do not require a single root element. A `<template>` with multiple top-level nodes inserts each node at the invocation point. A partial can wrap pre-content, slotted body, and post-content in one file without a synthetic wrapper `<div>`. For the partial format see [pk-file format reference](../../reference/pk-file-format.md).

## Author the multi-root template

```piko
<!-- partials/layout.pk -->
<template>
    <div>{{ state.Hello }}</div>
    <main><piko:slot></piko:slot></main>
    <div>{{ state.Goodbye }}</div>
</template>
```

The template has three top-level nodes. The framework keeps them as siblings.

## Invoke and inspect the output

```html
<piko:partial is="layout" :state.Hello="'Welcome'" :state.Goodbye="'See you soon'">
    <p>Body content</p>
</piko:partial>
```

Renders as:

```html
<div>Welcome</div>
<main><p>Body content</p></main>
<div>See you soon</div>
```

All three roots land at the call site as siblings. Surrounding markup in the parent template stays unaffected.

## Trade-offs

> **Note:** With a multi-root partial, attribute merging applies to the first root only. `<piko:partial is="layout" class="x">` adds `x` to the first sibling and silently leaves the others untouched, so reach for a single-root partial when the parent needs to attach attributes uniformly.

A single-root partial gives one DOM element to attach attributes to (id, class, scope). A multi-root partial does not. The framework distributes attribute merging over the first root only, so `<piko:partial is="layout" class="x">` only adds `x` to the first `<div>`. Avoid multi-root partials when the parent needs to style the partial's "outside" as a unit.

A multi-root partial is the right shape when the partial defines a *page region* (header + body + footer) where wrapping in an extra `<div>` would force semantically wrong HTML.

## See also

- [PK file format reference](../../reference/pk-file-format.md) for partials and props.
- [How to layout partials](../partials/layout.md) for the matching layout pattern with slots.
- [How to control component attribute merging](attribute-merging.md) to predict how attributes attach to the first root.
