---
title: Metadata & SEO
description: Optimise pages for search engines and social media with Piko metadata
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 130
---

# Metadata & SEO

Metadata controls how your pages appear in search engines, social media, and browser tabs. Piko makes SEO easy by letting you return metadata from your `Render()` function.

## Basic metadata

Return a `piko.Metadata` struct from your `Render()` function:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        // ... your response data
    }, piko.Metadata{
        Title:       "About Us | MyApp",
        Description: "Learn about our company and mission",
    }, nil
}
```

## Metadata fields reference

The `piko.Metadata` struct contains the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `Title` | `string` | Sets the `<title>` tag |
| `Description` | `string` | Sets `<meta name="description">` |
| `Keywords` | `string` | Sets `<meta name="keywords">` |
| `Language` | `string` | Sets the `lang` attribute on `<html>` |
| `CanonicalURL` | `string` | Sets `<link rel="canonical">` |
| `RobotsRule` | `string` | Sets `<meta name="robots">` |
| `Status` | `int` | HTTP status code (e.g., 200, 404, 500) |
| `StatusText` | `string` | Custom status text (e.g., "Not Found") |
| `ClientRedirect` | `string` | HTTP redirect URL (changes browser URL) |
| `ServerRedirect` | `string` | Internal URL rewrite (preserves browser URL) |
| `RedirectStatus` | `int` | HTTP status for redirects (301, 302, 303, 307) |
| `LastModified` | `*time.Time` | Last modification date for SEO |
| `CacheKey` | `string` | Custom cache key for server-side caching |
| `OGTags` | `[]struct{Property, Content string}` | Open Graph meta tags |
| `MetaTags` | `[]struct{Name, Content string}` | Generic meta tags |
| `AlternateLinks` | `[]map[string]string` | Alternate language links |

## Title

Sets the `<title>` tag:

```go
piko.Metadata{
    Title: "Contact Us | MyApp",
}
```

**Output**:
```html
<title>Contact Us | MyApp</title>
```

**Best practices**:
- Keep under 60 characters
- Include brand name
- Make unique per page
- Front-load important keywords

## Description

Sets the `<meta name="description">` tag:

```go
piko.Metadata{
    Description: "Get in touch with our team. We're here to help!",
}
```

**Output**:
```html
<meta name="description" content="Get in touch with our team. We're here to help!" />
```

**Best practices**:
- Keep 150-160 characters
- Include main keywords naturally
- Make compelling and accurate
- Unique per page

## Keywords

Sets the `<meta name="keywords">` tag:

```go
piko.Metadata{
    Keywords: "web framework, golang, SSR, server-side rendering",
}
```

> **Note**: Most search engines no longer use the keywords meta tag for ranking, but it doesn't hurt to include it.

## Language

Sets the `lang` attribute on `<html>`:

```go
piko.Metadata{
    Language: "en",
}
```

**Output**:
```html
<html lang="en">
```

## Canonical URL

Sets the canonical link to prevent duplicate content issues:

```go
piko.Metadata{
    CanonicalURL: "https://example.com/products/laptop",
}
```

**Output**:
```html
<link rel="canonical" href="https://example.com/products/laptop" />
```

**When to use**:
- Multiple URLs for same content
- URL parameters (e.g., `?sort=price`)
- Pagination
- Product variations

## Open Graph tags

Open Graph tags control how your pages appear when shared on social media (Facebook, LinkedIn, etc.).

### Basic open graph

```go
piko.Metadata{
    Title:       "Amazing Product | MyStore",
    Description: "Check out this amazing product!",
    OGTags: []struct{ Property, Content string }{
        {Property: "og:title", Content: "Amazing Product"},
        {Property: "og:description", Content: "Check out this amazing product!"},
        {Property: "og:type", Content: "website"},
        {Property: "og:url", Content: "https://example.com/products/amazing"},
        {Property: "og:image", Content: "https://example.com/images/product.jpg"},
    },
}
```

**Output**:
```html
<meta property="og:title" content="Amazing Product" />
<meta property="og:description" content="Check out this amazing product!" />
<meta property="og:type" content="website" />
<meta property="og:url" content="https://example.com/products/amazing" />
<meta property="og:image" content="https://example.com/images/product.jpg" />
```

### Complete product example

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    product := fetchProduct()

    return Response{
        Product: product,
    }, piko.Metadata{
        Title:       product.Name + " | MyStore",
        Description: product.Description,
        OGTags: []struct{ Property, Content string }{
            {Property: "og:title", Content: product.Name},
            {Property: "og:description", Content: product.Description},
            {Property: "og:type", Content: "product"},
            {Property: "og:url", Content: "https://example.com" + r.URL().Path},
            {Property: "og:image", Content: product.ImageURL},
            {Property: "og:image:width", Content: "1200"},
            {Property: "og:image:height", Content: "630"},
            {Property: "og:image:alt", Content: product.Name},
            {Property: "og:site_name", Content: "MyStore"},
            {Property: "og:locale", Content: "en_GB"},
            {Property: "product:price:amount", Content: fmt.Sprintf("%.2f", product.Price)},
            {Property: "product:price:currency", Content: "GBP"},
        },
    }, nil
}
```

