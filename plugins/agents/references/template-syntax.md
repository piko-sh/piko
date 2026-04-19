# Template Syntax

Use this guide when writing template expressions, directives, or interpolation in `.pk` or `.pkc` files.

## Expression language

Piko has its own expression language - **not JavaScript**. Custom, JavaScript-like with Go-influenced conventions. Use only the operators and features documented here.

## Interpolation

Display data with double curly braces **inside element content only**. All output is HTML-escaped by default:

```piko
<h1>{{ state.Title }}</h1>
<p>Count: {{ state.Count }}</p>
<span>{{ state.FirstName + " " + state.LastName }}</span>
```

**CRITICAL**: `{{ }}` is ONLY for text content between tags. It is NOT valid inside attributes:

```piko
<!-- WRONG - {{ }} inside attributes is NOT valid Piko -->
<a href="{{ state.URL }}">Link</a>
<img src="{{ state.ImageURL }}" />
<div class="{{ state.ClassName }}">

<!-- CORRECT - use : prefix for dynamic attributes -->
<a :href="state.URL">Link</a>
<img :src="state.ImageURL" />
<div :class="state.ClassName">
```

## Attribute binding rules

- **Plain attributes** are static text: `class="container"`, `id="header"`
- **`:` prefix** makes an attribute dynamic (an expression): `:href="state.URL"`, `:class="state.ClassName"`
- **`class` and `style` always merge**: static and dynamic concatenate (`class="x" :class="'y'"` → `class="x y"`). Same for `style` / `:style` / `p-style`
- **Other attributes are replaced**: dynamic binding replaces static value
- **Directives** (`p-if`, `p-for`, `p-show`, `p-class`, `p-style`, `p-on`, `p-event`, `p-model`, `p-text`, `p-html`, `p-bind`) are expressions by default
- **Exceptions**: `p-ref` and `p-slot` accept plain text only, not expressions

## Directives reference

### p-if / p-else-if / p-else

Conditional rendering - removes elements from the DOM when false:

```piko
<div p-if="state.IsLoggedIn">Welcome back!</div>
<div p-else-if="state.IsGuest">Hello, guest!</div>
<div p-else>Please log in.</div>
```

Rules:
- `p-else-if` and `p-else` must immediately follow `p-if`/`p-else-if` at the same level
- **Type-strict**: `p-if` requires a boolean. Use `~` to coerce: `p-if="~state.Name"`
- Truthy coercion with `~`: `""`, `"0"`, `"false"` (case-insensitive), `0`, `nil`/`null`, `false` → falsy; else truthy

### p-for

Loop over slices, maps, or strings. **The order is `(index, item)` - index first, like Go**, not `(item, index)` like JavaScript:

```piko
<!-- Value only -->
<li p-for="item in state.Items" p-key="item.ID">{{ item.Name }}</li>

<!-- Index + value (index FIRST, like Go) -->
<li p-for="(i, item) in state.Items" p-key="item.ID">{{ i }}: {{ item.Name }}</li>

<!-- Map iteration (key first, value second; string keys sort alphabetically, other key types are non-deterministic) -->
<li p-for="(key, val) in state.Settings" p-key="key">{{ key }}: {{ val }}</li>
```

**Always use `p-key`** with a unique identifier for efficient DOM reconciliation.

Use `<template p-for>` to loop without a wrapper element:

```piko
<template p-for="item in state.Items" p-key="item.ID">
  <dt>{{ item.Label }}</dt>
  <dd>{{ item.Value }}</dd>
</template>
```

Nil-safe: loops silently skip nil collections.

### p-show

Toggle visibility via CSS (`display: none`) without removing from DOM:

```piko
<p p-show="state.IsVisible">Shown or hidden</p>
```

Use `p-show` when toggling frequently. Use `p-if` when the condition rarely changes.

### p-bind / : (attribute binding)

Bind dynamic values to HTML attributes:

```piko
<a :href="state.URL">Link</a>
<img :src="state.ImageURL" :alt="state.ImageAlt" />
<button :disabled="state.IsSubmitting">Submit</button>
```

