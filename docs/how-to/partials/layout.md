---
title: How to use a layout partial
description: Build a shared layout partial and wrap pages with it.
nav:
  sidebar:
    section: "how-to"
    subsection: "partials"
    order: 10
---

# How to use a layout partial

A layout partial contains the common chrome of your site (navigation, footer, document `<head>`) and every page wraps itself in it. This guide shows how to build one. See the [PK file format reference](../../reference/pk-file-format.md) for the full partial syntax.

## Create the layout partial

Place the file under `partials/`:

```piko
<!-- partials/layout.pk -->
<template>
  <!DOCTYPE html>
  <html lang="en">
    <head>
      <meta charset="UTF-8" />
      <meta name="viewport" content="width=device-width, initial-scale=1" />
      <title>{{ state.PageTitle }}</title>
    </head>
    <body>
      <header>
        <nav>
          <piko:a href="/">Home</piko:a>
          <piko:a href="/about">About</piko:a>
        </nav>
      </header>

      <main>
        <piko:slot />
      </main>

      <footer>
        <p>&copy; 2026 MyApp</p>
      </footer>
    </body>
  </html>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    PageTitle string `prop:"page_title" default:"MyApp"`
}

type Response struct {
    PageTitle string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{PageTitle: props.PageTitle}, piko.Metadata{}, nil
}
</script>
```

The `<piko:slot />` tag is where the page's content lands.

## Wrap a page with the layout

Import the partial by its file path and use `<piko:partial>` in the template:

```piko
<!-- pages/about.pk -->
<template>
  <piko:partial is="layout" :server.page_title="'About us'">
    <h1>About our company</h1>
    <p>Piko powers this site.</p>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{Title: "About us"}, nil
}
</script>
```

The import path is the Go module path plus the relative path to the `.pk` file. The name after `import` is the Go alias, and the `is="..."` attribute uses that alias.

## Pass dynamic props

Prefix the attribute with `:server.` to evaluate the expression:

```piko
<piko:partial is="layout" :server.page_title="state.Post.Title">
  <!-- page content -->
</piko:partial>
```

Static string props can drop the prefix when the field has `coerce:"true"`:

```piko
<piko:partial is="layout" page_title="About us">
```

## See also

- [How to nested partials](nested.md) for using partials inside other partials.
- [How to passing props to partials](passing-props.md) for struct tags, defaults, and query binding.
- [PK file format reference](../../reference/pk-file-format.md) for the full file format.
- [Scenario 005: blog with layout](../../showcase/005-blog-with-layout.md) for a runnable walkthrough.
