---
title: How to capture a contention diagnostic
description: Run a one-shot block and mutex profile capture, configure the diagnostic window and rates, and choose between manual and auto-fire triggering.
nav:
  sidebar:
    section: "how-to"
    subsection: "observability"
    order: 40
---

# How to capture a contention diagnostic

A contention diagnostic flips the Go runtime's block and mutex profiling on at configurable rates, captures the resulting profiles after a short window, and turns them off again. Block and mutex profiling cost too much to run continuously. Running them for a deliberate window during an actual contention symptom is exactly the right fit. For the rationale see [about the watchdog](../../explanation/about-the-watchdog.md). For every option see [watchdog API reference](../../reference/watchdog-api.md).

## Trigger a one-shot diagnostic from the CLI

With the watchdog enabled, run:

```bash
piko watchdog contention-diagnostic
```

The call blocks for the configured window (default 60 s) plus capture overhead, then prints `Contention diagnostic completed`. The captured profiles land in the watchdog profile directory and show up in `piko watchdog list`:

```bash
piko watchdog list --type block
piko watchdog list --type mutex
piko watchdog download --latest --type block --output ./pprof
piko watchdog download --latest --type mutex --output ./pprof
go tool pprof -http=:8081 ./pprof/<block-file>
go tool pprof -http=:8081 ./pprof/<mutex-file>
```

Use this when an operator notices contention symptoms (rising scheduler latency, long-tail request latency, threads piling up) and wants to look at the source.

## Configure the diagnostic window

Shorter windows reduce the cost. Longer windows give more sampling. The valid range is 1 second to 5 minutes:

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogContentionDiagnosticWindow(30 * time.Second),
)
```

For most workloads, 30 to 60 seconds is enough to capture a representative contention pattern.

## Tune the block and mutex rates

The runtime's profile rates are global. The diagnostic sets them only for the window's duration, then restores them. Defaults are aggressive enough to catch contention without overwhelming the runtime:

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogContentionDiagnosticBlockProfileRate(1_000_000),    // 1 sample per 1 ms blocking
    piko.WithWatchdogContentionDiagnosticMutexProfileFraction(100),      // 1 in 100 mutex events
)
```

Lower the block-profile rate to sample more aggressively (for example `100_000` for 1 sample per 100 µs of blocking). Lower the mutex fraction to sample more events (for example `10`). On a contended workload the captured profiles are larger but the signal is cleaner.

## Auto-fire on repeated scheduler-latency events

The diagnostic can fire automatically when scheduler-latency events repeat within a short window. Useful for production where the operator is not watching the latency dashboard:

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogSchedulerLatencyP99Threshold(5 * time.Millisecond),
    piko.WithWatchdogContentionDiagnosticAutoFire(),
)
```

When the scheduler-latency threshold trips repeatedly, the watchdog runs a contention diagnostic instead of (or alongside) the threshold's normal warning event. The captured block and mutex profiles point at the source of the contention without operator involvement.

Without `WithWatchdogContentionDiagnosticAutoFire`, the diagnostic only runs when explicitly invoked via the CLI.

## Why the diagnostic is not part of the regular tick loop

Block and mutex profiling impose a per-goroutine cost that scales with the workload. The watchdog's tick loop (see [about the watchdog](../../explanation/about-the-watchdog.md)) keeps its overhead negligible by reading runtime metrics. Turning block and mutex profiling on continuously would itself become the contention. The contention diagnostic is the exception. It opens a short, deliberate window where the cost is acceptable because the goal is to capture exactly that cost.

## See also

- [Watchdog API reference](../../reference/watchdog-api.md) for every contention-diagnostic option.
- [About the watchdog](../../explanation/about-the-watchdog.md) for why continuous block/mutex profiling is a bad default.
- [How to configure the watchdog](configure-watchdog.md) for the surrounding watchdog wiring.
- [How to capture continuous profiles](continuous-profiling.md) for the heap/goroutine/allocs counterpart.
- [How to profile a Piko application](../profiling.md) for the manual `piko profile` flow that also captures block and mutex profiles.