**Boolean attribute rendering**: `true` renders as bare attribute (`disabled` or `disabled=""`), not `disabled="true"`. `false`/`nil` omits entirely (HTML spec).

```piko
<!-- state.IsSubmitting = true  → <button disabled="">Submit</button> -->
<!-- state.IsSubmitting = false → <button>Submit</button> -->
<button :disabled="state.IsSubmitting">Submit</button>
```

Template literals work:

```piko
<div :id="`item-${state.ID}`" :class="`category-${state.Type}`"></div>
```

### p-on (event handling)

Bind event listeners to functions in the **JavaScript/TypeScript** script block (not Go). In `.pk` files targets client-side `<script>`; in `.pkc` files targets the component script.

Three handler types:

```piko
<!-- Client-side function (no prefix) -->
<button p-on:click="handleClick">Click</button>

<!-- Server action (action. prefix) -->
<form p-on:submit.prevent="action.contact_submit()">

<!-- Helper function (helpers. prefix) -->
<button p-on:click="helpers.reloadPartial('#target')">Refresh</button>
```

**Three calling conventions**:

1. **No parentheses** - `p-on:click="myFn"` - event passed as first arg implicitly
2. **Empty parentheses** - `p-on:click="myFn()"` - no args passed
3. **With arguments** - `p-on:click="myFn('save', $event)"` - you control args; `$event` places event in any position

```piko
<!-- Event passed implicitly as first arg -->
<button p-on:click="handleClick">Click</button>

<!-- No args passed at all -->
<button p-on:click="handleClick()">Click</button>

<!-- $event passed explicitly in second position -->
<button p-on:click="handleClick('save', $event)">Save</button>
```

**`$form` special value**: Pass form data as a `FormDataHandle`:

```piko
<form p-on:submit.prevent="handleSubmit($form)">
```

**IMPORTANT**: `$event` and `$form` are opaque - cannot access properties (e.g. `$event.target` invalid). Pass to a function and access there.

