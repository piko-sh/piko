---
title: About indexability
description: Why Piko makes you declare whether a site belongs in search, why it cannot infer that from a build mode, and how the project-level and deploy-level settings differ.
nav:
  sidebar:
    section: "explanation"
    subsection: "operations"
    order: 85
---

# Indexability

Whether a site belongs in a search index is two questions, not one. Piko answers them in two places, because the two answers become available at different times.

The first question is whether the software is ever meant to be public. An admin console, a staff back office or an internal dashboard must stay out of search in every environment. That is a fixed property of the project, and it changes only when the project changes. `Robots.NeverIndex` records it.

The second question is whether a given deploy is the live one. A public marketing site wants search traffic in production and a block on staging, on a pull-request preview, and on a demo box. That is a property of the deploy, and two deploys of the same build can disagree. `Robots.PreviewDeployment` records it.

Neither implies the other, so they are separate fields instead of one flag with a mode attached.

## Why Piko does not guess

Piko cannot make the deploy-level decision for you. The information does not exist yet when the build writes `robots.txt`.

A generate mode such as `generator all` states what to build, not where the result lands. It carries no deployment intent. An earlier version of Piko inferred one anyway. It compared the build mode against the production run mode and treated every generated build as non-production. It then wrote a site-wide `Disallow: /` into artefacts that shipped to live sites. Those sites served a `robots.txt` that advertised a sitemap and forbade fetching it in the same file.

The rule that follows from this is to infer only from a signal that carries the information, never from a proxy that happens to correlate with it. The build mode was a proxy, and proxies fail quietly.

`NeverIndex` therefore lives in source, where the fact lives. Piko reads `PreviewDeployment` when it serves `robots.txt`, which is the first moment a deploy exists with an environment around it.

## Why the framework does not read your environment variable

Piko reads no environment variables, files or command-line flags of its own. Configuration reaches it through Go options. That boundary is deliberate, and it applies here too. Your application reads its own deployment variable and passes the result in.

```go
piko.WithSEO(piko.SEOConfig{
    Sitemap: piko.SitemapConfig{Hostname: "https://example.com"},
    Robots: piko.RobotsConfig{
        PreviewDeployment: !strings.EqualFold(os.Getenv("PIKO_ENV"), "production"),
    },
})
```

The decision stays visible in your source and in your review, instead of depending on a variable name buried in the framework.

Set that variable on the running container, not in the build stage. Piko never writes `PreviewDeployment` into a stored artefact, which is what lets you promote one image from staging to production. A value that leaked into the build stage could not bake a block even if you wanted one. A baked block is one no later deploy can lift.

## Serving only ever tightens

The serving process may add a block. It may never remove one. A build that declared itself unindexable stays that way whatever the deploy says. A mistake in deploy configuration can therefore cause an unwanted block, which you can undo in minutes. It cannot cause an unwanted exposure.

## Why robots.txt is not a de-indexing tool

A `Disallow` asks a crawler not to fetch a URL. It does not remove a URL already in the index, and a crawler can still list a URL that other sites link to. A crawler you have blocked never fetches the page, so it never reads a `noindex` directive in the body.

`NeverIndex` and `PreviewDeployment` therefore also set `X-Robots-Tag: noindex, nofollow` on every response. That header, not `robots.txt`, keeps a site out of an index. Per-page control uses the same mechanism through `Metadata.RobotsRule` and the `p-noindex` template attribute.

## Absence is not a safe default

A site with no SEO configuration serves no `robots.txt`, so `/robots.txt` answers 404. That is the most permissive outcome available, not the least. The robots exclusion standard treats a missing `robots.txt` as permission to crawl everything.

Neither default direction is safe in silence, so Piko does not pick a clever one. It logs the indexing posture it resolved for every build, including the permissive case, and warns when a configuration produces no artefacts. The default stays permissive so an unwired path cannot de-index a live site, and the logging stops that default being invisible.

## See also

- [How to generate a sitemap and robots.txt](../how-to/metadata-seo/sitemap-and-robots.md) for the settings themselves.
- [SEO API reference](../reference/seo-api.md) for the full truth table.
- [About configuration](about-configuration.md) for why Piko reads no environment variables.
