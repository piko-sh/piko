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

import type {StateContext} from "../types";

/**
 * Manages component state and tracks property changes.
 */
export interface StateManager {
    /** Gets the current reactive state object. */
    getState(): Record<string, unknown> | undefined;

    /** Sets partial state by merging with existing state. */
    setState(partialState: Record<string, unknown>): void;

    /** Gets the full context object including state and callbacks. */
    getContext(): StateContext | undefined;

    /** Sets the context during initialisation. */
    setContext(context: StateContext): void;

    /** Returns whether state has been initialised. */
    hasState(): boolean;

    /** Gets the set of changed property names since last clear. */
    getChangedProps(): Set<string>;

    /** Clears and returns a copy of the changed properties set. */
    clearChangedProps(): Set<string>;

    /** Records a property change for tracking. */
    recordChange(propertyName: string): void;
}

/**
 * Options for creating a StateManager.
 */
export interface StateManagerOptions {
    /** Tag name for warning messages. */
    tagName: string;
}

/**
 * Creates a StateManager for managing component state and context.
 *
 * @param options - Configuration options including tag name.
 * @returns A new StateManager instance.
 */
export function createStateManager(options: StateManagerOptions): StateManager {
    let context: StateContext | undefined;
    const changedPropsSet = new Set<string>();

    return {
        getState(): Record<string, unknown> | undefined {
            return context?.state;
        },

        setState(partialState: Record<string, unknown>): void {
            if (!context?.state) {
                console.warn(`PPElement ${options.tagName}: setState called before state was initialised.`);
                return;
            }
            Object.assign(context.state, partialState);
        },

        getContext(): StateContext | undefined {
            return context;
        },

        setContext(ctx: StateContext): void {
            context = ctx;
        },

        hasState(): boolean {
            return context?.state !== undefined;
        },

        getChangedProps(): Set<string> {
            return changedPropsSet;
        },

        clearChangedProps(): Set<string> {
            const copy = new Set(changedPropsSet);
            changedPropsSet.clear();
            return copy;
        },

        recordChange(propertyName: string): void {
            changedPropsSet.add(propertyName);
        },
    };
}
