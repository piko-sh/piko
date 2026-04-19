---
title: About the maths package
description: Why Piko ships BigInt, Decimal, and Money types, IEEE-754 pitfalls, error-chaining trade-offs, and fair-rounding rationale.
nav:
  sidebar:
    section: "explanation"
    subsection: "utilities"
    order: 70
---

# About the maths package

Go's numeric types are not safe for money. `float64` loses precision on the everyday additions a billing system performs. `int64` is safe but inconvenient (every amount lives in minor units, and currency conversion becomes awkward). Third-party libraries exist but have uneven ergonomics. Piko ships its own `BigInt`, `Decimal`, and `Money` types to fill the gap.

This page explains why. The details matter because the decisions show up in everyday code. The decisions cover when to reach for which type, how the error-chaining API works, and what `Allocate` solves.

## The IEEE-754 problem in one line

`0.1 + 0.2` in Go's `float64` is not `0.3`. It is `0.30000000000000004`. The difference is small and invisible on most inputs, and most code does not care. Financial code does care. A VAT calculation that drops a hundredth of a pence per transaction drifts over millions of transactions into a meaningful amount. The regulator does not find "IEEE-754 rounding" a persuasive excuse.

`float64` is not fixable for money. Even 80-bit long doubles have the same problem. They have more precision but still lose it on some inputs. The only fix is to use a number format that represents decimal fractions exactly. That is a Decimal.

Piko's `Decimal` uses 34 digits of precision backed by the `apd` library. Every operation is exact within that precision. Rounding is explicit and chosen by the caller. `Decimal.Round(places)` applies banker's rounding (round-half-to-even), and the separate `Decimal.Floor()`, `Decimal.Ceil()`, and `Decimal.Truncate()` methods cover the other directional choices. `Money.RoundTo(places, mode)` exposes a wider set of rounding modes (including half-up and half-down) for currency amounts. There is no hidden IEEE-754 drift.

## Why three types, not one

If `Decimal` is exact, why ship `BigInt` and `Money` as well?

`BigInt` exists because some values are genuinely integers and the operations reflect that. A loyalty-point balance is an integer. A share count is an integer. A quantity is an integer. Using `Decimal` for these is legal but awkward: comparisons against integer literals require `NewDecimalFromInt`, and the reader has to squint to see that the value has no fractional part.

`Money` exists because an amount without a currency is a dangerous value. Adding 100 USD to 100 GBP does not mean 200. Every `Money` operation validates that currencies match, and conversions require an explicit converter. A `Money(100, "GBP")` cannot accidentally become 100 USD through a missed conversion.

The three types interoperate. A `Decimal` can multiply a `BigInt`. A `Money` can add another `Money` of the same currency. A `Money` can accept a `Decimal` or `BigInt` scalar for multiplication (for example, a tax rate applied to a price). The methods enforce the invariants of each type while letting the composition flow.

## Immutability and fluent chains

Every operation returns a new value. A fluent chain reads naturally:

```go
total := subtotal.Multiply(quantity).AddPercent(vatPercent).Round(2)
```

Immutability means a caller can pass a value to another function without worrying about mutation. It also means every chain allocates more than an in-place alternative would. For hot paths where allocation pressure matters, the `*InPlace` variants exist and mutate the receiver. The default is the immutable chain because it composes more predictably.

## Error chaining instead of return tuples

Go code typically returns `(value, error)` and checks the error at each step. A fluent numeric chain with tuple returns is painful:

```go
a, err := maths.NewDecimalFromString(s)
if err != nil { return err }
b, err := a.Multiply(quantity)
if err != nil { return err }
c, err := b.AddPercent(vatPercent)
if err != nil { return err }
total, err := c.Round(2)
if err != nil { return err }
```

Piko's types chain the error internally. A failure at any step short-circuits the rest of the chain but preserves the error for a single check at the end:

```go
total := maths.NewDecimalFromString(s).
    Multiply(quantity).
    AddPercent(vatPercent).
    Round(2)

if err := total.Err(); err != nil { return err }
```

The trade-off is that the error sits on the value, not in the call signature. A caller who forgets to call `Err()` does not see the error. Linting the call sites for `Err()` after fluent chains is one mitigation. The pattern also includes `Must()` for init-time or test code where a panic on error is acceptable.

The `CheckIs*` and `MustIs*` variants extend the pattern to predicate methods. `IsZero()` returns a bool that is unsafe to trust blindly (a chain error would silently return false). `CheckIsZero()` returns `(bool, error)` for callers that want the safety check. `MustIsZero()` panics on chain error.

## Allocate for fair rounding

Splitting 100 pence across three line items in ratios 1:1:1 is a trap. `100 / 3 = 33.33`, but three times 33.33 is 99.99. Sum the parts and a penny goes missing. The customer sees an invoice whose line items add to one penny less than the total. Not a good look.

`BigInt.Allocate(ratios...)` solves this by guaranteeing the parts sum to the input. The implementation distributes the integer division first, then assigns the remainder one penny at a time across the ratios by a consistent rule. For the 1:1:1 split of 100, `Allocate` produces `[34, 33, 33]`. Sum: 100.

The rule for "which item gets the extra penny" is deterministic so two runs on the same input produce the same output. This matters for reproducibility. A re-run of a nightly invoice job must not produce different line-item amounts.

## Currency conversion: single-base vs matrix

Single-base conversions define every rate relative to one base currency (typically USD or EUR). A GBP-to-JPY conversion first converts GBP to USD, then USD to JPY. Each conversion applies its own rounding. For most retail applications this is fine.

For applications that care about cross-rate precision (foreign-exchange systems, multi-currency accounting), routing through a base introduces error. A direct GBP-to-JPY rate is more accurate than the two-step chain.

Piko offers both. `NewConverter` takes a single base and cross-converts through it. `NewMatrixConverter` takes a full currency matrix and converts directly between any pair. Applications pick based on their precision requirements.

## Why `MinorInt` matters

`NewMoneyFromString("29.99", "GBP")` works, but it involves parsing a string. `NewMoneyFromMinorInt(2999, "GBP")` skips the parse and stores the value directly as 2999 pence. Databases typically store currency as a minor-unit integer for exactly this reason (no fractional drift, no locale ambiguity). The `MinorInt` constructor is the natural entry point from a database column.

`NewMoneyFromFloat` exists but reach for it rarely. It accepts the precision cost of float-to-decimal parsing at the boundary. This cost is usually fine when the input is a human-typed amount but not fine when the float came from a database-stored fraction.

## The performance question

Exact decimal arithmetic is slower than `float64`. Each `Add` allocates. Each `Round` allocates. For a VAT calculation in an order path, the cost is invisible. For a Monte Carlo simulation over a million decimals, the cost matters.

Two mitigations live in the package. The `*InPlace` methods mutate the receiver and skip the allocation. The aggregate helpers (`SumDecimals`, `AverageDecimals`) reuse intermediate values.

For true numerical computing (statistics, ML preprocessing), reach for `float64` or a specialised library. Piko's maths package targets money and counts, not scientific computing.

## See also

- [Maths API reference](../reference/maths-api.md) for the full method surface.
- [How to maths](../how-to/maths.md) for tax, invoice splitting, and multi-currency recipes.
- [`wdk/maths/doc.go`](https://github.com/piko-sh/piko/blob/master/wdk/maths/doc.go) for the precision and allocation rationale in source form.
