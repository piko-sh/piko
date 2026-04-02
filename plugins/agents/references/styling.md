# Styling

Use this guide when working with CSS in Piko - scoping, global styles, theme variables, or PKC Shadow DOM styling.

## Scoping rules

| Component Type | Default | Override |
|----------------|---------|----------|
| Pages | Scoped to component | `<style global>` |
| Emails | Scoped to component | `<style global>` |
| Partials | Scoped to component | `<style global>` |
| PKC components | Shadow DOM encapsulated | N/A |

All `.pk` component types are scoped by default - only `<style global>` produces unscoped CSS.

## Partial scoping (default)

Partial styles are automatically scoped. Each partial gets a unique identifier, and CSS selectors are rewritten to match only that partial's elements:

```piko
<!-- partials/card.pk -->
<style>
.card { border: 1px solid #ccc; padding: 1rem; }
</style>
```

Compiled:

```css
.card[partial~=partials_card_a1b2c3d4] { border: 1px solid #ccc; padding: 1rem; }
```

Nested partials are isolated - parent CSS never leaks into child partials.

## Deep selectors

Style elements inside child partials with `:deep()`:

```piko
<style>
/* Reaches into nested partials */
:deep(.card) { border: 2px solid red; }

/* Chain with own selectors */
.wrapper :deep(.card .title) { color: blue; }
</style>
```

Use sparingly - it breaks encapsulation.

## Global styles

### Unscoped block

```piko
<style global>
.global-class { margin: 0; }
</style>
```

### :global() pseudo-class

Escape scoping for specific selectors within a scoped block:

```piko
<style>
/* Scoped */
.card { background: blue; }

/* Global */
:global(body) { font-family: sans-serif; }

/* Mix scoped parent with global child */
.wrapper :global(.external-library-class) { color: red; }
</style>
```

### Special elements

`html`, `body`, and `:root` are **never scoped**, even in `<style>` blocks.

## Theme variables

Define in `config.json`:

```json
{
  "theme": {
    "colour-primary": "#6F47EB",
    "colour-grey-200": "#CAD1D8",
    "spacing-md": "1rem",
    "font-family": "\"Inter\", sans-serif"
  }
}
```

Use with the `--g-` prefix:

```css
.button {
    background: var(--g-colour-primary);
    font-family: var(--g-font-family);
    padding: var(--g-spacing-md);
}

/* With fallback */
.text { color: var(--g-colour-grey-700, #374151); }
```

Theme variables work across scope boundaries (they're CSS custom properties on `:root`).

## CSS imports

Imported CSS inherits the scoping of its containing block:

```piko
<!-- Scoped imports -->
<style>
@import url('/assets/styles/card.css');
</style>

<!-- Unscoped imports -->
<style global>
@import url('/assets/styles/reset.css');
</style>
```

## Slotted content

Content passed into a partial's slot retains the **parent's** scope:

```piko
<!-- parent.pk -->
<card is="card">
  <span class="highlight">Text</span>  <!-- Gets parent's scope -->
</card>

<style>
.highlight { color: red; }  /* Styles the slotted content */
</style>
```

## PKC Shadow DOM styling

PKC components use Shadow DOM. Styles are automatically encapsulated:

```piko
<!-- components/pp-button.pkc -->
<style>
/* Only applies inside this component's shadow DOM */
.btn { padding: 0.5rem 1rem; }
</style>
```

### :host selector

Style the component element itself:

```css
:host {
    display: inline-block;
}
```

### :host with attribute selectors

Because PKC state syncs two-way with HTML attributes, you can style based on state:

```css
:host([variant="primary"]) .btn {
    background: var(--g-colour-primary);
    color: white;
}

:host([variant="secondary"]) .btn {
    background: transparent;
    border: 1px solid currentColor;
}

/* Boolean attrs: check presence, not value - true renders as bare attribute */
:host([disabled]) {
    opacity: 0.5;
    pointer-events: none;
}
```

This is a key pattern: set state → attribute updates → CSS matches automatically.

## LLM mistake checklist

- Assuming page styles are global (all component types including pages are scoped by default - use `<style global>` for unscoped styles)
- Using `:deep()` excessively instead of passing props or using slots
- Forgetting `--g-` prefix when using theme variables
- Using `<style scoped>` (Piko partials are scoped by default - `scoped` attribute is valid but redundant)
- Using regular CSS class selectors to style PKC internals from outside (Shadow DOM blocks them)
- Forgetting `:host()` when styling the PKC element itself

## Related

- `references/pk-file-format.md` - style block syntax
- `references/pkc-components.md` - Shadow DOM and attribute sync
- `references/project-structure.md` - config.json theme setup
