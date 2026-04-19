---
title: How to choose an i18n routing strategy
description: Compare URL-prefix, mixed-default, and domain strategies for multi-locale URLs.
nav:
  sidebar:
    section: "how-to"
    subsection: "i18n"
    order: 20
---

# How to choose an i18n routing strategy

Piko offers three URL strategies for multi-locale sites. The choice affects SEO, bookmarkability, and the complexity of the reverse-proxy configuration. This guide compares them. See the [i18n API reference](../../reference/i18n-api.md) for the surrounding API.

## `prefix_except_default`

The default locale uses bare URLs. Every other locale prefixes its code.

| Locale | URL |
|---|---|
| `en` (default) | `/about` |
| `fr` | `/fr/about` |
| `de` | `/de/about` |

Set in `config.json`:

```json
{
  "i18n": {
    "defaultLocale": "en",
    "strategy": "prefix_except_default",
    "locales": ["en", "fr", "de"]
  }
}
```

Use this strategy for sites with a clear primary language where the default is also the canonical SEO target.

## `prefix`

Every locale, including the default, has a prefix:

| Locale | URL |
|---|---|
| `en` | `/en/about` |
| `fr` | `/fr/about` |
| `de` | `/de/about` |

Use this strategy for sites with comparable locale weight where no single language is the canonical default.

## `domain`

Piko derives the locale from the request host:

| Host | Locale |
|---|---|
| `example.com` | `en` |
| `fr.example.com` | `fr` |
| `example.de` | `de` |

This strategy usually requires matching DNS and TLS configuration.

Use this strategy for sites with distinct regional brands or country-specific domains.

## Emit hreflang and canonical links

`piko.GenerateLocaleHead` produces canonical and `hreflang` alternate-link metadata for the current request. It returns three values: the active locale, the canonical URL, and a slice of alternate-link maps suitable for `Metadata.AlternateLinks`.

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
| `piko.I18nConfig{...}` | The site's i18n config; pass the same struct used at bootstrap. |
| `"/about"` | The page's URL path. The helper adds the locale prefix as needed. |
| `SupportedLocales()` | The page's locale opt-in; restricts alternates to the locales this page actually emits. Pass `nil` to use every locale in the config. |

> **Note:** `Strategy` is a plain string. There is no `piko.StrategyPrefixExceptDefault` constant. Use the literal `"prefix_except_default"`, `"prefix"`, or `"domain"`.

Search engines use the alternate links to understand which pages are translations of which.

## Detect the active locale

From any `Render` or action:

```go
locale := r.Locale()              // e.g. "fr"
defaultLocale := r.DefaultLocale() // e.g. "en"
```

## Serve locale-aware pages from a partial

Switch rendering based on the locale without conditionals sprinkled through the template:

```piko
<template>
  <piko:partial :is="'hero-' + state.Locale" />
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
