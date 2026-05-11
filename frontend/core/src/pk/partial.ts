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

import fragmentMorpher from '@/core/fragmentMorpher';
import {notifyDOMUpdated} from '@/pk/domUpdater';
import {applyLoadingIndicator, removeLoadingIndicator} from '@/pk/loadingState';
import {getNodeKey} from '@/pk/getNodeKey';

/**
 * How a partial reload reconciles the live partial root with the server response.
 *
 * - `merge` (default): server-emitted attributes overwrite their live counterparts;
 *   live-only attributes are preserved. The `partial` scope chain is merged
 *   rather than overwritten. Children are morphed.
 * - `replace`: server is fully authoritative for root attributes; live-only
 *   attributes are removed. Children are morphed.
 * - `children-only`: root attributes are not touched; only children are morphed.
 * - `attrs-only`: root attributes follow the merge rule; children are not
 *   touched.
 */
export type ReloadMode = 'merge' | 'replace' | 'children-only' | 'attrs-only';

/** Attribute names framework infrastructure owns; never copied across a morph. */
const NEVER_SYNC_ATTRS = new Set(['pk-ev-bound', 'pk-sync-bound']);

/**
 * The `partial` attribute is space-separated and stacks self-scope on top of
 * inherited parent scopes. It must be merged rather than overwritten on every
 * mode that touches root attributes, otherwise a child partial's parent CSS
 * scopes are lost on each reload.
 */
const SCOPE_ATTR = 'partial';

/** Options for partial reload with fine-grained control. */
export interface PartialReloadOptions {
    /** Query parameters to pass to the server. */
    data?: Record<string, string | number | boolean>;
    /** Reconciliation mode for the partial root. Defaults to `'merge'`. */
    mode?: ReloadMode;
    /**
     * Attributes the server is authoritative for on the partial root.
     * When set, only listed attribute names are copied from the server response.
     * All other live attributes are preserved.
     */
    ownedAttrs?: string[];
    /** Attribute names that must never be modified on the live partial root. */
    preserveAttrs?: string[];
}

/** Handle for interacting with a server-side partial. */
export interface PartialHandle {
    /** The container element for this partial. */
    element: HTMLElement | null;

    /**
     * Reloads the partial from the server using the default merge mode.
     *
     * @param data - Optional query parameters to pass.
     */
    reload(data?: Record<string, string | number | boolean>): Promise<void>;

    /**
     * Reloads with fine-grained options.
     *
     * @param options - Reload options.
     */
    reloadWithOptions(options: PartialReloadOptions): Promise<void>;
}

/**
 * Parses a server fragment response and returns the partial root element.
 *
 * Fragment responses are wrapped in `<head>...</head><body><div id="app">...partial root...</div>...</body>`.
 * Drilling into `#app` matches the parser hand-off RemoteRenderer.render does,
 * so both reload paths see the same partial root rather than the page wrapper.
 * Falls back to `doc.body` for unwrapped fragments (test harnesses, alternative
 * servers) so the function still produces a usable element.
 *
 * @param html - Raw HTML string.
 * @returns The partial root element, or null when no root can be found.
 */
function parseFragment(html: string): HTMLElement | null {
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, 'text/html');
    const container = doc.querySelector('#app') ?? doc.body;
    return (container.firstElementChild ?? null) as HTMLElement | null;
}

/**
 * Resolves the {@link ReloadMode} for a reload call. Explicit `mode` wins;
 * otherwise the default `'merge'` is used.
 *
 * @param options - Reload options.
 * @returns The resolved mode.
 */
function resolveMode(options: PartialReloadOptions): ReloadMode {
    return options.mode ?? 'merge';
}

/**
 * Merges the live element's parent-scope tail onto the server-emitted self
 * scope for the `partial` attribute. Mirrors `handlePartialScopePreservation`
 * inside fragmentMorpher so that modes which skip root-morphing still keep
 * the CSS scope chain attached to nested partials.
 *
 * @param el - Live partial root.
 * @param sourceEl - Newly rendered partial root.
 */
