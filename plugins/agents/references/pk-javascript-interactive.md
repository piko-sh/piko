# PK Client-Side Interactivity

Use this guide when adding client-side behaviour to `.pk` files - lifecycle hooks, event handling, DOM references, cleanup, the event bus, and the `piko` namespace API.

## Client-side script block

PK files support an optional `<script lang="ts">` block for client-side TypeScript. It runs once per partial/page instance and co-exists with the server-side Go block:

```piko
<template>
  <div>
    <p>Count: <span p-ref="counter">0</span></p>
    <button p-on:click="increment()">+1</button>
  </div>
</template>

<script type="application/x-go">
package main

import "piko.sh/piko"

func Render(r *piko.RequestData, props piko.NoProps) (piko.NoResponse, piko.Metadata, error) {
    return piko.NoResponse{}, piko.Metadata{}, nil
}
</script>

<script lang="ts">
let count = 0;

export function increment() {
    count++;
    pk.refs.counter.textContent = String(count);
}
</script>
```

**Key points**: Use `lang="ts"` (not `type="text/typescript"`). The block is TypeScript - type annotations are supported. Top-level code runs immediately when the partial mounts.

## The `pk` context object

`pk` is scoped to the current partial/page instance. It provides DOM references and lifecycle hooks.

### pk.refs

Access DOM elements marked with `p-ref="name"` in the template:

```piko
<template>
  <input p-ref="nameInput" type="text" />
  <span p-ref="output"></span>
</template>

<script lang="ts">
pk.onConnected(() => {
    pk.refs.nameInput.addEventListener('input', () => {
        pk.refs.output.textContent = pk.refs.nameInput.value;
    });
});
</script>
```

`pk.refs` returns `HTMLElement` references. They are available once the script runs (the DOM is ready at that point).

### Lifecycle hooks

| Hook | Signature | When it fires |
|------|-----------|---------------|
| `pk.onConnected(cb)` | `() => void` | Partial mounts in the DOM |
| `pk.onDisconnected(cb)` | `() => void` | Partial is removed from the DOM |
| `pk.onBeforeRender(cb)` | `() => void` | Before a partial re-render (refresh) |
| `pk.onAfterRender(cb)` | `() => void` | After a partial re-render |
| `pk.onUpdated(cb)` | `(context?: unknown) => void` | Server pushes updated content |
| `pk.onCleanup(cb)` | `() => void` | Registers teardown (fires on disconnect) |

Multiple callbacks per hook are allowed - they fire in registration order.

**onConnected** - set up listeners, timers, subscriptions, third-party libraries:

```typescript
pk.onConnected(() => {
    const handler = () => { /* ... */ };
    window.addEventListener('resize', handler);
    pk.onCleanup(() => window.removeEventListener('resize', handler));
});
```

**onDisconnected** - respond to removal (analytics, notifications). For resource cleanup, prefer `onCleanup` instead - it co-locates teardown with setup.

**onUpdated** - fires when the server pushes new HTML for this partial. Receives an optional `context` parameter. Use it to re-initialise JavaScript after server content changes.

> **PK vs PKC**: PK's `onUpdated` signals a server push with optional context. PKC's `onUpdated` fires after a reactive state change with a `Set<string>` of changed property names.

**onCleanup** - fires after `onDisconnected` callbacks, then clears. Co-locate with setup:

```typescript
pk.onConnected(() => {
    const id = setInterval(pollStatus, 5000);
    pk.onCleanup(() => clearInterval(id));

    const unsub = piko.bus.on('refresh', handleRefresh);
    pk.onCleanup(() => unsub());
});
```

## Exporting functions for `p-on:`

Functions referenced by `p-on:` directives in the template should be defined in the `<script lang="ts">` block. Use `export` to make intent clear:

