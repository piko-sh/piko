---
title: How to build a markdown-driven page
description: Read markdown files with frontmatter, generate routes from them, and render their content.
nav:
  sidebar:
    section: "how-to"
    subsection: "collections"
    order: 10
---

# How to build a Markdown-driven page

This guide shows how to turn a folder of markdown files into routes, with each file's frontmatter binding to the page's render context. See the [collections reference](../../reference/collections-api.md) for the full API.

## Create the content directory

Collections live under `content/`. One directory per collection:

```
content/
  blog/
    hello-world.md
    second-post.md
```

Each file has YAML frontmatter followed by the markdown body:

```markdown
---
title: Hello world
slug: hello-world
date: 2026-01-15
author: John Doe
---

# Welcome to my blog

This is the **content** of my first post.
```

The `slug` field becomes the URL segment. If the frontmatter omits `slug`, Piko uses the filename.

## Create the template

Place a PK file under `pages/` that declares the collection:

```piko
<template p-collection="blog">
  <article>
    <header>
      <h1 p-text="state.Title"></h1>
      <p class="meta">
        By <span p-text="state.Author"></span> on <span p-text="state.Date"></span>
      </p>
    </header>
    <main>
      <piko:content />
    </main>
  </article>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Post struct {
    Title  string `json:"title"`
    Slug   string `json:"slug"`
    Date   string `json:"date"`
    Author string `json:"author"`
}

type Response struct {
    Title  string
    Date   string
    Author string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    post := piko.GetData[Post](r)

    return Response{
        Title:  post.Title,
        Date:   post.Date,
        Author: post.Author,
    }, piko.Metadata{Title: post.Title + " | Blog"}, nil
}
</script>
```

Save this file as `pages/blog/{slug}.pk`. Piko generates one route per markdown file at build time:

- `/blog/hello-world`
- `/blog/second-post`

## Render the Markdown body

`<piko:content />` outputs the parsed markdown body as HTML. Style it with CSS targeting the container element.

## List all posts on an index page

`pages/blog/index.pk` renders the landing page. Use `piko.GetAllCollectionItems` to fetch every item's frontmatter:

```piko
<template>
  <h1>Recent posts</h1>
  <article p-for="(_, post) in state.Posts" p-key="post.Slug">
    <h2><a :href="'/blog/' + post.Slug" p-text="post.Title"></a></h2>
    <p class="meta">
      By <span p-text="post.Author"></span> on <span p-text="post.Date"></span>
    </p>
  </article>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Posts []map[string]any
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    posts, err := piko.GetAllCollectionItems("blog")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    return Response{Posts: posts}, piko.Metadata{Title: "Blog"}, nil
}
</script>
```

`GetAllCollectionItems` returns only frontmatter, not parsed markdown, which keeps the listing page cheap.

## Sort and filter

Use standard Go slice sorting and filtering inside `Render`:

```go
import "sort"

sort.Slice(posts, func(i, j int) bool {
    return posts[i]["Date"].(string) > posts[j]["Date"].(string)
})
```

For structured filtering with typed operators, see the [querying and filtering guide](querying-and-filtering.md).

## Generate a table of contents

Pull the current item's headings out of the markdown body:

```go
sections := piko.GetSectionsTree(r,
    piko.WithMinLevel(2),
    piko.WithMaxLevel(4),
)
```

`SectionNode` has `ID`, `Level`, `Title`, and `Children`. Render recursively in the template.

## See also

- [Collections reference](../../reference/collections-api.md) for every function and attribute.
- [How to querying and filtering](querying-and-filtering.md) for the Filter and SortOption APIs.
- [How to custom providers](custom-providers.md) for non-markdown sources.
- [How to catch-all routes](../routing/catch-all-routes.md) for arbitrary-depth content paths.
- [Scenario 015: markdown blog](../../showcase/015-markdown-blog.md) for a runnable walkthrough.
