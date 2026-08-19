---
title: How to set page titles and Open Graph tags
description: Return metadata from Render so pages display correctly in search results and social previews.
nav:
  sidebar:
    section: "how-to"
    subsection: "metadata-seo"
    order: 10
---

# How to set page titles and Open Graph tags

A page's `Render` function returns a `piko.Metadata` value that controls its title, description, canonical URL, and social-sharing tags. This guide covers the common fields. See the [metadata reference](../../reference/metadata-fields.md) for the full field list.

## Set the title and description

Return the values from `Render`:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{}, piko.Metadata{
        Title:       "About us | MyApp",
        Description: "Learn about our company and mission.",
    }, nil
}
```

Guidance:

- Keep titles under 60 characters.
- Keep descriptions between 150 and 160 characters.
- Make both unique per page.
- Front-load primary keywords.

## Set a canonical URL

A canonical URL tells search engines which URL is the authoritative one for a piece of content (useful when the same page is reachable from multiple paths):

```go
return Response{}, piko.Metadata{
    Title:        "Deploying Piko",
    CanonicalURL: "https://example.com/docs/deploying-piko",
}, nil
```

## Open Graph tags

Open Graph (OG) tags control how links render on social media. Append to `OGTags`:

```go
return Response{}, piko.Metadata{
    Title:       "Deploying Piko",
    Description: "A practical guide to shipping Piko to production.",
    OGTags: []piko.OGTag{
        {Property: "og:type",     Content: "article"},
        {Property: "og:url",      Content: "https://example.com/docs/deploying-piko"},
        {Property: "og:image",    Content: "https://example.com/og-images/deploying-piko.png"},
        {Property: "og:site_name", Content: "MyApp"},
    },
}, nil
```

Piko emits `<meta property="og:..." content="...">` tags in the document head.

## Twitter cards

Use the dedicated `TwitterCards` field - Piko emits each entry as a `<meta name="twitter:..." content="...">` tag in the document head. (Falling back to `MetaTags` works too, but `TwitterCards` is the typed surface for these tags.)

```go
TwitterCards: []piko.MetaTag{
    {Name: "twitter:card",    Content: "summary_large_image"},
    {Name: "twitter:site",    Content: "@MyApp"},
    {Name: "twitter:creator", Content: "@YourHandle"},
},
```

## Robots directives

Control indexing with `RobotsRule`:

```go
piko.Metadata{
    RobotsRule: "noindex,nofollow",
}
```

Common values: `"index,follow"`, `"noindex"`, `"noindex,nofollow"`, `"noarchive"`.

Piko renders the value as `<meta name="robots">` on the page and on the fragment a soft navigation swaps in. It also mirrors the value into the `X-Robots-Tag` response header, so a `HEAD` request and any non-HTML representation carry it too. A value containing `noindex` also drops the page from the sitemap.

To mark a page `noindex` without writing Go, add the `p-noindex` attribute to its `<template>`. That sets `RobotsRule` to `"noindex"` unless the page sets its own rule, so an explicit `RobotsRule` always wins.

To withhold a whole site instead of one page, see [keeping a site out of the index](sitemap-and-robots.md#keep-a-site-out-of-the-index). A `robots.txt` `Disallow` alone does not de-index. It stops the crawl, and a crawler that never fetches the page cannot read its `noindex`.

## Language

Set the document language:

```go
piko.Metadata{
    Language: "en-GB",
}
```

Piko applies this as `<html lang="en-GB">`. For multi-locale sites, see the [i18n basic setup guide](../i18n/basic-setup.md).

## Last modified date

`Metadata.LastModified` is a `*time.Time` field. The sitemap builder reads it to populate the `<lastmod>` element for the page in `sitemap.xml`. When the field is `nil`, the builder falls back to the source file's modification time, so most pages need not set it explicitly. Set it when the page's freshness differs from the file's mtime - for example, a content-driven page whose canonical timestamp lives in a database or front matter:

```go
publishedAt := article.UpdatedAt

return Response{}, piko.Metadata{
    LastModified: &publishedAt,
}, nil
```

The render path does not emit a `Last-Modified` HTTP header or a Schema.org `dateModified` element from the value. Apply those manually if you need them.

## Redirect

Return a redirect status with `ClientRedirect`:

```go
return Response{}, piko.Metadata{
    ClientRedirect: "/new-location",
    RedirectStatus: 301,
}, nil
```

`ServerRedirect` rewrites the response internally without changing the browser URL.

## See also

- [Metadata fields reference](../../reference/metadata-fields.md) for every available field.
- [How to structured data](structured-data.md).
- [How to i18n routing](../i18n/routing-strategy.md) for `AlternateLinks`.
