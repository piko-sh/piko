---
title: i18n advanced
description: Advanced internationalisation patterns including pluralisation, variable interpolation, and locale-aware formatting
nav:
  sidebar:
    section: "guide"
    subsection: "advanced"
    order: 760
---

# Advanced internationalisation

This guide covers advanced i18n patterns not found in the [basic i18n guide](/docs/guide/i18n), including CLDR-compliant pluralisation for complex languages, the full variable binding API, linked message references, locale-aware date/time formatting, and the translation lookup pipeline.

## Template syntax

Piko's i18n system supports three types of template interpolation:

### Expression interpolation

Use `${expression}` syntax to embed variables in translations:

```json
{
  "greeting": "Hello, ${name}!",
  "message": "${user} has ${count} items"
}
```

Expressions are evaluated at render time and support the full Piko expression syntax including operators and property access.

### Linked message references

Use `@key.path` syntax to reference other translation keys:

```json
{
  "common": {
    "appName": "My Application"
  },
  "welcome": "Welcome to @common.appName!"
}
```

Linked messages are resolved recursively with a maximum depth of 10 to prevent infinite loops from circular references. Variables from the parent scope are passed to linked messages.

### Escaping special characters

Use backslash to escape special characters:

```json
{
  "email": "Contact us at support\\@example.com",
  "price": "Cost: \\${99.99}"
}
```

## Pluralisation

Piko implements CLDR plural rules for accurate pluralisation across languages. Use the pipe (`|`) character to separate plural forms. For basic English pluralisation and the `.Count()` API, see [i18n](/docs/guide/i18n).

### French pluralisation

French treats 0 and 1 as singular:

```json
{
  "articles": "un article|${count} articles"
}
```

```go
// French: 0 → "un article", 1 → "un article", 2+ → "N articles"
T("articles").Count(0)  // "un article"
T("articles").Count(1)  // "un article"
T("articles").Count(2)  // "2 articles"
```

### Slavic languages (Russian, Ukrainian, Polish)

Slavic languages require three plural forms following complex rules based on the last digits:

```json
{
  "apples": "${n} яблоко|${n} яблока|${n} яблок"
}
```

| Count | Form | Examples |
|-------|------|----------|
| One | First form | 1, 21, 31, 101, 121... |
| Few | Second form | 2-4, 22-24, 32-34... |
| Many | Third form | 0, 5-20, 25-30, 100... |

### Arabic (six forms)

Arabic has the most complex plural rules with six forms:

```json
{
  "items": "صفر|واحد|اثنان|قليل|كثير|آخر"
}
```

| Category | Count |
|----------|-------|
| Zero | 0 |
| One | 1 |
| Two | 2 |
| Few | 3-10 |
| Many | 11-99 |
| Other | 100+ |

### East Asian languages

Chinese, Japanese, Korean, Vietnamese, Thai, Indonesian, and Malay use a single form for all counts:

```json
{
  "items": "${count}个项目"
}
```

### Escaping pipes in plurals

To include a literal pipe character, escape it with a double pipe (`||`):

```json
{
  "options": "Option A || B|Options: ${count}"
}
```

## Variable binding

The `Translation` type provides a fluent API for binding variables with type-safe methods. For basic `StringVar` and `IntVar` usage, see [i18n](/docs/guide/i18n).

### Numeric variables

```go
T("count").IntVar("n", itemCount)
T("price").FloatVar("amount", 19.99)
```

### High-precision decimals

For financial calculations requiring exact decimal representation:

```go
import "piko.sh/piko/wdk/maths"

price := maths.NewDecimalFromString("99.99")
T("total").DecimalVar("amount", price)
```

### Currency formatting

The `MoneyVar` method provides locale-aware currency formatting:

```go
import "piko.sh/piko/wdk/maths"

price := maths.NewMoney("GBP", "49.99")
T("price.formatted").MoneyVar("price", price)
```

Money values are formatted according to the current locale, including currency symbols, decimal separators, and thousand grouping.

### Large numbers

For numbers exceeding int64 range:

```go
import "piko.sh/piko/wdk/maths"

large := maths.NewBigIntFromString("123456789012345678901234567890")
T("bignum").BigIntVar("value", large)
```

### Method chaining

All variable methods return the Translation for chaining:

```go
result := T("summary").
    StringVar("name", user.Name).
    IntVar("items", cart.Count).
    MoneyVar("total", cart.Total).
    String()
```

## Date and time formatting

Piko provides locale-aware date/time formatting with four style levels.

### DateTime type

The `DateTime` wrapper provides formatting control:

```go
import "piko.sh/piko/internal/i18n/i18n_domain"

dt := i18n_domain.NewDateTime(time.Now())
T("event.date").DateTimeVar("date", dt)
```

### Formatting styles

| Style | Date Example (en-GB) | Time Example |
|-------|---------------------|--------------|
| Short | 02/01/2006 | 15:04 |
| Medium | 2 Jan 2006 | 15:04:05 |
| Long | 2 January 2006 | 15:04:05 MST |
| Full | Monday, 2 January 2006 | 15:04:05 MST |

### Fluent style methods

```go
// Date only with short style
dt.DateOnly().Short()

// Time only with full style
dt.TimeOnly().Full()

// Convert to UTC before formatting
dt.UTC().Medium()
```

### Using time.Time directly

For simple cases, pass `time.Time` directly:

```go
T("created").TimeVar("date", record.CreatedAt)
```

This uses medium style by default.

### Locale-specific patterns

Piko includes patterns for common locales:

| Locale | Short Date | Time |
|--------|-----------|------|
| en-GB | DD/MM/YYYY | 24-hour |
| en-US | MM/DD/YYYY | 12-hour AM/PM |
| de-DE | DD.MM.YYYY | 24-hour |
| ja-JP | YYYY/MM/DD | 24-hour |
| zh-CN | YYYY/MM/DD | 24-hour |

## Translation lookup order

When you call `T()`, Piko searches for translations in this order:

1. **Global store** - Translations from `i18n/*.json` files
2. **Local store** - Component-scoped `<i18n>` block translations
3. **Legacy maps** - Programmatically set translations
4. **Fallback** - Returns the key itself or provided fallback

### Using fallback values

Provide a fallback as the second argument:

```go
T("missing.key", "Default text")
```

## Performance considerations

### Zero-allocation rendering

Piko uses object pooling and pre-parsed templates for minimal allocations:

- `Translation` objects are pooled and reused
- Template expressions are parsed once and cached
- String buffers are pooled via `StrBufPool`

### Pre-parsed templates

Templates are parsed at load time, not render time. The parsed AST is stored with the translation entry for instant rendering.

### FlatBuffer storage

Production builds use FlatBuffer serialisation for:

- Zero-copy deserialisation
- Minimal memory footprint
- Fast random access to translations
