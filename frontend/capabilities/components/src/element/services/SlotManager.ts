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

import type {SlotChangeCallback} from "../types";

/**
 * Manages slot element queries and change listeners.
 */
export interface SlotManager {
    /** Gets elements assigned to a slot by name. */
    getSlottedElements(slotName?: string): Element[];

    /** Attaches a listener to be notified when slot content changes. Invokes the callback immediately with initial content. */
    attachSlotListener(slotName: string, callback: SlotChangeCallback): void;

    /** Returns whether a slot has any content assigned. */
    hasSlotContent(slotName?: string): boolean;

    /** Replays any slot listeners that were queued before the shadow root was available. */
    flushPendingListeners(): void;
}

/**
 * Options for creating a SlotManager.
 */
export interface SlotManagerOptions {
    /** Function to get the shadow root container. */
    getShadowRoot: () => ShadowRoot | null;

    /** Tag name for warning messages. */
    tagName: string;
}

/**
 * Builds the CSS selector string for a slot element.
 *
 * @param slotName - The name of the slot, or an empty string for the default slot.
 * @returns The CSS selector targeting the named or default slot.
 */
function buildSlotSelector(slotName: string): string {
    return slotName ? `slot[name="${slotName}"]` : `slot:not([name])`;
}

/**
 * Creates a SlotManager for managing slot element queries and listeners.
 *
 * @param options - Configuration options including shadow root access.
 * @returns A new SlotManager instance.
 */
export function createSlotManager(options: SlotManagerOptions): SlotManager {
    const {getShadowRoot, tagName} = options;
    const pendingListeners: Array<{slotName: string; callback: SlotChangeCallback}> = [];

    function doAttach(shadowRoot: ShadowRoot, slotName: string, callback: SlotChangeCallback): void {
        const selector = buildSlotSelector(slotName);
        const slotElement = shadowRoot.querySelector(selector) as HTMLSlotElement | null;

        if (!slotElement) {
            console.warn(
                `PPElement: Slot "${selector}" not found in ${tagName} for attaching listener.`
            );
            return;
        }

        const handleSlotChange = (): void => {
            callback(slotElement.assignedElements({flatten: true}));
        };

        slotElement.addEventListener("slotchange", handleSlotChange);
        handleSlotChange();
    }

    return {
        getSlottedElements(slotName = ""): Element[] {
            const shadowRoot = getShadowRoot();
            if (!shadowRoot) {
                return [];
            }

            const selector = buildSlotSelector(slotName);
            const slotElement = shadowRoot.querySelector(selector) as HTMLSlotElement | null;
            return slotElement?.assignedElements({flatten: true}) ?? [];
        },

        attachSlotListener(slotName: string, callback: SlotChangeCallback): void {
            const shadowRoot = getShadowRoot();
            if (!shadowRoot) {
                pendingListeners.push({slotName, callback});
                return;
            }

            doAttach(shadowRoot, slotName, callback);
        },

        flushPendingListeners(): void {
            if (pendingListeners.length === 0) {
                return;
            }

            const shadowRoot = getShadowRoot();
            if (!shadowRoot) {
                return;
            }

            const listeners = pendingListeners.splice(0);
            for (const {slotName, callback} of listeners) {
                doAttach(shadowRoot, slotName, callback);
            }
        },

        hasSlotContent(slotName = ""): boolean {
            return this.getSlottedElements(slotName).length > 0;
        },
    };
}