### Twitter cards

For Twitter-specific metadata, use the `MetaTags` field:

```go
piko.Metadata{
    MetaTags: []struct{ Name, Content string }{
        {Name: "twitter:card", Content: "summary_large_image"},
        {Name: "twitter:site", Content: "@mystore"},
        {Name: "twitter:creator", Content: "@username"},
        {Name: "twitter:title", Content: "Amazing Product"},
        {Name: "twitter:description", Content: "Check out this amazing product!"},
        {Name: "twitter:image", Content: "https://example.com/images/product.jpg"},
    },
}
```

**Card types**:
- `summary` - Small card with thumbnail
- `summary_large_image` - Large card with featured image
- `app` - Mobile app installation
- `player` - Video/audio player

## Robots control

### Robots meta tag

Control how search engines index your page:

```go
piko.Metadata{
    RobotsRule: "index, follow",
}
```

**Output**:
```html
<meta name="robots" content="index, follow" />
```

**Common values**:
- `index, follow` - Index page and follow links (default)
- `noindex, follow` - Don't index page, but follow links
- `index, nofollow` - Index page, don't follow links
- `noindex, nofollow` - Don't index, don't follow
- `noarchive` - Don't show cached version
- `nosnippet` - Don't show description snippet

### Conditional robots

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    itemID := r.PathParam("id")
    item := fetchItem(itemID)

    metadata := piko.Metadata{
        Title: item.Title,
    }

    // Don't index draft or unpublished items
    if item.Status != "published" {
        metadata.RobotsRule = "noindex, nofollow"
    }

    return Response{Item: item}, metadata, nil
}
```

## HTTP status codes

### 404 not found

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    itemID := r.PathParam("id")
    item, err := fetchItem(itemID)
    if errors.Is(err, ErrNotFound) {
        return Response{}, piko.Metadata{
            Status:     404,
            StatusText: "Not Found",
            Title:      "Page Not Found",
        }, nil
    }

    return Response{Item: item}, piko.Metadata{
        Title: item.Name,
    }, nil
}
```

### Custom status codes

```go
piko.Metadata{
    Status:     503,
    StatusText: "Service Unavailable",
    Title:      "Maintenance Mode",
}
```

## Redirects

Piko supports two types of redirects:

### Client redirect (HTTP redirect)

Changes the browser URL. The server returns an HTTP redirect response with a `Location` header.

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    oldID := r.PathParam("id")
    newID := getNewID(oldID)

    return Response{}, piko.Metadata{
        ClientRedirect: fmt.Sprintf("/products/%s", newID),
        RedirectStatus: 301, // Permanent redirect (default: 302)
    }, nil
}
```

**Valid `RedirectStatus` values**:
- `301` - Permanent redirect (SEO-friendly for moved content)
- `302` - Temporary redirect (default if not specified)
- `303` - See Other (after POST)
- `307` - Temporary redirect (preserves HTTP method)

### Server redirect (internal rewrite)

Internally renders a different page without changing the browser URL. Useful for showing login pages, error pages, or A/B testing.

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    if !isAuthenticated(r) {
        // Show login page, but keep the URL as-is
        return Response{}, piko.Metadata{
            ServerRedirect: "/login",
        }, nil
    }

    return Response{}, piko.Metadata{}, nil
}
```

> **Note**: Server redirects have a maximum of 3 hops to prevent infinite loops. If both `ServerRedirect` and `ClientRedirect` are set, `ServerRedirect` takes precedence.

### Maintenance mode example

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    if isMaintenanceMode() {
        return Response{}, piko.Metadata{
            ClientRedirect: "/maintenance",
            RedirectStatus: 302, // Temporary
        }, nil
    }

    return Response{}, piko.Metadata{}, nil
}
```

## Alternate links (i18n)

For multilingual sites, specify alternate language versions:

```go
piko.Metadata{
    AlternateLinks: []map[string]string{
        {
            "rel":      "alternate",
            "hreflang": "en",
            "href":     "https://example.com/products/laptop",
        },
        {
            "rel":      "alternate",
            "hreflang": "es",
            "href":     "https://example.com/es/products/laptop",
        },
        {
            "rel":      "alternate",
            "hreflang": "fr",
            "href":     "https://example.com/fr/products/laptop",
        },
        {
            "rel":      "alternate",
            "hreflang": "x-default",
            "href":     "https://example.com/products/laptop",
        },
    },
}
```

**Output**:
```html
<link rel="alternate" hreflang="en" href="https://example.com/products/laptop" />
<link rel="alternate" hreflang="es" href="https://example.com/es/products/laptop" />
<link rel="alternate" hreflang="fr" href="https://example.com/fr/products/laptop" />
<link rel="alternate" hreflang="x-default" href="https://example.com/products/laptop" />
```

### Using GenerateLocaleHead helper

For i18n sites, use the `piko.GenerateLocaleHead()` helper to automatically generate alternate links:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    i18nConfig := piko.I18nConfig{
        DefaultLocale: "en",
        Strategy:      "prefix",
        Locales:       []string{"en", "fr", "de"},
    }

    locale, canonical, alternates := piko.GenerateLocaleHead(r, i18nConfig, "/about", nil)

    return Response{}, piko.Metadata{
        Language:       locale,
        CanonicalURL:   canonical,
        AlternateLinks: alternates,
    }, nil
}
```

