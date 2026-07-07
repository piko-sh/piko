---
title: How to generate a sitemap and robots.txt
description: Turn on SEO generation so Piko serves a correct sitemap.xml and robots.txt.
nav:
  sidebar:
    section: "how-to"
    subsection: "metadata-seo"
    order: 30
---

# How to generate a sitemap and robots.txt

Enable `piko.WithSEO` and Piko serves `/sitemap.xml` and `/robots.txt`, generated from your pages and configuration. This guide covers turning generation on and getting the two files correct for production. See the [SEO API reference](../../reference/seo-api.md) for the full configuration surface.

## Enable SEO generation

Pass `piko.WithSEO` with a sitemap hostname. Piko only activates SEO when you supply an enabled configuration and a non-empty `Sitemap.Hostname`:

```go
app, err := piko.New(
    piko.WithSEO(piko.SEOConfig{
        Sitemap: piko.SitemapConfig{
            Hostname: "https://example.com",
        },
    }),
)
```

`SEOConfig.Enabled` defaults to `true`, so supplying the option with a hostname is enough. Piko now serves `/sitemap.xml` (built from discovered pages) and `/robots.txt`.

## Match the sitemap hostname to the canonical base URL

The sitemap `<loc>` elements use `Sitemap.Hostname`, while page canonical and hreflang hrefs use the website `canonicalBaseUrl`. If the two origins differ, those URLs disagree, which becomes an indexing problem. Piko warns at startup when you configure both and they differ:

```
SEO: sitemap hostname and canonical base URL differ; sitemap <loc> and page canonical/hreflang will disagree
```

Set both to the same origin:

```go
app, err := piko.New(
    piko.WithWebsiteConfig(piko.WebsiteConfig{
        CanonicalBaseURL: "https://example.com",
    }),
    piko.WithSEO(piko.SEOConfig{
        Sitemap: piko.SitemapConfig{
            Hostname: "https://example.com",
        },
    }),
)
```

When you set only the sitemap hostname, Piko copies it to `canonicalBaseUrl`, so the simplest correct setup is to configure the hostname once and leave the canonical base URL unset.

## Keep staging out of the index

In a non-production build, `robots.txt` blocks every crawler by default with a site-wide `User-agent: *` / `Disallow: /`, so a staging deploy is not indexed. Piko logs a warning when this happens:

```
SEO: non-production build - robots.txt will block all crawlers (set robots.allowNonProductionIndexing to override)
```

Opt in to indexing a non-production build with `Robots.AllowNonProductionIndexing`:

```go
piko.WithSEO(piko.SEOConfig{
    Sitemap: piko.SitemapConfig{Hostname: "https://example.com"},
    Robots: piko.RobotsConfig{
        AllowNonProductionIndexing: true,
    },
})
```

Guidance:

- Production builds always serve the permissive `User-agent: *` / `Allow: /` base group.
- The build mode decides the block, not this flag alone. The flag only opts a non-production build back in.
- Piko fails open: when the build mode is not wired at all, it assumes production and keeps a live site indexed.

## Tune caching and contents

`Sitemap.CacheMaxAgeSeconds` is a `*int` that sets the `Cache-Control` max-age (in seconds) for the served `sitemap.xml`, its chunks, and `robots.txt`. A `nil` pointer keeps the default of 600 seconds. A value of `0` disables caching:

```go
zero := 0

piko.WithSEO(piko.SEOConfig{
    Sitemap: piko.SitemapConfig{
        Hostname:              "https://example.com",
        CacheMaxAgeSeconds:    &zero,  // disable caching
        IncludeAuthGatedPages: false,  // default: exclude login-gated pages
        GitLastMod:            true,   // derive <lastmod> from git, not file mtime
        DiscoverImages:        true,   // default: include discovered images
    },
})
```

Guidance:

- `IncludeAuthGatedPages` defaults to `false`, so the sitemap builder excludes pages that declare an `AuthPolicy`. Set it to `true` to include them.
- `GitLastMod` defaults to `false`; set it to `true` to derive a static page's `<lastmod>` from its last git commit date instead of the file modification time.
- `DiscoverImages` defaults to `true`, so the sitemap builder adds images it finds on pages. Set it to `false` to omit them.

> **Note.** Do not also mount a manual `GET /robots.txt` handler. Piko already serves `/robots.txt` and `/sitemap.xml` when you enable SEO, and a manual route would shadow the generated file. See [custom handlers](../routing/custom-handlers.md) for serving static files and mounting raw HTTP handlers.

## See also

- [SEO API reference](../../reference/seo-api.md) for every configuration field.
- [How to build a multilingual sitemap](multilingual-sitemap.md) for hreflang and localised URLs.
- [How to set page titles and Open Graph tags](title-and-og.md).
