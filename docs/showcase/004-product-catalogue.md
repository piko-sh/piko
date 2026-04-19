---
title: "004: Product catalogue"
description: Loops, conditionals, and partials for data-driven layouts
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 50
---

# 004: Product catalogue

A product catalogue page that renders a grid of product cards using server-side data, loops, conditionals, and partials.

## What this demonstrates

- **`p-for`** for iterating over a slice of structs; supports `item in collection` and `(index, item) in collection`
- **`p-if`** for conditional rendering: removes elements from the DOM entirely (not CSS hiding)
- **Partials** (`<piko:partial>`) for reusable server-side components
- **Dynamic attribute bindings** (`:attr="expr"`): the colon prefix distinguishes dynamic bindings from static string attributes
- The `prop:"..."` struct tag for mapping kebab-case HTML attributes to CamelCase Go fields
- The pass-through pattern where a partial returns its `Props` as the `Response`

## Project structure

```text
src/
  pages/
    index.pk                   The catalogue page - grid layout with p-for loop
  partials/
    product-card.pk            Reusable product card - props, p-if, dynamic attrs
```

## How it works

The page's `Render` function returns a `Products []Product` slice. The template iterates with `p-for`, passing each product's fields to the partial:

> **Note:** `<piko:partial>` is a meta element. The compiler resolves the `is` attribute to a partial file and inlines its rendered output; nothing in the `piko:` namespace appears in the final HTML.

```piko
<piko:partial is="product-card"
  :name="product.Name"
  :price="product.Price"
  :in-stock="product.InStock"
></piko:partial>
```

The partial declares its inputs with `prop:"..."` tags:

```go
type Props struct {
    Name    string `prop:"name"`
    InStock bool   `prop:"in-stock"`
}
```

The pass-through pattern (`return props, piko.Metadata{}, nil`) means the template accesses prop values via `state`.

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/004_product_catalogue/src/
go mod tidy
air
```

## See also

- [Directives reference](../reference/directives.md).
- [How to loops](../how-to/templates/loops.md).
