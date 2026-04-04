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

/**
 * Refresh level for graduated partial refresh.
 *
 * - 0: children-only morph (default).
 * - 1: full morph preserving partial scopes (pk-refresh-root).
 * - 2: full morph with owned attributes only (pk-own-attrs).
 * - 3: full morph skipping attribute refresh (pk-no-refresh-attrs).
 */
type RefreshLevel = 0 | 1 | 2 | 3;

/** Refresh level for pk-no-refresh-attrs (skips attribute refresh). */
const REFRESH_LEVEL_NO_REFRESH_ATTRS = 3;

/** Refresh level for pk-own-attrs (only updates owned attributes). */
const REFRESH_LEVEL_OWN_ATTRS = 2;

/**
 * Detects the refresh level from element attributes.
 *
 * @param el - Element to inspect.
 * @returns The detected refresh level.
 */
function detectRefreshLevel(el: HTMLElement): RefreshLevel {
    if (el.hasAttribute('pk-no-refresh-attrs')) {
        return REFRESH_LEVEL_NO_REFRESH_ATTRS;
    }
    if (el.hasAttribute('pk-own-attrs')) {
        return REFRESH_LEVEL_OWN_ATTRS;
    }
    if (el.hasAttribute('pk-refresh-root')) {
        return 1;
    }
    return 0;
}

/**
 * Returns owned attributes from the pk-own-attrs attribute.
 *
 * @param el - Element to read from.
 * @returns Array of owned attribute names, or undefined if not set.
 */
export function getOwnedAttributes(el: HTMLElement): string[] | undefined {
    const attr = el.getAttribute('pk-own-attrs');
    if (!attr) {
        return undefined;
    }
    return attr.split(',').map(s => s.trim()).filter(s => s.length > 0);
}

/**
 * Parses an HTML string into an element for morphing.
 *
 * @param html - Raw HTML string.
 * @returns The first child element, or null if parsing fails.
 */
function parseHTML(html: string): HTMLElement | null {
    const parser = new DOMParser();
    const doc = parser.parseFromString(html, 'text/html');
    return doc.body.firstElementChild as HTMLElement | null;
}

/** Options for partial reload with fine-grained control. */
export interface PartialReloadOptions {
    /** Query parameters to pass to the server. */
    data?: Record<string, string | number | boolean>;
    /** Overrides the detected refresh level. */
    level?: RefreshLevel;
    /** Overrides owned attributes for Level 2 (comma-separated). */
    ownedAttrs?: string[];
}

/** Handle for interacting with a server-side partial. */
export interface PartialHandle {
    /** The container element for this partial. */
    element: HTMLElement | null;

    /**
     * Reloads the partial from the server.
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
 * Applies the appropriate morph based on refresh level.
 *
 * @param el - Current element in the DOM.
 * @param newContent - New element to morph into.
 * @param level - Refresh level to apply.
 * @param ownedAttrs - Optional list of owned attributes for Level 2.
 */
function applyRefresh(el: HTMLElement, newContent: HTMLElement, level: RefreshLevel, ownedAttrs?: string[]): void {
    switch (level) {
        case 0:
            fragmentMorpher(el, newContent, {childrenOnly: true});
            break;

        case 1:
            fragmentMorpher(el, newContent, {
                childrenOnly: false,
                preservePartialScopes: true
            });
            break;

        case REFRESH_LEVEL_OWN_ATTRS:
            fragmentMorpher(el, newContent, {
                childrenOnly: false,
                preservePartialScopes: true,
                ownedAttributes: ownedAttrs
            });
            break;

        case REFRESH_LEVEL_NO_REFRESH_ATTRS:
            fragmentMorpher(el, newContent, {
                childrenOnly: false,
                preservePartialScopes: true
            });
            break;
    }
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

    const level = options.level ?? detectRefreshLevel(el);

    applyLoadingIndicator(el);

    try {
        const response = await fetch(url);
        if (!response.ok) {
            throw new Error(`Failed to reload partial: ${response.status}`);
        }

        const html = await response.text();
        const newContent = parseHTML(html);

        if (!newContent) {
            console.warn(`[pk] partial "${name}" received empty or invalid response`);
            return;
        }

        const ownedAttrs = options.ownedAttrs ?? getOwnedAttributes(el);

        applyRefresh(el, newContent, level, ownedAttrs);

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
            level,
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
