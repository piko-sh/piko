---
title: How to add a dynamic route
description: Create a page whose URL segment captures a runtime value.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 20
---

# How to add a dynamic route

A dynamic route captures one URL segment at runtime using curly braces in the file name. This guide covers single and multi-parameter dynamic routes. See the [routing reference](../../reference/routing-rules.md) for the precedence rules.

## Single parameter

Create a file whose name contains `{paramName}`:

```
pages/blog/{slug}.pk
```

This matches `/blog/anything` and makes `anything` available as `slug`:

```piko
<template>
  <piko:partial is="layout" :server.page_title="state.Post.Title">
    <article>
      <h1>{{ state.Post.Title }}</h1>
      <p>{{ state.Post.Body }}</p>
    </article>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    "myapp/pkg/domain"
    layout "myapp/partials/layout.pk"
)

type Post struct {
    ID    string
    Title string
    Body  string
}

type Response struct {
    Post Post
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    slug := r.PathParam("slug")

    post, err := domain.GetPostBySlug(slug)
    if err != nil {
        return Response{}, piko.Metadata{}, &piko.NotFoundError{
            Resource: "post",
            ID:       slug,
        }
    }

    return Response{Post: post}, piko.Metadata{Title: post.Title}, nil
}
</script>
```

`r.PathParam("slug")` returns the captured value as a string.

## Multiple parameters

Every segment can be dynamic. Create `pages/users/{userId}/posts/{postId}.pk`:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    userID := r.PathParam("userId")
    postID := r.PathParam("postId")

    post, err := domain.GetUserPost(userID, postID)
    if err != nil {
        return Response{}, piko.Metadata{}, &piko.NotFoundError{
            Resource: "post",
            ID:       postID,
        }
    }

    return Response{Post: post}, piko.Metadata{}, nil
}
```

The URL `/users/123/posts/456` invokes this handler with `userID = "123"` and `postID = "456"`.

## Read query parameters alongside

Query parameters are separate from path parameters:

```go
// URL: /blog/123?sort=date&tag=go&tag=web

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    postID := r.PathParam("id")
    sort := r.QueryParam("sort")               // "date"
    tags := r.QueryParamValues("tag")          // []string{"go", "web"}

    return Response{PostID: postID, Sort: sort, Tags: tags}, piko.Metadata{}, nil
}
```

`QueryParam` returns the first value for a repeated parameter. Use `QueryParamValues` to get every value.

## Return a 404 when the parameter has no match

The page handler always writes `200 OK` on a successful render. To produce a 404, return a `piko.NotFoundError` (or any error implementing `piko.ActionError`). The router reads the status off the error type:

```go
if err != nil {
    return Response{}, piko.Metadata{}, &piko.NotFoundError{
        Resource: "post",
        ID:       slug,
    }
}
```

The default 404 page handles the response. See the [error pages guide](../error-pages.md) to customise it.

Only the test harness reads `piko.Metadata.Status`. The production response writer ignores it. Setting it without returning an error keeps the status at 200.

## See also

- [Routing rules reference](../../reference/routing-rules.md).
- [About routing](../../explanation/about-routing.md) for why filename patterns map to URL parameters.
- [How to catch-all routes](catch-all-routes.md) for paths with an unknown number of segments.
- [How to error pages](../error-pages.md) to customise 404 and other error responses.
- [Scenario 004: product catalogue](../../../examples/scenarios/004_product_catalogue/) uses dynamic slug routing end-to-end.
