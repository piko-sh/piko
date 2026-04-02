---
title: "005: Blog with layout"
description: Layout composition with nested partials, slots, and theming
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 60
---

# 005: Blog with layout

A blog page composed from nested partials acting as reusable layouts, with content projected into slots and theming via CSS custom properties.

## What this demonstrates

- **Layout partials**: using a partial as a page layout with navigation, main area, sidebar, and footer
- **Nested partials**: the layout includes a navigation partial; each partial's CSS is scoped independently
- **Slot content projection**: default and named slots (`<piko:slot>`, `<piko:slot name="sidebar">`); named slots allow a single partial to accept content for multiple regions
- **Partial props**: partials receive data through props, not URL parameters
- **CSS custom properties**: defining theme variables in the layout that cascade naturally through the DOM; ideal for theming across partials
- **`p-html` directive**: rendering trusted HTML content from the server
- **`p-for` loops**: iterating over a tags slice

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

```html
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
