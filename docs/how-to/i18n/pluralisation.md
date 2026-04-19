---
title: How to pluralise translations
description: Write CLDR-compliant plural forms for English, French, Slavic, Arabic, and East Asian languages with the pipe separator.
nav:
  sidebar:
    section: "how-to"
    subsection: "i18n"
    order: 730
---

# How to pluralise translations

Piko implements Common Locale Data Repository (CLDR) plural rules so a single key handles all plural forms of every supported language. Separate forms with the pipe character (`|`) and call `.Count(n)` on the translation. For the basic English case and the `Count` API see [i18n API reference](../../reference/i18n-api.md).

## Write English-style two-form plurals

```json
{
  "items": "one item|${count} items"
}
```

```go
T("items").Count(0)  // "0 items"
T("items").Count(1)  // "one item"
T("items").Count(5)  // "5 items"
```

The first form covers the singular (`n == 1`). Every other count uses the second form.

## Write two-form plurals for French

French treats both 0 and 1 as singular:

```json
{
  "articles": "un article|${count} articles"
}
```

```go
T("articles").Count(0)  // "un article"
T("articles").Count(1)  // "un article"
T("articles").Count(2)  // "2 articles"
```

The framework selects the form using the locale's CLDR rules, so the same key produces different output in `en` and `fr` without per-language code.

## Write Slavic three-form plurals

Russian, Ukrainian, and Polish need three forms based on the last digits:

```json
{
  "apples": "${n} яблоко|${n} яблока|${n} яблок"
}
```

<!-- vale Piko.WeaselWords = NO -->
| CLDR category | Form | Example counts |
|---|---|---|
| One | First | 1, 21, 31, 101, 121 |
| Few | Second | 2-4, 22-24, 32-34 |
| Many | Third | 0, 5-20, 25-30, 100 |
<!-- vale Piko.WeaselWords = YES -->

## Write Arabic six-form plurals

Arabic carries the most categories:

```json
{
  "items": "صفر|واحد|اثنان|قليل|كثير|آخر"
}
```

<!-- vale Piko.WeaselWords = NO -->
| CLDR category | Form | Example counts |
|---|---|---|
| Zero | First | 0 |
| One | Second | 1 |
| Two | Third | 2 |
| Few | Fourth | 3-10 |
| Many | Fifth | 11-99 |
| Other | Sixth | 100+ |
<!-- vale Piko.WeaselWords = YES -->

## Write single-form translations for East Asian languages

Chinese, Japanese, Korean, Vietnamese, Thai, Indonesian, and Malay use one form regardless of count. Provide a single string with no pipe separator:

```json
{
  "items": "${count}个项目"
}
```

`Count(n)` substitutes `${count}` and returns the same string for every value of `n`.

## Escape a literal pipe

A double pipe (`||`) emits a single literal `|`:

```json
{
  "options": "Option A || B|Options: ${count}"
}
```

The first form contains a literal pipe between `A` and `B`. The pipe between the two forms still acts as the separator.

## See also

- [I18n API reference](../../reference/i18n-api.md) for `T()`, `LT()`, and `.Count(n)`.
- [How to bind typed variables to translations](variable-binding.md) for typed substitution.
- [How to interpolate variables and reference other keys in translations](template-syntax.md) for the broader template DSL.
- [About i18n](../../explanation/about-i18n.md) for the design rationale behind CLDR plurals.
