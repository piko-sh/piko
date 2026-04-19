---
title: About WebAssembly in Piko
description: What a WASM Piko build can and cannot do, and where the split belongs.
nav:
  sidebar:
    section: "explanation"
    subsection: "architecture"
    order: 90
---

# About WebAssembly in Piko

Piko's WASM target is narrower than a typical Go build for the browser. A WASM binary ships the template engine and the bytecode interpreter, but it does not ship the server framework. This page explains where the line sits, why it sits there, and what WASM Piko is good for.

<p align="center">
  <img src="../diagrams/wasm-scope.svg"
       alt="Two boxes. Inside WASM contains the template parser, bytecode interpreter, and PK-to-HTML renderer; the use case is client-side template evaluation. Outside WASM contains the piko.sh/piko package, ActionMetadata, RequestData, storage, cache, email, LLM, PDF, notification, and database drivers; these stay server-side."
       width="600"/>
</p>

## What sits inside a WASM build

A WASM build parses and compiles PK templates at runtime. It runs the same bytecode interpreter Piko uses in development mode for server-side rendering. It produces HTML strings the host page can inject into the DOM.

The use cases fall into one shape. Client-side template evaluation. Live editors, dashboards where users compose their own layouts, interactive tutorials that render PK snippets, and playgrounds that preview arbitrary templates are all natural fits. Each wants to evaluate a template the server did not pre-compile, with data the server did not see, and wants to stay responsive without a round-trip.

## What sits outside

A WASM build excludes the `piko.sh/piko` package itself. The `piko-symbols-runtime.yaml` file carries a `!js` build tag on the runtime symbols. A WASM binary therefore has no `ActionMetadata`, no `RequestData`, no storage provider, no email provider, no cache service, no LLM service, no PDF renderer, no database driver. None of the hexagonal services ship into the browser.

Calls to server actions, health endpoints, and the orchestrator are also unavailable. Those are server-side concerns by design.

## Why the split is where it is

Server primitives assume a server. `RequestData` reads from an `http.Request`. `ActionMetadata` wires into the CSRF and rate-limit layers. Storage providers open network sockets. Shipping any of them into the browser would either need a compatibility shim that fakes the semantics or a second, browser-shaped version of each interface. Both paths multiply the surface area and divide what a "Piko app" means.

The narrow WASM scope keeps the template engine portable while keeping the rest of the framework honest about where it runs. The consequence is that WASM Piko is useful for template evaluation specifically, not for running a shrunken server in the browser.

## When to reach for WASM

Reach for a WASM build when the browser needs to render a template the server did not already render. The [live playground scenario](../showcase/021-live-playground.md) shows the reference shape. The user types a template, the browser compiles and runs it, and the host page displays the output without contacting the server.

Do not reach for WASM when the goal is "run the app in the browser". A Piko application is a server plus PK templates plus PKC components. The browser already runs PKC components (they compile to JavaScript). The server part belongs on a server.

## See also

- [How to compile Piko to WebAssembly](../how-to/wasm.md) for the build and serving steps.
- [Scenario 021: live playground](../showcase/021-live-playground.md) for a runnable example.
- [About reactivity](about-reactivity.md) for the wider PK/PKC split that motivates the WASM scope.
