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

Structured data describes a page's content to search engines in a machine-readable format. Piko pages embed JSON-LD by rendering a `<script type="application/ld+json">` block in the template. This guide covers the common shapes. See the [metadata reference](../../reference/metadata-fields.md) for the surrounding metadata fields.

## Article schema

For blog posts and news articles:

```piko
<template>
  <article>
    <h1 p-text="state.Title"></h1>
    <p p-text="state.Body"></p>
  </article>

  <script type="application/ld+json" p-html="state.SchemaJSON"></script>
</template>

<script type="application/x-go">
package main

import (
    "encoding/json"
    "piko.sh/piko"
)

type Response struct {
    Title      string
    Body       string
    SchemaJSON string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    schema := map[string]any{
        "@context":         "https://schema.org",
        "@type":            "Article",
        "headline":         "Deploying Piko to production",
        "author":           map[string]string{"@type": "Person", "name": "Alice Smith"},
        "datePublished":    "2026-01-15",
        "dateModified":     "2026-01-20",
        "publisher":        map[string]any{"@type": "Organization", "name": "MyApp"},
        "image":            "https://example.com/articles/deploying-piko.jpg",
        "description":      "A practical guide to shipping Piko to production.",
    }
    raw, _ := json.Marshal(schema)

    return Response{
        Title:      "Deploying Piko to production",
        Body:       "...",
        SchemaJSON: string(raw),
    }, piko.Metadata{
        Title:       "Deploying Piko | MyApp Blog",
        Description: "A practical guide to shipping Piko to production.",
    }, nil
}
</script>
```

`p-html` injects the JSON string without HTML-escaping it, so the output is valid JSON-LD.

## Product schema

For e-commerce pages:

```go
schema := map[string]any{
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
}
```

## Breadcrumb schema

For navigation trails:

```go
schema := map[string]any{
    "@context": "https://schema.org",
    "@type":    "BreadcrumbList",
    "itemListElement": []map[string]any{
        {"@type": "ListItem", "position": 1, "name": "Home", "item": "https://example.com/"},
        {"@type": "ListItem", "position": 2, "name": "Blog", "item": "https://example.com/blog"},
        {"@type": "ListItem", "position": 3, "name": "Deploying Piko"},
    },
}
```

## Organisation schema

Add once in a layout partial so every page emits it:

```go
schema := map[string]any{
    "@context": "https://schema.org",
    "@type":    "Organization",
    "name":     "MyApp Limited",
    "url":      "https://example.com",
    "logo":     "https://example.com/logo.png",
    "sameAs": []string{
        "https://twitter.com/MyApp",
        "https://github.com/MyApp",
    },
}
```

## Validate before shipping

Test JSON-LD with:

- Google's [Rich Results Test](https://search.google.com/test/rich-results).
- [Schema.org validator](https://validator.schema.org).

Missing or malformed properties do not surface in search results even when the page renders correctly.

## See also

- [Metadata fields reference](../../reference/metadata-fields.md).
- [How to title and OG tags](title-and-og.md).
- [Schema.org documentation](https://schema.org/docs/schemas.html) for every available type.
