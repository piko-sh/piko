---
title: "021: Live playground"
description: An in-browser PK playground backed by a WASM-compiled Piko runtime.
nav:
  sidebar:
    section: "showcase"
    subsection: "examples"
    order: 410
---

# 021: Live playground

A single-page playground where users write PK templates in the browser and see the rendered output immediately. The rendering happens entirely client-side. The build compiles the Piko runtime to WebAssembly, the page loads it once, and the runtime renders whatever the user types.

## What this demonstrates

- A PKC component that embeds a code editor and a preview pane.
- A WebAssembly build of the Piko template compiler, served alongside the site.
- Shadow-DOM isolation so the rendered output does not leak into the surrounding page.
- A secondary HTTP server that serves WASM artefacts with the correct MIME type and CSP.

## Project structure

```text
src/
  cmd/main/main.go               Bootstrap; also serves WASM artefacts on a second port.
  components/
    pp-playground.pkc            The playground component: editor + WASM runtime + preview.
  pages/
    index.pk                     Hosts the playground component.
  bin/wasm/                      Pre-built WASM artefacts loaded by the component.
```

## How to run this example

From the Piko repository root:

```bash
cd examples/scenarios/021_live_playground/src/
go mod tidy
# Build the WASM artefacts (see scenario README for the exact command).
air
```

## See also

- [Client components reference](../reference/client-components.md).
- [Runnable source](https://github.com/piko-sh/piko/tree/master/examples/scenarios/021_live_playground).
