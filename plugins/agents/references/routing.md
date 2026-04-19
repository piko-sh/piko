# Routing

Use this guide when creating pages, setting up routes, or working with URL parameters.

## File-based routing

Every `.pk` file in `pages/` automatically becomes a route:

```text
pages/
├── index.pk          → /
├── about.pk          → /about
├── blog/
│   ├── index.pk      → /blog
│   └── {slug}.pk     → /blog/{slug}
└── docs/
    └── {slug}*.pk    → /docs/{slug}*
```

Routes are determined at build time, not runtime.

## Static routes

One file, one URL:

```piko
<!-- pages/about.pk -->
<template>
  <piko:partial is="layout" :server.page_title="'About Us'">
    <h1>About Our Company</h1>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "About Us"}, nil
}
</script>
```

Nested directories create nested routes:

```text
pages/company/about.pk    → /company/about
pages/company/team.pk     → /company/team
```

## Dynamic routes

Use `{param}` for dynamic segments:

**File**: `pages/blog/{slug}.pk`

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    slug := r.PathParam("slug")  // "my-first-post"
    post, err := domain.GetPostBySlug(slug)
    if err != nil {
        return Response{}, piko.Metadata{}, piko.NotFound("post", slug)
    }
    return Response{Post: post}, piko.Metadata{Title: post.Title}, nil
}
```

Multiple parameters: `pages/users/{userId}/posts/{postId}.pk`

```go
userID := r.PathParam("userId")
postID := r.PathParam("postId")
```

## Catch-all routes

Use `{name}*` (or regex `{name:.+}`) to match multiple path segments:

**File**: `pages/docs/{slug}*.pk`

```go
slug := r.PathParam("slug")
// /docs/getting-started/install → slug = "getting-started/install"
// /docs/api/reference           → slug = "api/reference"
```

## Route priority

1. **Static routes** (exact match) - most specific
2. **Dynamic routes** (single parameter)
3. **Catch-all routes** - least specific

```text
pages/blog/featured.pk    → /blog/featured (static, wins)
pages/blog/{id}.pk        → /blog/123 (dynamic)
pages/blog/{slug}*.pk     → /blog/2024/jan/post (catch-all)
```

## Index routes

`index.pk` handles the base path of its directory:

```text
pages/index.pk         → /
pages/blog/index.pk    → /blog
```

## Accessing parameters

### Path parameters

```go
id := r.PathParam("id")           // Single value
params := r.PathParams()           // map[string]string of all params
```

### Query parameters

```go
sort := r.QueryParam("sort")              // First value
tags := r.QueryParamValues("tag")          // []string of all values
```

## 404 handling

Return a typed error from `Render` to trigger a 404. `piko.Metadata.Status` is not read by the production response writer (only by the pikotest harness); non-200 responses come from the returned error.

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    post, err := domain.GetPost(id)
    if err != nil {
        return Response{}, piko.Metadata{}, piko.NotFound("post", id)
    }
    // ...
}
```

`piko.NotFound("post", id)` is shorthand for `&piko.NotFoundError{Resource: "post", ID: id}` and triggers the matching `!404.pk` (or `!error.pk`) page.

## Error pages

Custom error pages use the `!` prefix in `pages/`: `!404.pk` (exact), `!400-499.pk` (range), `!error.pk` (catch-all). Priority: exact > range > catch-all. Deeper directories win within each tier.

Return typed errors from Render to trigger them: `piko.NotFound(...)`, `piko.Forbidden(...)`, `piko.BadRequest(...)`, `piko.PageError(code, msg)`.

Collection pages (`p-collection`) automatically trigger 404 error pages when a slug isn't found.

## Redirects

### Client redirect (HTTP redirect)

```go
// Temporary redirect (302, default)
return Response{}, piko.Metadata{ClientRedirect: "/new-location"}, nil

// Permanent redirect (301)
return Response{}, piko.Metadata{
    ClientRedirect: "/new-location",
    RedirectStatus: 301,
}, nil
```

Valid redirect status codes: `301`, `302`, `303`, `307`.

### Server redirect (internal rewrite)

Renders a different page without changing the browser URL:

```go
return Response{}, piko.Metadata{ServerRedirect: "/login"}, nil
```

Server redirects are limited to 3 hops. If both are set, `ServerRedirect` takes precedence.

## Collections with routes

Combine `p-collection` with dynamic routes for content-driven pages:

```piko
<!-- pages/blog/{slug}.pk -->
<template p-collection="blog" p-provider="markdown">
  <piko:partial is="layout" :server.page_title="state.Title">
    <article>
      <h1>{{ state.Title }}</h1>
      <piko:content />
    </article>
  </piko:partial>
</template>
```

Each markdown file in `content/blog/` gets its own route automatically.

## Common patterns

### Blog with categories

```text
pages/blog/
├── index.pk              → /blog
├── {slug}.pk             → /blog/{slug}
└── category/
    └── {category}.pk     → /blog/category/{category}
```

### Documentation

```text
pages/docs/
├── index.pk              → /docs
└── {slug}*.pk            → /docs/{slug}*
```

## LLM mistake checklist

- Using `:slug` (Express style) or `{...slug}` (Next.js style) instead of `{slug}` / `{slug}*` in filenames
- Putting pages outside the `pages/` directory
- Using `r.QueryParam` when you need `r.PathParam` (or vice versa)
- Forgetting to handle 404 for dynamic routes when data is missing
- Setting `piko.Metadata{Status: 404}` instead of returning a typed error such as `piko.NotFound(...)` (the `Status` field is not read by the response writer)

## Related

- `references/pk-file-format.md` - Render function and Metadata
- `references/collections.md` - content-driven routing
- `references/server-actions.md` - handling POST requests
