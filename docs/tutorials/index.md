---
title: Tutorials
description: A guided journey from "hello, page" to a multilingual, tested, data-backed Piko site.
nav:
  sidebar:
    section: "tutorials"
    subsection: "getting-started"
    order: 0
---

# Tutorials

The Piko tutorials are a guided journey. Each one builds on the last, so it pays to work through them in order. By the end of the seven, you have a working portfolio of skills. You will have written a styled marketing page, added client-side interactivity, and wired up server actions with validated forms. You will have shipped a real blog, persisted state to SQLite, covered the result with tests, and translated everything into a second language.

If you have not installed Piko yet, start with the [introduction](../get-started/introduction.md) and the [install and run](../get-started/install.md) guide. Once the dev server boots, come back here.

## The journey

The tutorials split into two arcs that share scaffolding and conventions:

- **Tutorials 01 through 04 plus 07** build the running blog example: pages, partials, components, actions, layouts, deployment, and i18n.
- **Tutorials 05 and 06** are a side-quest: they build a SQLite-backed task manager and add a test suite to it. The scaffolding from earlier tutorials carries over, but the example is its own project.

You can do either arc on its own. The tutorial pages call out their prerequisites at the top.

## Index

- [01: Your first page](01-your-first-page.md): build an "About" page with dynamic data, scoped CSS, conditionals, lists, and a layout partial.
- [02: Adding interactivity](02-adding-interactivity.md): drop a reactive counter and a todo list onto pages using PKC components.
- [03: Server actions and forms](03-server-actions-and-forms.md): write a validated contact form and a streaming progress action.
- [04: Shipping a real site](04-shipping-a-real-site.md): assemble layouts, markdown collections, slots, metadata, and a sitemap into a deployable blog.
- [05: Data-backed pages with the querier](05-data-backed-pages.md): build a SQLite-backed task manager with the type-safe querier (side-quest).
- [06: Testing what you built](06-testing-what-you-built.md): add component tests, an action test, and a snapshot to the task manager (side-quest).
- [07: Going multilingual](07-going-multilingual.md): translate the blog into French with locale files, prefix routing, and per-locale content.

## Where to next

- [How-to guides](../how-to/) for goal-oriented recipes once a tutorial wraps up.
- [Reference](../reference/) for the exact API surface of every helper, directive, and component.
- [Explanation](../explanation/) for the rationale: why PK files, why the action protocol, why the hexagonal architecture.
- [Runnable scenarios](../../examples/scenarios/) for end-to-end source you can clone and modify.
