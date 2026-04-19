---
title: Runtime symbols
description: Go packages and functions available inside PK template expressions.
nav:
  sidebar:
    section: "reference"
    subsection: "bootstrap"
    order: 20
---

# Runtime symbols

Piko's template expression language compiles to Go. The expressions have access to a curated subset of Go's standard library plus the `piko` runtime packages. This page enumerates the symbols vendored into the bytecode interpreter. Source of truth: [`piko-symbols.yaml`](https://github.com/piko-sh/piko/blob/master/piko-symbols.yaml) and [`piko-symbols-runtime.yaml`](https://github.com/piko-sh/piko/blob/master/piko-symbols-runtime.yaml).

## Standard-library packages

The following Go standard library packages are available as-is. Call them with the usual package-qualified syntax: `strings.ToUpper(state.Name)`.

| Package | Typical uses in templates |
|---|---|
| `bytes` | Byte-slice manipulation. |
| `context` | Context values (rarely used inside templates directly). |
| `encoding/base64` | Encode or decode base64. |
| `encoding/json` | Marshal into inline JSON-LD. |
| `errors` | Check for specific error types. |
| `fmt` | `Sprintf` and friends for formatted output. |
| `html` | HTML escaping beyond what interpolation already provides. |
| `io` | Reader/writer interfaces (rarely used). |
| `math` | Arithmetic helpers (`Ceil`, `Floor`, `Abs`, etc.). |
| `os` | Environment access (limit to read operations). |
| `path` | URL and file path manipulation. |
| `regexp` | Regex matching and replacement. |
| `sort` | Sort slices in place. |
| `strconv` | String-to-number and number-to-string conversions. |
| `strings` | String manipulation (`Contains`, `Split`, `ToUpper`, etc.). |
| `sync` | Concurrency primitives (rarely used inside templates). |
| `time` | `time.Now()`, duration parsing, formatting. |

## Generic packages

Some generic packages expose a restricted set of instantiations:

### `slices`

Element types: `string`, `int`, `int64`, `float64`, `byte`, `bool`.

| Function | Element types |
|---|---|
| `slices.BinarySearch` | string, int, int64, float64, byte |
| `slices.Compare` | string, int, int64, float64, byte |
| `slices.IsSorted` | string, int, int64, float64, byte |
| `slices.Max` | string, int, int64, float64, byte |
| `slices.Min` | string, int, int64, float64, byte |
| `slices.Sort` | string, int, int64, float64, byte |

### `maps`

Key types: `string`, `int`.
Value types: `string`, `int`, `float64`, `bool`, `any`.

### `cmp`

Element types: `string`, `int`, `int64`, `float64`, `byte`.

## Piko runtime packages

Piko registers the following packages for template use:

### `piko.sh/piko`

Top-level framework types accessible from templates and render functions. See the individual reference pages:

- [Server actions reference](server-actions.md) for `ActionMetadata`, `ValidationField`, cookie helpers.
- [Metadata fields reference](metadata-fields.md) for `Metadata`, `OGTag`, `MetaTag`, `AlternateLink`.
- [Collections API reference](collections-api.md) for `GetData`, `GetSections`, `GetAllCollectionItems`.
- [Errors reference](errors.md) for error types.

Note. The `piko.sh/piko` package carries the build tag `!js`, which excludes it from WebAssembly targets where the server runtime is unavailable.

### `piko.sh/piko/wdk/binder`

Template-binding utilities:

| Symbol | Purpose |
|---|---|
| `binder.Bind(target, source)` | Bind source values into a target struct honouring the Piko binding tags. |

### `piko.sh/piko/wdk/logger`

Structured logger bridge:

| Symbol | Purpose |
|---|---|
| `logger.Debug(ctx, msg, keysAndValues...)` | Debug-level log. |
| `logger.Info(ctx, msg, keysAndValues...)` | Info-level log. |
| `logger.Warn(ctx, msg, keysAndValues...)` | Warn-level log. |
| `logger.Error(ctx, msg, keysAndValues...)` | Error-level log. |

### `piko.sh/piko/wdk/runtime`

Runtime collection, section, and navigation helpers that mirror the top-level functions:

| Symbol | Purpose |
|---|---|
| `runtime.GetData[T](r)` | Type-safe data extraction from the current collection item. |
| `runtime.GetSections(r)` | Flat section list. |
| `runtime.GetSectionsTree(r, opts...)` | Nested section tree. |
| `runtime.GetAllCollectionItems(name)` | All items in a named collection. |

### `piko.sh/piko/wdk/safeconv`

Safe type conversions for untrusted template inputs:

| Symbol | Purpose |
|---|---|
| `safeconv.ToInt(v)` | Best-effort conversion to int with bounds checking. |
| `safeconv.ToString(v)` | Best-effort conversion to string. |
| `safeconv.ToBool(v)` | Strict boolean parsing. |

## Register custom symbols

A project can expose additional symbols to the interpreter with bootstrap option `WithSymbols(registrations...)`. See the [bootstrap options reference](bootstrap-options.md).

## See also

- [Template syntax reference](template-syntax.md).
- [Bootstrap options reference](bootstrap-options.md).
- [PK file format reference](pk-file-format.md).
- [How to collections/markdown](../how-to/collections/markdown.md) for a worked `GetData` / `GetSections` flow.
- [How to actions/forms](../how-to/actions/forms.md) for `ActionMetadata` and `ValidationField` in context.
- [Scenario 004: product catalogue](../showcase/004-product-catalogue.md) and [Scenario 005: blog with layout](../showcase/005-blog-with-layout.md) for runnable examples that exercise the symbols listed here.

Source files: [`piko-symbols.yaml`](https://github.com/piko-sh/piko/blob/master/piko-symbols.yaml), [`piko-symbols-runtime.yaml`](https://github.com/piko-sh/piko/blob/master/piko-symbols-runtime.yaml).
