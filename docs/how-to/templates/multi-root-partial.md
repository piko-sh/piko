---
title: How to build a multi-root partial
description: Author a partial whose template has more than one root element, and understand how Piko inserts every root at the call site.
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
    <div>{{ state.Hello }} {{ state.Place }}</div>
    <main><piko:slot></piko:slot></main>
    <div>{{ state.Goodbye }} {{ state.Place }}</div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    Place string `prop:"place"`
}

type Response struct {
    Hello   string
    Goodbye string
    Place   string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        Hello:   "Hello",
        Goodbye: "Goodbye",
        Place:   props.Place,
    }, piko.Metadata{}, nil
}
</script>
```

The template has three top-level nodes. Piko keeps them as siblings. State for the partial - `Hello`, `Goodbye`, `Place` - comes from the partial's own `Render` function. Callers pass values in via declared props (here, `place`), not by writing to `state.*` directly.

## Invoke and inspect the output

```html
<piko:partial is="layout" place="world">
    <p>Body content</p>
</piko:partial>
```

Renders as:

```html
<div>Hello world</div>
<main><p>Body content</p></main>
<div>Goodbye world</div>
```

All three roots land at the call site as siblings. Surrounding markup in the parent template stays unaffected. To pipe a value from the parent's Response, prefix the prop with `:server.` (or `:` when the attribute name and prop name match): `<piko:partial is="layout" :server.place="state.City">`.

## Trade-offs

> **Note:** Invocation attributes on a multi-root partial flow to **every** root, not just the first. `<piko:partial is="layout" class="x">` adds `x` to all three siblings, because `renderFragmentChildren` passes the fragment's attributes through to each child element via `writeFragmentAttrs` (see `internal/render/render_domain/orchestrator.go` and `orchestrator_writers.go`). This is good for shared classes and `data-*` attributes, but set `id` on a single-root partial - duplicating an `id` across siblings produces invalid HTML.

Piko also stamps each root with internal `p-fragment` and `p-fragment-id` attributes so the renderer can correlate the inlined siblings back to the originating partial. These markers appear on the rendered output by design.

A single-root partial gives one DOM element to attach attributes to. Reach for it when the parent needs a unique `id`, an outer wrapper for ARIA relationships, or a single attachment point for parent CSS scope chains. Reach for a multi-root partial when the partial defines a *page region* (header + body + footer) where wrapping in an extra `<div>` would force semantically wrong HTML. Use it only when any invocation attributes you intend to pass make sense on every root.

## See also

- [PK file format reference](../../reference/pk-file-format.md) for partials and props.
- [How to layout partials](../partials/layout.md) for the matching layout pattern with slots.
- [How to control component attribute merging](attribute-merging.md) to predict how invocation attributes combine with each root.
