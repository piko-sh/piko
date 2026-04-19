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

The safeconv package provides integer type conversions that clamp (saturate) to the destination type's range instead of silently wrapping. Negative values target zero for unsigned destinations. Values above the maximum target the maximum. Values below the minimum target the minimum. The package's purpose is to let `gosec G115` analysis pass without losing data to silent truncation or introducing `if` branches everywhere a narrowing conversion happens. Source of truth: [`wdk/safeconv/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/safeconv/facade.go).

## Typed conversions

```go
IntToUint64, IntToUint32, IntToUint16, IntToUint8
IntToInt32, IntToInt16, IntToInt8
Int64ToUint32, Int64ToUint16, Int64ToUint8
Int64ToInt32, Int64ToInt16, Int64ToInt8
Uint64ToInt64, Uint64ToInt32, Uint64ToInt16, Uint64ToInt8
Uint64ToUint32, Uint64ToUint16, Uint64ToUint8
Uint32ToInt32, Uint32ToInt16, Uint32ToInt8
Uint32ToUint16, Uint32ToUint8
Uint16ToInt16, Uint16ToInt8, Uint16ToUint8
Uint8ToInt8
```

Every function has the signature `FromTo(n From) To` and returns the clamped value.

## Generic conversions

```go
func ToUint64(n any) uint64
func ToInt64(n any) int64
func ToUint32(n any) uint32
func ToInt32(n any) int32
```

Accept any integer type via reflection, and clamp on overflow. Prefer the typed functions where the caller knows the source type.

## Semantics

- Negative to unsigned: returns `0`.
- Above destination max: returns destination max.
- Below destination min (signed targets): returns destination min.
- Never panics, never truncates silently, never allocates.

## See also

- [Binder API reference](binder-api.md) uses safeconv to bound deserialised numeric values.
