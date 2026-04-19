---
title: How to scope and bridge component CSS
description: Use Piko's automatic CSS scoping, escape it with :global() or :deep(), stack multiple style blocks, and understand scope inheritance through nested partials.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 730
---

# How to scope and bridge component CSS

Piko scopes each component's `<style>` block automatically so a `.card` in one partial does not collide with a `.card` in another. This guide covers the scope mechanics, how to reach across the boundary when needed, and how nested partials inherit scopes. For directives that interact with classes (`p-class`, `p-style`) see [directives reference](../../reference/directives.md).

## Trust the default scope

> **Note:** Piko scopes by selector context, not by hashing class names. Your `.card` stays `.card` in the DOM and CSS; the framework prepends `[partial~="..."]` to your rules so they only match elements that own that scope ID.

Each `<style>` block compiles to selectors prefixed with the partial's scope attribute. A rule like `.card` becomes `[partial~="partials_card_abc123"] .card`, so it only matches elements inside that partial. Nothing else needs configuration. Write CSS, ship the partial, and the framework prevents class collisions.

## Reach into a nested partial with `:deep()`

The default scope does not penetrate a child partial. To style elements rendered by a nested partial from the parent's stylesheet, wrap the selector in `:deep()`:

```css
:deep(.child-component-class) {
    /* Targets elements inside child partials of this component. */
}
```

Use `:deep()` sparingly. Reaching across the scope boundary couples the parent's styles to the child's internal structure, so a refactor inside the child can break the parent.

## Escape scoping entirely with `:global()`

For selectors that must apply across the whole document (third-party libraries, global resets), wrap in `:global()`:

```css
:global(.external-lib-class) {
    /* Applies globally, ignoring the scope. */
}
```

`:global()` opts a single selector out of scoping. Use `<style global>` to opt the whole block out.

## Stack multiple `<style>` blocks

A single `.pk` file can carry more than one `<style>` block. The framework concatenates them in order:

```html
<style>
    .layout { display: grid; }
</style>

<style>
    .header { background: #333; }
</style>

<template>
    <div class="layout">
        <header class="header">...</header>
    </div>
</template>
```

Two blocks help when one rule set comes from a generator (theme tokens, grid system) and another is hand-written for the page.

## Understand scope inheritance through nested partials

Piko's scoping is less strict than Vue's. It does not hash class names, only selector contexts. The `partial` attribute on each element holds a space-separated list of owning partial IDs (its own scope plus every parent scope). When a partial nests inside another, the child's root element carries both scopes, so a parent rule like `[partial~="parent_abc"] .child-thing` reaches into the child root.

Two consequences:

- Parent styles can deliberately reach down without `:deep()` if the child's root is part of the targeted selector.
- Overly general parent selectors can bleed into children. Keep parent selectors anchored to specific class names.

## Aggregate CSS at the page level

Piko collects CSS from every partial reachable from a page and emits one stylesheet per request. A page that uses partials A and B (where A also uses C) ships A, B, and C combined. Duplicates collapse. A partial used twice on a page does not double its CSS.

This means stylesheet size scales with the page's partial graph, not with the partial count in the project. Splitting a large component across multiple partials does not bloat the output.

## See also

- [Directives reference](../../reference/directives.md) for `p-class` and `p-style`, which let template logic drive selectors.
- [Template syntax reference](../../reference/template-syntax.md) for the surrounding template grammar.
- [How to control component attribute merging](attribute-merging.md) for the matching attribute-side rules.
