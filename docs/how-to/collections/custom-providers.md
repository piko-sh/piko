---
title: How to back a collection with a custom data source
description: Implement a runtime provider so a collection can read live data from a CMS, database, or HTTP API instead of markdown files.
nav:
  sidebar:
    section: "how-to"
    subsection: "collections"
    order: 30
---

# How to back a collection with a custom data source

Markdown files in `content/` are the default collection source, but any data source can back a collection. A database, a headless CMS, or a JSON API all work. This guide shows how to implement a runtime provider that satisfies `pikoruntime.Provider` and register it at startup so `r.GetCollection[T]("name")` calls fetch from your source.

> **Note:** Provider categories. Piko classifies providers as `static` (build-time only - markdown lives here), `dynamic` (runtime only - what this guide covers), or `hybrid` (build-time snapshot plus runtime revalidation, the Incremental Static Regeneration (ISR) pattern). Static providers are an internal extension point. Dynamic and hybrid providers are the supported user-facing extension surface, registered through `pikoruntime.RegisterRuntimeProvider`.

## Implement a runtime provider

A runtime provider satisfies `pikoruntime.Provider`:

```go
type Provider interface {
    Name() string
    Fetch(ctx context.Context, collectionName string, options *FetchOptions, target any) error
}
```

- `Name()` returns the identifier referenced by `<template p-collection="X" p-provider="THIS">` in `.pk` files.
- `Fetch` populates `target` (a pointer to a slice of the user's struct, for example `*[]Product`) with collection data. Inspect `options.Locale`, `options.Filters`, `options.Sort`, and `options.Pagination` if your source supports them.

A SQL-backed provider:

```go
package providers

import (
    "context"
    "database/sql"
    "fmt"

    pikoruntime "piko.sh/piko/wdk/runtime"
)

type Product struct {
    SKU   string
    Name  string
    Price int
}

type ProductsProvider struct {
    db *sql.DB
}

func NewProductsProvider(db *sql.DB) *ProductsProvider {
    return &ProductsProvider{db: db}
}

func (p *ProductsProvider) Name() string { return "products-sql" }

func (p *ProductsProvider) Fetch(
    ctx context.Context,
    collectionName string,
    _ *pikoruntime.FetchOptions,
    target any,
) error {
    out, ok := target.(*[]Product)
    if !ok {
        return fmt.Errorf("products-sql: target must be *[]Product, got %T", target)
    }

    rows, err := p.db.QueryContext(ctx,
        `SELECT sku, name, price FROM products`)
    if err != nil {
        return fmt.Errorf("products-sql: query: %w", err)
    }
    defer rows.Close()

    var items []Product
    for rows.Next() {
        var item Product
        if err := rows.Scan(&item.SKU, &item.Name, &item.Price); err != nil {
            return fmt.Errorf("products-sql: scan: %w", err)
        }
        items = append(items, item)
    }
    if err := rows.Err(); err != nil {
        return fmt.Errorf("products-sql: rows: %w", err)
    }

    *out = items
    return nil
}
```

The `target` parameter arrives typed: the generator emits `var items []Product; provider.Fetch(ctx, name, opts, &items)`. A type assertion to `*[]Product` is the simplest way to populate it. For providers that should work with any user struct, use reflection (`reflect.ValueOf(target).Elem()`) instead.

## Register at startup

Runtime providers register inside `cmd/main/main.go`, before `Run`:

```go
package main

import (
    "database/sql"
    "os"

    "piko.sh/piko"
    pikoruntime "piko.sh/piko/wdk/runtime"

    "myapp/providers"
)

func main() {
    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        panic(err)
    }

    if err := pikoruntime.RegisterRuntimeProvider(
        providers.NewProductsProvider(db),
    ); err != nil {
        panic(err)
    }

    ssr := piko.New()
    if err := ssr.Run(piko.RunModeProd); err != nil {
        panic(err)
    }
}
```

`RegisterRuntimeProvider` errors when two providers share a name. Register once at startup. Do not call it per request.

## Use the provider from a page

Reference the provider by name in the `p-provider` attribute on a `<template p-collection>` element. The generator wires the build-time `r.GetCollection[T]("collection-name")` lookup to call your provider's `Fetch` at request time.

```piko
<template p-collection="products" p-provider="products-sql">
  <article>
    <h1 p-text="state.Name"></h1>
    <p>Price: {{ state.Price }}</p>
  </article>
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

For listings (or anywhere you need the whole collection instead of a single item), call the typed accessor on `RequestData`:

```piko
<article p-for='(_, post) in r.GetCollection[Product]("products")'>
  <h2 p-text="post.Name"></h2>
</article>
```

The struct used at the call site (`Product` here) must match the struct your provider populates. Field names are case-sensitive.

## Pick a provider category

| Category | When | Registration | Cost |
|---|---|---|---|
| `static` (build-time) | Content rarely changes; rebuild per change is cheap | Internal - not a public extension point | Zero runtime overhead |
| `dynamic` (runtime) | Live data from an API or database | `pikoruntime.RegisterRuntimeProvider` | Per-request fetch + cache |
| `hybrid` (Incremental Static Regeneration) | Snapshot at build, refresh in background | `pikoruntime.RegisterRuntimeProvider` plus build-time hybrid metadata | Fast first byte + eventual freshness |

For markdown files, `piko.WithMarkdownParser(...)` registers the built-in static provider automatically. You need no custom code.

## See also

- [Collections reference](../../reference/collections-api.md) for the full type surface.
- [About collections](../../explanation/about-collections.md) for the provider model and design rationale.
- [How to markdown collections](markdown.md) for the default static path.
- [How to querying and filtering](querying-and-filtering.md) for using `FetchOptions`.
- Source: [`wdk/runtime/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/runtime/facade.go) for `Provider` and `RegisterRuntimeProvider`.
