# Collections

Use this guide when building content-driven pages with markdown, such as blogs, documentation, or knowledge bases.

## Overview

Collections connect markdown content files to Piko pages. Each markdown file in a collection directory becomes a route automatically.

## Setting up a collection

### 1. Create content files

Place markdown files in `content/{collection_name}/`:

```text
content/
└── blog/
    ├── first-post.md
    ├── second-post.md
    └── tutorial-intro.md
```

### 2. Add YAML frontmatter

Each markdown file needs frontmatter with metadata:

```markdown
---
title: "My First Post"
slug: "first-post"
date: "2025-01-15"
tags: ["go", "web"]
---

# My First Post

This is the content of the post...
```

### 3. Create the page template

```piko
<!-- pages/blog/{slug}.pk -->
<template p-collection="blog" p-provider="markdown">
  <layout is="layout" :server.page_title="state.Title">
    <article>
      <h1>{{ state.Title }}</h1>
      <time>{{ state.Date }}</time>
      <piko:content />
    </article>
  </layout>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

type Post struct {
    Title string
    Slug  string
    Date  string
    Tags  []string
}

type Response struct {
    Title string
    Date  string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    post := piko.GetData[Post](r)
    return Response{
        Title: post.Title,
        Date:  post.Date,
    }, piko.Metadata{Title: post.Title}, nil
}
</script>
```

## Key elements

### p-collection attribute

```piko
<template p-collection="blog" p-provider="markdown">
```

- `p-collection="name"` - matches the directory under `content/`
- `p-provider="markdown"` - specifies the content provider

### piko:content tag

```piko
<piko:content />
```

Renders the markdown body as HTML at that position in the template.

### piko.GetData[T]

Type-safe access to frontmatter data:

```go
post := piko.GetData[Post](r)
```

The type `T` must match the frontmatter fields.

## Listing collection items

Create an index page that lists all items:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    items, err := piko.GetAllCollectionItems("blog")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }
    return Response{Posts: items}, piko.Metadata{Title: "Blog"}, nil
}
```

## Searching collections

```go
results, err := piko.SearchCollection[Post](r, "blog", query)
```

## Automatic routing

Each markdown file generates a route based on the page's dynamic segment:

```text
content/blog/first-post.md    → /blog/first-post
content/blog/second-post.md   → /blog/second-post
```

## Missing collection items

When a visitor navigates to a collection route with a slug that doesn't exist (e.g., `/blog/nonexistent`), Piko automatically returns a 404 error and renders the appropriate `!404.pk` error page. No manual handling is needed.

## LLM mistake checklist

- Forgetting `p-provider` on the template tag (defaults to `"markdown"` if omitted)
- Mismatching collection name with directory name (case-sensitive)
- Forgetting `<piko:content />` tag (markdown body won't render)
- Using wrong type with `piko.GetData[T]` (must match frontmatter fields)
- Placing content files outside `content/` directory

## Related

- `references/routing.md` - dynamic routes and file-based routing
- `references/pk-file-format.md` - template and script structure
- `references/template-syntax.md` - piko:content and other built-in elements
