---
title: Collections
description: Work with content collections and markdown in Piko
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 65
---

# Collections

Collections are Piko's way of handling structured content like blog posts, documentation pages, products, or any other content-driven data. They enable you to manage markdown files with frontmatter and automatically generate routes.

> **Prerequisites:** This guide uses `piko.GetData[T](r)` for type-safe data access.

## What are collections?

A collection is a group of content items (usually markdown files) that share the same structure. Collections are perfect for:

- **Blogs** - Posts with titles, dates, authors
- **Documentation** - Pages with sections and order
- **Products** - Items with prices and descriptions
- **Team members** - Profiles with photos and bios

Collections in Piko:

- Live in the `content/` directory
- Use markdown with YAML frontmatter
- Can be queried in your Render functions
- Automatically generate routes

## Collection providers

Piko supports different collection providers. The most common is the **markdown provider**, which reads `.md` files from the `content/` directory. The markdown provider is the default when `p-provider` is not specified.

### Directory structure

```text
project/
├── content/
│   ├── blog/
│   │   ├── first-post.md
│   │   ├── second-post.md
│   │   └── third-post.md
│   └── docs/
│       ├── getting-started.md
│       └── routing.md
└── pages/
    ├── blog/
    │   ├── index.pk        # Blog listing page
    │   └── {slug}.pk       # Dynamic route for blog posts
    └── docs/
        └── {slug}.pk       # Dynamic route for docs
```

## Markdown files with frontmatter

Each markdown file in a collection has two parts:

1. **Frontmatter** (YAML metadata at the top)
2. **Content** (markdown body)

### Example blog post

**File**: `content/blog/hello-world.md`

```markdown
---
title: Hello World
slug: hello-world
date: 2024-01-15
author: John Doe
---

# Welcome to My Blog

This is the **content** of my first blog post.

- I can use markdown
- Lists work great
- So do [links](/about)

## Subheadings Too!
```

The frontmatter fields become properties you can access in your Render functions.

## Declarative collections: `p-collection`

Declarative collections create **dynamic routes automatically**. Each markdown file becomes its own page.

### Basic setup

**File**: `pages/blog/{slug}.pk`

```html
<template p-collection="blog" p-provider="markdown">
  <article class="blog-post">
    <header>
      <h1 p-text="state.Title">Blog Post Title</h1>
      <p class="meta">
        By <span p-text="state.Author">Author</span> on <span p-text="state.Date">Date</span>
      </p>
    </header>

    <!-- Render the markdown content -->
    <main class="content">
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
    URL    string
}

type Response struct {
    Title  string
    Slug   string
    Date   string
    Author string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Use piko.GetData to extract the page data type-safely
    post := piko.GetData[Post](r)

    return Response{
        Title:  post.Title,
        Slug:   post.Slug,
        Date:   post.Date,
        Author: post.Author,
    }, piko.Metadata{
        Title: post.Title + " | Blog",
    }, nil
}
</script>
```

### Key points

- `p-collection="blog"` - Specifies which collection to use
- `p-provider="markdown"` - Uses the markdown provider (this is the default)
- `piko.GetData[T](r)` - Type-safe extraction of collection data
- `<piko:content />` - Renders the markdown body as HTML

### Template attributes

| Attribute | Description | Default |
|-----------|-------------|---------|
| `p-collection` | Name of the collection (matches `content/{name}/` directory) | Required |
| `p-provider` | Provider to use for content fetching | `"markdown"` |
| `p-param` | URL parameter name for content lookup | `"slug"` |
| `p-collection-source` | Import alias for external module content | None |

### Custom parameter names

If your route uses a different parameter name (e.g., `{id}` instead of `{slug}`), use the `p-param` attribute:

```html
<template p-collection="products" p-param="id">
  <!-- Content for /products/{id} routes -->
</template>
```

### Generated routes

With this setup and three markdown files:

```text
content/blog/
├── hello-world.md       (slug: hello-world)
├── second-post.md       (slug: second-post)
└── learning-piko.md     (slug: learning-piko)
```

Piko automatically generates these routes:

- `/blog/hello-world`
- `/blog/second-post`
- `/blog/learning-piko`

The `{slug}` parameter matches the `slug` field in the frontmatter, or defaults to the filename without extension.

## Rendering markdown: `<piko:content />`

The `<piko:content />` tag renders the markdown body as HTML.

### Example

**Markdown File** (`content/blog/example.md`):

