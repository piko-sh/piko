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

Piko's maths package provides three numeric types for applications that cannot tolerate IEEE-754 rounding. `BigInt` handles arbitrary-precision integers, `Decimal` handles 34-digit fixed-point numbers, and `Money` adds currency awareness on top of `Decimal`. All three types are immutable. Fluent operations return new values. Errors propagate through the chain, so a single `Err()` check at the end of a fluent call is enough. For the rationale see [about maths](../explanation/about-maths.md). For task recipes see [how to maths](../how-to/maths.md). Source: [`wdk/maths/`](https://github.com/piko-sh/piko/tree/master/wdk/maths).

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

`NewMoneyFromMinorInt` accepts minor units (pence for GBP, cents for USD). `RegisterCurrency(code, decimalPlaces)` registers a currency outside the built-in ISO 4217 list.

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
| `Money` | `AddDecimal`, `AddBigInt`, `SubtractDecimal`, `SubtractBigInt`, `MultiplyBigInt`, `DivideBigInt` (plus `*Int`, `*Float`, `*String`, `*MinorInt` for scalars; `Multiply` and `Divide` are scalar-only, mixing currencies raises an error) |

## Logic and comparison

| Method | Purpose |
|---|---|
| `Cmp(other)` | Classic three-way compare. Returns `-1`, `0`, or `1`. |
| `Equals`, `LessThan`, `GreaterThan`, `LessThanOrEqual`, `GreaterThanOrEqual` | Boolean comparisons. `*Int` / `*String` / `*Float` variants exist on the types where cross-type comparison makes sense. |
| `IsZero`, `IsPositive`, `IsNegative` | Sign predicates. |
| `IsInteger` (`Decimal` only) | Checks whether the decimal has no fractional part. |
| `IsBetween(min, max)` | Inclusive range check. |
| `IsCloseTo(other, epsilon)` | Epsilon-based equality useful for floating-point bridges. |
| `IsEven`, `IsOdd`, `IsMultipleOf(divisor)` | Integer-style predicates (available on `Decimal` and `Money` for whole values). |

Every predicate has a `CheckIs*` variant that returns `(bool, error)` so chain errors surface, and a `MustIs*` variant that panics on chain error.

## Fluent transformations

Returns a new value of the same type. The chain retains any prior error state.

| Method | Purpose |
|---|---|
| `Abs()` | Absolute value. |
| `Negate()` | Sign flip. |
| `Round(places)` | Round to `places` decimal places using banker's rounding. |
| `Ceil()`, `Floor()`, `Truncate()` | Rounding modes. |
| `AddPercent(p)`, `SubtractPercent(p)`, `GetPercent(p)` | Percentage helpers. `GetPercent` returns `value * p / 100`. Variants: `*Int`, `*String`, `*Float`. |

## Aggregates

Aggregates take either a variadic list or a slice. Each type ships matching helpers.

```go
SumBigInts(...BigInt)    AverageBigInts(...BigInt)    MinBigInt(b1, ...others)   MaxBigInt(b1, ...others)
AbsSumBigInts(...BigInt) SortBigInts(slice)           SortBigIntsReverse(slice)
Allocate(ratios ...BigInt) // distributes a BigInt across ratios with fair rounding

SumDecimals(...Decimal)  AverageDecimals(...Decimal)  MinDecimal(d1, ...others)  MaxDecimal(d1, ...others)
AbsSumDecimals(...Decimal) SortDecimals(slice)        SortDecimalsReverse(slice)

SumMoneys(...Money)      AverageMoneys(...Money)      MinMoney(m1, ...others)    MaxMoney(m1, ...others)
AbsSumMoneys(...Money)   SortMoneys(slice)            SortMoneysReverse(slice)
```

`Allocate` splits a value across a set of ratios while ensuring the parts sum to the original. This is the right primitive for dividing an invoice total across line items without rounding drift.

## In-place mutations

Useful inside tight loops where allocation pressure matters. The `*InPlace` methods mutate the receiver instead of returning a new value.

```go
AddInPlace(other)
SubtractInPlace(other)
MultiplyInPlace(other)
```

Available on `BigInt`, `Decimal`, and `Money`. Use sparingly. Fluent chains are the default.

## Conversion and extraction

| Method | Purpose |
|---|---|
| `BigInt.ToDecimal()` | Widens a `BigInt` to a `Decimal`. |
| `Decimal.ToBigInt()` | Narrows a `Decimal` to a `BigInt`, truncating the fractional part. |
| `Money.Amount() (Decimal, error)` | Extracts the `Decimal` amount. |
| `Money.CurrencyCode() string` | Returns the ISO 4217 (or custom-registered) currency code. |
| `String()` | Canonical string form. |
| `Err() error` | Returns any error accumulated through the fluent chain. |
| `Must()` | Returns the value if no error, panics otherwise. Use in init code or tests. |

## Currency conversion

```go
func NewExchangeRates(baseCurrency string, rates map[string]Decimal) (ExchangeRates, error)
func NewConverter(rates ExchangeRates) *Converter
func (c *Converter) Convert(source Money, targetCode string) Money
```

`Converter` handles single-base conversions. For cross-rate accuracy (EUR to JPY without routing through USD), use the matrix form:

```go
func NewMatrixConverter(matrix RateMatrix) *MatrixConverter
func (c *MatrixConverter) Convert(source Money, targetCode string) Money
func (c *MatrixConverter) ConvertAll(sources []Money, targetCode string) ([]Money, error)
func (c *MatrixConverter) Supports(code string) bool
func (c *MatrixConverter) CanConvert(from, to string) bool
```

## Number formatting

```go
func NumberLocale(locale string) NumberLocale
func GetNumberLocale(locale string) NumberLocale
func FormatNumberString(s string, locale NumberLocale) string
```

`Money` additionally exposes `RoundedDefaultFormat()` which returns a locale-aware string suitable for display.

## Currency registration

```go
func RegisterCurrency(code string, definition CurrencyDefinition)
```

Registers a non-standard currency (cryptocurrency, test currency, legacy unit). Call once at bootstrap.

## Error propagation

Every `BigInt`, `Decimal`, and `Money` tracks an optional error. If any operation in a fluent chain fails, later operations become no-ops and the error remains accessible through `Err()`. Check once at the end instead of after every step.

```go
total := maths.NewDecimalFromString("19.99").
    Multiply(quantity).
    AddPercent(taxPercent).
    Round(2)

if err := total.Err(); err != nil {
    return err
}
```

The `Check*` method family returns `(bool, error)` so predicates can surface a chain error. The `Must*` family panics instead, suitable for initialisation and tests.

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
