---
title: How to interpolate variables and reference other keys in translations
description: Use ${expression}, @key.path, and the \$ and \@ escapes inside translation strings.
nav:
  sidebar:
    section: "how-to"
    subsection: "i18n"
    order: 720
---

# How to interpolate variables and reference other keys in translations

Translation strings support three template constructs. The constructs are variable interpolation with `${expression}`, key references with `@key.path`, and the two escape sequences `\$` and `\@` for literal `$` and `@` characters. For the surrounding API see [i18n API reference](../../reference/i18n-api.md). For the rationale behind a small DSL inside translations see [about i18n](../../explanation/about-i18n.md).

## Interpolate a variable

Use `${expression}` to embed a variable. The expression evaluates at render time and supports the full Piko expression syntax, including operators and property access:

```json
{
  "greeting": "Hello, ${name}!",
  "summary": "${user} has ${count} items"
}
```

Bind values from Go using the typed setters on `Translation`. See [How to bind typed variables to translations](variable-binding.md) for the binder methods.

## Reference another translation key

Use `@key.path` to embed one translation inside another. Piko resolves linked keys recursively, capped at depth 10 to break circular references. Variables from the outer scope flow through to the linked translation:

```json
{
  "common": {
    "appName": "My Application"
  },
  "welcome": "Welcome to @common.appName!"
}
```

Linked references reduce duplication when the same brand name, product label, or boilerplate phrase appears across multiple translation keys.

## Escape `$` and `@`

The parser recognises exactly two escape sequences. `\$` writes a literal `$` and `\@` writes a literal `@`. The parser processes no other backslash escape. A backslash before any other character stays verbatim, so `\\` in the rendered string stays as `\\`, not `\`.

Use these escapes when a literal `$` or `@` must sit next to text that would otherwise start an interpolation or key reference:

```json
{
  "email": "Contact us at support\\@example.com",
  "price": "The figure shown is in dollars: \\$"
}
```

The double backslash in JSON encodes a single backslash in the resulting string, which the template parser then consumes together with the following `$` or `@`. The `support\@example.com` form prevents the parser from treating `@example` as a key reference. The `\$` form yields a bare `$` without starting an interpolation.

## Combine all three in one entry

```json
{
  "common": {
    "currency": "USD"
  },
  "checkout.summary": "Total: \\$${amount} @common.currency for ${name}"
}
```

`amount` and `name` resolve from binders. `@common.currency` resolves from another translation. The leading `\\$` writes a literal `$` directly before the `${amount}` interpolation, so the rendered output starts with `Total: $` followed by the bound amount.

## See also

- [How to bind typed variables to translations](variable-binding.md) for `StringVar`, `IntVar`, `MoneyVar`, and chaining.
- [How to pluralise translations](pluralisation.md) for Common Locale Data Repository (CLDR) plural forms.
- [I18n API reference](../../reference/i18n-api.md) for `T()`, `LT()`, and the `Translation` surface.
