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

import type { PikoDebugAPI, PartialDebugInfo } from './debug';

/** A function that performs teardown when a partial is disconnected or navigated away from. */
type CleanupFn = () => void;

/** A callback with no arguments and no return value. */
type VoidCallback = () => void;

/** A callback invoked when a partial's content is updated, optionally receiving context about the change. */
type UpdatedCallback = (context?: unknown) => void;

/**
 * Lifecycle callbacks that can be registered for a PK partial instance.
 *
 * Mirrors the PKC lifecycle API for consistency.
 */
export interface PKLifecycleCallbacks {
    /** Called once when the partial is first connected to the DOM. */
    onConnected?: VoidCallback;
    /** Called when the partial is removed from the DOM. */
    onDisconnected?: VoidCallback;
    /** Called immediately before the partial is re-rendered. */
    onBeforeRender?: VoidCallback;
    /** Called immediately after the partial finishes re-rendering. */
    onAfterRender?: VoidCallback;
    /** Called when the partial's content is updated, with optional context. */
    onUpdated?: UpdatedCallback;
}

/** Internal lifecycle state that stores arrays of callbacks for additive registration. */
interface PartialLifecycleArrayCallbacks {
    /** Callbacks to run once when the partial first connects. */
    onConnected: VoidCallback[];
    /** Callbacks to run when the partial disconnects. */
    onDisconnected: VoidCallback[];
    /** Callbacks to run before re-render. */
    onBeforeRender: VoidCallback[];
    /** Callbacks to run after re-render. */
    onAfterRender: VoidCallback[];
    /** Callbacks to run when content is updated. */
    onUpdated: UpdatedCallback[];
}

/** State for a partial instance's lifecycle management. */
interface PartialLifecycleState {
    /** The registered lifecycle callback arrays for this partial. */
    callbacks: PartialLifecycleArrayCallbacks;
    /** Whether onConnected has already fired for the current mount cycle. */
    connectedOnce: boolean;
    /** Teardown functions to run when the partial is disconnected. */
    cleanups: CleanupFn[];
}

/** Page-level cleanup functions to run on navigation. */
const pageCleanups: CleanupFn[] = [];

/** Element-scoped cleanup functions, keyed by element. */
const elementCleanups = new WeakMap<Element, CleanupFn[]>();

/**
 * Stores lifecycle state per partial element.
 * Uses WeakMap so lifecycle state is GC'd when element is removed.
 */
const partialLifecycleState = new WeakMap<Element, PartialLifecycleState>();

/**
 * Tracks which partials have been connected (for onConnected once-only behaviour).
 */
const connectedPartials = new WeakSet<Element>();

/**
 * Returns the lifecycle state for a scope element, creating it if needed.
 *
 * @param scope - The partial's root element.
 * @returns The lifecycle state for the scope.
 */
function _getOrCreateState(scope: Element): PartialLifecycleState {
    let state = partialLifecycleState.get(scope);
    if (!state) {
        state = {
            callbacks: {
                onConnected: [],
                onDisconnected: [],
                onBeforeRender: [],
                onAfterRender: [],
                onUpdated: [],
            },
            connectedOnce: false,
            cleanups: [],
        };
        partialLifecycleState.set(scope, state);
    }
    return state;
}

/**
 * Registers lifecycle callbacks for a partial instance.
 *
 * Called by generated PK JavaScript when lifecycle functions are detected.
 * Kept for backward compatibility with already-compiled PK JS modules.
 *
 * @param scope - The partial's root element (element with [partial] attribute).
 * @param callbacks - Lifecycle callback functions.
 */
export function _registerLifecycle(scope: Element, callbacks: PKLifecycleCallbacks): void {
    const state = _getOrCreateState(scope);

    if (callbacks.onConnected) {
        state.callbacks.onConnected.push(callbacks.onConnected);
    }
    if (callbacks.onDisconnected) {
        state.callbacks.onDisconnected.push(callbacks.onDisconnected);
    }
    if (callbacks.onBeforeRender) {
        state.callbacks.onBeforeRender.push(callbacks.onBeforeRender);
    }
    if (callbacks.onAfterRender) {
        state.callbacks.onAfterRender.push(callbacks.onAfterRender);
    }
    if (callbacks.onUpdated) {
        state.callbacks.onUpdated.push(callbacks.onUpdated);
    }

    if (scope.isConnected && !state.connectedOnce) {
        _executeConnected(scope);
    }
}

/**
 * Adds a single lifecycle callback for a partial instance.
 *
 * Used by the `pk` context object for additive lifecycle registration.
 * Multiple calls to the same hook register multiple callbacks that all
 * fire in order.
 *
 * @param scope - The partial's root element.
 * @param hookName - The lifecycle hook name.
 * @param callback - The callback to register.
 */
export function _addLifecycleCallback(
    scope: Element,
    hookName: keyof PartialLifecycleArrayCallbacks,
    callback: VoidCallback | UpdatedCallback,
): void {
    const state = _getOrCreateState(scope);
    (state.callbacks[hookName] as Array<VoidCallback | UpdatedCallback>).push(callback);

    if (hookName === 'onConnected' && scope.isConnected && !state.connectedOnce) {
        _executeConnected(scope);
    }
}