```markdown
---
title: Example Post
slug: example-post
---

# Main Heading

This is a paragraph with **bold** and *italic* text.

- List item 1
- List item 2

[A link](/about)
```

**PK File** (`pages/blog/{slug}.pk`):

```html
<template p-collection="blog" p-provider="markdown">
  <article>
    <h1 p-text="state.Title">Title</h1>
    <!-- Renders the markdown body -->
    <piko:content />
  </article>
</template>
```

**Rendered HTML**:

```html
<article>
  <h1>Example Post</h1>
  <h1>Main Heading</h1>
  <p>This is a paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
  <ul>
    <li>List item 1</li>
    <li>List item 2</li>
  </ul>
  <a href="/about">A link</a>
</article>
```

> **Tip**: Style the rendered content with CSS targeting the container element. All standard HTML tags are rendered.

## Accessing collection data

### In declarative collections

Use `piko.GetData[T](r)` to access the current item type-safely:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Type-safe extraction - returns zero value if data doesn't match
    post := piko.GetData[Post](r)

    if post.Title == "" {
        return Response{}, piko.Metadata{}, fmt.Errorf("post not found")
    }

    return Response{
        Title: post.Title,
    }, piko.Metadata{
        Title: post.Title,
    }, nil
}
```

### Table of contents

Extract headings from markdown content for navigation:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    post := piko.GetData[Post](r)

    // Get flat list of sections
    sections := piko.GetSections(r)

    // Or get hierarchical tree (better for nested ToC)
    sectionTree := piko.GetSectionsTree(r,
        piko.WithMinLevel(2),  // Start from h2
        piko.WithMaxLevel(4),  // Include up to h4
    )

    return Response{
        Title:    post.Title,
        Sections: sectionTree,
    }, piko.Metadata{}, nil
}
```

For the full SectionNode struct, SectionTreeConfig, and a template example, see [advanced collections](/docs/guide/advanced-collections).

## Listing collection items

To display a list of all items in a collection, use `piko.GetAllCollectionItems()`:

**File**: `pages/blog/index.pk`

```html
<template>
  <h1>Recent Blog Posts</h1>

  <!-- Loop through all posts -->
  <article p-for='(_, post) in state.Posts'>
    <h2>
      <a :href="post.URL" p-text="post.Title"></a>
    </h2>
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
    // Fetch all items from the collection
    items, err := piko.GetAllCollectionItems("blog")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    return Response{
        Posts: items,
    }, piko.Metadata{
        Title: "Blog | MyApp",
    }, nil
}
</script>
```

### How it works

1. `piko.GetAllCollectionItems("blog")` queries the `content/blog/` directory
2. Piko reads all `.md` files at build time
3. Returns a slice `[]map[string]any` containing metadata for all items
4. Loop through in your template with `p-for`

> **Note**: `GetAllCollectionItems` returns metadata only (no content ASTs), making it lightweight for listing pages.

## Filtering and sorting

When using `piko.GetAllCollectionItems()`, you receive a Go slice that you can filter and sort in your Render function.

For structured filtering with the `collection.Filter` API, sort options, and pagination, see [advanced collections](/docs/guide/advanced-collections).

### Filter by field

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    allPosts, err := piko.GetAllCollectionItems("blog")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    // Filter by author
    var johnsPosts []map[string]any
    for _, post := range allPosts {
        if author, ok := post["Author"].(string); ok && author == "John Doe" {
            johnsPosts = append(johnsPosts, post)
        }
    }

    return Response{
        Posts: johnsPosts,
    }, piko.Metadata{}, nil
}
```

### Sort by date

```go
import "sort"

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    posts, err := piko.GetAllCollectionItems("blog")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    // Sort by date (newest first)
    sort.Slice(posts, func(i, j int) bool {
        dateI, _ := posts[i]["Date"].(string)
        dateJ, _ := posts[j]["Date"].(string)
        return dateI > dateJ
    })

    return Response{
        Posts: posts,
    }, piko.Metadata{}, nil
}
```

### Limit results

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    allPosts, err := piko.GetAllCollectionItems("blog")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    // Get only the first 5 posts
    limit := 5
    if len(allPosts) > limit {
        allPosts = allPosts[:limit]
    }

    return Response{
        Posts: allPosts,
    }, piko.Metadata{}, nil
}
```

## Searching collections

Piko provides built-in fuzzy search for collections with configurable field weights, similarity thresholds, and search modes:

