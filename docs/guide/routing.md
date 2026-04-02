---
title: Routing
description: File-based routing and dynamic routes in Piko
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 20
---

# Routing

Piko uses file-based routing where the structure of your `pages/` directory determines your application's routes.

## File-based routing

Every `.pk` file in the `pages/` directory automatically becomes a route:

```text
pages/
├── index.pk          → /
├── about.pk          → /about
├── blog/
│   ├── index.pk      → /blog
│   └── {slug}.pk     → /blog/:slug
└── docs/
    └── {...slug}.pk  → /docs/*
```

> **Note**: Routes are determined at build time, not runtime. The file structure you create becomes your URL structure.

## Static routes

The simplest routes are static, one file, one URL.

### Example: about page

**File**: `pages/about.pk`

```piko
<template>
  <piko:partial is="layout" :server.page_title="'About us'">
    <h1>About our company</h1>
    <p>We build amazing web applications with Piko.</p>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{
        Title: "About us | MyApp",
    }, nil
}
</script>
```

**Result**: Accessible at `http://localhost:8080/about`

### Nested static routes

Create nested routes with subdirectories:

```text
pages/
└── company/
    ├── about.pk      → /company/about
    ├── team.pk       → /company/team
    └── careers.pk    → /company/careers
```

## Dynamic routes

Use curly braces `{param}` to create dynamic route segments.

### Single parameter

**File**: `pages/blog/{id}.pk`

```piko
<template>
  <piko:partial is="layout" :server.page_title="state.Post.Title">
    <article>
      <h1>{{ state.Post.Title }}</h1>
      <p>{{ state.Post.Body }}</p>
    </article>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    "myapp/pkg/domain"
    layout "myapp/partials/layout.pk"
)

type Post struct {
    ID    string
    Title string
    Body  string
}

type Response struct {
    Post Post
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Get the ID from route parameters
    postID := r.PathParam("id")

    // Fetch the post
    post, err := domain.GetPostByID(postID)
    if err != nil {
        // Return 404 for missing posts
        return Response{}, piko.Metadata{Status: 404}, nil
    }

    return Response{
        Post: post,
    }, piko.Metadata{
        Title: post.Title,
    }, nil
}
</script>
```

**Matches**:
- `/blog/123` → `id = "123"`
- `/blog/my-post` → `id = "my-post"`
- `/blog/anything` → `id = "anything"`

**Does not match**:
- `/blog` → Use `pages/blog/index.pk`
- `/blog/foo/bar` → Use catch-all route

### Multiple parameters

**File**: `pages/users/{userId}/posts/{postId}.pk`

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    userID := r.PathParam("userId")   // "123"
    postID := r.PathParam("postId")   // "456"

    // Fetch user's specific post
    post, err := domain.GetUserPost(userID, postID)
    if err != nil {
        return Response{}, piko.Metadata{Status: 404}, nil
    }
    // ...
}
```

**Matches**: `/users/123/posts/456`

## Catch-all routes

Use `{...param}` to match multiple path segments.

### Example: documentation pages

**File**: `pages/docs/{...slug}.pk`

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Get all remaining path segments
    slug := r.PathParam("slug")

    // slug will contain the full path after /docs/
    // /docs/getting-started/installation → slug = "getting-started/installation"
    // /docs/api/reference → slug = "api/reference"

    doc, err := domain.GetDocBySlug(slug)
    if err != nil {
        return Response{}, piko.Metadata{Status: 404}, nil
    }
    // ...
}
```

**Matches**:
- `/docs/intro`
- `/docs/getting-started/install`
- `/docs/api/reference/metadata`
- Any depth under `/docs/`

## Route priority

When multiple routes could match a URL, Piko uses these priorities:

1. **Static routes** (exact match) - most specific
2. **Dynamic routes** (single parameter)
3. **Catch-all routes** (multiple parameters) - least specific

Within each category, routes with more path segments take precedence.

### Example:

```text
pages/blog/
├── featured.pk       # Priority 1: /blog/featured (static)
├── {id}.pk           # Priority 2: /blog/:id (dynamic)
└── {...slug}.pk      # Priority 3: /blog/* (catch-all)
```

- `/blog/featured` → Uses `featured.pk` (static)
- `/blog/123` → Uses `{id}.pk` (dynamic)
- `/blog/2024/january/post` → Uses `{...slug}.pk` (catch-all)

## Index routes

`index.pk` files handle the base path of their directory:

```text
pages/
├── index.pk          → /
└── blog/
    ├── index.pk      → /blog
    └── {id}.pk       → /blog/:id
```

## Accessing route parameters

### PathParam method

Use `r.PathParam(name)` to access route parameters:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Single parameter - returns empty string if not found
    id := r.PathParam("id")

    // Multiple parameters
    userID := r.PathParam("userId")
    postID := r.PathParam("postId")

    // Catch-all (full path)
    fullPath := r.PathParam("slug")

    // ...
}
```

### Getting all parameters

Use `r.PathParams()` to get all parameters as a map:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Get all path parameters as map[string]string
    params := r.PathParams()

    for key, value := range params {
        log.Printf("param %s = %s", key, value)
    }
    // ...
}
```

### With query parameters

Route parameters and query parameters use separate methods:

```go
// URL: /blog/123?sort=date&filter=published

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Route parameter
    postID := r.PathParam("id")              // "123"

    // Query parameters
    sort := r.QueryParam("sort")             // "date"
    filter := r.QueryParam("filter")         // "published"

    // Get multiple values for same query param
    tags := r.QueryParamValues("tag")        // []string{"go", "web"}

    // ...
}
```

## Common patterns

### Blog with categories

```text
pages/blog/
├── index.pk              → /blog (list)
├── {slug}.pk             → /blog/:slug (post)
└── category/
    └── {category}.pk     → /blog/category/:category
```

### User profiles

```text
pages/users/
├── index.pk              → /users (list)
├── {username}/
│   ├── index.pk          → /users/:username
│   ├── posts.pk          → /users/:username/posts
│   └── followers.pk      → /users/:username/followers
```

### Documentation

```text
pages/docs/
├── index.pk              → /docs (home)
└── {...slug}.pk          → /docs/* (all docs)
```

Combined with `content/docs/*.md` and `p-collection` for automatic page generation.
## Next steps

- [Learn about collections](/docs/guide/collections) → Content-driven routing
- [Server actions](/docs/guide/server-actions) → Handle POST requests
- [Metadata & SEO](/docs/guide/metadata) → Optimise routes for search engines
