# PKC Client Components

Use this guide when creating or modifying `.pkc` files - reactive client-side Web Components.

## File structure

A `.pkc` file has three sections: template, script, and style.

```piko
<!-- components/pp-counter.pkc -->
<template name="pp-counter">
  <div>
    <p>Count: {{ state.count }}</p>
    <button p-on:click="increment">+1</button>
  </div>
</template>

<script lang="ts">
const state = {
    count: 0 as number,
};

function increment() {
    state.count++;
}
</script>

<style>
div { padding: 1rem; }
</style>
```

## The `pkc` context object

`const pkc = this` is injected at the top of every component instance function. `pkc` is a stable reference to the Web Component's `HTMLElement`:

- **Lifecycle**: `pkc.onConnected()`, `pkc.onCleanup()`, `pkc.onUpdated()`
- **DOM references**: `pkc.refs.my_input`
- **Shadow DOM**: `pkc.shadowRoot.querySelector()`
- **Events**: `pkc.dispatchEvent(new CustomEvent(...))`
- **Attributes**: `pkc.hasAttribute()`, `pkc.toggleAttribute()`, `pkc.getAttribute()`

Prefer `pkc` over `this` - avoids scope confusion in callbacks.

## Template tag attributes

| Attribute | Required | Description |
|-----------|----------|-------------|
| `name="component-name"` | No | Tag name; falls back to source filename when omitted |
| `enable="form"` | No | Form association; also accepts `"animation"` or `"form animation"` |

Resolved name must contain a hyphen (Web Components spec). `components/pp-counter.pkc` resolves to `pp-counter`.

## Script tag attributes

| Attribute | Required | Description |
|-----------|----------|-------------|
| `lang="ts"` | No | Also accepts `"js"`, `"javascript"`, `"typescript"`; `"ts"` is the convention |

**Convention**: Use the `pp-` prefix for component names.

## Reactive state

State is declared as a `const state` object. Changes automatically trigger UI updates.

```typescript
const state = {
    count: 0 as number,
    message: 'Hello' as string,
    is_active: false as boolean,
    items: [] as { id: number; text: string }[],
    user: null as { name: string } | null,
};
```

### State variable naming: use snake_case

State variables use snake_case (not camelCase) because:

1. **Two-way attribute sync**: State auto-syncs to HTML attributes. HTML is case-insensitive, so `isActive` → `isactive`. Snake_case (`is_active`) is valid as both identifier and attribute name.
2. **CSS styling via attributes**:

```css
:host([variant="primary"]) {
    background: blue;
}
/* Boolean: check presence, not value */
:host([is_disabled]) {
    opacity: 0.5;
}
```

3. **`setAttribute` updates state**: `element.setAttribute('variant', 'secondary')` updates internal state. Only works cleanly with snake_case.
4. **Boolean attributes follow HTML conventions**: `true` renders as bare attribute, `false`/`nil` removes it. Use `pkc.hasAttribute()`, `pkc.toggleAttribute()` - don't check for `"true"`/`"false"` strings.

### Non-reactive variables

Variables outside `state` are not reactive:

```typescript
let internal_counter = 0;  // Not tracked, no re-renders

function increment() {
    internal_counter++;
    state.display_count = internal_counter;  // Triggers re-render
}
```

## Template directives

PKC templates support the same directives as PK templates:

| Directive | Example |
|-----------|---------|
| `p-if` / `p-else-if` / `p-else` | `<p p-if="state.show">Visible</p>` |
| `p-show` | `<p p-show="state.visible">Toggles CSS</p>` |
| `p-for` + `p-key` | `<li p-for="item in state.items" p-key="item.id">` (index form: `(i, item)` - index first) |
| `p-text` | `<p p-text="state.message"></p>` |
| `p-html` | `<div p-html="state.html_content"></div>` |
| `:attr` / `p-bind:attr` | `<a :href="state.url">Link</a>` |
| `p-class` | `<div p-class="{ active: state.is_active }">` |
| `p-style` | `<div p-style="{ color: state.text_colour }">` |
| `p-model` | `<input p-model="state.value" />` |
| `p-on:event` | `<button p-on:click="handle_click">` |
| `p-ref` | `<input p-ref="my_input" />` |

