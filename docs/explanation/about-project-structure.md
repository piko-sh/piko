---
title: About project structure
description: Why a Piko project's tree looks the way it does, what the shape commits to, and where new code naturally lands as a project grows.
nav:
  sidebar:
    section: "explanation"
    subsection: "architecture"
    order: 80
---

# About project structure

Every Piko project has the same tree because Piko treats the filesystem as part of its public interface. Folder names are not preferences. They are the contract the generator reads. This page steps back from that contract. It considers why Piko picked this shape, what it gives up by doing so, and what it implies for how a project evolves.

For the literal layout, the deep links to each folder, and the table of "what does this directory contain", see [project structure in get-started](../get-started/project-structure.md). What follows here is reflection, not reference.

## Why the tree looks like this

The shape comes from one decision. Piko separates *what the developer writes* from *what Piko derives*, then it lets the filesystem express that separation directly.

Source folders (`pages/`, `partials/`, `components/`, `actions/`, plus `pkg/` for shared Go packages, `lib/` for static assets and SVG icons, `e2e/` for browser-test specs, and the optional `emails/`, `pdfs/`, `content/`, `locales/`, `db/`) hold the project's intent. Generator folders (`.piko/` and `dist/`) hold what the toolchain builds out of that intent. The split is rigid. The runtime never reads a source folder directly, and a developer never edits a generator folder by hand. A request at runtime resolves only against the artefacts the generator produced before the binary started.

The reason for that rigidity is determinism. If page parsing, action discovery, and template compilation all happened on the request path, a typo in `pages/about.pk` would surface as a 500 in production. By forcing every parse through the generator, Piko turns that class of mistake into a build-time failure. The cost is a build step. The benefit is that the runtime has nothing left to fail at, beyond the application logic the developer wrote on purpose.

A second decision falls out of the first. The two `cmd/` entry points exist because the generator must run before the runtime, and the two are not the same program. `cmd/generator/main.go` produces manifests. `cmd/main/main.go` consumes them. Splitting the entry points lets continuous integration build the manifests in one stage and ship a runtime image that contains only the server binary and its compiled output. A monolithic binary that did both would either bloat the production image or smear build-time concerns into request handling.

## What a flat tree commits to

Other web frameworks reach for an `app/` or `src/` parent. Piko deliberately does not. The reasoning is the same as for [file-based routing](about-routing.md). Visibility wins. `ls pages/` enumerates every route. `ls actions/` enumerates every action namespace. There is no top-level container to descend into and no "where did this file go again" search. The cost is that the project root is busier than a more namespaced layout. The benefit is that no folder name conceals project structure a newcomer has to learn.

A flat tree also commits to a particular ceiling on conceptual scope. A Piko project is a website plus its server. It is not a monorepo, not a workspace of subpackages, and not a microservice mesh. Code that does not fit one of the conventional folders ends up in `internal/` or `pkg/`, and from there the project relies on standard Go layout. Piko intentionally stops having opinions at that boundary. Piko rejected an alternative shape that provided a `domain/` or `services/` directory and codified its own application architecture. Piko's [hexagonal core](about-the-hexagonal-architecture.md) already supplies the seam between Piko and the application. A second seam at the directory level would duplicate the abstraction without adding signal.

## Comparison: flat-package versus deeply namespaced layouts

It is worth contrasting Piko's tree with two extremes the wider ecosystem favours.

A flat-package framework, in the Express or Flask tradition, asks the developer to own the layout. There is no opinion about where routes live, where templates live, or where business logic lives. Every project ends up with a slightly different convention. A developer joining a new project spends the first week learning that project's bespoke organisation. The strength of that approach is that it accommodates any application shape. The weakness is that two teams looking at two flat-package projects share none of the savings. Nothing transfers, every codebase is its own dialect.

A deeply namespaced framework, in the Spring or .NET tradition, goes the other way. It dictates `controllers/`, `services/`, `repositories/`, `viewmodels/`, `dtos/`, and a dozen other folders, each with its own naming convention and registration mechanism. The intent is that two teams looking at two such projects find them familiar on first look. The cost is that the framework has now committed to a particular application architecture, and projects pay that cost whether they need it or not. A simple landing page carries the same folder topology as a multi-team monolith.

Piko picks a middle position. It dictates the folders that correspond to Piko concepts, those it has to read at build time, and is silent about everything else. `pages/`, `actions/`, `components/` are not optional. They are Piko's interface. `internal/`, `pkg/`, and any application-specific subdivision are entirely the project's call. Two Piko projects therefore share the Piko-shaped folders and diverge wherever application code begins. That divergence is what application code is for.

## Build-time and runtime, kept apart

The tension every web toolkit navigates is the line between "what the developer configured" and "what the toolkit computed from it". A toolkit that puts both on the request path is fast to iterate on (every change is hot) and slow at runtime (every request re-derives state). A toolkit that pre-computes everything is fast at runtime and brittle to iterate on (every change is a rebuild). Piko biases toward the second, with a hot-reload escape hatch for development. See [about interpreted mode](about-interpreted-mode.md).

The directory structure encodes that bias. The project commits `dist/` instead of gitignoring it. `cmd/main/main.go` blank-imports it. Action `init()` registration runs at process start, before the first request. That import is load-bearing. The project would compile fine without `dist/`, but every action call would fall through to a 404 because no package would have registered with the global action registry. Committing the generator output, even in source control, makes the build deterministic for downstream consumers. A CI job that runs `go build` without first running the generator still produces a working binary.

That choice has a price. `dist/` produces churn in pull requests. Reviewers learn to ignore it the way they learn to ignore `package-lock.json`.

## Where new code goes, and why

A Piko project rarely faces ambiguous "where does this file belong" decisions, because each folder corresponds to a concept Piko already reasons about. A new server endpoint is an action, not a controller, so it goes in `actions/<package>/`. A new piece of interactive UI is a client component, so it goes in `components/`. A new shared template fragment is a partial.

The implication is that growth happens within folders, not by adding new top-level concepts. A two-page demo and a hundred-page commercial site have the same tree shape. The difference is the number of files inside each folder.

## See also

- [Project structure in get-started](../get-started/project-structure.md) for the literal layout and what each folder holds.
- [PK file format reference](../reference/pk-file-format.md) for the syntax of the source files the generator walks.
- [CLI reference](../reference/cli.md) for the generator commands that turn source folders into `dist/`.
- [About PK files](about-pk-files.md), [About the action protocol](about-the-action-protocol.md), and [About the hexagonal architecture](about-the-hexagonal-architecture.md) for the architectural choices the tree encodes.
