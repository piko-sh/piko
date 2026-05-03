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
<template p-collection="blog" p-provider="markdown">
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

`pages/blog/index.pk` renders the landing page. Use `piko.GetAllCollectionItems` to fetch every item's frontmatter, then shape it into a typed slice your template can iterate:

```piko
<template>
  <h1>Recent posts</h1>
  <article p-for="(_, post) in state.Posts" p-key="post.Slug">
    <h2><a :href="post.URL" p-text="post.Title"></a></h2>
    <p class="meta">
      By <span p-text="post.Author"></span> on <span p-text="post.Date"></span>
    </p>
  </article>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type PostSummary struct {
    Title  string
    Slug   string
    URL    string
    Date   string
    Author string
}

type Response struct {
    Posts []PostSummary
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    items, err := piko.GetAllCollectionItems("blog")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    posts := make([]PostSummary, 0, len(items))
    for _, item := range items {
        posts = append(posts, PostSummary{
            Title:  stringField(item, "Title"),
            Slug:   stringField(item, "Slug"),
            URL:    stringField(item, "URL"),
            Date:   stringField(item, "date"),
            Author: stringField(item, "author"),
        })
    }

    return Response{Posts: posts}, piko.Metadata{Title: "Blog"}, nil
}

func stringField(item map[string]any, key string) string {
    if value, ok := item[key]; ok {
        if str, ok := value.(string); ok {
            return str
        }
    }
    return ""
}
</script>
```

`GetAllCollectionItems` returns each item's metadata map, omitting the parsed markdown body, which keeps the listing page cheap.

### Metadata key casing

The standard metadata keys produced by Piko are PascalCase: `Title`, `Slug`, `URL`, `Description`, `Tags`, `Draft`, `WordCount`, `ReadingTime`, `Sections`, `CreatedAt`, `UpdatedAt`, `PublishedAt`. Custom frontmatter fields preserve their YAML casing exactly - if your file says `author:` and `date:`, look them up as `"author"` and `"date"`. Mix-and-match casing in the same template is normal.

## Sort and filter

Use standard Go slice sorting and filtering inside `Render`:

```go
import "sort"

sort.Slice(items, func(i, j int) bool {
    return items[i]["date"].(string) > items[j]["date"].(string)
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

`SectionNode` has `Title`, `Slug`, `Children`, and `Level`. Render recursively in the template, and use `Slug` for the anchor target.

## See also

- [Collections reference](../../reference/collections-api.md) for every function and attribute.
- [About collections](../../explanation/about-collections.md) for why Piko treats content as a typed source.
- [How to querying and filtering](querying-and-filtering.md) for the Filter and SortOption APIs.
- [How to custom providers](custom-providers.md) for non-markdown sources.
- [How to catch-all routes](../routing/catch-all-routes.md) for arbitrary-depth content paths.
- [Scenario 015: markdown blog](../../../examples/scenarios/015_markdown_blog/) for a runnable walkthrough.
