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

import type {UpdatedCallback, VoidCallback} from "../types";

/**
 * Manages lifecycle callback registration and execution for custom elements.
 *
 * All lifecycle hooks use additive registration - multiple calls to the same
 * hook register multiple callbacks that all fire in order.
 */
export interface LifecycleManager {
    /** Registers a callback to run when the element connects to the DOM. */
    onConnected(callback: VoidCallback): void;

    /** Registers a callback to run when the element disconnects from the DOM. */
    onDisconnected(callback: VoidCallback): void;

    /** Registers a callback to run before rendering. */
    onBeforeRender(callback: VoidCallback): void;

    /** Registers a callback to run after rendering. */
    onAfterRender(callback: VoidCallback): void;

    /** Registers a callback to run after state updates. */
    onUpdated(callback: UpdatedCallback): void;

    /** Registers a cleanup function to run when the component disconnects. */
    onCleanup(callback: VoidCallback): void;

    /** Executes all registered connected callbacks. Fires once per lifecycle. */
    executeConnected(): void;

    /** Executes all registered disconnected callbacks. */
    executeDisconnected(): void;

    /** Executes all registered before-render callbacks. */
    executeBeforeRender(): void;

    /** Executes all registered after-render callbacks. */
    executeAfterRender(): void;

    /** Executes all registered updated callbacks with the set of changed properties. */
    executeUpdated(changedProperties: Set<string>): void;

    /** Executes all registered cleanup callbacks, then clears the array. */
    executeCleanups(): void;

    /** Resets connected state for reconnection handling. */
    resetConnectedState(): void;

    /** Returns whether the initial connection has occurred. */
    hasConnectedOnce(): boolean;
}

/**
 * Creates a LifecycleManager for managing lifecycle callback registration and execution.
 *
 * @returns A new LifecycleManager instance.
 */
export function createLifecycleManager(): LifecycleManager {
    const onConnectedCallbacks: VoidCallback[] = [];
    const onDisconnectedCallbacks: VoidCallback[] = [];
    const onBeforeRenderCallbacks: VoidCallback[] = [];
    const onAfterRenderCallbacks: VoidCallback[] = [];
    const onUpdatedCallbacks: UpdatedCallback[] = [];
    const onCleanupCallbacks: VoidCallback[] = [];

    let connectedOnce = false;

    return {
        onConnected(callback: VoidCallback): void {
            onConnectedCallbacks.push(callback);
        },

        onDisconnected(callback: VoidCallback): void {
            onDisconnectedCallbacks.push(callback);
        },

        onBeforeRender(callback: VoidCallback): void {
            onBeforeRenderCallbacks.push(callback);
        },

        onAfterRender(callback: VoidCallback): void {
            onAfterRenderCallbacks.push(callback);
        },

        onUpdated(callback: UpdatedCallback): void {
            onUpdatedCallbacks.push(callback);
        },

        onCleanup(callback: VoidCallback): void {
            onCleanupCallbacks.push(callback);
        },

        executeConnected(): void {
            if (connectedOnce) {
                return;
            }
            connectedOnce = true;

            onConnectedCallbacks.forEach((cb) => cb());
        },

        executeDisconnected(): void {
            onDisconnectedCallbacks.forEach((cb) => cb());
        },

        executeBeforeRender(): void {
            onBeforeRenderCallbacks.forEach((cb) => cb());
        },

        executeAfterRender(): void {
            onAfterRenderCallbacks.forEach((cb) => cb());
        },

        executeUpdated(changedProperties: Set<string>): void {
            onUpdatedCallbacks.forEach((cb) => cb(changedProperties));
        },

        executeCleanups(): void {
            onCleanupCallbacks.forEach((cb) => cb());
            onCleanupCallbacks.length = 0;
        },

        resetConnectedState(): void {
            connectedOnce = false;
        },

        hasConnectedOnce(): boolean {
            return connectedOnce;
        },
    };
}
