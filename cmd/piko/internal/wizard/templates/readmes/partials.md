# `/partials`: Your reusable server views

This directory contains your **Partials**. These are reusable `.pk` files that are not directly accessible via a URL. Think of them as **server-side components**. You'll find `layout.pk` (the master HTML layout) and `feature-card.pk` (a reusable card component) already scaffolded here.

Partials are the building blocks you use to construct your pages and create a consistent, maintainable user interface.

While `pages/` define *what* to show at a specific URL, `partials/` define *how* to show reusable pieces of content.

---

## What should be a partial?

Any piece of server-rendered UI that you plan to use in more than one place, or that represents a logical, self-contained unit, is a great candidate for a partial.

Common examples include:
*   **Layouts:** A master `layout.pk` that defines the `<html>`, `<head>`, and `<body>` structure for your entire application. This is the most important partial in any project.
*   **Shared UI Elements:** A `top_menu.pk` for your site's navigation, a `footer.pk`, or a `sidebar.pk`.
*   **Data-Driven Components:** A `customer_card.pk` that takes a customer object and displays it, or a `product_list.pk` that iterates over a slice of products.
*   **Dynamic Sections:** A `table_customers.pk` that contains the logic for displaying a paginated table of customers, which can be re-fetched and updated independently.

## Using a partial

To use a partial within a page (or even another partial), you follow a two-step process:

### 1. Import the partial in Go

In the `<script>` block of the parent file, use a standard Go `import` statement. You must give the imported partial a local alias. This alias will be used as the tag name in your template.

```go
// In pages/index.pk
package main

import (
    "piko.sh/piko"
    // Import the layout partial and give it the alias "layout"
    layout "my-piko-app/partials/layout.pk"
)
```

### 2. Render the partial in the template

In the `<template>` block of the parent file, use the alias as a custom HTML tag. The `is` attribute, also set to the alias, tells the Piko compiler that this is a server-side partial to be rendered.

```html
<!-- In pages/index.pk -->
<template>
  <!-- The tag name "layout" matches the Go import alias -->
  <piko:partial is="layout">
    <!-- Content for the layout's slot goes here -->
  </piko:partial>
</template>
```

## Passing data with props

Partials are most useful when you pass data to them. This is done using **server props**, which are attributes in the template prefixed with `:server.`.

The name of the attribute (`:server.page_title`) corresponds to a `prop` tag in the child partial's `Props` struct.

```html
<!-- In pages/index.pk -->
<template>
  <piko:partial is="layout" :server.page_title="'Welcome Home'">
    ...
  </piko:partial>
</template>
```

The `layout.pk` partial would define a `Props` struct to receive this data:

```go
// In partials/layout.pk
type Props struct {
    PageTitle string `prop:"page_title"`
}
```

## Accepting content with `<piko:slot>`

Partials can render content passed into them from their parent by using the special `<piko:slot>` element. Use this for creating layouts.

```html
<!-- In partials/layout.pk -->
<template>
  <html>
    <body>
      <main>
        <!-- Any content inside the <piko:partial> tags in the parent -->
        <!-- will be rendered here. -->
        <piko:slot></piko:slot>
      </main>
    </body>
  </html>
</template>
```

---

### To learn more

Partials are fundamental to building complex applications with Piko. For a deeper dive into their capabilities, see the official documentation:

*   **[Partials: reusable server components](https://piko.sh/docs/reference/pk-file-format)**
*   **[Slots](https://piko.sh/docs/how-to/templates/slots)**
