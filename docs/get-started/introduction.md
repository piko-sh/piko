---
title: Introduction
description: Piko is a website development kit for Go that builds server-side-rendered web applications with reactive client components and swappable backends.
nav:
  sidebar:
    section: "get-started"
    subsection: "overview"
    order: 10
---

# Piko documentation

Piko is a website development kit for building server-side-rendered web applications with interactive client-side components ([about SSR](../explanation/about-ssr.md), [about reactivity](../explanation/about-reactivity.md)). It combines Go's performance and type safety with a Vue-inspired templating syntax ([about Piko, Vue, and Nuxt](../explanation/about-piko-vs-vue.md)). The hexagonal architecture lets a project swap storage, caching, email, or AI backends without touching application code ([about the hexagonal architecture](../explanation/about-the-hexagonal-architecture.md)).

<p align="center">
  <img src="../diagrams/one-binary.svg"
       alt="Before and after architecture. On the left a traditional split: a Go API and a Next.js frontend connected across a CORS boundary, deployed as two services. On the right a single Piko binary containing both concerns, deployed once."
       width="540"/>
</p>

A Piko project ships as one Go binary. The HTTP server, compiled templates, actions, and assets all live inside that single file, which `go build` produces.

## A quick look

A single PK file contains template, Go, and CSS. Piko compiles the template against the Go types declared in the file and serves the rendered HTML.

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

A PK file looks similar to a Vue.js single-file component, with a template block and a script block. The script is Go (not JavaScript), and the expression language inside `{{ }}` compiles to pure Go, so no code runs on the client. The [PK file format reference](../reference/pk-file-format.md) documents every section. [Concepts](concepts.md) walks through every other named piece (PKC components, actions, partials, collections, the querier, services, i18n). [Install and run](install.md) gets a server running in under five minutes.

## Where to read next

This folder holds four onboarding pages. Introduction, install, concepts, and project structure. After reading them, the rest of the documentation splits into four areas:

- **Tutorials** teach Piko step by step. Start with [Your first page](../tutorials/01-your-first-page.md) after the dev server is running.
- **How-to guides** answer "how do I?" for specific tasks: routing, forms, collections, i18n, testing, deployment.
- **Reference** documents every public API, directive, file format, and configuration option.
- **Explanation** answers "why": the rendering model, the action protocol, the hexagonal architecture.

## Requirements

Piko targets Go 1.26 or later. See [Install and run](install.md) for the full setup steps.