function mergeRootScope(el: HTMLElement, sourceEl: HTMLElement): void {
    const existing = el.getAttribute(SCOPE_ATTR);
    const incoming = sourceEl.getAttribute(SCOPE_ATTR);
    if (!existing || !incoming) {
        return;
    }
    const parentScopes = existing.trim().split(/\s+/).slice(1);
    const selfScope = incoming.trim().split(/\s+/)[0];
    const merged = parentScopes.length === 0
        ? selfScope
        : [selfScope, ...parentScopes].join(' ');
    if (el.getAttribute(SCOPE_ATTR) !== merged) {
        el.setAttribute(SCOPE_ATTR, merged);
    }
}

/**
 * Copies attributes from a server-side source element onto the live element
 * using merge semantics: only attributes named in `owned` (defaulting to every
 * attribute the source actually has) are touched, anything else on the live
 * element is preserved. `preserve` attributes are skipped, framework-only
 * attributes are skipped, and the `partial` scope attribute is merged rather
 * than overwritten so nested partials keep their parent scope chain.
 *
 * @param el - Live partial root.
 * @param sourceEl - Newly rendered partial root.
 * @param owned - Optional explicit list of attribute names the server owns.
 * @param preserve - Optional list of attribute names that must not be touched.
 */
function syncRootAttrsFromSource(
    el: HTMLElement,
    sourceEl: HTMLElement,
    owned?: string[],
    preserve?: string[]
): void {
    const preserveSet = new Set(preserve ?? []);
    const candidateNames = owned ?? Array.from(sourceEl.attributes).map(a => a.name);
    let scopeHandled = false;

    for (const name of candidateNames) {
        if (preserveSet.has(name) || NEVER_SYNC_ATTRS.has(name)) {
            continue;
        }
        if (name === SCOPE_ATTR) {
            mergeRootScope(el, sourceEl);
            scopeHandled = true;
            continue;
        }
        const newValue = sourceEl.getAttribute(name);
        if (newValue === null) {
            if (el.hasAttribute(name)) {
                el.removeAttribute(name);
            }
            continue;
        }
        if (el.getAttribute(name) !== newValue) {
            el.setAttribute(name, newValue);
        }
    }

    if (!scopeHandled && !preserveSet.has(SCOPE_ATTR) && sourceEl.hasAttribute(SCOPE_ATTR)) {
        mergeRootScope(el, sourceEl);
    }
}

/**
 * Captures the current values of `preserve` attributes on the live element so
 * they can be restored after a full-replace morph clobbers them.
 *
 * @param el - Live partial root.
 * @param preserve - Attribute names to capture.
 * @returns Map of name to value, or null where the attribute was absent.
 */
function captureAttrs(el: HTMLElement, preserve?: string[]): Map<string, string | null> | null {
    if (!preserve || preserve.length === 0) {
        return null;
    }
    const snapshot = new Map<string, string | null>();
    for (const name of preserve) {
        snapshot.set(name, el.getAttribute(name));
    }
    return snapshot;
}

/**
 * Re-applies a captured attribute snapshot to the live element. Used to honour
 * `preserveAttrs` in `replace` mode, where the morph has just overwritten or
 * removed those attributes.
 *
 * @param el - Live partial root.
 * @param snapshot - Captured attribute snapshot from {@link captureAttrs}.
 */
function restoreAttrs(el: HTMLElement, snapshot: Map<string, string | null> | null): void {
    if (!snapshot) {
        return;
    }
    for (const [name, value] of snapshot) {
        if (value === null) {
            if (el.hasAttribute(name)) {
                el.removeAttribute(name);
            }
        } else if (el.getAttribute(name) !== value) {
            el.setAttribute(name, value);
        }
    }
}

/**
 * Applies the configured mode to merge the server response into the live DOM.
 * The live partial root element identity is preserved across every mode; only
 * its attributes and/or children change.
 *
 * @param el - Live partial root.
 * @param sourceEl - Newly rendered partial root.
 * @param mode - Reconciliation mode.
 * @param ownedAttrs - Optional explicit list of attributes the server owns on the root.
 * @param preserveAttrs - Optional list of attributes to leave untouched on the root.
 */
