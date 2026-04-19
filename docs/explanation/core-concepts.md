---
title: Core concepts
description: The mental model a Piko developer carries. Pages, components, actions, and collections explained together.
nav:
  sidebar:
    section: "explanation"
    subsection: "fundamentals"
    order: 20
---

# Core concepts

A Piko developer carries a short mental model. Pages render on the server. Components render on the client. Actions are the typed RPC in between. Collections turn content files into routes. Services plug into the application through interfaces. Everything compiles, and everything the compiler sees is a Go type.

<p align="center">
  <img src="../diagrams/core-concepts-map.svg"
       alt="Three lanes with wide gutters between them. Build time on the left chains content into piko generate, expands collections, emits .piko/generated.go, and runs go build. Server runtime in the centre holds pages, actions, and services behind ports; the binary at the bottom is the one go build produced. Browser runtime on the right hosts served HTML, PKC components, and the typed action call. A horizontal HTML arrow runs from server pages to browser served HTML in the upper gutter; a horizontal action RPC arrow runs from browser call back to server actions in the middle gutter."
       width="760"/>
</p>

This page traces each of those ideas and how they interlock. Individual pieces have their own explanation pages. This page is the overview.

## One language, one compiler, one binary

Piko is a Go framework. The server is Go, the template logic is Go, the build is `go build`. A Piko project ships as a single binary that contains the HTTP server, the compiled templates, the actions, and the assets. There is no second process, no adjacent Node runtime, no serialised API boundary between the render and the backend.

This is the load-bearing choice. Every other decision compounds on top of it. Types flow end-to-end because the types are Go types. Renaming a field is a compiler error because the template compiler has seen the struct. The cost is that everyone who touches the project writes Go, including the bits that feel like frontend work. The payoff is that the frontend bits are no longer separated from the domain types they display.

See [about PK files](about-pk-files.md) for the single-file rationale, and [about the hexagonal architecture](about-the-hexagonal-architecture.md) for how the binary stays testable.

## Two template formats, not one

The framework ships two template formats. PK files render on the server. PKC files render on the client. The choice of format is a choice of which side of the network owns the component.

A PK file compiles to Go. It runs inside the HTTP handler, reads request data, returns typed response data, and produces HTML. The browser receives the HTML and nothing else from that template.

A PKC file compiles to JavaScript wrapped in a Web Component. It ships to the browser alongside the HTML. It owns its own state, re-renders when state changes, and responds to events locally. The server sends the initial HTML once, and after that the PKC component runs on its own.

Both formats share one expression DSL (the Go-like syntax in `{{ ... }}` and directive attributes). The DSL compiles to Go in PK files and to JavaScript in PKC files. The syntax stays the same, and only the target differs.

This split is deliberate. [About reactivity](about-reactivity.md) explains why, and [about PK files](about-pk-files.md) covers the single-file shape both formats use.

## Actions are the RPC between them

When a PKC component needs to talk to the server, it calls a server action. An action is a Go struct that embeds `piko.ActionMetadata` and implements a `Call` method. The generator emits a TypeScript call shape so the client invokes the action as if it were local.

Actions are narrower than `http.Handler`. They are POST-only, they accept typed parameters or a typed input struct, and they return a typed response. The framework handles CSRF, rate limiting, and validation before `Call` runs. When the call would stream a long-running response, the action implements `StreamProgress` to emit Server-Sent Events instead of a single JSON body.

The action boundary is the only path between client and server in a Piko application. See [about the action protocol](about-the-action-protocol.md) for the reasoning.

## File-based routing, because files are the simplest truth

Routes come from the filesystem. A file at `pages/about.pk` serves `/about`. A file at `pages/blog/{slug}.pk` serves `/blog/:slug`. A file at `pages/admin/!404.pk` serves the 404 page for the `/admin` subtree. See [routing rules reference](../reference/routing-rules.md) for the filename-to-URL grammar.

The filesystem carries both the URL and the hierarchy. `ls pages/blog/` lists the blog routes. A request that fails points to exactly one file. No registration table, no route-ordering bugs, no reverse-engineering of middleware chains.

Two costs follow. Dynamic mounts and aliased URLs are awkward. Middleware composition works per-page through an optional `Middlewares()` function instead of in one central place. [About routing](about-routing.md) discusses the tradeoff.

## Collections promote data to routes

A page template that declares `p-collection="blog"` generates one route per item in the `blog` collection. The generator reads the collection at build time, creates routes, and makes each item's frontmatter available inside the page's `Render` function through `piko.GetData[T](r)`.

Collections answer a specific question. Where should a project keep "N pages with the same shape"? The blog, the docs site, the product catalogue, the team directory all fit this pattern. Without collections, these would live in code that registers routes from a data source at request time. With collections, they live in content directories that the generator sees at build time, and the output is one static route per item.

The provider model keeps the source pluggable. Markdown is the default. SQL, JSON APIs, and custom providers all fit the same interface. See [collections API reference](../reference/collections-api.md) for the provider contract and [how to collections/markdown](../how-to/collections/markdown.md) or [how to collections/custom providers](../how-to/collections/custom-providers.md) for worked setups. [About collections](about-collections.md) covers the build-time-versus-runtime tradeoff.

## Services hide behind interfaces

Piko applications consume services (storage, cache, email, LLM, notification, crypto, analytics, and more) through Go interfaces. The application wires concrete implementations at bootstrap via `With*` options. A test swaps in a mock. A production deploy swaps in S3, Redis, SES, or whatever else.

The hexagonal shape is pervasive. Every service the framework ships follows it, and custom services follow it too. Application code references the port (the interface), not the adapter. [About the hexagonal architecture](about-the-hexagonal-architecture.md) covers why the cost of upfront interface design is worth the downstream swappability.

## Typed props travel between components

Pages and partials pass data through typed props. A partial's script block defines a `Props` struct. A parent page instantiates the partial with `<piko:partial is="layout" :page_title="state.Title">`. The compiler checks that the parent sent the fields the partial declares.

PKC components accept props the same way. A parent page instantiates `<pp-counter start="10">`, and the PKC component's script block declares `const props = { start: 0 as number }` to receive them.

Props are the boundary between components in both formats. Inside a component, the owner holds state and props stay read-only. State mutation belongs to the component that owns the state.

## The mental model, in one pass

Files under `pages/` become routes. Each page file has a `Render` function that returns typed data. The template substitutes the data into HTML. If the page needs interactivity, a PKC component (a tag like `<pp-counter>`) sits inside the template. The PKC component ships its own JavaScript and runs in the browser. If the PKC component needs to reach the server, it calls an action. The action is a Go struct at `actions/<pkg>/<name>.go`. If the route list comes from content instead of code, the page declares `p-collection="x"` and the generator produces one route per item in `content/x/`. Everywhere Piko plugs into external systems, it does so through an interface registered at bootstrap.

That is the whole model. Every other concept in the docs is either a tooling detail (the CLI, the generator, the `piko.yaml` config) or a specialisation of one idea above. Scoped CSS, partial refresh, i18n, error pages, email templates, and PDF rendering all build on the same foundations.

## See also

- [About PK files](about-pk-files.md) for the single-file format rationale.
- [About reactivity](about-reactivity.md) for the PK/PKC split.
- [About the action protocol](about-the-action-protocol.md) for how client calls cross the wire.
- [About routing](about-routing.md) for the file-based rules.
- [About collections](about-collections.md) for the build-time generation model.
- [About the hexagonal architecture](about-the-hexagonal-architecture.md) for the service boundary.
- [How-to guides](/docs/how-to) for task recipes.
- [Reference](/docs/reference) for the API surface.
