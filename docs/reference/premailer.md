---
title: Premailer
description: The CSS-inlining, validation, and Outlook-compatibility pipeline that prepares email HTML.
nav:
  sidebar:
    section: "reference"
    subsection: "email"
    order: 310
---

# Premailer

The premailer is the stage of the email pipeline that turns a PML-transformed template into email-client-safe HTML. It resolves CSS variables, expands shorthand properties, inlines CSS into `style` attributes, and validates against the email-client compatibility matrix. Rules that the premailer cannot inline (pseudo-classes, `@media` queries) stay in a `<style>` block in the body. Source of truth: [`internal/premailer/`](https://github.com/piko-sh/piko/tree/master/internal/premailer).

## Why a dedicated premailer

Email clients disagree on CSS support. Gmail strips `<style>` tags in `<head>`. Outlook ignores flexbox and grid. Yahoo Mail ignores shorthand `margin`. The only CSS that every major client reads is inline `style=""` attributes on elements. The premailer's job is to turn normal CSS in the template's `<style>` block into inline attributes on the matched elements.

Piko does not use an external premailer library. It shares the CSS parser with the rest of the framework (esbuild's CSS parser at `internal/esbuild/css_parser`), which gives it correct specificity calculation, selector matching, and shorthand expansion.

## Pipeline stages

The premailer runs ten stages in order. Each stage reads the template AST produced by PML and the extracted CSS.

### 1. Collection and validation

[`internal/premailer/premailer.go`](https://github.com/piko-sh/piko/blob/master/internal/premailer/premailer.go) `collectAndValidate()` scans the template for `<style>` tags, concatenates their content, and flags email-incompatible HTML elements (`<script>`, `<form>`, `<iframe>`, `<svg>`) as diagnostics. The pipeline marks the original `<style>` tags for removal.

### 2. Variable resolution

[`internal/premailer/resolver.go`](https://github.com/piko-sh/piko/blob/master/internal/premailer/resolver.go) substitutes every `var()` reference with a concrete value from the theme passed to `WithTheme(...)`. Email clients cannot evaluate CSS custom properties, so the resolver must inline every `var()`. Undefined variables surface as diagnostics.

### 3. CSS parsing and cascade

[`internal/premailer/rules.go`](https://github.com/piko-sh/piko/blob/master/internal/premailer/rules.go) parses the CSS into an AST, calculates specificity for each selector, and matches selectors against template nodes. The parser splits matches into two sets:

- **Inlineable rules** target plain elements, and the pipeline copies their declarations to `style` attributes.
- **Leftover rules** target pseudo-classes (`:hover`, `:focus`), pseudo-elements (`::before`, `::after`), `@media` queries, keyframes, and animations. The pipeline preserves these for a later stage because they cannot live inside inline attributes.

### 4. Shorthand expansion

[`internal/premailer/shorthand.go`](https://github.com/piko-sh/piko/blob/master/internal/premailer/shorthand.go) and its siblings expand shorthand properties to longhands:

- `margin: 10px 20px` becomes `margin-top: 10px; margin-right: 20px; margin-bottom: 10px; margin-left: 20px`.
- `border: 1px solid red` becomes `border-width`, `border-style`, and `border-color`.
- `background: #fff url(x) no-repeat` becomes its expanded components.

Outlook and Yahoo Mail ignore most shorthand. The longhand form works reliably.

### 5. Colour normalisation

[`internal/premailer/colours.go`](https://github.com/piko-sh/piko/blob/master/internal/premailer/colours.go) converts colour values to hex. `rgb()`, `rgba()`, `hsl()`, and named colours become `#RRGGBB` (or `#RRGGBBAA` when alpha is non-opaque). Hex is the most broadly supported colour format.

### 6. Style inlining

[`internal/premailer/style_application.go`](https://github.com/piko-sh/piko/blob/master/internal/premailer/style_application.go) walks the inlineable rules in specificity order and writes each declaration into the matched node's `style` attribute. The writer merges existing inline styles, and `!important` flags preserve priority.

### 7. Link-parameter injection

When the builder passes `WithLinkQueryParams(...)`, every `<a>` tag has the configured query parameters appended (typical use: UTM tags). URL parsing catches existing parameters so the injector skips duplicates.

### 8. Leftover-rule placement

Rules that resist inlining (pseudo-classes, media queries) emit as a `<style>` block at the bottom of the `<body>`. Gmail strips `<style>` in `<head>` but respects `<style>` in `<body>`. If the builder passes `WithMakeLeftoverImportant(true)`, the premailer marks every declaration in the leftover block `!important` so it overrides Gmail's own injected rules when a recipient forwards the email.

### 9. Pseudo-element resolution

Email clients do not render pseudo-element rules (`::before`, `::after`), and the premailer does not inline them. The output object preserves the resolved property maps (`ResolvedProperties.PseudoElements`) so downstream code can use them if needed.

### 10. Cleanup

[`internal/premailer/dom_cleanup.go`](https://github.com/piko-sh/piko/blob/master/internal/premailer/dom_cleanup.go) removes the original `<style>` tags (already processed), strips empty text nodes and comments, normalises anchor targets, and validates the resulting HTML.

## Diagnostics

Validation emits diagnostics (info, warning, error) for patterns that email clients reject or handle unreliably. The warnings surface during development. CI can log them or fail the build on them.

| Category | Examples |
|---|---|
| Unsupported layout | `display: flex`, `display: grid`, `position: absolute`, `float` |
| Unsupported effects | `transform`, `filter`, `animation`, `@keyframes`, `transition` |
| Unreliable visuals | `background-blend-mode`, `box-shadow` inside Outlook, `object-fit`, `clip-path` |
| Multi-column | `columns`, `column-gap`, `column-rule` |
| Bad HTML | `<script>`, `<form>`, `<iframe>`, `<svg>` |
| Unknown variables | `var(--missing)` |

Diagnostics carry severity, source location (line and column), and a short message. The pipeline returns them alongside the rendered HTML.

## Configuration

The premailer accepts options via the `wdk/email` service's builder, or directly for custom invocations.

| Option | Purpose |
|---|---|
| `WithTheme(map[string]string)` | CSS custom-property values used by `var()` resolution. |
| `WithLinkQueryParams(map[string]string)` | Query parameters appended to every `<a href>`. |
| `WithMakeLeftoverImportant(bool)` | Marks leftover-block declarations `!important`. |
| `WithBreakpoint(px int)` | Mobile breakpoint for the responsive column CSS. Default 480. |
| `WithBaseURL(string)` | Base URL for relative links and images (sets the `<base>` tag). |

## Performance

The premailer reuses the Piko generator's CSS parser. Projects that generate templates and bundle CSS in the same build pay no duplicated parse cost.

Inlining is O(rules x matched-nodes) per template. For the typical transactional email (fewer than 200 nodes, fewer than 100 rules) the stage runs in under a millisecond.

## Outputs

The premailer produces:

- **Rendered HTML**, with inline styles on every matched element and a leftover `<style>` block in the body.
- **Diagnostics**, the validation and resolution warnings.
- **Resolved properties map**, including pseudo-element styles for downstream use.
- **Asset references**, the list of images the PML transformer flagged for CID embedding.

## See also

- [PML components reference](pml-components.md) for the tag vocabulary consumed by the premailer.
- [Email API reference](email-api.md) for the service that invokes the premailer and sends the output.
- [Email templates how-to](../how-to/email-templates.md) for end-to-end authoring.
- [About email rendering](../explanation/about-email-rendering.md) for the design rationale.
- Source: [`internal/premailer/`](https://github.com/piko-sh/piko/tree/master/internal/premailer).
