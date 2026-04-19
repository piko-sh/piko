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
    "piko.sh/piko/wdk/automemlimit"
)

func main() {
    server := piko.New(
        piko.WithAutoMemoryLimit(automemlimit.Provider()),
        // ... other options, including WithMonitoring(...) and the watchdog.
    )
    server.Run()
}
```

`automemlimit.Provider()` returns a function that detects the cgroup limit and returns a byte count. The framework calls it at startup, sets `GOMEMLIMIT`, and logs the value. On a host without a cgroup limit the provider returns the host's memory. The GC behaviour matches the runtime default.

## Why this matters for the watchdog

`WithWatchdogHeapThresholdPercent(0.85)` interprets `0.85` against `GOMEMLIMIT`. Without `WithAutoMemoryLimit`, the threshold falls back to `WithWatchdogHeapThresholdBytes` (default 512 MiB). On a 4 GiB container, this means:

| Configuration | Effective threshold |
|---|---|
| `WithAutoMemoryLimit` set | 85 % of 4 GiB = 3.4 GiB |
| Not set, default fallback | 512 MiB (unrelated to container size) |

The fallback fires constantly on a 4 GiB workload, drowning the alert channel. With `WithAutoMemoryLimit` the threshold tracks the actual ceiling, so the watchdog only captures when the process is genuinely near the out-of-memory kill.

The same logic applies to `WithWatchdogRSSThresholdPercent`, which always evaluates against the cgroup memory limit regardless of `GOMEMLIMIT`. The two thresholds together (heap and RSS) give a complete picture of memory pressure.

## What changes for the GC

Setting `GOMEMLIMIT` makes the GC pace itself toward the limit. As heap approaches the ceiling, GC frequency rises and GC cost rises with it. Behaviour to expect:

- Throughput drops slightly when heap is near `GOMEMLIMIT` because the GC runs more often.
- Out-of-memory kills become rare because the GC reclaims aggressively before the kernel limit hits.
- Peak heap stays below the ceiling instead of spiking past it momentarily.

The GC paces against the limit but does not enforce it absolutely. A workload that allocates faster than the GC can reclaim still hits the kernel out-of-memory kill. The watchdog's heap threshold catches that case earlier so the operator gets a profile.

## Verify the limit

After startup, check the log:

```text
INFO  GOMEMLIMIT set value=3.4GiB source=cgroup
```

Or query the running process from the TUI's system-info panel, or with:

```bash
piko info memory
```

A `GOMEMLIMIT` of `9223372036854775807` (the runtime's "no limit" sentinel) means the provider returned no value. Common causes include cgroup v1 environments without `memory.limit_in_bytes`, and a Linux kernel without cgroup memory accounting.

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
