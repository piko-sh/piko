---
title: Partials
description: Create reusable PK template components with partials
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 38
---

# Partials

Partials are reusable PK template components that help you build modular, maintainable templates. Think of them as server-side components that can receive props, project content via slots, and compose with other partials.

## Creating a partial

Partials are `.pk` files, typically placed in a `partials/` directory by convention.

**File**: `partials/button.pk`

```piko
<template>
  <button :class="`btn btn-${state.Type}`" :type="state.ButtonType">
    {{ state.Label }}
  </button>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    Label      string `prop:"label"`
    Type       string `prop:"type"`        // "primary", "secondary", etc.
    ButtonType string `prop:"button_type"` // "button", "submit", etc.
}

type Response struct {
    Label      string
    Type       string
    ButtonType string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        Label:      props.Label,
        Type:       props.Type,
        ButtonType: props.ButtonType,
    }, piko.Metadata{}, nil
}
</script>

<style>
.btn {
    padding: 0.75rem 1.5rem;
    border: none;
    border-radius: 0.375rem;
    font-weight: 600;
    cursor: pointer;
}

.btn-primary {
    background: #6F47EB;
    color: white;
}

.btn-secondary {
    background: #e5e7eb;
    color: #374151;
}
</style>
```

## Using a partial

### Import

Import the partial in your page's script block using a `.pk` import path:

```piko
<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    my_button "myapp/partials/button.pk"
)

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{}, nil
}
</script>
```

### Use in template

Reference the partial using the `is` attribute with your import alias:

```piko
<template>
  <div>
    <piko:partial is="my_button"
      :server.label="'Click Me'"
      :server.type="'primary'"
      :server.button_type="'button'"
    />

    <piko:partial is="my_button"
      :server.label="'Submit'"
      :server.type="'secondary'"
      :server.button_type="'submit'"
    />
  </div>
</template>
```

**Key points**:
- The `is` attribute specifies which partial to render (must match an import alias)
- Partials always use the `<piko:partial>` tag with the `is` attribute (`<piko:partial is="my_button">`)
- Props are passed with the `:server.prop_name` prefix for dynamic values
- Static string props can omit the prefix: `type="primary"` (with `coerce:"true"`)

## Props

Props are the primary way to pass data from a parent component to a partial. Piko provides struct tags to control prop behaviour at compile-time.

### Defining props

In your partial's script block, define a `Props` struct with struct tags:

```go
type Props struct {
    Title       string `prop:"title"`
    Description string `prop:"description"`
    Count       int    `prop:"count"`
    IsActive    bool   `prop:"is_active"`
}
```

### Passing props

From your page template, use the `:server.` prefix for dynamic expressions:

```piko
<piko:partial is="card"
  :server.title="state.Product.Name"
  :server.description="state.Product.Description"
  :server.count="state.CartCount"
  :server.is_active="state.IsActive"
/>
```

For static values with coercion enabled, you can use plain attributes:

```piko
<piko:partial is="card" count="5" is_active="true" />
```

### Prop struct tags reference

Piko supports these struct tags:

| Tag | Purpose | Example |
|-----|---------|---------|
| `prop:"name"` | Custom HTML attribute name | `prop:"card-title"` |
| `validate:"required"` | Compile-time required check | `validate:"required"` |
| `default:"value"` | Static default value | `default:"light"` |
| `factory:"FuncName"` | Factory function for complex defaults | `factory:"GetDefaults"` |
| `coerce:"true"` | Enable string-to-type coercion | `coerce:"true"` |
| `query:"param"` | Bind to URL query parameter | `query:"page"` |

---

### Custom attribute names (`prop:"name"`)

The `prop` tag maps a Go field to a custom HTML attribute name:

```go
type Props struct {
    Title       string `prop:"card-title"`      // HTML: card-title
    IsHighlight bool   `prop:"is-highlighted"`  // HTML: is-highlighted
}
```

**Usage:**

```piko
<piko:partial is="card" :server.card-title="'My Title'" :server.is-highlighted="true" />
```

If omitted, the attribute name defaults to the lowercase field name.

---

### Required props (`validate:"required"`)

Mark props as mandatory with `validate:"required"`. The compiler will emit an error if a required prop is not provided:

```go
type Props struct {
    Title string `prop:"title" validate:"required"`  // Must be provided
    Theme string `prop:"theme"`                       // Optional
}
```

**Compile-time error if omitted:**

```text
error: Missing required prop 'title' for component <card>
```

---

### Static default values (`default:"value"`)

Provide a default value that's used when the prop is omitted:

```go
type Props struct {
    Theme string `prop:"theme" default:"light"`
    Size  string `prop:"size" default:"medium"`
    Label string `prop:"label" default:"Click me"`
}
```

**Usage:**

```piko
<!-- Only provide theme, others use defaults -->
<piko:partial is="button" :server.theme="'dark'" />
<!-- Compiled as: {Theme: "dark", Size: "medium", Label: "Click me"} -->
```

The default value is injected at compile-time into the Props struct literal.

---

