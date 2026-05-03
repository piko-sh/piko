---
title: How to capture continuous profiles
description: Enable the watchdog's routine-profiling loop, choose interval, profile types, retention, and decide whether routine captures should notify.
nav:
  sidebar:
    section: "how-to"
    subsection: "observability"
    order: 30
---

# How to capture continuous profiles

The watchdog's continuous-profiling loop captures profiles on a fixed schedule, independent of any threshold breach. The captures sit on disk for two purposes. The on-call engineer can compare a breach profile against a healthy baseline, or grab a recent profile after a transient symptom that did not trip a threshold. For the rationale see [about the watchdog](../../explanation/about-the-watchdog.md). For every option see [watchdog API reference](../../reference/watchdog-api.md).

## Turn the loop on

Continuous profiling is opt-in. It also depends on `WithMonitoringProfiling()` - the loop calls into the profiling controller that option constructs, and without it every capture silently drops:

```go
piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringProfiling(),
    piko.WithMonitoringWatchdog(
        piko.WithWatchdogContinuousProfiling(),
    ),
)
```

With nothing else, the loop captures a heap profile every 10 minutes and retains the most recent 6 files. If the profile directory only contains `startup_history.json` after the loop has been running long enough to have produced files, `WithMonitoringProfiling()` is the missing piece - piko logs a startup `WARN` saying so.

## Pick the interval and profile types

Most projects benefit from a heap baseline and an occasional goroutine snapshot:

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogContinuousProfiling(),
    piko.WithWatchdogContinuousProfilingInterval(60 * time.Second),
    piko.WithWatchdogContinuousProfilingTypes("heap", "goroutine", "allocs"),
)
```

The validator enforces a minimum interval of one minute. Allowed types are `heap`, `goroutine`, and `allocs`. A short interval gives finer-grained baselines at the cost of disk. A long interval costs less but increases the gap between an incident and the nearest baseline.

## Tune retention to your incident-investigation window

```go
piko.WithWatchdogContinuousProfilingRetention(24),
```

Retention is per profile type. With an interval of 60 seconds and a retention of 24, the loop keeps the most recent 24 minutes of heap profiles available. Pick a retention long enough to cover the time between an incident and the on-call engineer logging in.

## Decide whether routine captures should notify

The default suppresses routine notifications because a Slack message every 10 minutes is noise. Enable them only when the project has a downstream system that wants the artefact stream (an audit pipeline, a profile-archival service):

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogContinuousProfiling(),
    piko.WithWatchdogContinuousProfilingNotify(),
)
```

When notifications are on, every routine capture fires a `WatchdogPriorityNormal` event through the configured notifier.

## List and download the routine captures

Routine captures share the profile store with threshold-fired captures, so the same CLI commands work:

```bash
piko watchdog list
piko watchdog list --type heap
piko watchdog download --latest --type heap --output ./pprof
```

The sidecar JSON stored alongside each profile records whether the capture was routine or threshold-fired, which threshold tripped, and the runtime metrics at the moment of capture.

## Diff a breach profile against a routine baseline

When a heap-threshold capture fires, pick the most recent routine capture before it as the baseline:

```bash
piko watchdog list --type heap
# Identify the breach file and the immediately-prior routine file by timestamp.
piko watchdog download <breach-file> --output ./pprof
piko watchdog download <baseline-file> --output ./pprof
go tool pprof -diff_base=./pprof/<baseline-file> ./pprof/<breach-file>
```

`WithWatchdogDeltaProfiling` (see [How to configure the watchdog](configure-watchdog.md)) automates the same workflow when the breach is the dominant signal. Continuous profiling fits the post-mortem case where the breach capture has already rotated out.

## See also

- [Watchdog API reference](../../reference/watchdog-api.md) for every option.
- [How to configure the watchdog](configure-watchdog.md) for thresholds and notifier wiring.
- [How to capture a contention diagnostic](contention-diagnostic.md) for the block and mutex profile counterpart.
- [How to profile a Piko application](../profiling.md) for the manual `piko profile` flow.
- [About the watchdog](../../explanation/about-the-watchdog.md) for why routine profiling matters.
