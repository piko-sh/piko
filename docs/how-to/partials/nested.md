---
title: How to nest partials and use slots
description: Compose partials by placing them inside other partials and projecting content through slots.
nav:
  sidebar:
    section: "how-to"
    subsection: "partials"
    order: 20
---

# How to nest partials and use slots

Partials compose freely. One partial can include another, and the caller can project content into the partial through `<piko:slot>`. This guide covers the patterns. See the [PK file format reference](../../reference/pk-file-format.md) for the full syntax.

## Default slot

A partial with a bare `<piko:slot />` accepts arbitrary content from its caller:

```piko
<!-- partials/card.pk -->
<template>
  <article class="card">
    <piko:slot />
  </article>
</template>
```

The caller fills the slot by placing content between the opening and closing `<piko:partial>` tags:

```piko
<piko:partial is="card">
  <h2>Title</h2>
  <p>Body</p>
</piko:partial>
```

## Named slots

Use `name` on the slot to project different content into different regions:

```piko
<!-- partials/article.pk -->
<template>
  <article>
    <header>
      <piko:slot name="header" />
    </header>
    <main>
      <piko:slot />
    </main>
    <footer>
      <piko:slot name="footer" />
    </footer>
  </article>
</template>
```

The caller attaches content to a named slot with `p-slot="name"`:

```piko
<piko:partial is="article">
  <h1 p-slot="header">My post</h1>
  <p>The body of the post.</p>
  <p p-slot="footer">Posted on 31 January 2026</p>
</piko:partial>
```

Content without a `p-slot` attribute flows into the default (unnamed) slot.

## Slot fallback content

Place default content between `<piko:slot>` and `</piko:slot>`. It renders when the caller does not provide a slotted value:

```piko
<template>
  <div class="panel">
    <piko:slot>
      <p>No content provided.</p>
    </piko:slot>
  </div>
</template>
```

## Nesting partials

A partial can use another partial inside its own template. Import the child and use `<piko:partial is="child-alias">`:

```piko
<!-- partials/dashboard.pk -->
<template>
  <div class="dashboard">
    <piko:partial is="card">
      <h3>Revenue</h3>
      <p>{{ state.Revenue }}</p>
    </piko:partial>

    <piko:partial is="card">
      <h3>Active users</h3>
      <p>{{ state.ActiveUsers }}</p>
    </piko:partial>
  </div>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    card "myapp/partials/card.pk"
)

type Props struct {
    Revenue     string `prop:"revenue"`
    ActiveUsers int    `prop:"active_users"`
}

type Response struct {
    Revenue     string
    ActiveUsers int
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{Revenue: props.Revenue, ActiveUsers: props.ActiveUsers}, piko.Metadata{}, nil
}
</script>
```

Each level adds its own scope. Nothing leaks out, and the composition remains type-checked.

## Looping over partials

A `p-for` loop can render one partial per item:

```piko
<template>
  <div class="product-grid">
    <piko:partial
      p-for="product in state.Products"
      p-key="product.SKU"
      is="product-card"
      :server.sku="product.SKU"
      :server.name="product.Name"
      :server.price="product.Price" />
  </div>
</template>
```

Specify `p-key` when looping partials so Piko can keep identities stable across updates.

## See also

- [How to layout partials](layout.md).
- [How to passing props to partials](passing-props.md).
- [How to slots](../templates/slots.md) for the full slot directive reference.
- [PK file format reference](../../reference/pk-file-format.md).
- [Scenario 004: product catalogue](../../showcase/004-product-catalogue.md) and [Scenario 005: blog with layout](../../showcase/005-blog-with-layout.md).
