# Partials and Slots

Use this guide when creating reusable `.pk` partials, passing props, or projecting content with slots.

## Creating a partial

Partials live in the `partials/` directory. They are regular `.pk` files with a Props struct:

```piko
<!-- partials/card.pk -->
<template>
  <div class="card">
    <h3>{{ state.Title }}</h3>
    <piko:slot />
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    Title string `prop:"title"`
}

type Response struct {
    Title string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{Title: props.Title}, piko.Metadata{}, nil
}
</script>
```

## Importing and using partials

Import in the Go script, then use with the `is` attribute:

```go
import (
    "piko.sh/piko"
    card "myapp/partials/card.pk"
)
```

```piko
<card is="card" :server.title="state.PageTitle">
  <p>This goes in the default slot.</p>
</card>
```

The element tag name (`card`) is arbitrary - the `is` attribute determines which partial renders. The `is` value must match the import alias and **must be static** (not dynamic).

## Props

### Defining props

```go
type Props struct {
    Title       string  `prop:"title"`
    IsPrimary   bool    `prop:"is-primary" coerce:"true"`
    ItemCount   int     `prop:"item-count" coerce:"true"`
    Description *string `prop:"description"`       // Optional (pointer = nullable)
    Page        string  `prop:"page" query:"p"`    // Bound to ?p= query param
}
```

### Prop tags

| Tag | Purpose |
|-----|---------|
| `prop:"name"` | Maps attribute to field (use kebab-case) |
| `default:"value"` | Default if not provided |
| `factory:"FuncName"` | Factory function for complex defaults |
| `validate:"required"` | Mark as required (compile-time check) |
| `coerce:"true"` | Type coercion (string to int, bool, etc.) |
| `query:"param"` | Bind to URL query parameter |

### Passing props

Use the `:server.` prefix for dynamic server props:

```piko
<!-- Dynamic props (evaluated at render time) -->
<card is="card"
    :server.title="state.PageTitle"
    :server.is-primary="state.IsFeatured"
    :server.item-count="len(state.Items)">
</card>

<!-- Static props (string values, use coerce for non-string types) -->
<card is="card"
    server.title="Welcome"
    server.is-primary="true">
</card>
```

### Alternative syntax

```piko
<piko:partial is="card" :server.title="state.PageTitle">
  <p>Content</p>
</piko:partial>
```

## Slots

Slots allow parent components to project content into partial templates.

### Default slot

In the partial:

```piko
<template>
  <div class="wrapper">
    <piko:slot />
  </div>
</template>
```

From the caller:

```piko
<wrapper is="wrapper">
  <p>This content fills the default slot.</p>
</wrapper>
```

### Named slots

In the partial:

```piko
<template>
  <div class="layout">
    <header><piko:slot name="header" /></header>
    <main><piko:slot /></main>
    <footer><piko:slot name="footer" /></footer>
  </div>
</template>
```

From the caller (two syntaxes):

```piko
<!-- Wrapper syntax -->
<layout is="layout">
  <piko:slot name="header">
    <h1>Page Title</h1>
  </piko:slot>

  <p>Default slot content.</p>

  <piko:slot name="footer">
    <small>Footer text</small>
  </piko:slot>
</layout>

<!-- Attribute syntax -->
<layout is="layout">
  <h1 p-slot="header">Page Title</h1>
  <p>Default slot content.</p>
  <small p-slot="footer">Footer text</small>
</layout>
```

### Fallback content

Provide default content inside `<piko:slot>`:

```piko
<piko:slot name="actions">
  <button>Default Action</button>
</piko:slot>
```

If the caller doesn't provide content for this slot, the fallback renders.

### Slot validation

Piko validates slots at compile time:
- Warning if content is provided for a slot that doesn't exist
- Typo suggestions when a slot name is similar to an existing one

### Context preservation

Slotted content retains the **caller's** scope. Expressions resolve against the invoking component, not the partial:

```piko
<!-- Page (caller) -->
<my_card is="my_card">
  <!-- state.PageMessage comes from the page's Response -->
  <p>{{ state.PageMessage }}</p>
</my_card>
```

## Nested partials

Partials can be nested arbitrarily deep:

```piko
<page_layout is="page_layout">
  <piko:slot name="sidebar">
    <my_sidebar is="my_sidebar">
      <piko:slot name="top">
        <h2>Navigation</h2>
      </piko:slot>
    </my_sidebar>
  </piko:slot>
</page_layout>
```

## Partial refresh

Partials support graduated refresh control:

| Attribute | Behaviour |
|-----------|-----------|
| `pk-refresh-root` | Only the root partial element refreshes |
| `pk-own-attrs` | Only the partial's own attributes refresh |
| `pk-no-refresh-attrs` | Attributes never refresh |
| `pk-no-refresh` | Element never refreshes (on any element) |
| `pk-refresh` | Force refresh (overrides no-refresh ancestor) |
| `pk-loading` | CSS class applied during partial reload |

## Request overrides

Pass request-time values with `:request.` prefix:

```piko
<pager is="pagination" :request.page="request.QueryParam('p')"></pager>
```

## LLM mistake checklist

- Note: a bare `:foo="..."` matching a declared prop is consumed as a prop only (not also rendered as an HTML attribute on the partial root). A bare attribute that does NOT match any declared prop forwards as a plain HTML attribute on the partial root. Use `:server.foo` to keep the value server-only and raise an error if the prop is not declared.
- Using `<slot>` instead of `<piko:slot>` in server partials (`<slot>` is for PKC components)
- Making `is` attribute dynamic (must be static - resolved at compile time)
- Forgetting to import the partial in the Go script block
- Using `piko.NoProps` when the partial needs props
- Passing props without `coerce:"true"` for non-string types
- Defining multiple default slots in one partial (only one allowed)

## Related

- `references/pk-file-format.md` - full .pk file structure and Props reference
- `references/template-syntax.md` - directives used inside slots
- `references/styling.md` - CSS scope inheritance with partials
