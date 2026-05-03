---
title: About PK files
description: Why Piko uses single-file components and what the tradeoffs are.
nav:
  sidebar:
    section: "explanation"
    subsection: "architecture"
    order: 10
---

# About PK files

A PK file is one file holding up to five languages. HTML lives in the template, Go in the server script, optional TypeScript or JavaScript in a client script, CSS in the style, and JSON in the i18n block. On first look that choice is surprising. Go projects tend to separate concerns aggressively. This page explains why Piko chose the single-file shape and what that decision costs.

The mechanical shape, with an arrow from each block to its compiled output, is in the [PK file anatomy diagram](../reference/pk-file-format.md). This page concerns itself with the reasoning behind that shape.

## The shape

Every PK component has up to five sections:

```piko
<template>
  <!-- HTML with Piko directives -->
</template>

<script type="application/x-go">
  // Go: Render function, types, helpers
</script>

<script lang="ts">
  // Optional: small frontend interactivity for this page.
  // No reactive state lives here; that belongs in a .pkc component.
</script>

<style>
  /* CSS, scoped to this component */
</style>

<i18n lang="json">
  { "en": { "greeting": "Hello" } }
</i18n>
```

Any section is optional. A template-only file is valid. A script-only file is not, because there is nothing to render.

The `<i18n>` block accepts only `lang="json"`. Other `lang` values (such as `yaml`) are silently ignored by the parser, so keep translations in JSON.

### What the client script block is for

The TypeScript or JavaScript block is a small escape hatch for frontend behaviour that is specific to one PK page. Typical uses include wiring a click handler that does not need reactive state, updating a CSS class based on scroll position, or calling a server action and showing its result. It has native access to the Piko client runtime, so calling `action.contact.Submit({...}).call()` or subscribing to `piko.bus` works without any setup. Three runtime roots are in scope. `pk` covers the per-page context (refs and lifecycle), `piko` covers the global runtime helpers (bus, nav, form, event, and the rest), and `action` covers typed server actions.

What the client script block does not do is own reactive state. It has no `state` proxy and nothing re-renders when its variables change. The moment a piece of interactivity needs reactive state, a keyed loop, or two-way form binding, it belongs in a `.pkc` client component, not in a PK script block.

Treat the client script block as the place for linking glue, that is, a short run of lines that connect an element to an action or the event bus. If it grows past that, extract a PKC component.

### Why the style block scopes per component

