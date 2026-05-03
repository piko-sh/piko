---
title: How to add translations to a site
description: Create locale files, wire them up, and render translated strings in templates and Render functions.
nav:
  sidebar:
    section: "how-to"
    subsection: "i18n"
    order: 10
---

# How to add translations to a site

This guide walks through adding translations to a Piko project. It covers creating locale files, configuring the site, and rendering translated strings from templates and Go. See the [i18n API reference](../../reference/i18n-api.md) for the full surface.

## Create locale files

Create one JSON file per locale under `locales/` (the Piko default, configurable via `piko.WithI18nSourceDir(...)`):

```
myapp/
  locales/
    en.json
    fr.json
    de.json
```

`locales/en.json`:

```json
{
  "greeting": "Hello",
  "farewell": "Goodbye",
  "user": {
    "welcome": "Welcome, ${name}"
  }
}
```

Nested keys flatten to dot notation. Look up `user.welcome` from templates and Go.

> **Note:** Piko loads only `.json` files from the directory. The basename (without the extension) is the locale code.

## Configure the site

Pass the i18n configuration through `piko.WithWebsiteConfig` in `func main`:

```go
ssr := piko.New(
    piko.WithWebsiteConfig(piko.WebsiteConfig{
        Name: "MyApp",
        I18n: piko.I18nConfig{
            DefaultLocale: "en",
            Strategy:      "prefix_except_default",
            Locales:       []string{"en", "fr", "de"},
        },
    }),
)
```

| Field | Purpose |
|---|---|
| `DefaultLocale` | Piko uses this when the URL or host does not pick a specific locale. |
| `Strategy` | Controls how the URL reflects the locale. See [how to choose an i18n routing strategy](routing-strategy.md). |
| `Locales` | Supported locale codes. Each code needs a matching `locales/<code>.json` file. |

## Look up translations in templates

Call `T` inside `{{ }}`. The result is a `*Translation` which renders to a string automatically through `fmt.Stringer`:

```piko
<template>
  <h1>{{ T("greeting") }}</h1>
  <p>{{ T("user.welcome").StringVar("name", state.UserName) }}</p>
</template>
```

`T`, `LT`, `F`, and `LF` are template-DSL globals: the annotator registers them as builtin identifiers inside the `{{ }}` expression language and the code generator emits them as `r.T(...)`, `r.LT(...)`, `r.F(...)`, and `r.LF(...)` calls on the request data. They are not exported as package-level Go functions on `piko.sh/piko`. From a Go `Render` body call the methods directly on `r` (see the next section).

`StringVar` binds a value to the `${name}` placeholder. Each setter takes a specific type: `IntVar` accepts an `int`, `MoneyVar` accepts `maths.Money`, and so on. See [how to bind typed variables to translations](variable-binding.md) for the full list.

The optional second argument to `T` is a fallback string Piko uses when the key is missing in every locale:

```piko
<button>{{ T("button.save", "Save") }}</button>
```

## Look up translations in Go

Call `r.T` (or `r.LT`) from `Render`. Both return a `*Translation`. Terminate with `String()` to get a plain string:

```go
type Props struct {
    UserName string `prop:"userName"`
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    greeting := r.T("greeting", "Hello").String()

    welcome := r.T("user.welcome").
        StringVar("name", props.UserName).
        String()

    return Response{
        Greeting: greeting,
        Welcome:  welcome,
    }, piko.Metadata{}, nil
}
```

The same builder methods are available from Go and from templates. Templates only differ in that they call `String()` implicitly.

## Component-scoped translations

For strings used only in one `.pk` file, declare them inline:

```piko
<template>
  <button>{{ LT("button_label") }}</button>
</template>

<i18n lang="json">
{
  "en": { "button_label": "Click me" },
  "fr": { "button_label": "Cliquez ici" }
}
</i18n>
```

`LT` (local translation) only consults the page's `<i18n>` block. `T` consults the global store first and the local block as a fallback. Use `LT` when you want to be explicit that the string lives in this file.

> **Note:** `<i18n>` blocks only honour `lang="json"`. Other `lang` values such as `yaml` or `json5` are silently ignored.

## Switch locales in the URL

With `"strategy": "prefix_except_default"`, the same page responds at:

- `/about` (English, default).
- `/fr/about` (French).
- `/de/about` (German).

The active locale is available from `r.Locale()`. The default is `r.DefaultLocale()`.

A page only emits multi-locale routes if it declares `SupportedLocales()` in its Go script block:

```go
func SupportedLocales() []string {
    return []string{"en", "fr", "de"}
}
```

Without it the page produces a single route under the default locale, even when other pages opt in. See [how to enable i18n routing for a page](../routing/i18n-page-opt-in.md).

## Add another locale to an existing site

To add a third (or fourth) language to a site that already has two:

1. Add the locale code to the `Locales` slice in your `piko.WithWebsiteConfig(piko.WebsiteConfig{I18n: piko.I18nConfig{Locales: ...}})` call.
2. Create `locales/<code>.json` with translations for every key already used by `T(...)` and `LT(...)`.
3. If you use locale-scoped collections, create `content/<collection>/<code>/` and copy each item, translating its body and frontmatter. The slug stays the same so cross-locale links resolve.
4. Redeploy. Every page that already calls `T` or `LT` picks up the new locale with no further code changes.

Missing keys fall back to the default locale, so a partially translated rollout is safe.

## See also

- [i18n API reference](../../reference/i18n-api.md) for `T`, `LT`, and the `*Translation` surface.
- [About i18n](../../explanation/about-i18n.md) for why locales, fallback chains, and routing work the way they do.
- [How to choose an i18n routing strategy](routing-strategy.md).
- [How to interpolate variables and reference other keys in translations](template-syntax.md).
- [How to pluralise translations](pluralisation.md).
- [How to bind typed variables to translations](variable-binding.md).
- [How to format dates and times for a locale](date-time-formatting.md).
- [How to enable i18n routing for a page](../routing/i18n-page-opt-in.md).
- [How to metadata and OG tags](../metadata-seo/title-and-og.md) for multi-locale SEO.
