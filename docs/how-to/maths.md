---
title: How to do financial arithmetic with Decimal and Money
description: Tax calculations, invoice splitting, multi-currency conversion, and the error-propagation pattern.
nav:
  sidebar:
    section: "how-to"
    subsection: "utilities"
    order: 30
---

# Financial arithmetic with decimal and money types

This guide covers the common money-handling recipes. For the full method surface see [maths API reference](../reference/maths-api.md). For why Piko ships BigInt, Decimal, and Money types instead of `float64` see [about maths](../explanation/about-maths.md).

## Compute a total with VAT

Never multiply money values with `float64`. Use `Decimal`:

```go
import "piko.sh/piko/wdk/maths"

unitPrice := maths.NewDecimalFromString("19.99")
quantity  := maths.NewDecimalFromInt(3)
vatPercent := maths.NewDecimalFromString("20")

subtotal := unitPrice.Multiply(quantity)
total    := subtotal.AddPercent(vatPercent).Round(2)

if err := total.Err(); err != nil {
    return err
}

// total.MustString() == "71.96"
```

`AddPercent` is shorthand for `Multiply(1 + percent/100)`. `Round(2)` uses banker's rounding to two decimal places. The error check lives at the end of the chain. Any error earlier in the pipeline propagates through to `total.Err()`.

## Split an invoice without losing a penny

`BigInt.Allocate` distributes a value across ratios while making sure the parts sum to the original. This is the right primitive when the caller has to divide an integer number of pennies unevenly.

```go
totalMinor := maths.NewBigIntFromInt(10000) // £100.00 in pence

// Split 1:2:1 between three line items. Allocate takes int64 weights.
parts, err := totalMinor.Allocate(1, 2, 1)
if err != nil {
    return err
}

// parts == [2500, 5000, 2500] (pence)
```

A naive `Divide(2500, 5000, 2500)` risks losing a penny to rounding. `Allocate` guarantees the parts sum to the input.

## Work in minor units

Store currency in minor units (pence, cents) to avoid fractional math entirely:

```go
cart := maths.NewMoneyFromMinorInt(2499, "GBP")   // £24.99
shipping := maths.NewMoneyFromMinorInt(499, "GBP") // £4.99

total := cart.Add(shipping)
if err := total.Err(); err != nil {
    return err
}

// total == £29.98
```

`Add` refuses to mix currency codes at runtime. Mixing GBP and USD returns an error instead of a silently wrong value.

## Convert between currencies

Single-base conversions use `NewConverter`:

```go
rates, err := maths.NewExchangeRates("USD", map[string]maths.Decimal{
    "GBP": maths.NewDecimalFromString("0.79"),
    "EUR": maths.NewDecimalFromString("0.92"),
    "JPY": maths.NewDecimalFromString("149.5"),
})
if err != nil {
    return err
}

converter := maths.NewConverter(rates)

priceUSD := maths.NewMoneyFromString("29.99", "USD")
priceGBP := converter.Convert(priceUSD, "GBP")
```

All conversions route through the base currency (`USD` above). For cross-rate accuracy (EUR to JPY without rounding twice), use the matrix form:

```go
matrix := maths.NewMatrixConverter(maths.RateMatrix{
    BaseCurrency: "USD",
    Rates: map[string]map[string]maths.Decimal{
        "EUR": {"JPY": maths.NewDecimalFromString("162.5")},
        "JPY": {"EUR": maths.NewDecimalFromString("0.00615")},
    },
})

if matrix.CanConvert("EUR", "JPY") {
    priceJPY := matrix.Convert(priceEUR, "JPY")
}
```

The matrix form stores direct pair rates for every currency pair, and the conversion does not pass through an intermediate base.

## Sort and aggregate

Aggregate helpers exist for each type:

```go
prices := []maths.Decimal{
    maths.NewDecimalFromString("19.99"),
    maths.NewDecimalFromString("29.99"),
    maths.NewDecimalFromString("9.99"),
}

subtotal := maths.SumDecimals(prices...)
avg      := maths.AverageDecimals(prices...)
cheapest := maths.MinDecimal(prices[0], prices[1:]...)

if err := maths.SortDecimalsReverse(prices); err != nil { // in-place descending sort
    return err
}
```

Equivalents for `BigInt` (`SumBigInts`, `SortBigInts`, and so on) and `Money` (`SumMoneys`, `SortMoneys`, and so on) follow the same shape.

## Use the error-propagation pattern

Every type carries an optional error through fluent chains. If any step fails (parse error, currency mismatch, division by zero), the chain short-circuits and preserves the error:

```go
result := maths.NewDecimalFromString("not a number").
    Add(maths.OneDecimal()).
    Multiply(maths.TenDecimal()).
    Round(2)

if err := result.Err(); err != nil {
    return err // reports the initial parse error
}
```

This removes the need for `if err != nil` after every step. Check once at the end.

For hot paths where allocation pressure matters, use the in-place variants (`AddInPlace`, `SubtractInPlace`, `MultiplyInPlace`). They mutate the receiver and skip the per-step allocation, at the cost of being harder to reason about. The fluent chain is the default.

## Round for display

Rounding modes:

| Method | Behaviour |
|---|---|
| `Round(places)` | Banker's rounding. Half-to-even. |
| `Ceil()` | Up. |
| `Floor()` | Down. |
| `Truncate()` | Towards zero. |

For currency display:

```go
label := total.Round(2).MustString()
```

`Decimal.String()` returns `(string, error)`. Use `MustString()` when the chain has already been error-checked. `Money` also offers `RoundedDefaultFormat()` which applies locale-aware thousands separators and the currency symbol.

## Register a non-standard currency

```go
maths.RegisterCurrency("XTK", maths.CurrencyDefinition{
    NumericCode:   "999",
    Digits:        8,
    DefaultSymbol: "XTK",
})

balance := maths.NewMoneyFromString("0.12345678", "XTK")
```

`CurrencyDefinition` is the upstream `bojanz/currency.Definition` shape. `NumericCode` is an ISO-4217-style three-digit identifier. Use a private-use value such as `"999"` for non-standard codes. `Digits` is the number of fractional digits. `DefaultSymbol` overrides the rendered symbol across locales. `RegisterCurrency` does not return an error. It overwrites any prior registration for the same code.

Useful for cryptocurrencies, test currencies, or legacy internal units. Register once at bootstrap.

## See also

- [Maths API reference](../reference/maths-api.md) for the complete method surface.
- [About maths](../explanation/about-maths.md) for IEEE-754 pitfalls and the immutability/error-chaining trade-offs.
- [`wdk/maths/doc.go`](https://github.com/piko-sh/piko/blob/master/wdk/maths/doc.go) for the precision rationale in source form.