function applyMorph(
    el: HTMLElement,
    sourceEl: HTMLElement,
    mode: ReloadMode,
    ownedAttrs?: string[],
    preserveAttrs?: string[]
): void {
    const morphOptions = {
        getNodeKey,
        preservePartialScopes: true
    };

    switch (mode) {
        case 'merge':
            syncRootAttrsFromSource(el, sourceEl, ownedAttrs, preserveAttrs);
            fragmentMorpher(el, sourceEl, {...morphOptions, childrenOnly: true});
            return;

        case 'replace': {
            const snapshot = captureAttrs(el, preserveAttrs);
            fragmentMorpher(el, sourceEl, morphOptions);
            restoreAttrs(el, snapshot);
            return;
        }

        case 'children-only':
            fragmentMorpher(el, sourceEl, {...morphOptions, childrenOnly: true});
            return;

        case 'attrs-only':
            syncRootAttrsFromSource(el, sourceEl, ownedAttrs, preserveAttrs);
            return;

        default:
            assertExhaustive(mode);
    }
}

/**
 * Compile-time exhaustiveness guard. If a new {@link ReloadMode} value is
 * added without a matching case in {@link applyMorph}, TypeScript fails to
 * narrow the parameter to `never` and the call site stops type-checking.
 *
 * @param _ - The unreachable value that proves all cases are covered.
 */
function assertExhaustive(_: never): never {
    throw new Error(`Unhandled ReloadMode: ${String(_)}`);
}

/**
 * Performs the partial reload with the given options.
 *
 * @param el - Partial container element.
 * @param name - Partial name.
 * @param options - Reload options.
 */
async function performReload(el: HTMLElement, name: string, options: PartialReloadOptions): Promise<void> {
    const baseSrc = el.getAttribute('partial_src');
    if (!baseSrc) {
        throw new Error(`Partial "${name}" has no partial_src attribute. Is the partial's template marked as public?`);
    }
    let effectiveData = options.data;
    if (!effectiveData) {
        const partialProps = el.getAttribute('partial_props');
        if (partialProps) {
            effectiveData = Object.fromEntries(new URLSearchParams(partialProps));
        }
    }

    const params = new URLSearchParams(effectiveData as Record<string, string> | undefined);
    params.set('_f', 'true');
    const url = `${baseSrc}?${params.toString()}`;

    const mode = resolveMode(options);

    applyLoadingIndicator(el);

    try {
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`Failed to reload partial: ${response.status}`);
        }

        const html = await response.text();
        const sourceEl = parseFragment(html);

        if (!sourceEl) {
            console.warn(`[pk] partial "${name}" received empty or invalid response`);
            return;
        }

        applyMorph(el, sourceEl, mode, options.ownedAttrs, options.preserveAttrs);

        if (effectiveData) {
            el.setAttribute('partial_props',
                new URLSearchParams(effectiveData as Record<string, string>).toString()
            );
        }

        notifyDOMUpdated(el);
    } catch (error) {
        console.error(`[pk] Failed to reload partial "${name}":`, {
            url,
            args: options.data,
            mode,
            error
        });
        throw error;
    } finally {
        removeLoadingIndicator(el);
    }
}

/**
 * Returns a handle for a server-side partial by name or element.
 *
 * When given a string, looks up the partial by its partial_name attribute.
 * When given an Element, uses it directly (it must have partial_src for reload).
 *
 * @param nameOrElement - The partial name (matches partial_name attribute) or a partial root element.
 * @returns A handle for reloading the partial.
 */
export function partial(nameOrElement: string | Element): PartialHandle {
    let el: HTMLElement | null;
    let name: string;

    if (typeof nameOrElement === 'string') {
        name = nameOrElement;
        el = document.querySelector(`[partial_name="${name}"]`) as HTMLElement | null;
    } else {
        el = nameOrElement as HTMLElement;
        name = el.getAttribute('partial_name') ?? el.getAttribute('partial') ?? 'unknown';
    }

    return {
        element: el,

        async reload(data?: Record<string, string | number | boolean>): Promise<void> {
            if (!el) {
                console.warn(`[pk] partial "${name}" not found`);
                return;
            }
            return performReload(el, name, {data});
        },

        async reloadWithOptions(options: PartialReloadOptions): Promise<void> {
            if (!el) {
                console.warn(`[pk] partial "${name}" not found`);
                return;
            }
            return performReload(el, name, options);
        }
    };
}
