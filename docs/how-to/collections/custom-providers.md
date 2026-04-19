---
title: How to write a custom collection provider
description: Back a Piko collection with a SQL database, a JSON API, or any data source other than markdown.
nav:
  sidebar:
    section: "how-to"
    subsection: "collections"
    order: 30
---

# How to write a custom collection provider

The default collection provider reads markdown files from `content/`. Any data source that can enumerate items and fetch one by slug can back a collection. This guide shows how to register a custom provider. See the [collections reference](../../reference/collections-api.md) for the full API.

## The provider interface

A provider satisfies the `collection.Provider` interface:

```go
type Provider interface {
    Name() string
    List(ctx context.Context, opts ListOptions) ([]Item, error)
    Get(ctx context.Context, slug string) (Item, error)
}
```

- `Name()` returns the unique provider identifier used in `p-provider="..."`.
- `List` enumerates items for the build-time route generation and for `GetAllCollectionItems`.
- `Get` fetches a single item by the URL parameter.

## Implement the provider

A SQL-backed provider might look like:

```go
package providers

import (
    "context"
    "database/sql"

    "piko.sh/piko/collection"
)

type ProductsProvider struct {
    db *sql.DB
}

func (p ProductsProvider) Name() string { return "products-sql" }

func (p ProductsProvider) List(ctx context.Context, opts collection.ListOptions) ([]collection.Item, error) {
    rows, err := p.db.QueryContext(ctx, `SELECT sku, name, price FROM products`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []collection.Item
    for rows.Next() {
        var sku, name string
        var price int
        if err := rows.Scan(&sku, &name, &price); err != nil {
            return nil, err
        }
        out = append(out, collection.Item{
            Slug: sku,
            Data: map[string]any{
                "SKU":   sku,
                "Name":  name,
                "Price": price,
            },
        })
    }
    return out, rows.Err()
}

func (p ProductsProvider) Get(ctx context.Context, slug string) (collection.Item, error) {
    row := p.db.QueryRowContext(ctx, `SELECT name, price FROM products WHERE sku = $1`, slug)
    var name string
    var price int
    if err := row.Scan(&name, &price); err != nil {
        return collection.Item{}, err
    }
    return collection.Item{
        Slug: slug,
        Data: map[string]any{"SKU": slug, "Name": name, "Price": price},
    }, nil
}
```

## Register the provider

Pass the provider into the server at bootstrap:

```go
ssr := piko.New(
    piko.WithCollectionProvider(providers.ProductsProvider{DB: db}),
)
```

## Use the provider from a template

Reference the provider by name in the page's `p-provider` attribute:

```piko
<template p-collection="products" p-provider="products-sql">
  <h1 p-text="state.Name"></h1>
  <p>Price: {{ state.Price }}</p>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Product struct {
    SKU   string
    Name  string
    Price int
}

type Response struct {
    Name  string
    Price int
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    p := piko.GetData[Product](r)
    return Response{Name: p.Name, Price: p.Price}, piko.Metadata{Title: p.Name}, nil
}
</script>
```

Piko generates one route per item at build time, calling `Get` on request.

## When to write a custom provider

- The data lives in a system that is not a markdown file (a database, an API, a headless CMS).
- The enumeration rules require computation beyond "list files".
- The content changes often enough that a rebuild per change is too expensive.

For static content in markdown, the default provider is sufficient.

## See also

- [Collections reference](../../reference/collections-api.md).
- [How to markdown collections](markdown.md).
- [How to querying and filtering](querying-and-filtering.md).
