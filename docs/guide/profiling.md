---
title: Profiling
description: Profiling Piko applications with pprof and the piko CLI
nav:
  sidebar:
    section: "guide"
    subsection: "best-practices"
    order: 11
---

# Profiling

Piko provides built-in profiling support through Go's `pprof` tooling. There are two modes: **server profiling** for long-running applications and **generator profiling** for short-lived build processes. The `piko profile` CLI command ties everything together by load-testing your application and collecting profiles in one step.

## Enabling profiling

### Server profiling (long-running applications)

Add `WithProfiling()` to your application options. This starts a dedicated pprof HTTP server on `localhost:6060` during bootstrap and stops it during graceful shutdown.

```go
server := piko.New(
    piko.WithProfiling(),
)
```

The profiling server listens on **localhost only** by default. All endpoints are grouped under the `/_piko/` prefix, making it straightforward to block in production via reverse-proxy rules.

Available sub-options:

```go
server := piko.New(
    piko.WithProfiling(
        piko.WithProfilingPort(7070),                // default: 6060
        piko.WithProfilingBindAddress("0.0.0.0"),     // default: "localhost"
        piko.WithProfilingBlockRate(500),              // default: 1000 (nanoseconds)
        piko.WithProfilingMutexFraction(5),            // default: 10 (1/n events)
        piko.WithProfilingMemProfileRate(4096),        // default: 0 (runtime default, 512KB)
        piko.WithProfilingRollingTrace(),              // default: disabled
        piko.WithProfilingRollingTraceMinAge(30*time.Second),
        piko.WithProfilingRollingTraceMaxBytes(32*1024*1024),
    ),
)
```

| Sub-option | Default | Description |
|---|---|---|
| `WithProfilingPort` | `6060` | HTTP port for the pprof server |
| `WithProfilingBindAddress` | `"localhost"` | Network address to bind to |
| `WithProfilingBlockRate` | `1000` | Block profiling granularity in nanoseconds |
| `WithProfilingMutexFraction` | `10` | Fraction of mutex events reported (1/n) |
| `WithProfilingMemProfileRate` | `0` | Memory sample rate in bytes (0 = runtime default) |
| `WithProfilingRollingTrace` | disabled | Enables a bounded in-memory rolling execution trace |
| `WithProfilingRollingTraceMinAge` | `15s` when rolling trace is enabled via `WithProfilingRollingTrace()` | Retention target for the rolling trace window |
| `WithProfilingRollingTraceMaxBytes` | `16 MiB` when rolling trace is enabled via `WithProfilingRollingTrace()` | Memory budget hint for the rolling trace buffer |

Once enabled, the standard pprof endpoints are available at:

```text
http://localhost:6060/_piko/debug/pprof/          # index
http://localhost:6060/_piko/debug/pprof/profile   # CPU profile
http://localhost:6060/_piko/debug/pprof/heap      # heap profile
http://localhost:6060/_piko/debug/pprof/allocs    # allocation profile
http://localhost:6060/_piko/debug/pprof/block     # blocking profile
http://localhost:6060/_piko/debug/pprof/mutex     # mutex contention profile
http://localhost:6060/_piko/debug/pprof/goroutine # goroutine dump
http://localhost:6060/_piko/debug/pprof/trace     # execution trace
```

When rolling trace capture is enabled, Piko also exposes:

```text
http://localhost:6060/_piko/profiler/status        # profiler capabilities / metadata
http://localhost:6060/_piko/profiler/trace/recent  # rolling trace snapshot download
```

The rolling trace endpoint is complementary to `/debug/pprof/trace`: it lets you dump the most recent in-memory trace window on demand without having to start a new trace at the moment you notice a problem.

### Generator profiling (build-time)

For profiling the static site generator (short-lived build process), use `WithGeneratorProfiling()`. This captures CPU, trace, heap, block, mutex, goroutine, and allocs profiles to disk rather than starting an HTTP server.

```go
server := piko.New(
    piko.WithGeneratorProfiling(),
)
```

Profiles are written to `./profiles/` by default. Each profile type produces a separate `.pprof` file that can be analysed with `go tool pprof`.

Available sub-options:

```go
server := piko.New(
    piko.WithGeneratorProfiling(
        piko.WithGeneratorProfilingOutputDir("/tmp/profiles"),
        piko.WithGeneratorProfilingBlockRate(1),
        piko.WithGeneratorProfilingMutexFraction(1),
        piko.WithGeneratorProfilingMemProfileRate(4096),
    ),
)
```

## Using `piko profile`

The `piko profile` CLI command automates the entire profiling workflow: it generates sustained HTTP load against your running application while capturing pprof profiles from the profiling server.

