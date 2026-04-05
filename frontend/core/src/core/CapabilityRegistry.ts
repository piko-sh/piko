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

/** Callback invoked when a capability becomes available. */
type CapabilityCallback = (impl: unknown) => void;

/** Registered capability implementations keyed by name. */
const capabilities = new Map<string, unknown>();

/** Pending callbacks for capabilities that have not yet registered. */
const pendingCallbacks = new Map<string, CapabilityCallback[]>();

/**
 * Registers a capability implementation by name.
 *
 * Stores the implementation and drains any pending callbacks that were
 * queued before the capability loaded.
 *
 * @param name - The capability name (e.g. 'navigation', 'actions').
 * @param impl - The capability implementation object.
 */
export function _registerCapability(name: string, impl: unknown): void {
    capabilities.set(name, impl);

    const callbacks = pendingCallbacks.get(name);
    if (callbacks) {
        pendingCallbacks.delete(name);
        for (const cb of callbacks) {
            cb(impl);
        }
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
 * If the capability is already registered, the callback fires
 * synchronously. Otherwise it is queued until the capability loads.
 *
 * @param name - The capability name.
 * @param callback - Invoked with the capability implementation.
 */
export function _onCapabilityReady(name: string, callback: CapabilityCallback): void {
    const existing = capabilities.get(name);
    if (existing !== undefined) {
        callback(existing);
        return;
    }

    const queue = pendingCallbacks.get(name);
    if (queue) {
        queue.push(callback);
    } else {
        pendingCallbacks.set(name, [callback]);
    }
}

/**
 * Clears all registered capabilities and pending callbacks.
 *
 * Intended for test cleanup to prevent state leaking between test files.
 */
export function _clearCapabilities(): void {
    capabilities.clear();
    pendingCallbacks.clear();
}
