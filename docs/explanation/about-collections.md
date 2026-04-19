---
title: About collections
description: Why Piko generates routes and data from content sources at build time, and the provider model that makes it swappable.
nav:
  sidebar:
    section: "explanation"
    subsection: "architecture"
    order: 60
---

# About collections

A collection in Piko is a declarative mapping between a content source (a directory of markdown files, an SQL table, a JSON API) and a set of generated routes. At build time, Piko reads the source, creates one route per entry, and passes the entry's structured data (frontmatter, columns, JSON keys) to the rendering page. This page explains why collections exist as a first-class concept and how the provider model keeps the surface uniform across sources that are nothing alike.

<p align="center">
  <img src="../diagrams/collections-pipeline.svg"
       alt="Build time reads source files, parses them through a collection provider, and emits one generated route per item. Runtime matches a request to a generated route and calls Render with the item data through the GetData helper."
       width="600"/>
</p>

## The problem

Most websites have two kinds of content. The first is code, meaning pages with unique layouts. The second is data, meaning sets of pages with the same shape. A product catalogue, a blog, a docs site, a team-member directory are each a template plus a dataset. Building this in a conventional framework involves three separate concerns. The framework has to read the data, generate the routes, and render each route's template.

A framework can glue those together by convention, for example file-based routing for markdown, or a plugin for SQL sources. It can also push them onto the application, for example looping over rows in `main.go` and calling a router registration function for each. The convention route is brittle as data types multiply. The application route duplicates boilerplate per data source.

Piko's answer is to lift the concept of "data source bound to a template" into a first-class declaration. A PK template adds `p-collection="blog"` and `p-provider="markdown"` in its root element. The build looks up the `blog` provider, asks it for every entry, and generates one route per entry. The rendering template reads each entry's frontmatter through `piko.GetData[T](r)` with a Go type that matches the frontmatter shape.

## Why build-time, not runtime

Collections could have been a runtime concept, for example a query in the `Render` function, called once per request. We chose build-time because the cost and the semantics both improve.

Consider the cost side first. If a blog has two hundred posts and they do not change between deploys, fetching the list at every request (or caching it in memory) is wasteful. Generating two hundred static HTML files at build time eliminates both the query and the render cost at serving time. The hybrid provider model supports Incremental Static Regeneration for content that does change, so the build-time default does not close the door on dynamic content.

The semantics improve too. Building the list at build time means missing-slug handling and 404 behaviour are deterministic. A request to `/blog/foo` either matches a generated route or falls through to the error page. There is no race where the framework fetches the list between request arrival and response, no split-brain where one replica has a row another does not.

For projects where data truly changes per-request (user dashboards, search results), collections are the wrong tool. Actions or regular dynamic routes handle those cases by running their own queries.

## The provider interface

A provider implements a small interface: "given a collection name, return the list of items," with an optional "given an item key, return the full item" for lazy detail fetching. The markdown provider reads a directory of `.md` files with YAML frontmatter. The SQL provider runs a query and maps rows to struct fields. A custom provider can wrap any data source that supports enumeration.

This is deliberately the same pattern as Piko's storage, cache, email, and other services. Each has a domain-defined port (interface), one or more adapter implementations (providers), and a registration call at bootstrap (`WithCollectionProvider(...)`). The uniformity makes the codebase smaller and lets users who understand one service understand the others.

## Filters, sort, and pagination

Collections grow, and pages that enumerate them (a blog index, a product catalogue listing) need filtering, sorting, and pagination. Piko puts these on the collection service instead of making each page implement them.

A page calls `piko.SearchCollection[T]` with a `Filter` (built from `Eq`, `And`, `Or`, `Gt`, etc.), a `SortOption`, and a `PaginationOptions`. The collection service pushes the work to the provider when the provider supports it. SQL providers translate filters into WHERE clauses. When a provider cannot push down, the service performs the work in memory. The markdown provider loads everything and filters client-side.

This keeps the page template agnostic of the data backend. Swapping from markdown to SQL is a provider change, not a template rewrite.

## Trade-offs

Build-time generation is uncomfortable at extreme scale. Ten thousand blog posts is still fast. Ten million rows is not, and the output blows up to ten million HTML files. Piko handles this with hybrid routes, a model sometimes called Incremental Static Regeneration. Some routes render at build time, others render on demand and cache. The decision is per-collection, not global.

Heavy frontmatter schemas sometimes drift between what the template expects and what the files contain. The generated Go struct for frontmatter is an opt-in layer. A page can call `piko.GetData[Post](r)` to get the frontmatter typed, or `piko.GetDataMap(r)` to get it as a `map[string]any`. The second form is looser and traded against the first's compile-time guarantees.

## See also

- [Collections API reference](../reference/collections-api.md) for every function and directive.
- [How to markdown collections](../how-to/collections/markdown.md) for the default source.
- [How to querying and filtering](../how-to/collections/querying-and-filtering.md) for the `Filter`, `SortOption`, and `Pagination` APIs.
- [How to custom providers](../how-to/collections/custom-providers.md) for backing a collection with a non-markdown source.
