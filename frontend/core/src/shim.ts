// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

/** Callback invoked when a capability implementation becomes available. */
type CapabilityCallback = (impl: unknown) => void;

/** Registered capability implementations keyed by name. */
const capabilities = new Map<string, unknown>();

/** Pending callbacks for capabilities that have not yet registered. */
const capabilityPending = new Map<string, CapabilityCallback[]>();

/**
 * Registers a capability implementation or factory.
 *
 * @param name - The capability name (e.g. 'navigation', 'actions').
 * @param impl - The capability implementation or factory function.
 */
export function _registerCapability(name: string, impl: unknown): void {
    capabilities.set(name, impl);
    const cbs = capabilityPending.get(name);
    if (cbs) {
        capabilityPending.delete(name);
        for (const cb of cbs) { cb(impl); }
    }
}

/**
 * Retrieves a registered capability by name.
 *
 * @param name - The capability name.
 * @returns The capability implementation, or undefined if not yet loaded.
 */
export function _getCapability<T = unknown>(name: string): T | undefined {
    return capabilities.get(name) as T | undefined;
}

/**
 * Checks whether a capability has been registered.
 *
 * @param name - The capability name.
 * @returns True if the capability is available.
 */
export function _hasCapability(name: string): boolean {
    return capabilities.has(name);
}

/**
 * Registers a callback that fires when a capability becomes available.
 *
 * @param name - The capability name.
 * @param cb - Invoked with the capability implementation.
 */
export function _onCapabilityReady(name: string, cb: CapabilityCallback): void {
    const existing = capabilities.get(name);
    if (existing !== undefined) { cb(existing); return; }
    const queue = capabilityPending.get(name);
    if (queue) { queue.push(cb); } else { capabilityPending.set(name, [cb]); }
}

/**
 * Clears all registered capabilities and pending callbacks.
 * Intended for test cleanup to prevent state leaking between test files.
 */
export function _clearCapabilities(): void {
    capabilities.clear();
    capabilityPending.clear();
}

/** Signature for registered helper functions. */
type HelperFn = (el: HTMLElement, event: Event, ...args: string[]) => void | Promise<void>;

/** Registered helper functions keyed by name. */
const helpers = new Map<string, HelperFn>();

/**
 * Registers a helper function for use in templates.
 *
 * @param name - The helper name used in p-on attributes.
 * @param fn - The helper function implementation.
 */
function registerHelper(name: string, fn: HelperFn): void {
    helpers.set(name, fn);
}

/** Callback invoked when a hook event fires. */
type HookCb = (payload: unknown) => void;

/** Hook listener sets keyed by event name. */
const hookListeners = new Map<string, Set<HookCb>>();

/**
 * Subscribes to a hook event.
 *
 * @param event - The hook event name.
 * @param cb - The callback to invoke when the event fires.
 * @returns An unsubscribe function.
 */
function hooksOn(event: string, cb: HookCb): () => void {
    let set = hookListeners.get(event);
    if (!set) { set = new Set(); hookListeners.set(event, set); }
    set.add(cb);
    return () => { set.delete(cb); };
}

/**
 * Removes a specific hook listener.
 *
 * @param event - The hook event name.
 * @param cb - The callback to remove.
 */
function hooksOff(event: string, cb: HookCb): void {
    hookListeners.get(event)?.delete(cb);
}

/**
 * Clears hook listeners for a specific event or all events.
 *
 * @param event - Optional event name. Omit to clear all listeners.
 */
function hooksClear(event?: string): void {
    if (event) { hookListeners.delete(event); } else { hookListeners.clear(); }
}

/**
 * Emits a hook event to all registered listeners.
 *
 * @param event - The hook event name.
 * @param payload - Data to pass to registered callbacks.
 */
function emitHook(event: string, payload: unknown): void {
    hookListeners.get(event)?.forEach(cb => {
        try { cb(payload); } catch (e) { console.error('[piko] Hook error:', e); }
    });
}

/** Function exported from a page or partial script. */
type PageFn = (...args: unknown[]) => unknown;

/** Global page function exports keyed by name. */
const globalExports = new Map<string, PageFn>();