## Event handling

`p-on` binds event listeners to functions in the component's script block.

**Three calling conventions**:

1. **No parentheses** - `p-on:click="myFn"` - event passed as first arg implicitly
2. **Empty parentheses** - `p-on:click="myFn()"` - no args passed (not even event)
3. **With arguments** - `p-on:click="myFn('x', $event)"` - you control what is passed

```piko
<template>
  <!-- Event passed implicitly as first arg -->
  <button p-on:click="handle_click">Click</button>

  <!-- No args passed -->
  <button p-on:click="increment()">+1</button>

  <!-- Event passed explicitly -->
  <input p-on:input="handle_input($event)" />
</template>

<script lang="ts">
function handle_click(event) {
    // event is the implicit first arg
    console.log('Clicked at:', event.clientX);
}

function increment() {
    // no args - called with ()
    state.count++;
}

function handle_input(event) {
    console.log('Value:', event.target.value);
}
</script>
```

**Explicit positioning**: Use `$event` to place the event in any argument position:

```piko
<button p-on:click="greet('Hello', $event)">Greet</button>
```

**`$form` special value**: Pass form data as a `FormDataHandle`:

```piko
<form p-on:submit.prevent="handle_submit($form)">
```

**IMPORTANT**: `$event` and `$form` are opaque in expressions - cannot access properties (e.g. `$event.target` invalid in template). Pass to a function and access there.

**Event modifiers**: `.prevent`, `.stop`, `.once`, `.self`:

```piko
<form p-on:submit.prevent="handle_submit">
<a p-on:click.prevent.stop="navigate()">Link</a>
```

## Props from state defaults

State properties double as props. Default values come from the state declaration:

```piko
<template name="pp-button">
  <!-- ... -->
</template>

<script lang="ts">
const state = {
    variant: 'primary' as string,
    disabled: false as boolean,
    size: 'md' as string,
};
</script>
```

Usage:

```html
<pp-button variant="secondary" disabled="true" size="lg">
  Click Me
</pp-button>
```

## Custom events

Emit events to communicate with parents:

```typescript
function save() {
    pkc.dispatchEvent(new CustomEvent('save-clicked', {
        bubbles: true,
        composed: true,
        detail: { id: 123 }
    }));
}
```

Parent listens:

```html
<pp-modal p-on:save-clicked="handleSave"></pp-modal>
```

## Lifecycle hooks

Register via the `pkc` context object:

```typescript
pkc.onConnected(() => {
    window.addEventListener('scroll', handle_scroll);

    // Co-locate teardown with setup
    pkc.onCleanup(() => {
        window.removeEventListener('scroll', handle_scroll);
    });
});

pkc.onUpdated((changed_props) => {
    console.log('Changed:', Array.from(changed_props));
});
```

| Hook | Parameter | Description |
|------|-----------|-------------|
| `onConnected` | None | Component connects to DOM |
| `onDisconnected` | None | Component disconnects from DOM |
| `onBeforeRender` | None | Before each render cycle |
| `onAfterRender` | None | After each render cycle |
| `onUpdated` | `changedProps: Set<string>` | After re-render with changed property names |
| `onCleanup` | None | Teardown on disconnection (runs after `onDisconnected`) |

## Element references

```piko
<template>
  <input type="text" p-ref="my_input" />
  <button p-on:click="focus_input">Focus</button>
</template>

<script lang="ts">
function focus_input() {
    pkc.refs.my_input.focus();
}
</script>
```

## Shadow DOM

PKC components use Shadow DOM for encapsulation. Access via `pkc.shadowRoot`:

```typescript
function query_element() {
    const el = pkc.shadowRoot.querySelector('.my-class');
}
```

## Slots (Web Component standard)

PKC components use standard Web Component `<slot>` elements - **not** `<piko:slot>`:

```piko
<template name="pp-card">
  <div class="card">
    <header><slot name="header">Default Header</slot></header>
    <main><slot>Default Content</slot></main>
  </div>
</template>

<script lang="ts"></script>
```

Usage:

```html
<pp-card>
  <span slot="header">Custom Header</span>
  <p>Main content</p>
</pp-card>
```

## Cannot nest PKC components directly

PKC components should not be nested inside each other's templates. Use the slot system instead - compose from PK pages/partials:

```html
<!-- WRONG: nesting PKCs in PKC template -->
<template>
  <pp-inner></pp-inner>
</template>

<!-- RIGHT: use slots, compose from PK pages/partials -->
<pp-outer>
  <pp-inner slot="content"></pp-inner>
</pp-outer>
```

## Imports

```typescript
import { formatDate } from './utils.js';
import confetti from 'https://esm.sh/canvas-confetti';
```

## Piko helpers

```typescript
function reload() {
    piko.partials.render({
        src: '/my/partial',
        querySelector: '#target',
        patchMethod: 'morph'
    });
}
```

## Using components in PK files

Include directly in PK page or partial templates:

```piko
<template>
  <h1>My Page</h1>
  <pp-counter></pp-counter>
  <pp-button variant="primary" p-on:button-clicked="handleClick">
    Click Me
  </pp-button>
</template>
```

## Complete example

```piko
<!-- components/pp-button.pkc -->
<template name="pp-button">
  <button
    class="btn"
    p-class="{ loading: state.loading }"
    :disabled="state.disabled || state.loading"
    p-on:click="handle_click"
  >
    <span class="spinner" p-show="state.loading"></span>
    <span class="content" p-show="!state.loading">
      <slot></slot>
    </span>
  </button>
</template>

<script lang="ts">
const state = {
    variant: 'primary' as string,
    size: 'md' as string,
    disabled: false as boolean,
    loading: false as boolean,
};

function handle_click(event) {
    if (state.disabled || state.loading) {
        event.preventDefault();
        return;
    }
    pkc.dispatchEvent(new CustomEvent('button-clicked', {
        bubbles: true,
        composed: true
    }));
}
</script>

<style>
:host([variant="primary"]) .btn {
    background: var(--g-colour-primary, #6F47EB);
    color: white;
}
:host([variant="secondary"]) .btn {
    background: transparent;
    border: 1px solid currentColor;
}
.btn { padding: 0.5rem 1rem; border-radius: 0.375rem; cursor: pointer; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
```

## LLM mistake checklist

- **Using `{{ }}` inside attributes** - `{{ }}` is ONLY for text content between tags. For dynamic attributes use `:` prefix: `:href="state.url"`. Never write `href="{{ state.url }}"` or `href={{ state.url }}`
- Putting `name` or `enable` on `<script>` instead of `<template>` (they live on `<template>`; `<script>` only takes `lang`)
- Using camelCase for state variables (breaks HTML attribute sync - use snake_case)
- Trying to nest PKC components inside each other's templates (use slots instead)
- Using `<piko:slot>` instead of `<slot>` (PKC uses Web Component slots, not server slots)
- Using Vue/React patterns (`v-if`, `useState`, `@click`) instead of Piko directives
- Writing JavaScript in template expressions (Piko templates use a custom expression language, not JS)
- Using `(item, index)` order in for loops (it's `(index, item)` - Go order)
- Forgetting `as type` annotations on state properties
- Using `this.state` instead of just `state`
- Forgetting `bubbles: true, composed: true` on custom events (needed to cross Shadow DOM)
- Checking `getAttribute('disabled') === "true"` for booleans (use `hasAttribute('disabled')` - true renders as bare attribute, not `"true"`)
- Using `:host([disabled="true"])` in CSS (use `:host([disabled])` - check presence, not value)
- Not co-locating cleanup with setup (use `pkc.onCleanup()` inside `pkc.onConnected()`)

## Related

- `references/template-syntax.md` - directives shared with PK files
- `references/styling.md` - Shadow DOM styling with `:host`
- `references/examples.md` - complete component examples
