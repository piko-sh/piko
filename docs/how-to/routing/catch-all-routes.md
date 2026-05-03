---
title: How to add a catch-all route
description: Use a content collection to capture every trailing path segment under a base URL.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 30
---

# How to add a catch-all route

A catch-all route matches one or more trailing path segments. Piko captures the entire remainder of the URL into a single named parameter. Use it for documentation paths, wildcard file serving, or dispatching to a content collection. See the [routing reference](../../reference/routing-rules.md) for precedence rules.

## Catch-alls require a collection directive

Piko only generates a working catch-all when the page declares a content collection through the `p-collection` template directive. When `p-collection` is present, Piko rewrites the trailing dynamic parameter from `{slug}` to `{slug:.+}` (a multi-segment match). The router then translates that to chi's bare-`*` catch-all internally.

A bare filename such as `pages/docs/{slug}*.pk` is not a stable form. Without a `p-collection` directive, the trailing parameter stays single-segment, and the literal `*` suffix is not understood by the route parser. Always pair a catch-all page with the directive.

## The pattern

Name the file with a single trailing parameter and declare the collection on the `<template>` element:

```
pages/docs/{slug}.pk
```

```piko
<template p-collection="docs" p-provider="markdown">
  <piko:partial is="layout" :server.page_title="state.Title">
    <article>
      <h1 p-text="state.Title"></h1>
      <piko:content />
    </article>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

type Doc struct {
    Title       string
    Description string
    Slug        string
}

type Response struct {
    Title string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    doc := piko.GetData[Doc](r)

    return Response{Title: doc.Title}, piko.Metadata{
        Title:       doc.Title,
        Description: doc.Description,
    }, nil
}
</script>
```

Piko reads the `p-collection` directive at build time, looks up the named collection through the configured provider, and feeds the matching item to `piko.GetData[T](r)` at render time. The runtime returns a not-found error when no item matches the URL parameter. The error satisfies `piko.ActionError` and resolves to a 404, so unknown paths route through the error-page system without any explicit lookup in user code.

Because the page declares a `p-collection`, Piko promotes `{slug}` to a multi-segment match. `r.PathParam("slug")` returns the trailing path joined by slashes:

| Request URL | `slug` |
|---|---|
| `/docs/intro` | `intro` |
| `/docs/getting-started/install` | `getting-started/install` |
| `/docs/api/reference/metadata` | `api/reference/metadata` |
| `/docs/` (no segment after prefix) | does not match; use `pages/docs/index.pk` for that |

## Combine with static and dynamic routes

Catch-all routes have the lowest precedence. Static and single-parameter routes at the same depth win first:

```
pages/blog/
  featured.pk        # /blog/featured          (static wins)
  {id}.pk            # /blog/42                (single-segment dynamic wins)
  {slug}.pk          # /blog/2024/january/post (catch-all matches when paired with a collection)
```

## Tune the collection lookup

Three template directives control how the page resolves a collection item from the URL:

- `p-collection="<name>"` (required) names the collection. Piko promotes the trailing parameter to a catch-all whenever this directive is present.
- `p-provider="<name>"` selects the provider that supplies the items. The default is `markdown`. The shipped providers are `markdown` and the test-only `mock_cms`; users can register their own.
- `p-param="<name>"` chooses which path parameter the provider uses for lookup. The default is `slug`, so `pages/docs/{slug}.pk` works without any override. Set `p-param="id"` when the file is `pages/products/{id}.pk` and the collection key is `id`.
- `p-collection-source="<alias>"` points at a Go import alias whose module supplies the markdown content. Use it when the content lives in a separate Go module instead of alongside the page.

```piko
<template p-collection="products" p-provider="markdown" p-param="id">
  ...
</template>
```

## Use cases

Use a catch-all for markdown-driven sites by declaring `p-collection`. Piko binds the matching item to the page automatically. `piko.GetData[T](r)` returns the typed view.

Use a catch-all for legacy URL compatibility. Inspect the captured path and redirect unknown subpaths to new locations by returning `piko.Metadata{ClientRedirect: "/new/path", RedirectStatus: 301}`.

Use a catch-all for virtual file serving when the content lives outside `pages/` under a single parent URL.

## Return 404 for unknown content

When the page declares `p-collection`, the runtime emits a not-found error automatically on a lookup miss. The error satisfies `piko.ActionError` and resolves to a 404 without any extra code. To force a 404 from your own logic, return one of the typed errors:

```go
return Response{}, piko.Metadata{}, &piko.NotFoundError{
    Resource: "doc",
    ID:       r.PathParam("slug"),
}
```

Only the test harness reads `piko.Metadata.Status`. Setting it in production keeps the response at 200.

## See also

- [Routing rules reference](../../reference/routing-rules.md).
- [About routing](../../explanation/about-routing.md) for the priority model that catch-all sits beneath.
- [How to dynamic routes](dynamic-routes.md) for fixed-depth parameters.
- [Collections reference](../../reference/collections-api.md) for content-driven page generation.
- [Scenario 015: markdown blog](../../../examples/scenarios/015_markdown_blog/) uses a catch-all route with a collection.
