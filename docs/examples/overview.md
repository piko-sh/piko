---
title: Examples overview
description: Learn Piko through runnable, tested example projects
nav:
  sidebar:
    section: "examples"
    subsection: "examples"
    order: 10
---

# Examples

Each example is a self-contained project that demonstrates a specific set of Piko features. Examples progress from basic to advanced.

## Server-side (PK)

| # | Example | Features |
|---|---------|----------|
| [001](/docs/examples/001-hello-world) | Hello world | Template sections, `Render` function, text interpolation, scoped CSS |
| [004](/docs/examples/004-product-catalogue) | Product catalogue | `p-for`, `p-if`, partials, prop passing, dynamic attributes |
| [005](/docs/examples/005-blog-with-layout) | Blog with layout | Layout partials, nested partials, slots, CSS custom properties |
| [006](/docs/examples/006-data-table) | Sortable data table | Query parameters, server-side sorting, `p-class` |

## Client-side (PKC)

| # | Example | Features |
|---|---------|----------|
| [003](/docs/examples/003-reactive-counter) | Reactive counter | Reactive state, `p-on:click`, `p-class`, Shadow DOM, custom elements |
| [007](/docs/examples/007-todo-app) | Todo app | `p-for` with `p-key`, `p-model`, array reactivity, event args |
| [009](/docs/examples/009-form-wizard) | Form wizard | `p-if`/`p-else-if`/`p-else`, `p-model`, lifecycle hooks, validation |

## Server actions

| # | Example | Features |
|---|---------|----------|
| [002](/docs/examples/002-contact-form) | Contact form | Actions, form data mapping, validation, `$form` |
| [010](/docs/examples/010-progress-tracker) | Progress tracker | `StreamProgress`, SSE streaming, action builder API |
| [016](/docs/examples/016-cached-api) | Cached api | `CacheConfig`, TTL caching, `X-Action-Cache` header |
| [017](/docs/examples/017-rate-limited-api) | Rate-limited API | `RateLimit()`, token bucket, HTTP 429, rate-limit headers |

## Cross-component communication

| # | Example | Features |
|---|---------|----------|
| [008](/docs/examples/008-event-bus-chat) | Event bus chat | `piko.bus.emit()`, `piko.bus.on()`, decoupled components |
| [011](/docs/examples/011-instant-messaging) | Instant messaging | Real-time SSE chat, `withRetryStream()`, event ID resumption |

## Storage & infrastructure

| # | Example | Features |
|---|---------|----------|
| [012](/docs/examples/012-file-upload) | S3 file upload | Storage providers, `UploadBuilder`, presigned URLs |

## Component libraries

| # | Example | Features |
|---|---------|----------|
| [018](/docs/examples/018-builtin-components) | Built-in components | `components.Piko()`, `piko-counter`, `piko-card`, named slots |
| [019](/docs/examples/019-m3e-components) | M3E components | `components.M3E()`, Material Design 3, 40+ components across 6 categories |

## Full application

| # | Example | Features |
|---|---------|----------|
| [020](/docs/examples/020-m3e-recipe-app) | M3E recipe app | M3E components, SSE streaming, email actions, LLM integration, multi-page routing |
