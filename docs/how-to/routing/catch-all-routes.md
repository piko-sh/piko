---
title: How to add a catch-all route
description: Create a page that captures every trailing path segment under a base URL.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 30
---

# How to add a catch-all route

A catch-all route matches one or more trailing path segments. Piko captures the entire remainder of the URL into a single parameter. Use it for documentation paths, wildcard file serving, or dispatching to a content collection. See the [routing reference](../../reference/routing-rules.md) for precedence rules.

## The pattern

Name the file `{...paramName}.pk`:

```
pages/docs/{...slug}.pk
```

This matches every URL under `/docs/` that has at least one segment after the prefix.

## Read the captured path

`r.PathParam("slug")` returns the trailing path joined by slashes:

```go
package main

import (
    "piko.sh/piko"
    "myapp/pkg/domain"
)

type Response struct {
    Doc domain.Doc
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    slug := r.PathParam("slug")

    doc, err := domain.GetDocBySlug(slug)
    if err != nil {
        return Response{}, piko.Metadata{Status: 404}, nil
    }

    return Response{Doc: doc}, piko.Metadata{Title: doc.Title}, nil
}
```

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
  {id}.pk            # /blog/42                (dynamic wins)
  {...slug}.pk       # /blog/2024/january/post (catch-all matches)
```

## Use cases

Use a catch-all for markdown-driven sites by pairing it with a content collection. The page's `Render` looks up the slug in the collection and renders the matching document.

Use a catch-all for legacy URL compatibility. Inspect the captured path and redirect unknown subpaths to new locations by returning `piko.Metadata{Status: 301, Redirect: "..."}`.

Use a catch-all for virtual file serving when the content lives outside `pages/` under a single parent URL.

## Return 404 for unknown content

As with dynamic routes, return a non-zero status when the captured path does not resolve:

```go
if err != nil {
    return Response{}, piko.Metadata{Status: 404}, nil
}
```

## See also

- [Routing rules reference](../../reference/routing-rules.md).
- [How to dynamic routes](dynamic-routes.md) for fixed-depth parameters.
- [Collections reference](../../reference/collections-api.md) for content-driven page generation.
- [Scenario 015: markdown blog](../../showcase/015-markdown-blog.md) uses a catch-all route with a collection.
