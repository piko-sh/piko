---
title: How to control route priority
description: Order overlapping routes so the right page handles the request, using path depth, static-segment count, and registration logs.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 750
---

# How to control route priority

Piko orders overlapping routes by category first (static > dynamic > catch-all), then by tiebreakers within a category. For the base priority rules see [routing rules reference](../../reference/routing-rules.md). This guide covers the tiebreakers and how to verify the order at build time.

## Tiebreaker order within a category

When two routes share a category:

1. **Path depth.** More specific paths (more segments) take priority.
2. **Static segments.** Routes with more static segments win.
3. **Alphabetical.** Identical specificity falls back to lexical order.

## Resolve a nested-dynamic ambiguity

Given:

```text
pages/
├── {category}/
│   ├── index.pk           ->  /:category
│   └── {id}.pk            ->  /:category/:id
└── docs/
    └── {id}.pk            ->  /docs/:id
```

A request to `/docs/42` could match either `/docs/:id` or `/:category/:id`. The router picks `/docs/:id` because it carries one static segment (`docs`), and the other route carries zero. Static beats dynamic at the same depth.

## Verify the registration order

The build prints every registered route in priority order. Run a build and grep for the page log lines:

```text
INFO  Registering page route pattern=/docs/:id
INFO  Registering page route pattern=/docs
INFO  Registering page route pattern=/:category/:id
INFO  Registering page route pattern=/:category
```

If a page is missing or appears in an unexpected position, the file layout, not the router, is wrong. Check that each conflicting page is in the file system path the router would expect.

## Disambiguate by adding a static segment

When two dynamic routes collide and the priority rules do not pick the intended winner, the most reliable fix is to add a static segment to one of them. Renaming `pages/{category}/{id}.pk` to `pages/{category}/items/{id}.pk` changes the route to `/:category/items/:id`, which sorts above any sibling `/{a}/{b}` route.

## See also

- [Routing rules reference](../../reference/routing-rules.md) for the base rules and pattern syntax.
- [How to use catch-all routes](catch-all-routes.md) for the lowest-priority matcher.
- [How to use dynamic routes](dynamic-routes.md) for parameter binding.
