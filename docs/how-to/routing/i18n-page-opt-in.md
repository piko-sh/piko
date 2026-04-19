---
title: How to enable i18n routing for a page
description: Define SupportedLocales on a page so the router generates a route per locale under the active i18n strategy.
nav:
  sidebar:
    section: "how-to"
    subsection: "routing"
    order: 720
---

# How to enable i18n routing for a page

Pages opt into i18n routing one at a time. Without `SupportedLocales`, a page receives a single route under the default locale, even when global i18n configuration enables multi-locale routing. For the routing primitives see [routing rules reference](../../reference/routing-rules.md). For the i18n service surface see [i18n API reference](../../reference/i18n-api.md). For the rationale behind opt-in see [about i18n](../../explanation/about-i18n.md).

> **Note:** Multi-locale routing is page-scoped, not project-scoped. Configuring `i18n.locales` in `config.json` enables the strategy; adding `SupportedLocales()` to a page is what makes that page emit one route per locale.

## Declare the page's supported locales

Add a `SupportedLocales` function to the page's `<script type="application/x-go">` block:

```go
// SupportedLocales enables i18n routing for this page.
// One route is generated for each locale listed here.
func SupportedLocales() []string {
    return []string{"en", "fr", "de"}
}
```

The router reads this list at build time and registers a route per locale, using the global i18n routing strategy (see [How to choose a routing strategy](../i18n/routing-strategy.md)).

## Resulting routes under `prefix_except_default`

For a page at `pages/about.pk` with the default English locale:

- `en` (default): `/about`
- `fr`: `/fr/about`
- `de`: `/de/about`

Switching the strategy to `prefix` adds the prefix to every locale, including the default. Switching to `domain` derives the locale from the request host instead of the URL path.

## Combine with dynamic segments

Dynamic segments compose with locale prefixes:

```go
// pages/blog/{slug}.pk
func SupportedLocales() []string {
    return []string{"en", "fr"}
}
```

Generates `/blog/:slug` and `/fr/blog/:slug`. The locale resolves before parameter binding, so the action sees the bound `slug` regardless of locale.

## See also

- [Routing rules reference](../../reference/routing-rules.md) for path patterns and priority.
- [I18n API reference](../../reference/i18n-api.md) for the global strategy options.
- [About i18n](../../explanation/about-i18n.md) for the design rationale.
- [How to choose a routing strategy](../i18n/routing-strategy.md) for picking between `prefix`, `prefix_except_default`, and `domain`.
