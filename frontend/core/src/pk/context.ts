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

import {createRefs} from './refs';
import {_addLifecycleCallback, _createPageLifecycle, onCleanup} from './lifecycle';

/** A callback with no arguments and no return value. */
type VoidCallback = () => void;

/** A callback invoked when a partial's content is updated, optionally receiving context about the change. */
type UpdatedCallback = (context?: unknown) => void;

/**
 * File-scoped context object for PK pages and partials.
 *
 * Provides access to refs, lifecycle hooks, and cleanup registration
 * scoped to the current partial instance or page visit.
 *
 * A partial's lifecycle follows its root element. A page script's lifecycle follows the
 * page visit. It connects when the page is shown and disconnects when SPA navigation
 * leaves the page, and each visit gets a fresh context.
 */
export interface PKContext {
    /** Proxy for accessing elements by p-ref attribute within the partial or page scope. */
    readonly refs: Record<string, HTMLElement | null>;
    /**
     * Creates a scoped refs proxy for a container element.
     *
     * @param scope - The container element to scope queries to.
     * @returns A proxy that returns elements by their p-ref name.
     */
    createRefs(scope?: Element): Record<string, HTMLElement | null>;
    /**
     * Registers a callback to run when the partial connects to the DOM, or when the page
     * visit starts. Each callback runs once per connection. A callback registered after
     * the connection runs straight away. A page context is connected for its whole visit.
     *
     * @param callback - The callback to invoke on connection.
     */
    onConnected(callback: VoidCallback): void;
    /**
     * Registers a callback to run when the partial disconnects from the DOM, or when SPA
     * navigation leaves the page.
     *
     * @param callback - The callback to invoke on disconnection.
     */
    onDisconnected(callback: VoidCallback): void;
    /**
     * Registers a callback to run immediately before the partial re-renders. On a page
     * context, it runs before a partial reload re-renders any element inside the page
     * that carries a `partial` attribute. That element can be a partial's root or any
     * element a partial rendered.
     *
     * @param callback - The callback to invoke before rendering.
     */
    onBeforeRender(callback: VoidCallback): void;
    /**
     * Registers a callback to run immediately after the partial finishes re-rendering. On
     * a page context, it runs after a partial reload re-renders any element inside the
     * page that carries a `partial` attribute, as onBeforeRender does.
     *
     * @param callback - The callback to invoke after rendering.
     */
    onAfterRender(callback: VoidCallback): void;
    /**
     * Registers a callback to run when the partial's content is updated. On a page
     * context, it runs after a partial reload re-renders any element inside the page that
     * carries a `partial` attribute, as onAfterRender does, with the same context.
     *
     * @param callback - The callback to invoke on update, optionally receiving context.
     */
    onUpdated(callback: UpdatedCallback): void;
    /**
     * Registers a cleanup function to run when the partial is disconnected, or when SPA
     * navigation leaves the page.
     *
     * @param cleanupFunction - The cleanup function to call.
     */
    onCleanup(cleanupFunction: VoidCallback): void;
}

/**
 * Creates a file-scoped context object for a PK partial instance.
 *
 * The returned object provides refs, lifecycle registration, and cleanup
 * scoped to the given DOM element. Lifecycle callbacks are additive -
 * multiple calls to the same hook register multiple callbacks that all
 * fire in order.
 *
 * @param scope - The partial's root element.
 * @returns The context object for use as `pk` in PK script blocks.
 */
export function _createPKContext(scope: Element): PKContext {
    return {
        refs: createRefs(scope),
        createRefs: (root?: Element) => createRefs(root ?? scope),
        onConnected: (callback: VoidCallback) => _addLifecycleCallback(scope, 'onConnected', callback),
        onDisconnected: (callback: VoidCallback) => _addLifecycleCallback(scope, 'onDisconnected', callback),
        onBeforeRender: (callback: VoidCallback) => _addLifecycleCallback(scope, 'onBeforeRender', callback),
        onAfterRender: (callback: VoidCallback) => _addLifecycleCallback(scope, 'onAfterRender', callback),
        onUpdated: (callback: UpdatedCallback) => _addLifecycleCallback(scope, 'onUpdated', callback),
        onCleanup: (cleanupFunction: VoidCallback) => onCleanup(cleanupFunction, scope),
    };
}

/**
 * Creates the context of a page script instance for a page visit.
 *
 * Refs resolve against the scope element as they do for partials. The lifecycle is the
 * instance's own and ends with the page visit, even though the scope element, the page
 * root, stays in place across SPA navigation.
 *
 * Code at the top level of a page script runs once per visit. Work that must be undone
 * when the page is left, such as `piko.hooks.on(...)` or a listener on `window`, belongs
 * in `pk.onConnected`, with its removal in `pk.onCleanup`.
 *
 * A context created for a visit that is not the one shown now has already ended. Its
 * onConnected and other callbacks never run, and a cleanup added to it runs straight
 * away. Generated page scripts pass the visit the runtime loaded them for, so code from
 * an old visit, such as a timer that calls the module's `__reinit__`, cannot connect on
 * the page shown now. A context created with no visit joins the visit shown now.
 *
 * @param scope - The page root the page script is scoped to.
 * @param visit - The page visit the context is for, from `_currentPageVisit()`. Leave it
 *     out to use the visit shown now.
 * @returns The context object for use as `pk` in the page's script block.
 */
export function _createPagePKContext(scope: Element, visit?: object): PKContext {
    const lifecycle = _createPageLifecycle(scope, visit);
    return {
        refs: createRefs(scope),
        createRefs: (root?: Element) => createRefs(root ?? scope),
        onConnected: (callback: VoidCallback) => lifecycle.add('onConnected', callback),
        onDisconnected: (callback: VoidCallback) => lifecycle.add('onDisconnected', callback),
        onBeforeRender: (callback: VoidCallback) => lifecycle.add('onBeforeRender', callback),
        onAfterRender: (callback: VoidCallback) => lifecycle.add('onAfterRender', callback),
        onUpdated: (callback: UpdatedCallback) => lifecycle.add('onUpdated', callback),
        onCleanup: (cleanupFunction: VoidCallback) => lifecycle.addCleanup(cleanupFunction),
    };
}
