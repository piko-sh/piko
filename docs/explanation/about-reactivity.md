---
title: About reactivity
description: How Piko divides server rendering from client reactivity and what that implies for the mental model.
nav:
  sidebar:
    section: "explanation"
    subsection: "architecture"
    order: 20
---

# About reactivity

Piko ships two component formats. `.pk` files produce server-rendered HTML, and `.pkc` files produce client-side Web Components. The split is deliberate. This page explains how the two formats relate and why Piko rejects the full-page hydration model that dominant React and Vue setups use.

<p align="center">
  <img src="../diagrams/pk-vs-pkc-lifecycle.svg"
       alt="Two swim lanes. The PK lane runs only on the server: request, Render, HTML sent, browser paints once, done. The PKC lane runs only in the browser: HTML arrives with a PKC tag, the component boots, state mutations trigger local re-renders, and the loop continues on every interaction."
       width="600"/>
</p>

The two lanes never share a reactive graph. A PK page renders once on the server. A PKC component owns its state in the browser. When a PK page needs interactivity it embeds a PKC tag, and the PKC component runs independently of the page it sits inside.

## The two formats at a glance

A PK file runs on the server. Its `Render` function returns a `Response` value and the template uses that value to produce HTML. The server sends that HTML to the browser, which displays it. The PK file does nothing after that, and the browser sees static HTML.

A PKC file runs in the browser. It compiles to a native Web Component that carries its own reactive state, re-renders when the state changes, and responds to events. The server knows about PKC components (it serves the JavaScript bundle) but does not run them.

A page can mix the two freely. One mix puts interactive quantity selectors on a server-rendered product page ([Scenario 004: product catalogue](../showcase/004-product-catalogue.md)). Another gives a blog post a reactive counter ([Scenario 003: reactive counter](../showcase/003-reactive-counter.md)). Another backs a form with client-side validation and a server action ([Scenario 002: contact form](../showcase/002-contact-form.md)).

## Why split the two

Three reasons.

The first reason involves **cost**. Server rendering is cheap, meaning one `Render` call, one HTML response, done. Hydration, where the client rebuilds the server's render tree so it can re-render the whole page, is expensive. Piko sidesteps hydration entirely. [about SSR](about-ssr.md) covers the full argument. The server HTML arrives and stays. Only the islands of the page that need interactivity ship JavaScript, and each island reacts on its own.

The second reason is **clarity of responsibility**. A PK file is server state made visible. The `Render` function owns the data shape. A PKC file is client state made interactive. Its `state` object is its own. When a field lives in both worlds (a server-rendered list that the client can filter), the boundary is explicit. The server renders the initial state, and the client takes over from there. No shared reactive graph spans the network.

The third reason involves **typed signatures all the way down**. A PK file's Go types bind directly to its template expressions. A PKC file's TypeScript state binds directly to its template expressions. Neither format introduces an intermediate reactive layer that could lose the type information, so a server change that renames `state.CategoryId` to `state.CategoryID` fails the template compile immediately.

## Where the seam lives

The HTTP boundary. The server emits HTML. The browser receives HTML. Any subsequent interactivity comes from PKC components the server embedded in that HTML, plus server actions the PKC components call back to.

> **Note:** If you are coming from Next.js or Nuxt: there is no hydration step in Piko. The server's render is final HTML that the browser displays as-is, and PKC components boot independently inside it. The two never share a reactive graph or a render pass.

This is the opposite of the "everything is a React component" model. In React, the server pre-renders the same component tree that the browser then re-renders. In Piko, the server renders server-shaped components and the browser runs browser-shaped components. The seam is where they meet on the wire.

## What PKC reactivity actually does

A PKC component's `state` object is reactive. Writes to its fields trigger a re-render of the DOM subtree the component owns. The mechanism:

- On construction, the framework wraps `state` in a proxy that records reads and writes.
- During render, reads build a dependency set for the rendered subtree.
- After render, a write to any tracked field marks the subtree dirty.
- The next animation frame runs the render again, diffing output against the previous DOM.

This is similar to Vue's reactivity. The scope is local to the component, not global. A PKC component cannot reach into another's state. It must dispatch events through `piko.bus` ([how to event bus](../how-to/client-components/event-bus.md)) or pass props.

## Why PKC compiles to raw JavaScript instead of a Web-Components-native shape

A PKC file becomes a custom element, and the browser treats it as a Web Component. The render and update logic inside the element, however, lowers to direct JavaScript DOM operations. It does not target a web-components-native template or shadow-DOM-native reactivity pipeline. Two reasons motivate that choice.

Raw JavaScript DOM work still outperforms web-components-native rendering for the kinds of updates Piko cares about. A targeted set of `textContent` writes, `classList` toggles, and `insertBefore` calls is faster than most framework-internal template update paths. A web-components-centric build would pull in more machinery and run more slowly for the same UI.

The compiled output is also smaller and arrives faster, which matters more in practice than the runtime speed difference. A lighter PKC bundle loads before the browser would otherwise have time to parse, compile, and execute a larger framework payload. Ironically, committing to raw JavaScript makes Piko feel more responsive than a heavier framework that promised to be the fast choice.

A secondary benefit follows. Compiling to imperative JavaScript keeps the line between frontend and backend crisp. There is no shared reactive abstraction that tempts code to reach across the network. The frontend's contract with the backend is the HTTP boundary, not a shared framework runtime.

## What server actions add

A PKC component cannot save to a database. Server actions bridge the gap in four steps. The PKC component calls `piko.actions.customer.Upsert.call({...})`. The browser posts to the server. The server runs the typed `Call` method. The PKC component receives the typed response.

Server actions are RPC over HTTP with typed request and response shapes. They are not general request handlers. They are not route handlers. They are the only way a client component talks to the server. The [action protocol page](about-the-action-protocol.md) covers the design in detail.

## The cost of the split

PKC and PK components cannot share code directly. A function that belongs to both must live in two places, once in a `.go` file reachable by the PK script and once in a `.ts` file bundled with the PKC. This is a real cost for large applications. The mitigation is that most "shared" logic is actually server-only (it talks to a database or calls an external API) and belongs on one side of the seam. The duplicated cases are usually validation rules and formatters.

## When to reach for PK, when to reach for PKC

- Use **PK** for everything that does not need in-browser reactivity. Blog posts, product listings, dashboards that reload on navigation, even most forms. PK pages are cheap to render, cheap to cache, SEO-friendly by default.
- Use **PKC** when a component genuinely needs to respond to the user between server round-trips. A live search bar, a form wizard, a reactive counter, a chat interface. Do not reach for PKC because "modern sites use JavaScript"; reach for it when state changes would otherwise require a page reload.

## See also

- [About PK files](about-pk-files.md) for the server-rendered component format.
- [About the action protocol](about-the-action-protocol.md) for how PKC talks to the server.
- [Client components reference](../reference/client-components.md) for the PKC file format.
- [How to reactivity](../how-to/client-components/reactivity.md).