### Factory function defaults (`factory:"FuncName"`)

For complex types that cannot be expressed as string literals, use a factory function:

```go
// Factory function - must be zero-argument and return the prop type
func GetDefaultOptions() AvatarOptions {
    return AvatarOptions{
        Size:  48,
        Shape: "circle",
    }
}

type Props struct {
    Options AvatarOptions `prop:"options" factory:"GetDefaultOptions"`
}
```

**Generated code when prop is omitted:**

```go
props := Props{
    Options: GetDefaultOptions(),  // Factory called at render time
}
```

**Note:** You cannot use both `default` and `factory` on the same field. This will cause a compile-time error.

---

### Type coercion (`coerce:"true"`)

Enable automatic conversion from string literals to Go primitive types:

```go
type Props struct {
    IsActive bool    `prop:"is-active" coerce:"true"`
    Count    int     `prop:"count" coerce:"true"`
    Price    float64 `prop:"price" coerce:"true"`
}
```

**Usage with string literals (no `:server.` prefix needed):**

```piko
<piko:partial is="display" is-active="true" count="42" price="19.99" />
```

**Generated code:**

```go
props := Props{
    IsActive: true,   // "true" → bool
    Count:    42,     // "42" → int
    Price:    19.99,  // "19.99" → float64
}
```

The `coerce` tag can be set to `"true"` or left empty (`coerce:""`), both enable coercion.

---

### Query parameter binding (`query:"param"`)

Bind a prop to a URL query parameter as a fallback when the prop isn't passed by a parent component:

```go
type Props struct {
    Page        int    `prop:"page" query:"page" coerce:"true"`
    EnvID       string `prop:"env_id" query:"environment_id"`
    BlueprintID string `prop:"blueprint_id" query:"blueprint_id"`
}
```

When the page is requested with `?page=2&environment_id=prod`, props not explicitly passed will be populated from the query string.

**Notes:**
- Query binding requires `string` or `*string` types, unless `coerce:"true"` is also set
- Slice and map types are not supported for query binding
- Query values are only used as fallbacks; explicitly passed props take precedence

---

### Optional pointer props

Use pointer types for truly optional props. When omitted, the pointer is `nil`:

```go
type Props struct {
    Title    string              `prop:"title"`    // Zero value if omitted
    Subtitle *string             `prop:"subtitle"` // nil if omitted
    Profile  *models.UserProfile `prop:"profile"`  // Optional complex type
}
```

**Handling nil in your Render function:**

```go
func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    resp := Response{}

    if props.Profile != nil {
        resp.ProfileName = props.Profile.Name
    }

    if props.Subtitle != nil {
        resp.SubtitleText = *props.Subtitle
    }

    return resp, piko.Metadata{}, nil
}
```

**Passing a value to a pointer prop:**

```piko
<!-- Piko automatically takes the address -->
<piko:partial is="profile" :server.profile="state.UserProfile" />
```

The compiler generates `&pageData.UserProfile` when passing a non-pointer value to a pointer prop.

---

### Combining multiple tags

You can combine multiple tags on a single field:

```go
type Props struct {
    // Required with custom name
    Title string `prop:"card-title" validate:"required"`

    // Default with custom name
    Theme string `prop:"card-theme" default:"default"`

    // Coercion with default
    Priority int `prop:"priority" coerce:"true" default:"1"`

    // Coercion for boolean
    IsHighlighted bool `prop:"highlighted" coerce:"true"`

    // Factory for complex type
    Options models.Config `prop:"options" factory:"GetDefaultConfig"`

    // Query binding with coercion
    Page int `prop:"page" query:"page" coerce:"true"`
}
```

---

### Defaults in Render (dynamic defaults)

While struct tags are preferred for static defaults, you can set dynamic defaults in the `Render` function:

```go
func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    // Dynamic default based on request
    if props.Locale == "" {
        props.Locale = r.AcceptLanguage()
    }

    return Response{Locale: props.Locale}, piko.Metadata{}, nil
}
```

Use struct tags for static defaults, and `Render` logic for dynamic defaults.

## Slots

Slots let you pass content (not just data) to partials. For default slots, named slots, fallback content, slot validation, and context preservation, see [slots](/docs/guide/slots).

## Common patterns

### Layout partial

**Partial** (`partials/layout.pk`):

```piko
<template>
  <html>
    <head>
      <title>{{ state.PageTitle }} | MyApp</title>
      <meta name="description" :content="state.Description" />
    </head>
    <body>
      <nav class="navbar">
        <a href="/">Home</a>
        <a href="/about">About</a>
        <a href="/contact">Contact</a>
      </nav>

      <main class="container">
        <piko:slot />
      </main>

      <footer>
        <p>&copy; 2025 MyApp</p>
      </footer>
    </body>
  </html>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    PageTitle   string `prop:"page_title" validate:"required"`
    Description string `prop:"description" default:"Welcome to MyApp"`
}

type Response struct {
    PageTitle   string
    Description string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        PageTitle:   props.PageTitle,
        Description: props.Description,
    }, piko.Metadata{}, nil
}
</script>
```

