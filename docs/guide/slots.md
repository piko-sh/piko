---
title: Slots
description: Content projection with slots in Piko partials
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 39
---

# Slots

Slots enable content projection in partials, allowing you to pass HTML content (not just data) to components. This pattern creates flexible, reusable layout components where the parent can inject arbitrary content into designated areas.

## Overview

A slot is a placeholder in a partial template that gets filled with content provided by the parent when the partial is invoked. Piko uses the `<piko:slot>` element for server-side content projection.

**Common use cases:**
- Layout partials that wrap page content
- Card components with customisable headers, bodies, and footers
- Modal dialogs with configurable sections
- Any partial that needs to accept arbitrary HTML content

## Defining slots

### Default slot

A default slot captures any content not explicitly assigned to a named slot.

**Partial** (`partials/card.pk`):

```piko
<template>
  <div class="card">
    <h2>Card title</h2>
    <piko:slot />
  </div>
</template>
```

### Named slots

Use the `name` attribute to create multiple content areas within a single partial.

**Partial** (`partials/layout.pk`):

```piko
<template>
  <div class="page-layout">
    <header>
      <piko:slot name="header"></piko:slot>
    </header>
    <main>
      <piko:slot></piko:slot>
    </main>
    <footer>
      <piko:slot name="footer"></piko:slot>
    </footer>
  </div>
</template>
```

### Slots with fallback content

Provide default content inside a slot that is used when the parent does not supply content for that slot.

```piko
<template>
  <div class="page-layout">
    <header>
      <piko:slot name="header">
        <h2>Default header</h2>
      </piko:slot>
    </header>
    <main>
      <piko:slot>
        <p>This is the default fallback content.</p>
      </piko:slot>
    </main>
    <footer>
      <piko:slot name="footer">
        <p>Default footer</p>
      </piko:slot>
    </footer>
  </div>
</template>
```

## Passing content to slots

### Default slot content

Content placed directly inside a partial invocation (not wrapped in a named slot) goes to the default slot.

```piko
<template>
  <piko:partial is="my_card">
    <p>This content goes into the default slot.</p>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
  "piko.sh/piko"
  my_card "myapp/partials/card.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
  return piko.NoResponse{}, piko.Metadata{}, nil
}
</script>
```

### Named slot content with `<piko:slot>` wrapper

Wrap content in a `<piko:slot name="...">` element to direct it to a specific named slot.

```piko
<template>
  <piko:partial is="my_layout">
    <piko:slot name="header">
      <h1>Main page header</h1>
    </piko:slot>

    <p>This is default slot content.</p>

    <piko:slot name="footer">
      <p>Copyright 2025</p>
    </piko:slot>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
  "piko.sh/piko"
  my_layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
  return piko.NoResponse{}, piko.Metadata{}, nil
}
</script>
```

### Named slot content with `piko:slot` attribute

Alternatively, use the `piko:slot` attribute on any element to direct it to a named slot.

```piko
<template>
  <piko:partial is="my_layout">
    <article p-slot="content">
      <h2>Article title</h2>
      <p>This is the main article content for the page.</p>
    </article>

    <div p-slot="header-actions">
      <a href="/new" class="button-primary">Create new</a>
      <a href="/export" class="button-secondary">Export</a>
    </div>

    <p>This is some sidebar information (goes to default slot).</p>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
  "piko.sh/piko"
  my_layout "myapp/partials/layout.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
  return piko.NoResponse{}, piko.Metadata{}, nil
}
</script>
```

## Slot validation

Piko validates slot usage at compile time:

- **Missing slot warning.** A warning is emitted if content is provided for a slot that does not exist in the partial
- **Typo suggestions.** When a slot name is similar to an existing slot, Piko suggests the correct name

```text
warning: Component <card> does not have a slot named 'fotter'. Did you mean 'footer'?
warning: Component <card> does not have a default slot, but content was provided.
```

## Context preservation

Slotted content retains its original context for expression binding. Expressions in slotted content resolve against the invoking component's scope, not the partial's scope.

```piko
<!-- Page component -->
<template>
  <piko:partial is="my_card">
    <!-- state.PageMessage is from the page's Response, not the card's -->
    <p>{{ state.PageMessage }}</p>
  </piko:partial>
</template>
```

The `state.PageMessage` expression is evaluated in the context of the page component, even though the paragraph is rendered inside the card partial.

## Advanced patterns

### Nested partials with slots

Partials can be nested within slot content, creating complex component hierarchies.

**Page** (`pages/dashboard.pk`):

```piko
<template>
  <piko:partial is="page_layout" class="theme-dark" :data-logged-in="state.IsLoggedIn">

    <piko:slot name="main-content">
      <h1>Welcome to the Dashboard</h1>
      <p>This is the main content area provided by the page.</p>
    </piko:slot>

    <piko:slot name="sidebar">
      <!-- Nested partial within a slot -->
      <piko:partial is="my_sidebar" :is-collapsible="true">

        <piko:slot name="top">
          <h2>Main navigation</h2>
        </piko:slot>

        <p>Hello, {{ state.Username }}!</p>

      </piko:partial>
    </piko:slot>

  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
  "piko.sh/piko"
  page_layout "myapp/partials/page_layout.pk"
  my_sidebar "myapp/partials/sidebar.pk"
)

type Response struct {
  IsLoggedIn bool
  Username   string
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
  return Response{
    IsLoggedIn: true,
    Username:   "Alice",
  }, piko.Metadata{}, nil
}
</script>
```

### Conditional slot rendering

Use directives to conditionally render slot containers.

**Partial** (`partials/product-card.pk`):

```piko
<template>
  <div class="product-card">
    <h3>{{ state.Title }}</h3>

    <div p-if="state.ShowBadge" class="badge">
      <piko:slot name="badge">
        <span>New</span>
      </piko:slot>
    </div>

    <piko:slot />
  </div>
</template>
```

## Next steps

- [Partials](/docs/guide/partials) → Creating and using partials with props
- [Advanced templates](/docs/guide/advanced-templates) → Attribute merging, fragment components, scoped CSS
