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

Piko uses a package-level JSON facade so it can swap the underlying encoder. The default is Go's `encoding/json`. An optional Sonic-based provider is available for projects that need faster encoding and decoding. Source of truth: [`wdk/json/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/json/facade.go).

## Type aliases

```go
type Provider = pikojson.Provider
type API      = pikojson.API
type Config   = pikojson.Config
type Encoder  = pikojson.Encoder
type Decoder  = pikojson.Decoder
```

JSON backends implement `Provider`. `API` is a frozen lazy-resolving handle for encode/decode operations. `Config` describes encoding behaviour.

## Functions

```go
func Marshal(v any) ([]byte, error)
func MarshalIndent(v any, prefix, indent string) ([]byte, error)
func MarshalString(v any) (string, error)
func Unmarshal(data []byte, v any) error
func UnmarshalString(s string, v any) error
func ValidString(s string) bool
func NewEncoder(w io.Writer) Encoder
func NewDecoder(r io.Reader) Decoder
func Freeze(config Config) API
func Pretouch(t reflect.Type) error
func StdConfig() API
func DefaultConfig() API
```

| Function | Purpose |
|---|---|
| `Marshal` / `Unmarshal` | Encode and decode against a `[]byte`. |
| `MarshalIndent` | Pretty-print encode. |
| `MarshalString` / `UnmarshalString` | String-typed variants for hot paths that already hold a `string`. |
| `ValidString` | Reports whether a string is well-formed JSON. |
| `NewEncoder` / `NewDecoder` | Streaming encode and decode. |
| `Freeze(config)` | Returns a frozen `API` that resolves its backing provider lazily on first use. |
| `Pretouch` | Pre-compiles encode and decode paths for the given type so the first call avoids reflection cost. |
| `StdConfig` | Returns an `API` whose behaviour matches the standard library. |
| `DefaultConfig` | Returns the high-performance default `API`. |

## Provider registration

Custom providers satisfy the `Provider` interface and register during bootstrap. For the Sonic provider:

```go
import _ "piko.sh/piko/wdk/json/json_provider_sonic"
```

The blank import runs the provider's `init` function, which replaces the default. Remove the import to return to `encoding/json`.

## Pretouch

Use `Pretouch` to pre-compile a type's marshal and unmarshal paths before first use:

```go
import (
    "reflect"

    "piko.sh/piko/wdk/json"
)

func init() {
    if err := json.Pretouch(reflect.TypeOf(MyType{})); err != nil {
        // log and continue; Pretouch is an optimisation, not a hard requirement
    }
}
```

Call from package `init` or early in `main` for hot types to avoid first-request overhead. The package delegates the call to whichever provider is active. If the active provider does not implement pre-touching the call is a no-op.

## See also

- [`doc.go`](https://github.com/piko-sh/piko/blob/master/wdk/json/doc.go) for the swap rationale.
