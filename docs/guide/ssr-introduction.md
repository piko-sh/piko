---
title: Server-side rendering
description: Build SEO-friendly, fast-loading web applications with Piko's SSR
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 25
---

# Server-side rendering

Piko generates fully-formed HTML on the server before sending it to the browser. This approach provides SEO visibility, fast initial loads, and works without JavaScript.

## What is SSR?

Server-Side Rendering (SSR) generates complete HTML on the server for each request. The browser receives ready-to-display content rather than an empty page that requires JavaScript to populate.

**Benefits:**

- **SEO.** Search engines index your content directly from the HTML
- **Performance.** Users see content immediately without waiting for JavaScript to execute
- **Social sharing.** Open Graph and meta tags are present in the initial response
- **Accessibility.** Content is available without JavaScript

## How Piko SSR works

Piko uses `.pk` files (PK templates) that combine Go server logic with HTML templates. At build time, .pk files compile to Go packages with `Render()` and `BuildAST()` functions. There is no runtime template parsing.

**Request lifecycle:**

1. Router matches the URL to a page entry
2. The page's `Render()` function executes, fetching data and returning a `Response` struct
3. The AST builder constructs the template tree using the response data
4. The render orchestrator streams HTML via QuickTemplate
5. The browser receives fully-formed HTML

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // Fetch data using request context
    posts, err := fetchRecentPosts(r.Context(), 10)
    if err != nil {
        return Response{}, piko.Metadata{}, err
    }

    return Response{
        Title: "Recent Posts",
        Posts: posts,
    }, piko.Metadata{
        Title:       "Recent Posts | My Blog",
        Description: "Browse our latest blog posts",
    }, nil
}
```

The template accesses the returned data via `state`:

```piko
<template>
    <h1>{{ state.Title }}</h1>
    <ul>
        <li p-for="post in state.Posts">
            <a :href="post.URL">{{ post.Title }}</a>
        </li>
    </ul>
</template>
```

## Progressive enhancement

Piko separates server and client concerns into distinct file types:

### .pk files (server-side)

.pk files handle:
- Data fetching from databases, APIs, and collections
- SEO metadata (title, description, Open Graph)
- Initial page rendering
- Form handling via server actions

Pages work fully without JavaScript. The HTML is complete and functional.

### .pkc files (client-side)

.pkc files (client components) compile to native Web Components with:
- Reactive state that updates the UI automatically
- Event handling for user interactions
- Client-side animations and transitions
- Shadow DOM encapsulation

Use client components to add interactivity where needed without making the entire page dependent on JavaScript.

## When to use each

**.pk files (server-side) for:**
- Content pages (blog posts, documentation, articles)
- Marketing and landing pages
- Product pages requiring SEO
- Forms processed by server actions
- Any page where search indexing matters

**.pkc files (client-side) for:**
- Interactive widgets (modals, dropdowns, accordions)
- Real-time features (counters, live updates)
- Complex form validation with immediate feedback
- Animations tied to user interaction

Most applications combine both: .pk files render the page structure and content, while .pkc components handle isolated interactive elements.

## Detailed references

For detailed reference on these topics, see:

- **[PK templates](/docs/guide/pk-templates)**: Render function, RequestData API, optional functions
- **[Directives](/docs/guide/directives)**: complete directive reference
- **[Metadata](/docs/guide/metadata)**: SEO, Open Graph, redirects, status codes

## Next steps

- [Learn .pk file syntax](/docs/guide/pk-templates) → Complete guide to template structure
- [Handle forms with server actions](/docs/guide/server-actions) → Process form submissions
- [Add interactivity with .pkc components](/docs/guide/client-components) → Build reactive widgets
