---
title: Binder API
description: Bind form data, maps, and JSON into Go structs with security-bounded decoding.
nav:
  sidebar:
    section: "reference"
    subsection: "utilities"
    order: 220
---

# Binder API

Piko's action dispatch layer uses the binder to turn HTTP form data, JSON bodies, and generic maps into the typed parameter structs defined on each action. The binder enforces configurable upper bounds on field count, path length, value length, nesting depth, and slice size so malicious inputs cannot exhaust memory or CPU. Source of truth: [`wdk/binder/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/binder/facade.go).

## Entry points

```go
func Bind(ctx context.Context, destination any, source map[string][]string, opts ...Option) error
func BindMap(ctx context.Context, destination any, source map[string]any, opts ...Option) error
func BindJSON(ctx context.Context, destination any, source []byte, opts ...Option) error
```

## Options

```go
func IgnoreUnknownKeys(ignore bool) Option
func WithMaxSliceSize(size int) Option
func WithMaxPathDepth(depth int) Option
func WithMaxPathLength(length int) Option
func WithMaxFieldCount(count int) Option
func WithMaxValueLength(length int) Option
```

## Custom converters

```go
func RegisterConverter(typ any, converter ConverterFunc)
```

Register a converter for a custom type (for example, a typed ID or a timezone-aware datetime). `ConverterFunc` takes the raw string and returns the typed value.

## Global defaults

```go
func SetMaxFormFields(count int)
func SetMaxFormPathLength(length int)
func SetMaxFormValueLength(length int)
func SetMaxFormPathDepth(depth int)
func SetMaxSliceSize(size int)
func SetIgnoreUnknownKeys(ignore bool)
```

These change the process-wide defaults. Per-call `Option` values still override.

## Defaults

| Limit | Default | Purpose |
|---|---|---|
| `MaxFormFields` | 1000 | Prevents hash-flooding. |
| `MaxFormPathLength` | 4096 | Bounds CPU from long paths. |
| `MaxFormValueLength` | 65536 | Bounds memory per field. |
| `MaxFormPathDepth` | 32 | Prevents stack overflow from deep nesting. |
| `MaxSliceSize` | 10000 | Bounds slice allocations. |
| `IgnoreUnknownKeys` | false | Rejects unknown keys unless explicitly opted in. |

## See also

- [Server actions reference](server-actions.md) for the action dispatch surface.
- [How to forms](../how-to/actions/forms.md) for task recipes.
