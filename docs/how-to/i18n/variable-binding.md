---
title: How to bind typed variables to translations
description: Bind strings, integers, floats, decimals, money, and big integers to translations with the typed Translation API and method chaining.
nav:
  sidebar:
    section: "how-to"
    subsection: "i18n"
    order: 740
---

# How to bind typed variables to translations

The `Translation` value returned from `T()` exposes typed setters for every value kind the i18n layer renders. Use them instead of pre-formatting in Go so the locale controls grouping separators, decimal points, and currency symbols. For `StringVar` and `IntVar` basics see [i18n API reference](../../reference/i18n-api.md).

## Bind numeric variables

```go
T("count").IntVar("n", itemCount)
T("price").FloatVar("amount", 19.99)
```

`IntVar` and `FloatVar` cover the cases where a Go primitive maps cleanly onto the locale's numeric format.

## Bind high-precision decimals

For amounts that must round-trip exactly (financial calculations, billing), use `DecimalVar`:

```go
import "piko.sh/piko/wdk/maths"

price := maths.NewDecimalFromString("99.99")
T("total").DecimalVar("amount", price)
```

The decimal type avoids the binary floating-point error that bites `FloatVar` on values like `0.1 + 0.2`.

## Bind money with locale-aware currency formatting

```go
import "piko.sh/piko/wdk/maths"

price := maths.NewMoneyFromString("49.99", "GBP")
T("price.formatted").MoneyVar("price", price)
```

`NewMoneyFromString` takes the amount first and the ISO 4217 currency code second. Other constructors include `NewMoneyFromDecimal`, `NewMoneyFromInt`, `NewMoneyFromMinorInt`, and `NewMoneyFromFloat`; see the [maths API reference](../../reference/maths-api.md).

`MoneyVar` formats the value according to the active locale. That covers currency symbol position, decimal separator, thousand grouping, and the symbol or ISO code. The same translation key renders `£49.99` in `en-GB` and `49,99 £` in `fr-FR`.

## Bind big integers

For values that exceed `int64`:

```go
import "piko.sh/piko/wdk/maths"

large := maths.NewBigIntFromString("123456789012345678901234567890")
T("bignum").BigIntVar("value", large)
```

`BigIntVar` accepts arbitrary-precision integers and applies locale-aware grouping.

## Chain binders fluently

Every typed setter returns the `Translation` so calls chain into a single expression:

```go
result := T("summary").
    StringVar("name", user.Name).
    IntVar("items", cart.Count).
    MoneyVar("total", cart.Total).
    String()
```

`String()` materialises the final rendered text and releases the builder back to the pool. Without `String()`, the chain returns the unrendered `*Translation`, which renders implicitly in templates through `fmt.Stringer`. Inside Go where you need a `string` (for example, `Metadata.Title`), call `String()` explicitly.

## Provide a fallback for missing keys

If the key is missing from every store, the second argument to `T` becomes the rendered string:

```go
T("missing.key", "Default text").String()
```

Useful for new translations that have not yet shipped to every locale file.

## See also

- [I18n API reference](../../reference/i18n-api.md) for the full `Translation` surface and translation lookup order.
- [How to format dates, times, and currency for a locale](date-time-formatting.md) for `TimeVar`, `DateTimeVar`, and the four formatting styles.
- [How to interpolate variables and reference other keys in translations](template-syntax.md) for the `${expression}` and `@key.path` constructs that bound variables fill.
- [Maths API reference](../../reference/maths-api.md) for `Money`, `Decimal`, and `BigInt`.
