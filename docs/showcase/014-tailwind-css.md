---
title: "014: Tailwind CSS"
description: Integrate Tailwind CSS with a Piko project via a shared styles library.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 260
---

# 014: Tailwind CSS

A Piko project using Tailwind CSS for styling. The example shows how to include Tailwind's utility classes inside scoped `<style>` blocks and share a common utility library across partials and pages.

## What this demonstrates

- Including Tailwind CSS inside `.pk` `<style>` blocks alongside scoped CSS.
- A shared `lib/` folder for common styling utilities.
- Piko's CSS reset working alongside Tailwind.

## Project structure

```text
src/
  cmd/main/main.go      Bootstrap with WithCSSReset and dev-mode options.
  pages/                Pages using Tailwind utilities.
  partials/             Shared partials.
  lib/                  Shared styling helpers.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/014_tailwind_css/src/
go mod tidy
air
```

## See also

- [Bootstrap options reference](../reference/bootstrap-options.md).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/014_tailwind_css).
