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

import {getOwnedAttributes} from '@/pk/partial';

/** Callbacks invoked by the SyncPartialManager for remote rendering operations. */
export interface SyncPartialCallbacks {
    /**
     * Performs a remote render of a partial.
     * @param options - The rendering options including source, form data, and patch configuration.
     */
    onRemoteRender: (options: {
        /** URL path for the partial source. */
        src: string;
        /** Form data to send with the request. */
        formData: Record<string, string[]> | undefined;
        /** Method for applying the patch. */
        patchMethod: 'morph';
        /** Whether to only update children. */
        childrenOnly: boolean;
        /** Whether to preserve parent CSS scopes. */
        preservePartialScopes?: boolean;
        /** Attributes that should be updated. */
        ownedAttributes?: string[];
        /** Selector for finding the element in the response. */
        querySelector: string;
        /** Element to patch with the response. */
        patchLocation: HTMLElement;
    }) => Promise<void>;
}

/** Manages synchronised partial updates via IntersectionObserver and form input events. */
export interface SyncPartialManager {
    /**
     * Parses and binds sync containers within a root element.
     * @param root - The root element to scan for sync containers.
     */
    bind(root: HTMLElement): void;
}

/** Debounce delay for visibility changes in milliseconds. */
const VISIBILITY_DEBOUNCE_MS = 150;

/** Debounce delay for input changes in milliseconds. */
const INPUT_DEBOUNCE_MS = 400;

/** Attribute marker indicating a sync container has been bound. */
const SYNC_BOUND_MARKER = 'pk-sync-bound';

/** HTML tags that trigger immediate sync updates on change. */
const SYNC_TRIGGER_TAGS = ['SELECT', 'INPUT', 'PP-SELECT', 'PP-CHECKBOX'];

/**
 * Extracts the primary (last) value from a space-separated attribute string.
 * Partial attributes can contain multiple values (outer to inner), and the last value
 * is the innermost/primary partial.
 * @param attrValue - The space-separated attribute string.
 * @returns The last value from the string, or an empty string if null.
 */
function extractPrimaryValue(attrValue: string | null): string {
    if (!attrValue) {
        return '';
    }
    const parts = attrValue.trim().split(/\s+/);
    return parts[parts.length - 1] || '';
}

/**
 * Gathers form data from a form element into a record of string arrays keyed by field name.
 * @param form - The form element to gather data from, or null.
 * @returns The gathered form data, or undefined if the form is null.
 */
function gatherFormData(form: HTMLFormElement | null): Record<string, string[]> | undefined {
    if (!form) {
        return undefined;
    }
    const formData = new FormData(form);
    const data: Record<string, string[]> = {};
    for (const key of new Set(formData.keys())) {
        data[key] = formData.getAll(key) as string[];
    }
    return data;
}

/**
 * Checks whether an element is currently visible within the viewport.
 * @param el - The element to check visibility for.
 * @returns True if the element is at least partially visible.
 */
function isElementVisible(el: HTMLElement): boolean {
    const rect = el.getBoundingClientRect();
    return (
        rect.top < window.innerHeight &&
        rect.bottom > 0 &&
        rect.left < window.innerWidth &&
        rect.right > 0 &&
        rect.width > 0 &&
        rect.height > 0
    );
}

/** Refresh level type for partial updates. */
type RefreshLevel = 0 | 1 | 2 | 3;

/** Refresh level constant for pk-no-refresh-attrs mode. */
const REFRESH_LEVEL_NO_REFRESH_ATTRS = 3;

/**
 * Detects the refresh level from element attributes.
 * Level 0 is the default (children only), level 1 is root refresh with scope preservation,
 * level 2 updates only listed attributes, and level 3 updates all attributes except listed ones.
 * @param el - The element to inspect.
 * @returns The detected refresh level.
 */
function detectRefreshLevel(el: HTMLElement): RefreshLevel {
    if (el.hasAttribute('pk-no-refresh-attrs')) {
        return REFRESH_LEVEL_NO_REFRESH_ATTRS;
    }
    if (el.hasAttribute('pk-own-attrs')) {
        return 2;
    }
    if (el.hasAttribute('pk-refresh-root')) {
        return 1;
    }
    return 0;
}

/** Tracks the internal state for a bound sync container. */
interface ContainerBinding {
    /** The container element being managed. */
    containerEl: HTMLElement;
    /** The partial source URL. */
    partialSrc: string;
    /** Callbacks for remote rendering. */
    callbacks: SyncPartialCallbacks;
    /** Debounce timers for each element. */
    debounceTimers: WeakMap<HTMLElement, number>;
}

/**
 * Creates a function that triggers a remote render for the bound container.
 * @param binding - The container binding state.
 * @returns An asynchronous function that sends the update to the server.
 */