```piko
<template>
  <button p-on:click="handleClick()">Click</button>
  <button p-on:click="handleEvent">No parens</button>
  <form p-on:submit.prevent="handleSubmit($form)">...</form>
</template>

<script lang="ts">
export function handleClick() {
    // Called on click
}

export function handleEvent(event: Event) {
    // No parentheses in p-on: passes the native event as first argument
}

export function handleSubmit(formData: FormDataHandle) {
    // $form passes a FormDataHandle with .toObject(), .get(key), etc.
}
</script>
```

**`$form` special argument**: When used inside `p-on:submit.prevent`, `$form` provides a `FormDataHandle` with methods: `toObject()`, `toFormData()`, `toJSON()`, `get(key)`, `has(key)`, `getAll(key)`.

## Event handling and cleanup patterns

### Manual listeners with cleanup

```typescript
pk.onConnected(() => {
    const onKeydown = (e: KeyboardEvent) => {
        if (e.key === 'Escape') closePanel();
    };
    document.addEventListener('keydown', onKeydown);
    pk.onCleanup(() => document.removeEventListener('keydown', onKeydown));
});
```

### AbortController pattern

```typescript
pk.onConnected(() => {
    const controller = new AbortController();
    window.addEventListener('scroll', handleScroll, { signal: controller.signal });
    window.addEventListener('resize', handleResize, { signal: controller.signal });
    pk.onCleanup(() => controller.abort());
});
```

## The `piko` namespace

`piko` is a global namespace available in all `<script lang="ts">` blocks. It provides cross-partial communication, navigation, form utilities, and more.

### piko.bus - event bus

Pub/sub communication between partials:

```typescript
// Emit
piko.bus.emit('item-added', { id: '123', name: 'Widget' });

// Subscribe (returns unsubscribe function)
const unsub = piko.bus.on('item-added', (data) => {
    console.log('Added:', data);
});

// One-shot listener
piko.bus.once('init-complete', (data) => { /* ... */ });

// Unsubscribe
unsub();

// Remove all listeners for an event
piko.bus.off('item-added');

// Remove all listeners
piko.bus.off();
```

### piko.hooks - global lifecycle hooks

Subscribe to framework-level events for analytics, integrations, and side effects:

```typescript
piko.hooks.on('page:view', (payload) => {
    analytics.track('pageview', { url: payload.url, title: payload.title });
});

piko.hooks.on('action:complete', (payload) => {
    if (!payload.success) showErrorBanner();
});
```

All payloads include a `timestamp` field (Unix ms). Other fields per event:

| Hook | Payload | When it fires |
|------|---------|---------------|
| `framework:ready` | `{ version, loadTime }` | Framework initialised |
| `page:view` | `{ url, title, referrer, isInitialLoad }` | Page navigation |
| `navigation:start` | `{ url, previousUrl? }` | SPA navigation begins |
| `navigation:complete` | `{ url, previousUrl?, duration }` | SPA navigation finishes |
| `navigation:error` | `{ url, error }` | Navigation fails |
| `action:start` | `{ action, method, elementTag }` | Server action begins |
| `action:complete` | `{ action, method, elementTag, success, statusCode, duration, validationFailed? }` | Server action finishes |
| `modal:open` | `{ modalId?, url? }` | Modal opens |
| `modal:close` | `{ modalId?, url? }` | Modal closes |
| `partial:render` | `{ src, patchLocation }` | Partial rendered |
| `form:dirty` | `{ formId? }` | Form becomes dirty |
| `form:clean` | `{ formId? }` | Form becomes clean |
| `network:online` | `{}` | Browser goes online |
| `network:offline` | `{}` | Browser goes offline |
| `error` | `{ type, message, context, url?, stack? }` | Error occurs (`context: 'navigation' \| 'action' \| 'render' \| 'unknown'`) |
| `analytics:track` | `{ eventName, params }` | `piko.analytics.track()` called |

Methods: `on(event, cb, opts?)` returns unsubscribe fn, `once(event, cb, opts?)`, `off(event, id)`, `clear(event?)`.

### piko.nav - SPA navigation