/** Scoped page function exports keyed by partial scope ID. */
const scopedExports = new Map<string, Map<string, PageFn>>();

/** Partial name to ID mappings for cross-partial function resolution. */
const partialInstances = new Map<string, string[]>();

/**
 * Registers page-exported functions, optionally scoped to a partial.
 *
 * @param fns - Map of function names to implementations.
 * @param scopeId - Optional partial scope ID for scoped registration.
 */
function setExports(fns: Record<string, PageFn>, scopeId?: string): void {
    for (const [name, fn] of Object.entries(fns)) {
        globalExports.set(name, fn);
        if (scopeId) {
            let scoped = scopedExports.get(scopeId);
            if (!scoped) { scoped = new Map(); scopedExports.set(scopeId, scoped); }
            scoped.set(name, fn);
        }
    }
}

/**
 * Retrieves a globally exported function by name.
 *
 * @param name - The function name.
 * @returns The function, or undefined if not registered.
 */
function getFunction(name: string): PageFn | undefined {
    return globalExports.get(name);
}

/**
 * Checks whether a function is registered globally.
 *
 * @param name - The function name.
 * @returns True if the function exists.
 */
function hasFunction(name: string): boolean {
    return globalExports.has(name);
}

/**
 * Retrieves a scoped function from the first scope in a partial attribute.
 *
 * @param name - The function name.
 * @param scopeId - The space-separated partial scope attribute value.
 * @returns The scoped function, or undefined if not found.
 */
function getScopedFunction(name: string, scopeId: string): PageFn | undefined {
    const firstScope = scopeId.split(/\s+/)[0];
    return scopedExports.get(firstScope)?.get(name);
}

/**
 * Returns the names of all globally exported functions.
 *
 * @returns Array of exported function names.
 */
function getExportedFunctions(): string[] {
    return Array.from(globalExports.keys());
}

/** Clears all registered page exports and scoped exports. */
function clearPageContext(): void {
    globalExports.clear();
    scopedExports.clear();
}

/**
 * Registers a partial instance mapping for cross-partial calls.
 *
 * @param partialName - The human-readable partial name.
 * @param partialId - The hashed partial scope ID.
 */
function registerPartialInstance(partialName: string, partialId: string): void {
    let ids = partialInstances.get(partialName);
    if (!ids) { ids = []; partialInstances.set(partialName, ids); }
    if (!ids.includes(partialId)) { ids.push(partialId); }
}

/**
 * Dynamically imports an ES module by URL.
 *
 * @param url - The module URL to import.
 */
async function loadModule(url: string): Promise<void> {
    await import(/* @vite-ignore */ url);
}

/**
 * Returns the global page context singleton.
 *
 * Provides access to the page function registry, scoped exports,
 * and partial instance mappings used by generated page scripts.
 *
 * @returns The page context object.
 */
export function getGlobalPageContext(): {
    setExports: typeof setExports;
    getFunction: typeof getFunction;
    hasFunction: typeof hasFunction;
    getScopedFunction: typeof getScopedFunction;
    getExportedFunctions: typeof getExportedFunctions;
    clear: typeof clearPageContext;
    registerPartialInstance: typeof registerPartialInstance;
    loadModule: typeof loadModule;
} {
    return {
        setExports, getFunction, hasFunction, getScopedFunction,
        getExportedFunctions, clear: clearPageContext,
        registerPartialInstance, loadModule,
    };
}

/**
 * Creates a proxy-based refs object that lazily queries elements by p-ref.
 *
 * @param scope - The element to scope queries to (defaults to document.body).
 * @returns A proxy that returns elements by their p-ref name.
 */
function createRefs(scope: Element = document.body): Record<string, HTMLElement | null> {
    const partialId = scope.getAttribute('partial') ?? scope.closest('[partial]')?.getAttribute('partial');
    return new Proxy({} as Record<string, HTMLElement | null>, {
        get(_, name: string | symbol) {
            if (typeof name !== 'string' || name === 'then') { return undefined; }
            let el: HTMLElement | null = null;
            if (partialId) {
                el = document.querySelector(`[partial~="${partialId}"][p-ref="${name}"]`) as HTMLElement | null;
            }
            el ??= scope.querySelector(`[p-ref="${name}"]`) as HTMLElement | null;
            return el;
        }
    });
}

