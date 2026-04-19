---
title: How to control component attribute merging
description: Predict and override how Piko merges id, class, and other attributes when invoking a partial.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 720
---

# How to control component attribute merging

When a parent template invokes a partial, the framework merges the invocation's attributes with the partial's root element using three rules. This guide explains the rules and how to predict the result. For the full template syntax see [template syntax reference](../../reference/template-syntax.md). For the partial format see [pk-file format reference](../../reference/pk-file-format.md).

<p align="center">
  <img src="../../diagrams/attribute-merging.svg"
       alt="The invocation contributes id, class, and other attributes; the partial root contributes its own id, class, and aria-label. Three rules combine them: id overrides with the invocation, class merges both lists, other attributes override on collision and join when only one side defines them. The merged result appears at the bottom."
       width="640"/>
</p>

## Override the `id`

The invocation's `id` replaces the partial's `id`:

```html
<!-- partials/card.pk -->
<template>
    <div id="original-id" class="card">...</div>
</template>

<!-- Invocation -->
<div is="card" id="custom-id"></div>

<!-- Output -->
<div id="custom-id" class="card">...</div>
```

The framework keeps `id` overrideable so the parent can produce unique IDs for accessibility (`aria-labelledby`) or DOM lookups.

## Merge classes additively

Classes from the partial and the invocation join into one space-separated list:

```html
<!-- partials/card.pk -->
<template>
    <div class="card base-style">...</div>
</template>

<!-- Invocation -->
<div is="card" class="highlighted special"></div>

<!-- Output -->
<div class="card base-style highlighted special">...</div>
```

The partial keeps full ownership of its base styles. The parent decorates without rewriting the partial.

## Override or join other attributes

Any other attribute that appears on both the invocation and the partial root takes the invocation's value. Attributes only on one side pass through:

```html
<!-- partials/card.pk -->
<template>
    <div class="card" aria-label="Default label">...</div>
</template>

<!-- Invocation -->
<div is="card" data-testid="my-card" aria-hidden="true"></div>

<!-- Output -->
<div class="card" aria-label="Default label" data-testid="my-card" aria-hidden="true">...</div>
```

`aria-label` keeps its partial value because the invocation does not override it. `data-testid` and `aria-hidden` pass through.

## Attributes the framework strips

Three attribute namespaces never reach the partial root:

- `is`: the partial-invocation marker itself.
- `server.*`: server-only attributes consumed at render time.
- `request.*`: request-scoped attributes that should not appear in HTML output.

If a custom attribute starts with `server.` or `request.`, rename it to avoid the strip behaviour.

## See also

- [Template syntax reference](../../reference/template-syntax.md) for the broader expression and directive surface.
- [PK file format reference](../../reference/pk-file-format.md) for partials and props.
- [How to scope and bridge component CSS](scoped-css.md) for the styling counterpart of attribute merging.
- [How to passing props to partials](../partials/passing-props.md) for typed prop binding.
