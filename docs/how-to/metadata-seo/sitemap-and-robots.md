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

Pass `piko.WithSEO` with a sitemap hostname. Piko activates SEO when you supply the option with a non-empty `Sitemap.Hostname`:

```go
app, err := piko.New(
    piko.WithSEO(piko.SEOConfig{
        Sitemap: piko.SitemapConfig{
            Hostname: "https://example.com",
        },
    }),
)
```

Supplying the option is itself the signal that you want SEO, so a hostname is all you need. Piko now serves `/sitemap.xml` (built from discovered pages) and `/robots.txt`. To turn SEO off, omit `piko.WithSEO` entirely instead of setting `Enabled: false`.

Without `piko.WithSEO`, `/robots.txt` returns 404. That is not a quiet default. Crawlers treat a missing `robots.txt` as permission to crawl everything, so Piko logs the outcome at startup.

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

## Keep a site out of the index

By default `robots.txt` is permissive: a site-wide `User-agent: *` / `Allow: /`. Two separate settings withhold a site from search, and they answer different questions.

### An internal app: `NeverIndex`

Use `Robots.NeverIndex` for software that must never appear in a search index anywhere, such as an admin console, a staff back office or an internal tool. It is a property of the project, so it lives in source and travels with every build and every deploy of that build.

```go
piko.WithSEO(piko.SEOConfig{
    Sitemap: piko.SitemapConfig{Hostname: "https://admin.example.com"},
    Robots: piko.RobotsConfig{
        NeverIndex: true,
    },
})
```

Generation bakes a site-wide `Disallow: /` into `robots.txt`, every serving process independently serves the same block, and responses carry `X-Robots-Tag: noindex, nofollow`. No deploy setting, environment variable or run mode loosens it. The one exception is your own `CustomRules`: a group granting a named user agent `Allow: /` still applies to that agent, because a crawler obeys the most specific group that matches it. Piko logs the block so it is visible in the build output:

```
SEO: robots.neverIndex is set - robots.txt will block all crawlers
```

### A staging or preview deploy: `PreviewDeployment`

Use `Robots.PreviewDeployment` for a copy of a site that is not the live one. It is a property of the *deploy*, not the build. Piko reads it when serving `robots.txt` and never writes it into a stored artefact. You can therefore promote one build to staging and to production and get a different answer in each.

Piko reads no environment variables of its own, so your application decides:

```go
piko.WithSEO(piko.SEOConfig{
    Sitemap: piko.SitemapConfig{Hostname: "https://example.com"},
    Robots: piko.RobotsConfig{
        PreviewDeployment: !strings.EqualFold(os.Getenv("PIKO_ENV"), "production"),
    },
})
```

Set that variable on the running container, not in the build stage. A value that leaks into a build cannot bake a block. That is deliberate, because a baked block is one no later deploy can lift.

### Why this is not inferred

The build writes `robots.txt`. A generate mode (`generator all`) says what to build, not where the result lands, so the generator cannot tell a staging build from a production one. Earlier versions inferred "not production" from the generate mode and shipped `Disallow: /` to live sites as a result.

Guidance:

- The default is permissive, and Piko logs the resolved posture of every build instead of deciding quietly.
- `NeverIndex` is for the project; `PreviewDeployment` is for the deploy. Neither implies the other.
- A `Disallow` stops a crawl but does not remove a URL already in an index, and a blocked crawler can never read a `noindex` in the body. Both settings therefore also send `X-Robots-Tag`, which is what actually keeps a site out of an index.
- A `dev` or `dev-i` run always writes a blocking `robots.txt` into its local registry, so a development rebuild cannot leave a permissive file behind. This never touches a production build.

## Tune caching and contents

`Sitemap.CacheMaxAgeSeconds` is a `*int` that sets the `Cache-Control` max-age (in seconds) for the served `sitemap.xml`, its chunks, and `robots.txt`. A `nil` pointer keeps the default of 600 seconds. A value of `0` disables caching:

```go
piko.WithSEO(piko.SEOConfig{
    Sitemap: piko.SitemapConfig{
        Hostname:              "https://example.com",
        CacheMaxAgeSeconds:    new(0),     // disable caching
        IncludeAuthGatedPages: false,      // default excludes login-gated pages
        GitLastMod:            true,       // derive <lastmod> from git, not file mtime
        DiscoverImages:        new(false), // default is true, which includes images
    },
})
```

Guidance:

- `IncludeAuthGatedPages` defaults to `false`, so the sitemap builder excludes pages that declare an `AuthPolicy`. Set it to `true` to include them.
- `GitLastMod` defaults to `false`; set it to `true` to derive a static page's `<lastmod>` from its last git commit date instead of the file modification time.
- `DiscoverImages` is a `*bool`. Leave it `nil` for the default of `true`, where the sitemap builder adds images it finds on pages, or pass `new(false)` to omit them.

> **Note.** Do not mount your own `GET /robots.txt` handler. Piko registers `/robots.txt` and `/sitemap.xml` as exact routes on the main router and mounts the application router beneath them, so your handler never runs and Piko serves the generated file. Configure `Robots` instead of writing the file yourself. See [custom handlers](../routing/custom-handlers.md) for serving other static files and mounting raw HTTP handlers.

## See also

- [SEO API reference](../../reference/seo-api.md) for every configuration field.
- [How to build a multilingual sitemap](multilingual-sitemap.md) for hreflang and localised URLs.
- [How to set page titles and Open Graph tags](title-and-og.md).
