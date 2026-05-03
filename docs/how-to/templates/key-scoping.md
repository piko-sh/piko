---
title: How to scope keys with p-context
description: Use p-context to reset the key namespace inside a section so child p-key values do not collide with siblings or hydration scopes.
nav:
  sidebar:
    section: "how-to"
    subsection: "templates"
    order: 750
---

# How to scope keys with p-context

The `p-context` directive declares a key-namespace boundary. Inside the boundary, Piko interprets `p-key` values relative to the context, so two sections of a template can use the same key without collision. For the full directive list see [directives reference](../../reference/directives.md).

## Wrap a section in a context

```html
<template>
    <section p-context="'user-profile'">
        <h1 p-text="state.Name"></h1>
        <div p-key="'details'">
            <p p-text="state.Email"></p>
        </div>
    </section>
</template>
```

The `details` key resolves under the `user-profile` context. A separate section in the same template using `p-context="'order-summary'"` can also use `p-key="'details'"` without collision because each section has its own namespace.

## Use a dynamic context

Pass any expression that evaluates to a string. Dynamic contexts let one template generate multiple isolated scopes from a list:

```html
<template>
    <article
        p-for="(_, post) in state.Posts"
        p-key="post.ID"
        p-context="'post-' + post.ID"
    >
        <h2 p-text="post.Title"></h2>
        <div p-key="'body'" p-html="post.Body"></div>
    </article>
</template>
```

Each post gets a unique context so the inner `body` key namespaces under the post ID, not the parent template.

## When to add a context

Use `p-context` when:

- A partial mounts twice on the same page and the inner `p-key` values would otherwise collide.
- Hydration scopes need isolation so the client runtime tracks each instance independently.
- A dynamic list of components each carries internal keyed children.

A flat template with unique keys does not need `p-context`. Reach for it only once a collision appears.

## See also

- [Directives reference](../../reference/directives.md) for `p-key` and the directive list.
- [Template syntax reference](../../reference/template-syntax.md) for the expression grammar.
- [How to control partial refresh behaviour](partial-refresh.md) for how key scopes interact with morph-based refresh.