**Event modifiers** are shorthand suffixes (see [Event modifiers](#event-modifiers)):

```piko
<form p-on:submit.prevent="handleSubmit">
<a p-on:click.prevent.stop="navigate()">Link</a>
```

Cross-partial calls:

```piko
<button p-on:click="@modal.open()">Open Modal</button>
```

### p-event (custom DOM events)

Listens for custom DOM events from a child component. Same prefix scheme, modifiers, and reserved identifiers as `p-on:`:

```piko
<my-component p-event:update="onUpdate($event)"></my-component>
<my-component p-event:save.once="action.handleSave($form)"></my-component>
```

Use `p-on:` for native DOM events; `p-event:` for custom events from child components.

### p-model

Two-way binding for form inputs:

```piko
<input type="text" p-model="state.name" />
<input type="checkbox" p-model="state.is_checked" />
```

### p-class

Conditional class binding with object or array syntax. Both `p-class` and `:class` merge with static `class`:

```piko
<!-- Object form: keys are class names, values are boolean expressions -->
<div class="container" p-class="{ active: state.IsActive, 'text-danger': state.HasError }">

<!-- Single dynamic class via :class also merges with static class -->
<div class="btn" :class="state.IsActive ? 'btn--active' : ''">

<!-- Array form -->
<div p-class="['card', state.IsFeatured ? 'card--featured' : '']">
```

`p-class` does not accept a colon argument suffix. Use the object form to toggle named classes.

### p-style

Dynamic inline styles. Like `p-class`, `p-style` **merges** with static `style`:

```piko
<p style="font-weight: bold" p-style="{ color: state.TextColour, fontSize: state.Size + 'px' }">Styled</p>
```

### p-text

Set text content (auto-escaped):

```piko
<p p-text="state.Message"></p>
```

### p-html

Set raw HTML content (**use with caution** - bypasses escaping):

```piko
<div p-html="state.RichContent"></div>
```

### p-ref

Get element references for client-side access:

```piko
<input p-ref="myInput" />
```

### p-key

Unique identifier for loop reconciliation:

```piko
<li p-for="item in state.Items" p-key="item.ID">{{ item.Name }}</li>
```

## Expression operators

| Operator | Description | Example |
|----------|-------------|---------|
| `+`, `-`, `*`, `/`, `%` | Arithmetic | `state.Price * state.Quantity` |
| `==`, `!=` | Strict equality (type must match) | `state.Status == "active"` |
| `~=`, `!~=` | Loose equality (type coercion) | `state.Count ~= "5"` |
| `<`, `>`, `<=`, `>=` | Comparison | `state.Age >= 18` |
| `&&`, `\|\|`, `!` | Logical | `state.A && !state.B` |
| `~` (prefix) | Truthy coercion (non-bool to bool) | `~state.Name` (true if non-empty) |
| `??` | Nullish coalescing | `state.Name ?? "Anonymous"` |
| `?.` | Optional chaining | `state.User?.Name` |
| `? :` | Ternary | `state.IsAdmin ? "Admin" : "User"` |

**Important**: `===` does not exist. Use `==` for strict equality and `~=` for loose equality.

## Built-in functions

Piko uses **Go-style built-in functions**, not JavaScript property access:

```piko
<!-- WRONG - JavaScript style -->
<span>{{ state.Items.length }}</span>
<span>{{ state.Name.toString() }}</span>
<span>{{ parseInt(state.Value) }}</span>

<!-- CORRECT - Go-style built-in functions -->
<span>{{ len(state.Items) }}</span>
<span>{{ string(state.Value) }}</span>
<span>{{ int(state.Value) }}</span>
```

| Function | Purpose | Example |
|----------|---------|---------|
| `len(x)` | Length of slice, array, string, or map | `len(state.Items) > 0` |
| `cap(x)` | Capacity of a slice | `cap(state.Buffer)` |
| `append(slice, items...)` | Append elements to a slice | `append(state.List, item)` |
| `min(a, b)` | Minimum value | `min(state.X, state.Y)` |
| `max(a, b)` | Maximum value | `max(state.X, state.Y)` |
| `string(x)` | Convert to string | `string(state.Count)` |
| `int(x)` | Convert to int | `int(state.FloatVal)` |
| `int64(x)` / `int32(x)` / `int16(x)` | Convert to sized int | `int64(state.ID)` |
| `float(x)` | Convert to float64 | `float(state.IntVal)` |
| `float64(x)` / `float32(x)` | Convert to sized float | `float32(state.Precise)` |
| `bool(x)` | Convert to boolean | `bool(state.Flag)` |
| `decimal(x)` | Convert to Decimal | `decimal(state.Price)` |
| `bigint(x)` | Convert to BigInt | `bigint(state.Large)` |

## Built-in literal types

| Syntax | Type | Example |
|--------|------|---------|
| `99.99d` | Decimal | `state.Price > 10.00d` |
| `d'2025-01-15'` | Date | `state.Date > d'2025-01-01'` |
| `t'09:30:00'` | Time | `state.Time < t'17:00:00'` |
| `dt'2025-01-15T09:30:00'` | DateTime | |
| `du'1h30m'` | Duration | |
| `123n` | BigInt | |
| `r'a'` | Rune | |

## Template literals

Use backticks with `${expr}`:

```piko
<a :href="`/users/${state.UserID}/profile`">Profile</a>
<span>{{ `Hello, ${state.Name}!` }}</span>
```

## nil and null

`nil` and `null` are interchangeable in Piko expressions.

## Event modifiers

| Modifier | Description |
|----------|-------------|
| `.prevent` | Calls `event.preventDefault()` before the handler |
| `.stop` | Calls `event.stopPropagation()` before the handler |
| `.once` | Removes the listener after first invocation |
| `.self` | Only fires when `event.target === event.currentTarget` |
| `.passive` | Registers the listener with `{ passive: true }` |
| `.capture` | Registers the listener in the capture phase |

```piko
<form p-on:submit.prevent="action.save()">
<a p-on:click.prevent.stop="navigate()">Link</a>
<div p-on:scroll.passive="onScroll">Scrollable</div>
```

## Built-in elements

### piko:a (SPA navigation)

Renders as `<a>` but intercepts clicks for SPA navigation. Supports locale-aware URL transformation:

```piko
<piko:a href="/about" class="nav-link">About Us</piko:a>
<piko:a :href="`/blog/${state.Slug}`">Read More</piko:a>
<piko:a href="/about" lang="fr">Voir en français</piko:a>
```

**IMPORTANT**: `piko:` prefix is stripped in rendered HTML - `<piko:a>` becomes `<a>`. Target plain `<a>` in CSS/JS. The rendered element has a `piko:a=""` marker (target via `a[piko\\:a]`).

| Attribute | Purpose |
|-----------|---------|
| `href` | Target URL (supports `:href` for dynamic binding) |
| `lang` | Override locale; empty string disables locale transformation |
| Standard attrs | `class`, `id`, `target`, `rel`, etc. pass through |

Links work without JS (graceful degradation). External URLs, `mailto:`, `tel:`, `#fragment` links are never intercepted.

### piko:img (optimised images)

Renders as `<img>` with automatic path transformation, `srcset` generation, and format variants:

```piko
<piko:img src="mymodule/assets/hero.jpg" alt="Hero"
    widths="640, 1280, 1920" formats="webp, avif, jpg"
    sizes="100vw" loading="lazy" />

<!-- CMS media with variant -->
<piko:img :src="state.HeroImage" variant="thumb_200" alt="Hero" />
```

**IMPORTANT**: `piko:` prefix is stripped - `<piko:img>` becomes `<img>`. Target `img` in CSS/JS.

| Attribute | Purpose |
|-----------|---------|
| `src` / `:src` | Image path (auto-prefixed with `/_piko/assets/`) |
| `alt` | Alt text (required) |
| `widths` | Comma-separated widths for srcset (e.g. `"320, 640, 1280"`) |
| `densities` | Comma-separated densities for srcset (e.g. `"1x, 2x"`) - use EITHER widths OR densities |
| `formats` | Comma-separated formats (e.g. `"webp, avif, jpg"`) |
| `sizes` | CSS sizes attribute |
| `variant` | Select specific CMS media variant by name |
| `loading` | `"lazy"` or `"eager"` |

### piko:picture (multi-format images)

Renders as `<picture>` with per-format `<source>` and a fallback `<img>`. Use when the browser should pick between formats (AVIF + WebP + JPEG):

```piko
<piko:picture src="mymodule/assets/hero.jpg" alt="Hero"
    widths="640, 1280" formats="avif, webp"
    sizes="100vw" />

<!-- CMS media with variant -->
<piko:picture :src="state.HeroImage" variant="w1200" alt="Hero" />
```

**IMPORTANT**: `<piko:picture>` becomes `<picture>`. Passthrough attributes (`alt`, `class`, `loading`) are placed on the fallback `<img>`, not on `<picture>` or `<source>`.

| Attribute | Purpose |
|-----------|---------|
| `src` / `:src` | Image path (auto-prefixed with `/_piko/assets/`) |
| `alt` | Alt text on fallback `<img>` (required) |
| `widths` | Comma-separated widths for srcset (e.g. `"640, 1280"`) |
| `densities` | Comma-separated densities for srcset (e.g. `"1x, 2x"`) - use EITHER widths OR densities |
| `formats` | Comma-separated formats in preference order (e.g. `"avif, webp"`); default: `"webp"` |
| `sizes` | CSS sizes attribute (appears on each `<source>` and the fallback `<img>`) |
| `variant` | Select specific CMS media variant by name |
| `loading` | `"lazy"` or `"eager"` |

Use `<piko:img>` for single-format. Use `<piko:picture>` for multi-format with browser negotiation.

### piko:element (dynamic tag)

Static `is` resolves at compile time; dynamic `:is` at runtime:

```piko
<piko:element is="h2">Static</piko:element>
<piko:element :is="state.Tag">Dynamic</piko:element>
```

When `:is` resolves to a piko: tag (`piko:a`, `piko:img`, `piko:svg`), it gets the same special behaviour. Empty/null falls back to `<div>`.

### piko:content (collection content)

Renders the markdown body of a collection item as HTML. Used inside collection layout templates:

```piko
<template p-collection="blog" p-provider="markdown">
  <article>
    <h1>{{ state.Title }}</h1>
    <piko:content />
  </article>
</template>
```

See `references/collections.md` for full collection setup.

## Event bus

Client-side pub/sub for decoupled inter-component communication:

```typescript
// Emit an event
piko.bus.emit('item-added', { count: 1, message: 'Added' });

// Listen for events; on() and once() return an unsubscribe function
const unsubscribe = piko.bus.on('item-added', (data) => {
    console.log(data.count, data.message);
});

// One-time listener (auto-unsubscribes after first invocation)
piko.bus.once('init-complete', (data) => { /* ... */ });

// Unsubscribe a specific handler - call the function returned by on()/once()
unsubscribe();

// Remove all listeners for an event
piko.bus.off('item-added');

// Remove every listener for every event
piko.bus.off();
```

| Method | Purpose |
|--------|---------|
| `piko.bus.emit(name, data)` | Broadcast event to all listeners |
| `piko.bus.on(name, callback)` | Register listener; returns an unsubscribe function |
| `piko.bus.once(name, callback)` | One-time listener (auto-unsubscribes); returns an unsubscribe function |
| `piko.bus.off(event?)` | Remove all listeners for `event` (or all listeners if no arg). For single handler, call the function returned by `on()`/`once()` |

## Server props

Pass data to child partials with the `:server.` prefix:

```piko
<card is="card"
    :server.title="state.PageTitle"
    :server.is_primary="true">
</card>
```

Server props are stripped from rendered HTML. See `references/partials-and-slots.md`.

## LLM mistake checklist

- **Using `{{ }}` inside attributes** - `{{ }}` is ONLY for text content between tags. For dynamic attributes use `:` prefix: `:href="state.URL"`. Never write `href="{{ state.URL }}"` or `href={{ state.URL }}`
- Using `v-if`, `v-for`, `v-bind` instead of `p-if`, `p-for`, `p-bind` / `:`
- Writing JavaScript in expressions (Piko has its own expression language - not JS)
- Using `===` (does not exist - use `==` for strict, `~=` for loose)
- Using a non-bool directly in `p-if` (e.g. `p-if="state.Name"` - use `p-if="~state.Name"` for truthy coercion)
- Using `(item, index)` order in for loops (it's `(index, item)` - Go order, not JS order)
- Forgetting `p-key` in loops (causes reconciliation bugs)
- Using `this.state` in templates (just `state`)
- Using `{{ }}` for raw HTML (use `p-html` directive instead)
- Assuming `:class` replaces the static `class` - both `:class` and `p-class` always merge with the static `class` attribute. Same for `:style` / `p-style` and `style`. For other attributes the dynamic binding does replace the static value
- Writing `p-class:foo="..."` to toggle a single class - this syntax is not supported. Only `p-on:`, `p-event:`, `p-bind:` and `p-timeline:` accept a colon argument suffix. Use the object form instead: `p-class="{ foo: state.IsFoo }"`
- Note: `:prop` renders value as an HTML attribute too; `:server.prop` is server-only (use `:server.prop` when you don't want the prop exposed in rendered HTML)
- Accessing properties on `$event` or `$form` in templates (e.g. `$event.target` is not valid - pass `$event` to your function and access properties there)
- Forgetting `.prevent` on form submit (`p-on:submit.prevent`)
- Using `piko:a` or `piko:img` in CSS/JS selectors - the `piko:` prefix is stripped in rendered HTML. Target `a` or `img` instead
- Using `<a>` instead of `<piko:a>` when you want SPA navigation (regular `<a>` causes full page reload)
- Using `.length` instead of `len()` - Piko uses Go-style built-ins, not JavaScript property access. Write `len(state.Items)`, not `state.Items.length`. Same for conversions: `int(x)` not `parseInt(x)`, `string(x)` not `x.toString()`, `float(x)` not `parseFloat(x)`

## Related

- `references/pk-file-format.md` - full .pk file structure
- `references/pkc-components.md` - client component directives
- `references/partials-and-slots.md` - server props and slots
- `references/collections.md` - collection setup and piko:content usage
