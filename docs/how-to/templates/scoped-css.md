---
title: How to scope and bridge component CSS
description: Use automatic CSS scoping, escape it with :global() or :deep(), and stack multiple style blocks.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 730
---

# How to scope and bridge component CSS

Piko scopes each component's `<style>` block automatically so a `.card` in one partial does not collide with a `.card` in another. For the underlying mechanism see [about PK files](../../explanation/about-pk-files.md#why-the-style-block-scopes-per-component). For directives that interact with classes (`p-class`, `p-style`) see [directives](../../reference/directives.md).

## Style elements inside a child partial

The default scope does not penetrate a nested partial. To target elements rendered by a nested partial from the parent's stylesheet, wrap the selector in `:deep()`:

```css
:deep(.child-component-class) {
    /* Targets elements inside child partials of this component. */
}
```

Use `:deep()` sparingly. Reaching across the scope boundary couples the parent's styles to the child's internal structure, so a refactor inside the child can break the parent.

## Apply a rule globally

For selectors that must apply across the whole document (third-party libraries, global resets), wrap in `:global()`:

```css
:global(.external-lib-class) {
    /* Applies globally, ignoring the scope. */
}
```

To opt the whole block out of scoping, use `<style global>`.

## Stack multiple `<style>` blocks

A single `.pk` file can carry more than one `<style>` block. Piko concatenates them in order:

```html
<style>
    .layout { display: grid; }
</style>

<style>
    .header { background: #333; }
</style>
```

Two blocks help when one rule set comes from a generator (theme tokens, grid system) and another is hand-written for the page.

## See also

- [About PK files](../../explanation/about-pk-files.md#why-the-style-block-scopes-per-component) for the scoping mechanism.
- [Directives reference](../../reference/directives.md) for `p-class` and `p-style`.
- [Template syntax reference](../../reference/template-syntax.md) for the surrounding template grammar.
- [How to control component attribute merging](attribute-merging.md) for the matching attribute-side rules.