function createUpdateServer(binding: ContainerBinding) {
    const {containerEl, partialSrc, callbacks} = binding;
    return async (formData: Record<string, unknown> | null = null) => {
        const form = containerEl.closest('form');
        const gatheredData = formData ?? gatherFormData(form);
        const level = detectRefreshLevel(containerEl);

        const childrenOnly = level === 0;
        const preservePartialScopes = level >= 1;
        const ownedAttributes = level === 2 ? getOwnedAttributes(containerEl) : undefined;

        await callbacks.onRemoteRender({
            src: partialSrc,
            formData: gatheredData as Record<string, string[]> | undefined,
            patchMethod: 'morph',
            childrenOnly,
            preservePartialScopes,
            ownedAttributes,
            querySelector: `[partial_src="${partialSrc}"]`,
            patchLocation: containerEl
        });
    };
}

/**
 * Sets up input, change, and refresh-partial event listeners on a sync container.
 * @param binding - The container binding state.
 */
function setupContainerEventListeners(binding: ContainerBinding): void {
    const {containerEl, debounceTimers} = binding;
    const updateServer = createUpdateServer(binding);

    containerEl.addEventListener('input', _event => {
        clearTimeout(debounceTimers.get(containerEl));
        debounceTimers.set(containerEl, setTimeout(() => void updateServer(), INPUT_DEBOUNCE_MS) as unknown as number);
    });

    containerEl.addEventListener('change', event => {
        const target = event.target as HTMLElement;
        if (!SYNC_TRIGGER_TAGS.includes(target.tagName)) {
            return;
        }
        if (target.tagName === 'INPUT' && (target as HTMLInputElement).type === 'text') {
            return;
        }
        clearTimeout(debounceTimers.get(containerEl));
        void updateServer();
    });

    containerEl.addEventListener('refresh-partial', event => {
        event.stopPropagation();
        const customEvent = event as CustomEvent<{ formData?: Record<string, unknown>; afterMorph?: () => void } | undefined>;
        void updateServer(customEvent.detail?.formData ?? null).then(() => {
            customEvent.detail?.afterMorph?.();
        }).catch((err: unknown) => {
            console.error('[SyncPartialManager] refresh-partial failed:', err);
        });
    });
}

/**
 * Creates a SyncPartialManager for handling synchronised partial updates.
 * Uses IntersectionObserver to detect visibility changes and debounces updates.
 * @param callbacks - The callbacks for remote rendering operations.
 * @returns A new SyncPartialManager instance.
 */
export function createSyncPartialManager(callbacks: SyncPartialCallbacks): SyncPartialManager {
    const debounceVisibleElements = new Set<HTMLElement>();
    let visibilityDebounceTimer: number;
    const visibilityState = new WeakMap<HTMLElement, 'visible' | 'hidden'>();

    /**
     * Dispatches refresh-partial events to all elements that became visible during the debounce window.
     */
    const processVisibleBatch = () => {
        debounceVisibleElements.forEach(el => el.dispatchEvent(new CustomEvent('refresh-partial', {bubbles: false})));
        debounceVisibleElements.clear();
    };

    const observer = new IntersectionObserver(
        entries => {
            for (const entry of entries) {
                const containerEl = entry.target as HTMLElement;
                const wasVisible = visibilityState.get(containerEl) === 'visible';
                if (entry.isIntersecting && !wasVisible) {
                    debounceVisibleElements.add(containerEl);
                }
                visibilityState.set(containerEl, entry.isIntersecting ? 'visible' : 'hidden');
            }
            if (debounceVisibleElements.size > 0) {
                clearTimeout(visibilityDebounceTimer);
                visibilityDebounceTimer = setTimeout(processVisibleBatch, VISIBILITY_DEBOUNCE_MS) as unknown as number;
            }
        },
        {root: null, threshold: 0.1}
    );

    return {
        bind(rootElement: HTMLElement) {
            const containers = rootElement.querySelectorAll<HTMLElement>(`[partial_mode="sync"]:not([${SYNC_BOUND_MARKER}])`);

            containers.forEach(containerEl => {
                const partialSrcAttr = containerEl.getAttribute('partial_src');
                const partialSrc = extractPrimaryValue(partialSrcAttr);
                if (!partialSrc) {
                    console.warn('SyncPartialManager: A sync container is missing its "partial_src" attribute.', containerEl);
                    return;
                }

                const binding: ContainerBinding = {containerEl, partialSrc, callbacks, debounceTimers: new WeakMap()};
                setupContainerEventListeners(binding);

                requestAnimationFrame(() => requestAnimationFrame(() => {
                    visibilityState.set(containerEl, isElementVisible(containerEl) ? 'visible' : 'hidden');
                    observer.observe(containerEl);
                }));

                containerEl.setAttribute(SYNC_BOUND_MARKER, 'true');
            });
        }
    };
}
