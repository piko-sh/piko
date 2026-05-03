---
title: Logger API
description: Structured, context-aware logger based on slog, with OpenTelemetry and integration support.
nav:
  sidebar:
    section: "reference"
    subsection: "services"
    order: 190
---

# Logger API

Piko ships a structured logger built on `slog` with three shipped outputs (pretty console, JSON, rotating file), context-propagation helpers, and a set of attribute constructors. Log levels map to a seven-level scheme: `Trace`, `Debug`, `Info`, `Notice`, `Warn`, `Error`, plus Piko-internal variants. Source of truth: [`wdk/logger/facade.go`](https://github.com/piko-sh/piko/blob/master/wdk/logger/facade.go).

## Accessors

```go
func GetLogger(name string) Logger
func From(ctx context.Context, fallback Logger) (context.Context, Logger)
func WithLogger(ctx context.Context, l Logger) context.Context
func MustFrom(ctx context.Context) Logger
func HasLogger(ctx context.Context) bool
```

The `name` parameter groups loggers for filtering and level overrides. Typical convention: `GetLogger("myapp.actions.customer")`.

## Log levels

| Constant | Meaning |
|---|---|
| `LevelTrace` | Most verbose: Piko internals. |
| `LevelDebug` | Detailed debugging. |
| `LevelInfo` | General operational (default). |
| `LevelNotice` | Important events. |
| `LevelWarn` | Recoverable issues. |
| `LevelError` | Errors that need attention. |

## Attribute constructors

Prefer attribute constructors over raw `slog.Attr` for type safety:

```go
String(key, value string) Attr
Strings(key string, value []string) Attr
Int(key string, value int) Attr
Int64(key string, value int64) Attr
Uint64(key string, value uint64) Attr
Float64(key string, value float64) Attr
Bool(key string, value bool) Attr
Time(key string, value time.Time) Attr
Duration(key string, value time.Duration) Attr
Error(err error) Attr
Field(key string, value any) Attr
```

`Error` is a single-argument helper that always emits the attribute under the key `"error"`. To attach an error under a different key, use `Field("custom_key", err.Error())` or build the `slog.Attr` manually.

## Standard field keys

Use these constants to keep field names consistent across packages:

| Constant | Value |
|---|---|
| `FieldStrContext` | Contextual attributes. |
| `FieldStrMethod` | HTTP method. |
| `FieldStrComponent` | Component name. |
| `FieldStrAdapter` | Adapter name. |
| `FieldStrService` | Service name. |
| `FieldStrError` | Error. |
| `FieldStrPath` | URL path. |
| `FieldStrFile` | File path. |
| `FieldStrDir` | Directory path. |

## Integrations

Shipped integrations attach to the existing logger pipeline. Enable via their own constructor:

| Sub-package | Purpose |
|---|---|
| `logger_integration_sentry` | Sentry error reporting. |
| `logger_integration_otel_grpc` | OpenTelemetry export over gRPC. |
| `logger_integration_otel_http` | OpenTelemetry export over HTTP. |

## Bootstrap

`piko.New(...)` initialises the logger automatically. Application code configures output handlers through `AddPrettyOutput`, `AddJSONOutput`, and `AddFileOutput` inside the initialiser, not via bootstrap options.

```go
func AddPrettyOutput(opts ...OutputOption)
func AddJSONOutput(opts ...OutputOption)
func AddFileOutput(ctx context.Context, name, path string, opts ...OutputOption)
```

`AddPrettyOutput` and `AddJSONOutput` write to stdout. `AddFileOutput` requires a context (controls the rotation goroutine), a name (identifies the output in logs), and a target path. It rotates the file in the background.

Each helper takes one or more `OutputOption` values:

| Option | Effect |
|---|---|
| `WithLevel(level slog.Level)` | Override the level for this output. Defaults to the `LOG_LEVEL` environment variable, or `INFO` when unset. |
| `WithJSON()` | Emit JSON instead of the pretty/text format. |
| `WithNoColour()` | Strip ANSI colour codes (set automatically for file outputs). |

Example:

```go
logger.AddPrettyOutput(logger.WithLevel(slog.LevelDebug))
logger.AddFileOutput(ctx, "errors", "/var/log/app.errors.log",
    logger.WithLevel(slog.LevelError),
    logger.WithJSON(),
)
```

For custom pipelines, construct a `Logger` directly and register it with the relevant integrations.

## See also

- [Piko observability conventions](../how-to/profiling.md) for runtime logging and tracing patterns.
- [`doc.go`](https://github.com/piko-sh/piko/blob/master/wdk/logger/doc.go) for the design rationale.