```typescript
piko.nav.navigate('/products/123');
piko.nav.navigate('/login', { replace: true });
piko.nav.back();
piko.nav.forward();
piko.nav.go(-2);

const route = piko.nav.current();
// route.path, route.query, route.hash, route.href, route.getParam('id')

const url = piko.nav.buildUrl('/search', { q: 'widget', page: '2' });
await piko.nav.updateQuery({ sort: 'price', page: null }); // null removes param

// Navigation guard
const unguard = piko.nav.guard({
    beforeNavigate: (to, from) => {
        if (hasUnsavedChanges()) return false; // Cancel navigation
        return true;
    },
    afterNavigate: (to, from) => { /* ... */ },
});
```

### piko.form - form utilities

```typescript
const handle = piko.form.data('#my-form');
const obj = handle.toObject();
const val = handle.get('email');

piko.form.reset('#my-form');
piko.form.setValues('#my-form', { name: 'Alice', email: 'alice@example.com' });

const result = piko.form.validate('#my-form', {
    email: { required: true, format: 'email' },
    name: { required: true, minLength: 2 },
});
if (!result.isValid) result.focus(); // Focus first invalid field
```

### piko.ui - loading and retry

```typescript
// Show loading state while promise resolves
await piko.ui.loading('#submit-btn', fetchData(), {
    className: 'loading',
    text: 'Saving...',
    disabled: true,
    minDuration: 300,
});

// Wrap an async function with loading state
await piko.ui.withLoading('#submit-btn', async () => {
    return await saveData();
});

// Retry with exponential backoff
const result = await piko.ui.withRetry(() => fetchData(), {
    attempts: 3,
    backoff: 'exponential',
    delay: 1000,
});
```

### piko.event - custom DOM events

```typescript
// Dispatch
piko.event.dispatch('#my-el', 'item-selected', { id: '123' });

// Listen (returns unsubscribe fn)
const unsub = piko.event.listen('#my-el', 'item-selected', (e) => {
    console.log(e.detail);
});

// Listen once
piko.event.listenOnce('#my-el', 'ready', (e) => { /* ... */ });

// Wait for event (returns promise, optional timeout)
const data = await piko.event.waitFor('#my-el', 'loaded', 5000);
```

### piko.partials - partial reload

```typescript
// Reload a single partial
await piko.partials.reload('customer-list', {
    args: { page: 2 },
    loading: true,
    retry: 2,
});

// Reload multiple partials
await piko.partials.reloadGroup(['sidebar', 'content'], {
    mode: 'parallel',
    onProgress: (done, total) => { /* ... */ },
});

// Auto-refresh a partial
const stop = piko.partials.autoRefresh('notifications', {
    interval: 30000,
    when: () => document.visibilityState === 'visible',
});
pk.onCleanup(stop);
```

### piko.sse - server-sent events

```typescript
const unsub = piko.sse.subscribe('live-feed', {
    url: '/api/sse/feed',
    onMessage: (data) => { /* ... */ },
    onOpen: () => console.log('Connected'),
    onClose: () => console.log('Closed'),
    onError: 'reconnect',
    reconnectDelay: 3000,
    maxReconnects: 10,
});
pk.onCleanup(unsub);
```

### piko.timing - debounce, throttle, poll

```typescript
const debouncedSearch = piko.timing.debounce(doSearch, 300);
const throttledScroll = piko.timing.throttle(handleScroll, 100);

// Poll until condition
const stopPolling = piko.timing.poll(() => checkStatus(), {
    interval: 2000,
    until: () => isComplete,
    maxAttempts: 30,
});
pk.onCleanup(stopPolling);

// Async variants with cancellation
const debouncedFetch = piko.timing.debounceAsync(fetchResults, 300);
debouncedFetch.cancel(); // Cancel pending

// Cancellable timeout
const { promise, cancel } = piko.timing.timeout(5000);

// Wait for animation frames
await piko.timing.nextFrame();
await piko.timing.waitFrames(3);
```

### piko.util - advanced utilities

