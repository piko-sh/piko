---
title: Advanced collections
description: Filter, sort, paginate, build navigation trees, and configure hybrid caching for collections.
nav:
  sidebar:
    section: "how-to"
    subsection: "collections"
    order: 720
---

# Advanced collections

This guide covers tasks beyond `piko.GetData[T](r)`: filtering items, sorting them, paginating, building navigation, and tuning caching. For the full type and constructor surface see [collections-api](../../reference/collections-api.md).

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/collection"
)
```

## Filter items by a metadata field

To filter posts by an exact field value, build a `Filter` and apply it with `ApplyFilterGroup`:

```go
filter := collection.Filter{
    Field:    "featured",
    Operator: collection.FilterOpEquals,
    Value:    true,
}
```

Other common operators: `FilterOpGreaterThan` for date ranges, `FilterOpIn` for multi-value matches, `FilterOpFuzzyMatch` for approximate text, `FilterOpExists` for presence. The full list of operators is in [collections-api](../../reference/collections-api.md#filter-api).

## Combine filters with `AND` or `OR`

Wrap two or more `Filter` values in a `FilterGroup` and set `Logic` to `"AND"` or `"OR"`:

```go
filterGroup := &collection.FilterGroup{
    Logic: "AND",
    Filters: []collection.Filter{
        {Field: "published", Operator: collection.FilterOpEquals, Value: true},
        {Field: "category", Operator: collection.FilterOpEquals, Value: "tutorial"},
    },
}

for _, item := range items {
    if collection.ApplyFilterGroup(item, filterGroup) {
        // item matches every filter in the group
    }
}
```

`ApplyFilterGroup(item *ContentItem, group *FilterGroup) bool` takes both arguments by pointer (`items` is a `[]*ContentItem`). `Logic` is a plain string set to `"AND"` or `"OR"`.

## Sort items by one or more fields

To sort newest first, pass a `SortOption` slice to `SortItems`:

```go
sortOptions := []collection.SortOption{
    {Field: "publishedAt", Order: collection.SortDesc},
}

collection.SortItems(items, sortOptions)
```

Multiple options apply in sequence (primary, secondary). `SortRandom` shuffles. Sorting handles strings, numbers, dates (RFC3339, `YYYY-MM-DD`, and `YYYY-MM-DD HH:MM:SS`), and booleans without explicit type hints.

## Paginate a result set

To produce a page slice, build `PaginationOptions` and call `PaginateItems`:

```go
pagination := &collection.PaginationOptions{
    Page:     2,
    PageSize: 10,
}

paginatedItems := collection.PaginateItems(items, pagination)
```

`Offset` and `Limit` work as an alternative to `Page`/`PageSize`. To compute totals for UI controls, call `CalculatePaginationMeta(totalItems, pagination)`.

## Build a sidebar navigation tree

Collections expose their structure as `NavigationGroups` and `NavigationTree` values on `piko`. To drive a sidebar, declare a section in each item's frontmatter:

```yaml
---
title: Installation guide
nav:
  sidebar:
    section: "getting-started"
    subsection: "basics"
    order: 1
    label: "Install"
    icon: "download"
---
```

Then walk the tree returned by the runtime, using `node.IsCategory()` for grouping nodes, `node.IsLeaf()` for items, and `node.GetBreadcrumb()` for the path to a node. See [collections-api](../../reference/collections-api.md#navigation-types) for the full method set.

## Render a table of contents from headings

To list headings as a flat slice:

```go
sections := piko.GetSections(r)
```

For a nested tree limited to `h2` and `h3`:

```go
tree := piko.GetSectionsTree(r,
    piko.WithMinLevel(2),
    piko.WithMaxLevel(3),
)
```

Render the tree in a template:

```piko
<nav class="toc">
  <ul>
    <li p-for="(_, section) in state.TableOfContents">
      <a :href="'#' + section.Slug">{{ section.Title }}</a>
      <ul p-if="len(section.Children) > 0">
        <li p-for="(_, child) in section.Children">
          <a :href="'#' + child.Slug">{{ child.Title }}</a>
        </li>
      </ul>
    </li>
  </ul>
</nav>
```

## Cache fetch results at request time

`FetchOptions.Cache` is a request-time `*CacheConfig`. It picks one of four strategies (`"cache-first"`, `"network-first"`, `"stale-while-revalidate"`, `"no-cache"`) and a TTL in seconds:

```go
options := collection.FetchOptions{
    Cache: &collection.CacheConfig{
        Strategy: "stale-while-revalidate",
        TTL:      300,
    },
}
```

`Tags` and `Key` on `CacheConfig` let you override the auto-generated cache key or attach invalidation tags.

## Cache a collection with Incremental Static Regeneration

You configure ETag-driven hybrid caching at build time, not on `FetchOptions`. The `HybridConfig` struct lives on `DynamicCollectionInfo.HybridConfig`, and the provider populates it when it returns its build-time blueprint. `collection.DefaultHybridConfig()` returns sensible defaults: `RevalidationTTL` of 60 seconds, `MaxStaleAge` of zero (no maximum), `StaleIfError` true, and an `ETagSource` of `"modtime-hash"`. Use `"content-hash"` when modtime is unreliable, or `"provider-etag"` for sources that already publish HTTP ETag headers. `RevalidationTTL` and `MaxStaleAge` are `time.Duration` values.

A custom `pikoruntime.Provider` opts into hybrid mode through the build-time blueprint API. Runtime callers receive the cached snapshot and let the framework revalidate in the background. See [custom providers](custom-providers.md) and the [collections reference](../../reference/collections-api.md) for the wiring.

## Fetch only the locales you need

Set one of `Locale`, `ExplicitLocales`, or `AllLocales` on `FetchOptions`. They are mutually exclusive. Use `ExplicitLocales` to fetch a subset (for example a language-switcher menu), and `AllLocales` only when one pass requires every locale.

To trim the payload, attach a `FieldProjection`:

```go
options := collection.FetchOptions{
    Projection: &collection.FieldProjection{
        IncludeFields: []string{"title", "publishedAt", "slug"},
    },
}
```

`MaxArrayItems` caps array fields. Useful when items have long `tags` arrays you do not need on a list page.

## Read standard frontmatter fields

Every item exposes `ID`, `Slug`, `URL`, `Locale`, `TranslationKey`, `CreatedAt`, `UpdatedAt`, `PublishedAt`, and `ReadingTime`. For custom frontmatter, use the typed helpers:

```go
title := item.GetMetadataString("title", "Untitled")
viewCount := item.GetMetadataInt("views", 0)
isDraft := item.GetMetadataBool("draft", false)
tags := item.GetMetadataStringSlice("tags", nil)

if item.IsPublished() {
    // PublishedAt is set
}
```

## See also

- [Collections API reference](../../reference/collections-api.md) for every type, operator, constructor, and helper.
- [About collections](../../explanation/about-collections.md) for the design rationale behind the query surface.
- [Markdown collections how-to](markdown.md) for declarative collections and `piko:content`.
- [Custom providers how-to](custom-providers.md) for non-markdown sources.
