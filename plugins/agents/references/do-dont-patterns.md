# Do and Don't Patterns

Use this guide as a fast checklist for correct Piko output.

## PK files

Do:
- Use `<script type="application/x-go">` with `package main`
- Return `(Response, piko.Metadata, error)` from Render
- Use `piko.NoResponse` when no state is needed
- Use `piko.NoProps` for pages
- Keep Render focused - delegate to `pkg/domain`

Don't:
- Forget `type="application/x-go"` on the script tag
- Put business logic in Render
- Set `piko.Metadata{Status: 404}` - response writer ignores it; use `piko.NotFound(...)` instead
- Use `v-if` / `v-for` - Piko uses `p-if` / `p-for`
- Write JavaScript in template expressions - Piko has its own expression language
- Use `(item, index)` order in for loops - it's `(index, item)` (Go order)

## PKC components

Do:
- Set `name="component-name"` on `<template>` (must contain a hyphen); `name` on `<script>` is ignored
- Use `pp-` prefix for component names
- Prefer snake_case for state variables (round-trips to attributes unchanged)
- Use `as type` annotations on all state properties
- Co-locate cleanup with setup (`pkc.onCleanup()` inside `pkc.onConnected()`)
- Use `<slot>` for content projection (Web Component slots)

Don't:
- Forget the `name` attribute on the template
- Use camelCase state vars (lowers to kebab-case attributes; snake_case avoids the rename)
- Nest PKC components inside each other's templates (use slots from PK files)
- Use `<piko:slot>` in PKC files (that's for server partials)
- Use React/Vue patterns (`useState`, `@click`, `v-bind`)
- Forget `bubbles: true, composed: true` on custom events

## Templates and directives

Do:
- Use `{{ state.Field }}` for interpolation **inside element content only** (auto-escaped)
- Use `:` prefix for dynamic attributes: `:href="state.URL"`
- Use plain text for static attributes: `class="container"`
- Directives (`p-if`, `p-for`, `p-on`, etc.) are expressions by default - no `:` prefix
- `p-ref` and `p-slot` are exceptions - they accept plain text
- Use `==` for strict equality, `~=` for loose (not `===`)
- Use `~state.Var` to coerce non-booleans in conditions (`p-if="~state.Name"`)
- Use `p-key` with a unique identifier in every `p-for`
- Use `:server.prop` to pass props to partials
- Use `.prevent` on form submit
- Use `$event` to pass the native event to handlers

Don't:
- Use `{{ }}` inside attributes - it is ONLY for text content between tags. Use `:` prefix for dynamic attributes
- Use `this.state` - just `state`
- Access properties on `$event` / `$form` in **PK** template expressions - rejected by server-side analyser. Pass to a function and access there. PKC client templates **do** accept `$event.X`
- Use `{{ }}` for raw HTML - use `p-html`
- Treat `:prop="..."` as wrong - both `:prop` (bare) and `:server.prop` are accepted; `:server.prop` errors at compile time on wrong prop name
- Use `:class` / `:style` to combine static and dynamic - `:class` **replaces**. Use `p-class` / `p-style` which **merge** (e.g. `class="x" p-class="'y'"` → `class="x y"`)
- Use `===` (doesn't exist - `==` strict, `~=` loose)
- Use a non-bool directly in `p-if` without `~`

## Partials and slots

Do:
- Import partials in the Go script: `import card "myapp/partials/card.pk"`
- Invoke with `<piko:partial is="card">` - only `piko:partial` triggers expansion; `is` on other tags is a normal attribute
- Use `<piko:slot />` for default slots, `<piko:slot name="x">` for named slots
- Use `coerce:"true"` tag for non-string prop types
- Provide meaningful fallback content in slots

Don't:
- Make `is` attribute dynamic (must be static, resolved at compile time)
- Use `<slot>` in server partials (that's PKC)
- Define multiple default slots in one partial
- Forget `coerce:"true"` when passing booleans or numbers as props

## Server actions

Do:
- Embed `piko.ActionMetadata` in every action struct
- Run `go run ./cmd/generator/main.go all` after creating new actions
- Invoke as `action.<package>.<StructNameMinusActionSuffix>(...)` - e.g. `SubmitAction` in `package contact` → `action.contact.Submit($form).call()`
- Match input `name` attributes to `Call` parameter names
- Use `a.Response().AddHelper()` for user feedback
- Use `piko.ValidationField()` for field-level validation errors
- Use pointer types for optional parameters

Don't:
- Forget to run the generator (404 at runtime)
- Use `a.Response().AddFieldError()` (does not exist - return `piko.ValidationField()` as error)
- Use `a.Response().AddCookie()` (the method is `SetCookie`)
- Return errors without user-facing feedback (use `showToast`)
- Use snake_case action methods in templates - method segment is **PascalCase**; only the package segment is snake_case
- Forget `.prevent` on form submit

## Routing

Do:
- Use `{param}` for dynamic segments in filenames
- Use `{name}*` (or `{name:.+}`) for catch-all routes - e.g. `pages/docs/{slug}*.pk`
- Use `r.PathParam("name")` for route parameters
- Return a typed error (`piko.NotFound("post", id)`) for missing resources
- Keep route hierarchies shallow (max 3-4 levels)

Don't:
- Use `:param` (Express) or `{...param}` (Next.js) - Piko uses `{param}` and `{name}*`
- Put pages outside `pages/` directory
- Confuse `PathParam` with `QueryParam`
- Set `piko.Metadata{Status: 404}` - return a typed error instead

## Error pages

Do:
- Use `!NNN.pk` for exact codes, `!NNN-NNN.pk` for ranges, `!error.pk` for catch-all
- Use `piko.GetErrorContext(r)` for `StatusCode`, `Message`, `OriginalPath`
- Return typed errors (`piko.NotFound(...)`, `piko.Forbidden(...)`) to trigger error pages
- Place error pages in subdirectories for section-specific handling

Don't:
- Use invalid `!` filenames (e.g. `!pancakes.pk`) - build will fail
- Forget that exact > range > catch-all priority applies
- Manually handle missing collection items - `p-collection` automatically triggers 404

## Styling

Do:
- Use `var(--g-name)` for theme variables passed via `piko.WithWebsiteConfig`
- Use `:host([attr="value"])` for string attrs and `:host([attr])` for boolean attrs in PKC
- Use `hasAttribute()` / `toggleAttribute()` for boolean state (not `getAttribute() === "true"`)
- Use `:deep()` sparingly to reach into child partials

Don't:
- Assume page styles are global - all component types are scoped by default; use `<style global>` for unscoped
- Style PKC internals from outside with regular selectors (Shadow DOM)
- Edit files in `dist/` (regenerated)

## Built-in elements

Do:
- Use `<piko:a>` for internal navigation (SPA-style, no full reload)
- Use `<piko:img>` for images that need srcset, format variants, or CMS media
- Target rendered elements as plain `a` and `img` in CSS/JS - `piko:` prefix is stripped
- Use `widths` OR `densities` on `piko:img` (not both)

Don't:
- Use `piko:a` / `piko:img` in CSS selectors or JS queries - prefix stripped in rendered HTML
- Use a regular `<a>` when you want SPA navigation (full page reload)
- Use both `widths` and `densities` on the same `piko:img`

## CLI and workflow

Do:
- Run `piko fmt -r .` before committing
- Run `go run ./cmd/generator/main.go all` after adding pages/partials/actions
- Use `air` for development with live reloading

Don't:
- Skip the generator step after structural changes
- Edit generated files in `dist/`
- Forget `go mod tidy` after dependency changes

## Related

- `references/pk-file-format.md` - .pk file structure
- `references/pkc-components.md` - client component patterns
- `references/template-syntax.md` - directives and expressions
- `references/server-actions.md` - action patterns
