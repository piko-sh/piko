---
title: SEO configuration
description: Every field, type, template attribute, environment variable, and flag on the Piko SEO surface.
nav:
  sidebar:
    section: "reference"
    subsection: "runtime"
    order: 25
---

# SEO configuration

You supply `piko.SEOConfig` through the `piko.WithSEO(...)` functional option, not through a YAML `seo:` block. It controls `sitemap.xml`, `robots.txt`, hreflang alternates, and per-page and per-item overrides. SEO generation is only active when a caller passes `WithSEO` with `Enabled: true` and a non-empty `Sitemap.Hostname`. This page enumerates every field, type, template attribute, environment variable, and flag. For task recipes see the how-to guides on [sitemap and robots.txt](../how-to/metadata-seo/sitemap-and-robots.md) and [a multilingual sitemap](../how-to/metadata-seo/multilingual-sitemap.md).

## Enabling SEO

Three functional options configure the SEO surface. `WithSitemapURLProvider` and `WithRouteSource` both require `WithSEO`.

```go
func WithSEO(seoConfig config.SEOConfig) Option
func WithSitemapURLProvider(provider func(ctx context.Context) ([]seo_dto.SitemapURLInput, error)) Option
func WithRouteSource(source seo_domain.RouteSource) Option
```

| Option | Meaning |
|---|---|
| `piko.WithSEO` | Provides the SEO configuration for sitemap and robots.txt generation. SEO is only active when a caller passes this option with an enabled configuration and a non-empty sitemap hostname. |
| `piko.WithSitemapURLProvider` | Registers a build-time, in-process provider of additional sitemap URLs for dynamic routes whose slugs come from application data instead of content collections. Piko ignores a `nil` provider. Requires `WithSEO`. |
| `piko.WithRouteSource` | Registers a composable build-time `RouteSource` for a page bound to it with the `p-route-source` directive, expanding param values against the page's real route pattern so localised paths and hreflang are correct. Preferred over `WithSitemapURLProvider` for dynamic pages templated over a Go registry. Piko ignores a `nil` source. Requires `WithSEO`. |

## `SEOConfig`

```go
type SEOConfig struct {
    Robots  RobotsConfig
    Sitemap SitemapConfig
    Enabled bool
}
```

