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
 * Slim action module retained in the core shim.
 *
 * Re-exports registry functions from actionRegistry.ts and provides a
 * minimal ActionBuilder that satisfies the actions.gen.js import contract.
 * The full ActionBuilder with fluent API lives in the actions capability.
 */

export {
    registerActionFunction,
    getActionFunction,
    isActionDescriptor,
    createActionError,
    type ActionDescriptor,
    type ActionError,
    type ActionMethod,
    type RetryConfig,
    type RetryBackoff,
    type RetryStreamConfig,
} from './actionRegistry';

/**
 * Minimal ActionBuilder that implements the ActionDescriptor interface.
 *
 * Provides just enough to satisfy actions.gen.js which creates instances
 * via createActionBuilder(). The full fluent API with setOnSuccess, setRetry,
 * etc. is provided by the actions capability.
 */
export class ActionBuilder {
    /** Server action name. */
    action: string;
    /** Arguments to pass to the action. */
    args?: unknown[];

    /**
     * Creates a new ActionBuilder.
     *
     * @param actionName - Server action name.
     * @param actionArgs - Arguments for the action.
     */
    constructor(actionName: string, actionArgs?: unknown[]) {
        this.action = actionName;
        this.args = actionArgs;
    }
}

/**
 * Creates an ActionBuilder with variadic arguments.
 *
 * @param name - Server action name.
 * @param args - Arguments to pass to the action.
 * @returns An ActionBuilder instance.
 */
export function action(name: string, ...args: unknown[]): ActionBuilder {
    return new ActionBuilder(name, args);
}

/**
 * Creates an ActionBuilder for use by generated actions.gen.js.
 *
 * @param name - Server action name.
 * @param args - Arguments object to pass to the action.
 * @returns An ActionBuilder instance.
 */
export function createActionBuilder(name: string, args: Record<string, unknown>): ActionBuilder {
    if (typeof args.toObject === 'function') {
        args = (args as unknown as { toObject(): Record<string, unknown> }).toObject();
    }
    return new ActionBuilder(name, [args]);
}
