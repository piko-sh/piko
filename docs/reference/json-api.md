---
title: JSON API
description: Pluggable JSON encoding with an encoding/json default and optional Sonic backend.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 295
---

# JSON API

Piko uses a package-level JSON facade so the framework can swap the underlying encoder. The default is Go's `encoding/json`. An optional Sonic-based provider is available for projects that need faster encoding and decoding. Source of truth: [`wdk/json/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/json/facade.go).

## Functions

```go
func Marshal(v any) ([]byte, error)
func MarshalIndent(v any, prefix, indent string) ([]byte, error)
func Unmarshal(data []byte, v any) error
func NewEncoder(w io.Writer) Encoder
func NewDecoder(r io.Reader) Decoder
func Freeze()
```

`Freeze` disables further provider changes after bootstrap. Call it once, early in `main`, when the application has registered its final provider.

## Provider registration

Custom providers satisfy a small interface and register during bootstrap. For the Sonic provider:

```go
import _ "piko.sh/piko/wdk/json/json_provider_sonic"
```

The blank import runs the provider's `init` function, which replaces the default. Remove the import to return to `encoding/json`.

## Pretouch

Sonic supports `Pretouch` to pre-compile a type's marshal path before first use:

```go
sonic.Pretouch(reflect.TypeOf(MyType{}))
```

Call this during startup for hot types to avoid first-request overhead.

## See also

- [`doc.go`](https://github.com/piko-sh/piko/blob/master/wdk/json/doc.go) for the swap rationale.
