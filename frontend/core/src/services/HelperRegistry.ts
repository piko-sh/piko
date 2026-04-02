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

/** Helper function type that can be synchronous or asynchronous. */
export type PPHelper = (element: HTMLElement, event: Event, ...args: string[]) => void | Promise<void>;

/** Manages helper functions that can be called from templates. */
export interface HelperRegistry {
    /**
     * Registers a helper function by name. Warns if a helper with the same name already exists.
     * @param name - The name to register the helper under.
     * @param helper - The helper function to register.
     */
    register(name: string, helper: PPHelper): void;

    /**
     * Gets a helper function by name.
     * @param name - The name of the helper to retrieve.
     * @returns The helper function, or undefined if not found.
     */
    get(name: string): PPHelper | undefined;

    /**
     * Executes a helper function by name, awaiting if asynchronous.
     * @param name - The name of the helper to execute.
     * @param element - The DOM element context.
     * @param event - The triggering event.
     * @param args - The string arguments to pass to the helper.
     */
    execute(name: string, element: HTMLElement, event: Event, args: string[]): Promise<void>;

    /**
     * Checks whether a helper exists by name.
     * @param name - The name of the helper to check.
     * @returns True if the helper is registered.
     */
    has(name: string): boolean;
}

/**
 * Creates a HelperRegistry for managing template helper functions.
 * @returns A new HelperRegistry instance.
 */
export function createHelperRegistry(): HelperRegistry {
    const registry = new Map<string, PPHelper>();

    return {
        register(name, helper) {
            if (registry.has(name)) {
                console.warn(`HelperRegistry: Overwriting already registered helper "${name}".`);
            }
            registry.set(name, helper);
        },

        get(name) {
            return registry.get(name);
        },

        async execute(name, element, event, args) {
            const helper = registry.get(name);
            if (helper) {
                await helper(element, event, ...args);
            } else {
                console.warn(`HelperRegistry: Unknown helper "${name}"`);
            }
        },

        has(name) {
            return registry.has(name);
        }
    };
}