/** Lifecycle callback storage keyed by scope element. */
const lifecycleCallbacks = new WeakMap<Element, Partial<Record<string, Array<(...args: unknown[]) => void>>>>();

/**
 * Registers a lifecycle callback for a partial element.
 *
 * @param scope - The partial root element.
 * @param hookName - The lifecycle hook name (e.g. 'onConnected').
 * @param cb - The callback to register.
 */
function _addLifecycleCallback(scope: Element, hookName: string, cb: (...args: unknown[]) => void): void {
    let state = lifecycleCallbacks.get(scope);
    if (!state) { state = {}; lifecycleCallbacks.set(scope, state); }
    const bucket = state[hookName] ?? [];
    bucket.push(cb);
    state[hookName] = bucket;
}

/** Element-scoped cleanup functions keyed by their root element. */
const elementCleanups = new WeakMap<Element, Array<() => void>>();

/** Page-level cleanup functions that run on navigation. */
const pageCleanups: Array<() => void> = [];

/**
 * Registers a cleanup function to run when a partial disconnects or the page navigates.
 *
 * @param fn - The cleanup function.
 * @param scope - Optional element scope. Omit for page-level cleanup.
 */
function onCleanup(fn: () => void, scope?: Element): void {
    if (scope) {
        let arr = elementCleanups.get(scope);
        if (!arr) { arr = []; elementCleanups.set(scope, arr); }
        arr.push(fn);
    } else {
        pageCleanups.push(fn);
    }
}

/**
 * Creates a file-scoped context object for a PK partial instance.
 *
 * Provides refs, lifecycle registration, and cleanup scoped to the
 * given DOM element. Used as `pk` in generated page script blocks.
 *
 * @param scope - The partial's root element.
 * @returns The context object for use in PK script blocks.
 */
export function _createPKContext(scope: Element): {
    refs: Record<string, HTMLElement | null>;
    createRefs: (s?: Element) => Record<string, HTMLElement | null>;
    onConnected: (cb: () => void) => void;
    onDisconnected: (cb: () => void) => void;
    onBeforeRender: (cb: () => void) => void;
    onAfterRender: (cb: () => void) => void;
    onUpdated: (cb: (ctx?: unknown) => void) => void;
    onCleanup: (fn: () => void) => void;
} {
    return {
        refs: createRefs(scope),
        createRefs: (s?: Element) => createRefs(s ?? scope),
        onConnected: (cb) => _addLifecycleCallback(scope, 'onConnected', cb),
        onDisconnected: (cb) => _addLifecycleCallback(scope, 'onDisconnected', cb),
        onBeforeRender: (cb) => _addLifecycleCallback(scope, 'onBeforeRender', cb),
        onAfterRender: (cb) => _addLifecycleCallback(scope, 'onAfterRender', cb),
        onUpdated: (cb) => _addLifecycleCallback(scope, 'onUpdated', cb),
        onCleanup: (fn) => onCleanup(fn, scope),
    };
}

/** Factory function that creates an ActionDescriptor from arguments. */
type ActionFactory = (...args: unknown[]) => { action: string; args?: unknown[] };

/** Global registry of action functions keyed by Go action name. */
const actionRegistry = new Map<string, ActionFactory>();

/**
 * ActionBuilder implements the ActionDescriptor interface, storing a server
 * action name and its arguments for dispatch through the actions capability.
 */
export class ActionBuilder {
    /** Server action name. */
    action: string;
    /** Arguments to pass to the action. */
    args?: unknown[];

    /**
     * Creates a new ActionBuilder.
     *
     * @param actionName - Server action name.
     * @param actionArgs - Arguments for the action.
     */
    constructor(actionName: string, actionArgs?: unknown[]) {
        this.action = actionName;
        this.args = actionArgs;
    }
}

/**
 * Creates an ActionBuilder from a name and an arguments object, unwrapping
 * protobuf wrappers via `toObject()` when present.
 *
 * @param name - Server action name.
 * @param args - Arguments object to pass to the action.
 * @returns An ActionBuilder instance.
 */
