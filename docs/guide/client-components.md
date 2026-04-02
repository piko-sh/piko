---
title: Client components
description: Building reactive client-side components with .pkc files
nav:
  sidebar:
    section: "guide"
    subsection: "concepts"
    order: 50
---

# Client components

Build interactive, reactive UI components with .pkc files using TypeScript. Client components compile to native Web Components with automatic reactivity.

## What are client components?

Client component (.pkc) files are client-side reactive components built on the [Web Components](https://developer.mozilla.org/en-US/docs/Web/API/Web_components) standard. They compile to native custom elements with Shadow DOM encapsulation, meaning they work in any browser without a framework runtime.

Key characteristics:

- **Written in TypeScript.** Type-safe client-side code.
- **Reactive state.** UI updates automatically when state changes.
- **Web Components.** Compile to native custom elements with Shadow DOM.
- **Reusable.** Use across pages, partials, and other components.

## File structure

.pkc files have three sections: `<template>`, `<script>`, and `<style>`.

```piko
<!-- components/pp-counter.pkc -->
<template>
    <div>
        <p>Count: {{ state.count }}</p>
        <button p-on:click="increment">Increment</button>
    </div>
</template>

<script lang="ts" name="pp-counter">
    const state = {
        count: 0 as number,
    };

    function increment() {
        state.count++;
    }
</script>

<style>
    div {
        padding: 1rem;
        border: 1px solid #eee;
    }
</style>
```

### Script attributes

The `<script>` tag supports the following attributes:

| Attribute | Description |
|-----------|-------------|
| `lang="ts"` | Indicates TypeScript (required) |

### Template attributes

The `<template>` tag supports the following attributes:

| Attribute | Description |
|-----------|-------------|
| `name="component-name"` | Defines the custom element tag name (required) |
| `enable="form"` | Enables form association for the component (optional) |

The `name` attribute determines the HTML tag. For example, `name="pp-counter"` creates the `<pp-counter>` custom element.

> **Convention**: Use the `pp-` prefix for your component names to avoid conflicts with built-in elements and third-party components.

### Form-associated components

Add `enable="form"` to the `<template>` tag to create a form-associated custom element:

```piko
<template name="pp-input" enable="form">
    <input
        type="text"
        p-ref="nativeInput"
        p-model="state.value"
        :name="state.name"
        :required="state.required"
    >
</template>

<script lang="ts">
    const state = {
        value: '',
        name: 'custom-field',
        required: true
    };
</script>
```

## Reactive state

State is defined as a `const state` object. Changes to state properties automatically trigger UI updates:

```typescript
const state = {
    count: 0 as number,
    message: 'Hello' as string,
    items: [] as { id: number; text: string }[],
};

function increment() {
    state.count++;  // UI updates automatically
}

function addItem() {
    state.items.push({ id: state.items.length, text: 'New item' });
    // Arrays are reactive too
}
```

### Type annotations

Use TypeScript type assertions to specify types:

```typescript
const state = {
    count: 0 as number,
    name: 'Guest' as string,
    isActive: false as boolean,
    items: [] as string[],
    user: null as { name: string; email: string } | null,
};
```

### Non-reactive variables

Variables declared outside of `state` are not reactive and won't trigger re-renders:

```typescript
const state = {
    displayCount: 0 as number,
};

// Non-reactive counter for internal tracking
let internalCounter = 0;

function increment() {
    internalCounter++;
    // Only update state when you want to trigger a re-render
    state.displayCount = internalCounter;
}
```

## Template directives

Client component templates support the following directives:

### Conditional rendering

```piko
<template>
    <div>
        <p p-if="state.status === 'loading'">Loading...</p>
        <p p-else-if="state.status === 'error'">Error occurred</p>
        <p p-else>Ready</p>
    </div>
</template>
```

> **Key point**: `p-else-if` and `p-else` must immediately follow a `p-if` or another `p-else-if` at the same nesting level.

### Visibility toggle

Use `p-show` to toggle visibility without removing from DOM:

```piko
<template>
    <div>
        <p p-show="state.visible">This toggles visibility</p>
        <button p-on:click="toggle">Toggle</button>
    </div>
</template>

<script lang="ts" name="pp-visibility">
    const state = {
        visible: true as boolean,
    };

    function toggle() {
        state.visible = !state.visible;
    }
</script>
```

### Lists and loops

**Value-only syntax** (most common):

```piko
<template>
    <ul>
        <li p-for="item in state.items" p-key="item.id">
            {{ item.text }}
        </li>
    </ul>
</template>

<script lang="ts" name="pp-list">
    const state = {
        items: [
            { id: 1, text: 'First' },
            { id: 2, text: 'Second' },
        ],
    };
</script>
```

**Index and value syntax**:

```piko
<template>
    <ul>
        <li p-for="(index, item) in state.items" p-key="item.id">
            {{ index }}: {{ item.text }}
        </li>
    </ul>
</template>
```

> **Note**: Always use `p-key` with a unique identifier to enable efficient DOM reconciliation.

### Text and HTML binding

```piko
<template>
    <div>
        <!-- Text binding (auto-escaped) -->
        <p p-text="state.message"></p>

        <!-- Or use interpolation -->
        <p>{{ state.message }}</p>

        <!-- Raw HTML (use with caution) -->
        <div p-html="state.htmlContent"></div>
    </div>
</template>
```

> **Security warning**: Never use `p-html` with user input or untrusted content. This bypasses HTML escaping and can lead to XSS attacks.

### Attribute binding

Use `:` prefix or `p-bind:` for dynamic attributes:

```piko
<template>
    <div>
        <!-- Shorthand syntax -->
        <a :href="state.url">Link</a>
        <img :src="state.imageSrc" :alt="state.imageAlt">

        <!-- Boolean attributes -->
        <button :disabled="state.isLoading">Submit</button>
        <input :readonly="state.isReadOnly">
    </div>
</template>
```

### Class binding

Use `p-class` with object or array syntax for conditional classes:

**Object syntax**:

```piko
<template>
    <div class="base" p-class="{ active: state.isActive, 'text-danger': state.hasError }">
        Styled element
    </div>
</template>

<script lang="ts" name="pp-styled">
    const state = {
        isActive: true,
        hasError: false,
    };
</script>
```

**Array syntax** (supports mixed values):

```piko
<template>
    <div p-class="['base-class', state.dynamicClass, { 'conditional': state.isConditional }]">
        Mixed class binding
    </div>
</template>

<script lang="ts" name="pp-mixed-class">
    const state = {
        dynamicClass: 'theme-dark',
        isConditional: true
    };
</script>
```

### Style binding

Use `p-style` for dynamic inline styles:

```piko
<template>
    <p style="font-weight: bold;" p-style="{ color: state.textColor, fontSize: state.size + 'px' }">
        Styled text
    </p>
</template>

<script lang="ts" name="pp-styled-text">
    const state = {
        textColor: 'blue',
        size: 16,
    };
</script>
```

### Two-way binding

Use `p-model` for form inputs:

```piko
<template>
    <div>
        <input type="text" p-model="state.name">
        <p>Hello, {{ state.name }}</p>
    </div>
</template>

<script lang="ts" name="pp-input">
    const state = {
        name: 'World',
    };
</script>
```

`p-model` works with text inputs, checkboxes, and other input types:

```piko
<template>
    <div>
        <label>
            <input type="checkbox" p-model="state.isChecked">
            Is Checked
        </label>
        <p>Status: {{ state.isChecked ? 'Checked' : 'Not Checked' }}</p>
    </div>
</template>

<script lang="ts" name="pp-checkbox">
    const state = {
        isChecked: false
    };
</script>
```

## Event handling

Use `p-on:event` to attach event handlers:

```piko
<template>
    <div>
        <button p-on:click="handleClick">Click me</button>
        <input p-on:input="handleInput" p-on:keyup="handleKeyup">
    </div>
</template>

<script lang="ts" name="pp-events">
    const state = {
        count: 0 as number,
    };

    function handleClick() {
        state.count++;
    }

    function handleInput(event) {
        console.log('Input value:', event.target.value);
    }

    function handleKeyup(event) {
        if (event.key === 'Enter') {
            console.log('Enter pressed');
        }
    }
</script>
```

### Passing arguments with $event

Access the native event object using `$event`:

```piko
<template>
    <div>
        <button p-on:click="setMessage('Hello', state.name, $event)">Greet</button>
        <p>{{ state.message }}</p>
    </div>
</template>

<script lang="ts" name="pp-greet">
    const state = {
        message: 'Initial',
        name: 'World'
    };

    function setMessage(greeting, name, event) {
        console.log('Event target:', event.target.tagName);
        state.message = `${greeting}, ${name}!`;
    }
</script>
```

## Props

State properties are automatically exposed as component props. Values in `state` serve as defaults:

```piko
<script lang="ts" name="pp-button">
    const state = {
        variant: 'primary' as string,   // Default: 'primary'
        disabled: false as boolean,      // Default: false
        size: 'md' as string,            // Default: 'md'
    };
</script>
```

Usage with props:

```html
<pp-button variant="secondary" disabled="true" size="lg">
    Click Me
</pp-button>
```

## Custom events

Emit events to communicate with parent components:

```piko
<script lang="ts" name="pp-modal">
    function save() {
        this.dispatchEvent(new CustomEvent('save-clicked', {
            bubbles: true,
            composed: true,
            detail: { id: 123, data: state.formData }
        }));
    }

    function close() {
        this.dispatchEvent(new CustomEvent('close', {
            bubbles: true,
            composed: true
        }));
    }
</script>
```

Parent listens with `p-on`:

```html
<pp-modal p-on:save-clicked="handleSave" p-on:close="handleClose">
</pp-modal>
```

## Lifecycle hooks

Client components support lifecycle hooks via the `pkc` context object. Register callbacks with `pkc.onXxx(() => { ... })`:

```piko
<script lang="ts" name="pp-lifecycle">
    const state = {
        scrollY: 0 as number,
    };

    const handleScroll = () => {
        state.scrollY = Math.round(window.scrollY);
    };

    pkc.onConnected(() => {
        window.addEventListener('scroll', handleScroll);
        handleScroll();

        // Co-locate teardown with setup
        pkc.onCleanup(() => {
            window.removeEventListener('scroll', handleScroll);
        });
    });

    pkc.onUpdated((changedProps) => {
        console.log('Changed:', Array.from(changedProps));
    });
</script>
```

Multiple callbacks can be registered for the same hook; they fire in registration order.

### Lifecycle hook reference

| Hook | Parameter | Description |
|------|-----------|-------------|
| `onConnected` | None | Called when the component connects to the DOM |
| `onDisconnected` | None | Called when the component disconnects from the DOM |
| `onBeforeRender` | None | Called before each render cycle |
| `onAfterRender` | None | Called after each render cycle |
| `onUpdated` | `changedProps: Set<string>` | Called after re-render with the set of changed property names |
| `onCleanup` | None | Registers a teardown function that runs on disconnection, after `onDisconnected` |

> **Best practice**: Use `onCleanup` inside `onConnected` to co-locate setup and teardown. This is less error-prone than splitting logic across `onConnected` and `onDisconnected`.

## Element references

Use `p-ref` to get references to DOM elements:

```piko
<template>
    <div>
        <input type="text" p-ref="myInput">
        <button p-on:click="focusInput">Focus Input</button>
    </div>
</template>

<script lang="ts" name="pp-refs">
    function focusInput() {
        this.refs.myInput.focus();
    }
</script>
```

## Shadow DOM access

Access the component's shadow root via `this.shadowRoot`:

```piko
<script lang="ts" name="pp-shadow">
    function queryElement() {
        const element = this.shadowRoot.querySelector('.my-class');
        console.log('Found:', element.textContent);
    }
</script>
```

## Slots

Use standard Web Component slots for content projection:

```piko
<template>
    <div class="card">
        <header>
            <slot name="header">Default Header</slot>
        </header>
        <main>
            <slot>Default Content</slot>
        </main>
    </div>
</template>

<script lang="ts" name="pp-card"></script>
```

Usage:

```html
<pp-card>
    <span slot="header">Custom Header</span>
    <p>This goes in the default slot</p>
</pp-card>
```

## Async methods

Client components support async/await in methods:

```piko
<script lang="ts" name="pp-async">
    const state = {
        status: 'Idle' as string,
        data: null as any,
    };

    async function fetchData() {
        state.status = 'Loading...';

        try {
            const response = await fetch('/api/data');
            state.data = await response.json();
            state.status = 'Loaded';
        } catch (error) {
            state.status = 'Error';
        }
    }
</script>
```

## Imports

Import external modules in your script:

```piko
<script lang="ts" name="pp-imports">
    import { formatDate } from './utils.js';

    const state = {
        date: new Date(),
    };

    function getFormattedDate() {
        return formatDate(state.date);
    }
</script>
```

You can also import from external URLs:

```piko
<script lang="ts" name="pp-external">
    import confetti from 'https://esm.sh/canvas-confetti';

    pkc.onConnected(() => {
        confetti({
            particleCount: 100,
            spread: 70,
            origin: { y: 0.6 }
        });
    });
</script>
```

## Piko helpers

Client components have access to Piko helpers via the `piko` namespace:

```piko
<script lang="ts" name="pp-helpers">
    function updatePartial() {
        piko.partials.render({
            src: '/my/partial',
            querySelector: '#target-div',
            patchMethod: 'morph'
        });
    }
</script>
```

## Using components

Include client components in PK pages or partials:

```piko
<template>
    <div>
        <h1>My Page</h1>
        <pp-counter></pp-counter>
        <pp-button variant="primary" p-on:button-clicked="handleClick">
            Click Me
        </pp-button>
    </div>
</template>
```

## Complete example

A full-featured button component:

```piko
<!-- components/pp-button.pkc -->
<template>
    <button
        class="btn"
        p-class="{ loading: state.loading }"
        :disabled="state.disabled || state.loading"
        p-on:click="handleClick"
    >
        <span class="spinner" p-show="state.loading"></span>
        <span class="content" p-show="!state.loading">
            <slot></slot>
        </span>
    </button>
</template>

<script lang="ts" name="pp-button">
    const state = {
        variant: 'primary' as string,
        size: 'md' as string,
        disabled: false as boolean,
        loading: false as boolean,
    };

    function handleClick(event) {
        if (state.disabled || state.loading) {
            event.preventDefault();
            return;
        }

        this.dispatchEvent(new CustomEvent('button-clicked', {
            bubbles: true,
            composed: true
        }));
    }
</script>

<style>
    .btn {
        padding: 0.5rem 1rem;
        border-radius: 0.375rem;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }

    .btn:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }

    .btn.loading {
        position: relative;
    }

    .spinner {
        display: inline-block;
        width: 1rem;
        height: 1rem;
        border: 2px solid currentColor;
        border-right-color: transparent;
        border-radius: 50%;
        animation: spin 0.75s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }
</style>
```

## Directive reference

| Directive | Purpose | Example |
|-----------|---------|---------|
| `p-if` | Conditional render | `<div p-if="state.show">...</div>` |
| `p-else-if` | Chained condition | `<div p-else-if="state.other">...</div>` |
| `p-else` | Fallback | `<div p-else>...</div>` |
| `p-show` | Toggle visibility | `<div p-show="state.visible">...</div>` |
| `p-for` | Loop | `<li p-for="item in state.items">...</li>` |
| `p-key` | Unique identifier | `<li p-for="item in items" p-key="item.id">` |
| `p-text` | Text content | `<p p-text="state.message"></p>` |
| `p-html` | Raw HTML | `<div p-html="state.html"></div>` |
| `:attr` | Attribute binding | `<a :href="state.link">Link</a>` |
| `p-bind:attr` | Explicit binding | `<a p-bind:href="state.link">Link</a>` |
| `p-class` | Conditional classes | `<div p-class="{ active: state.isActive }">` |
| `p-style` | Dynamic styles | `<div p-style="{ color: state.colour }">` |
| `p-model` | Two-way binding | `<input p-model="state.value" />` |
| `p-on:event` | Event handler | `<button p-on:click="handleClick">` |
| `p-ref` | Element reference | `<input p-ref="myInput" />` |
