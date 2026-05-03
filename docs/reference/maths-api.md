---
title: Maths API
description: Arbitrary-precision BigInt, Decimal, and Money with fluent operations and error propagation.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 260
---

# Maths API

Piko's maths package provides three numeric types for applications that cannot tolerate IEEE-754 rounding. `BigInt` handles arbitrary-precision integers, `Decimal` handles 34-digit fixed-point numbers, and `Money` adds currency awareness on top of `Decimal`. All three types are immutable. Fluent operations return new values. Each value carries an optional error that propagates through the chain and is observable through `Err()`. For the rationale see [about maths](../explanation/about-maths.md). For task recipes see [how to maths](../how-to/maths.md). Source: [`wdk/maths/`](https://github.com/piko-sh/piko/tree/master/wdk/maths).

## Constructors

### BigInt

```go
func NewBigIntFromString(s string) BigInt
func NewBigIntFromInt(value int64) BigInt
func NewBigIntFromApd(value apd.BigInt) BigInt
func ZeroBigInt() BigInt
func OneBigInt() BigInt
func TenBigInt() BigInt
func HundredBigInt() BigInt
func ZeroBigIntWithError(err error) BigInt
```

### Decimal

```go
func NewDecimalFromString(s string) Decimal
func NewDecimalFromInt(value int64) Decimal
func NewDecimalFromFloat(value float64) Decimal
func NewDecimalFromApd(value apd.Decimal) Decimal
func ZeroDecimal() Decimal
func OneDecimal() Decimal
func TenDecimal() Decimal
func HundredDecimal() Decimal
func ZeroDecimalWithError(err error) Decimal
```

### Money

```go
func NewMoneyFromDecimal(amount Decimal, code string) Money
func NewMoneyFromString(amount string, code string) Money
func NewMoneyFromInt(amount int64, code string) Money
func NewMoneyFromMinorInt(amount int64, code string) Money
func NewMoneyFromFloat(amount float64, code string) Money
func ZeroMoney(code string) Money
func OneMoney(code string) Money
func HundredMoney(code string) Money
func ZeroMoneyWithError(code string, err error) Money
```

`NewMoneyFromMinorInt` accepts minor units (pence for GBP, cents for USD). `RegisterCurrency(code string, definition CurrencyDefinition)` registers a currency outside the built-in ISO 4217 list.

