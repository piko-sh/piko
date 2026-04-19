---
title: How to set a memory limit automatically
description: Use WithAutoMemoryLimit and the automemlimit provider so the Go GC respects the container's cgroup memory limit and the watchdog's percent thresholds align with the actual ceiling.
nav:
  sidebar:
    section: "how-to"
    subsection: "observability"
    order: 60
---

# How to set a memory limit automatically

In a containerised deployment, the cgroup imposes a memory limit the Go runtime does not see by default. The GC sizes itself against the host's memory and the heap can grow until the kernel sends `SIGKILL`. `WithAutoMemoryLimit` reads the cgroup limit and sets `GOMEMLIMIT` so the GC runs more aggressively as the heap approaches the ceiling. The watchdog's percent thresholds then resolve to the real limit instead of the absolute fallback. For the watchdog explanation see [about the watchdog](../../explanation/about-the-watchdog.md).

## Wire the provider

```go
package main

import (
    "piko.sh/piko"
    "piko.sh/piko/wdk/system/system_memlimit_automemlimit"
)

func main() {
    server := piko.New(
        piko.WithAutoMemoryLimit(system_memlimit_automemlimit.Provider()),
        // ... other options, including WithMonitoring(...) and the watchdog.
    )
    server.Run()
}
```

`system_memlimit_automemlimit.Provider()` returns a function that detects the cgroup limit and returns a byte count. Piko calls it at startup, sets `GOMEMLIMIT`, and logs the value. The provider applies a default ratio of `0.9` to the detected cgroup limit, so `GOMEMLIMIT` lands at 90% of the container ceiling. This leaves headroom for non-heap allocations (stacks, runtime metadata, cgo arenas) before the kernel out-of-memory kill. Override with `system_memlimit_automemlimit.Provider(system_memlimit_automemlimit.WithRatio(0.95))` if the workload tolerates a tighter margin. On a host without a cgroup limit the provider falls back to the host's memory. The GC behaviour matches the runtime default.

## Why this matters for the watchdog

`WithWatchdogHeapThresholdPercent(0.85)` interprets `0.85` against `GOMEMLIMIT`. Without `WithAutoMemoryLimit`, the threshold falls back to `WithWatchdogHeapThresholdBytes` (default 512 MiB). On a 4 GiB container, the provider applies its default `0.9` ratio first (so `GOMEMLIMIT` lands at 3.6 GiB), and the heap threshold then resolves to 0.85 of that:

| Configuration | `GOMEMLIMIT` | Effective heap threshold |
|---|---|---|
| `WithAutoMemoryLimit` set | 0.9 \* 4 GiB = 3.6 GiB | 0.85 \* 3.6 GiB ~= 3.06 GiB |
| Not set, default fallback | runtime default (no limit) | 512 MiB (unrelated to container size) |

The fallback fires constantly on a 4 GiB workload, drowning the alert channel. With `WithAutoMemoryLimit` the threshold tracks the actual ceiling, so the watchdog only captures when the process is genuinely near the out-of-memory kill.

The same logic applies to `WithWatchdogRSSThresholdPercent`, which always evaluates against the cgroup memory limit regardless of `GOMEMLIMIT`. The two thresholds together (heap and RSS) give a complete picture of memory pressure.

## What changes for the GC

Setting `GOMEMLIMIT` makes the GC pace itself toward the limit. As heap approaches the ceiling, GC frequency rises and GC cost rises with it. Behaviour to expect:

- Throughput drops slightly when heap is near `GOMEMLIMIT` because the GC runs more often.
- Out-of-memory kills become rare because the GC reclaims aggressively before the kernel limit hits.
- Peak heap stays below the ceiling instead of spiking past it momentarily.

The GC paces against the limit but does not enforce it absolutely. A workload that allocates faster than the GC can reclaim still hits the kernel out-of-memory kill. The watchdog's heap threshold catches that case earlier so the operator gets a profile.

## Verify the limit

After startup, check the log. Piko writes a structured `Auto memory limit applied` entry at info level with a `GOMEMLIMIT` field carrying the value in MiB. Rendered through a typical text encoder it looks like:

```text
Auto memory limit applied GOMEMLIMIT="3686 MiB"
```

The exact prefix and field separator depend on the configured logger encoder (text, JSON, console, OpenTelemetry export). The `Auto memory limit applied` line specifically prints the value in MiB. Other system-info paths may scale further. On a 4 GiB cgroup with the default 0.9 ratio, expect `3686 MiB` (3.6 GiB). Or query the running process from the TUI's system-info panel, or with:

```bash
piko info memory
```

`piko info memory` opens a gRPC connection to the monitoring server, so the application must be running with `WithMonitoring` (see [How to enable the monitoring endpoint](enable-monitoring.md)). Without `WithMonitoring` the command exits with a connection error before it can read the runtime memory state.

If the cgroup is unreadable or the kernel disables memory accounting, the provider falls back to `memlimit.FromSystem`, which returns total host RAM instead of the runtime's "no limit" sentinel. To detect that case, compare the logged `GOMEMLIMIT` against the container's known cgroup limit.

## Combine with explicit heap thresholds

When the workload genuinely runs above the auto-detected limit during normal operation (a memory-bound service that fills the heap deliberately), keep `WithAutoMemoryLimit` and loosen the watchdog's heap percent:

```go
piko.WithMonitoringWatchdog(
    piko.WithWatchdogHeapThresholdPercent(0.95),
)
```

The GC still paces against `GOMEMLIMIT`, but the watchdog only captures when heap is genuinely uncomfortably close to the ceiling.

## See also

- [Monitoring API reference](../../reference/monitoring-api.md) for the surrounding bootstrap surface.
- [Watchdog API reference](../../reference/watchdog-api.md) for the heap and RSS threshold options.
- [How to configure the watchdog](configure-watchdog.md) for thresholds and notifier wiring.
- [About the watchdog](../../explanation/about-the-watchdog.md) for the rationale behind percent thresholds.
