---
title: Advanced templates
description: Advanced .pk template patterns including attribute merging, scoped CSS, fragment components, and troubleshooting
nav:
  sidebar:
    section: "guide"
    subsection: "advanced"
    order: 700
---

# Advanced templates

This guide covers advanced PK template techniques not found in the individual feature guides. You should already be familiar with [basic template syntax](/docs/guide/template-syntax) and [directives](/docs/guide/directives).

For foundational topics covered in dedicated guides, see:

- **[Partials](/docs/guide/partials)** → Importing components, passing props, prop struct tags (`validate`, `default`, `coerce`, `factory`), type coercion
- **[Slots](/docs/guide/slots)** → Default and named slots, fallback content, slot validation, context preservation
- **[Directives](/docs/guide/directives)** → `p-if`, `p-for`, `p-key`, `p-on`, `p-ref`, `p-class`, `p-style`, `p-show`, `p-model`, `p-bind`
- **[PK templates](/docs/guide/pk-templates)** → File structure, client-side scripts (JS/TS)
- **[Conditionals and loops](/docs/guide/conditionals-loops)** → Performance patterns, nesting best practices

## Attribute merging

When invoking a component, attributes are merged according to specific rules:

### ID attribute

The invocation's `id` **overrides** the component's `id`:

```piko
<!-- partials/card.pk -->
<template>
    <div id="original-id" class="card">...</div>
</template>

<!-- Invocation -->
<div is="card" id="custom-id"></div>

<!-- Output -->
<div id="custom-id" class="card">...</div>
```

### Class attribute

Classes are **merged** (partial classes preserved, invocation classes added):

```piko
<!-- partials/card.pk -->
<template>
    <div class="card base-style">...</div>
</template>

<!-- Invocation -->
<div is="card" class="highlighted special"></div>

<!-- Output -->
<div class="card base-style highlighted special">...</div>
```

### Other attributes

Other attributes from the invocation **override** matching attributes from the component, or are added if not present:

```piko
<!-- partials/card.pk -->
<template>
    <div class="card" aria-label="Default label">...</div>
</template>

<!-- Invocation -->
<div is="card" data-testid="my-card" aria-hidden="true"></div>

<!-- Output -->
<div class="card" aria-label="Default label" data-testid="my-card" aria-hidden="true">...</div>
```

### Attributes not merged

The following attributes are not passed through to the component:
- `is` - the partial invocation marker itself
- `server.*` - server-side only attributes
- `request.*` - request-scoped attributes

## Advanced scoped CSS

Each component's `<style>` block is automatically scoped to prevent cross-component conflicts. CSS selectors are transformed to include a scope attribute; for example, `.card` becomes `[partial~="partials_card_abc123"] .card`, which means styles only apply to elements within that partial.

### CSS aggregation

CSS from all imported partials is aggregated at the page level. If a page uses partials A and B, and partial A uses partial C, all three stylesheets are combined in the page output.

### Global and deep selectors

Use `:global()` to escape scoping for specific selectors:

```css
:global(.external-lib-class) {
    /* Applies globally, not scoped */
}
```

Use `:deep()` to target child component elements:

```css
:deep(.child-component-class) {
    /* Targets elements in child partials */
}
```

### Multiple style blocks

You can have multiple `<style>` blocks in a single file:

```piko
<style>
    .layout { display: grid; }
</style>

<style>
    .header { background: #333; }
</style>

<template>
    <div class="layout">
        <header class="header">...</header>
    </div>
</template>
```

## `p-show` vs `p-if`

Both directives control element visibility, but they work differently. See [directives](/docs/guide/directives) for full syntax.

| Directive | DOM presence | Use when |
|-----------|--------------|----------|
| `p-if` | Removed when false | Content rarely changes, or you want to avoid rendering overhead |
| `p-show` | Always present, hidden via CSS | Content toggles frequently, or you need to preserve element state |

When the expression is falsy, `p-show` hides elements with `display: none !important;`.

## Context directive (`p-context`)

The `p-context` directive provides a context identifier for key scoping. It resets the key namespace for child elements, useful for isolating hydration scopes:

```piko
<template>
    <section p-context="'user-profile'">
        <h1 p-text="state.Name"></h1>
        <div p-key="'details'">
            <p p-text="state.Email"></p>
        </div>
    </section>
</template>
```

The context value can be static (`'prefix'`) or dynamic (`state.ComponentName`).

## Fragment components

Components can have multiple root elements. When invoked, all root nodes are inserted:

