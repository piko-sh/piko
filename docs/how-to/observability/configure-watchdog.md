---
title: How to configure the watchdog
description: Set heap, RSS, goroutine, FD, and scheduler-latency thresholds, choose a profile directory, enable delta profiling, and wire a notifier so the watchdog captures profiles and pages on-call when symptoms appear.
nav:
  sidebar:
    section: "how-to"
    subsection: "observability"
    order: 20
---

# How to configure the watchdog

The watchdog is a runtime supervisor that catches heap pressure, RSS pressure, goroutine leaks, FD pressure, and scheduler latency, and captures diagnostic profiles the moment a threshold breaches. This guide covers the wiring choices that make it useful for a typical production deployment. For the rationale see [about the watchdog](../../explanation/about-the-watchdog.md). For every option see [watchdog API reference](../../reference/watchdog-api.md).

## Enable the watchdog with sensible defaults

The watchdog plugs into `WithMonitoring`, so wire the monitoring transport first (see [How to enable the monitoring endpoint](enable-monitoring.md)). Add `WithMonitoringProfiling()` alongside it: the watchdog uses the profiling controller it constructs to actually capture profiles, and without it every capture path silently no-ops. With those two in place, the defaults are reasonable:

```go
piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringProfiling(),
    piko.WithMonitoringWatchdog(),
)
```

Defaults: 0.85 heap fraction of `GOMEMLIMIT`, 0.85 RSS fraction of cgroup limit, 10,000 goroutines, 0.80 FD fraction. The loop ticks at 500 ms with a 2 minute cooldown and keeps 5 profiles per type. The scheduler-latency p99 threshold is 10 ms. The profile directory sits under `os.TempDir()/piko-watchdog`.

If the profile directory only ever contains `startup_history.json` after threshold breaches, the missing dependency is `WithMonitoringProfiling()` - piko logs a startup `WARN` to the same effect.

## Pin the profile directory

The default temp-directory location is fine for development but lost across container restarts in production. Pick a stable directory the operator can mount:

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogProfileDirectory("/var/lib/piko/profiles"),
)
```

If the application also calls `WithDiagnosticDirectory(root)` at the container level, the watchdog only joins `<root>/profiles/` when `WithWatchdogProfileDirectory` was not set. An explicit `WithWatchdogProfileDirectory` always wins, so prefer it when the deployment needs the profiles in a specific location distinct from the container-wide diagnostic root.

## Tune the thresholds for the workload

For a memory-bound service that keeps heap close to the limit deliberately, the default 0.85 heap fraction fires constantly. Loosen it:

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogHeapThresholdPercent(0.92),
    piko.WithWatchdogRSSThresholdPercent(0.92),
)
```

For a service that spawns short-lived goroutines per request, the default 10,000 goroutine threshold is conservative. Raise it:

```go
piko.WithWatchdogGoroutineThreshold(50000),
```

For a network-heavy service approaching FD limits, tighten the threshold so a leak fires earlier:

```go
piko.WithWatchdogFDPressureThresholdPercent(0.65),
```

Only `WithWatchdogFDPressureThresholdPercent` and `WithWatchdogSchedulerLatencyP99Threshold` honour `0` as "disable". `WithWatchdogHeapThresholdPercent` rejects `0` (and any value outside the open-closed interval `(0.0, 1.0]`) at startup with `ErrInvalidWatchdogConfig`. `WithWatchdogRSSThresholdPercent` accepts `0` because the RSS rule is optional, but rejects negatives and values above `1.0`. To loosen instead of disable a heap threshold, set it close to the ceiling (`0.99`) or omit the option to keep the default.

## Tighten or loosen the loop and budgets

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogCheckInterval(250 * time.Millisecond),
    piko.WithWatchdogCooldown(5 * time.Minute),
    piko.WithWatchdogMaxProfilesPerType(10),
    piko.WithWatchdogMaxWarningsPerWindow(20),
)
```

Shorter check intervals catch faster transients at negligible CPU cost. Longer cooldowns suppress repeat captures from a flapping threshold. Larger profile budgets keep more history at the cost of disk. Larger warning budgets let the FD-pressure and scheduler-latency rules log more before getting throttled.

## Enable delta profiling for easier diffs

Delta profiling stores a baseline heap snapshot beside each capture. With it, comparing the breach against the baseline takes one `pprof` flag instead of a separate continuous capture:

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogDeltaProfiling(),
)
```

After a breach:

```bash
piko watchdog download --latest --type heap --output ./pprof
go tool pprof -diff_base=./pprof/heap-<timestamp>.baseline.pprof ./pprof/heap-<timestamp>.pprof
```

The diff shows what changed between the moment before and the moment after the threshold trip.

