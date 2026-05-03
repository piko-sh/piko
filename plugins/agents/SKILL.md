---
name: piko
description: "Build websites with Piko - a Go-based server-side rendering framework with single-file components (.pk), client-side Web Components (.pkc), file-based routing, partials with slots, server actions, i18n, and integrated tooling. Use when generating, reviewing, or refactoring Piko templates, components, pages, partials, actions, or project configuration."
---

# Piko: Website Development Workflow

Run this workflow top to bottom.

## 1) Capture project context

Record before writing code:
- Go module path and Piko version
- existing directory layout (pages, partials, components, actions)
- whether the project uses collections, i18n, or custom themes
- target deployment (dev server with Air, production build, Docker)

If unknown, state assumptions explicitly.

## 2) Classify the task

Select the primary category:
- new page or route
- new or modified partial (with props/slots)
- new or modified client component (.pkc)
- server action (form handling, CRUD, SSE streaming, debouncing, optimistic updates)
- SPA navigation (piko:a), optimised images (piko:img), event bus
- page caching (CachePolicy)
- styling or theming change
- i18n / localisation
- project scaffolding or configuration
- content collection
- WDK service integration (data, email, events, crypto, media, LLM)
- testing (component tests, action tests, E2E browser tests, benchmarks)
- deployment or infrastructure (Docker, K8s, observability)

## 3) Load only the relevant reference file(s)

Primary references (load for most tasks):
- `references/pk-file-format.md` - .pk structure, script, template, style, i18n
- `references/pkc-components.md` - .pkc Web Components, reactive state, attribute sync
- `references/template-syntax.md` - directives, interpolation, expressions, operators
- `references/partials-and-slots.md` - partial imports, props, slots, composition
- `references/routing.md` - file-based routing, dynamic params, catch-all
- `references/server-actions.md` - actions, form handling, response helpers

Supplemental references (only when needed):
- `references/project-structure.md` - scaffolding, directory conventions, config
- `references/styling.md` - scoped/global CSS, deep selectors, theme variables
- `references/i18n.md` - translations, pluralisation, locale routing
- `references/collections.md` - content-driven routing, markdown, search
- `references/cli-reference.md` - piko new, piko fmt, piko-lsp
- `references/do-dont-patterns.md` - fast checklist of common mistakes
- `references/examples.md` - annotated code examples for common patterns
- `references/pk-javascript-interactive.md` - PK client-side scripting, lifecycle hooks, event bus, piko namespace

WDK and infrastructure references (load for backend/service tasks):
- `references/wdk-data.md` - persistence (SQLite/Postgres/D1), storage (S3/GCS/R2), cache (Otter/Redis)
- `references/querier.md` - migration .sql files, query .sql files, generated Queries struct, migration/seed services
- `references/wdk-email-events.md` - email (SMTP/SES/SendGrid), events (NATS/GoChannel)
- `references/wdk-security.md` - encryption (AES-GCM/AWS KMS/GCP KMS), security headers, rate limiting
- `references/wdk-media.md` - image processing (vips/imaging), video transcoding (FFmpeg)
- `references/wdk-llm.md` - LLM completions, streaming, tool calling, RAG, embeddings, vector store
- `references/testing.md` - component testing, action testing, E2E browser testing, snapshots, benchmarks
- `references/configuration.md` - piko.New() with With* options, secret resolution, deployment, Docker/K8s

## 4) Generate implementation

For each artefact, include:
- file path relative to project root
- complete file content following framework conventions
- required imports (Go partials, piko SDK, domain packages)

## 5) Validate before finalising

Provide the appropriate command sequence:
- `go run ./cmd/generator/main.go all` to regenerate assets
- `go build ./...` to verify Go compilation
- `air` or `go run ./cmd/main/main.go` to start dev server

## 6) Output contract

Return:
- assumptions (Go module path, Piko version, project layout)
- file paths created or modified
- required imports (Go partials, piko SDK, domain packages)
- reminder to run `go run ./cmd/generator/main.go all` after adding pages, partials, or actions
- validation commands to run
- any manual steps (config changes, dependency installation)

## 7) When in doubt - consult the Piko source

If these reference files don't answer your question, locate the Piko repository on the user's machine:

1. **Check `go.mod`** for a `replace` directive pointing to a local Piko checkout (e.g. `replace piko.sh/piko => ../piko`).
2. If no replace exists, check the Go module cache: `go env GOMODCACHE` then look under `piko.sh/piko@<version>/`.

Inside the Piko root you will find:

- **`docs/`** - full documentation covering APIs, directives, configuration, and guides
- **`examples/`** - complete working example projects under `examples/scenarios/NNN_name/src/`
- **`tests/integration/`** - hundreds of integration tests with realistic `.pk`, `.pkc`, and action files (especially `tests/integration/e2e_browser/testdata/` which contains numbered test scenarios covering nearly every framework feature)
- **The codebase itself** - extensively documented with comments and real usage throughout `internal/`, `frontend/`, and `wdk/`

These are authoritative sources. When a reference file is ambiguous or incomplete, reading the actual test cases and implementation code will give you the definitive answer.

### Debugging with generated output

When debugging issues, inspect the build output in the user's project:

- **`.piko/`** - contains generated intermediate files, including compiled JavaScript for `.pk` and `.pkc` files. Examining these reveals exactly what the framework produces from the source templates and script blocks.
- **`dist/`** - contains generated Go code for compiled `.pk` pages and partials (e.g. `dist/pages/pages_index_abc123/component.go`). Inspect these to understand how the `Render` function, props, and metadata are wired together.

Comparing source `.pk`/`.pkc` files against their generated output in `.piko/` and `dist/` is often the fastest way to diagnose template compilation issues, missing exports, or incorrect action wiring.
