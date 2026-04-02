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
import {_addLifecycleCallback, onCleanup} from './lifecycle';

/** A callback with no arguments and no return value. */
type VoidCallback = () => void;

/** A callback invoked when a partial's content is updated, optionally receiving context about the change. */
type UpdatedCallback = (context?: unknown) => void;

/**
 * File-scoped context object for PK pages and partials.
 *
 * Provides access to refs, lifecycle hooks, and cleanup registration
 * scoped to the current partial instance.
 */
export interface PKContext {
    /** Proxy for accessing elements by p-ref attribute within the partial scope. */
    readonly refs: Record<string, HTMLElement | null>;
    /**
     * Creates a scoped refs proxy for a container element.
     *
     * @param scope - The container element to scope queries to.
     * @returns A proxy that returns elements by their p-ref name.
     */
    createRefs(scope?: Element): Record<string, HTMLElement | null>;
    /**
     * Registers a callback to run once when the partial connects to the DOM.
     *
     * @param callback - The callback to invoke on connection.
     */
    onConnected(callback: VoidCallback): void;
    /**
     * Registers a callback to run when the partial disconnects from the DOM.
     *
     * @param callback - The callback to invoke on disconnection.
     */
    onDisconnected(callback: VoidCallback): void;
    /**
     * Registers a callback to run immediately before the partial re-renders.
     *
     * @param callback - The callback to invoke before rendering.
     */
    onBeforeRender(callback: VoidCallback): void;
    /**
     * Registers a callback to run immediately after the partial finishes re-rendering.
     *
     * @param callback - The callback to invoke after rendering.
     */
    onAfterRender(callback: VoidCallback): void;
    /**
     * Registers a callback to run when the partial's content is updated.
     *
     * @param callback - The callback to invoke on update, optionally receiving context.
     */
    onUpdated(callback: UpdatedCallback): void;
    /**
     * Registers a cleanup function to run when the partial is disconnected.
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
        createRefs: (s?: Element) => createRefs(s ?? scope),
        onConnected: (callback: VoidCallback) => _addLifecycleCallback(scope, 'onConnected', callback),
        onDisconnected: (callback: VoidCallback) => _addLifecycleCallback(scope, 'onDisconnected', callback),
        onBeforeRender: (callback: VoidCallback) => _addLifecycleCallback(scope, 'onBeforeRender', callback),
        onAfterRender: (callback: VoidCallback) => _addLifecycleCallback(scope, 'onAfterRender', callback),
        onUpdated: (callback: UpdatedCallback) => _addLifecycleCallback(scope, 'onUpdated', callback),
        onCleanup: (cleanupFunction: VoidCallback) => onCleanup(cleanupFunction, scope),
    };
}
