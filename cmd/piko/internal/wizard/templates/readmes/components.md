# `/components`: Your client-side building blocks

This directory is where you build your application's interactive, client-side user interface. It contains all of your **`.pkc` (Piko Component)** files. This directory starts empty. Create your first `.pkc` file here when you need client-side interactivity.

Each `.pkc` file is a self-contained, reusable component that Piko compiles into a standard, dependency-free JavaScript **Web Component**. These components are the "islands of interactivity" on your server-rendered pages. They manage UI state and logic that needs to be fast and responsive, without requiring a round-trip to the server for every user interaction.

---

## What should be a component?

You should create a `.pkc` file for any piece of UI that requires client-side state or interactivity. Common examples include:

*   **Interactive UI Elements:** Modals (`pp-modal.pkc`), dropdowns (`pp-dropdown.pkc`), tabs, and toasts.
*   **Complex Form Controls:** Custom `<select>` elements with search, date pickers, or dynamic lists where users can add and remove items.
*   **Stateful Widgets:** A shopping cart summary that updates as items are added, or a real-time notification indicator.
*   **UI Orchestrators:** Components that manage the state of other components, such as a form that enables/disables its submit button based on the validity of its child inputs.

## The anatomy of a `.pkc` file

A Piko Component file encapsulates everything it needs into a single file. It consists of three parts:

1.  **`<template>` (Required):** The HTML structure of your component. It uses the Piko Templating Language (`p-if`, `p-for`, `:attribute`, etc.) to declaratively render the component's state.
2.  **`<script>` (Required):** The component's logic, written in **TypeScript**. This is where you define its reactive state, props (public properties), and methods (event handlers). The `lang="ts"` and `name` attributes are both required; the `name` attribute defines the component's HTML tag name (e.g., `<script lang="ts" name="pp-button">` creates the `<pp-button>` tag).
3.  **`<style>` (Optional):** The component's CSS. These styles are **automatically scoped** to the component, meaning they won't accidentally affect other parts of your application.

> **Convention**: Use the `pp-` prefix for your component names to avoid conflicts with built-in elements and third-party components.

### Example: a simple counter button

```html
<!-- components/pp-counter.pkc -->
<template>
  <button p-on:click="increment">
    Count is: {{ state.count }}
  </button>
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
  /* These styles only apply inside the <pp-counter> component. */
  :host {
    display: inline-block;
  }
  button {
    background-color: var(--g-colour-primary, #6F47EB);
    color: white;
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
  }
</style>
```

### Using a component

Once defined, you can use your component in any of your server-side `.pk` files (or even inside other `.pkc` components) just like a regular HTML tag.

```html
<!-- pages/index.pk -->
<template>
  <piko:partial is="layout">
    <h1>My Awesome App</h1>
    <p>Here is an interactive counter:</p>

    <!-- Use the component and pass it an initial value -->
    <pp-counter :count="10"></pp-counter>
  </piko:partial>
</template>
```

---

### To learn more

To learn more about component creation, refer to the official Piko documentation:

*   **[Introduction to client components](https://piko.sh/docs/reference/client-components)**
*   **[Component patterns](https://piko.sh/docs/guide/component-patterns)**
*   **[Directives](https://piko.sh/docs/reference/directives)**