It keeps the existing delta-snapshot behaviour for allocation-oriented analysis and, when available, auto-detects profiler capabilities from `/_piko/profiler/status` so it can save additional metadata and rolling trace artifacts without changing the core pprof flow.

### Prerequisites

1. Your application must have `WithProfiling()` enabled
2. Your application must be running

### Basic usage

```bash
piko profile http://localhost:8080/
```

This runs a baseline phase followed by CPU, allocs, heap, mutex, and block profiling phases (each 30 seconds by default), then produces a text report and raw `.pprof` files in `./pprof/`.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--pprof-port` | `6060` | Port where pprof endpoints are exposed |
| `--concurrency` | `100` | Number of concurrent HTTP connections |
| `--duration` | `30` | Phase duration in seconds |
| `--output` | `./pprof` | Output directory for profiles and report |
| `--header` | - | HTTP header to send with requests (repeatable) |
| `--cookie` | - | Cookie header shorthand |
| `--top` | `60` | Number of top entries per report section |
| `--focus` | - | Regex filter to focus on matching function names |
| `--tui` | `false` | Enable live terminal dashboard |

### Examples

Profile with the live TUI dashboard:

```bash
piko profile http://localhost:8080/ --tui
```

Profile with higher concurrency and longer duration:

```bash
piko profile http://localhost:8080/ --concurrency 200 --duration 60
```

Profile a specific page with authentication:

```bash
piko profile http://localhost:8080/dashboard \
    --header "Authorization: Bearer token123" \
    --duration 30
```

Focus the report on rendering functions:

```bash
piko profile http://localhost:8080/ --focus "render"
```

Use cookies for session-based auth:

```bash
piko profile http://localhost:8080/ --cookie "session_id=abc123"
```

### Duration warning

Durations below 30 seconds produce unreliable CPU profile statistics. The CLI warns you when `--duration` is set below this threshold. For quick sanity checks this is acceptable, but for actionable data use at least 30 seconds.

### TUI dashboard

The `--tui` flag enables a live terminal dashboard that shows:

- **Phase progress**: which profiling phase is active
- **Requests per second**: sparkline of throughput over time
- **Latency**: sparkline of mean response time with percentile breakdown (p50, p80, p99, p100)
- **Goroutine count**: sparkline of active goroutines (useful for spotting leaks)
- **Totals**: cumulative requests, failures, and bytes received

### Analysing results

After profiling completes, the output directory contains:

```text
./pprof/
  cpu.pprof
  heap.pprof
  allocs.pprof
  block.pprof
  mutex.pprof
  baseline.goroutines.txt
  cpu.goroutines.txt
  allocs.goroutines.txt
  heap.goroutines.txt
  mutex.goroutines.txt
  block.goroutines.txt
  baseline.stats
  cpu.pprof.stats
  allocs.pprof.stats
  heap.pprof.stats
  mutex.pprof.stats
  block.pprof.stats
  profiler_status.json      # when the status endpoint is available
  rolling_trace.out         # when rolling trace capture is enabled
  live_performance_report.txt
```

`profiler_status.json` records which profiler features were available during the run. `rolling_trace.out` is a raw Go execution trace that can be inspected with `go tool trace`.

The text report highlights top offenders. For interactive analysis, use Go's built-in tools:

```bash
# Interactive web UI for CPU profile
go tool pprof -http=:8081 ./pprof/cpu.pprof

# Interactive web UI for allocation profile
go tool pprof -http=:8081 ./pprof/allocs.pprof

# Compare two profiles (before/after optimisation)
go tool pprof -diff_base=./before/cpu.pprof ./after/cpu.pprof

# Inspect the rolling trace snapshot
go tool trace ./pprof/rolling_trace.out
```

## Security considerations

- The profiling server defaults to **localhost only**: it is not accessible from the network unless you explicitly set `WithProfilingBindAddress("0.0.0.0")`
- All profiling endpoints are grouped under `/_piko/` so they can be blocked with a single reverse-proxy rule (e.g. `deny /_piko/`)
- Profiling endpoints expose internal application state: **never expose them to untrusted networks in production**
- For production diagnostics, enable profiling temporarily with localhost binding and use SSH tunnelling or `kubectl port-forward` to access it

## Build flag detection

When profiling is enabled, Piko checks whether the binary was built with optimisation-disabling flags (`-l` or `-N` in `-gcflags`). If detected, a warning is logged because unoptimised binaries produce misleading profile data: functions that would normally be inlined appear as separate call sites, skewing CPU and allocation attribution.

Build with default optimisations for accurate profiling:

```bash
go build ./cmd/main
```

Not with:

```bash
go build -gcflags="all=-N -l" ./cmd/main
```
