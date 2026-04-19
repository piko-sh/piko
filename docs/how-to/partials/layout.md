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
  <piko:partial is="layout" page_title="About us">
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

Piko recognises any attribute on `<piko:partial>` whose name matches a field on the partial's `Props` struct (after the `prop:"..."` mapping) as a prop. The leading colon `:` controls whether Piko evaluates the value as an expression or treats it as a literal string. The optional `server.` and `request.` prefixes opt the attribute into prop-only treatment. Use them when the prop name clashes with a real HTML attribute, or when you want a build-time error instead of silent forwarding to the partial root. A bare attribute that does not match any prop name forwards as a plain HTML attribute on the partial root.

## Pass static and dynamic props

Bare attribute names work for any prop. A literal string uses the prop name without a leading colon:

```piko
<piko:partial is="layout" page_title="About us">
```

Add a leading colon (`:`) to evaluate the value as an expression instead:

```piko
<piko:partial is="layout" :page_title="state.Post.Title">
  <!-- page content -->
</piko:partial>
```

The `server.` prefix is interchangeable when you want to be explicit that an attribute is a prop:

```piko
<piko:partial is="layout" :server.page_title="state.Post.Title">
```

The compiler treats `server.page_title` and `page_title` as the same prop. Pick whichever convention reads better in your template. Examples in this guide use the bare form for brevity.

For non-string fields (bool, int, float) bound to a literal, add `coerce:"true"` to the prop's struct tag so Piko converts the string to the field's type. The colon-versus-no-colon distinction still applies.

## See also

- [How to nested partials](nested.md) for using partials inside other partials.
- [How to passing props to partials](passing-props.md) for struct tags, defaults, and query binding.
- [PK file format reference](../../reference/pk-file-format.md) for the full file format.
- [About PK files](../../explanation/about-pk-files.md) for why partials are first-class PK files.
