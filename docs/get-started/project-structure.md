---
title: Project structure
description: The folder layout that piko new creates, with a one-line purpose and a deep-link for each directory.
nav:
  sidebar:
    section: "get-started"
    subsection: "overview"
    order: 40
---

# Project structure

This page is a map. When `piko new` finishes, the project directory looks like the tree below. Each folder has a role and a canonical place in the docs where it gets discussed in depth. Use the table that follows the tree to jump to whatever you are looking at right now.

<p align="center">
  <img src="../diagrams/project-structure-map.svg"
       alt="Four lifecycle-phase cards stacked top to bottom. Build-time lists the folders piko generate parses into typed Go and compiled assets: pages, partials, components, actions, content, emails, pdfs, i18n, db, assets, and styles, with output landing under .piko and dist. Bootstrap lists the entry points and configuration that wire the framework together: cmd/main/main.go, cmd/generator/main.go, piko.yaml, and config.json. Standard Go layout covers internal and pkg. Generated covers .piko and dist, produced by piko generate and read by the binary at request time, marked do not edit and do not commit. The diagram has no separate request-time card because the runtime never reads any folder the user edits."
       width="760"/>
</p>

For the "why this shape" (the reasoning behind the folder conventions and the generator pipeline that relies on them), see [about project structure](../explanation/about-project-structure.md).

## The scaffolded tree

```text
my-app/
├── .piko/                # Generator output read by the binary. Do not edit. Do not commit.
├── dist/                 # Compiled assets and bundles served at runtime. Do not edit. Do not commit.
├── actions/              # Server actions (Go)
├── components/           # Client components (.pkc)
├── pages/                # Page routes (.pk)
├── partials/             # Reusable template fragments (.pk)
├── emails/               # Email templates (.pk with PML)
├── pdfs/                 # PDF templates (.pk)
├── content/              # Markdown collections (optional)
├── i18n/                 # Locale files (optional)
├── assets/               # Static assets (images, fonts)
├── styles/               # Shared CSS
├── db/                   # SQL queries and migrations (optional)
├── internal/             # Private Go packages
├── pkg/                  # Public Go packages (optional)
├── cmd/
│   ├── generator/main.go # Asset and manifest generator entry point
│   └── main/main.go      # Server entry point
├── config.json           # Runtime theme, frontend config
├── piko.yaml             # Base configuration
├── piko-prod.yaml        # Optional per-environment override
├── piko.local.yaml       # Gitignored per-machine override
├── .env                  # Gitignored environment variables
├── .air.toml             # Air live-reload configuration
├── .gitignore
├── Dockerfile
├── go.mod
└── go.sum
```

Not every project has every directory. `emails/` is only for when the project sends mail. `content/` is only for when it uses markdown collections. `components/` is only for when it uses client reactivity. `db/` is only for when it talks to a database. `i18n/` is only for when it supports more than one locale.

## Folder-by-folder

| Folder | Contains | Where to read next |
|---|---|---|
| `pages/` | `.pk` files. One file per route. The filename (`pages/about.pk`) maps to the URL (`/about`). | [routing rules reference](../reference/routing-rules.md), [tutorial 01: your first page](../tutorials/01-your-first-page.md) |
| `partials/` | Reusable PK fragments composed into pages with `<piko:partial is="layout">`. Not routable. | [how to layout partials](../how-to/partials/layout.md) |
| `components/` | PKC files. Each becomes a Web Component on the client. Convention: prefix project components with `pp-`. | [client components reference](../reference/client-components.md), [tutorial 02: adding interactivity](../tutorials/02-adding-interactivity.md) |
| `actions/` | Go structs embedding `piko.ActionMetadata`. Each becomes a typed RPC endpoint. One subdirectory per package. | [server actions reference](../reference/server-actions.md), [tutorial 03: server actions and forms](../tutorials/03-server-actions-and-forms.md) |
| `emails/` | PK templates rendered to PML for email delivery. | [how to email templates](../how-to/email-templates.md), [about email rendering](../explanation/about-email-rendering.md) |
| `pdfs/` | PK templates rendered to PDF. | [how to PDF generation](../how-to/pdf-generation.md) |
| `content/` | Markdown files (or other collection sources). Each file becomes a generated route when a page declares `p-collection`. | [collections API reference](../reference/collections-api.md), [tutorial 04: shipping a real site](../tutorials/04-shipping-a-real-site.md) |
| `i18n/` | One JSON file per locale. Global translations accessed through `T("key")`. | [i18n API reference](../reference/i18n-api.md), [tutorial 07: going multilingual](../tutorials/07-going-multilingual.md) |
| `assets/` | Static files (images, fonts, icons). Served directly or processed through the asset pipeline. | [how to assets](../how-to/assets.md) |
| `styles/` | Shared CSS imported by pages and partials. | [how to scope and bridge component CSS](../how-to/templates/scoped-css.md) |
| `db/` | SQL query files (`db/queries/*.sql`) and migrations (`db/migrations/*.sql`). The generator emits typed Go code from these. | [tutorial 05: data-backed pages](../tutorials/05-data-backed-pages.md), [querier reference](../reference/querier.md) |
| `internal/` | Private Go packages used only by this project. | (standard Go layout) |
| `pkg/` | Public Go packages exported for reuse. | (standard Go layout) |
| `cmd/generator/` | Entry point that runs the Piko generator. Invoke with `go run ./cmd/generator/main.go`. | [CLI reference](../reference/cli.md) |
| `cmd/main/` | Server entry point. Wires `piko.New(...)` with `With*` bootstrap options and calls `ssr.Run(mode)`. | [bootstrap options reference](../reference/bootstrap-options.md), [about the hexagonal architecture](../explanation/about-the-hexagonal-architecture.md) |

## Configuration files

| File | Purpose | Where to read next |
|---|---|---|
| `config.json` | Runtime theme, frontend module configuration, declared locales. | [config JSON schema reference](../reference/config-json-schema.md) |
| `piko.yaml` | Base server configuration (port, paths, health probe, OTLP). | [about configuration](../explanation/about-config.md) |
| `piko-prod.yaml` | Optional per-environment override. Loaded when running in `prod` mode. | [how to environment config](../how-to/deployment/environment-config.md) |
| `piko.local.yaml` | Gitignored per-machine override for developer-specific settings. | [how to environment config](../how-to/deployment/environment-config.md) |
| `.env` | Environment variable overrides, loaded during development. Gitignored. | [how to environment config](../how-to/deployment/environment-config.md), [how to secrets resolvers](../how-to/deployment/secrets-resolvers.md) |
| `.air.toml` | Air live-reload configuration. Watches Go files and templates, restarts the server. | [Air on GitHub](https://github.com/air-verse/air) |
| `Dockerfile` | Production container build. Scaffolded with sensible defaults. | [how to production build](../how-to/deployment/production-build.md) |

## What stays out of version control

The generator produces output under `.piko/` and `dist/`. Both directories are auto-managed. The binary reads them at request time. Do not edit them by hand. `.gitignore` excludes them, along with `.env`, `piko.local.yaml`, and compiled binaries.

## What to do next

- [Your first page](../tutorials/01-your-first-page.md) makes the first real edit in `pages/`.
- [Concepts](concepts.md) names every piece so the folders and their contents make sense in context.
- [About project structure](../explanation/about-project-structure.md) explains why the layout is what it is and how the generator reads it.
