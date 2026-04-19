---
title: "001: Hello world"
description: A minimal Piko page with server-side rendering and scoped CSS
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 20
---

# 001: Hello world

A minimal Piko page that renders server-side data with scoped CSS. This is the simplest possible Piko application. A single `.pk` file demonstrates how the template, Go script, and style sections work together.

## What this demonstrates

- The three sections of a `.pk` file: `<template>`, `<script>`, and `<style>`
- Every `.pk` page must have a `Render` function and a `Response` struct
- `Render` receives `r *piko.RequestData` (the HTTP request) and `props piko.NoProps` (none for pages)
- It returns a `Response` (becomes `state` in the template), `piko.Metadata`, and an `error`
- Text interpolation with `{{ }}` syntax; HTML-escaped by default
- `piko.Metadata` for setting the page title and description
- The filename determines the URL path: `pages/index.pk` serves at `/index`
- CSS in the `<style>` section is automatically scoped to this page

## Project structure

```text
src/
  pages/
    index.pk             The page: template + Go script + CSS
```

## How to run this example

In the root directory of the Piko repository:

```bash
cd examples/scenarios/001_hello_world/src/
go mod tidy
air
```

## See also

- [PK file format reference](../reference/pk-file-format.md) for the structure used here.
- [Tutorials: Your first page](../tutorials/01-your-first-page.md) for the same pattern with explanation.
