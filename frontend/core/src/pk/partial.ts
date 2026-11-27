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
import {getNodeKey, transformRelativeKeys} from '@/pk/getNodeKey';
import {runtimeVersionGate} from '@/core/RuntimeVersion';
import {reportDiagnostic} from '@/services/DiagnosticReporter';

/**
 * Controls which parts of a partial root a reload updates.
 *
 * - `merge` overwrites attributes present in the response and preserves live-only
 *   attributes. It merges the partial scope chain and morphs children.
 * - `replace` replaces root attributes and morphs children.
 * - `children-only` morphs children and preserves root attributes.
 * - `attrs-only` merges root attributes and preserves children.
 */
export type ReloadMode = 'merge' | 'replace' | 'children-only' | 'attrs-only';

/** Lists framework attributes to keep during root attribute merges. */
const NEVER_SYNC_ATTRS = new Set(['pk-ev-bound', 'pk-sync-bound']);

/** Names the attribute that stores the partial scope before its parent scopes. */
const SCOPE_ATTR = 'partial';

/** Configures a partial reload. */
export interface PartialReloadOptions {
    /** Query parameters to pass to the server. */
    data?: Record<string, string | number | boolean>;
    /** Reconciliation mode for the partial root. Defaults to `'merge'`. */
    mode?: ReloadMode;
    /** Limits which attributes to sync in merge and attrs-only modes. */
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
 * Finds the partial root in a fragment response.
 *
 * @param html - The response body.
 * @returns The first child element of #app if present, otherwise of the body.
 * The result is null if that container has no child element.
 */
function parseFragment(html: string): HTMLElement | null {
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, 'text/html');
    const container = doc.querySelector('#app') ?? doc.body;
    return (container.firstElementChild ?? null) as HTMLElement | null;
}

/**
 * Chooses the requested reconciliation mode.
 *
 * @param options - Reload options that can specify a mode.
 * @returns The requested mode or merge.
 */
function resolveMode(options: PartialReloadOptions): ReloadMode {
    return options.mode ?? 'merge';
}

/**
 * Replaces the live partial scope and keeps its parent scopes.
 *
 * @param el - The live partial root.
 * @param sourceEl - The element that supplies the new partial scope.
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
 * Updates server-owned attributes and merges the partial scope.
 *
 * @param el - The live partial root.
 * @param sourceEl - The element that supplies the new attributes and scope.
 * @param owned - Attributes to sync, defaulting to those on sourceEl.
 * @param preserve - Additional attributes to keep unchanged.
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
 * Saves live attributes for restoration after a replace morph.
 *
 * @param el - The element whose attributes to save.
 * @param preserve - Attributes to save.
 * @returns A map with null entries for missing attributes, or null for an empty list.
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
 * Restores saved attributes, including their absence.
 *
 * @param el - The live partial root.
 * @param snapshot - The saved values.
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
 * Updates the live partial in place using the selected reload mode.
 *
 * @param el - The live partial root.
 * @param sourceEl - The element that supplies the new attributes and children.
 * @param mode - What to reconcile.
 * @param ownedAttrs - Attributes to update in merge and attrs-only modes.
 * @param preserveAttrs - Root attributes to keep unchanged.
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
 * Throws for an unhandled reload mode.
 *
 * @param _ - An unreachable mode in the type system.
 */
function assertExhaustive(_: never): never {
    throw new Error(`Unhandled ReloadMode: ${String(_)}`);
}

/** Tracks the latest pending reload of each partial. */
const inFlightReloads = new WeakMap<HTMLElement, AbortController>();

/**
 * Tracks a new partial reload and aborts its predecessor.
 *
 * @param el - The live partial root.
 * @returns The controller for the new reload.
 */
function beginReload(el: HTMLElement): AbortController {
    inFlightReloads.get(el)?.abort(new DOMException('Superseded by a newer reload', 'AbortError'));
    const controller = new AbortController();
    inFlightReloads.set(el, controller);
    return controller;
}

/**
 * Checks whether a reload still owns the partial.
 *
 * @param el - The live partial root.
 * @param controller - The controller that identifies the reload.
 * @returns Whether the reload is still the latest pending one.
 */
function isLatestReload(el: HTMLElement, controller: AbortController): boolean {
    return inFlightReloads.get(el) === controller;
}

/**
 * Fetches and reconciles the server fragment for a partial. A newer reload of the same
 * element aborts the pending reload.
 *
 * @param el - The live partial root.
 * @param name - The partial name to use in errors.
 * @param options - The request data and reconciliation settings.
 * @returns A promise that resolves after the reload, including empty or superseded
 * responses, or rejects if the request fails or partial_src is missing.
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

    const controller = beginReload(el);
    applyLoadingIndicator(el);

    try {
        const response = await fetch(url, {signal: controller.signal});
        runtimeVersionGate.observe(response.headers);
        if (!response.ok) {
            throw new Error(`Failed to reload partial: ${response.status}`);
        }

        const html = await response.text();
        if (controller.signal.aborted) {
            return;
        }
        const sourceEl = parseFragment(html);

        if (!sourceEl) {
            console.error(`[pk] partial "${name}" received empty or invalid response`);
            return;
        }

        transformRelativeKeys(sourceEl, el);
        applyMorph(el, sourceEl, mode, options.ownedAttrs, options.preserveAttrs);

        if (effectiveData) {
            el.setAttribute('partial_props',
                new URLSearchParams(effectiveData as Record<string, string>).toString()
            );
        }

        notifyDOMUpdated(el);
    } catch (error) {
        if (controller.signal.aborted) {
            return;
        }
        console.error(`[pk] Failed to reload partial "${name}":`, {
            url,
            args: options.data,
            mode,
            error
        });
        throw error;
    } finally {
        if (isLatestReload(el, controller)) {
            inFlightReloads.delete(el);
            removeLoadingIndicator(el);
        }
    }
}

/**
 * Warns in development and sends a diagnostic event for a partial that is not
 * in the page.
 *
 * @param name - The partial name that was looked up.
 * @param source - The API that looked it up.
 */
function reportMissingPartial(name: string, source: string): void {
    console.warn(`[pk] partial "${name}" not found`);
    reportDiagnostic({kind: 'partial-not-found', name, source});
}

/**
 * Finds a server-side partial and creates a reload handle.
 *
 * @param nameOrElement - A partial_name or its root element.
 * @returns A handle with the element if found and methods to reload it.
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
                reportMissingPartial(name, 'piko.partial().reload');
                return;
            }
            return performReload(el, name, {data});
        },

        async reloadWithOptions(options: PartialReloadOptions): Promise<void> {
            if (!el) {
                reportMissingPartial(name, 'piko.partial().reloadWithOptions');
                return;
            }
            return performReload(el, name, options);
        }
    };
}
