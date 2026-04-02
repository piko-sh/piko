# Do and Don't Patterns

Use this guide as a fast checklist for correct Piko output.

## PK files

Do:
- Use `<script type="application/x-go">` with `package main`
- Return `(Response, piko.Metadata, error)` from Render
- Use `piko.NoResponse` when no state is needed
- Use `piko.NoProps` for pages (pages don't receive props)
- Keep Render focused - delegate to `pkg/domain` for business logic

Don't:
- Forget the `type="application/x-go"` attribute on the script tag
- Put business logic in Render - use a domain layer
- Return raw errors to users - set `piko.Metadata{Status: 404}` for missing resources
- Use `v-if` or `v-for` - Piko uses `p-if` and `p-for`
- Write JavaScript in template expressions - Piko has its own expression language, not JS
- Use `(item, index)` order in for loops - it's `(index, item)` (Go order, not JS order)

## PKC components

Do:
- Set `name="component-name"` on the script tag (required, must contain a hyphen)
- Use `pp-` prefix for component names
- Use snake_case for state variables (valid JavaScript + valid HTML attributes)
- Use `as type` annotations on all state properties
- Co-locate cleanup with setup (`pkc.onCleanup()` inside `pkc.onConnected()`)
- Use `<slot>` for content projection (standard Web Component slots)

Don't:
- Forget the `name` attribute (component won't register)
- Use camelCase state variables (HTML attributes are case-insensitive)
- Nest PKC components inside each other's templates (use slots from PK files)
- Use `<piko:slot>` in PKC files (that's for server partials)
- Use React/Vue patterns (`useState`, `@click`, `v-bind`)
- Forget `bubbles: true, composed: true` on custom events

## Templates and directives

Do:
- Use `{{ state.Field }}` for interpolation **inside element content only** (auto-escaped)
- Use `:` prefix for dynamic attributes: `:href="state.URL"`, `:class="state.ClassName"`
- Use plain text for static attributes: `class="container"`, `id="header"`
- Remember that directives (`p-if`, `p-for`, `p-on`, etc.) are expressions by default - no `:` prefix needed
- Remember that `p-ref` and `p-slot` are the exceptions - they accept plain text, not expressions
- Use `==` for strict equality and `~=` for loose equality (not `===`)
- Use `~state.Var` to coerce non-booleans to truthy in conditions (`p-if="~state.Name"`)
- Use `p-key` with a unique identifier in every `p-for` loop
- Use `:server.prop` to pass props to partials
- Use `.prevent` on form submit: `p-on:submit.prevent="action.name()"`
- Use `$event` to pass the native event to handlers

Don't:
- Use `{{ }}` inside attributes - `{{ }}` is ONLY for text content between tags (e.g. `<p>{{ state.X }}</p>`). For dynamic attributes use `:` prefix: `:href="state.URL"`. Never write `href="{{ state.URL }}"` or `href={{ state.URL }}`
- Use `this.state` - just `state`
- Access properties on `$event` or `$form` in templates (e.g. `$event.target` is not valid - pass `$event` to your function and access properties there)
- Use `{{ }}` for raw HTML - use `p-html` directive
- Use `:prop` instead of `:server.prop` for partials
- Use `:class` or `:style` to combine static and dynamic values - `:class` **replaces** the static `class` attribute. Use `p-class` / `p-style` instead, which **merge** with the static value (e.g. `class="x" p-class="'y'"` → `class="x y"`)
- Use `===` (doesn't exist - `==` is strict, `~=` is loose)
- Use a non-bool directly in `p-if` without `~` (e.g. `p-if="state.Name"` won't work - use `p-if="~state.Name"`)

## Partials and slots

Do:
- Import partials in the Go script: `import card "myapp/partials/card.pk"`
- Use `is` attribute to specify the partial: `<card is="card">`
- Use `<piko:slot />` for default slots and `<piko:slot name="x">` for named slots
- Use `coerce:"true"` tag for non-string prop types
- Provide meaningful fallback content in slots

Don't:
- Make `is` attribute dynamic (must be static, resolved at compile time)
- Use `<slot>` in server partials (that's for PKC components)
- Define multiple default slots in one partial
- Forget to add `coerce:"true"` when passing booleans or numbers as props

## Server actions

Do:
- Embed `piko.ActionMetadata` in every action struct
- Run `go run ./cmd/generator all` after creating new actions (auto-registers them)
- Match input `name` attributes to `Call` parameter names
- Use `a.Response().AddHelper()` for user feedback
- Use `piko.ValidationField()` for field-level validation errors
- Use pointer types for optional parameters

Don't:
- Forget to run the generator after adding actions (404 at runtime)
- Use `a.Response().AddFieldError()` (does not exist - return `piko.ValidationField()` as an error)
- Use `a.Response().AddCookie()` (the method is `SetCookie`)
- Return errors without user-facing feedback (use `showToast`)
- Use camelCase action names in templates (use snake_case: `action.my_action()`)
- Forget `.prevent` on form submit

## Routing

Do:
- Use `{param}` for dynamic segments in filenames
- Use `{...param}` for catch-all routes
- Use `r.PathParam("name")` to access route parameters
- Return `piko.Metadata{Status: 404}` for missing resources
- Keep route hierarchies shallow (max 3-4 levels)

Don't:
- Use `:param` syntax (Piko uses `{param}`)
- Put pages outside `pages/` directory
- Confuse `PathParam` with `QueryParam`
- Return raw errors for not-found resources

## Error pages

Do:
- Use `!NNN.pk` for exact codes, `!NNN-NNN.pk` for ranges, `!error.pk` for catch-all
- Use `piko.GetErrorContext(r)` to access `StatusCode`, `Message`, `OriginalPath`
- Return typed errors (`piko.NotFound(...)`, `piko.Forbidden(...)`) to trigger error pages
- Place error pages in subdirectories for section-specific error handling

Don't:
- Use invalid `!` filenames (e.g., `!pancakes.pk`) - build will fail
- Forget that exact > range > catch-all priority applies
- Manually handle missing collection items - `p-collection` automatically triggers 404

## Styling

Do:
- Use `var(--g-name)` for theme variables from config.json
- Use `:host([attr="value"])` for string attrs and `:host([attr])` for boolean attrs in PKC styling
- Use `hasAttribute()` / `toggleAttribute()` for boolean state in PKC code (not `getAttribute() === "true"`)
- Use `:deep()` sparingly to reach into child partials

Don't:
- Assume page styles are global (all component types including pages are scoped by default - use `<style global>` for unscoped styles)
- Try to style PKC internals from outside with regular selectors (Shadow DOM)
- Edit files in `dist/` (they get regenerated)

## Built-in elements

Do:
- Use `<piko:a>` for internal navigation (SPA-style, no full reload)
- Use `<piko:img>` for images that need srcset, format variants, or CMS media
- Target rendered elements as plain `a` and `img` in CSS/JS - the `piko:` prefix is stripped in output
- Use `widths` OR `densities` on `piko:img` (not both)

Don't:
- Use `piko:a` or `piko:img` in CSS selectors or JavaScript queries - the `piko:` prefix is stripped in rendered HTML. `<piko:a>` becomes `<a>`, `<piko:img>` becomes `<img>`
- Use a regular `<a>` when you want SPA navigation (it causes a full page reload)
- Use both `widths` and `densities` on the same `piko:img`

## CLI and workflow

Do:
- Run `piko fmt -r .` before committing
- Run `go run ./cmd/generator all` after adding pages/partials/actions
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
