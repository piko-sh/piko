---
title: How to add structured data (Schema.org)
description: Embed JSON-LD blocks for articles, products, events, and organisations.
nav:
  sidebar:
    section: "how-to"
    subsection: "metadata-seo"
    order: 20
---

# How to add structured data (Schema.org)

Structured data describes a page's content to search engines in a machine-readable format. Piko has a typed first-class field for this: `Metadata.StructuredData []string`. Each entry renders as a separate `<script type="application/ld+json">` block in the document head. The renderer silently drops invalid JSON entries with a warning log, so malformed payloads cannot break the rendered page. See the [metadata reference](../../reference/metadata-fields.md) for the surrounding metadata fields.

## Use the typed metadata field

Marshal each Schema.org payload to JSON, then assign the strings to `Metadata.StructuredData`:

```go
import (
    "encoding/json"
    "piko.sh/piko"
)

type Response struct {
    Title string
    Body  string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    article, _ := json.Marshal(map[string]any{
        "@context":      "https://schema.org",
        "@type":         "Article",
        "headline":      "Deploying Piko to production",
        "author":        map[string]string{"@type": "Person", "name": "Alice Smith"},
        "datePublished": "2026-01-15",
        "dateModified":  "2026-01-20",
        "publisher":     map[string]any{"@type": "Organization", "name": "MyApp"},
        "image":         "https://example.com/articles/deploying-piko.jpg",
        "description":   "A practical guide to shipping Piko to production.",
    })

    return Response{
            Title: "Deploying Piko to production",
            Body:  "...",
        }, piko.Metadata{
            Title:          "Deploying Piko | MyApp Blog",
            Description:    "A practical guide to shipping Piko to production.",
            StructuredData: []string{string(article)},
        }, nil
}
```

Piko emits each entry in `StructuredData` as its own `<script type="application/ld+json">` block, in order. Multiple entries are common - for example an `Article` payload alongside a `BreadcrumbList`.

## Product schema

For e-commerce pages:

```go
product, _ := json.Marshal(map[string]any{
    "@context":    "https://schema.org",
    "@type":       "Product",
    "name":        product.Name,
    "description": product.Description,
    "sku":         product.SKU,
    "offers": map[string]any{
        "@type":         "Offer",
        "price":         product.Price,
        "priceCurrency": "GBP",
        "availability":  "https://schema.org/InStock",
    },
    "aggregateRating": map[string]any{
        "@type":       "AggregateRating",
        "ratingValue": product.Rating,
        "reviewCount": product.ReviewCount,
    },
})

return Response{}, piko.Metadata{
    StructuredData: []string{string(product)},
}, nil
```

## Breadcrumb schema

For navigation trails:

```go
breadcrumb, _ := json.Marshal(map[string]any{
    "@context": "https://schema.org",
    "@type":    "BreadcrumbList",
    "itemListElement": []map[string]any{
        {"@type": "ListItem", "position": 1, "name": "Home", "item": "https://example.com/"},
        {"@type": "ListItem", "position": 2, "name": "Blog", "item": "https://example.com/blog"},
        {"@type": "ListItem", "position": 3, "name": "Deploying Piko"},
    },
})
```

Combine with the article payload by appending both strings:

```go
piko.Metadata{
    StructuredData: []string{string(article), string(breadcrumb)},
}
```

## Organisation schema

Add once in a layout partial so every page emits it:

```go
organisation, _ := json.Marshal(map[string]any{
    "@context": "https://schema.org",
    "@type":    "Organization",
    "name":     "MyApp Limited",
    "url":      "https://example.com",
    "logo":     "https://example.com/logo.png",
    "sameAs": []string{
        "https://twitter.com/MyApp",
        "https://github.com/MyApp",
    },
})
```

## Fallback: rendering inside a template

Sometimes a page genuinely needs to author the JSON-LD block in the template, weaving in dynamic state alongside other markup. In that case, `p-html` can inject the payload as raw HTML without escaping it:

```piko
<template>
  <script type="application/ld+json" p-html="state.SchemaJSON"></script>
</template>
```

`Metadata.StructuredData` is the preferred path. The template approach skips Piko's JSON validation and ordering guarantees, so reach for it only when the metadata field cannot express the requirement.

## Validate before shipping

Test JSON-LD with:

- Google's [Rich Results Test](https://search.google.com/test/rich-results).
- [Schema.org validator](https://validator.schema.org).

Missing or malformed properties do not surface in search results even when the page renders correctly.

## See also

- [Metadata fields reference](../../reference/metadata-fields.md).
- [How to title and OG tags](title-and-og.md).
- [Schema.org documentation](https://schema.org/docs/schemas.html) for every available type.
