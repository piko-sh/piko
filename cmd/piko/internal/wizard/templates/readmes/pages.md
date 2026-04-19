# `/pages`: Your application's routes

This is the most important directory for defining the structure of your website. Every **`.pk` (Piko Template)** file you create in this directory becomes a page in your application, and its file path is automatically mapped to a URL route. You'll find `index.pk` (the welcome page) and `!404.pk` (the error page) already scaffolded here.

These files are the entry points for your server-side rendering. Each page is responsible for fetching its own data, composing reusable partials, and defining the overall content for a specific URL.

---

## File-based routing

The routing is simple and intuitive, based directly on the file system:

-   `pages/index.pk` → `/`
-   `pages/about.pk` → `/about`
-   `pages/customers/index.pk` → `/customers`
-   `pages/customers/view.pk` → `/customers/view`

### Dynamic routes

To create dynamic routes (e.g., for viewing a specific customer by ID), you can use curly braces `{}` in your filenames. The name inside the braces becomes a parameter that you can access in your `Render` function.

-   `pages/customers/{customerID}.pk` → `/customers/:customerID` (e.g., `/customers/123`)

You can access this parameter from the `RequestData` object in your page's Go script:

```go
// In pages/customers/{customerID}.pk
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    customerIDStr := r.PathParam("customerID") // "123"
    // ... logic to fetch customer by ID ...
}
```

## The anatomy of a page

A page is a `.pk` file with a `<template>` and a Go `<script>` block. Its primary job is to orchestrate the rendering of a full HTML document, often by using a master layout partial.

```html title="pages/index.pk"
<template>
  <!-- 1. Use a layout partial for the main HTML structure -->
  <piko:partial is="layout" :server.page_title="state.Title">
    <!-- 2. This is the unique content for this page -->
    <div class="container">
      <h1>{{ state.Title }}</h1>
      <p>Your Piko application is running!</p>
    </div>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    // Import the layout partial
    layout "my-piko-app/partials/layout.pk"
)

// The Response struct defines the data available to the template
type Response struct {
    Title string
}

// The Render function is the entry point for the page
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    // 3. Fetch data and prepare the response for the template
    return Response{Title: "Welcome to Piko"}, piko.Metadata{}, nil
}
</script>
```

### Key responsibilities of a page:

1.  **Define the Route:** Its location in the `pages/` directory determines its URL.
2.  **Fetch Top-Level Data:** The `Render` function is responsible for fetching the primary data needed for that specific page.
3.  **Compose the UI:** The `<template>` block typically uses a layout partial and other server-side partials to construct the final HTML.
4.  **Set Page Metadata:** The `Render` function should return metadata like the page `<title>`, description, or status codes (e.g., 404 Not Found).

By convention, pages should contain minimal markup themselves. Their main role is to act as a "controller" that gathers data and passes it to reusable partials for the actual rendering.

---

### To learn more

To understand the full capabilities of `.pk` files, refer to the official Piko documentation:

-   **[Introduction to server templates (.pk)](https://piko.sh/docs/reference/pk-file-format)**
-   **[Partials: reusable server components](https://piko.sh/docs/reference/pk-file-format)**
-   **[Data fetching](https://piko.sh/docs/guide/data-fetching)**