```typescript
// Run callback when element is visible
const stop = piko.util.whenVisible('#lazy-section', (entry) => {
    loadContent();
}, { once: true, threshold: 0.5 });

// Managed abort signal
const op = piko.util.withAbortSignal(async (signal) => {
    const res = await fetch('/api/data', { signal });
    return res.json();
});
pk.onCleanup(() => op.abort());

// Watch DOM mutations
const stopWatch = piko.util.watchMutations('#container', (mutations) => {
    // React to DOM changes
}, { childList: true, subtree: true });

// Run during idle time
piko.util.whenIdle(() => precomputeData());

// One-shot function
const loadOnce = piko.util.once(() => expensiveSetup());
```

### piko.trace - debugging

```typescript
piko.trace.enable({ partialReloads: true, events: true });
piko.trace.log('custom-event', { someData: 123 });
await piko.trace.async('data-fetch', async () => {
    return await fetchData();
});
const entries = piko.trace.getEntries();
const metrics = piko.trace.getMetrics();
piko.trace.disable();
```

### piko.ready and piko.registerHelper

```typescript
// Run when framework is initialised
piko.ready(() => {
    console.log('Piko framework ready');
});

// Register custom action response helper
piko.registerHelper('showToast', (el, ev, message, variant) => {
    document.dispatchEvent(new CustomEvent('pk-show-toast', {
        detail: { message, variant: variant ?? 'info' }
    }));
});
```

## Action builder API

Call server actions from `<script lang="ts">` using the action builder pattern:

```typescript
// Basic call
const result = await action.customer.Create({
    name: 'Acme Corp',
    email: 'contact@acme.com',
}).call();

// With debounce and loading indicator
const result = await action.search.Query({ q: searchTerm })
    .withDebounce(300)
    .withLoading('#search-btn')
    .call();

// SSE streaming with progress
const result = await action.stream.Progress({ taskId: 'abc' })
    .withOnProgress((data, eventType) => {
        updateProgress(data);
    })
    .call();

// Suppress automatic helpers (e.g. toasts)
const result = await action.item.Delete({ id: '123' })
    .suppressHelpers()
    .call();
```

Pattern: `action.<namespace>.<Method>({args}).<chainable>().call()`.

Chainable methods: `.withDebounce(ms)`, `.withLoading(selector|element)`, `.withOnProgress(cb)`, `.suppressHelpers()`.

## Partial refresh attributes

Control how partials behave during server-side refresh:

| Attribute | Purpose |
|-----------|---------|
| `pk-no-refresh` | Skip this element during partial refresh |
| `pk-refresh-root` | Mark this element as the refresh boundary |
| `pk-own-attrs` | Comma-separated list of server-owned attributes that ARE updated during refresh; all unlisted attributes are preserved |
| `pk-no-refresh-attrs` | Skip attribute refresh entirely on this element |
| `pk-loading` | CSS class automatically toggled during partial reload |

## LLM mistake checklist

- Using `pkc.` instead of `pk.` in PK files - PK files use `pk.refs`, `pk.onConnected`, etc. `pkc.` is for `.pkc` Web Components only
- Forgetting cleanup - always pair `addEventListener` with `removeEventListener` via `pk.onCleanup()` or `AbortController`
- Using `type="text/typescript"` instead of `lang="ts"` on the script tag
- Referencing `pk.refs.name` before the DOM is ready - refs are available at script execution time (DOM is ready), but be careful with elements inside `p-if`/`p-for` that may not exist yet
- Writing `piko.bus.on(...)` without storing and cleaning up the unsubscribe function
- Using `this.refs` - there is no `this` context in PK scripts; use `pk.refs`
- Confusing PK's `onUpdated` (server push, receives optional context) with PKC's `onUpdated` (reactive state change, receives `Set<string>`)

## Related

- `references/pk-file-format.md` - .pk structure, Go server script, template, style
- `references/pkc-components.md` - .pkc Web Components, reactive state, attribute sync
- `references/template-syntax.md` - directives, interpolation, expressions
- `references/server-actions.md` - actions, form handling, response helpers
- `references/examples.md` - annotated code examples for common patterns