export function createActionBuilder(name: string, args: Record<string, unknown>): ActionBuilder {
    if (typeof args.toObject === 'function') {
        args = (args as unknown as { toObject(): Record<string, unknown> }).toObject();
    }
    return new ActionBuilder(name, [args]);
}

/**
 * Creates an ActionBuilder with variadic arguments.
 *
 * @param name - Server action name.
 * @param args - Arguments to pass to the action.
 * @returns An ActionBuilder instance.
 */
export function action(name: string, ...args: unknown[]): ActionBuilder {
    return new ActionBuilder(name, args);
}

/**
 * Registers an action function in the global registry.
 *
 * @param name - The Go action name (e.g. "email.Contact").
 * @param factory - The wrapper function that returns an ActionDescriptor.
 */
export function registerActionFunction(name: string, factory: ActionFactory): void {
    actionRegistry.set(name, factory);
}

/**
 * Looks up an action function by its Go action name.
 *
 * @param name - The Go action name.
 * @returns The wrapper function, or undefined if not registered.
 */
export function getActionFunction(name: string): ActionFactory | undefined {
    return actionRegistry.get(name);
}

/**
 * Checks if a value is an ActionDescriptor using duck typing.
 *
 * @param value - The value to check.
 * @returns True if the value has an 'action' property of type string.
 */
export function isActionDescriptor(value: unknown): boolean {
    return value !== null && typeof value === 'object' && typeof (value as { action: unknown }).action === 'string';
}

registerHelper('submitForm', (el) => {
    const form = el.closest('form');
    if (form) { form.requestSubmit(); }
});

registerHelper('submitModalForm', (el) => {
    const form = el.closest('form');
    if (form) { form.requestSubmit(); }
});

registerHelper('resetForm', (el) => {
    const form = el.closest('form');
    if (form) { form.reset(); }
});

registerHelper('redirect', (_el, _event, ...args) => {
    const url = args[0];
    if (!url) { return; }
    const lower = url.toLowerCase();
    if (BLOCKED_SCHEMES.some(s => lower.startsWith(s))) { return; }
    window.location.href = url;
});

registerHelper('emitEvent', (el, _event, ...args) => {
    const eventName = args[0];
    if (eventName) {
        el.dispatchEvent(new CustomEvent(eventName, { bubbles: true, composed: true, detail: args.slice(1) }));
    }
});

registerHelper('dispatchEvent', (_el, _event, ...args) => {
    const eventName = args[0];
    if (eventName) {
        window.dispatchEvent(new CustomEvent(eventName, { detail: args.slice(1) }));
    }
});

/** Attribute marker indicating an element's events have been bound. */
const BOUND_MARKER = 'pk-ev-bound';

/** Prefix for standard event bindings (p-on:click, p-on:submit, etc.). */
const EVENT_ATTR_PREFIX = 'p-on:';

/** Prefix for custom event bindings (p-event:my-event, etc.). */
const CUSTOM_EVENT_ATTR_PREFIX = 'p-event:';

/** URI schemes that should be blocked entirely for security reasons. */
// eslint-disable-next-line no-script-url
const BLOCKED_SCHEMES = ['javascript:', 'data:', 'blob:', 'file:'];

/** URI schemes handled natively by the browser (tel:, mailto:, etc.). */
const NATIVE_SCHEMES = ['tel:', 'mailto:', 'sms:', 'geo:'];

/**
 * Binds click handlers to all piko:a anchor elements within a root.
 *
 * @param root - The root element to scan for piko:a links.
 * @param onNavigate - Callback invoked for SPA navigation.
 */
function bindLinks(root: HTMLElement, onNavigate: (url: string, event: MouseEvent) => void): void {
    root.querySelectorAll<HTMLAnchorElement>('a[piko\\:a]').forEach(link => {
        const existing = (link as unknown as { __pkNav?: EventListener }).__pkNav;
        if (existing) { link.removeEventListener('click', existing); }
        const handler = (event: MouseEvent) => {
            const href = link.getAttribute('href');
            if (!href) { return; }
            const lower = href.toLowerCase();
            if (BLOCKED_SCHEMES.some(s => lower.startsWith(s))) { event.preventDefault(); return; }
            if (NATIVE_SCHEMES.some(s => lower.startsWith(s))) { return; }
            event.preventDefault();
            onNavigate(href, event);
        };
        (link as unknown as { __pkNav: EventListener }).__pkNav = handler as EventListener;
        link.addEventListener('click', handler);
    });
}