| Field | Type | Meaning / default |
|---|---|---|
| `Robots` | `RobotsConfig` | robots.txt settings. See [RobotsConfig](#robotsconfig). |
| `Sitemap` | `SitemapConfig` | sitemap.xml settings. See [SitemapConfig](#sitemapconfig). |
| `Enabled` | `bool` | Controls whether SEO file generation is active. Default `true`. |

## `SitemapConfig`

```go
type SitemapConfig struct {
    Sitemaps              map[string]SitemapChunkConfig
    CacheMaxAgeSeconds    *int
    Hostname              string
    Exclude               []string
    Sources               []string
    Defaults              SitemapEntryDefaults
    RouteRules            []SitemapRouteRule
    MaxURLsPerSitemap     int
    DiscoverImages        bool
    IncludeAuthGatedPages bool
    GitLastMod            bool
}
```

| Field | Type | Meaning / default |
|---|---|---|
| `Sitemaps` | `map[string]SitemapChunkConfig` | Named sitemap chunks for large sites. Each chunk gets its own sources, and the builder lists it in a sitemap index. |
| `CacheMaxAgeSeconds` | `*int` | Cache-Control max-age in seconds for the served sitemap.xml, its chunks, and robots.txt. `nil` uses `600`; `0` disables caching. |
| `Hostname` | `string` | Canonical base URL (for example `https://example.com`). Required to build full URLs; SEO is inactive when empty. |
| `Exclude` | `[]string` | Glob patterns for routes to leave out of the sitemap. |
| `Sources` | `[]string` | Runtime API endpoints, each returning a JSON array of [`SitemapURLInput`](#routesource-routesourcefunc-routeurl-sitemapurlinput-alternatelink). |
| `Defaults` | `SitemapEntryDefaults` | Default values for sitemap entry fields. See [SitemapEntryDefaults](#sitemapentrydefaults). |
| `RouteRules` | `[]SitemapRouteRule` | Per-route SEO metadata matched by glob, without editing pages. See [SitemapRouteRule](#sitemaprouterule). |
| `MaxURLsPerSitemap` | `int` | When the URL count exceeds this value, the builder splits the sitemap and generates an index. Default `5000`; validated `min=1,max=50000`. |
| `DiscoverImages` | `bool` | Automatically discover and include images in the sitemap. Default `true`. |
| `IncludeAuthGatedPages` | `bool` | Include `AuthPolicy`-gated pages in the sitemap. Default `false` (excluded). |
| `GitLastMod` | `bool` | Derive a static page's `<lastmod>` from its last git commit date instead of file mtime. Default `false`. |

### `SitemapEntryDefaults`

```go
type SitemapEntryDefaults struct {
    ChangeFreq string
    Priority   float32
}
```

| Field | Type | Meaning / default |
|---|---|---|
| `ChangeFreq` | `string` | Default changefreq. Default `weekly`; validated `oneof=always hourly daily weekly monthly yearly never`. |
| `Priority` | `float32` | Default priority for entries. Default `0.5`; validated `min=0,max=1`. |

### `SitemapChunkConfig`

```go
type SitemapChunkConfig struct {
    Sources []string
}
```

| Field | Type | Meaning / default |
|---|---|---|
| `Sources` | `[]string` | Runtime API endpoints for this named sitemap chunk. |

## `RobotsConfig`

```go
type RobotsConfig struct {
    CustomRules                []RobotsRuleGroup
    BlockAiBots                bool
    BlockNonSeoBots            bool
    AllowNonProductionIndexing bool
}
```

| Field | Type | Meaning / default |
|---|---|---|
| `CustomRules` | `[]RobotsRuleGroup` | Custom robots.txt rules per user agent. Appended after the base and bot-blocking groups. |
| `BlockAiBots` | `bool` | Block known AI crawler bots (the `seo_dto.AIBots` list). Default `false`. See [AI and non-SEO bot lists](#ai-and-non-seo-bot-lists). |
| `BlockNonSeoBots` | `bool` | Block known non-SEO web scrapers (the `seo_dto.NonSEOBots` list). Default `false`. See [AI and non-SEO bot lists](#ai-and-non-seo-bot-lists). |
| `AllowNonProductionIndexing` | `bool` | Opt out of blocking crawlers in non-production builds. Default `false`. See [Non-production behaviour](#non-production-behaviour). |

### `RobotsRuleGroup`

```go
type RobotsRuleGroup struct {
    UserAgents []string
    Disallow   []string
    Allow      []string
}
```

| Field | Type | Meaning / default |
|---|---|---|
| `UserAgents` | `[]string` | User agent names this group applies to. Use `"*"` for all bots. Validated `required,min=1`. |
| `Disallow` | `[]string` | URL path patterns these user agents cannot crawl. |
| `Allow` | `[]string` | URL path patterns these user agents may crawl, overriding more general `Disallow` rules. |

### Non-production behaviour

When a build is non-production and `AllowNonProductionIndexing` is `false`, the base group is `User-agent: *` / `Disallow: /` (a site-wide block) and the robots builder logs a warning. When `AllowNonProductionIndexing` is `true`, the base group is the permissive `User-agent: *` / `Allow: /` even in non-production, and the robots builder logs a different warning. In production the base group is always the permissive `User-agent: *` / `Allow: /`. The SEO service assumes production when no caller supplies a production-mode signal, so a missing wiring path fails open instead of de-indexing a live site. `BlockAiBots` and `BlockNonSeoBots` are independent of production mode. Each, when true, appends an extra group with `Disallow: /` targeting its fixed user-agent list.

### AI and non-SEO bot lists

`BlockAiBots` blocks the `seo_dto.AIBots` list: `GPTBot`, `ChatGPT-User`, `ClaudeBot`, `anthropic-ai`, `Applebot-Extended`, `Bytespider`, `CCBot`, `cohere-ai`, `Diffbot`, `FacebookBot`, `Google-Extended`, `ImagesiftBot`, `PerplexityBot`, `OmigiliBot`, `Omigili`.

`BlockNonSeoBots` blocks the `seo_dto.NonSEOBots` list: `AhrefsBot`, `SemrushBot`, `DotBot`, `Baiduspider`, `Nuclei`, `WikiDo`, `Riddler`, `PetalBot`, `Zoominfobot`, `Go-http-client`, `Node/simplecrawler`, `CazoodleBot`, `dotbot/1.0`, `Gigabot`, `Barkrowler`, `BLEXBot`, `magpie-crawler`.

## `SitemapRouteRule`

```go
type SitemapRouteRule struct {
    Pattern    string
    Priority   *float32
    ChangeFreq string
    Robots     string
    Exclude    bool
}
```

| Field | Type | Meaning / default |
|---|---|---|
| `Pattern` | `string` | Glob matched against the route pattern. Uses the same matcher as `Sitemap.Exclude`. |
| `Priority` | `*float32` | Overrides priority for matching routes; `nil` inherits. Validated `min=0,max=1`. |
| `ChangeFreq` | `string` | Overrides changefreq for matching routes; empty inherits. Validated `oneof=always hourly daily weekly monthly yearly never`. |
| `Robots` | `string` | Robots rule for matching routes. When it contains `"noindex"`, the builder drops the routes from the sitemap. |
| `Exclude` | `bool` | When `true`, removes matching routes from the sitemap entirely. |

The builder evaluates rules first-match-wins against the route pattern. Glob semantics use `filepath.Match`, plus a trailing `**` that matches every route sharing the preceding prefix, so `Pattern` values such as `/`, `/blog/**`, and `/*/search/*` match as expected. The overall precedence for priority and changefreq is:

1. Page declaration (a per-page override on `PageSEOMetadata`, from a `p-*` template attribute or collection-item frontmatter).
2. Matching route rule.
3. Configured `Sitemap.Defaults`.

The resolved order runs page declaration, then route rule, then defaults.

## `RouteSource`, `RouteSourceFunc`, `RouteURL`, `SitemapURLInput`, `AlternateLink`

`RouteSource` is a build-time enumerator of the concrete URLs for a page whose dynamic segment is not a content collection. Register one with `WithRouteSource` and bind a page to it with the `p-route-source` directive.

```go
type RouteSource interface {
    Name() string
    Enumerate(ctx context.Context, rc RouteContext) ([]RouteURL, error)
}

type RouteSourceFunc struct {
    Fn         func(ctx context.Context, rc RouteContext) ([]RouteURL, error)
    SourceName string
}

type RouteContext struct {
    SourceName    string
    RoutePattern  string
    ParamName     string
    DefaultLocale string
    Strategy      string
    Locales       []string
}

type RouteURL struct {
    ParamValue string
    Locales    []string
    Alternates []AlternateLink
    SEO        SitemapURLInput
}
```

`RouteContext` fields work as follows. `SourceName` is the `p-route-source` value that bound the source to the page. `RoutePattern` is the brace-retaining route pattern (for example `/services{locationslug}/kubernetes`). `ParamName` is the dynamic segment the source enumerates (for example `locationslug`). `DefaultLocale`, `Strategy`, and `Locales` carry the i18n configuration. `RouteContext` exposes the method `Expand(paramValue, locale string) string`. `Expand` applies the strategy, substitutes the param value, and percent-encodes each path segment so the emitted URL matches the served route.

`RouteURL` fields work as follows. `ParamValue` substitutes into the pattern's `{param}` (for example `-jersey`). An empty `Locales` falls back to `RouteContext.Locales`. When a source populates `Alternates`, the builder uses it verbatim instead of automatic cross-linking. `SEO` carries the per-URL lastmod, changefreq, priority, images, videos, and news.

```go
type SitemapURLInput struct {
    News         *NewsInputEntry   `json:"news,omitempty"`
    Location     string            `json:"loc"`
    LastMod      string            `json:"lastmod,omitempty"`
    ChangeFreq   string            `json:"changefreq,omitempty"`
    Images       []string          `json:"images,omitempty"`
    Videos       []VideoInputEntry `json:"videos,omitempty"`
    ImageEntries []ImageInputEntry `json:"imageEntries,omitempty"`
    Alternates   []AlternateLink   `json:"alternates,omitempty"`
    Priority     float32           `json:"priority,omitempty"`
}
```

`SitemapURLInput` is the JSON input shape for both `Sources` endpoints and `WithSitemapURLProvider`. The builder normalises `LastMod` to `YYYY-MM-DD` on ingest. `ImageEntries` takes precedence over `Images`, and `Priority` is `0.0` to `1.0`.

```go
type AlternateLink struct {
    Rel      string `xml:"rel,attr"`
    Hreflang string `xml:"hreflang,attr"`
    Href     string `xml:"href,attr"`
}
```

`AlternateLink` is one hreflang alternate link for a sitemap URL. `Rel` must be `"alternate"`. `Hreflang` is a language code (for example `en`, `en-GB`, `fr`, `es-MX`), and `Href` is the absolute alternate URL.

A representative `RouteSource` contribution using the `RouteSourceFunc` adapter:

```go
piko.WithRouteSource(piko.RouteSourceFunc{
    SourceName: "office-locations",
    Fn: func(ctx context.Context, rc piko.RouteContext) ([]piko.RouteURL, error) {
        return []piko.RouteURL{
            {
                ParamValue: "-jersey",
                Locales:    []string{"en", "fr"},
                SEO: piko.SitemapURLInput{
                    LastMod:    "2026-07-09",
                    ChangeFreq: "monthly",
                    Priority:   0.8,
                },
            },
        }, nil
    },
})
```

### Per-source caps and skip conditions

- The builder caps a build-time `WithSitemapURLProvider` contribution at `100000` URLs, then truncates and logs the excess.
- The builder caps all build-time `RouteSource`s combined at `100000` URLs, counted after per-locale fan-out. On reaching the cap, enumeration stops, and the builder truncates to the cap and logs the event.
- The builder skips a `RouteURL` whose `ParamValue` is `""`, `.`, or `..`.
- The builder skips and logs a page declaring `p-route-source` without a `p-param` (empty param name).
- The builder skips and logs a page naming an unregistered source.
- The builder excludes a page from the sitemap when it matches `Sitemap.Exclude`, is auth-gated without `IncludeAuthGatedPages`, has a `Robots` value containing `"noindex"` (from `PageSEOMetadata` or a matching route rule), or matches a route rule with `Exclude: true`.
- The builder additionally filters discovered pages by an indexability check. It drops any route still containing a `{` placeholder, or any segment starting with `!` (such as `/!404` or `/!error`). The builder deliberately does not apply this brace and bang filter to route-source pages.
- The builder deduplicates URLs by `Location`, with discovered URLs winning over provider, dynamic, and route-source URLs, then sorts by `Location`.

## Per-page and per-item overrides

Two override surfaces feed `seo_dto.PageSEOMetadata`: `<template>`-tag attributes on a page component, and reserved frontmatter keys on a collection item. For recipes see the [sitemap and robots.txt](../how-to/metadata-seo/sitemap-and-robots.md) and [multilingual sitemap](../how-to/metadata-seo/multilingual-sitemap.md) how-tos. This section is the field-list home.

### Template attributes

These are attributes on the page's `<template>` tag.

| Attribute | Value | Effect |
|---|---|---|
| `p-route-source` | source name | Binds the page to a build-time `RouteSource` that enumerates its dynamic URLs. |
| `p-param` | segment name | Names the dynamic route segment the bound `RouteSource` enumerates (for example `locationslug`). No default for route-source binding. |
| `p-noindex` | presence-only | Keeps the page out of the sitemap and marks it noindex. |
| `p-sitemap-priority` | `0.0`-`1.0` string | Overrides the sitemap priority. |
| `p-sitemap-changefreq` | changefreq string | Overrides the sitemap changefreq. |
| `p-canonical` | URL | Sets an explicit canonical URL for the page. |

A page binds to a `RouteSource` by declaring `p-route-source` and `p-param` on its `<template>` tag, with a matching `{param}` segment in its file path:

```html
<template p-route-source="stores" p-param="locationslug">
  <main>
    <h1>{{ props.Store.Name }}</h1>
  </main>
</template>
```

### PageSEOMetadata

```go
type PageSEOMetadata struct {
    LastModified     *time.Time
    Priority         *float32
    News             *NewsInputEntry
    RobotsRule       string
    ChangeFrequency  string
    Canonical        string
    SupportedLocales []string
    ImageURLs        []string
    Videos           []VideoInputEntry
}
```

| Field | Type | Meaning / default |
|---|---|---|
| `LastModified` | `*time.Time` | Last-changed date; `nil` falls back to file modification time. |
| `Priority` | `*float32` | Sitemap priority override; `nil` inherits route rule or default. |
| `News` | `*NewsInputEntry` | Optional news sitemap entry. |
| `RobotsRule` | `string` | Robots meta value; a value containing `"noindex"` drops the page from the sitemap. |
| `ChangeFrequency` | `string` | Changefreq override; empty inherits route rule or default. |
| `Canonical` | `string` | Explicit canonical URL; empty lets the framework derive one. |
| `SupportedLocales` | `[]string` | Language codes for hreflang alternates. |
| `ImageURLs` | `[]string` | Image URLs for the image sitemap extension. |
| `Videos` | `[]VideoInputEntry` | Video sitemap entries. |

`NewsInputEntry` fields (all `string`): `PublicationName`, `PublicationLanguage`, `PublicationDate`, `Title`.

### Collection-item frontmatter keys

For a collection item (one markdown file), the following keys under the item's `page` frontmatter map onto `PageSEOMetadata`. Basic overrides:

| Key | Type | Effect |
|---|---|---|
| `noindex` | bool | When `true`, sets `RobotsRule` to `"noindex"` (drops the item from the sitemap). |
| `changefreq` | string | Sets `ChangeFrequency`. |
| `canonical` | string | Sets `Canonical`. |
| `priority` | number or numeric string | Sets `Priority`. The builder rejects non-finite values. |

Rich-media overrides (flat scalar keys). The builder emits a video entry only when both `sitemapVideoTitle` and `sitemapVideoThumbnail` are present. It emits a news entry only when both `sitemapNewsPublication` and `sitemapNewsDate` are present:

| Key | Type | Effect |
|---|---|---|
| `sitemapImage` | string | Appended to `ImageURLs` (renders `<image:image>`). |
| `sitemapVideoTitle` | string | Video title (required to emit a video entry). |
| `sitemapVideoThumbnail` | string | Video thumbnail (required to emit a video entry). |
| `sitemapVideoDescription` | string | Video description. |
| `sitemapVideoPlayer` | string | Player location. |
| `sitemapVideoContent` | string | Content location. |
| `sitemapVideoDate` | string | Publication date. |
| `sitemapVideoDuration` | int | Duration in seconds, clamped to `0`-`28800`. |
| `sitemapNewsPublication` | string | News publication name (required to emit a news entry). |
| `sitemapNewsDate` | string | News date (required to emit a news entry). |
| `sitemapNewsLanguage` | string | News publication language. |
| `sitemapNewsTitle` | string | News article title. |

The builder derives an item's last-modified from the standard `UpdatedAt` then `PublishedAt` metadata keys.

### Multi-locale opt-in

A page opts into locale routing by declaring a `SupportedLocales()` function in its `<script type="application/x-go">` block. The sitemap builder then enrols the page in the full configured locale set (from `piko.WithWebsiteConfig(...)` `I18n.Locales`), not the literal list it returns. For a page with more than one locale, the sitemap builder emits one self-referential `<xhtml:link rel="alternate">` per locale. It also emits one `hreflang="x-default"` pointing at the default-locale variant. A single-locale page emits no alternates. i18n strategies are `prefix`, `prefix_except_default`, `query-only` (default), and `disabled`.

## Environment variables and flags

Fields not listed below (`Sitemaps`, `Defaults` as a whole, `RouteRules`, `CustomRules`, and every `SitemapRouteRule` and `RobotsRuleGroup` field) are YAML/JSON only.

| Environment variable | Flag | Default | Field |
|---|---|---|---|
| `PIKO_SEO_ENABLED` | `seoEnabled` | `true` | `SEOConfig.Enabled` |
| `PIKO_SEO_SITEMAP_CACHE_MAX_AGE` | `sitemapCacheMaxAge` | `600` | `SitemapConfig.CacheMaxAgeSeconds` |
| `PIKO_SEO_SITEMAP_HOSTNAME` | `sitemapHostname` | (none) | `SitemapConfig.Hostname` |
| `PIKO_SEO_SITEMAP_EXCLUDE` | `sitemapExclude` | (none) | `SitemapConfig.Exclude` |
| `PIKO_SEO_SITEMAP_SOURCES` | `sitemapSources` | (none) | `SitemapConfig.Sources` |
| `PIKO_SEO_SITEMAP_MAX_URLS` | `sitemapMaxUrls` | `5000` | `SitemapConfig.MaxURLsPerSitemap` |
| `PIKO_SEO_SITEMAP_DISCOVER_IMAGES` | `sitemapDiscoverImages` | `true` | `SitemapConfig.DiscoverImages` |
| `PIKO_SEO_SITEMAP_INCLUDE_AUTH_GATED` | `sitemapIncludeAuthGated` | `false` | `SitemapConfig.IncludeAuthGatedPages` |
| `PIKO_SEO_SITEMAP_GIT_LASTMOD` | `sitemapGitLastMod` | `false` | `SitemapConfig.GitLastMod` |
| `PIKO_SEO_SITEMAP_DEFAULT_CHANGEFREQ` | `sitemapDefaultChangeFreq` | `weekly` | `SitemapEntryDefaults.ChangeFreq` |
| `PIKO_SEO_SITEMAP_DEFAULT_PRIORITY` | `sitemapDefaultPriority` | `0.5` | `SitemapEntryDefaults.Priority` |
| `PIKO_SEO_ROBOTS_BLOCK_AI_BOTS` | `robotsBlockAiBots` | `false` | `RobotsConfig.BlockAiBots` |
| `PIKO_SEO_ROBOTS_BLOCK_NON_SEO_BOTS` | `robotsBlockNonSeoBots` | `false` | `RobotsConfig.BlockNonSeoBots` |
| `PIKO_SEO_ROBOTS_ALLOW_NONPROD_INDEXING` | `robotsAllowNonProdIndexing` | `false` | `RobotsConfig.AllowNonProductionIndexing` |

## See also

- [Sitemap and robots.txt how-to](../how-to/metadata-seo/sitemap-and-robots.md).
- [Multilingual sitemap how-to](../how-to/metadata-seo/multilingual-sitemap.md).
- [i18n page opt-in how-to](../how-to/routing/i18n-page-opt-in.md).
- [Metadata fields reference](metadata-fields.md) for the per-page `<title>` and `<meta>` surface.
- [i18n API reference](i18n-api.md) for the i18n strategy and locale configuration.

**Used in.** Any project supplying `piko.WithSEO(...)` to generate `sitemap.xml` and `robots.txt`.
