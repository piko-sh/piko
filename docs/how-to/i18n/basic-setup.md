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

Create one JSON file per locale under `i18n/`:

```
myapp/
  i18n/
    en.json
    fr.json
    de.json
```

`i18n/en.json`:

```json
{
  "greeting": "Hello",
  "farewell": "Goodbye",
  "user": {
    "welcome": "Welcome, ${name}"
  }
}
```

Nested keys flatten to dot notation: look up `user.welcome` from templates and Go.

> **Note:** Only `.json` files are loaded from the i18n directory. The basename (without the extension) is the locale code.

## Configure the site

Add an `i18n` section to `config.json`:

```json
{
  "name": "MyApp",
  "i18n": {
    "defaultLocale": "en",
    "strategy": "prefix_except_default",
    "locales": ["en", "fr", "de"]
  }
}
```

| Field | Purpose |
|---|---|
| `defaultLocale` | Used when the URL or host does not pick a specific locale. |
| `strategy` | Controls how the URL reflects the locale. See [how to choose an i18n routing strategy](routing-strategy.md). |
| `locales` | Supported locale codes. Each code needs a matching `i18n/<code>.json` file. |

## Look up translations in templates

Call `T` (global) inside `{{ }}`. The result is a `*Translation` which renders to a string automatically through `fmt.Stringer`:

```piko
<template>
  <h1>{{ T("greeting") }}</h1>
  <p>{{ T("user.welcome").StringVar("name", state.UserName) }}</p>
</template>
```

`StringVar` binds a value to the `${name}` placeholder. The setters are typed: `IntVar` takes an `int`, `MoneyVar` takes `maths.Money`, and so on. See [how to bind typed variables to translations](variable-binding.md) for the full list.

The optional second argument to `T` is a fallback string used when the key is missing in every locale:

```piko
<button>{{ T("button.save", "Save") }}</button>
```

## Look up translations in Go

Call `r.T` (or `r.LT`) from `Render`. Both return a `*Translation`. Terminate with `String()` to get a plain string:

```go
func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
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

> **Note:** Only `lang="json"` is honoured on `<i18n>` blocks. Other `lang` values (yaml, json5, ...) are silently ignored.

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

Without this function the page produces a single route under the default locale, even when other pages opt in. See [how to enable i18n routing for a page](../routing/i18n-page-opt-in.md).

## See also

- [i18n API reference](../../reference/i18n-api.md) for `T`, `LT`, and the `*Translation` surface.
- [How to choose an i18n routing strategy](routing-strategy.md).
- [How to interpolate variables and reference other keys in translations](template-syntax.md).
- [How to pluralise translations](pluralisation.md).
- [How to bind typed variables to translations](variable-binding.md).
- [How to format dates and times for a locale](date-time-formatting.md).
- [How to enable i18n routing for a page](../routing/i18n-page-opt-in.md).
- [How to metadata and OG tags](../metadata-seo/title-and-og.md) for multi-locale SEO.