/**
 * Converts a URL-safe base64 string (RFC 4648 §5) to standard base64 for atob().
 *
 * @param s - The URL-safe base64 string.
 * @returns The decoded string.
 */
function b64Decode(s: string): string {
    const BASE64_BLOCK = 4;
    let std = s.replace(/-/g, '+').replace(/_/g, '/');
    const pad = (BASE64_BLOCK - (std.length % BASE64_BLOCK)) % BASE64_BLOCK;
    std += '='.repeat(pad);
    return atob(std);
}

/** Shape of an encoded argument in a base64 action payload. */
interface EncodedArg { t?: string; v?: unknown }

/**
 * Unwraps an encoded argument, substituting $event and $form placeholders.
 *
 * @param arg - The encoded argument object.
 * @param el - The element that triggered the event.
 * @param event - The browser event.
 * @returns The resolved argument value.
 */
function unwrapArgWithInjection(arg: unknown, el: HTMLElement, event: Event): unknown {
    if (arg && typeof arg === 'object') {
        const encoded = arg as EncodedArg;
        if (encoded.t === 'e') { return event; }
        if (encoded.t === 'f') {
            const form = el.closest('form');
            if (!form) { return Object.create(null) as Record<string, unknown>; }
            const fd = new FormData(form);
            const obj = Object.create(null) as Record<string, unknown>;
            for (const [k, v] of fd.entries()) { obj[k] = v; }
            return obj;
        }
        return encoded.v;
    }
    return undefined;
}

/**
 * Dispatches an action descriptor through the actions capability when available.
 *
 * @param descriptor - The value to check and dispatch.
 * @param el - The element that triggered the action.
 * @param event - The browser event.
 */
function dispatchActionDescriptor(descriptor: unknown, el: HTMLElement, event: Event): void {
    if (!isActionDescriptor(descriptor)) { return; }
    const api = _getCapability<{ handleAction(d: unknown, el: HTMLElement, ev?: Event): Promise<void> }>('actions');
    if (api) { void api.handleAction(descriptor, el, event); }
}

/**
 * Attempts to resolve and invoke a registered action function.
 *
 * @param fnName - The action function name.
 * @param encodedArgs - The encoded argument list from the payload.
 * @param el - The element that triggered the event.
 * @param event - The browser event.
 * @returns True if a matching action function was found and invoked.
 */
function tryInvokeActionFn(fnName: string, encodedArgs: unknown[] | undefined, el: HTMLElement, event: Event): boolean {
    const actionFn = actionRegistry.get(fnName);
    if (!actionFn) { return false; }
    const args = encodedArgs?.map(a => unwrapArgWithInjection(a, el, event)) ?? [];
    dispatchActionDescriptor(actionFn(...args), el, event);
    return true;
}

/**
 * Resolves a `@partialName.fn` qualified reference by walking every registered
 * instance of the named partial until a scoped export matches.
 *
 * @param fnName - The qualified function name, starting with '@'.
 * @returns The bare function name and its resolved function, or null for invalid input.
 */
function resolveQualifiedPartialFn(fnName: string): { bareName: string; fn: PageFn | undefined } | null {
    const dotIdx = fnName.indexOf('.');
    if (dotIdx <= 1) { return null; }
    const partialName = fnName.slice(1, dotIdx);
    const bareName = fnName.slice(dotIdx + 1);
    const ids = partialInstances.get(partialName);
    if (!ids) { return { bareName, fn: undefined }; }
    for (const id of ids) {
        const fn = getScopedFunction(bareName, id);
        if (fn) { return { bareName, fn }; }
    }
    return { bareName, fn: undefined };
}

/**
 * Attempts to resolve and invoke a page-exported function.
 *
 * @param fnName - The function name to look up.
 * @param encodedArgs - The encoded argument list from the payload.
 * @param el - The element that triggered the event.
 * @param event - The browser event.
 * @returns True if a matching page function was found and invoked.
 */
