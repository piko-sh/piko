---
title: Markdown Collections
description: How Piko turns a folder of markdown files into a fully rendered blog, with zero configuration.
date: 2026-03-12
tags:
  - collections
  - markdown
  - content
---

# Markdown Collections

One of Piko's most useful features is its collection system. Drop markdown files into a `content/` directory, add a collection page template, and the framework does the rest.

## How It Works

1. Place your `.md` files in `content/blog/` (or any name you choose)
2. Add YAML frontmatter at the top of each file for metadata
3. Create a `pages/blog/{slug}.pk` template with the `p-collection` directive
4. The framework discovers your files and generates a page for each one

## Frontmatter

Every markdown file starts with a YAML block between `---` fences:

```yaml
---
title: My Post Title
description: A short summary
date: 2026-03-12
tags:
  - example
---
```

These fields are available in your page's `Render` function as typed struct fields. You can add any custom fields you like - they all end up in the metadata map.

## The Collection Directive

The magic happens on the `<template>` tag:

```html
<template p-collection="blog" p-provider="markdown">
```

This tells Piko to look for a `content/blog/` directory, parse every markdown file in it, and generate a page for each one. The `<piko:content />` element in your template is replaced with the rendered markdown.

## No Configuration Required

There is no config file to write, no plugin to install, and no build step to configure. The collection system is built into the framework and works out of the box with the markdown provider.
