---
title: i18n API
description: Configuration, request accessors, the Translation builder, the DateTime wrapper, template syntax, locale fallback, and the SEO helper.
nav:
  sidebar:
    section: "reference"
    subsection: "runtime"
    order: 50
---

# i18n API

Piko's internationalisation surface has five parts: configuration in `config.json`, accessors on `*piko.RequestData`, the `*Translation` fluent builder, the `DateTime` wrapper for date/time variables, and the `GenerateLocaleHead` SEO helper. Translation sources are JSON files in `i18n/<locale>.json` and per-page `<i18n>` blocks. For task recipes see the [basic setup how-to](../how-to/i18n/basic-setup.md), the [routing strategy how-to](../how-to/i18n/routing-strategy.md), and the [pluralisation how-to](../how-to/i18n/pluralisation.md). For the design rationale see [about i18n](../explanation/about-i18n.md).

## Configuration

### `piko.I18nConfig`

```go
type I18nConfig struct {
    DefaultLocale string   `json:"defaultLocale"`
    Strategy      string   `json:"strategy"`
    Locales       []string `json:"locales"`
}
```

| Field | Type | Purpose |
|---|---|---|
| `DefaultLocale` | `string` | Locale served when no other selection applies. Example `"en"`. |
| `Strategy` | `string` | URL-encoding strategy; one of the values below. |
| `Locales` | `[]string` | Every supported locale code. Each code needs a matching `i18n/<code>.json` file. An empty slice disables i18n. |

### Strategy values

`Strategy` is a string. Three values are recognised:

| Value | Behaviour |
|---|---|
| `"prefix"` | Every locale is prefixed (`/en/about`, `/fr/about`). |
| `"prefix_except_default"` | Default locale is bare; non-default locales are prefixed (`/about`, `/fr/about`). |
| `"domain"` | Locale derived from the request host (`example.com` vs `fr.example.com`). |

> **Note:** No `piko.StrategyPrefix` constant exists. Pass the string literal directly in code (`"prefix_except_default"`) or read it from `config.json`.

### Per-page opt-in

A page enables multi-locale routing by declaring a `SupportedLocales` function in its `<script type="application/x-go">` block:

```go
func SupportedLocales() []string {
    return []string{"en", "fr", "de"}
}
```

Without this function, the page produces a single route under the default locale even when other pages opt in. The router reads the list at build time.

## Request accessors

Methods on `*piko.RequestData`. Available in template expressions (the `r.` prefix is implicit) and in `Render()`.

| Method | Returns | Purpose |
|---|---|---|
| `r.Locale()` | `string` | Active locale for the request (e.g. `"en-GB"`). |
| `r.DefaultLocale()` | `string` | Configured default locale. |
| `r.T(key, fallback...)` | `*Translation` | Look up `key` in the global store first, then the per-page `<i18n>` block. |
| `r.LT(key, fallback...)` | `*Translation` | Look up `key` only in the per-page `<i18n>` block. |
| `r.LF(value)` | `*FormatBuilder` | Wrap a value for locale-aware formatting outside a translation. |

Inside templates, the `T` and `LT` shortcuts call `r.T` and `r.LT`:

```piko
<h1>{{ T("nav.home") }}</h1>
<p>{{ LT("page.heading") }}</p>
```

### Fallback argument

`T` and `LT` are variadic. The first argument is the key; the second optional argument is a fallback string used if every step of the lookup chain fails:

```go
r.T("button.save", "Save")
```

### T versus LT

`r.T(key)` walks the global store first and falls back to the page's local store. Use it for translations that come from `i18n/*.json` and may also be locally overridden.

`r.LT(key)` only ever consults the local store. Use it when a string lives in this PK file's `<i18n>` block and you do not want a same-named global key to shadow it accidentally.

## The Translation builder

`r.T(...)` and `r.LT(...)` return a `*Translation` (alias `piko.Translation`). The builder is mutable, fluent, and pooled; every setter returns the same `*Translation` so calls chain.

Terminate the chain with `String()` to render. The `*Translation` also implements `fmt.Stringer`, which means template expressions render it implicitly:

```piko
{{ T("greeting").StringVar("name", state.User) }}
```

### Variable setters

