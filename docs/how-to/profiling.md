---
title: How to profile a Piko application
description: Capture CPU, heap, goroutine, and trace profiles from a running Piko server or a generator build.
nav:
  sidebar:
    section: "how-to"
    subsection: "operations"
    order: 11
---

# How to profile a Piko application

Capture CPU, allocation, and trace profiles from a running Piko server or from a generator build, then analyse them with `go tool pprof`. See the [CLI reference](../reference/cli.md) for the `piko profile` command this guide uses.

## Enabling profiling

### Server profiling (long-running applications)

Add `WithProfiling()` to your application options. This starts a dedicated pprof HTTP server on `localhost:6060` during bootstrap and stops it during graceful shutdown.

```go
server := piko.New(
    piko.WithProfiling(),
)
```

The profiling server listens on **localhost only** by default. Piko groups all endpoints under the `/_piko/` prefix, making it straightforward to block in production via reverse-proxy rules.

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
| `WithProfilingRollingTraceMinAge` | `15s` when `WithProfilingRollingTrace()` activates rolling trace | Retention target for the rolling trace window |
| `WithProfilingRollingTraceMaxBytes` | `16 MiB` when `WithProfilingRollingTrace()` activates rolling trace | Memory budget hint for the rolling trace buffer |

Once you enable it, Piko serves the standard pprof endpoints at:

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

When you enable rolling trace capture, Piko also exposes:

```text
http://localhost:6060/_piko/profiler/status        # profiler capabilities / metadata
http://localhost:6060/_piko/profiler/trace/recent  # rolling trace snapshot download
```

The rolling trace endpoint complements `/debug/pprof/trace`. It lets you dump the most recent in-memory trace window on demand, so you do not have to start a new trace at the moment you notice a problem.

### Generator profiling (build-time)

For profiling the static site generator (short-lived build process), use `WithGeneratorProfiling()`. This captures CPU, trace, heap, block, mutex, goroutine, and allocs profiles to disk instead of starting an HTTP server.

```go
server := piko.New(
    piko.WithGeneratorProfiling(),
)
```

Piko writes profiles to `./profiles/` by default. Each profile type produces a separate `.pprof` file that you can analyse with `go tool pprof`.

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

The `piko profile` CLI command automates the entire profiling workflow. It generates sustained HTTP load against your running application while capturing pprof profiles from the profiling server.

It keeps the existing delta-snapshot behaviour for allocation-oriented analysis. When available, it auto-detects profiler capabilities from `/_piko/profiler/status` so it can save extra metadata and rolling trace artefacts without changing the core pprof flow.

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
| `--pprof-port` | `6060` | Port where the server exposes pprof endpoints |
| `--concurrency` | `100` | Number of concurrent HTTP connections |
| `--duration` | `30` | Phase duration in seconds |
| `--output` | `pprof` | Output directory for profiles and report |
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

Durations below 30 seconds produce unreliable CPU profile statistics. The CLI warns you when you pass `--duration` below this threshold. Quick sanity checks tolerate shorter runs, but for actionable data use at least 30 seconds.

### TUI dashboard

The `--tui` flag enables a live terminal dashboard that shows:

| Panel | Content |
|---|---|
| Phase progress | Which profiling phase is active |
| Requests per second | Sparkline of throughput over time |
| Latency | Sparkline of mean response time with percentile breakdown (p50, p80, p99, p100) |
| Goroutine count | Sparkline of active goroutines, useful for spotting leaks |
| Totals | Cumulative requests, failures, and bytes received |

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

`profiler_status.json` records which profiler features were available during the run. `rolling_trace.out` is a raw Go execution trace that you can inspect with `go tool trace`.

The text report lists top offenders. For interactive analysis, use Go's built-in tools:

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

- The profiling server defaults to **localhost only**. It stays off the network unless you explicitly set `WithProfilingBindAddress("0.0.0.0")`.
- Piko groups profiling endpoints under `/_piko/` so a single reverse-proxy rule (for example `deny /_piko/`) blocks them all.
- Profiling endpoints expose internal application state. **Never expose them to untrusted networks in production.**
- For production diagnostics, enable profiling temporarily with localhost binding and use SSH tunnelling or `kubectl port-forward` to reach it.

## Build flag detection

When you enable profiling, Piko checks whether the build used optimisation-disabling flags (`-l` or `-N` in `-gcflags`). If it detects them, Piko logs a warning because unoptimised binaries produce misleading profile data. Functions that would normally inline appear as separate call sites, skewing CPU and allocation attribution.

Build with default optimisations for accurate profiling:

```bash
go build ./cmd/main
```

Not with:

```bash
go build -gcflags="all=-N -l" ./cmd/main
```

## See also

- [CLI reference](../reference/cli.md) for the `piko profile` command and its flags.
- [Monitoring how-to](deployment/monitoring.md) for always-on observability.
- [Bootstrap options reference](../reference/bootstrap-options.md) for `WithProfiling` and `WithGeneratorProfiling`.
