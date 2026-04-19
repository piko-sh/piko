---
title: How to inspect a running app with piko tui
description: Connect piko tui to the monitoring endpoint, navigate the watchdog, metrics, traces, and inspector panels, and triage an incident from the terminal.
nav:
  sidebar:
    section: "how-to"
    subsection: "observability"
    order: 50
---

# How to inspect a running app with piko tui

`piko tui` is an interactive terminal UI that talks to the monitoring transport on a running Piko process. It is the fastest way to look at health, metrics, traces, the watchdog event ring, the profile store, and the orchestrator and registry inspector views in one place. This guide covers connecting, the panels, and a typical incident triage flow. For the underlying transport see [monitoring API reference](../../reference/monitoring-api.md).

## Connect

Make sure the application is running with `WithMonitoring` (see [How to enable the monitoring endpoint](enable-monitoring.md)). Then:

```bash
piko tui
```

The TUI connects to `127.0.0.1:9091` by default. For a different address:

```bash
piko tui --endpoint 192.0.2.10:9091
piko tui --endpoint /var/run/piko/monitoring.sock
```

For a remote process exposed over the loopback only, use SSH tunnelling or `kubectl port-forward` first:

```bash
ssh -L 9091:127.0.0.1:9091 prod-host
piko tui   # then point at localhost
```

## Panels available

The exact panels depend on which monitoring services the application registered:

| Panel | Available when |
|---|---|
| Health | Always (default `HealthService`). |
| Metrics | OTel factories registered (`WithMonitoringOtelFactories`). |
| Traces | OTel factories registered. |
| Watchdog: events, profiles, diagnostic, configuration | `WithMonitoringWatchdog` enabled. |
| Profiling control | `WithMonitoringProfiling` enabled. |
| Orchestrator inspector | Orchestrator service enabled in the application. |
| Registry inspector | Registry service enabled in the application. |
| System info | Always. |

Empty panels usually mean the matching service was not enabled. Returning to the application's bootstrap and adding the corresponding `With*` option fixes the gap.

## Triage an incident from the TUI

A typical contention incident flow:

1. Open the **watchdog events** panel. Filter by recent activity and event type.
2. If a `heap_threshold_exceeded` or `scheduler_latency` event fired, switch to **watchdog profiles**, find the matching capture, and either inspect it inline or copy the filename to download with `piko watchdog download`.
3. Cross-check the **metrics** panel for the corresponding window to confirm the trend (heap rising, p99 climbing, FD count growing).
4. If the symptom is contention and no profile is yet captured, switch to **profiling control** and run `enable 30s` plus `capture block` and `capture mutex` (or trigger the contention diagnostic from the watchdog panel directly).
5. Open **system info** for the runtime context: Go version, GOMAXPROCS, GOMEMLIMIT, current process ID. The startup-history ring under **watchdog configuration** shows whether the process recently crashed.

The TUI stays read-only for everything except the profiling control panel and the contention diagnostic trigger. It cannot change application configuration. Restart the process for that.

## Compose with the CLI

The TUI does not replace the CLI. Use them together. Use the TUI for live state and navigation. Use the CLI for output that needs to live in a shell session, a script, or an alert payload.

```bash
piko watchdog events --tail               # stream events live
piko watchdog events -o json | jq ...     # script around the event ring
piko get metrics                          # snapshot for a CI check
piko watch health --interval 2s           # poll a health probe
```

## See also

- [Monitoring API reference](../../reference/monitoring-api.md) for the surface the TUI consumes.
- [Watchdog API reference](../../reference/watchdog-api.md) for the watchdog panels' source data.
- [CLI reference](../../reference/cli.md) for the commands that complement the TUI.
- [How to enable the monitoring endpoint](enable-monitoring.md) for the wiring that makes the TUI work.
- [How to configure the watchdog](configure-watchdog.md) for the supervisor whose state populates the watchdog panels.