## Wire a notifier for alerts

Without a notifier, watchdog events stay in the in-memory ring and are visible only through `piko watchdog events`. Pass a `piko.WatchdogNotifier` to push them outward. Piko ships eight built-in notification providers (Slack, Discord, PagerDuty, Teams, Google Chat, ntfy.sh, Webhook, Stdout) reached through the public `piko.sh/piko/wdk/notification` package, where `notification.ProviderPort` is the alias the providers satisfy. Piko does not ship a ready-made adapter that bridges `WatchdogNotifier` to a `notification.ProviderPort` because the message shape varies per project. Implement `WatchdogNotifier.Notify` directly:

```go
import (
    "context"

    "piko.sh/piko"
    monitoring_grpc "piko.sh/piko/wdk/monitoring/monitoring_transport_grpc"
    "piko.sh/piko/wdk/notification"
)

// slackWatchdogNotifier satisfies piko.WatchdogNotifier by formatting each
// watchdog event into a notification payload and forwarding it to a
// notification.ProviderPort (a Slack provider here, but any of the eight
// built-in providers fits).
type slackWatchdogNotifier struct {
    provider notification.ProviderPort
    service  string
}

func (n *slackWatchdogNotifier) Notify(ctx context.Context, event piko.WatchdogEvent) error {
    return n.provider.Send(ctx, notification.Message{
        Title:    string(event.EventType),
        Body:     event.Message,
        Severity: mapPriority(event.Priority),
        Fields:   event.Fields, // host, version, profile filename, etc.
    })
}

notifier := &slackWatchdogNotifier{provider: slackProvider, service: "myapp"}

piko.WithMonitoring(
    piko.WithMonitoringTransport(monitoring_grpc.Transport()),
    piko.WithMonitoringWatchdog(
        piko.WithWatchdogProfileDirectory("/var/lib/piko/profiles"),
    ),
    piko.WithWatchdogNotifier(notifier),
)
```

The exact `notification.Message` field set depends on the provider package. The point is that `WatchdogNotifier` is a one-method interface and the project owns the mapping. Translate `event.EventType`, `event.Priority`, `event.Message`, and `event.Fields` (which carries the captured profile filename, host, and version) into the destination's message format.

The notifier receives every watchdog event, including continuous-profiling events when the application turns `WithWatchdogContinuousProfilingNotify` on. For a chatty workload, leave routine notifications off and only alert on threshold breaches.

## Upload profiles off-host

For containerised or ephemeral hosts, store the profile files in object storage so they survive a restart:

```go
piko.WithMonitoring(
    piko.WithMonitoringWatchdog(
        piko.WithWatchdogProfileDirectory("/var/lib/piko/profiles"),
    ),
    piko.WithWatchdogProfileUploader(NewS3WatchdogUploader(s3Client, "piko-profiles")),
)
```

The uploader runs after the local file write completes. The local copy stays on disk for the configured retention. The notifier can include a remote URL by inspecting the upload result.

## Match the memory limit to the runtime

When running in a container with a cgroup memory limit, also wire `WithAutoMemoryLimit` so the GC observes the same ceiling the watchdog enforces:

```go
import "piko.sh/piko/wdk/system/system_memlimit_automemlimit"

piko.New(
    piko.WithAutoMemoryLimit(system_memlimit_automemlimit.Provider()),
    piko.WithMonitoring(
        // ...
        piko.WithMonitoringWatchdog(
            piko.WithWatchdogHeapThresholdPercent(0.85),
        ),
    ),
)
```

`system_memlimit_automemlimit.Provider()` reads the cgroup memory limit and sets `GOMEMLIMIT`. Without it, the heap threshold's percent applies to the absolute fallback (512 MiB by default), not the container's actual ceiling. See [How to set a memory limit automatically](automemorylimit.md).

## Inspect what the watchdog is doing

```bash
piko watchdog status                # current configuration and lifecycle
piko watchdog events --since 1h     # what fired in the last hour
piko watchdog list                  # stored profiles
piko watchdog history               # process start/stop ring (catches crash loops)
```

Or open the TUI: `piko tui` then navigate to the watchdog panels.

## See also

- [Watchdog API reference](../../reference/watchdog-api.md) for every option, event type, and CLI subcommand.
- [About the watchdog](../../explanation/about-the-watchdog.md) for the design rationale.
- [How to enable the monitoring endpoint](enable-monitoring.md) for the underlying transport.
- [How to capture continuous profiles](continuous-profiling.md) for routine captures alongside threshold-fired ones.
- [How to capture a contention diagnostic](contention-diagnostic.md) for block and mutex profiling.
- [How to set a memory limit automatically](automemorylimit.md) for the cgroup-aware `GOMEMLIMIT`.