/**
 * Executes the onConnected lifecycle for a partial element.
 *
 * Only fires once per element lifecycle (reset on disconnection).
 *
 * @param scope - The partial's root element.
 */
export function _executeConnected(scope: Element): void {
    const state = partialLifecycleState.get(scope);
    if (!state || state.connectedOnce) {
        return;
    }

    state.connectedOnce = true;
    connectedPartials.add(scope);

    for (const callback of state.callbacks.onConnected) {
        try {
            callback();
        } catch (error) {
            console.error('[pk] Error in onConnected:', error);
        }
    }
}

/**
 * Executes the onDisconnected lifecycle for a partial element.
 *
 * Resets connectedOnce so onConnected can fire again if re-mounted.
 *
 * @param scope - The partial's root element.
 */
export function _executeDisconnected(scope: Element): void {
    const state = partialLifecycleState.get(scope);
    if (!state) {
        return;
    }

    for (const callback of state.callbacks.onDisconnected) {
        try {
            callback();
        } catch (error) {
            console.error('[pk] Error in onDisconnected:', error);
        }
    }

    state.connectedOnce = false;
    connectedPartials.delete(scope);

    for (const cleanup of state.cleanups) {
        try {
            cleanup();
        } catch (error) {
            console.error('[pk] Error in partial cleanup:', error);
        }
    }
    state.cleanups.length = 0;
}

/**
 * Executes the onBeforeRender lifecycle for a partial element.
 *
 * Called before a partial reload/re-render.
 *
 * @param scope - The partial's root element.
 */
export function _executeBeforeRender(scope: Element): void {
    const state = partialLifecycleState.get(scope);
    if (!state || state.callbacks.onBeforeRender.length === 0) {
        return;
    }

    for (const callback of state.callbacks.onBeforeRender) {
        try {
            callback();
        } catch (error) {
            console.error('[pk] Error in onBeforeRender:', error);
        }
    }
}

/**
 * Executes the onAfterRender lifecycle for a partial element.
 *
 * Called after a partial reload/re-render completes.
 *
 * @param scope - The partial's root element.
 */
export function _executeAfterRender(scope: Element): void {
    const state = partialLifecycleState.get(scope);
    if (!state || state.callbacks.onAfterRender.length === 0) {
        return;
    }

    for (const callback of state.callbacks.onAfterRender) {
        try {
            callback();
        } catch (error) {
            console.error('[pk] Error in onAfterRender:', error);
        }
    }
}

/**
 * Executes the onUpdated lifecycle for a partial element.
 *
 * Called after partial content is updated.
 *
 * @param scope - The partial's root element.
 * @param context - Optional context about what was updated.
 */
export function _executeUpdated(scope: Element, context?: unknown): void {
    const state = partialLifecycleState.get(scope);
    if (!state || state.callbacks.onUpdated.length === 0) {
        return;
    }

    for (const callback of state.callbacks.onUpdated) {
        try {
            callback(context);
        } catch (error) {
            console.error('[pk] Error in onUpdated:', error);
        }
    }
}

/**
 * Executes onConnected for all partial elements within a container.
 *
 * Called after DOM patching to ensure new partials receive lifecycle events.
 *
 * @param container - The container element or document to scan.
 */
export function _executeConnectedForPartials(container: Element | Document): void {
    const partials = container.querySelectorAll('[partial]');
    for (const partial of partials) {
        if (partialLifecycleState.has(partial) && !connectedPartials.has(partial)) {
            _executeConnected(partial);
        }
    }
}

/**
 * Checks whether a partial has lifecycle callbacks registered.
 *
 * @param scope - The partial's root element.
 * @returns True if lifecycle callbacks are registered.
 */
export function _hasLifecycleCallbacks(scope: Element): boolean {
    return partialLifecycleState.has(scope);
}

/**
 * Registers a cleanup function to be called when navigating away from the page
 * or when an element is removed from the DOM.
 *
 * @param cleanupFunction - Cleanup function to call.
 * @param scope - Optional element to scope the cleanup to. If provided, the cleanup
 *                runs when this element is removed from the DOM. If omitted, the
 *                cleanup runs on page navigation.
 */
export function onCleanup(cleanupFunction: CleanupFn, scope?: Element): void {
    if (scope) {
        let scopeCleanups = elementCleanups.get(scope);
        if (!scopeCleanups) {
            scopeCleanups = [];
            elementCleanups.set(scope, scopeCleanups);
        }
        scopeCleanups.push(cleanupFunction);
    } else {
        pageCleanups.push(cleanupFunction);
    }
}

/**
 * Runs all page-level cleanup functions.
 *
 * Called by PPFramework on navigation.
 */
export function _runPageCleanup(): void {
    for (const cleanupFunction of pageCleanups) {
        try {
            cleanupFunction();
        } catch (error) {
            console.error('[pk] Error in page cleanup:', error);
        }
    }
    pageCleanups.length = 0;
}