```piko
<!-- partials/layout.pk -->
<template>
    <div>{{ state.Hello }}</div>
    <main><piko:slot></piko:slot></main>
    <div>{{ state.Goodbye }}</div>
</template>
```

When this partial is invoked, all three root elements are inserted at the invocation point.

## Partial refresh behaviour

When a partial is refreshed via JavaScript (using `piko.partial().reload()`), Piko uses intelligent DOM morphing to preserve state while updating content. This section explains the graduated control system for managing how attributes behave during refresh.

### Why this matters

When a partial is embedded in another partial, the parent can contribute attributes to the child's root element. JavaScript may also add dynamic attributes at runtime. During island refresh:

- **Parent-contributed attributes** would be lost (parent didn't re-render)
- **JS-added dynamic attributes** would be lost (no DOM state preservation)
- **CSS scopes** (`partial` attribute) lose parent context

The graduated refresh system solves these problems.

### Refresh levels

Piko provides four levels of control over how partial refresh handles attributes:

| Level | Attribute | Behaviour |
|-------|-----------|-----------|
| 0 (Default) | None | Children only - root element attrs preserved |
| 1 | `pk-refresh-root` | Root refreshed with CSS scope preservation |
| 2 | `pk-own-attrs="..."` | Only listed attrs update |
| 3 | `pk-no-refresh-attrs="..."` | All attrs update except listed |

### Level 0: content-only (default)

By default, partial refresh only updates children. The root element's attributes (including parent CSS scopes and JS-added attributes) are preserved:

```html
<div data-partial="cart" data-partial-src="/partials/cart">
  <!-- Only this content updates -->
</div>
```

**Best for:** Most cases where the container is stable.

```ts
// Default behaviour - children only
piko.partial('cart').reload();
```

### Level 1: root refresh with scope preservation

Use `pk-refresh-root` when the root element should update, but parent CSS scopes must be preserved:

```html
<div data-partial="cart" pk-refresh-root class="{{ dynamicClass }}">
  ...
</div>
```

**Best for:** When root element attrs change but you're nested in another partial.

The `partial` attribute's parent scopes are preserved:
- Before: `partial="child_xyz parent_abc"`
- Server returns: `partial="child_new"`
- After merge: `partial="child_new parent_abc"`

### Level 2: owned attributes

Declare which attributes your partial template owns. Only these update; JS-added attrs and parent contributions survive:

```html
<div data-partial="cart" pk-own-attrs="class,data-count">
  ...
</div>
```

**Best for:** Fine-grained control when partial owns some attrs but JS owns others.

```ts
// Can also override via API
piko.partial('cart').reloadWithOptions({
    level: 2,
    ownedAttrs: ['class', 'data-count']
});
```

### Level 3: preserve specific

List attributes that must NOT update (existing `pk-no-refresh-attrs` behaviour):

```html
<div data-partial="cart" pk-no-refresh-attrs="data-initialized,aria-expanded">
  ...
</div>
```

**Best for:** When most attrs should update but a few must survive.

### CSS scope inheritance

When partials are nested, the `partial` attribute contains space-separated scope IDs. The first value is the element's own scope, and subsequent values are parent scopes:

```html
<!-- Parent partial adds its scope -->
<div partial="child_xyz parent_abc grandparent_123">
```

During refresh, parent scopes are automatically preserved (at Level 1+):

```text
Before refresh: "child_xyz parent_abc grandparent_123"
Server returns: "child_new"
After merge:    "child_new parent_abc grandparent_123"
```

This ensures `:deep()` CSS from parent partials continues working after refresh.

### JavaScript API

The `piko.partial()` function provides two methods for reloading:

```ts
// Simple reload with query params
piko.partial('cart').reload({ highlight: 'new-item' });

// Advanced reload with options
piko.partial('cart').reloadWithOptions({
    data: { highlight: 'new-item' },
    level: 1,  // Override detected level
    ownedAttrs: ['class', 'data-count']  // For Level 2
});
```

### Choosing a level

| Scenario | Recommended Level |
|----------|-------------------|
| Simple content update | 0 (default) |
| Root class changes based on state | 1 (`pk-refresh-root`) |
| Mix of template and JS attributes | 2 (`pk-own-attrs`) |
| Preserve initialisation state | 3 (`pk-no-refresh-attrs`) |

### Element-level morph control

Independent of the refresh level system, you can mark individual elements within a partial to control how the DOM morpher handles them:

**`pk-no-refresh`**: Preserve an element and its entire subtree during partial refresh. The element's attributes and children will not be updated:

```html
<div data-partial="player">
  <span id="score">{{ state.Score }}</span>
  <span id="client-counter" pk-no-refresh data-count="0">0</span>
</div>
```

**`pk-refresh`**: Force-update an element even if a parent has `pk-no-refresh`. Use this to opt specific children back into morphing:

```html
<div pk-no-refresh>
  <span>This is preserved</span>
  <span pk-refresh>This updates normally</span>
</div>
```

### Form tracking opt-out

By default, Piko tracks form dirty state to warn users about unsaved changes. Use `pk-no-track` on a `<form>` to opt out:

```html
<form pk-no-track p-on:submit.prevent="action.search()">
  <input type="text" name="query">
  <button type="submit">Search</button>
</form>
```

**Best for:** Search forms, filter forms, and other forms where "unsaved changes" warnings are not relevant.

### Loading state

During async operations (action submissions, partial reloads), Piko automatically adds the `pk-loading` CSS class to the relevant element. You can use this to style loading states:

```css
.pk-loading {
  opacity: 0.6;
  pointer-events: none;
}
```

The class is removed when the operation completes.

### Internal framework attributes

These attributes are set automatically by the Piko runtime. You should not set them manually, but you may see them in browser devtools:

| Attribute | Purpose |
|-----------|---------|
| `pk-ev-bound` | Marks an element whose event handlers have been bound |
| `pk-sync-bound` | Marks a sync partial container as initialised |
| `pk-page` | Marks `<style>` blocks containing page-scoped CSS |
| `data-pk-tracked` | Marks a form as being tracked for dirty state |
| `data-pk-action-method` | Stores the HTTP method for action form submissions |
| `data-pk-partial` | Stores the hashed component identifier |
| `data-pk-style-key` | Deduplication key for injected style blocks |

## Complete component example

Here's an example combining multiple advanced features: props with validation, coercion, and defaults; scoped CSS; `p-class`; `p-show`; `p-if`/`p-else`; and named slots with fallbacks:

```piko
<!-- partials/user_card.pk -->
<script type="application/x-go">
package user_card

import "piko.sh/piko"

type User struct {
    ID       int
    Name     string
    Email    string
    IsActive bool
}

type Props struct {
    User      User   `prop:"user" validate:"required"`
    ShowEmail bool   `prop:"show-email" default:"true" coerce:"true"`
    Theme     string `prop:"theme" default:"light"`
}

type Response struct {
    User      User
    ShowEmail bool
    Theme     string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        User:      props.User,
        ShowEmail: props.ShowEmail,
        Theme:     props.Theme,
    }, piko.Metadata{}, nil
}
</script>

<style>
    .user-card {
        border: 1px solid #e0e0e0;
        border-radius: 8px;
        padding: 1rem;
    }
    .user-card.theme-dark {
        background: #1a1a1a;
        color: #fff;
    }
    .status-active { color: green; }
    .status-inactive { color: red; }
</style>

<template>
    <div
        class="user-card"
        p-class="{ 'theme-dark': state.Theme == 'dark' }"
        :data-user-id="state.User.ID">

        <header>
            <h3 p-text="state.User.Name"></h3>
            <span
                p-if="state.User.IsActive"
                class="status-active">Active</span>
            <span
                p-else
                class="status-inactive">Inactive</span>
        </header>

        <div p-show="state.ShowEmail" class="email">
            {{ state.User.Email }}
        </div>

        <footer>
            <piko:slot name="actions">
                <button>Default Action</button>
            </piko:slot>
        </footer>
    </div>
</template>
```

**Usage:**

```piko
<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    userCard "myapp/partials/user_card.pk"
)

type User struct {
    ID       int
    Name     string
    Email    string
    IsActive bool
}

type Response struct {
    Users []User
}

func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{
        Users: []User{
            {ID: 1, Name: "Alice", Email: "alice@example.com", IsActive: true},
            {ID: 2, Name: "Bob", Email: "bob@example.com", IsActive: false},
        },
    }, piko.Metadata{}, nil
}
</script>

<template>
    <div p-for="(_, user) in state.Users" p-key="user.ID">
        <userCard is="userCard"
            :user="user"
            theme="dark"
            show-email="false">

            <piko:slot name="actions">
                <button p-on:click="action.editUser(user.ID)">Edit</button>
                <button p-on:click="action.deleteUser(user.ID)">Delete</button>
            </piko:slot>
        </userCard>
    </div>
</template>
```

## Next steps

- [Slots](/docs/guide/slots) → Deep dive into the slots system
- [Partials](/docs/guide/partials) → Component reuse patterns
- [Directives](/docs/guide/directives) → All template directives