| Method | Signature | Locale-aware? | Purpose |
|---|---|---|---|
| `StringVar(name, value string)` | `(string, string) *Translation` | No | Bind a string. |
| `IntVar(name string, value int)` | `(string, int) *Translation` | No | Bind a Go `int`. |
| `FloatVar(name string, value float64)` | `(string, float64) *Translation` | Yes | Bind a `float64`; locale picks the decimal separator and thousands grouping. |
| `DecimalVar(name string, value maths.Decimal)` | `(string, maths.Decimal) *Translation` | Yes | Bind an arbitrary-precision decimal from `piko.sh/piko/wdk/maths`. |
| `MoneyVar(name string, value maths.Money)` | `(string, maths.Money) *Translation` | Yes | Bind a money value; locale picks the currency-symbol position, separators, and digit grouping. |
| `BigIntVar(name string, value maths.BigInt)` | `(string, maths.BigInt) *Translation` | Yes | Bind an arbitrary-precision integer with locale grouping. |
| `TimeVar(name string, value time.Time)` | `(string, time.Time) *Translation` | Yes | Bind a `time.Time` with the medium date-time style for the locale. |
| `DateTimeVar(name string, value DateTime)` | `(string, DateTime) *Translation` | Yes | Bind a date/time with an explicit style; see [The DateTime wrapper](#the-datetime-wrapper). |
| `Var(name string, value any)` | `(string, any) *Translation` | No | Generic fallback; uses `fmt.Stringer` if implemented, else `%v`. |

### Pluralisation

| Method | Signature | Purpose |
|---|---|---|
| `Count(n int)` | `(int) *Translation` | Set the count used for CLDR plural form selection; auto-binds `${count}`. |

```go
r.T("cart.items").Count(5).String()
// "5 items" with "one item|${count} items" in en
```

See [How to pluralise translations](../how-to/i18n/pluralisation.md) for the pipe-separated form syntax and per-language rule sets.

### Terminus

| Method | Signature | Purpose |
|---|---|---|
| `String()` | `() string` | Resolve key, select plural form, render template parts, return the string. Releases the builder back to the pool. |

The `*Translation` is also `fmt.Stringer`, so passing it directly into a template expression invokes `String()` implicitly.

### Lifecycle

The builder is pooled. Calling `String()` (or invoking it through `fmt.Stringer`) releases the builder. Do not retain a `*Translation` past the render that produced it.

## Date and time formatting

Two builders cover locale-aware temporal formatting from user code: `TimeVar` on a `*Translation` and `r.LF(value)` for standalone formatting.

### `TimeVar`

Embeds a `time.Time` inside a translation with the medium style for the active locale:

```go
r.T("event.starts").TimeVar("at", record.StartsAt)
```

For a different style, format outside the translation with `LF` and bind the result with `StringVar`:

```go
formatted := r.LF(record.StartsAt).Long().DateOnly().String()
r.T("event.starts").StringVar("at", formatted)
```

### `LF` (FormatBuilder)

`r.LF(value)` returns a `*FormatBuilder` for inline locale-aware formatting. The bare `LF` and `F` helpers are also available in templates. The builder accepts `time.Time`, `int*`, `uint*`, `float32`/`float64`, `string`, `bool`, `maths.Decimal`, `maths.BigInt`, `maths.Money`, and `time.Duration`.

Style methods (each returns the same builder for chaining):

| Method | Style | Date example (`en-GB`) | Time example |
|---|---|---|---|
| `Short()` | Compact | `02/01/2006` | `15:04` |
| `Medium()` | Default | `2 Jan 2026` | `15:04:05` |
| `Long()` | Explicit | `2 January 2026` | `15:04:05 GMT` |
| `Full()` | Most verbose | `Monday, 2 January 2026` | `15:04:05 GMT` |
| `DateOnly()` | Render only the date portion. | | |
| `TimeOnly()` | Render only the time portion. | | |
| `UTC()` | Convert to UTC before formatting. | | |
| `Precision(n)` | Set decimal precision for numeric values. | | |
| `Locale(code)` | Override the active locale for this builder only. | | |

Terminate with `String()` (or rely on `fmt.Stringer` in templates).

> **Note:** `DateTimeVar` exists on `*Translation` and accepts an internal `DateTime` value type. The type lives under `piko.sh/piko/internal/...`, which Go's internal-package rule prevents user code from importing. The same outcomes are reachable through `TimeVar` plus `LF`; use those from user code.

## Translation sources

### Global JSON files

Place one file per locale at `i18n/<locale>.json`. Files outside this directory are ignored. The file's basename (`en.json` → `en`) is the locale code.

```json
{
  "nav": {
    "home": "Home",
    "about": "About"
  },
  "items": "no items|one item|${count} items"
}
```

Nested objects flatten to dot-separated keys. `nav.home` is the lookup key for `"Home"`.

### Per-page `<i18n>` blocks

Declare page-scoped translations inside a `.pk` file:

```html
<i18n lang="json">
{
  "en": { "heading": "Welcome" },
  "fr": { "heading": "Bienvenue" }
}
</i18n>
```

Multiple `<i18n>` blocks per file are allowed and merge.

> **Note:** Only `lang="json"` is honoured. Blocks declared with any other `lang` attribute are silently skipped.

The block's top-level keys are locale codes. Each locale's value is the same nested-key shape as a global file.

## Template-string syntax

Translation values support three constructs.

### Variable interpolation

`${expression}` evaluates a Piko expression at render time. The expression sees variables bound through the `*Var` setters and the implicit `count` from `Count(n)`.

```json
{
  "greeting": "Hello, ${name}!",
  "summary": "${user} has ${count} items"
}
```

Operators and property access are supported (`${user.firstName}`, `${count + 1}`).

### Linked-message references

`@key.path` embeds another translation. The reference is resolved against the same locale and store, with all bound variables passed through:

```json
{
  "common": {
    "appName": "MyApp"
  },
  "welcome": "Welcome to @common.appName, ${name}!"
}
```

Recursion is capped at depth 10 to break cycles.

### Backslash escaping

`\$` and `\@` produce literal `$` and `@` characters:

```json
{
  "email": "support\\@example.com",
  "price": "\\${amount}"
}
```

(The double backslash is JSON's escape; the parser sees `\@` and `\$`.)

## Pluralisation

Translation values may carry pipe-separated plural forms:

```json
{
  "items": "no items|one item|${count} items"
}
```

`Count(n)` selects the form for the active locale. CLDR rules cover six categories (`zero`, `one`, `two`, `few`, `many`, `other`) and the form ordering follows the rule set per locale family. See [How to pluralise translations](../how-to/i18n/pluralisation.md) for ordering tables.

A literal pipe inside a form is escaped as `||`.

## Locale fallback chain

When `r.T(key)` or `r.LT(key)` resolves, the lookup walks:

1. The exact requested locale (e.g. `en-GB`).
2. The base language (e.g. `en`).
3. `DefaultLocale` from `I18nConfig`.
4. The fallback string passed as the second argument to `T`/`LT`.
5. The literal key.

`r.T(key)` consults the global store first and the local store second. `r.LT(key)` only consults the local store.

## SEO metadata

### `piko.GenerateLocaleHead`

```go
func GenerateLocaleHead(
    r *RequestData,
    i18nConfig I18nConfig,
    pagePath string,
    supportedLocalesOverride []string,
) (locale string, canonicalURL string, alternateLinks []map[string]string)
```

Produces canonical URL and `hreflang` alternate-link metadata for the current request.

| Argument | Purpose |
|---|---|
| `r` | Current request. |
| `i18nConfig` | The site's i18n config (typically the same struct passed to bootstrap). |
| `pagePath` | The page's URL path (e.g. `"/about"`); locale prefix is added by the helper. |
| `supportedLocalesOverride` | Optional. Pass the page's `SupportedLocales()` list to restrict the alternates to only the locales that page emits. Pass `nil` to use every locale in `i18nConfig`. |

| Return | Purpose |
|---|---|
| `locale` | Same as `r.Locale()`; convenient for setting `<html lang>`. |
| `canonicalURL` | Canonical URL for the active locale's variant of this page. |
| `alternateLinks` | One map per locale with `"hreflang"` and `"href"` keys, suitable for `<link rel="alternate" hreflang="..." href="..."/>` emission. |

Wire it into a layout partial:

```go
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

## Locale-aware links

The `<piko:a>` element rewrites its `href` for the active locale:

```piko
<piko:a href="/about">{{ T("nav.about") }}</piko:a>
```

A request on `/fr/about` clicking that link lands on `/fr/about`. With `data-locale="en"` the link points at the English variant regardless of the active locale, useful for language switchers.

## See also

- [About i18n](../explanation/about-i18n.md) for the design rationale.
- [How to add translations to a site](../how-to/i18n/basic-setup.md).
- [How to choose an i18n routing strategy](../how-to/i18n/routing-strategy.md).
- [How to interpolate variables and reference other keys in translations](../how-to/i18n/template-syntax.md).
- [How to pluralise translations](../how-to/i18n/pluralisation.md).
- [How to bind typed variables to translations](../how-to/i18n/variable-binding.md).
- [How to format dates and times for a locale](../how-to/i18n/date-time-formatting.md).
- [Metadata reference](metadata-fields.md) for `Metadata.AlternateLinks`.

Integration tests: [`tests/integration/i18n/`](https://github.com/piko-sh/piko/tree/master/tests/integration/i18n).