Piko scopes a `<style>` block to its component. Selectors match only elements the template emits, not the page outside. The mechanism differs between the two formats. PK uses attribute-based selector rewriting. PKC uses shadow-DOM encapsulation through the Web Component it compiles to. The outcome is the same. Two components can define `.card` without fighting. For the exact scoping rules see [pk-file format reference](../reference/pk-file-format.md#style-section) and [client components reference](../reference/client-components.md#shadow-dom-and-styling).

### Why script tags use mixed conventions

The Go block spells its identifier as `<script type="application/x-go">`, a MIME type. The TypeScript block spells its identifier as `<script lang="ts">`, a short attribute. The inconsistency is deliberate.

`lang="go"` works identically to `type="application/x-go"` and both render the same. IDEs and language servers, however, vary on which form they pick up without extra plugin configuration. `application/x-go` is the more reliable spelling for Go-aware tooling, and `lang="ts"` is the more reliable spelling for TypeScript-aware tooling. Piko's convention leans into each ecosystem's expectation instead of enforcing uniformity.

## Why a single file

Three reasons lead to the single-file format.

The first is **locality of reference**. The markup, the data shape, the styling, and the translations that describe one component all change together. Putting them in one file means a change to "the product card" is a change to one file, not an edit split across `components/product-card/index.go`, `components/product-card/ProductCard.tmpl`, `components/product-card/ProductCard.css`, and `i18n/en/product-card.json`. Files that change together belong together.

The second is **direct compilation against the right types**. The Go in the script block declares the types that the template binds to. The compiler reads the script first, builds the type table, then parses the template expressions against those types. A template reference to `state.Count` that the script has not declared fails the build with a location that points at the template line. If the script lived in a separate file, Piko could still do this, but the file boundary would require the compiler to resolve an external reference first. Inline is simpler and produces better error messages.

The third is **a familiar model for people coming from Vue**. Vue's single-file component (SFC) has been a successful shape for nearly a decade. Developers arriving from that world find PK immediately recognisable and do not have to learn a new mental model. Piko deliberately borrows the SFC convention even though the implementations are different in every other way.

## What the single file costs

Four real costs come with the format.

Large PK files can become unwieldy. A product page with rich behaviour might reach 500 lines across template, script, and style. There is no syntactic way to split a PK file. It stays as one file. The mitigation is that PK encourages splitting big pages into partials, which are themselves PK files. When a file grows past a thousand lines, that is a signal that it wants to be two files.

The multi-language shape confuses tooling. Editors need dedicated support to colour tokens and navigate a PK file correctly. The Piko LSP and the bundled extensions handle this, but a plain text editor shows a PK file as a single big HTML blob. Due to this we treat tooling support as a hard requirement, not a nice-to-have.

Tests cannot import one section without the others. When a Go test wants to test the `Render` function's business logic, it imports the whole file. Piko's test harness handles this. The [testing how-to](../how-to/testing.md) shows the pattern. But a function-level unit test is less ergonomic than it would be if the Go lived in a plain `.go` file. Keep domain logic outside of pk files.

The single file tangles template-compilation errors with Go-compilation errors. An incorrect directive surfaces as an error during the Piko generator pass, not during `go build`. The ordering is learnable but adds a step for newcomers.

## The script is Go; the expression language is a Go-like DSL

The script section is plain Go. The types it declares are the same types the template references, and because the script is pure Go, any Go language feature works there.

The expression language inside `{{ ... }}` and directive attributes (`{{ state.Count }}`, `p-if="state.Role == 'admin'"`) is a separate, smaller language, a Go-like DSL. One DSL covers both PK and PKC templates. When a PK template compiles, the DSL lowers to Go. When a PKC template compiles, the same DSL lowers to JavaScript. Writing templates uses one syntax, and Piko picks the target.

The DSL looks like Go because that shape makes lowering to Go straightforward. It also adds ergonomic extensions that plain Go does not support, including ternary expressions, `~=` loose equality, `~expr` truthy coercion, `??` nullish coalescing, and template literals. [template syntax reference](../reference/template-syntax.md) lists every operator with examples.

The DSL deliberately omits matters too. Struct literals, function declarations, and most block constructs are not permitted. The reason is partly practical (struct definitions inside a template attribute would be unreadable) and partly a commitment to keeping the expression parser small enough to reason about.

Template expressions in Vue or React run on the client at runtime against whatever shape the data happens to be. Piko's DSL compiles ahead of time against the types declared in the script (Go for PK, TypeScript for PKC). If the data shape changes, the template stops compiling. The cost is that some patterns (dynamic keys, untyped maps) are harder to express. The benefit is that the template cannot silently render nothing when a field is missing.

## What belongs in the script block

The script block is for the transformations and logic that shape data for the template. It is not the right place for domain logic. Fetching from a database, calling an external API, enforcing business rules, and running long-lived computations all belong in plain Go packages that the script block calls into.

Good uses of the script block:

- Decoding collection data into a typed struct.
- Deriving display fields (formatting a date, computing a total).
- Conditionally setting metadata (title, canonical URL) based on the data.

Bad uses of the script block:

- Writing an entire authentication flow inside `Render`.
- Performing multi-step database transactions.
- Maintaining long-lived shared state between requests.

Keep the script block thin. The `Render` function should read as "here is what this page displays, given its data". If it reads as "here is how this business operates", extract the logic into a Go package and call it.

## When a PK file is not the right shape

Three cases suggest reaching for a plain `.go` file or a different component:

- Pure business logic with no template: write a plain Go package and call it from the Render function.
- A client-only interactive component: write a `.pkc` file instead. PKC compiles to a Web Component with reactive TypeScript state and has its own single-file shape.
- A long-form static document: consider a markdown collection with a PK template rendering the body.

## See also

- [PK file format reference](../reference/pk-file-format.md) for the exact syntax of each section.
- [About reactivity](about-reactivity.md) for how PK (server) and PKC (client) divide responsibility.
- [About the hexagonal architecture](about-the-hexagonal-architecture.md) for the surrounding Piko design.
- [About Piko vs Vue](about-piko-vs-vue.md) for the SFC lineage and where it diverges.
