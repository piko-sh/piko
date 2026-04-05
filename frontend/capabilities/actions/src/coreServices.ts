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

/**
 * Minimal interfaces for core services that the actions capability
 * depends on. Satisfied by real implementations from the core shim.
 */

/** Emits typed hook events for analytics and lifecycle tracking. */
export interface HookManagerLike {
    /** Emits an event with its payload. */
    emit(event: string, payload: unknown): void;
}

/** Tracks form dirty state. */
export interface FormStateManagerLike {
    /** Scans the DOM and begins tracking all forms. */
    scanAndTrackForms(): void;
    /** Marks a specific form as clean after successful submission. */
    markFormClean(form: HTMLFormElement | string): void;
}

/** Hook event constants used by actions for analytics. */
export const ActionsHookEvent = {
    ACTION_START: 'action:start',
    ACTION_COMPLETE: 'action:complete',
} as const;

/** Signature for registered helper functions. */
export type PPHelperLike = (
    element: HTMLElement,
    event: Event,
    ...args: string[]
) => void | Promise<void>;

/** Registry for looking up and executing helper functions. */
export interface HelperRegistryLike {
    /** Retrieves a helper by name. */
    get(name: string): PPHelperLike | undefined;
    /** Checks if a helper exists. */
    has(name: string): boolean;
    /** Executes a helper by name. */
    execute(name: string, element: HTMLElement, event: Event, args: string[]): Promise<void>;
}

/**
 * Services injected by the core shim when the actions capability loads.
 */
export interface ActionsCoreServices {
    /** Hook manager for emitting analytics events. */
    hookManager: HookManagerLike;
    /** Form state manager for dirty-state tracking. */
    formStateManager: FormStateManagerLike | null;
    /** Helper registry for executing server-response helpers. */
    helperRegistry: HelperRegistryLike;
}
