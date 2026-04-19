---
title: Encoding API
description: Base-36 / Base-58 codecs, the generic Base-X encoder, UUID v7 helpers, and short-UUID strings.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 270
---

# Encoding API

The `piko.sh/piko/wdk/encoding` package provides:

- A generic Base-X encoder (`*Encoding`) usable with any non-ambiguous alphabet.
- Convenience functions for Base-36 and Base-58 (the most common short-string formats).
- UUID v7 generators (sortable IDs).
- Short-UUID helpers that pack a UUID into a compact Base-58 string.

There are no `Base64()` / `Base32()` / `Hex()` factory functions. Go's standard library already covers those formats (`encoding/base64`, `encoding/base32`, `encoding/hex`). Use them directly.

## Base-X encoder

```go
type Encoding struct { /* ... */ }

func NewEncoding(alphabet string) (*Encoding, error)

func (enc *Encoding) EncodeBytes(data []byte) string
func (enc *Encoding) DecodeBytes(input string) ([]byte, error)
func (enc *Encoding) EncodeUint64(value uint64) string
func (enc *Encoding) DecodeUint64(input string) (uint64, error)
```

`NewEncoding` rejects empty alphabets and alphabets with repeated characters (`ErrAlphabetEmpty`, `ErrAlphabetAmbiguous`). The alphabet length implies the base.

## Alphabet constants

| Constant | Alphabet |
|---|---|
| `StdBase64Alphabet` | RFC 4648 standard Base-64 (`A-Z a-z 0-9 + /`). |
| `URLBase64Alphabet` | RFC 4648 URL-safe Base-64 (`A-Z a-z 0-9 - _`). |
| `StdHexAlphabetLower` | `0123456789abcdef`. |
| `StdHexAlphabetUpper` | `0123456789ABCDEF`. |
| `StdBase32Alphabet` | RFC 4648 Base-32 (`A-Z 2-7`). |
| `HexBase32Alphabet` | RFC 4648 extended-hex Base-32 (`0-9 A-V`). |

Callers that need a custom encoder constructed from a known alphabet can pass these directly (`NewEncoding(StdBase64Alphabet)`).

## Base-36

```go
func EncodeBytesBase36(data []byte) string
func DecodeBytesBase36(input string) ([]byte, error)
func EncodeUint64Base36(value uint64) string
func DecodeUint64Base36(input string) (uint64, error)
```

## Base-58

```go
func EncodeBytesBase58(data []byte) string
func DecodeBytesBase58(input string) ([]byte, error)
func EncodeUint64Base58(value uint64) string
func DecodeUint64Base58(input string) (uint64, error)
```

The Base-58 alphabet excludes the visually ambiguous `0`, `O`, `I`, and `l` so encoded strings are safe to read aloud or transcribe.

## UUID v7

UUID v7 values are sortable by creation time, which makes them cheap to index in a database.

```go
func NewV7At(t time.Time) (uuid.UUID, error)
func NewV7AtFromReader(r io.Reader, t time.Time) (uuid.UUID, error)
func NewV7MinAt(t time.Time) uuid.UUID
func NewV7MaxAt(t time.Time) uuid.UUID
func ResetLastV7timeAt()
```

| Function | Purpose |
|---|---|
| `NewV7At(t)` | A new v7 UUID stamped at `t`. Monotonic regarding a per-process clock so consecutive calls in the same millisecond still sort correctly. |
| `NewV7AtFromReader(r, t)` | Same, but reads randomness from `r` (useful in tests). |
| `NewV7MinAt(t)` | The lexicographically smallest UUID v7 with timestamp `t`. Useful as a range lower bound. |
| `NewV7MaxAt(t)` | The lexicographically largest UUID v7 with timestamp `t`. Useful as a range upper bound. |
| `ResetLastV7timeAt()` | Reset the per-process monotonic state. Tests only. |

## Short-UUID helpers

Pack a UUID into a compact human-friendly string. Three flavours:

```go
// Base64URL short string (RFC 4648 URL-safe, 22 chars).
func UUIDToShortString(id uuid.UUID) string
func ShortStringToUUID(s string) (uuid.UUID, error)
func MustShortStringToUUID(s string) uuid.UUID

// Base-58 short string (21-22 chars). Uses the unambiguous Base-58 alphabet.
func UUIDToBase58String(id uuid.UUID) string
func Base58StringToUUID(s string) (uuid.UUID, error)
func MustBase58StringToUUID(s string) uuid.UUID

// Even shorter (drops the variant/version bits and reconstructs them on decode).
// Requires you to specify which version when decoding. Output is 21 chars.
func UUIDToShorterString(id uuid.UUID) string
func ShorterStringToUUID(s string, version int) (uuid.UUID, error)
func ShorterStringToUUIDv1(s string) (uuid.UUID, error)
func ShorterStringToUUIDv4(s string) (uuid.UUID, error)
func ShorterStringToUUIDv7(s string) (uuid.UUID, error)
func MustShorterStringToUUID(s string, version int) uuid.UUID
```

`UUIDToShortString` and `UUIDToBase58String` are different encodings, not aliases. Pick `UUIDToShortString` when you want the canonical URL-safe representation. Pick `UUIDToBase58String` when humans read the string aloud or transcribe it. The "shorter" form trades flexibility for length. Because the encoded string omits the version, you must know which version you are decoding.

## Errors

| Sentinel | Meaning |
|---|---|
| `ErrAlphabetEmpty` | The caller passed an empty alphabet to `NewEncoding`. |
| `ErrAlphabetAmbiguous` | The caller passed an alphabet with a repeated character to `NewEncoding`. |

Both match via `errors.Is`.

## See also

- Source: [`wdk/encoding/`](https://github.com/piko-sh/piko/tree/master/wdk/encoding).
- `encoding/base64`, `encoding/base32`, `encoding/hex` in the Go standard library for those alphabets.