/** Active MutationObserver for element removal cleanup, or null if not initialised. */
let cleanupObserver: MutationObserver | null = null;

/**
 * Initialises the MutationObserver that watches for element removal
 * and triggers associated cleanup functions.
 */
export function _initCleanupObserver(): void {
    if (cleanupObserver) {
        return;
    }

    cleanupObserver = new MutationObserver((mutations) => {
        for (const mutation of mutations) {
            for (const node of mutation.removedNodes) {
                if (typeof Element !== 'undefined' && node instanceof Element) {
                    runElementCleanups(node);
                }
            }
        }
    });

    cleanupObserver.observe(document.body, {
        childList: true,
        subtree: true
    });
}

/**
 * Disconnects the cleanup observer. Used for testing.
 */
export function _disconnectCleanupObserver(): void {
    if (cleanupObserver) {
        cleanupObserver.disconnect();
        cleanupObserver = null;
    }
}

/**
 * Runs cleanup functions and lifecycle events for an element and all its descendants.
 *
 * Triggers onDisconnected for any partial elements, then executes registered
 * element-scoped cleanups.
 *
 * @param element - The element being removed.
 */
function runElementCleanups(element: Element): void {
    executeDisconnectedForRemovedPartials(element);

    const cleanups = elementCleanups.get(element);
    if (cleanups) {
        for (const cleanupFunction of cleanups) {
            try {
                cleanupFunction();
            } catch (error) {
                console.error('[pk] Error in element cleanup:', error);
            }
        }
        elementCleanups.delete(element);
    }

    for (const child of element.querySelectorAll('*')) {
        const childCleanups = elementCleanups.get(child);
        if (!childCleanups) {
            continue;
        }
        for (const cleanupFunction of childCleanups) {
            try {
                cleanupFunction();
            } catch (error) {
                console.error('[pk] Error in element cleanup:', error);
            }
        }
        elementCleanups.delete(child);
    }
}

/**
 * Fires onDisconnected for any partial elements within the removed subtree,
 * including the element itself if it carries the `[partial]` attribute.
 *
 * @param element - The root element of the removed subtree.
 */
function executeDisconnectedForRemovedPartials(element: Element): void {
    if (element.hasAttribute('partial')) {
        _executeDisconnected(element);
    }

    const partials = element.querySelectorAll('[partial]');
    for (const partial of partials) {
        _executeDisconnected(partial);
    }
}

/**
 * Returns the names of all non-undefined callbacks registered on the given lifecycle callbacks object.
 *
 * @param callbacks - The lifecycle callbacks to inspect.
 * @returns Array of callback names.
 */
function getCallbackNames(callbacks: PartialLifecycleArrayCallbacks): string[] {
    const names: string[] = [];
    if (callbacks.onConnected.length > 0) {
        names.push('onConnected');
    }
    if (callbacks.onDisconnected.length > 0) {
        names.push('onDisconnected');
    }
    if (callbacks.onBeforeRender.length > 0) {
        names.push('onBeforeRender');
    }
    if (callbacks.onAfterRender.length > 0) {
        names.push('onAfterRender');
    }
    if (callbacks.onUpdated.length > 0) {
        names.push('onUpdated');
    }
    return names;
}

/**
 * Creates the debug inspection API for E2E testing.
 *
 * Provides read-only access to internal lifecycle state.
 *
 * @returns The debug API object.
 */
function createDebugAPI(): PikoDebugAPI {
    return {
        getPartialInfo(element: Element): PartialDebugInfo {
            const state = partialLifecycleState.get(element);
            const cleanups = elementCleanups.get(element);

            return {
                exists: state !== undefined,
                partialName: element.getAttribute('partial_name') ?? element.getAttribute('data-partial-name'),
                partialId: element.getAttribute('partial') ?? element.getAttribute('data-partial'),
                isConnected: connectedPartials.has(element),
                connectedOnce: state?.connectedOnce ?? false,
                registeredCallbacks: state ? getCallbackNames(state.callbacks) : [],
                cleanupCount: (state?.cleanups.length ?? 0) + (cleanups?.length ?? 0),
            };
        },

        isConnected(element: Element): boolean {
            return connectedPartials.has(element);
        },

        getCleanupCount(element: Element): number {
            const state = partialLifecycleState.get(element);
            const cleanups = elementCleanups.get(element);
            return (state?.cleanups.length ?? 0) + (cleanups?.length ?? 0);
        },

        getRegisteredCallbacks(element: Element): string[] {
            const state = partialLifecycleState.get(element);
            return state ? getCallbackNames(state.callbacks) : [];
        },

        getAllConnectedPartials(): Element[] {
            const partials = document.querySelectorAll('[partial], [data-partial]');
            return Array.from(partials).filter(el => connectedPartials.has(el));
        },

        isAvailable(): boolean {
            return true;
        },
    };
}

if (typeof window !== 'undefined') {
    window.__pikoDebug = createDebugAPI();
}
