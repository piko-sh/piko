---
title: PML components
description: Every `PML` component, its attributes, and the HTML it renders.
nav:
  sidebar:
    section: "reference"
    subsection: "email"
    order: 300
---

# PML components

`PML` (`Piko Mail Language`) is the tag vocabulary used inside email `.pk` templates. Each `PML` tag renders to email-safe HTML with table-based layout, automatic Outlook VML fallbacks for buttons and backgrounds, and responsive mobile behaviour. This page documents every tag and its attributes. For task recipes see the [email templates how-to](../how-to/email-templates.md). For the rendering pipeline that wraps this vocabulary see the [premailer reference](premailer.md). Source of truth: [`internal/pml/pml_components/`](https://github.com/piko-sh/piko/tree/master/internal/pml/pml_components).

## Layout hierarchy

PML enforces a parent-child structure that mirrors the HTML tables it emits:

```
<pml-body>
├── <pml-container>           # Optional grouping wrapper
│   └── <pml-row>
│       └── <pml-col>
│           ├── <pml-p>
│           ├── <pml-button>
│           ├── <pml-img>
│           ├── <pml-br>
│           ├── <pml-hr>
│           ├── <pml-ul> / <pml-ol>
│           │   └── <pml-li>
│           └── <pml-hero>
└── <pml-row>                 # Rows can live directly in body
```

The validator at [`internal/pml/pml_domain/validator.go`](https://github.com/piko-sh/piko/blob/master/internal/pml/pml_domain/validator.go) rejects templates that break this structure at build time.

## Structural components

### `pml-container`

Groups multiple `pml-row`s under a shared background without adding extra padding.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `background-color` | colour | | Container background. |
| `background-url` | string | | Background image; triggers Outlook VML fallback. |
| `background-repeat` | `repeat` \| `no-repeat` | `repeat` | |
| `background-size` | string | `auto` | |
| `background-position` | string | `top center` | |
| `border` | string | | Shorthand border. |
| `border-radius` | unit | | |
| `direction` | `ltr` \| `rtl` | `ltr` | |
| `full-width` | boolean | `false` | Spans viewport instead of 600 px. |
| `padding` | unit | `0` | Overrides the row default so grouped sections do not double-pad. |
| `text-align` | `left` \| `center` \| `right` | `center` | |

**Renders as** nested `<table>` with `<tbody>`/`<tr>`/`<td>`, equivalent to `pml-row` with `stack-children="true"`.

**Allowed parents**: `pml-body`.  
**Allowed children**: `pml-row`.

### `pml-row`

A horizontal strip of content. Columns inside sit side by side on desktop and stack on mobile.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `background-color` | colour | | |
| `background-url` | string | | Triggers a `<v:rect>` VML block for Outlook. |
| `background-repeat` | `repeat` \| `no-repeat` | `repeat` | |
| `background-size` | string | `auto` | |
| `background-position` | string | `top center` | Accepts `background-position-x` / `-y` overrides. |
| `border` | string | | |
| `border-radius` | unit | | |
| `css-class` | string | | Added to the wrapper. |
| `direction` | `ltr` \| `rtl` | `ltr` | |
| `full-width` | boolean | `false` | Makes the background span the viewport; the content stays within 600 px. |
| `padding` | unit | `20px` | Outer padding. |
| `padding-top/-right/-bottom/-left` | unit | | Individual overrides. |
| `stack-children` | boolean | `false` | Force mobile stacking even on desktop. |
| `text-align` | `left` \| `center` \| `right` | `center` | |
| `text-padding` | unit | `0` | Padding applied around text-only cells. |

**Renders as** an outer `<table>` (full-width mode) or a boxed `<div>` + table with Outlook `<!--[if mso | IE]>` conditionals wrapping each child column in a `<tr>`/`<td>`.

**Allowed parents**: `pml-body`, `pml-container`.  
**Allowed children**: `pml-col`, `pml-no-stack`.

### `pml-col`

A vertical cell inside a row.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `background-color` | colour | | Outer-cell background. |
| `inner-background-color` | colour | | Inner-content background. |
| `border` | string | | |
| `inner-border` | string | | Applied to the gutter table. |
| `border-radius` | unit | | |
| `inner-border-radius` | unit | | |
| `padding` | unit | `0` | |
| `vertical-align` | `top` \| `middle` \| `bottom` | `top` | |
| `width` | unit or `%` | auto | When omitted, equal width splits among siblings. |
| `direction` | `ltr` \| `rtl` | `ltr` | |
| `css-class` | string | | |

**Renders as** a `<div>` with `display: inline-block`. The parent `pml-row` inserts Outlook conditional tables so columns stay side-by-side in Outlook.

**Responsive behaviour.** The compiler emits a CSS class named `pml-col-N` (for example `pml-col-50` when `width="50%"`). On viewports below the mobile breakpoint, that class forces `display: block` with full width.

**Allowed parents**: `pml-row`, `pml-no-stack`.  
**Allowed children**: `pml-p`, `pml-button`, `pml-img`, `pml-hero`, `pml-br`, `pml-ul`, `pml-ol`, `pml-hr`.

### `pml-no-stack`

Opts its `pml-col` children out of mobile stacking.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| No attributes. | | | |

**Allowed parents**: `pml-body`, `pml-container`.  
**Allowed children**: `pml-col`.

## Hero

### `pml-hero`

A full-width banner that layers content over a background image.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `mode` | `fluid-height` \| `fixed-height` | | `fixed-height` requires `height`. |
| `height` | unit | | Required for `fixed-height`. |
| `background-url` | string | | |
| `background-width` | unit | | Image intrinsic width. |
| `background-height` | unit | | Image intrinsic height. |
| `background-position` | string | `center` | |
| `background-size` | string | `cover` | |
| `background-color` | colour | | Fallback background behind the image. |
| `container-background-color` | colour | | |
| `inner-background-color` | colour | | |
| `padding` | unit | | Outer padding. |
| `inner-padding` | unit | | Content padding. |
| `vertical-align` | `top` \| `middle` \| `bottom` | | |
| `align` | `left` \| `center` \| `right` | | |
| `border-radius` | string | | |

**Allowed parents**: `pml-body`, `pml-container`.  
**Allowed children**: `pml-p`, `pml-button`, `pml-img`.

## Content components

### `pml-p`

A paragraph with text styling.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `align` | `left` \| `center` \| `right` \| `justify` | `left` | |
| `color` | colour | `#000000` | |
| `background-color` | colour | | |
| `font-family` | string | `Ubuntu, Helvetica, Arial, sans-serif` | |
| `font-size` | unit | `13px` | |
| `font-style` | string | `normal` | |
| `font-weight` | string | `normal` | |
| `line-height` | string | `1` | |
| `padding` | unit | `10px 25px` | |

**Renders as** a `<td>` inside a wrapper `<table>`. The renderer passes inline HTML inside (`<b>`, `<i>`, `<a>`, `<span>`) through unchanged.

**Allowed parents**: `pml-col`, `pml-hero`.

### `pml-button`

A call-to-action button. Generates a VML `<v:roundrect>` fallback for Outlook.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `href` | string | | Required for a linked button. |
| `target` | string | `_blank` | |
| `rel` | string | | |
| `title` | string | | |
| `align` | `left` \| `center` \| `right` | `center` | |
| `background-color` | colour | `#414141` | |
| `container-background-color` | colour | | Applied to the button's container cell. |
| `color` | colour | `#ffffff` | Text colour. |
| `border` | string | `none` | |
| `border-radius` | unit | `3px` | |
| `font-family` | string | `Ubuntu, Helvetica, Arial, sans-serif` | |
| `font-size` | unit | `13px` | |
| `font-weight` | string | `normal` | |
| `height` | unit | | |
| `width` | unit | | |
| `inner-padding` | unit | `10px 25px` | Padding around the text. |
| `padding` | unit | `10px 25px` | Outer padding on the container cell. |
| `line-height` | unit | `120%` | |
| `text-decoration` | string | `none` | |
| `text-transform` | string | `none` | |
| `vertical-align` | `top` \| `middle` \| `bottom` | `middle` | |

**Allowed parents**: `pml-col`, `pml-hero`.  
**Children.** Plain text or inline HTML (span, strong, em). Complex nested HTML does not survive the VML fallback, so keep button labels simple.

### `pml-img`

An image, optionally linked, with responsive and high-DPI support.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `src` | string | | Required. |
| `alt` | string | | |
| `href` | string | | Wraps the image in an `<a>`. |
| `target`, `rel`, `title`, `name` | string | | |
| `width` | unit | `auto` | |
| `height` | unit | `auto` | |
| `max-height` | unit | | |
| `align` | `left` \| `center` \| `right` | `center` | |
| `border` / `border-*` | string | `none` | |
| `border-radius` | unit | | |
| `padding` | unit | `10px 25px` | |
| `padding-*` | unit | | |
| `container-background-color` | colour | | |
| `font-size` | unit | `0` | |
| `fluid-on-mobile` | boolean | `false` | Makes the image 100 % wide below the mobile breakpoint. |
| `full-width` | boolean | `false` | Always 100 % wide. |
| `srcset` | string | | Browser-native responsive-image `srcset`. |
| `sizes` | string | | |
| `densities` | string | | Shortcut: `"x1 x2"` generates an automatic `srcset` at 1x and 2x. |
| `profile` | string | | Asset transform profile name for the Piko asset pipeline. |

**CID embedding.** When rendering the template for an email, the pipeline automatically rewrites `src` references to project assets as `cid:` URLs and attaches the assets to the outgoing message. The `EmailAssetRegistry` inside the PML transformer collects these.

**Allowed parents**: `pml-col`, `pml-hero`.

### `pml-br`

A context-sensitive line break or vertical spacer.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `height` | unit | `20px` | Spacer height when used inside a column. |
| `padding` | unit | | |
| `css-class` | string | | |

**Rendering depends on parent**:

- Inside `pml-col` or `pml-hero`: renders as a `<table>` with an empty `<td>` styled to the given `height`.
- Inside `pml-p` or `pml-li`: renders as a plain `<br>`.

**Allowed parents.** Any.

### `pml-hr`

A horizontal rule.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `border` | string | `4px solid #000000` | Short-hand for colour, width, style. |
| `border-color` | colour | | |
| `border-width` | unit | | |
| `border-style` | string | | |
| `padding` | unit | `10px 25px` | |
| `align` | `left` \| `center` \| `right` | `center` | |
| `css-class` | string | | |

**Allowed parents**: `pml-col`, `pml-hero`.

## List components

### `pml-ul`

An unordered list.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `align`, `color`, `font-family`, `font-size`, `line-height`, `padding` | inherited | | Same set as `pml-p`. |

**Allowed parents**: `pml-col`, `pml-hero`.  
**Allowed children**: `pml-li`.

### `pml-ol`

An ordered list.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| `list-style` | `ordered` \| `unordered` | `ordered` | `pml-ul` is the same tag with `unordered`. |
| `align`, `color`, `font-family`, `font-size`, `line-height`, `padding` | inherited | | |

**Allowed parents**: `pml-col`, `pml-hero`.  
**Allowed children**: `pml-li`.

### `pml-li`

A list item.

| Attribute | Type | Default | Effect |
|---|---|---|---|
| Inherits text styling from the parent list. | | | |

**Allowed parents**: `pml-ul`, `pml-ol`.

## Outlook VML fallbacks

Two tags automatically emit Microsoft-specific VML markup wrapped in `<!--[if mso | IE]>` conditionals:

| Component | VML emitted | Purpose |
|---|---|---|
| `pml-row` with `background-url` | `<v:rect>` with `<v:fill>` | Renders the background image in Outlook desktop. |
| `pml-button` with `href` | `<v:roundrect>` with `<v:textbox>` | Renders a clickable pill button in Outlook desktop. |

Outside Outlook, the conditionals are invisible comments.

## Mobile breakpoint

The default breakpoint is 480 px. Below it:

- `pml-col`s stack full-width unless the containing row uses `stack-children="false"` or sits inside `pml-no-stack`.
- `pml-img` with `fluid-on-mobile="true"` stretches to 100 % width.

Override the breakpoint through premailer config (see the [premailer reference](premailer.md)).

## See also

- [Email templates how-to](../how-to/email-templates.md) for composing emails end-to-end.
- [Premailer reference](premailer.md) for the CSS-inlining and validation pipeline.
- [Email API reference](email-api.md) for the service that sends rendered templates.
- [About email rendering](../explanation/about-email-rendering.md) for the design rationale and MJML comparison.
- [Scenario 026: email contact form](../showcase/026-email-contact.md) for a runnable walk-through.
