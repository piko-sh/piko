---
title: Monitoring API
description: WithMonitoring and the monitoring transport options, including TLS, OTel factories, profiling, and the diagnostic-directory and crash-output options that compose with monitoring.
nav:
  sidebar:
    section: "reference"
    subsection: "operations"
    order: 200
---

# Monitoring API

The monitoring API enables a gRPC transport on the running Piko process. The transport exposes telemetry, watchdog state, profiling control, and inspector services to local consumers, primarily `piko tui` and the `piko get`, `piko watch`, `piko describe`, and `piko watchdog` CLI families. For the design rationale see [about monitoring](../explanation/about-monitoring.md). For task recipes see [how to enable the monitoring endpoint](../how-to/observability/enable-monitoring.md). For the watchdog that plugs into this transport see [watchdog API reference](watchdog-api.md). Source: [`options.go`](https://github.com/piko-sh/piko/blob/master/options.go).

## Bootstrap entry point

```go
func WithMonitoring(opts ...MonitoringOption) Option
```

Registers the monitoring service on the container. Without `WithMonitoring`, no gRPC transport runs, no inspector services are reachable, and `piko tui` cannot connect.

> **Note:** `piko tui`, `piko get`, `piko watch`, and `piko watchdog` all need this option. Without `WithMonitoring(...)` in the bootstrap, those CLIs report a service-not-registered error and the gRPC port stays silent.

| Function | Purpose |
|---|---|
| `WithMonitoring(opts...)` | Enables the monitoring service. |
| `WithMonitoringAddress(addr)` | Sets the listen port (default `":9091"`). |
| `WithMonitoringBindAddress(addr)` | Sets the bind interface (default `"127.0.0.1"`). |
| `WithMonitoringAutoNextPort(enabled)` | When the configured port is busy, tries up to 100 consecutive ports. |
| `WithMonitoringTransport(factory)` | Picks the transport implementation. Required. |
| `WithMonitoringOtelFactories(factories)` | Replaces the default no-op OTel factories with real SDK implementations. |
| `WithMonitoringProfiling()` | Registers the remote pprof control service. |
| `WithMonitoringWatchdog(opts...)` | Enables the runtime watchdog. See [watchdog API reference](watchdog-api.md). |
| `WithMonitoringNotifier(notifier)` | Injects a `WatchdogNotifier`. See [watchdog API reference](watchdog-api.md). |
| `WithMonitoringProfileUploader(uploader)` | Injects a `WatchdogProfileUploader`. See [watchdog API reference](watchdog-api.md). |

## Transport

```go
func WithMonitoringTransport(factory monitoring_domain.TransportFactory) MonitoringOption
```

Wires a transport implementation into the monitoring service. The shipped factory is `monitoring_grpc.Transport()`. Without a transport, the service starts no listener and exposes no RPCs.

```go
import (
    "piko.sh/piko"
    monitoring_grpc "piko.sh/piko/wdk/monitoring/monitoring_transport_grpc"
)

piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
)
```

## OpenTelemetry factories

```go
func WithMonitoringOtelFactories(factories monitoring_domain.ServiceFactories) MonitoringOption
```

Replaces the default no-op factories. Without real factories the gRPC transport starts but the metrics, traces, and span backlog views remain empty.

```go
import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/logger/logger_otel_sdk"
)

piko.WithMonitoring(
    piko.WithMonitoringOtelFactories(logger_otel_sdk.OtelServiceFactories()),
)
```

The same factories drive both the OTLP push pipeline and the gRPC transport's read views, so a single configuration covers both surfaces.

## Address and port

| Function | Default | Purpose |
|---|---|---|
| `WithMonitoringAddress(addr)` | `":9091"` | The listen port. Pass a port-only form (`":9092"`) or a host+port form. |
| `WithMonitoringBindAddress(addr)` | `"127.0.0.1"` | The interface to bind. Localhost by default; set to `"0.0.0.0"` to expose on all interfaces. |
| `WithMonitoringAutoNextPort(enabled)` | `false` | When `true`, falls forward up to 100 ports if the configured one is in use. The actual port lands in the startup log. |

## TLS

`WithMonitoringTLS(opts...)` enables TLS. Without it, the transport runs in cleartext, which is acceptable on `127.0.0.1` and inappropriate on any other interface.

| Function | Purpose |
|---|---|
| `WithMonitoringTLS(opts...)` | Toggles TLS on. |
| `WithMonitoringTLSCertFile(path)` | Sets the certificate file. |
| `WithMonitoringTLSKeyFile(path)` | Sets the private key file. |
| `WithMonitoringTLSClientCA(path)` | Sets the client-CA file. Enables mTLS. |
| `WithMonitoringTLSClientAuth(authType)` | Sets the client-cert verification mode (`"require_and_verify"`, `"request"`, etc.). |
| `WithMonitoringTLSMinVersion(version)` | Sets the minimum TLS version (`"1.2"` or `"1.3"`). |
| `WithMonitoringTLSHotReload(enabled)` | When `true`, the server re-reads cert and key files on change without a restart. |

## Profiling control

```go
func WithMonitoringProfiling() MonitoringOption
```

Registers the gRPC profiling control service. With it enabled, the CLI commands work:

```bash
piko profiling enable 30m       # turn block + mutex profiling on for a window
piko profiling capture heap     # write a heap profile to the profile directory
piko profiling capture cpu 30s  # CPU profile for a fixed duration
piko profiling status           # report whether profiling is active
piko profiling disable          # turn profiling off early
```

Captured profiles land in the same directory the watchdog uses, so `piko watchdog list` shows both watchdog-fired and operator-fired captures.

## Diagnostic directory and crash capture

These options sit alongside `WithMonitoring` instead of inside it. They configure the on-disk surface that the monitoring tools, the watchdog, and the Go runtime write to.

| Function | Purpose |
|---|---|
| `WithDiagnosticDirectory(directory)` | Single root for all runtime diagnostic artefacts. The crash mirror writes to `<dir>/crash.log`, the watchdog writes profiles to `<dir>/profiles/`, and startup history lives under the same root. |
| `WithCrashOutput(path)` | Mirrors fatal-error output (panics, stack overflows, concurrent map writes, out-of-memory aborts) to the given file. Append mode. The runtime keeps the file open for the process lifetime. Empty path leaves the feature disabled. |
| `WithCrashTraceback(level)` | Sets `GOTRACEBACK`. Levels: `"none"`, `"single"` (Go default), `"all"`, `"system"`, `"crash"` (raises SIGABRT after the traceback so the kernel or systemd-coredump can capture a coredump), `"wer"` (Windows error reporting). |
| `WithAutoMemoryLimit(provider)` | Sets `GOMEMLIMIT` from a cgroup-aware provider. Use with `automemlimit.Provider()` from the `wdk/` package. Prevents kernel out-of-memory kills in containers by making the GC aware of the memory ceiling. |

## gRPC services exposed

When enabled, the transport registers these services on the listen port:

| Service | Used by |
|---|---|
| `HealthService` | `piko get health`, `piko diagnostics`. |
| `MetricsService` | `piko get metrics`, `piko get traces`, `piko info`. Driven by the OTel factories. |
| `WatchdogService` | `piko watchdog *`. Driven by the watchdog. |
| `ProfilingService` | `piko profiling *`. Registers only when the application calls `WithMonitoringProfiling()`. |
| `OrchestratorInspectorService` | `piko get tasks`, `piko get workflows`. Registers only when the application enables the orchestrator. |
| `RegistryInspectorService` | `piko get artefacts`, `piko get variants`. Registers only when the application enables the registry. |

## Default endpoint summary

| Aspect | Default |
|---|---|
| Listen port | `:9091` |
| Bind address | `127.0.0.1` |
| TLS | off |
| Auto-next-port | off |
| Profile directory (when watchdog enabled and no `WithDiagnosticDirectory`) | `os.TempDir()/piko-watchdog` |

## See also

- [About monitoring](../explanation/about-monitoring.md) for the design rationale.
- [About the watchdog](../explanation/about-the-watchdog.md) for the supervisor that plugs into this transport.
- [Watchdog API reference](watchdog-api.md) for thresholds, capture types, and the `piko watchdog` CLI.
- [How to enable the monitoring endpoint](../how-to/observability/enable-monitoring.md) for the wiring recipe.
- [How to inspect a running app with piko tui](../how-to/observability/inspect-with-tui.md) for the consumer side.
- [Bootstrap options reference](bootstrap-options.md) for the broader option surface.
- [CLI reference](cli.md) for `piko get`, `piko watch`, `piko describe`, `piko tui`, and the profiling subcommands.
