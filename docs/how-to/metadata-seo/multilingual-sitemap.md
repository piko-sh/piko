---
title: How to emit hreflang and localised URLs in the sitemap
description: Get localised loc entries and hreflang alternates into sitemap.xml for an i18n site, kept in lockstep with the served routes.
nav:
  sidebar:
    section: "how-to"
    subsection: "metadata-seo"
    order: 40
---

# How to emit hreflang and localised URLs in the sitemap

With i18n configured and `SupportedLocales()` declared on a page, `piko.WithSEO` emits a localised `<loc>` for that page plus one hreflang alternate per locale. The sitemap builder keeps those alternates in lockstep with the routes it actually serves. The builder derives every localised URL from the same helper that registers the runtime routes, so `sitemap.xml` never advertises a path the router does not honour. This guide covers only the `sitemap.xml` side. The HTML-head hreflang tags belong to [How to choose a routing strategy](../i18n/routing-strategy.md). For the full option and type surface see the [SEO API reference](../../reference/seo-api.md).

## Configure i18n and the sitemap hostname

The sitemap fan-out reuses your existing i18n configuration. Set the strategy, locales, and default locale through `piko.WithWebsiteConfig(...)` as described in [How to choose a routing strategy](../i18n/routing-strategy.md). Opt individual pages into locale routing as described in [How to enable i18n routing for a page](../routing/i18n-page-opt-in.md). This how-to does not re-teach those strategies. It assumes they are already in place.

Enable SEO and set the canonical hostname the builder constructs the localised URLs against:

```go
piko.WithSEO(piko.SEOConfig{
    Enabled: true,
    Sitemap: piko.SitemapConfig{
        Hostname: "https://www.example.com",
    },
})
```

You must set `Hostname`. Without it the SEO service returns an error and produces no sitemap.

## Mark a page as multi-locale

A page joins the hreflang fan-out only when it declares a `SupportedLocales()` function in its `<script type="application/x-go">` block:

```go
// SupportedLocales enrols this page in the configured locale set,
// so the sitemap emits a localised <loc> and hreflang alternates.
func SupportedLocales() []string {
    return []string{"en", "fr", "de"}
}
```

Guidance:

- A page without `SupportedLocales()` is single-locale. Its sitemap entry is a bare `<loc>` with no `xhtml:link` alternates.
- The presence of the function is what matters. The sitemap fans out over the project's configured locale set from `piko.WithWebsiteConfig(...)`, not the literal slice inside the function body.

## Read the localised output

For a two-locale page (default `en`, second locale `fr`) under the `prefix_except_default` strategy, the emitted `<url>` block looks like this:

```xml
<url>
  <loc>https://www.example.com/about</loc>
  <lastmod>2026-07-09</lastmod>
  <changefreq>weekly</changefreq>
  <priority>0.5</priority>
  <xhtml:link rel="alternate" hreflang="en" href="https://www.example.com/about"></xhtml:link>
  <xhtml:link rel="alternate" hreflang="fr" href="https://www.example.com/fr/about"></xhtml:link>
  <xhtml:link rel="alternate" hreflang="x-default" href="https://www.example.com/about"></xhtml:link>
</url>
```

What to expect:

- The primary `<loc>` uses the default-locale localised URL. Each supported locale then gets a self-referential `xhtml:link rel="alternate"`, including the default locale itself, plus one `hreflang="x-default"` pointing at the default-locale variant.
- The sitemap builder declares the `xmlns:xhtml="http://www.w3.org/1999/xhtml"` namespace on the `<urlset>` root only when at least one URL in the file carries alternates.
- The exact hreflang hrefs depend on your strategy (`prefix`, `prefix_except_default`, `query-only`, or `disabled`). Under `prefix` the default locale is also prefixed; under `query-only` the paths stay bare and the locale rides on a query parameter.
- Every `<loc>` and `href` matches the route the router registers, because the same routing helper produces both. There is no risk of the sitemap and the served path diverging on trailing slashes.

## Give externally sourced URLs their own hreflang

URLs that come from a build-time `RouteSource` (for example a page templated over a Go registry of locations) are the preferred way to contribute localised dynamic paths. A route source expands each param value against the page's real route pattern and locale set. It percent-encodes each path segment so the emitted URLs stay valid. Bind a page to a source with the `p-route-source` directive and register the source with `piko.WithRouteSource`. See the [SEO API reference](../../reference/seo-api.md) for the `RouteSource`, `RouteContext`, and `RouteURL` shapes.

Sometimes the alternates for a URL are not a simple per-locale fan-out of one path, for example cross-host or hand-curated variants. Supply them verbatim instead of relying on the automatic fan-out. A route source sets them on `RouteURL.Alternates`. A URL provider or dynamic source sets them on `SitemapURLInput.Alternates`. Either way the builder emits the links as-is and does not auto-fan them across the configured locales:

```go
piko.SitemapURLInput{
    Location: "https://www.example.com/about",
    Alternates: []piko.AlternateLink{
        {Rel: "alternate", Hreflang: "en", Href: "https://www.example.com/about"},
        {Rel: "alternate", Hreflang: "fr", Href: "https://fr.example.com/a-propos"},
        {Rel: "alternate", Hreflang: "x-default", Href: "https://www.example.com/about"},
    },
}
```

`Rel` must be the literal string `"alternate"`. This is the escape hatch for cross-host or hand-curated alternates. For ordinary same-pattern localised pages let the automatic fan-out from `SupportedLocales()` or a `RouteSource` handle it.

## See also

- [How to choose a routing strategy](../i18n/routing-strategy.md) for the strategies and the HTML-head hreflang tags.
- [How to enable i18n routing for a page](../routing/i18n-page-opt-in.md) for `SupportedLocales()` opt-in.
- [SEO API reference](../../reference/seo-api.md) for `WithRouteSource`, `SitemapURLInput`, and `AlternateLink`.
- [How to configure the sitemap and robots.txt](sitemap-and-robots.md) for hostname, exclusions, and route rules.
