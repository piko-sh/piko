---
title: How to add a static route
description: Create a page that responds at a fixed URL.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 10
---

# How to add a static route

A static route is a page whose URL never changes. This guide covers adding one. For the mapping rules, see the [routing reference](../../reference/routing-rules.md).

## Create the page file

Every `.pk` file in `pages/` becomes a route. The directory structure determines the URL.

Create `pages/about.pk`:

```piko
<template>
  <piko:partial is="layout" :server.page_title="'About us'">
    <h1>About our company</h1>
    <p>Piko powers the site you are reading right now.</p>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{
        Title: "About us | MyApp",
    }, nil
}
</script>
```

The page responds at `http://localhost:8080/about` once the dev server reloads.

## Add nested static routes

Subdirectories translate to URL segments:

```
pages/
  company/
    about.pk       # /company/about
    team.pk        # /company/team
    careers.pk     # /company/careers
```

Piko needs no extra configuration. Each `.pk` file gets a route, and the file path determines the URL.

## Use `index.pk` for directory base paths

To respond at `/blog` instead of only at `/blog/something`, create `pages/blog/index.pk`:

```
pages/
  blog/
    index.pk       # /blog
    {id}.pk        # /blog/:id
```

## Set the HTTP status

Return a `piko.Metadata` with a non-zero `Status` to change the response code (for example, to return a "gone" response):

```go
return Response{}, piko.Metadata{Status: 410}, nil
```

## See also

- [Routing rules reference](../../reference/routing-rules.md) for the full mapping and precedence rules.
- [How to dynamic routes](dynamic-routes.md) for routes with parameters.
- [How to catch-all routes](catch-all-routes.md) for wildcard paths.
- [Scenario 001: hello world](../../showcase/001-hello-world.md) for the smallest runnable page.
