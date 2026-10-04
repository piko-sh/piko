---
title: Runtime symbols
description: Go packages and functions available inside PK template expressions and the interpreted-mode runtime.
nav:
  sidebar:
    section: "reference"
    subsection: "bootstrap"
    order: 20
---

# Runtime symbols

Piko compiles template expressions to Go. In interpreted mode (`dev-i`), [Pipit](https://github.com/piko-sh/pipit) executes that code using its standard-library symbol tables and Piko runtime symbols from `wdk/interp/interp_piko_symbols`.

The package manifests are [`pipit-symbols-stdlib.yaml`](https://github.com/piko-sh/pipit/blob/master/pipit-symbols-stdlib.yaml) and [`piko-symbols-runtime.yaml`](https://github.com/piko-sh/piko/blob/master/piko-symbols-runtime.yaml). Local project packages can also compile from source. Registered native packages take precedence over source compilation.

## Standard-library packages

The table below lists standard-library packages commonly used in templates. Pipit's [standard-library manifest](https://github.com/piko-sh/pipit/blob/master/pipit-symbols-stdlib.yaml) lists the full set. Expressions use package-qualified names such as `strings.ToUpper(state.Name)`.

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
| `os` | Environment access. |
| `path` | URL and file path manipulation. |
| `regexp` | Regex matching and replacement. |
| `sort` | Sort slices in place. |
| `strconv` | String-to-number and number-to-string conversions. |
| `strings` | String manipulation (`Contains`, `Split`, `ToUpper`, etc.). |
| `sync` | Concurrency primitives (rarely used inside templates). |
| `time` | `time.Now()`, duration parsing, formatting. |

## Generic packages

Generic packages expose a fixed set of fast-path instantiations chosen at vendoring time. Calls with other element types fall back to slower reflection-based implementations where Pipit provides an implementation. Other calls need a non-generic equivalent such as `sort.Slice`.

### `slices`

Element types: `string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8` (`byte`), `uint16`, `uint32`, `uint64`, `float32`, `float64`, `bool`.

| Function | Element types |
|---|---|
| `slices.BinarySearch` | all the above except `bool` |
| `slices.Compare` | all the above except `bool` |
| `slices.IsSorted` | all the above except `bool` |
| `slices.Max` | all the above except `bool` |
| `slices.Min` | all the above except `bool` |
| `slices.Sort` | all the above except `bool` |

### `maps`

Key types: `string`, `int`, `int32`, `int64`. Value types: `string`, `int`, `int32`, `int64`, `float64`, `bool`, `any`.

### `cmp`

Element types: `string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8` (`byte`), `uint16`, `uint32`, `uint64`, `float32`, `float64`.

## Piko runtime packages

The Piko tree vendors the following packages into `wdk/interp/interp_piko_symbols`.

### `piko.sh/piko`

Top-level Piko types and helpers. Carries the build tag `!js` - excluded from WebAssembly targets where the server runtime is unavailable. See:

- [Server actions reference](server-actions.md) for `ActionMetadata`, `ValidationField`, cookie helpers.
- [Metadata fields reference](metadata-fields.md) for `Metadata`, `OGTag`, `MetaTag`, `AlternateLink`.
- [Collections API reference](collections-api.md) for `GetData`, `GetSections`, `GetAllCollectionItems`.
- [Errors reference](errors.md) for error types.

### `piko.sh/piko/wdk/binder`

Form, JSON, and map binding into typed Go structs. Each entry point takes a context, a destination pointer, and the source payload.

| Symbol | Signature |
|---|---|
| `binder.Bind` | `Bind(ctx, dest any, src map[string][]string, opts ...Option) error` - bind URL/form values. |
| `binder.BindMap` | `BindMap(ctx, dest any, src map[string]any, opts ...Option) error` - bind a generic map. |
| `binder.BindJSON` | `BindJSON(ctx, dest any, src []byte, opts ...Option) error` - bind a JSON payload. |
| `binder.IgnoreUnknownKeys`, `binder.WithMaxSliceSize`, `binder.WithMaxPathDepth`, `binder.WithMaxPathLength`, `binder.WithMaxFieldCount`, `binder.WithMaxValueLength` | Per-call `Option` constructors. |
| `binder.RegisterConverter`, `binder.SetMaxFormFields`, and related helpers | Process-wide configuration. Call once at bootstrap. |

### `piko.sh/piko/wdk/logger`

Structured logging via `log/slog`. The package exposes context helpers and attribute constructors. Retrieve the logger from context instead of calling `logger.Info(...)` directly.

| Symbol | Purpose |
|---|---|
| `logger.From(ctx, fallback)` | Returns the logger stored on `ctx`, falling back to `fallback`. |
| `logger.WithLogger(ctx, log)` | Stores a logger on `ctx` for downstream retrieval. |
| `logger.MustFrom(ctx)` | Like `From`, but panics if no logger is present. |
| `logger.GetLogger(name)` | Retrieves a named package/component logger. |
| `logger.String`, `logger.Int`, `logger.Int64`, `logger.Uint64`, `logger.Float64`, `logger.Bool`, `logger.Time`, `logger.Duration`, `logger.Error`, `logger.Field`, `logger.Strings` | Attribute constructors for structured fields. |
| `logger.LevelTrace`, `logger.LevelDebug`, `logger.LevelInfo`, `logger.LevelNotice`, `logger.LevelWarn`, `logger.LevelError` | Level constants. |
| `logger.AddPrettyOutput`, `logger.AddJSONOutput`, `logger.AddFileOutput`, `logger.WithLevel`, `logger.WithJSON`, `logger.WithNoColour` | Output configuration. |

Typical use inside a template script block:

```go
ctx, log := logger.From(r.Context(), logger.GetLogger("page"))
log.Info("rendering", logger.String("path", r.URL.Path))
```

### `piko.sh/piko/wdk/runtime`

Runtime collection, section, navigation, and search helpers. The package mirrors the top-level `piko.*` collection symbols for use inside generated code and interpreted scripts. See [Collections API reference](collections-api.md) for the full surface.

Common entry points: `GetData[T]`, `GetSections`, `GetSectionsTree`, `GetAllCollectionItems`, `BuildNavigationFromMetadata`, `SearchCollection`, `QuickSearch`, `FetchCollection`, `RegisterRuntimeProvider`, `Filter`/`FilterGroup`/`SortOption`/`PaginationOptions` and the `FilterOp*` / `Sort*` constants.

### `piko.sh/piko/wdk/safeconv`

Saturating numeric conversions plus boolean and string parsers. The package exposes around 30 typed conversions (for example `IntToUint8`, `Int64ToInt32`), their `Must*` panicking variants, and the generic `ToUint64[T integer]`. See [safeconv API reference](safeconv-api.md) for the full list.

## Register custom symbols

`(*SSRServer).WithSymbols` registers native package symbols before `Run`. Each map entry associates an import path with its exported names and reflection values.

```go
import (
    "reflect"

    "piko.sh/piko"
    "piko.sh/piko/wdk/interp/interp_provider_pipit"

    "myapp/util"
)

func main() {
    server := piko.New()
    server.WithInterpreterProvider(interp_provider_pipit.NewProvider())

    server.WithSymbols(map[string]map[string]reflect.Value{
        "myapp/util": {
            "FormatPrice": reflect.ValueOf(util.FormatPrice),
            "Currency":    reflect.ValueOf((*util.Currency)(nil)),
        },
    })

    // server.Run(actions, piko.RunModeDevInterpreted)
}
```

`WithSymbols` and `WithInterpreterProvider` are methods on `*SSRServer`. Interpreted mode requires a provider, while additional symbols are optional. Compiled `dev` and `prod` builds resolve the same identifiers at compile time.

`piko extract generate` produces symbol tables from `piko-symbols.yaml`. Its default output package, `internal/piko_symbols`, exports the map as `piko_symbols.Symbols`. See [how to run interpreted mode](../how-to/interpreted-mode.md#add-native-package-symbols) for discovery, generation, and registration steps.

## See also

- [Template syntax reference](template-syntax.md).
- [About interpreted mode](../explanation/about-interpreted-mode.md) for when `dev-i` and custom symbols matter.
- [PK file format reference](pk-file-format.md).
- [Collections API reference](collections-api.md) for the full `piko.sh/piko/wdk/runtime` surface.
- [How to collections/markdown](../how-to/collections/markdown.md) for a worked `GetData` / `GetSections` flow.
- [How to actions/forms](../how-to/actions/forms.md) for `ActionMetadata` and `ValidationField` in context.
- Source files: [`pipit-symbols-stdlib.yaml`](https://github.com/piko-sh/pipit/blob/master/pipit-symbols-stdlib.yaml), [`piko-symbols-runtime.yaml`](https://github.com/piko-sh/piko/blob/master/piko-symbols-runtime.yaml).