function tryInvokePageFn(fnName: string, encodedArgs: unknown[] | undefined, el: HTMLElement, event: Event): boolean {
    let resolvedName = fnName;
    let pageFn: PageFn | undefined;

    if (fnName.startsWith('@')) {
        const qualified = resolveQualifiedPartialFn(fnName);
        if (qualified) {
            resolvedName = qualified.bareName;
            pageFn = qualified.fn;
        }
    }

    if (!pageFn) {
        const scopeId = el.closest('[partial]')?.getAttribute('partial') ?? '';
        pageFn = getScopedFunction(resolvedName, scopeId) ?? getFunction(resolvedName);
    }
    if (!pageFn) { return false; }

    const args = encodedArgs?.map(a => unwrapArgWithInjection(a, el, event)) ?? [];
    const result = pageFn(event, ...args);
    if (result instanceof Promise) {
        void result.then((resolved: unknown) => {
            dispatchActionDescriptor(resolved, el, event);
        }).catch((err: unknown) => {
            console.error('[piko] Async handler error:', err);
        });
    } else {
        dispatchActionDescriptor(result, el, event);
    }
    return true;
}

/**
 * Attempts to resolve and invoke a registered helper by name.
 *
 * Helpers receive stringified arguments: $event becomes the event type
 * string, $form becomes a JSON-serialised form-data object, and plain
 * values are coerced via String(). This matches the helper signature
 * which expects variadic string arguments only.
 *
 * @param fnName - The helper name to look up.
 * @param encodedArgs - The encoded argument list from the payload.
 * @param el - The element that triggered the event.
 * @param event - The browser event.
 * @returns True if a matching helper was found and invoked.
 */
function tryInvokeHelper(fnName: string, encodedArgs: unknown[] | undefined, el: HTMLElement, event: Event): boolean {
    const helper = helpers.get(fnName);
    if (!helper) { return false; }
    const args = encodedArgs?.map(a => {
        const encoded = a as EncodedArg;
        if (encoded.t === 'e') { return event.type; }
        if (encoded.t === 'f') {
            const form = el.closest('form');
            if (!form) { return ''; }
            const obj = Object.create(null) as Record<string, unknown>;
            for (const [k, v] of new FormData(form).entries()) { obj[k] = v; }
            return JSON.stringify(obj);
        }
        return String(encoded.v);
    }) ?? [];
    void Promise.resolve(helper(el, event, ...args));
    return true;
}

/**
 * Resolves a base64-encoded action payload and dispatches it.
 *
 * Looks the handler name up in the action registry, the page-exported
 * function registry, and the helper registry, invoking the first match
 * with unwrapped arguments (including $event and $form placeholder
 * injection). Custom events (p-event:*) prefer page functions over action
 * factories so user-defined handlers shadow generated ones; native events
 * (p-on:*) prefer action factories so declarative server actions bind
 * directly without needing a wrapper.
 *
 * @param payload - The base64-encoded JSON action payload.
 * @param el - The element that triggered the event.
 * @param event - The browser event.
 * @param isCustomEvent - True when the binding comes from a p-event:* attribute.
 */
function resolveAndDispatch(payload: string, el: HTMLElement, event: Event, isCustomEvent: boolean): void {
    try {
        const decoded = JSON.parse(b64Decode(payload)) as { f: string; a?: unknown[] };
        const fnName = decoded.f;
        if (isCustomEvent) {
            if (tryInvokePageFn(fnName, decoded.a, el, event)) { return; }
            if (tryInvokeHelper(fnName, decoded.a, el, event)) { return; }
            if (tryInvokeActionFn(fnName, decoded.a, el, event)) { return; }
        } else {
            if (tryInvokeActionFn(fnName, decoded.a, el, event)) { return; }
            if (tryInvokePageFn(fnName, decoded.a, el, event)) { return; }
            if (tryInvokeHelper(fnName, decoded.a, el, event)) { return; }
        }
        console.warn(`[piko] Handler "${fnName}" not found.`);
    } catch (e) {
        console.error('[piko] Failed to resolve action payload:', e);
    }
}

