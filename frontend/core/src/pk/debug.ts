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

/** Debug information about a partial element's lifecycle state. */
export interface PartialDebugInfo {
    /** Whether the partial has lifecycle state registered. */
    exists: boolean;
    /** The partial's name from data-partial-name attribute. */
    partialName: string | null;
    /** The partial's ID from partial or data-partial attribute. */
    partialId: string | null;
    /** Whether the partial is currently connected (in DOM). */
    isConnected: boolean;
    /** Whether onConnected has fired for this partial instance. */
    connectedOnce: boolean;
    /** Names of registered lifecycle callbacks. */
    registeredCallbacks: string[];
    /** Total count of registered cleanup functions. */
    cleanupCount: number;
}

/** Debug inspection API for E2E testing. */
export interface PikoDebugAPI {
    /**
     * Returns full debug information for a partial element.
     *
     * @param element - The partial's root element.
     * @returns Debug info object.
     */
    getPartialInfo(element: Element): PartialDebugInfo;

    /**
     * Checks if a partial element is currently connected.
     *
     * @param element - The partial's root element.
     * @returns True if the partial is connected.
     */
    isConnected(element: Element): boolean;

    /**
     * Returns the count of registered cleanup functions for a partial.
     *
     * Includes both element-scoped and lifecycle cleanups.
     *
     * @param element - The partial's root element.
     * @returns Number of registered cleanup functions.
     */
    getCleanupCount(element: Element): number;

    /**
     * Returns names of registered lifecycle callbacks for a partial.
     *
     * @param element - The partial's root element.
     * @returns Array of callback names ('onConnected', 'onDisconnected', etc.).
     */
    getRegisteredCallbacks(element: Element): string[];

    /**
     * Returns all currently connected partial elements in the DOM.
     *
     * @returns Array of partial elements that are currently connected.
     */
    getAllConnectedPartials(): Element[];

    /**
     * Checks if the debug API is available.
     *
     * @returns True if the debug API is available.
     */
    isAvailable(): boolean;
}

declare global {
    interface Window {
        /** Debug inspection API for Piko partials, available in E2E mode. */
        __pikoDebug?: PikoDebugAPI;
    }
}
