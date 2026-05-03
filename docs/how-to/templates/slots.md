---
title: How to project content into a partial with slots
description: Pass content from a page into a partial through a default slot or named slots.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 39
---

# How to project content into a partial with slots

Use `<piko:slot>` to let a partial accept arbitrary HTML from the page that invokes it. For the full PK syntax see [pk-file-format](../../reference/pk-file-format.md).

> **Note:** This guide covers `.pk` (server-rendered) partials only. `.pkc` files use the standard HTML `<slot>` element because they compile to native Web Components.

## Pass content into a partial

To accept content, declare a `<piko:slot />` inside the partial:

```piko
<!-- partials/card.pk -->
<template>
  <div class="card">
    <h2>Card title</h2>
    <piko:slot />
  </div>
</template>
```

The page invokes the partial and places content directly inside the tag:

```piko
<template>
  <piko:partial is="my_card">
    <p>This goes into the card's slot.</p>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
  "piko.sh/piko"
  my_card "myapp/partials/card.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
  return piko.NoResponse{}, piko.Metadata{}, nil
}
</script>
```

## Use multiple slots in one partial

To project content into more than one place, give each slot a `name`:

```piko
<!-- partials/layout.pk -->
<template>
  <div class="page-layout">
    <header><piko:slot name="header" /></header>
    <main><piko:slot /></main>
    <footer><piko:slot name="footer" /></footer>
  </div>
</template>
```

The page assigns content to a named slot either by wrapping with `<piko:slot name="...">` or by adding `p-slot="..."` to any element:

```piko
<template>
  <piko:partial is="my_layout">
    <piko:slot name="header">
      <h1>Main page header</h1>
    </piko:slot>

    <p>This goes to the default slot.</p>

    <div p-slot="footer">
      <p>Copyright 2025</p>
    </div>
  </piko:partial>
</template>
```

Use `p-slot` when the marked element is itself the content. Use `<piko:slot name="...">` when the content is a fragment.

## Use any tag name with `is`

`<piko:partial is="X">` is one form. Any tag name works as long as `is="..."` matches an imported alias:

```piko
<card is="my_card">
  <p>Slot content.</p>
</card>
```

## Conditionally render a slot

Combine `p-if` with the slot's wrapper to drop both when a condition is false. The wrapper suppresses the caller's sidebar content entirely when `state.HasSidebar` is false:

```piko
<template>
  <div class="layout">
    <aside p-if="state.HasSidebar">
      <piko:slot name="sidebar" />
    </aside>
    <main><piko:slot /></main>
  </div>
</template>
```

## Nest partials inside slot content

Slot content may itself invoke another partial. Piko resolves the inner invocation against the page's imports, not the layout's:

```piko
<!-- partials/page_layout.pk -->
<template>
  <div class="page">
    <main><piko:slot name="main-content" /></main>
    <aside><piko:slot name="sidebar" /></aside>
  </div>
</template>
```

```piko
<template>
  <piko:partial is="layout">
    <piko:slot name="main-content"><h1>Welcome</h1></piko:slot>
    <piko:slot name="sidebar">
      <piko:partial is="sidebar" :is-collapsible="true">
        <p>Hello, {{ state.Username }}!</p>
      </piko:partial>
    </piko:slot>
  </piko:partial>
</template>
```

## Provide a fallback when the parent supplies no content

To render default content when the parent does not supply any, place it inside the slot:

```piko
<template>
  <div class="card">
    <piko:slot>
      <p>This is the default fallback content.</p>
    </piko:slot>
  </div>
</template>
```

## Bind slot expressions to the invoker's state

Expressions inside slotted content resolve against the *invoking* component, not the partial. A `{{ state.X }}` in slotted content reads from the page that supplied the content:

```piko
<template>
  <piko:partial is="my_card">
    <p>{{ state.PageMessage }}</p>
  </piko:partial>
</template>
```

Piko resolves `state.PageMessage` against the page's `Response`, even though `my_card` renders the surrounding markup.

## Slot validation

The annotator emits warnings when slot content does not match what the partial declares.

If you target a slot that the partial does not define, you see:

```
Component <X> does not have a slot named 'Y'. Did you mean 'Z'?
```

The suggestion is the closest match, if any. Fix the typo on the caller side, or add a matching `<piko:slot name="Y" />` inside the partial.

If you place default-slot content inside a partial that has no default `<piko:slot />`, you see:

```
Component <X> does not have a default slot, but content was provided.
```

Either move the content into a named slot, or add a default `<piko:slot />` to the partial template.

## See also

- [Template syntax reference](../../reference/template-syntax.md), [directives reference](../../reference/directives.md): grammar for `<piko:slot>`, `p-slot`, and `p-if`.
- [PK file format reference](../../reference/pk-file-format.md): structure of `.pk` partials.
- [About .pk files](../../explanation/about-pk-files.md): why server-rendered partials exist.
- [About reactivity](../../explanation/about-reactivity.md): how slot expressions resolve against the invoker's state.
- [How to layout partials](../partials/layout.md), [how to nest partials](../partials/nested.md), [how to pass props](../partials/passing-props.md).
- [How to build a multi-root partial](multi-root-partial.md).
