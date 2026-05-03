---
title: About Piko, Vue, and Nuxt
description: Where Piko borrows from Vue and Nuxt, where it deliberately diverges, and what each choice costs.
nav:
  sidebar:
    section: "explanation"
    subsection: "comparisons"
    order: 10
---

# About Piko, Vue, and Nuxt

Piko's template syntax looks a lot like Vue. Developers reaching for Piko from a Vue background find much of the surface familiar. `p-if` plays the role of `v-if`, `p-for` plays the role of `v-for`, and single-file components hold template-script-style the same way Vue SFCs do. The similarity is deliberate. The differences underneath are also deliberate. This page walks through both.

A closer comparison than Vue proper is [Nuxt](https://nuxt.com), which pairs Vue with Server-Side Rendering, file-based routing, and an opinionated project layout. Piko overlaps more with Nuxt than with bare Vue. Both take opinions, go full-stack, and build around SFC-style components and conventional folder layouts. The comparisons below call out Nuxt or Vue specifically where the difference matters.

## What Piko borrows

**Single-file components, from Vue.** The five-section PK file (template, Go script, optional client script, style, i18n) draws its shape from Vue's SFC. The reasoning is the same. Markup, logic, and styling that change together should live together.

**Directives for control flow, from Vue.** `p-if`, `p-else-if`, `p-else`, `p-for`, `p-show`, `p-model`, `p-on`, `p-bind`, and `p-class` all mirror Vue's `v-*` directives (see [directives reference](../reference/directives.md) for the complete list and semantics). The semantics are close enough that translating Vue templates by find-and-replace often produces working PK templates.

**Scoped styles by default, from Vue.** A `<style>` block inside a PK file scopes to the component, mirroring Vue's `<style scoped>` behaviour. The mechanism differs (Piko uses attribute scoping with some relaxed inheritance, Vue generates hashed selectors), but the developer experience is the same.

**Reactive client components, from Vue.** PKC files are Piko's client-side reactive components. Like Vue, each component declares its state inline. Writes to state fields trigger re-renders. The scope of reactivity is the component, not a global store.

**File-based routing, from Nuxt.** Piko derives routes from the `pages/` directory ([routing rules reference](../reference/routing-rules.md) for the exact filename-to-URL grammar). Nuxt does the same. Bare Vue projects typically declare routes in a TypeScript array via Vue Router. Piko's convention matches Nuxt, not Vue proper.

**An opinionated project layout, from Nuxt.** Nuxt expects `pages/`, `components/`, `composables/`, and other conventional folders. Piko expects `pages/`, `partials/`, `components/`, `actions/`, and `content/`. The folder names differ, but the philosophy of convention over configuration is the same.

## Where Piko diverges

**The server script is Go, not JavaScript or TypeScript.** A PK file's `<script type="application/x-go">` block is pure Go. The types declared there flow into the template, and the compiler checks every expression against them. Nuxt runs the server script in a Node.js environment using the Vue runtime. Piko runs it as compiled Go code with no runtime template engine.

**Templates compile to Go or JavaScript, not evaluated at runtime.** Piko uses one Go-like expression DSL across both PK and PKC templates. When a PK template compiles, the DSL lowers to Go. When a PKC template compiles, the DSL lowers to JavaScript. Vue and Nuxt evaluate template expressions in a runtime template engine regardless of target. Piko's lower-ahead-of-time approach trades some dynamism for compile-time type safety.

**PKC components use a small in-house virtual DOM and a microtask render scheduler.** A PKC file produces a custom element, and the browser treats it as a Web Component. The render and update logic inside the element runs through a per-component virtual DOM that diffs successive `render()` outputs and applies the minimal DOM patch (`frontend/extensions/components/src/vdom/`, scheduled via `queueMicrotask` in `RenderScheduler.ts`). The virtual DOM is intentionally minimal. There are no priorities, no suspense, no concurrent mode. It just diffs one component subtree against its previous render and patches the differences. The web-component wrapper supplies lifecycle hooks, shadow DOM, and form association. The virtual DOM keeps the inner update path fast enough that the runtime footprint stays small. Vue and React both ship larger reconciliation engines aimed at app-wide trees. Piko keeps the engine scoped to one component because cross-component coordination happens through events and server actions, not a shared reactive graph.

**Piko rejects hydration.** A Vue SPA ships the JavaScript needed to re-render the whole page on the client. Nuxt ships both the server-rendered HTML and the hydration JavaScript. Piko does neither. Server-rendered PK pages arrive as HTML and stay as HTML. Only the PKC islands embedded in the page ship JavaScript, and each island runs on its own. This removes hydration cost entirely at the price of giving up one unified component tree across the wire.

**Two component formats, not one.** Vue has one component model. Components run the same way on server and client (SSR in Nuxt is an adaptation, not a different format). Piko has two formats, PK for server and PKC for client. The seam is explicit. You pick which side of the wire a component lives on when you name the file.

**No reactivity graph across the network.** Vue's Composition API and Pinia treat reactive state uniformly whether it lives on the server or the client. Piko does not. Server state lives in PK render functions. Client state lives in PKC components. The two meet at the HTTP boundary through server actions and nowhere else. [Reference server actions](../reference/server-actions.md) documents the contract, and the [forms how-to](../how-to/actions/forms.md) walks through a contact form end-to-end.

**Bootstrap is hexagonal.** A Nuxt application wires services through modules and plugins. A Piko application wires them in the bootstrap via `With*` options ([bootstrap options reference](../reference/bootstrap-options.md) lists every option), and Piko presents them as interfaces to the rest of the code. Piko's pattern is more upfront cost and less ongoing cost. Nuxt's pattern favours quick prototyping.

## What each choice costs

**Compile-time types cost flexibility.** A dynamic page that wants to render fields the compiler does not know about at build time has a harder time in Piko than in Vue or Nuxt. The escape hatches (`map[string]any`, `piko:content`, `p-html`) exist but they feel like escape hatches.

**Rejecting hydration costs interactivity density.** A Nuxt page where every piece of text can become clickable, editable, and reactive pushes you toward "the whole page is a Nuxt app". A Piko page wants most of the page to be static and specific regions to be interactive. If an application genuinely needs whole-page interactivity, Piko's split feels limiting.

**Two formats cost duplication.** Validation logic that runs on both server and client has to exist in two places, once in Go (for the action) and once in TypeScript (for the PKC component). Nuxt can share the code because both ends run JavaScript. Piko requires the boundary to be explicit.

**File-based routing costs dynamic paths.** A route that depends on a database lookup instead of a file system is harder in Piko than in Vue Router. Collections cover most of this gap, but edge cases remain.

## When Vue or Nuxt is the right choice

Piko is not a drop-in replacement for Vue or Nuxt. Use them when:

- The application is fundamentally a single-page app with server-side data fetching; SSR is an optimisation, not the model (Vue with Vue Router).
- The application is fundamentally server-rendered with uniform hydration across the page (Nuxt).
- The development team has deep JavaScript or TypeScript expertise and no Go expertise.
- The interactivity model is rich enough that the cost of hydration is worth paying for the programming-model uniformity.
- The deployment target is a Node.js environment the team already operates.

## When Piko is the right choice

Piko is the right choice when:

- The application is fundamentally server-rendered with islands of interactivity.
- Compile-time type safety across templates is worth more than runtime flexibility.
- Go is already the server language and a separate Node.js server process is unwelcome.
- Deployment is a single Go binary instead of a bundled SPA or a Node.js server.

## See also

- [About PK files](about-pk-files.md) for the single-file format in detail.
- [About reactivity](about-reactivity.md) for the PK/PKC split.
- [About the action protocol](about-the-action-protocol.md) for how the client reaches the server.
- [About the hexagonal architecture](about-the-hexagonal-architecture.md) for the dependency-injection model.
