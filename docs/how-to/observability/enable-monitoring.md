---
title: How to enable the monitoring endpoint
description: Wire WithMonitoring with a gRPC transport, OTel SDK factories, optional TLS, and the profiling control service so piko tui and the watchdog can connect.
nav:
  sidebar:
    section: "how-to"
    subsection: "observability"
    order: 10
---

# How to enable the monitoring endpoint

The monitoring endpoint is a gRPC server the running Piko process exposes for `piko tui`, the watchdog, and the `piko get`, `piko watch`, `piko describe`, `piko profiling`, and `piko watchdog` CLI families. This guide wires it from a clean bootstrap. For the design rationale see [about monitoring](../../explanation/about-monitoring.md). For the full option surface see [monitoring API reference](../../reference/monitoring-api.md).

## Wire the minimum monitoring transport

```go
package main

import (
    "piko.sh/piko"
    monitoring_grpc "piko.sh/piko/wdk/monitoring/monitoring_transport_grpc"
    "piko.sh/piko/wdk/logger/logger_otel_sdk"
)

func main() {
    server := piko.New(
        piko.WithMonitoring(
            piko.WithMonitoringTransport(monitoring_grpc.Transport()),
            piko.WithMonitoringOtelFactories(logger_otel_sdk.OtelServiceFactories()),
        ),
    )
    server.Run()
}
```

`WithMonitoringTransport` matters. Without it the service starts no listener. `WithMonitoringOtelFactories` populates the metrics, traces, and span backlog views. With the default no-op factories the transport runs but those views stay empty. The endpoint binds to `127.0.0.1:9091` by default.

## Verify with the CLI

With the server running, in another shell:

```bash
piko diagnostics
```

The command resolves the endpoint, opens a connection, and reports health. A success means `piko tui`, `piko watch`, and the watchdog CLI commands also reach the same endpoint. A failure usually means the application did not call `WithMonitoring`, the transport factory was not passed, or another process holds the port.

## Pick a non-default port

```go
piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringAddress(":9192"),
)
```

CLI consumers point at the new endpoint with `--endpoint`:

```bash
piko diagnostics --endpoint 127.0.0.1:9192
piko tui --endpoint 127.0.0.1:9192
```

## Run multiple Piko processes on the same host

In multi-process developer setups, the default port collides. Enable auto-next-port so each process steps forward to the next free port and logs the choice:

```go
piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringAutoNextPort(true),
)
```

The startup log includes a line like `monitoring listening on 127.0.0.1:9092`. Copy that into the CLI's `--endpoint` flag.

In production, leave auto-next-port off. A port that silently shifts is usually a worse problem than a port that fails to bind.

## Expose the endpoint over the network with TLS

The default bind address is localhost. To reach the endpoint from a remote shell or another node, bind to the network interface and turn on TLS:

```go
piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringBindAddress("0.0.0.0"),
    piko.WithMonitoringTLS(
        piko.WithMonitoringTLSCertFile("/etc/piko/tls/monitoring.pem"),
        piko.WithMonitoringTLSKeyFile("/etc/piko/tls/monitoring.key"),
        piko.WithMonitoringTLSMinVersion("1.3"),
        piko.WithMonitoringTLSHotReload(true),
    ),
)
```

Hot reload picks up a rotated certificate without a restart. For mutual TLS, add a client CA and a verification mode:

```go
piko.WithMonitoringTLS(
    piko.WithMonitoringTLSCertFile("/etc/piko/tls/monitoring.pem"),
    piko.WithMonitoringTLSKeyFile("/etc/piko/tls/monitoring.key"),
    piko.WithMonitoringTLSClientCA("/etc/piko/tls/ops-ca.pem"),
    piko.WithMonitoringTLSClientAuth("require_and_verify"),
    piko.WithMonitoringTLSMinVersion("1.3"),
)
```

When the endpoint stays on `127.0.0.1`, TLS is unnecessary. Reach the loopback address from a remote operator with SSH tunnelling or `kubectl port-forward`.

## Add the profiling control service

`WithMonitoringProfiling` registers a remote pprof control surface. The CLI commands that flip block and mutex profiling on, capture profiles on demand, and disable profiling all rely on it:

```go
piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringProfiling(),
)
```

Without `WithMonitoringProfiling`, `piko profiling enable` returns an unimplemented error. The application takes no performance hit when the profiling service exists but no profile is active.

## Compose with the watchdog

The watchdog plugs into the monitoring service through `WithMonitoringWatchdog`. See [How to configure the watchdog](configure-watchdog.md) for thresholds and notifier wiring:

```go
piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringOtelFactories(logger_otel_sdk.OtelServiceFactories()),
    piko.WithMonitoringWatchdog(
        // thresholds, profile directory, continuous profiling, etc.
    ),
)
```

## See also

- [Monitoring API reference](../../reference/monitoring-api.md) for the full option surface.
- [About monitoring](../../explanation/about-monitoring.md) for the design rationale.
- [How to configure the watchdog](configure-watchdog.md) for the supervisor that plugs into this transport.
- [How to inspect a running app with piko tui](inspect-with-tui.md) for the consumer side.
- [CLI reference](../../reference/cli.md) for `piko diagnostics`, `piko tui`, `piko get`, `piko watch`, and the profiling subcommands.
