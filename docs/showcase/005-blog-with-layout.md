---
title: "005: Blog with layout"
description: Layout composition with nested partials, slots, and theming
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 60
---

# 005: Blog with layout

A blog page composed from nested partials acting as reusable layouts, with content projected into slots and theming via CSS custom properties.

## What this demonstrates

A partial can act as a page layout with navigation, main area, sidebar, and footer. Layouts nest, so the outer layout includes a navigation partial. Piko scopes each partial's CSS independently. Default and named slots (`<piko:slot>`, `<piko:slot name="sidebar">`) let a single partial accept content for multiple regions. Partials receive data through props instead of URL parameters.

CSS custom properties defined in the layout cascade naturally through the DOM, which suits theming across partials. The `p-html` directive renders trusted HTML content from the server, and `p-for` loops iterate over a tags slice.

## Project structure

```text
src/
  pages/
    index.pk                  Blog page that wraps content in the layout
    about.pk                  About page
    blog.pk                   Blog page
    contact.pk                Contact page
  partials/
    layout.pk                 Layout partial with nav, main, sidebar, footer
    navigation.pk             Navigation bar with active-link highlighting
```

## How it works

The page wraps its content inside the layout partial:

```piko
<piko:partial is="blog_layout" :title="state.Title">
    <article class="blog-post">...</article>
    <div p-slot="sidebar" class="sidebar-content">...</div>
</piko:partial>
```

Content without `p-slot` goes into the layout's default slot. Content with `p-slot="sidebar"` fills the named slot. Named slots support fallback content for when the caller provides nothing.

The layout defines CSS custom properties that the navigation partial references with fallbacks:

```css
.layout { --nav-bg: #1f2937; }
.site-nav { background-color: var(--nav-bg, #1f2937); }
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/005_blog_with_layout/src/
go mod tidy
air
```

## See also

- [PK file format reference](../reference/pk-file-format.md).
- [How to layout partials](../how-to/partials/layout.md).
