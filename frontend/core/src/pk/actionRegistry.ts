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

/** HTTP status code for validation errors. */
const HTTP_STATUS_UNPROCESSABLE = 422;

/** HTTP status code for unauthorised access. */
const HTTP_STATUS_UNAUTHORIZED = 401;

/** HTTP status code for forbidden access. */
const HTTP_STATUS_FORBIDDEN = 403;

/** HTTP methods supported by action descriptors. */
export type ActionMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

/** Retry backoff strategy. */
export type RetryBackoff = 'linear' | 'exponential';

/** Retry configuration for action requests. */
export interface RetryConfig {
    /** Number of retry attempts. */
    attempts: number;
    /** Backoff strategy (default: exponential). */
    backoff?: RetryBackoff;
}

/** Retry configuration for SSE streams with auto-reconnection. */
export interface RetryStreamConfig {
    /** Maximum number of reconnection attempts (default: Infinity). */
    maxReconnects?: number;
    /** Base delay between reconnection attempts in milliseconds (default: 3000). */
    baseDelay?: number;
    /** Maximum delay cap in milliseconds (default: 30000). */
    maxDelay?: number;
    /** Whether to use exponential backoff (default: true). */
    exponentialBackoff?: boolean;
    /** Callback invoked when a reconnection attempt starts. */
    onReconnect?: (attempt: number) => void;
    /** Callback invoked when all reconnection attempts are exhausted. */
    onReconnectFailed?: () => void;
}

/** Error returned when an action fails. */
export interface ActionError {
    /** HTTP status code (0 for network errors). */
    status: number;
    /** Error message. */
    message: string;
    /** Field-level validation errors. */
    validationErrors?: Record<string, string[]>;
    /** Raw response data. */
    data?: unknown;
    /** Server-response helpers attached to error responses. */
    _helpers?: Array<{ name: string; args?: unknown[] }>;
}

/**
 * Creates an ActionError with computed convenience properties.
 *
 * @param status - HTTP status code (0 for network errors).
 * @param message - Human-readable error message.
 * @param validationErrors - Optional field-level validation errors.
 * @param data - Optional raw response data.
 * @param helpers - Optional server-response helpers.
 * @returns An ActionError with isNetworkError, isValidationError, and isAuthError getters.
 */
export function createActionError(
    status: number,
    message: string,
    validationErrors?: Record<string, string[]>,
    data?: unknown,
    helpers?: Array<{ name: string; args?: unknown[] }>
): ActionError & { isNetworkError: boolean; isValidationError: boolean; isAuthError: boolean } {
    return {
        status,
        message,
        validationErrors,
        data,
        _helpers: helpers,
        get isNetworkError(): boolean {
            return this.status === 0;
        },
        get isValidationError(): boolean {
            return this.status === HTTP_STATUS_UNPROCESSABLE && this.validationErrors !== undefined;
        },
        get isAuthError(): boolean {
            return this.status === HTTP_STATUS_UNAUTHORIZED || this.status === HTTP_STATUS_FORBIDDEN;
        }
    };
}

/** Describes a server action with lifecycle callbacks. */
export interface ActionDescriptor<T = unknown> {
    /** Server action name to call. */
    action: string;
    /** Arguments to pass to the action. */
    args?: unknown[];
    /** HTTP method (default: POST). */
    method?: ActionMethod;
    /** Runs immediately before the server call. */
    optimistic?: () => void;
    /** Runs after successful response; can return another action to chain. */
    onSuccess?: (response: T) => void | ActionDescriptor;
    /** Runs if the server returns an error. */
    onError?: (error: ActionError) => void;
    /** Runs always (success or error). */
    onComplete?: () => void;
    /** Loading state target: true = element, string = selector, HTMLElement = specific element. */
    loading?: string | HTMLElement | boolean;
    /** Debounce in milliseconds before sending. */
    debounce?: number;
    /** Retry configuration. */
    retry?: RetryConfig;
    /** Timeout in milliseconds. */
    timeout?: number;
    /** External AbortSignal for request cancellation. */
    signal?: AbortSignal;
    /** Whether to suppress automatic helper execution. */
    shouldSuppressHelpers?: boolean;
    /** Progress callback for SSE streaming. */
    onProgress?: (data: unknown, eventType: string) => void;
    /** Retry configuration for SSE streams. */
    retryStream?: RetryStreamConfig;
}

/**
 * Checks if a value is an ActionDescriptor using duck typing.
 *
 * @param value - The value to check.
 * @returns True if the value has an 'action' property of type string.
 */
export function isActionDescriptor(value: unknown): value is ActionDescriptor {
    return value !== null &&
        typeof value === 'object' &&
        typeof (value as ActionDescriptor).action === 'string';
}

/** Action factory function type for the registry. */
type ActionFactory = (...args: unknown[]) => ActionDescriptor;

/** Global registry of action functions keyed by Go action name. */
const actionFunctionRegistry = new Map<string, ActionFactory>();

/**
 * Registers an action function in the global registry.
 *
 * Called by generated actions.gen.js to make action wrapper functions
 * discoverable by the DOMBinder for template-bound action calls.
 *
 * @param name - The Go action name (e.g. "email.Contact").
 * @param actionFactory - The wrapper function that returns an ActionDescriptor.
 */
export function registerActionFunction(name: string, actionFactory: ActionFactory): void {
    actionFunctionRegistry.set(name, actionFactory);
}

/**
 * Looks up an action function by its Go action name.
 *
 * @param name - The Go action name (e.g. "email.Contact").
 * @returns The wrapper function, or undefined if not registered.
 */
export function getActionFunction(name: string): ActionFactory | undefined {
    return actionFunctionRegistry.get(name);
}