/**
 * Binds action and modal handlers to all elements within a root.
 *
 * Scans for p-on:*, p-event:*, and p-modal:selector attributes and attaches
 * event listeners that resolve payloads through the action, page, and helper
 * registries.
 *
 * @param root - The root element to scan for action attributes.
 */
function bindActions(root: HTMLElement): void {
    root.querySelectorAll<HTMLElement>('*').forEach(el => {
        if (el.hasAttribute(BOUND_MARKER)) { return; }
        let hasBound = false;

        for (const {name: attrName, value: attrValue} of Array.from(el.attributes)) {
            if (attrName.startsWith(EVENT_ATTR_PREFIX) || attrName.startsWith(CUSTOM_EVENT_ATTR_PREFIX)) {
                const isCustom = attrName.startsWith(CUSTOM_EVENT_ATTR_PREFIX);
                const prefixLength = isCustom ? CUSTOM_EVENT_ATTR_PREFIX.length : EVENT_ATTR_PREFIX.length;
                const key = attrName.slice(prefixLength);
                const parts = key.split('.');
                const eventName = parts[0].trim();
                if (!eventName) { continue; }
                const modifiers = new Set(parts.slice(1));
                const listenerOptions: AddEventListenerOptions = {};
                if (modifiers.has('capture')) { listenerOptions.capture = true; }
                if (modifiers.has('passive')) { listenerOptions.passive = true; }

                const boundEl = el;
                const payload = attrValue;
                const isCustomEvent = isCustom;
                let firedOnce = false;
                el.addEventListener(eventName, (event) => {
                    if (modifiers.has('self') && event.target !== event.currentTarget) { return; }
                    if (modifiers.has('prevent')) { event.preventDefault(); }
                    if (modifiers.has('stop')) { event.stopPropagation(); }
                    if (modifiers.has('once')) {
                        if (firedOnce) { return; }
                        firedOnce = true;
                    }

                    resolveAndDispatch(payload, boundEl, event, isCustomEvent);
                }, listenerOptions);
                hasBound = true;
            }
        }

        if (el.hasAttribute('p-modal:selector')) {
            el.addEventListener('click', () => {
                const params = new Map<string, string>();
                for (const {name, value} of Array.from(el.attributes)) {
                    if (name.startsWith('p-modal-param:')) {
                        params.set(name.slice('p-modal-param:'.length).trim(), value.trim());
                    }
                }
                el.dispatchEvent(new CustomEvent('pk-open-modal', {
                    bubbles: true,
                    detail: {
                        selector: el.getAttribute('p-modal:selector')?.trim() ?? '',
                        params,
                        title: el.getAttribute('p-modal:title')?.trim() ?? '',
                        message: el.getAttribute('p-modal:message')?.trim() ?? '',
                        cancelLabel: el.getAttribute('p-modal:cancel_label')?.trim() ?? '',
                        confirmLabel: el.getAttribute('p-modal:confirm_label')?.trim() ?? '',
                        confirmAction: el.getAttribute('p-modal:confirm_action')?.trim() ?? '',
                        element: el,
                    }
                }));
            });
            hasBound = true;
        }

        if (hasBound) { el.setAttribute(BOUND_MARKER, 'true'); }
    });
}

/** Cached module configuration parsed from the DOM. */
let moduleConfigCache: Record<string, unknown> | null = null;

/**
 * Retrieves runtime configuration for a frontend module.
 *
 * Reads from the `<script id="pk-module-config">` element in the DOM,
 * caching the parsed result for subsequent calls.
 *
 * @param moduleName - The module name to retrieve configuration for.
 * @returns The module configuration, or null if not found.
 */
function getModuleConfig<T = unknown>(moduleName: string): T | null {
    if (moduleConfigCache === null) {
        const configEl = document.getElementById('pk-module-config');
        if (configEl?.textContent) {
            try {
                moduleConfigCache = JSON.parse(configEl.textContent) as Record<string, unknown>;
            } catch {
                moduleConfigCache = {};
            }
        } else {
            moduleConfigCache = {};
        }
    }
    return (moduleConfigCache[moduleName] as T) ?? null;
}

/** Registered ready callbacks, or null once the framework has finished initialising. */
let readyCallbacks: Array<() => void> | null = [];