`CurrencyDefinition` is a type alias for [`github.com/bojanz/currency.Definition`](https://pkg.go.dev/github.com/bojanz/currency#Definition):

| Field | Type | Purpose |
|---|---|---|
| `NumericCode` | `string` | The three-digit ISO 4217 numeric code (for example `"999"`). |
| `Digits` | `uint8` | The number of fraction digits used by the currency (for example `2` for GBP). |
| `DefaultSymbol` | `string` | The default symbol used across all locales. Leave empty when overriding an existing currency to retain the built-in locale-specific symbols. |

## Arithmetic

Each type exposes the same family of arithmetic operations. The `Int`, `String`, and `Float` variants accept the operand as that scalar type. The base form takes the same type as the receiver.

| Method | Also available as |
|---|---|
| `Add`, `Subtract`, `Multiply`, `Divide` | `*Int`, `*String`, `*Float` |
| `Modulus`, `Remainder` | `*Int`, `*String`, `*Float` |
| `Power` | `*Int`, `*String`, `*Float` |

### Cross-type arithmetic

`BigInt` and `Decimal` can operate on each other directly. Money operations accept `Decimal` amounts through `*Decimal` variants.

| Receiver | Available cross-type variants |
|---|---|
| `BigInt` | `AddDecimal`, `AddFloat`, `SubtractDecimal`, `SubtractFloat`, `MultiplyDecimal`, `MultiplyFloat`, `DivideDecimal`, `DivideFloat`, `PowerDecimal`, `PowerFloat` |
| `Decimal` | `AddDecimal`, `AddBigInt`, `SubtractDecimal`, `SubtractBigInt`, `MultiplyDecimal`, `MultiplyBigInt`, `DivideDecimal`, `DivideBigInt` (plus `Modulus*`, `Remainder*`, `Power*` variants) |
| `Money` | `AddDecimal`, `AddBigInt`, `SubtractDecimal`, `SubtractBigInt`, `MultiplyBigInt`, `DivideBigInt` (plus `*Int`, `*Float`, `*String`, `*MinorInt` for scalars; `Multiply(factor Decimal)` and `Divide(factor Decimal)` take a unitless `Decimal` factor; mixing currencies in addition or subtraction raises an error) |

## Logic and comparison

| Method | Purpose |
|---|---|
| `BigInt.Cmp(other) (int, error)` | Three-way compare for `BigInt` only. Returns `-1`, `0`, or `1` plus a chain error. `Decimal` and `Money` do not expose `Cmp`; use the `Check*` predicates below. |
| `Equals`, `LessThan`, `GreaterThan` | Boolean comparisons returning `(bool, error)`. Available on `BigInt`, `Decimal`, and `Money`. `*Int` / `*String` / `*Float` variants exist where cross-type comparison makes sense. |
| `LessThanOrEqual`, `GreaterThanOrEqual` | Available on `Decimal` only. Returns `(bool, error)`. `BigInt` and `Money` do not expose the `OrEqual` pair. |
| `CheckEquals`, `CheckLessThan`, `CheckGreaterThan` | Plain `bool` form on all three types. A chain error folds to `false`. |
| `MustEquals`, `MustLessThan`, `MustGreaterThan` | Plain `bool` form. Panics if the chain carries an error. |
| `IsZero`, `IsPositive`, `IsNegative` | Sign predicates. |
| `IsInteger` | Checks whether the value has no fractional part. Available on `BigInt`, `Decimal`, and `Money`. |
| `IsBetween(min, max)` | Inclusive range check. |
| `IsCloseTo(target, tolerance)` | Tolerance-based equality: returns `true` when `|self - target| <= tolerance`. Decimal signature `IsCloseTo(target, tolerance Decimal)`; Money signature `IsCloseTo(target, tolerance Money)`. Available on `Decimal` and `Money`. |
| `IsEven`, `IsOdd`, `IsMultipleOf(divisor)` | Integer-style predicates (available on `BigInt`, `Decimal`, and `Money` for whole values). |

The `Check*` family returns plain `bool` with chain errors folded to `false`. The bare-named methods return `(bool, error)`. The `Must*` family panics on chain error.

## Fluent transformations

Returns a new value of the same type. The chain retains any prior error state.

| Method | Purpose |
|---|---|
| `Abs()` | Absolute value. |
| `Negate()` | Sign flip. |
| `Decimal.Round(places int32)` | Round to `places` decimal places using banker's rounding. |
| `Money.RoundToStandard()`, `Money.RoundTo(places uint8, mode currency.RoundingMode)` | Round to the currency's standard precision, or to a chosen precision and rounding mode. |
| `Ceil()`, `Floor()`, `Truncate()` | Rounding modes. |
| `AddPercent(p)`, `SubtractPercent(p)`, `GetPercent(p)` | Percentage helpers. `GetPercent` returns `value * p / 100`. Variants: `*Int`, `*String`, `*Float`. |

## Aggregates

Aggregates take either a variadic list or a slice. Each type ships matching helpers.

```go
SumBigInts(...BigInt) BigInt           AverageBigInts(...BigInt) Decimal     MinBigInt(b1, ...others) BigInt
MaxBigInt(b1, ...others) BigInt        AbsSumBigInts(...BigInt) BigInt       SortBigInts(slice)
SortBigIntsReverse(slice)
// b.Allocate(ratios ...int64) ([]BigInt, error): method on BigInt; distributes b across ratios with fair rounding

SumDecimals(...Decimal) Decimal        AverageDecimals(...Decimal) Decimal   MinDecimal(d1, ...others) Decimal
MaxDecimal(d1, ...others) Decimal      AbsSumDecimals(...Decimal) Decimal    SortDecimals(slice)
SortDecimalsReverse(slice)

SumMoney(...Money) Money               AverageMoney(...Money) Money          MinMoney(m1, ...others) Money
MaxMoney(m1, ...others) Money          AbsSumMoney(...Money) Money           SortMoney(slice)
SortMoneyReverse(slice)
```

Note: `AverageBigInts` returns a `Decimal` (not a `BigInt`) because the average of integers can have a fractional part.

`Allocate` is a method on each numeric type (`BigInt.Allocate`, `Decimal.Allocate`, `Money.Allocate`) that splits the receiver across a set of integer ratios while ensuring the parts sum to the original. This is the right primitive for dividing an invoice total across line items without rounding drift.

## In-place mutations

The `*InPlace` methods mutate the receiver instead of returning a new value. They are available on `BigInt`, `Decimal`, and `Money`.

```go
AddInPlace(other)
SubtractInPlace(other)
MultiplyInPlace(other)
```

## Conversion and extraction

| Method | Purpose |
|---|---|
| `BigInt.ToDecimal()` | Widens a `BigInt` to a `Decimal`. |
| `Decimal.ToBigInt()` | Narrows a `Decimal` to a `BigInt`, truncating the fractional part. |
| `Money.Amount() (Decimal, error)` | Extracts the `Decimal` amount. |
| `Money.CurrencyCode() (string, error)` | Returns the ISO 4217 (or custom-registered) currency code, propagating any chain error. |
| `String() (string, error)` | Canonical string form. Returns the chain error alongside the rendered text. |
| `MustString() string` | Canonical string form. Panics on chain error. |
| `Err() error` | Returns any error accumulated through the fluent chain. |
| `Must()` | Returns the value if no error; panics otherwise. |

## Currency conversion

```go
func NewExchangeRates(baseCurrency string, rates map[string]Decimal) (ExchangeRates, error)
func InvertRates(original ExchangeRates, newBaseCurrency string) (ExchangeRates, error)
func NewConverter(rates ExchangeRates) *Converter
func (c *Converter) Convert(source Money, targetCode string) Money
```

`Converter` handles single-base conversions. `InvertRates` rebuilds an `ExchangeRates` set against a new base currency without losing precision.

For cross-rate accuracy (EUR to JPY without routing through USD), use the matrix form. `RateMatrix` stores a full source-to-target rate map, so the converter handles any pair of supported currencies directly:

```go
type RateMatrix struct {
    Rates        map[string]map[string]Decimal
    BaseCurrency string
}

func NewRateMatrix(baseCurrency string, baseRates map[string]Decimal, overrides map[string]map[string]Decimal) (RateMatrix, error)
func NewMatrixConverter(matrix RateMatrix) *MatrixConverter
func (c *MatrixConverter) Convert(source Money, targetCode string) Money
func (c *MatrixConverter) ConvertAll(sources []Money, targetCode string) ([]Money, error)
func (c *MatrixConverter) Supports(code string) bool
func (c *MatrixConverter) CanConvert(from, to string) bool
```

`NewRateMatrix` initialises identity rates for every currency it sees, populates the matrix from `baseRates` (and their inverses), then applies any direct `overrides` so cross-rates can bypass routing through the base currency.

## Number formatting

```go
type NumberLocale struct { /* ... */ }
func GetNumberLocale(locale string) NumberLocale
func FormatNumberString(s string, locale NumberLocale) string
```

`NumberLocale` is a struct describing the formatting rules for a locale. `GetNumberLocale` is the constructor that resolves a locale tag to a populated `NumberLocale` value. Unrecognised locales fall back to `LocaleEnGB`.

### Decimal formatting

```go
func (d Decimal) Format(locale string) (string, error)
func (d Decimal) MustFormat(locale string) string
```

`Format` keeps full precision while applying the locale's separators. `MustFormat` panics on chain error.

### Money formatting

```go
func (m Money) RoundedNumber() (string, error)        // Currency-rounded amount, plain digits
func (m Money) FormattedNumber() (string, error)      // Default-locale grouped number
func (m Money) MustFormattedNumber() string           // Panic on error
func (m Money) Format(localeID string) (string, error)        // Currency string with symbol for the locale
func (m Money) MustFormat(localeID string) string             // Panic on error
func (m Money) DefaultFormat() string                         // Currency string in en-GB
func (m Money) RoundedDefaultFormat() string                  // Currency-rounded amount in en-GB
```

`Money.Format` emits a fully localised currency string (symbol position, separators, fraction digits) while `RoundedNumber` and `FormattedNumber` return digit-only renderings useful for embedding in custom layouts.

## Currency registration

```go
func RegisterCurrency(code string, definition CurrencyDefinition)
```

Registers a non-standard currency (cryptocurrency, test currency, legacy unit).

## Error propagation

Every `BigInt`, `Decimal`, and `Money` carries an optional error. When an operation in a fluent chain fails, subsequent operations become no-ops and the error is observable through `Err()`.

```go
total := maths.NewDecimalFromString("19.99").
    Multiply(quantity).
    AddPercent(taxPercent).
    Round(2)

if err := total.Err(); err != nil {
    return err
}
```

The `Check*` family of predicates folds chain errors to `false`. The `Must*` family panics on chain error.

## Constants

| Constant | Value |
|---|---|
| `Base10` | Common base for formatting. |
| `Value10` | Decimal literal `10`. |
| `Value100` | Decimal literal `100`. |

## See also

- [About maths](../explanation/about-maths.md) for IEEE-754 pitfalls and design rationale.
- [How to maths](../how-to/maths.md) for tax calculation, invoice splitting, and multi-currency recipes.
- [`doc.go`](https://github.com/piko-sh/piko/blob/master/wdk/maths/doc.go) for allocation and precision rationale in source form.
