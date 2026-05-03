---
title: Watchdog API
description: Every WithWatchdog* bootstrap option, watchdog event types and priorities, the WatchdogNotifier and WatchdogProfileUploader interfaces, and the piko watchdog CLI subcommands.
nav:
  sidebar:
    section: "reference"
    subsection: "operations"
    order: 210
---

# Watchdog API

The watchdog is a runtime supervisor that monitors heap, RSS, goroutines, file descriptors, and scheduler latency, and captures diagnostic profiles when thresholds breach. The application configures it under `WithMonitoring`. It exposes its state to `piko tui` and the `piko watchdog` CLI through the gRPC monitoring transport. For the design rationale see [about the watchdog](../explanation/about-the-watchdog.md). For task recipes see [how to configure the watchdog](../how-to/observability/configure-watchdog.md). Source: [`options.go`](https://github.com/piko-sh/piko/blob/master/options.go), [`cmd_watchdog.go`](https://github.com/piko-sh/piko/blob/master/cmd/piko/internal/cli/cmd_watchdog.go).

## Bootstrap entry point

```go
func WithMonitoringWatchdog(opts ...WatchdogOption) MonitoringOption
```

Enables the watchdog inside the monitoring service. Without this option the watchdog never starts and `piko watchdog` calls fail with a "service not registered" gRPC error.

> **Note:** This option must sit *inside* `WithMonitoring(...)`, not alongside it. The signature returns `MonitoringOption` (not `Option`); placing it at the top level fails to compile.

> **Prerequisite:** Pair this with `WithMonitoringProfiling()`. The watchdog's capture paths (continuous, threshold-triggered, pre-death, contention) all dispatch through the profiling controller that option constructs. Without it, every capture silently no-ops and the profile directory only ever contains `startup_history.json`. Piko emits a startup `WARN` when this dependency is missing.

```go
piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringProfiling(),
    piko.WithMonitoringWatchdog(
        piko.WithWatchdogProfileDirectory("/var/lib/piko/profiles"),
        piko.WithWatchdogContinuousProfiling(),
    ),
)
```

## Threshold options

| Option | Default | Purpose |
|---|---|---|
| `WithWatchdogHeapThresholdPercent(p)` | `0.85` | Heap fraction of `GOMEMLIMIT` that triggers a heap profile. |
| `WithWatchdogHeapThresholdBytes(b)` | `512 MiB` | Absolute heap threshold when `GOMEMLIMIT` is unset. |
| `WithWatchdogRSSThresholdPercent(p)` | `0.85` | RSS fraction of the cgroup memory limit. |
| `WithWatchdogGoroutineThreshold(n)` | `10000` | Goroutine count that triggers a goroutine profile. |
| `WithWatchdogFDPressureThresholdPercent(p)` | `0.80` | File-descriptor fraction of soft `RLIMIT_NOFILE`. Pass `0` to disable. |
| `WithWatchdogSchedulerLatencyP99Threshold(d)` | `10ms` | p99 scheduler latency. Pass zero to disable. |

Floating-point thresholds use the `0.0-1.0` range. Counts and durations use their native types.

## Loop, capture, and budget options

| Option | Default | Purpose |
|---|---|---|
| `WithWatchdogCheckInterval(d)` | `500ms` | Tick frequency for threshold evaluation. |
| `WithWatchdogCooldown(d)` | `2m` | Minimum gap between captures for the same metric type. |
| `WithWatchdogMaxProfilesPerType(n)` | `5` | Files retained per profile type. Oldest rotates out. |
| `WithWatchdogMaxWarningsPerWindow(n)` | `10` | Warning-only events permitted per capture window. |
| `WithWatchdogProfileDirectory(dir)` | `os.TempDir()/piko-watchdog` | Local directory for profile, sidecar, and history files. |
| `WithWatchdogDeltaProfiling()` | off | Stores a baseline heap snapshot beside each capture for `pprof -diff_base`. |

When the application calls `WithDiagnosticDirectory` on the container, profile files land at `<dir>/profiles/` only when the application has not called `WithWatchdogProfileDirectory`. `WithWatchdogProfileDirectory` takes precedence. The diagnostic-directory override only kicks in when the watchdog profile directory is empty.

## Continuous profiling

A separate routine loop captures profiles at a fixed interval, independent of any threshold breach.

| Option | Default | Purpose |
|---|---|---|
| `WithWatchdogContinuousProfiling()` | off | Enables the routine loop. |
| `WithWatchdogContinuousProfilingInterval(d)` | `10m` | Period between routine captures. Minimum `1m`. |
| `WithWatchdogContinuousProfilingTypes(t...)` | `["heap"]` | Profile types per interval. Allowed: `heap`, `goroutine`, `allocs`. |
| `WithWatchdogContinuousProfilingRetention(n)` | `6` | Files retained per type. |
| `WithWatchdogContinuousProfilingNotify()` | off | Emits informational notifications for each routine capture. |

## Contention diagnostic

A short-window diagnostic that flips block and mutex profiling on at configurable rates, captures the resulting profiles, and turns them off again.

| Option | Default | Purpose |
|---|---|---|
| `WithWatchdogContentionDiagnosticWindow(d)` | `60s` | Time block + mutex profiling stays active. Range `1s-5m`. |
| `WithWatchdogContentionDiagnosticAutoFire()` | off | Fires automatically on repeated scheduler-latency events. |
| `WithWatchdogContentionDiagnosticBlockProfileRate(rate)` | `1e6` | Runtime block profile rate (one sample per `rate` ns of blocking). |
| `WithWatchdogContentionDiagnosticMutexProfileFraction(frac)` | `100` | Runtime mutex profile fraction (1 in `frac` events sampled). |

The diagnostic is a one-shot, blocking call when triggered manually:

```bash
piko watchdog contention-diagnostic
```

## Notifier and uploader

These two ports plug into the monitoring level (not inside `WithMonitoringWatchdog`), so every watchdog notification flows through the same notifier.

```go
type WatchdogNotifier        = monitoring_domain.WatchdogNotifier
type WatchdogProfileUploader = monitoring_domain.WatchdogProfileUploader
```

| Option | Purpose |
|---|---|
| `WithWatchdogNotifier(notifier)` | Delivers `WatchdogEvent`s to an external system (Slack, PagerDuty, email). |
| `WithWatchdogProfileUploader(uploader)` | Uploads each captured profile to remote storage after the local write. |

The notifier receives every event the watchdog emits, including the typed event categories below.

## Event types

```go
type WatchdogEvent         = monitoring_domain.WatchdogEvent
type WatchdogEventType     = monitoring_domain.WatchdogEventType
type WatchdogEventPriority = monitoring_domain.WatchdogEventPriority
```

Priorities:

| Constant | Meaning |
|---|---|
| `WatchdogPriorityNormal` | Informational. Safe to ignore in alerting. |
| `WatchdogPriorityHigh` | Warrants prompt investigation. |
| `WatchdogPriorityCritical` | Imminent system instability. |

The CLI's `piko watchdog events --type <type>` flag filters by `WatchdogEventType`. The full set of constants in `internal/monitoring/monitoring_domain/watchdog_notifier.go` is:

| Constant | String value |
|---|---|
| `WatchdogEventHeapThresholdExceeded` | `heap_threshold_exceeded` |
| `WatchdogEventRSSThresholdExceeded` | `rss_threshold_exceeded` |
| `WatchdogEventGoroutineThresholdExceeded` | `goroutine_threshold_exceeded` |
| `WatchdogEventGoroutineSafetyCeiling` | `goroutine_safety_ceiling` |
| `WatchdogEventGCPressureWarning` | `gc_pressure_warning` |
| `WatchdogEventCaptureError` | `capture_error` |
| `WatchdogEventGomemlimitNotConfigured` | `gomemlimit_not_configured` |
| `WatchdogEventMemProfileRateDisabled` | `memprofilerate_disabled` |
| `WatchdogEventHeapTrendWarning` | `heap_trend_warning` |
| `WatchdogEventGoroutineLeakDetected` | `goroutine_leak_detected` |
| `WatchdogEventPreDeathSnapshot` | `pre_death_snapshot` |
| `WatchdogEventLoopPanicked` | `loop_panicked` |
| `WatchdogEventFDPressureExceeded` | `fd_pressure_exceeded` |
| `WatchdogEventSchedulerLatencyHigh` | `scheduler_latency_high` |
| `WatchdogEventCrashLoopDetected` | `crash_loop_detected` |
| `WatchdogEventPreviousCrashClassified` | `previous_crash_classified` |
| `WatchdogEventRoutineProfileCaptured` | `routine_profile_captured` |
| `WatchdogEventContentionDiagnostic` | `contention_diagnostic` |

## CLI: `piko watchdog`

Connects to the monitoring transport (default `127.0.0.1:9091`) and operates on the running process's watchdog state.

| Subcommand | Purpose |
|---|---|
| `piko watchdog status` | Prints lifecycle, thresholds, crash-loop, continuous-profiling, and contention-diagnostic configuration. |
| `piko watchdog list [--type <type>]` | Lists stored profiles. Type, timestamp, size, filename. |
| `piko watchdog download [<filename> \| --latest --type <type>] [--output <dir>] [--skip-sidecar]` | Downloads a profile file (and its JSON sidecar by default) to the local directory. |
| `piko watchdog prune [--type <type>]` | Removes stored profiles. Without `--type` removes everything. |
| `piko watchdog history` | Prints the startup-history ring (process ID, started, stopped, reason, host, version). |
| `piko watchdog events [--since <duration>] [--type <type>] [--limit <n>] [--tail]` | Lists or streams events from the in-memory ring. `--tail` subscribes to new events as they fire. |
| `piko watchdog contention-diagnostic` | Runs a one-shot contention diagnostic. Blocks for the configured window. |

The global flags from [CLI reference](cli.md#global-flags-monitoring-commands) (`-e/--endpoint`, `-o/--output`, `--no-colour`, etc.) apply.

## Examples

```bash
# Status and recent activity
piko watchdog status
piko watchdog events --since 1h
piko watchdog events --type heap_threshold_exceeded
piko watchdog list

# Download the most recent heap profile and inspect it
piko watchdog download --latest --type heap --output ./pprof
go tool pprof ./pprof/heap-<timestamp>.pprof

# Watch for new events live
piko watchdog events --tail

# Trigger a contention diagnostic on demand
piko watchdog contention-diagnostic

# Clean up stored heap profiles
piko watchdog prune --type heap

# Detect crash loops
piko watchdog history
```

## See also

- [About the watchdog](../explanation/about-the-watchdog.md) for the design rationale and budget design.
- [Monitoring API reference](monitoring-api.md) for the transport that exposes the watchdog.
- [How to configure the watchdog](../how-to/observability/configure-watchdog.md) for the wiring recipe.
- [How to capture continuous profiles](../how-to/observability/continuous-profiling.md) for the routine baseline.
- [How to capture a contention diagnostic](../how-to/observability/contention-diagnostic.md) for the on-demand block and mutex profile.
- [How to profile a Piko application](../how-to/profiling.md) for the manual `piko profile` flow.
- [CLI reference](cli.md) for the surrounding subcommands.