/** Whether the framework has finished initialising and is ready for use. */
let isReady = false;

/**
 * The global Piko namespace object assigned to window.piko.
 *
 * Exposes the API surface that capabilities and extensions use to register
 * themselves, subscribe to hooks, and access module config.
 */
const pikoNamespace = {
    ready(cb: () => void): void {
        if (isReady) { cb(); return; }
        readyCallbacks?.push(cb);
    },
    _markReady(): void {
        isReady = true;
        const cbs = readyCallbacks;
        readyCallbacks = null;
        if (cbs) { for (const cb of cbs) { try { cb(); } catch (e) { console.error('[piko] Ready callback error:', e); } } }
    },
    registerHelper,
    getModuleConfig,
    _registerCapability,
    _emitHook: emitHook,
    hooks: {
        on: hooksOn,
        once(event: string, cb: HookCb): () => void {
            const unsub = hooksOn(event, (payload) => { unsub(); cb(payload); });
            return unsub;
        },
        off: hooksOff,
        clear: hooksClear,
        events: {
            FRAMEWORK_READY: 'framework:ready',
            PAGE_VIEW: 'page:view',
            NAVIGATION_START: 'navigation:start',
            NAVIGATION_COMPLETE: 'navigation:complete',
            NAVIGATION_ERROR: 'navigation:error',
            ACTION_START: 'action:start',
            ACTION_COMPLETE: 'action:complete',
            MODAL_OPEN: 'modal:open',
            MODAL_CLOSE: 'modal:close',
            PARTIAL_RENDER: 'partial:render',
            FORM_DIRTY: 'form:dirty',
            FORM_CLEAN: 'form:clean',
            NETWORK_ONLINE: 'network:online',
            NETWORK_OFFLINE: 'network:offline',
            ERROR: 'error',
        },
    },
    bus: (() => {
        const listeners = new Map<string, Set<(data: unknown) => void>>();
        return {
            on(event: string, cb: (data: unknown) => void): () => void {
                let set = listeners.get(event);
                if (!set) { set = new Set(); listeners.set(event, set); }
                set.add(cb);
                return () => { set.delete(cb); };
            },
            once(event: string, cb: (data: unknown) => void): () => void {
                const wrapper = (data: unknown) => { listeners.get(event)?.delete(wrapper); cb(data); };
                return this.on(event, wrapper);
            },
            off(event?: string): void {
                if (event) { listeners.delete(event); } else { listeners.clear(); }
            },
            emit(event: string, data?: unknown): void {
                listeners.get(event)?.forEach(cb => {
                    try { cb(data); } catch (e) { console.error(`[pk] Bus error for "${event}":`, e); }
                });
            },
        };
    })(),
    nav: {
        navigate(url: string): void {
            const nav = _getCapability<{ navigateTo(url: string): Promise<void> }>('navigation');
            if (nav) { void nav.navigateTo(url); } else { window.location.href = url; }
        },
        back(): void { window.history.back(); },
        forward(): void { window.history.forward(); },
        go(delta: number): void { window.history.go(delta); },
    },
    context: {
        get: getGlobalPageContext,
    },
};

if (typeof window !== 'undefined') {
    (window as unknown as { piko: typeof pikoNamespace }).piko = pikoNamespace;
    (window as unknown as { __pikoShimData__: unknown }).__pikoShimData__ = {
        hookListeners, helpers, capabilities, capabilityPending,
        globalExports, scopedExports, partialInstances,
        actionRegistry, lifecycleCallbacks, elementCleanups, pageCleanups,
        readyCallbacks: () => readyCallbacks,
        moduleConfigCache: () => moduleConfigCache,
    };
}

if (typeof document !== 'undefined') {
    const appRoot = document.querySelector('#app') as HTMLElement | null;
    if (appRoot) {
        bindLinks(appRoot, (url) => {
            pikoNamespace.nav.navigate(url);
        });
        bindActions(appRoot);
    }
}

pikoNamespace._markReady();

/** Re-exported event bus for generated code and cross-component messaging. */
export const bus = pikoNamespace.bus;

/** Re-exported refs factory for generated page scripts that import _createRefs. */
export { createRefs as _createRefs };
