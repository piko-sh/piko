---
title: Install and run
description: Install the Piko CLI, scaffold a project, and start the dev server in under five minutes.
nav:
  sidebar:
    section: "get-started"
    subsection: "overview"
    order: 20
---

# Install and run

This page gets a Piko project running on your machine. Expect five minutes from a clean shell to a live dev server. For what to edit first once the server is up see [Your first page](../tutorials/01-your-first-page.md).

## Prerequisites

Piko targets Go 1.26 or later. Check your version with `go version`. If you do not have Go, install it from [go.dev/dl](https://go.dev/dl/).

## Install the CLI

```bash
go install piko.sh/piko/cmd/piko@latest
```

Confirm the install:

```bash
piko version
```

The binary lands in `$GOPATH/bin` (usually `~/go/bin`). Add that to your `PATH` if `piko version` cannot find the command. For every subcommand the CLI supports see the [CLI reference](../reference/cli.md).

## Create a project

```bash
piko new
```

The interactive wizard asks for a project name, location, and Go module path. Answer the prompts and step into the created directory:

```bash
cd my-project
```

The scaffolded tree contains the folders a typical Piko project uses (`pages/`, `components/`, `actions/`, `partials/`, `pkg/`, `lib/icons/`, `dist/`, `e2e/`, `cmd/`, `internal/`). See [Project structure](project-structure.md) for a folder-by-folder map.

## Install Air for live reload (optional)

[Air](https://github.com/air-verse/air) rebuilds the binary and restarts the server when source files change. Recommended for day-to-day development:

```bash
go install github.com/air-verse/air@latest
```

Piko scaffolds an `.air.toml` configuration so Air knows which files to watch and which command to run.

## Start the dev server

With Air:

```bash
air
```

Without Air:

```bash
go run ./cmd/generator/main.go all
go run ./cmd/main/main.go dev
```

Either form listens on `http://localhost:8080` (or the next free port). Open that URL in a browser to confirm the server is up.

## Run modes

| Mode | Command | Purpose |
|---|---|---|
| `dev` | `go run ./cmd/main/main.go dev` | Development mode with compiled templates. The scaffolded bootstrap enables the dev widget and hot-reload. |
| `dev-i` | `go run ./cmd/main/main.go dev-i` | Interpreted mode. For testing the interpreter pipeline, not the default fast-feedback dev mode. The scaffolded bootstrap enables the dev widget and hot-reload here too, and requires `ssr.WithSymbols(...)` registration in `cmd/main/main.go`. See [about interpreted mode](../explanation/about-interpreted-mode.md). |
| `prod` | `go run ./cmd/main/main.go prod` | Production mode. Dev widgets off. |

If `cmd/main/main.go` runs without an argument it falls back to `dev`. The scaffolded `.air.toml` points at `dev` by default. To wire up `dev-i`, see [about interpreted mode](../explanation/about-interpreted-mode.md) for the bootstrap setup and [runtime symbols reference](../reference/runtime-symbols.md) for exposing symbols.

## What to do next

With the server running:

- Read [Concepts](concepts.md) for the Piko vocabulary: PK files, PKC components, actions, partials, collections, the querier, services, i18n.
- Start [Your first page](../tutorials/01-your-first-page.md) to make your first edit with narrative guidance.

For the full CLI surface (every subcommand, flag, and usage pattern) see the [CLI reference](../reference/cli.md).
