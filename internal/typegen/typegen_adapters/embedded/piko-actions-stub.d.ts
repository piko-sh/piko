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
 * Piko Action TypeScript Definitions (Stub)
 *
 * These are placeholder definitions to demonstrate the TypeScript
 * generation pipeline. Real action definitions will be generated
 * from Go action handlers in the future.
 *
 * @packageDocumentation
 * @since 1.0.0
 */

/**
 * Result of an action execution.
 *
 * @template T - The response data type
 */
interface ActionResult<T> {
    /** Whether the action succeeded */
    ok: boolean;
    /** The response data (present if ok is true) */
    data?: T;
    /** The error (present if ok is false) */
    error?: ActionErrorResult;
}

/**
 * Error result from an action.
 */
interface ActionErrorResult {
    /** Error code */
    code: string;
    /** Human-readable error message */
    message: string;
    /** Additional error details */
    details?: Record<string, unknown>;
}

/**
 * Builder for configuring and executing actions.
 *
 * @template T - The response data type
 */
interface ActionBuilder<T> {
    /**
     * Registers a callback for successful action completion.
     *
     * @param callback - Function called with the response data
     * @returns This builder for chaining
     *
     * @example
     * ```typescript
     * action.echo("Hello").onSuccess((data) => {
     *     console.log("Response:", data.message);
     * });
     * ```
     */
    onSuccess(callback: (data: T) => void): ActionBuilder<T>;

    /**
     * Registers a callback for action errors.
     *
     * @param callback - Function called with the error
     * @returns This builder for chaining
     *
     * @example
     * ```typescript
     * action.echo("Hello").onError((err) => {
     *     console.error("Action failed:", err.message);
     * });
     * ```
     */
    onError(callback: (error: ActionErrorResult) => void): ActionBuilder<T>;

    /**
     * Registers a callback that runs after success or error.
     *
     * @param callback - Function called when action completes
     * @returns This builder for chaining
     *
     * @example
     * ```typescript
     * action.echo("Hello").onFinally(() => {
     *     hideLoadingSpinner();
     * });
     * ```
     */
    onFinally(callback: () => void): ActionBuilder<T>;

    /**
     * Executes the action and returns a promise.
     *
     * @returns Promise resolving to the action result
     *
     * @example
     * ```typescript
     * const result = await action.echo("Hello").execute();
     * if (result.ok) {
     *     console.log(result.data.message);
     * }
     * ```
     */
    execute(): Promise<ActionResult<T>>;
}

/**
 * Stub action namespace for testing the type generation pipeline.
 *
 * These are placeholder actions that demonstrate how action types
 * will be exposed to TypeScript. In production, these will be
 * generated from your Go action handlers.
 *
 * @example
 * ```typescript
 * // Call an action with type-safe parameters and response
 * action.echo("Hello, World!")
 *     .onSuccess((data) => console.log(data.message))
 *     .onError((err) => console.error(err.message));
 *
 * // Or use async/await
 * const result = await action.getCurrentUser().execute();
 * if (result.ok) {
 *     console.log("User:", result.data.name);
 * }
 * ```
 */
declare namespace action {
    /**
     * Echo a message back (stub action for testing).
     *
     * This is a demonstration action that echoes the provided message.
     *
     * @param message - The message to echo
     * @returns ActionBuilder that resolves with the echoed message
     *
     * @example
     * ```typescript
     * action.echo("Hello, Piko!")
     *     .onSuccess((res) => console.log(res.message));
     * ```
     *
     * @since 1.0.0
     */
    function echo(message: string): ActionBuilder<{ message: string }>;

    /**
     * Get current user info (stub action for testing).
     *
     * This is a demonstration action that returns mock user information.
     *
     * @returns ActionBuilder that resolves with user information
     *
     * @example
     * ```typescript
     * action.getCurrentUser()
     *     .onSuccess((user) => {
     *         console.log("Logged in as:", user.name);
     *         console.log("Email:", user.email);
     *     });
     * ```
     *
     * @since 1.0.0
     */
    function getCurrentUser(): ActionBuilder<StubUser>;

    /**
     * Create a new item (stub action for testing).
     *
     * This is a demonstration action that creates a mock item.
     *
     * @param input - Item creation data
     * @returns ActionBuilder that resolves with the created item
     *
     * @example
     * ```typescript
     * action.createItem({
     *     name: "New Task",
     *     description: "A sample task"
     * }).onSuccess((item) => {
     *     console.log("Created item:", item.id);
     * });
     * ```
     *
     * @since 1.0.0
     */
    function createItem(input: StubCreateItemInput): ActionBuilder<StubItem>;
}

/**
 * Stub user type for testing.
 *
 * @since 1.0.0
 */
interface StubUser {
    /** Unique user identifier */
    id: number;
    /** User's display name */
    name: string;
    /** User's email address */
    email: string;
}

/**
 * Input for creating a stub item.
 *
 * @since 1.0.0
 */
interface StubCreateItemInput {
    /** Item name (required) */
    name: string;
    /** Optional item description */
    description?: string;
}

/**
 * Stub item type for testing.
 *
 * @since 1.0.0
 */
interface StubItem {
    /** Unique item identifier */
    id: number;
    /** Item name */
    name: string;
    /** Item description (null if not provided) */
    description: string | null;
    /** ISO 8601 timestamp of creation */
    createdAt: string;
}
