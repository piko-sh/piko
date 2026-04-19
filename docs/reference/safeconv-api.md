---
title: Safeconv API
description: Saturating integer type conversions that clamp instead of overflowing.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 290
---

# Safeconv API

The `piko.sh/piko/wdk/safeconv` package provides integer type conversions that clamp (saturate) to the destination type's range instead of silently wrapping. Negative values become zero for unsigned destinations. Values above the maximum become the maximum. Values below the minimum become the minimum. The package exists so `gosec G115` analysis passes without sprinkling `if` branches at every narrowing conversion.

## Saturating conversions

Each function has the form `FromTo(n From) To` and returns a clamped value. None panic and none allocate.

### From `int`

| Function | Target |
|---|---|
| `IntToUint64(n int) uint64` | `uint64` |
| `IntToUint32(n int) uint32` | `uint32` |
| `IntToUint16(n int) uint16` | `uint16` |
| `IntToUint8(n int) uint8` | `uint8` |
| `IntToInt32(n int) int32` | `int32` |
| `IntToInt16(n int) int16` | `int16` |
| `IntToInt8(n int) int8` | `int8` |
| `IntToUintptr(n int) uintptr` | `uintptr` |

### From `int64`

| Function | Target |
|---|---|
| `Int64ToInt32(n int64) int32` | `int32` |
| `Int64ToInt16(n int64) int16` | `int16` |
| `Int64ToInt(n int64) int` | `int` |
| `Int64ToUint64(n int64) uint64` | `uint64` |
| `Int64ToUint32(n int64) uint32` | `uint32` |
| `Int64ToUint16(n int64) uint16` | `uint16` |

### From `int32`

| Function | Target |
|---|---|
| `Int32ToInt64(n int32) int64` | `int64` |
| `Int32ToInt(n int32) int` | `int` |
| `Int32ToUint32(n int32) uint32` | `uint32` |

### From `int16`

| Function | Target |
|---|---|
| `Int16ToUint16(n int16) uint16` | `uint16` |
| `Int16ToByte(n int16) byte` | `byte` |

### From `uint64`

| Function | Target |
|---|---|
| `Uint64ToInt64(n uint64) int64` | `int64` |
| `Uint64ToInt(n uint64) int` | `int` |
| `Uint64ToUint32(n uint64) uint32` | `uint32` |
| `Uint64ToUint16(n uint64) uint16` | `uint16` |
| `Uint64ToUint8(n uint64) uint8` | `uint8` |

### From `uint32`

| Function | Target |
|---|---|
| `Uint32ToInt64(n uint32) int64` | `int64` |
| `Uint32ToInt16(n uint32) int16` | `int16` |
| `Uint32ToByte(n uint32) byte` | `byte` |

### From `uint16`

| Function | Target |
|---|---|
| `Uint16ToInt16(n uint16) int16` | `int16` |

### From `rune`

| Function | Target |
|---|---|
| `RuneToUint16(r rune) uint16` | `uint16` |
| `RuneToByte(r rune) byte` | `byte` |

## Must-variants (panic on overflow)

Use only when overflow is genuinely impossible at the call site (and you want a fast crash if that assumption ever breaks).

| Function | Behaviour |
|---|---|
| `MustIntToUint8(n int) uint8` | Panics if `n < 0` or `n > 255`. |
| `MustIntToUint16(n int) uint16` | Panics on overflow. |
| `MustIntToInt16(n int) int16` | Panics on overflow. |
| `MustUintToUint8(n uint) uint8` | Panics on overflow. |
| `MustUint8ToInt8(n uint8) int8` | Panics if `n > 127`. |
| `MustInt8ToUint8(n int8) uint8` | Panics if `n < 0`. |

## Generic conversion

```go
type integer interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 |
    ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

func ToUint64[T integer](n T) uint64
```

Saturating cast from any integer-kind type to `uint64`. Prefer the typed functions above where the source type is statically known. The generic exists for libraries that accept `any` integer-kind input.

## Semantics

- Negative to unsigned: returns `0`.
- Above destination max: returns destination max.
- Below destination min (signed targets): returns destination min.
- Never panics (except the `Must*` variants), never truncates silently, never allocates.

## See also

- Source: [`wdk/safeconv/safeconv.go`](https://github.com/piko-sh/piko/blob/master/wdk/safeconv/safeconv.go).
