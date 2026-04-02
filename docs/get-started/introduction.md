---
title: Introduction
description: A website development kit for building server-side rendered applications with reactive client components
nav:
  sidebar:
    section: "get-started"
    subsection: "basics"
    order: 10
---

# Piko documentation

Piko is a website development kit for building server-side rendered web applications with interactive client-side components. It combines Go's performance and type safety with a Vue-inspired templating syntax for building fast, SEO-friendly web applications.

Beyond templating, Piko includes swappable backends for storage, caching, AI integration, and other services. You can start simple and swap implementations as you scale without changing application code.

Piko is built on a hexagonal architecture, so programmers can supply their own backends to extend what the framework supports.

## Preview & core features

Piko renders HTML on the server using Go. This results in fast initial page loads and excellent search engine optimisation. Pages are generated from `.pk` single file components that combine Go code with HTML templates.

This is an example PK file:

```piko
<template>
  <div class="container">
    <h1>HTTP Headers</h1>
    <ul>
      <li p-for="(key, value) in state.Headers" p-key="key">
        {{ key }}: {{ value }}
      </li>
    </ul>
  </div>
</template>

<script type="application/x-go">
  package main

  import "piko.sh/piko"

  type Response struct {
    Headers map[string]string
  }

  func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
      Headers: map[string]string{
        "Content-Type": "text/html",
        "X-Request-ID": "abc-123",
      },
    }, piko.Metadata{}, nil
  }
</script>

<style>
.container {
    font-family: monospace;
    ul {
        list-style: circle;
    }
}
</style>
```

You will notice that the elements of a PK file are very similar to a VueJS file, where there is a template block, and a script block. In a VueJS file the script block is in Javascript, and in Piko it is SSR, so it is in Go.

Piko is only inspired by VueJS and will often have clear differences, for example, the `p-for` above aligns more with the order of a `for` in Go. The expression language in templates is Piko's own and is in no way Javascript - we need an expression language that can be generated down to pure Go.

## Quick start

Create a new Piko project using the interactive CLI wizard:

```bash
# Install the Piko CLI
go install piko.sh/piko/cmd/piko@latest

# Run the project creation wizard
piko new

# Follow the prompts to configure your project
cd my-project
```

The wizard asks for your project name, location, and Go module path.

## Run the development server

You have two options for running the development server:

### Option 1: Using Air (recommended)

[Air](https://github.com/cosmtrek/air) provides live reloading when you modify Go files:

```bash
# Install Air (if not already installed)
go install github.com/air-verse/air@latest

# Start the development server with live reloading
air
```

### Option 2: Direct execution

Run the server directly without live reloading:

```bash
# Build assets for the first time
go run ./cmd/generator/main.go all

# Run server in dev mode
go run ./cmd/main/main.go dev
```

Your application is now running at `http://localhost:8080` (or next available port).

## Run modes

Piko supports three run modes:

| Mode    | Command                           | Description                                                  |
| ------- | --------------------------------- | ------------------------------------------------------------ |
| `dev`   | `go run ./cmd/main/main.go dev`   | Development mode with compiled templates                     |
| `dev-i` | `go run ./cmd/main/main.go dev-i` | Interpreted mode (heavily experimental, likely won't even work) |
| `prod`  | `go run ./cmd/main/main.go prod`  | Production mode with optimisations and disabling of development services |

The included `.air.toml` configuration uses `dev-i` mode for the fastest development feedback loop.

## Next steps

1. **[Your first page](/docs/get-started/first-page)** → Build your first Piko component
2. **[Core concepts](/docs/get-started/core-concepts)** → Understand the mental model behind Piko
3. **[PK file format](/docs/guide/pk-templates)** → Learn the template syntax and directives
