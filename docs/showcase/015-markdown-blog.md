---
title: "015: Markdown blog"
description: A blog with markdown collections, generated routes, table of contents, and RSS.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 280
---

# 015: Markdown blog

A full blog built on a markdown collection. Each `.md` file becomes a route. The index page lists every post. Each post page renders the markdown body with a generated table of contents.

## What this demonstrates

- A declarative collection page (`p-collection="blog"`).
- `piko.GetAllCollectionItems` for the index page.
- `piko.GetSectionsTree` for a nested table of contents.
- `<piko:content />` for rendering the markdown body as HTML.
- An RSS feed generated from the same collection.

## Project structure

```text
src/
  pages/
    blog/
      index.pk          Listing page for all posts.
      {slug}.pk         Individual post page (declarative collection).
    feed.xml.pk         Generated RSS feed.
  content/
    blog/
      *.md              Post source files.
  partials/
    layout.pk           Shared site layout.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/015_markdown_blog/src/
go mod tidy
air
```

## See also

- [How to markdown collections](../how-to/collections/markdown.md).
- [Collections API reference](../reference/collections-api.md).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/015_markdown_blog).
