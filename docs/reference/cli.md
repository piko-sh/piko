---
title: CLI reference
description: Every piko subcommand, flag, and usage pattern.
nav:
  sidebar:
    section: "reference"
    subsection: "tooling"
    order: 10
---

# CLI reference

The `piko` CLI scaffolds new projects, formats and inspects `.pk` files, profiles running servers, and queries the monitoring endpoint. This page lists every subcommand. Source of truth: [`cmd/piko/main.go`](https://github.com/piko-sh/piko/blob/master/cmd/piko/main.go) and [`cmd/piko/internal/cli/`](https://github.com/piko-sh/piko/tree/master/cmd/piko/internal/cli).

## Usage

```
piko [subcommand] [flags]
```

Run `piko help` for a condensed list. Every subcommand accepts `--help` for its own usage.

## Project commands

| Command | Description |
|---|---|
| `piko new` | Create a new Piko project through the interactive wizard. |
| `piko fmt` | Format `.pk` template files. Supports `-w` (write), `-r` (recurse), and file or directory arguments. |
| `piko extract` | Extract Go package symbols for the bytecode interpreter. |
| `piko inspect <type> <file>` | Inspect FlatBuffers binary files. Types: `manifest`, `i18n`, `collection`, `search`, `bytecode`. |
| `piko bytecode` | Inspect and analyse compiled bytecode files (sub-subcommands: `inspect`, `disassemble`). |
| `piko agents` | Configure AI coding tools with Piko knowledge (Claude Code, Codex, Cursor, etc.). |
| `piko profile <url>` | Profile a live server under load: CPU, memory, mutex, and blocking profiles. Supports `--focus` to scope results. See [how to profiling](../how-to/profiling.md) for interpretation. |

## Monitoring commands

Monitoring commands connect to the gRPC monitoring server (default `127.0.0.1:9091`) that a running Piko server exposes when started with [`WithMonitoring`](bootstrap-options.md#monitoring).

| Command | Description |
|---|---|
| `piko get <resource>` | Display a resource: `health`, `tasks`, `workflows`, `artefacts`, `variants`, `metrics`, `traces`, `resources`, `dlq`, `ratelimiter`. |
| `piko describe <resource> [id]` | Show detailed information for one resource entry. |
| `piko info [area]` | Display system information. Areas: `system`, `build`, `runtime`, `memory`, `gc`, `process`. |
| `piko watch <resource>` | Stream resource updates in real time. Supports `--interval` (default `2s`). |
| `piko profiling enable <duration>` | Enable on-demand profiling for a window. |
| `piko profiling status` | Report whether profiling is active. |
| `piko profiling capture <kind> [duration]` | Capture a specific profile: `heap`, `cpu`, `goroutine`, `mutex`, `block`, `allocs`. |
| `piko profiling disable` | Disable on-demand profiling. |
| `piko diagnostics` | Test connectivity to the monitoring server. |
| `piko tui` | Launch the interactive terminal UI. |
| `piko watchdog status` | Print the watchdog's lifecycle, thresholds, and continuous-profiling configuration. |
| `piko watchdog list [--type <t>]` | List stored watchdog profiles. |
| `piko watchdog download [<file> \| --latest --type <t>]` | Download a stored profile (and its sidecar JSON) to a local directory. |
| `piko watchdog prune [--type <t>]` | Remove stored watchdog profiles. |
| `piko watchdog history` | Print the startup-history ring; an unclean stop indicates a crash. |
| `piko watchdog events [--since <d>] [--type <t>] [--tail]` | List or stream watchdog events from the in-memory ring. |
| `piko watchdog contention-diagnostic` | Run a one-shot block and mutex contention diagnostic. |

## Other commands

| Command | Description |
|---|---|
| `piko version` | Show the CLI version. |
| `piko help` | Show the usage message. |

## Global flags (monitoring commands)

| Flag | Default | Purpose |
|---|---|---|
| `-e`, `--endpoint` | `127.0.0.1:9091` | gRPC monitoring server address. |
| `-o`, `--output` | `table` | Output format: `table`, `wide`, `json`. |
| `-n`, `--limit` | none | Maximum number of items to return. |
| `-t`, `--timeout` | `5s` | Connection and request timeout. |
| `--no-colour` | off | Disable coloured output. |
| `--raw` | off | Alias for `--no-colour`. |
| `--no-headers` | off | Omit table headers from output. |

## Examples

```bash
# Project bootstrap
piko new
piko fmt -w -r ./components

# Monitoring
piko get health
piko get health Liveness
piko get tasks -n 10
piko get health -o wide
piko get health -o json
piko get tasks --no-headers
piko describe health
piko describe task <id>
piko info memory
piko watch health --interval 2s

# Dispatcher dead-letter queue
piko get dlq
piko get dlq email -n 10

# Diagnostics and TUI
piko diagnostics
piko tui

# AI tooling integration
piko agents install

# Binary inspection
piko inspect manifest dist/manifest.bin
piko bytecode inspect dist/pages/x/bytecode-y.bin

# Live-server profiling
piko profile http://localhost:8080/
piko profile http://localhost:8080/ --focus "render"

# On-demand runtime profiling
piko profiling enable 30m
piko profiling status
piko profiling capture heap
piko profiling capture cpu 30s
piko profiling disable

# Watchdog
piko watchdog status
piko watchdog events --since 1h
piko watchdog events --tail
piko watchdog list --type heap
piko watchdog download --latest --type heap --output ./pprof
piko watchdog history
piko watchdog contention-diagnostic
```

## See also

- [Bootstrap options reference](bootstrap-options.md) for the server-side options that the monitoring commands read.
- [Monitoring API reference](monitoring-api.md) for the gRPC transport that the `piko get`, `piko watch`, `piko describe`, `piko tui`, and `piko watchdog` commands consume.
- [Watchdog API reference](watchdog-api.md) for the watchdog options and event types referenced by the `piko watchdog` subcommands.
- [Health API reference](health-api.md) for the endpoints `piko get health` and `piko describe health` query.
- [How to profiling](../how-to/profiling.md) for worked flows using `piko profile` and `piko profiling`.
- [How to production build](/docs/how-to/production-build) for `piko generate` and `piko fmt` in a release pipeline.
- Source: [`cmd/piko/`](https://github.com/piko-sh/piko/tree/master/cmd/piko).
