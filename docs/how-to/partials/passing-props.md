---
title: How to pass props to partials
description: Declare typed props, set defaults, coerce strings to typed values, and bind props to query parameters.
nav:
  sidebar:
    section: "how-to"
    subsection: "partials"
    order: 30
---

# How to pass props to partials

A partial's `Props` struct defines the typed interface the partial exposes to its caller. Struct tags control required, default, coercion, and query-binding behaviour. See the [PK file format reference](../../reference/pk-file-format.md) for the broader context.

## Declare props

Inside the partial's `<script>`, declare a `Props` struct and accept it in `Render`:

```go
type Props struct {
    Title       string `prop:"title"`
    Description string `prop:"description"`
    Count       int    `prop:"count"`
    IsActive    bool   `prop:"is_active"`
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        Title:       props.Title,
        Description: props.Description,
        Count:       props.Count,
        IsActive:    props.IsActive,
    }, piko.Metadata{}, nil
}
```

The `prop:"name"` tag maps a Go field to an HTML attribute name. If omitted, the HTML attribute defaults to the lowercase field name.

## Pass values from the caller

Prefix dynamic values with `:server.` to have the compiler evaluate the expression:

```piko
<piko:partial is="card"
  :server.title="state.Product.Name"
  :server.description="state.Product.Description"
  :server.count="state.CartCount"
  :server.is_active="state.IsActive" />
```

## Coerce strings to typed values

Add `coerce:"true"` to accept raw string literals and convert them to the field's type:

```go
type Props struct {
    IsActive bool    `prop:"is-active" coerce:"true"`
    Count    int     `prop:"count" coerce:"true"`
    Price    float64 `prop:"price" coerce:"true"`
}
```

The caller can drop the `:server.` prefix:

```piko
<piko:partial is="display" is-active="true" count="42" price="19.99" />
```

## Require a prop

Mark a prop as required with `validate:"required"`. A missing attribute fails the build:

```go
type Props struct {
    Title string `prop:"title" validate:"required"`
    Theme string `prop:"theme"`
}
```

Build error:

```
error: Missing required prop 'title' for component <card>
```

## Set a default value

Static defaults use `default:"value"`:

```go
type Props struct {
    Theme string `prop:"theme" default:"light"`
    Size  string `prop:"size" default:"medium"`
}
```

For defaults that do not fit a string literal, use a factory function. The function must take no arguments and return the prop type:

```go
func GetDefaultOptions() AvatarOptions {
    return AvatarOptions{Size: 48, Shape: "circle"}
}

type Props struct {
    Options AvatarOptions `prop:"options" factory:"GetDefaultOptions"`
}
```

Piko calls the factory at render time when the caller omits the prop.

`default` and `factory` are mutually exclusive on a single field.

## Bind a prop to a query parameter

`query:"param"` binds a prop to a URL query parameter when the caller does not supply a value:

```go
type Props struct {
    Page int `prop:"page" query:"page" coerce:"true"`
}
```

A request to `/products?page=2` populates `Page = 2` when the caller omits an explicit `page` attribute.

Query binding requires `string`, `*string`, or a type with `coerce:"true"`. Slices and maps are not supported.

## Optional props with pointer types

Pointer types mark a prop as truly optional. The pointer is `nil` when omitted:

```go
type Props struct {
    Title    string              `prop:"title"`
    Subtitle *string             `prop:"subtitle"`
    Profile  *models.UserProfile `prop:"profile"`
}
```

Callers pass pointer-type props the same way. Piko takes the address automatically when the caller supplies a non-pointer expression.

## Combine tags

Multiple tags coexist on a single field:

```go
type Props struct {
    Title string `prop:"card-title" validate:"required"`
    Theme string `prop:"card-theme" default:"default"`
    Priority int `prop:"priority" coerce:"true" default:"1"`
    Options models.Config `prop:"options" factory:"GetDefaultConfig"`
    Page int `prop:"page" query:"page" coerce:"true"`
}
```

## Dynamic defaults inside `Render`

For values that depend on the request, set defaults inside `Render` instead of in struct tags:

```go
func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    if props.Locale == "" {
        props.Locale = r.AcceptLanguage()
    }
    return Response{Locale: props.Locale}, piko.Metadata{}, nil
}
```

## See also

- [How to layout partials](layout.md).
- [How to nested partials and slots](nested.md).
- [PK file format reference](../../reference/pk-file-format.md).
- [Scenario 004: product catalogue](../../showcase/004-product-catalogue.md) for a runnable walkthrough.