**Usage**:

```piko
<template>
  <piko:partial is="layout" :server.page_title="'Home'" :server.description="'Welcome page'">
    <h1>Welcome!</h1>
    <p>This is my homepage content.</p>
  </piko:partial>
</template>
```

### Card component

```piko
<template>
  <div :class="`card ${state.Variant}`">
    <div p-if="state.ImageUrl != ''" class="card-image">
      <img :src="state.ImageUrl" :alt="state.ImageAlt" />
    </div>

    <div class="card-content">
      <h3 p-if="state.Title != ''" class="card-title">
        {{ state.Title }}
      </h3>

      <piko:slot />
    </div>

    <div p-if="state.ShowActions" class="card-actions">
      <piko:slot name="actions">
        <button>Learn More</button>
      </piko:slot>
    </div>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    Title       string `prop:"title"`
    ImageUrl    string `prop:"image_url"`
    ImageAlt    string `prop:"image_alt"`
    Variant     string `prop:"variant" default:"default"`
    ShowActions bool   `prop:"show_actions"`
}

type Response struct {
    Title       string
    ImageUrl    string
    ImageAlt    string
    Variant     string
    ShowActions bool
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        Title:       props.Title,
        ImageUrl:    props.ImageUrl,
        ImageAlt:    props.ImageAlt,
        Variant:     props.Variant,
        ShowActions: props.ShowActions,
    }, piko.Metadata{}, nil
}
</script>
```

### Alert component

```piko
<template>
  <div :class="`alert alert-${state.Type}`" role="alert">
    <div class="alert-icon">
      <span p-if="state.Type == 'success'">✓</span>
      <span p-else-if="state.Type == 'error'">✗</span>
      <span p-else-if="state.Type == 'warning'">⚠</span>
      <span p-else>ℹ</span>
    </div>

    <div class="alert-content">
      <strong p-if="state.Title != ''">{{ state.Title }}</strong>
      <piko:slot />
    </div>

    <button p-if="state.Dismissible" class="alert-close">×</button>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    Type        string `prop:"type" default:"info"`
    Title       string `prop:"title"`
    Dismissible bool   `prop:"dismissible"`
}

type Response struct {
    Type        string
    Title       string
    Dismissible bool
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        Type:        props.Type,
        Title:       props.Title,
        Dismissible: props.Dismissible,
    }, piko.Metadata{}, nil
}
</script>
```

### Modal component

```piko
<template>
  <div class="modal-overlay">
    <div :class="`modal modal-${state.Size}`">
      <div class="modal-header">
        <h2>{{ state.Title }}</h2>
        <button class="modal-close">×</button>
      </div>

      <div class="modal-body">
        <piko:slot />
      </div>

      <div class="modal-footer">
        <piko:slot name="footer">
          <button>Close</button>
        </piko:slot>
      </div>
    </div>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

type Props struct {
    Title string `prop:"title" validate:"required"`
    Size  string `prop:"size" default:"medium"`
}

type Response struct {
    Title string
    Size  string
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        Title: props.Title,
        Size:  props.Size,
    }, piko.Metadata{}, nil
}
</script>
```

## Nested partials

Partials can use other partials:

**Partial** (`partials/product-card.pk`):

```piko
<template>
  <piko:partial is="card" :server.title="state.ProductName">
    <p>{{ state.ProductDescription }}</p>
    <p class="price">${{ state.ProductPrice }}</p>

    <piko:slot name="actions">
      <piko:partial is="my_button"
        :server.label="'Add to Cart'"
        :server.type="'primary'"
      />
    </piko:slot>
  </piko:partial>
</template>

<script type="application/x-go">
package main

import (
    "piko.sh/piko"
    card "myapp/partials/card.pk"
    my_button "myapp/partials/button.pk"
)

type Product struct {
    Name        string
    Description string
    Price       float64
}

type Props struct {
    Product Product `prop:"product"`
}

type Response struct {
    ProductName        string
    ProductDescription string
    ProductPrice       float64
}

func Render(r *piko.RequestData, props Props) (Response, piko.Metadata, error) {
    return Response{
        ProductName:        props.Product.Name,
        ProductDescription: props.Product.Description,
        ProductPrice:       props.Product.Price,
    }, piko.Metadata{}, nil
}
</script>
```

## Partial refresh behaviour

When a partial is refreshed via JavaScript (using `piko.partial().reload()`), Piko uses intelligent DOM morphing to preserve state while updating content. This includes graduated control over how attributes behave during refresh (4 levels from content-only to full attribute control), CSS scope inheritance for nested partials, loading states, and form tracking.

For the full refresh API including `pk-refresh-root`, `pk-own-attrs`, `pk-no-refresh-attrs`, `pk-no-refresh`, `pk-refresh`, `pk-no-track`, and the JavaScript `piko.partial()` API, see [advanced templates](/docs/guide/advanced-templates).

## Next steps

- [Template syntax](/docs/guide/template-syntax) → Using props in templates
