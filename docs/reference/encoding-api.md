---
title: Encoding API
description: Base-X encoding and UUID v7 helpers.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 270
---

# Encoding API

Piko's encoding package provides arbitrary-base encoders with safe alphabets, together with UUID v7 helpers for short, sortable identifiers. Standard alphabets (Base64, Base32, Hex) delegate to the Go standard library. Other bases (Base36, Base58) are native. Source file: [`wdk/encoding/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/encoding/facade.go).

## Encoding

```go
func NewEncoding(alphabet string) (*Encoding, error)
func Base64() *Encoding
func Base32() *Encoding
func Hex() *Encoding
func Base36() *Encoding
func Base58() *Encoding
```

Methods on `*Encoding`:

```go
func (e *Encoding) EncodeUint64(n uint64) string
func (e *Encoding) DecodeUint64(s string) (uint64, error)
func (e *Encoding) EncodeBytes(data []byte) string
func (e *Encoding) DecodeBytes(s string) ([]byte, error)
func (e *Encoding) Alphabet() string
func (e *Encoding) Base() int
```

`NewEncoding` rejects empty alphabets and alphabets with ambiguous characters (for example `0` and `O` together).

## UUID v7

```go
func NewUUIDv7(ts time.Time) (uuid.UUID, error)
func NewUUIDv7WithTimestamp(timestamp time.Time) (uuid.UUID, error)
func EncodeUUID(u uuid.UUID) string
func DecodeUUID(s string) (uuid.UUID, error)
func EncodeUUIDv7Timestamp(ts time.Time) (uuid.UUID, error)
```

UUID v7 values are sortable by creation time, which makes them cheap to index in a database. `EncodeUUID` emits a compact Base58 representation, and `DecodeUUID` is its inverse.

## Alphabets

| Constant | Characters |
|---|---|
| `StdBase64Alphabet` | Standard Base64 alphabet. |
| `URLBase64Alphabet` | URL-safe Base64. |
| `StdHexAlphabetLower` | Lowercase hex. |
| `StdHexAlphabetUpper` | Uppercase hex. |
| `StdBase32Alphabet` | Standard Base32. |
| `HexBase32Alphabet` | Extended-hex Base32. |

## Errors

`ErrAlphabetEmpty`, `ErrAlphabetAmbiguous`.

## See also

- `encoding/base64`, `encoding/base32`, `encoding/hex` in the Go standard library.