See [i18n](/docs/guide/i18n) for full internationalisation setup including routing strategies and locale configuration.

## Additional meta tags

Use `MetaTags` for any other meta tags:

```go
piko.Metadata{
    MetaTags: []struct{ Name, Content string }{
        {Name: "author", Content: "John Doe"},
        {Name: "viewport", Content: "width=device-width, initial-scale=1"},
        {Name: "theme-color", Content: "#6F47EB"},
        {Name: "apple-mobile-web-app-capable", Content: "yes"},
    },
}
```

## Complete examples

### Blog post

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    slug := r.PathParam("slug")
    post := piko.GetData[BlogPost](r)

    // Full canonical URL
    canonicalURL := fmt.Sprintf("https://myblog.com/blog/%s", post.Slug)

    // Open Graph image with fallback
    ogImage := post.FeaturedImage
    if ogImage == "" {
        ogImage = "https://myblog.com/images/default-og.jpg"
    }

    return Response{
        Post: post,
    }, piko.Metadata{
        Title:        fmt.Sprintf("%s | My Blog", post.Title),
        Description:  post.Excerpt,
        CanonicalURL: canonicalURL,
        Keywords:     strings.Join(post.Tags, ", "),

        OGTags: []struct{ Property, Content string }{
            {Property: "og:type", Content: "article"},
            {Property: "og:title", Content: post.Title},
            {Property: "og:description", Content: post.Excerpt},
            {Property: "og:url", Content: canonicalURL},
            {Property: "og:image", Content: ogImage},
            {Property: "og:site_name", Content: "My Blog"},
            {Property: "article:published_time", Content: post.PublishedAt.Format(time.RFC3339)},
            {Property: "article:author", Content: post.Author.Name},
            {Property: "article:section", Content: post.Category},
        },

        MetaTags: []struct{ Name, Content string }{
            {Name: "twitter:card", Content: "summary_large_image"},
            {Name: "twitter:title", Content: post.Title},
            {Name: "twitter:description", Content: post.Excerpt},
            {Name: "twitter:image", Content: ogImage},
            {Name: "author", Content: post.Author.Name},
        },

        LastModified: &post.UpdatedAt,
    }, nil
}
```

### E-commerce product page

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    productID := r.PathParam("id")
    product := fetchProduct(productID)

    availability := "in stock"
    if product.Stock == 0 {
        availability = "out of stock"
    }

    return Response{
        Product: product,
    }, piko.Metadata{
        Title:        fmt.Sprintf("%s | MyStore", product.Name),
        Description:  product.ShortDescription,
        CanonicalURL: fmt.Sprintf("https://mystore.com/products/%s", product.Slug),

        OGTags: []struct{ Property, Content string }{
            {Property: "og:type", Content: "product"},
            {Property: "og:title", Content: product.Name},
            {Property: "og:description", Content: product.ShortDescription},
            {Property: "og:image", Content: product.Images[0].URL},
            {Property: "og:url", Content: fmt.Sprintf("https://mystore.com/products/%s", product.Slug)},
            {Property: "product:price:amount", Content: fmt.Sprintf("%.2f", product.Price)},
            {Property: "product:price:currency", Content: "GBP"},
            {Property: "product:availability", Content: availability},
            {Property: "product:condition", Content: "new"},
        },

        MetaTags: []struct{ Name, Content string }{
            {Name: "twitter:card", Content: "product"},
            {Name: "twitter:label1", Content: "Price"},
            {Name: "twitter:data1", Content: fmt.Sprintf("£%.2f", product.Price)},
            {Name: "twitter:label2", Content: "Availability"},
            {Name: "twitter:data2", Content: availability},
        },
    }, nil
}
```

## Testing

### Preview in browser

Check `<head>` tags:
1. View page source
2. Look for `<title>`, `<meta>`, `<link>` tags

### Test social sharing

**Facebook**:
- https://developers.facebook.com/tools/debug/

**Twitter**:
- https://cards-dev.twitter.com/validator

**LinkedIn**:
- https://www.linkedin.com/post-inspector/

### Google rich results

- https://search.google.com/test/rich-results

## Next steps

- [Collections](/docs/guide/collections) → SEO for collection pages
