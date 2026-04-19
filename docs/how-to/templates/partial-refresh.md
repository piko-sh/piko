---
title: How to control partial refresh behaviour
description: Use the four refresh levels and element-level morph attributes to keep parent CSS scopes, JS state, and form data through a piko.partial().reload() cycle.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 770
---

# How to control partial refresh behaviour

`piko.partial('name').reload()` replaces the partial's content via DOM morphing. The morpher preserves state where possible, but parent-contributed attributes, JS-added attributes, and CSS scope chains need explicit handling to survive a refresh. This guide covers the four refresh levels and the element-level controls that compose with them. For the directive surface see [directives reference](../../reference/directives.md).

<p align="center">
  <img src="../../diagrams/partial-refresh-levels.svg"
       alt="Four columns, one per refresh level. Level zero (default) refreshes only children and preserves every root attribute and the parent CSS scope chain. Level one (pk-refresh-root) refreshes the root attributes while preserving the parent scope chain. Level two (pk-own-attrs) only refreshes the listed attributes; JS-owned attributes survive. Level three (pk-no-refresh-attrs) refreshes everything except the listed attributes. The scope chain rule preserves parent scope IDs across all levels."
       width="640"/>
</p>

## Why this matters

A nested partial inherits attributes from its parent's root. JavaScript may add more attributes at runtime. A naive refresh that replaces the root element loses both:

- Parent-contributed attributes disappear because the parent did not re-render.
- Runtime-added attributes disappear because the replace does not preserve them.
- CSS scope IDs stored in the `partial` attribute lose their parent context, so parent styles stop matching.

The graduated refresh levels solve each loss in turn.

## The four refresh levels

| Level | Attribute on the partial root | Behaviour |
|---|---|---|
| 0 (default) | none | Children update; root attributes preserved |
| 1 | `pk-refresh-root` | Root refreshes with CSS scope preservation |
| 2 | `pk-own-attrs="..."` | Only listed attributes update |
| 3 | `pk-no-refresh-attrs="..."` | All attributes update except the listed ones |

### Level 0 (default)

```html
<div data-partial="cart" data-partial-src="/partials/cart">
    <!-- Only this content updates on reload. -->
</div>
```

The default fits stable containers. The root's class, id, scope chain, and JS-added attributes all survive untouched.

```ts
piko.partial('cart').reload();
```

### Level 1 (root refresh with scope preservation)

Pick this when the root's own attributes change on refresh but parent CSS scopes must remain:

```html
<div data-partial="cart" pk-refresh-root class="{{ dynamicClass }}">
    ...
</div>
```

The merge keeps parent scopes attached:

```text
Before: partial="child_xyz parent_abc"
Server: partial="child_new"
After:  partial="child_new parent_abc"
```

Parent `:deep()` rules and scope-bound styles continue matching after the refresh.

### Level 2 (owned attributes)

Declare which attributes the partial template owns. Only listed attributes update on refresh. The rest survive:

```html
<div data-partial="cart" pk-own-attrs="class,data-count">
    ...
</div>
```

```ts
piko.partial('cart').reloadWithOptions({
    level: 2,
    ownedAttrs: ['class', 'data-count'],
});
```

Use Level 2 when the template owns some attributes (`class` driven by server state) and JS owns others (`data-active` toggled by a client interaction).

### Level 3 (preserve specific attributes)

List attributes that must not change. Everything else on the root refreshes:

```html
<div data-partial="cart" pk-no-refresh-attrs="data-initialized,aria-expanded">
    ...
</div>
```

Level 3 fits when most of the root refreshes but a small set of runtime flags must survive.

## Pick a level

| Scenario | Level |
|---|---|
| Stable container, only inner content updates | 0 |
| Root class or scope changes per refresh | 1 |
| Mix of template-owned and JS-owned attributes | 2 |
| Initialisation flags must survive | 3 |

## Control individual elements with `pk-no-refresh` and `pk-refresh`

Independent of the partial-level system, mark individual elements:

`pk-no-refresh` preserves an element and its subtree:

```html
<div data-partial="player">
    <span id="score">{{ state.Score }}</span>
    <span id="client-counter" pk-no-refresh data-count="0">0</span>
</div>
```

`pk-refresh` opts a child back into morphing inside a `pk-no-refresh` subtree:

```html
<div pk-no-refresh>
    <span>Preserved</span>
    <span pk-refresh>Updates normally</span>
</div>
```

## Drive refresh from JavaScript

```ts
// Simple reload with query params.
piko.partial('cart').reload({ highlight: 'new-item' });

// Advanced reload with explicit options.
piko.partial('cart').reloadWithOptions({
    data:       { highlight: 'new-item' },
    level:      1,
    ownedAttrs: ['class', 'data-count'],
});
```

Passing `level` overrides the level the framework would otherwise infer from attributes on the root.

## Style the loading state

The framework adds the `pk-loading` class to the partial element while the refresh is in flight:

```css
.pk-loading {
    opacity: 0.6;
    pointer-events: none;
}
```

The framework removes the class when the response replaces the content. A single rule covers every partial.

## Opt a form out of dirty tracking

The framework tracks form state so users get a warning before navigating away from unsaved changes. Search and filter forms do not benefit from this. Opt out with `pk-no-track`:

```html
<form pk-no-track p-on:submit.prevent="action.search()">
    <input type="text" name="query" />
    <button type="submit">Search</button>
</form>
```

## See also

- [Directives reference](../../reference/directives.md) for the directive list and the full set of internal `pk-*` attributes.
- [Template syntax reference](../../reference/template-syntax.md) for the expression grammar.
- [How to scope and bridge component CSS](scoped-css.md) for how scope chains travel through the refresh.
- [How to passing props to partials](../partials/passing-props.md) for the prop-binding side of partial composition.
