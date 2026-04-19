---
title: How to format dates and times for a locale
description: Use TimeVar in translations and the LF format builder for standalone locale-aware date and time rendering.
nav:
  sidebar:
    section: "how-to"
    subsection: "i18n"
    order: 750
---

# How to format dates and times for a locale

Two paths render dates and times in the active locale. Use `TimeVar` to embed a `time.Time` inside a translation string. Use `LF` (the locale-aware format builder) for standalone formatting outside a translation. Both honour the four standard styles: short, medium, long, full. For binding non-temporal values see [how to bind typed variables to translations](variable-binding.md).

## Embed a time in a translation

For a sentence like "Event starts at ${date}", bind a `time.Time` with `TimeVar`:

```piko
{{ T("event.starts").TimeVar("date", state.EventAt) }}
```

`TimeVar` formats with the locale's medium style by default. For other styles, format outside the translation with `LF` and pass the result as a string:

```piko
{{ T("event.starts").StringVar("date", LF(state.EventAt).Long().DateOnly()) }}
```

`LF` returns a `*FormatBuilder` that implements `fmt.Stringer`, so the templater renders it inline.

## Format a date or time standalone

When the locale-aware string is not embedded in a translation, use `LF` directly. From templates:

```piko
<time :datetime="state.EventAt.Format(time.RFC3339)">
  {{ LF(state.EventAt).Long().DateOnly() }}
</time>
```

From Go:

```go
formatted := r.LF(record.CreatedAt).Short().String()
```

## Pick a formatting style

Each style produces a progressively more verbose rendering. The methods chain on the `FormatBuilder`:

| Method | Effect | Date example (`en-GB`) | Time example |
|---|---|---|---|
| `Short()` | Compact | `02/01/2026` | `15:04` |
| `Medium()` | Default | `2 Jan 2026` | `15:04:05` |
| `Long()` | Explicit | `2 January 2026` | `15:04:05 GMT` |
| `Full()` | Most verbose | `Monday, 2 January 2026` | `15:04:05 GMT` |

Filter the output to date or time only with `DateOnly()` and `TimeOnly()`. Convert to UTC before formatting with `UTC()`:

```go
r.LF(t).Short()              // "02/01/2026 15:04"
r.LF(t).Long().DateOnly()    // "2 January 2026"
r.LF(t).Full().TimeOnly()    // "15:04:05 GMT"
r.LF(t).UTC().Medium()       // converts to UTC, then medium style
```

## Compare locale-specific patterns

The same `time.Time` renders differently per locale without changes to the call site:

| Locale | Short date | Time format |
|---|---|---|
| en-GB | DD/MM/YYYY | 24-hour |
| en-US | MM/DD/YYYY | 12-hour AM/PM |
| de-DE | DD.MM.YYYY | 24-hour |
| ja-JP | YYYY/MM/DD | 24-hour |
| zh-CN | YYYY/MM/DD | 24-hour |

A single `LF(t).Short()` call renders the right format for whichever locale the request resolves to.

## See also

- [i18n API reference](../../reference/i18n-api.md) for `LF` and the full `*Translation` surface.
- [How to bind typed variables to translations](variable-binding.md) for the broader binder API.
- [How to interpolate variables and reference other keys in translations](template-syntax.md) for the placeholder syntax inside translation strings.
- [About i18n](../../explanation/about-i18n.md) for the design rationale behind locale-aware formatting.
