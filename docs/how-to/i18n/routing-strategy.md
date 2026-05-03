---
title: How to choose an i18n routing strategy
description: Compare query-only (default), prefix, and prefix_except_default strategies for multi-locale URLs.
nav:
  sidebar:
    section: "how-to"
    subsection: "i18n"
    order: 20
---

# How to choose an i18n routing strategy

Piko offers three URL strategies for multi-locale sites. The choice affects SEO, bookmarkability, and the complexity of the reverse-proxy configuration. This guide compares them. See the [i18n API reference](../../reference/i18n-api.md) for the surrounding API.

## `query-only` (default)

A single path per page. Piko detects the locale from a query parameter, the request, or the default. The URL stays bare.

| Locale | URL |
|---|---|
| `en` (default) | `/about` |
| `fr` | `/about?locale=fr` (or detected from `Accept-Language`, cookie, etc.) |
| `de` | `/about?locale=de` |

The canonical query parameter is `locale` (read in `internal/templater/templater_domain/parse_request.go`).

Configure it in `func main` (this is also the default when you omit `Strategy`):

```go
ssr := piko.New(
    piko.WithWebsiteConfig(piko.WebsiteConfig{
        I18n: piko.I18nConfig{
            DefaultLocale: "en",
            Strategy:      "query-only",
            Locales:       []string{"en", "fr", "de"},
        },
    }),
)
```

Use this strategy when SEO impact is minor or when a CDN handles per-locale variants below the URL layer. It keeps the URL space simple.

## `prefix_except_default`

The default locale uses bare URLs. Every other locale prefixes its code.

| Locale | URL |
|---|---|
| `en` (default) | `/about` |
| `fr` | `/fr/about` |
| `de` | `/de/about` |

```go
piko.WithWebsiteConfig(piko.WebsiteConfig{
    I18n: piko.I18nConfig{
        DefaultLocale: "en",
        Strategy:      "prefix_except_default",
        Locales:       []string{"en", "fr", "de"},
    },
})
```

Use this strategy for sites with a clear primary language where the default is also the canonical SEO target.

## `prefix`

Every locale, including the default, has a prefix:

| Locale | URL |
|---|---|
| `en` | `/en/about` |
| `fr` | `/fr/about` |
| `de` | `/de/about` |

```go
piko.WithWebsiteConfig(piko.WebsiteConfig{
    I18n: piko.I18nConfig{
        DefaultLocale: "en",
        Strategy:      "prefix",
        Locales:       []string{"en", "fr", "de"},
    },
})
```

Use this strategy for sites with comparable locale weight where no single language is the canonical default.

> **Note:** Piko routes pages per locale only when the page declares `SupportedLocales()` in its Go script block. A page that omits that function emits a single route under the default locale regardless of the configured strategy. See [how to enable i18n routing for a page](../routing/i18n-page-opt-in.md).

## Emit hreflang and canonical links

`piko.GenerateLocaleHead` produces canonical and `hreflang` alternate-link metadata for the current request. It returns the active locale, the canonical URL, and a slice of alternate-link maps suitable for `Metadata.AlternateLinks`.

```go
import "piko.sh/piko"

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    _, canonical, alternates := piko.GenerateLocaleHead(
        r,
        piko.I18nConfig{
            DefaultLocale: "en",
            Strategy:      "prefix_except_default",
            Locales:       []string{"en", "fr", "de"},
        },
        "/about",
        SupportedLocales(),
    )

    return Response{}, piko.Metadata{
        Title:          props.PageTitle,
        CanonicalURL:   canonical,
        AlternateLinks: alternates,
    }, nil
}
```

| Argument | Purpose |
|---|---|
| `r` | The current request. |
| `piko.I18nConfig{...}` | The site's i18n config. Pass the same struct used at bootstrap. |
| `"/about"` | The page's URL path. The helper adds the locale prefix as needed. |
| `SupportedLocales()` | The page's locale opt-in. Restricts alternates to the locales this page actually emits. Pass `nil` to use every locale in the config. |

> **Note:** `Strategy` is a plain string. There is no `piko.StrategyPrefixExceptDefault` constant. Use the literal `"prefix_except_default"`, `"prefix"`, or `"query-only"`.

Search engines use the alternate links to understand which pages are translations of which.

## Detect the active locale

From any `Render` or action:

```go
locale := r.Locale()              // e.g. "fr"
defaultLocale := r.DefaultLocale() // e.g. "en"
```

## Serve locale-aware pages from a partial

The compiler resolves the `is` attribute on `<piko:partial>` at compile time, and it must be a static literal. The annotator rejects a dynamic expression like `:is="'hero-' + state.Locale"`. Branch with `p-if` to pick a literal partial per locale instead, then put the locale-specific content inside each:

```piko
<template>
  <piko:partial p-if="state.Locale == 'fr'" is="hero-fr" />
  <piko:partial p-else-if="state.Locale == 'es'" is="hero-es" />
  <piko:partial p-else is="hero-en" />
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Response struct {
    Locale string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{Locale: r.Locale()}, piko.Metadata{}, nil
}
</script>
```

With partials named `hero-en`, `hero-fr`, and `hero-de`, Piko selects the correct one per request.

## See also

- [i18n API reference](../../reference/i18n-api.md).
- [How to add translations to a site](basic-setup.md).
- [How to enable i18n routing for a page](../routing/i18n-page-opt-in.md).
- [How to format dates and times for a locale](date-time-formatting.md).
- [Metadata reference](../../reference/metadata-fields.md) for `AlternateLinks`.