```go
results, err := piko.SearchCollection[Post](r, "blog", query,
    piko.WithSearchFields(
        piko.SearchField{Name: "Title", Weight: 2.0},
        piko.SearchField{Name: "Body", Weight: 1.0},
    ),
    piko.WithFuzzyThreshold(0.3),
)
```

For simple searches, use `piko.QuickSearch[Post](r, "blog", query)`. For the full search options API, see [advanced collections](/docs/guide/advanced-collections).

## Building navigation

Generate hierarchical navigation from collection metadata:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Get all docs metadata (lightweight - no content ASTs)
    allDocuments, err := piko.GetAllCollectionItems("docs")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    // Build navigation tree
    config := piko.DefaultNavigationConfig()
    navGroups := piko.BuildNavigationFromMetadata(allDocuments, config)

    return Response{
        Sidebar: navGroups.Groups["sidebar"],
    }, piko.Metadata{}, nil
}
```

For the full NavigationTree API (node methods, frontmatter configuration, NavigationConfig), see [advanced collections](/docs/guide/advanced-collections).

## Common patterns

### Blog with index and detail pages

**Index Page** (`pages/blog/index.pk`):

```html
<template>
  <h1>Blog Posts</h1>
  <article p-for='(_, post) in state.Posts'>
    <h2><a :href="post.URL" p-text="post.Title"></a></h2>
    <p p-text="post.Author"></p>
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

**Detail Page** (`pages/blog/{slug}.pk`):

```html
<template p-collection="blog">
  <article>
    <h1 p-text="state.Title"></h1>
    <p p-text="state.Author"></p>
    <piko:content />
  </article>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type BlogPost struct {
    Title  string `json:"title"`
    Author string `json:"author"`
    URL    string
}

type Response struct {
    Title  string
    Author string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    post := piko.GetData[BlogPost](r)

    return Response{
        Title:  post.Title,
        Author: post.Author,
    }, piko.Metadata{
        Title: post.Title,
    }, nil
}
</script>
```

### Product catalogue

```go
type Response struct {
    Products []map[string]any
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    products, err := piko.GetAllCollectionItems("products")
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    // Filter by category from query params
    category := r.QueryParam("category")
    if category != "" {
        var filtered []map[string]any
        for _, p := range products {
            if cat, ok := p["Category"].(string); ok && cat == category {
                filtered = append(filtered, p)
            }
        }
        products = filtered
    }

    return Response{Products: products}, piko.Metadata{}, nil
}
```

## Custom frontmatter fields

You can add any fields to your frontmatter:

```markdown
---
title: Advanced Tutorial
slug: advanced-tutorial
tags:
  - advanced
  - tutorial
  - piko
difficulty: intermediate
reading_time: 15
published: true
---
```

Match them in your struct using JSON tags:

```go
type Tutorial struct {
    Title       string   `json:"title"`
    Slug        string   `json:"slug"`
    Tags        []string `json:"tags"`
    Difficulty  string   `json:"difficulty"`
    ReadingTime int      `json:"reading_time"`
    Published   bool     `json:"published"`
    URL         string
}
```

> **Note**: Use JSON tags to map frontmatter field names (snake_case) to Go struct field names (PascalCase).
## Common mistakes

### Missing `p-collection` attribute

```html
<!-- Wrong: No collection specified -->
<template>
  <piko:content />
</template>

<!-- Correct: Collection specified -->
<template p-collection="blog">
  <piko:content />
</template>
```

### Wrong directory name

```go
// Wrong: Collection name doesn't match directory
items, _ := piko.GetAllCollectionItems("posts")  // But directory is content/blog/

// Correct: Matches content/blog/ directory
items, _ := piko.GetAllCollectionItems("blog")
```

### Missing JSON tags on struct fields

```go
// Wrong: Fields won't match frontmatter
type Post struct {
    Title string  // Frontmatter key is "title" (lowercase)
}

// Correct: Use JSON tags
type Post struct {
    Title string `json:"title"`
}
```

### Not using piko.GetData in collection pages

```go
// Wrong: state will be empty
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{}, piko.Metadata{}, nil
}

// Correct: Extract collection data
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    post := piko.GetData[Post](r)
    return Response{Title: post.Title}, piko.Metadata{}, nil
}
```

## Next steps

- [Advanced collections](/docs/guide/advanced-collections) → Structured filtering, sorting, pagination, navigation trees, and hybrid caching
