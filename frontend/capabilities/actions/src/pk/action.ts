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

import {callServerActionDirect, type DirectCallResponse} from '@/core/ActionExecutor';

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

/**
 * Retry configuration for SSE streams with auto-reconnection.
 *
 * When an SSE stream disconnects unexpectedly, the action builder
 * will automatically reconnect with configurable backoff. This enables
 * long-lived streams (notification feeds, live dashboards) to survive
 * transient network issues.
 */
export interface RetryStreamConfig {
    /** Maximum reconnection attempts (Infinity for long-lived streams). */
    maxReconnects: number;
    /** Backoff strategy (default: 'linear'). */
    backoff?: RetryBackoff;
    /** Base delay between reconnection attempts in ms (default: 3000). */
    baseDelay?: number;
    /** Maximum delay cap in ms (default: 30000). */
    maxDelay?: number;
    /** Called when SSE connection drops (before reconnect attempt). */
    onDisconnect?: () => void;
    /** Called when reconnection succeeds. */
    onReconnect?: (attemptNumber: number) => void;
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
 * Creates an ActionError instance with computed convenience properties for
 * checking network, validation, and authentication error categories.
 *
 * @param status - HTTP status code (0 for network errors).
 * @param message - Human-readable error message.
 * @param validationErrors - Optional field-level validation errors.
 * @param data - Optional raw response data.
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

    /** Timeout in milliseconds; the request is aborted after this duration. */
    timeout?: number;

    /** External AbortSignal for request cancellation. */
    signal?: AbortSignal;

    /** Whether to suppress automatic helper execution from server response. */
    shouldSuppressHelpers?: boolean;

    /** Progress callback for SSE streaming, called for each intermediate event. */
    onProgress?: (data: unknown, eventType: string) => void;

    /** Retry configuration for SSE streams, enables auto-reconnection on connection drop. */
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

/**
 * Fluent builder for ActionDescriptor.
 *
 * Implements the ActionDescriptor interface so it can be used directly without calling build().
 */
export class ActionBuilder<T = unknown> implements ActionDescriptor<T> {
    private readonly _action: string;
    private readonly _args: unknown[];
    private _method?: ActionMethod;
    private _optimistic?: () => void;
    private _onSuccess?: (response: T) => void | ActionDescriptor;
    private _onError?: (error: ActionError) => void;
    private _onComplete?: () => void;
    private _loading?: string | HTMLElement | boolean;
    private _debounce?: number;
    private _retry?: RetryConfig;
    private _timeout?: number;
    private _signal?: AbortSignal;
    private _suppressHelpers = false;
    private _onProgress?: (data: unknown, eventType: string) => void;
    private _retryStream?: RetryStreamConfig;

    /**
     * Creates a new ActionBuilder.
     *
     * @param actionName - Server action name.
     * @param args - Arguments to pass to the action.
     */
    constructor(actionName: string, args: unknown[]) {
        this._action = actionName;
        this._args = args;
    }

    /** Returns the server action name. */
    get action(): string {
        return this._action;
    }

    /** Returns the arguments to pass to the action. */
    get args(): unknown[] {
        return this._args;
    }

    /** Returns the configured HTTP method. */
    get method(): ActionMethod | undefined {
        return this._method;
    }

    /** Returns the optimistic update callback. */
    get optimistic(): (() => void) | undefined {
        return this._optimistic;
    }

    /** Returns the success callback. */
    get onSuccess(): ((response: T) => void | ActionDescriptor) | undefined {
        return this._onSuccess;
    }

    /** Returns the error callback. */
    get onError(): ((error: ActionError) => void) | undefined {
        return this._onError;
    }

    /** Returns the completion callback. */
    get onComplete(): (() => void) | undefined {
        return this._onComplete;
    }

    /** Returns the loading state target. */
    get loading(): string | HTMLElement | boolean | undefined {
        return this._loading;
    }

    /** Returns the debounce delay in milliseconds. */
    get debounce(): number | undefined {
        return this._debounce;
    }

    /** Returns the retry configuration. */
    get retry(): RetryConfig | undefined {
        return this._retry;
    }

    /** Returns the timeout in milliseconds. */
    get timeout(): number | undefined {
        return this._timeout;
    }

    /** Returns the external abort signal. */
    get signal(): AbortSignal | undefined {
        return this._signal;
    }

    /** Returns whether automatic helper execution is suppressed. */
    get shouldSuppressHelpers(): boolean {
        return this._suppressHelpers;
    }

    /** Returns the SSE progress callback. */
    get onProgress(): ((data: unknown, eventType: string) => void) | undefined {
        return this._onProgress;
    }

    /** Returns the SSE stream retry configuration. */
    get retryStream(): RetryStreamConfig | undefined {
        return this._retryStream;
    }

    /**
     * Sets the HTTP method.
     *
     * @param method - The HTTP method to use.
     * @returns This builder for chaining.
     */
    setMethod(method: ActionMethod): this {
        this._method = method;
        return this;
    }

    /**
     * Sets the optimistic update callback.
     *
     * Runs immediately before the server request.
     *
     * @param optimisticUpdate - Callback to run before the request.
     * @returns This builder for chaining.
     */
    setOptimistic(optimisticUpdate: () => void): this {
        this._optimistic = optimisticUpdate;
        return this;
    }

    /**
     * Sets the success callback.
     *
     * Can return another ActionDescriptor to chain actions.
     *
     * @param successHandler - Callback receiving the typed response.
     * @returns This builder for chaining.
     */
    setOnSuccess(successHandler: (response: T) => void | ActionDescriptor): this {
        this._onSuccess = successHandler;
        return this;
    }

    /**
     * Sets the error callback.
     *
     * Use this to roll back optimistic updates.
     *
     * @param errorHandler - Callback receiving the ActionError.
     * @returns This builder for chaining.
     */
    setOnError(errorHandler: (error: ActionError) => void): this {
        this._onError = errorHandler;
        return this;
    }

    /**
     * Sets the completion callback.
     *
     * Runs after success or error, always.
     *
     * @param completeHandler - Callback to run on completion.
     * @returns This builder for chaining.
     */
    setOnComplete(completeHandler: () => void): this {
        this._onComplete = completeHandler;
        return this;
    }

    /**
     * Sets the loading state target.
     *
     * - `true`: Apply to trigger element.
     * - `string`: CSS selector.
     * - `HTMLElement`: Specific element.
     *
     * @param target - The loading state target.
     * @returns This builder for chaining.
     */
    setLoading(target: string | HTMLElement | boolean): this {
        this._loading = target;
        return this;
    }

    /**
     * Sets the debounce delay.
     *
     * @param ms - Delay in milliseconds.
     * @returns This builder for chaining.
     */
    setDebounce(ms: number): this {
        this._debounce = ms;
        return this;
    }

    /**
     * Sets the retry configuration.
     *
     * @param attempts - Number of retry attempts.
     * @param backoff - Optional backoff strategy.
     * @returns This builder for chaining.
     */
    setRetry(attempts: number, backoff?: RetryBackoff): this {
        this._retry = {attempts, backoff};
        return this;
    }

    /**
     * Sets the timeout in milliseconds.
     *
     * The request is aborted after this duration.
     *
     * @param ms - Timeout duration.
     * @returns This builder for chaining.
     */
    setTimeout(ms: number): this {
        this._timeout = ms;
        return this;
    }

    /**
     * Sets the external abort signal for cancellation.
     *
     * @param signal - The AbortSignal to use.
     * @returns This builder for chaining.
     */
    setSignal(signal: AbortSignal): this {
        this._signal = signal;
        return this;
    }

    /**
     * Sets the progress callback for SSE streaming.
     *
     * When set, the action framework automatically uses SSE transport
     * (POST with Accept: text/event-stream). The callback fires for each
     * intermediate event; terminal events route to onSuccess/onError.
     *
     * @param progressHandler - Callback receiving event data and event type.
     * @returns This builder for chaining.
     */
    setOnProgress(progressHandler: (data: unknown, eventType: string) => void): this {
        this._onProgress = progressHandler;
        return this;
    }

    /**
     * Sets the retry configuration for SSE streams.
     *
     * When set alongside onProgress, the action builder automatically
     * reconnects the SSE stream on connection drops with configurable
     * backoff. Use maxReconnects: Infinity for long-lived streams.
     *
     * @param config - Retry stream configuration.
     * @returns This builder for chaining.
     */
    setRetryStream(config: RetryStreamConfig): this {
        this._retryStream = config;
        return this;
    }

    /**
     * Sets the HTTP method (alias for setMethod).
     *
     * @param method - The HTTP method to use.
     * @returns This builder for chaining.
     */
    withMethod(method: ActionMethod): this {
        return this.setMethod(method);
    }

    /**
     * Sets the optimistic update callback (alias for setOptimistic).
     *
     * @param optimisticUpdate - Callback to run before the request.
     * @returns This builder for chaining.
     */
    withOptimistic(optimisticUpdate: () => void): this {
        return this.setOptimistic(optimisticUpdate);
    }

    /**
     * Sets the success callback (alias for setOnSuccess).
     *
     * @param successHandler - Callback receiving the typed response.
     * @returns This builder for chaining.
     */
    withOnSuccess(successHandler: (response: T) => void | ActionDescriptor): this {
        return this.setOnSuccess(successHandler);
    }

    /**
     * Sets the error callback (alias for setOnError).
     *
     * @param errorHandler - Callback receiving the ActionError.
     * @returns This builder for chaining.
     */
    withOnError(errorHandler: (error: ActionError) => void): this {
        return this.setOnError(errorHandler);
    }

    /**
     * Sets the completion callback (alias for setOnComplete).
     *
     * @param completeHandler - Callback to run on completion.
     * @returns This builder for chaining.
     */
    withOnComplete(completeHandler: () => void): this {
        return this.setOnComplete(completeHandler);
    }

    /**
     * Sets the loading state target (alias for setLoading).
     *
     * @param target - The loading state target.
     * @returns This builder for chaining.
     */
    withLoading(target: string | HTMLElement | boolean): this {
        return this.setLoading(target);
    }

    /**
     * Sets the debounce delay (alias for setDebounce).
     *
     * @param ms - Delay in milliseconds.
     * @returns This builder for chaining.
     */
    withDebounce(ms: number): this {
        return this.setDebounce(ms);
    }

    /**
     * Sets the retry configuration (alias for setRetry).
     *
     * @param attempts - Number of retry attempts.
     * @param backoff - Optional backoff strategy.
     * @returns This builder for chaining.
     */
    withRetry(attempts: number, backoff?: RetryBackoff): this {
        return this.setRetry(attempts, backoff);
    }

    /**
     * Sets the timeout (alias for setTimeout).
     *
     * @param ms - Timeout duration in milliseconds.
     * @returns This builder for chaining.
     */
    withTimeout(ms: number): this {
        return this.setTimeout(ms);
    }

    /**
     * Sets the external abort signal (alias for setSignal).
     *
     * @param signal - The AbortSignal to use.
     * @returns This builder for chaining.
     */
    withSignal(signal: AbortSignal): this {
        return this.setSignal(signal);
    }

    /**
     * Sets the SSE progress callback (alias for setOnProgress).
     *
     * @param progressHandler - Callback receiving event data and event type.
     * @returns This builder for chaining.
     */
    withOnProgress(progressHandler: (data: unknown, eventType: string) => void): this {
        return this.setOnProgress(progressHandler);
    }

    /**
     * Sets the SSE stream retry configuration (alias for setRetryStream).
     *
     * @param config - Retry stream configuration.
     * @returns This builder for chaining.
     */
    withRetryStream(config: RetryStreamConfig): this {
        return this.setRetryStream(config);
    }

    /**
     * Suppress automatic execution of server-response helpers.
     *
     * When called, helpers returned by the server (e.g. redirect, resetForm)
     * will NOT be executed automatically. This is useful when you need to
     * process the response programmatically and don't want side effects.
     *
     * The helpers are still available on the DirectCallResponse for manual
     * inspection when using `.call()`.
     *
     * @returns This builder for chaining.
     */
    suppressHelpers(): this {
        this._suppressHelpers = true;
        return this;
    }

    /**
     * Builds the final ActionDescriptor.
     *
     * Note: The builder itself implements ActionDescriptor, so calling build()
     * is optional. You can use the builder directly.
     *
     * @returns The constructed ActionDescriptor.
     */
    build(): ActionDescriptor<T> {
        return {
            action: this._action,
            args: this._args,
            method: this._method,
            optimistic: this._optimistic,
            onSuccess: this._onSuccess,
            onError: this._onError,
            onComplete: this._onComplete,
            loading: this._loading,
            debounce: this._debounce,
            retry: this._retry,
            timeout: this._timeout,
            signal: this._signal,
            shouldSuppressHelpers: this._suppressHelpers || undefined,
            onProgress: this._onProgress,
            retryStream: this._retryStream
        };
    }

    /**
     * Execute the action directly and return the typed response.
     *
     * This is an imperative alternative to the callback pattern, useful in
     * component scripts where you need the response data directly.
     *
     * Helpers from the server response are processed automatically before
     * the promise resolves.
     *
     * @returns Promise resolving to the typed response data.
     */
    async call(): Promise<T> {
        const response: DirectCallResponse<T> = await callServerActionDirect<T>(
            this._action,
            this._args,
            this._method ?? 'POST',
            {
                timeout: this._timeout,
                signal: this._signal,
                suppressHelpers: this._suppressHelpers || undefined,
                onProgress: this._onProgress,
                retryStream: this._retryStream
            }
        );

        return response.data;
    }
}

/**
 * Creates an action builder for the given action name and arguments.
 *
 * @param name - Server action name.
 * @param args - Arguments to pass to the action.
 * @returns ActionBuilder with fluent API.
 */
export function action<T = unknown>(name: string, ...args: unknown[]): ActionBuilder<T> {
    return new ActionBuilder<T>(name, args);
}

/**
 * Creates an action builder for generated action functions.
 *
 * This is used by the auto-generated actions.gen.js file. It differs from `action()`
 * in that it accepts a single args object (named parameters) rather than variadic args.
 *
 * @param name - Server action name (e.g., 'media.Search').
 * @param args - Arguments object to pass to the action.
 * @returns ActionBuilder with fluent API.
 */
export function createActionBuilder<T = unknown>(name: string, args: Record<string, unknown>): ActionBuilder<T> {
    if (typeof (args as Record<string, unknown>).toObject === 'function') {
        args = (args as unknown as { toObject(): Record<string, unknown> }).toObject();
    }
    return new ActionBuilder<T>(name, [args]);
}

/**
 * Global registry mapping Go action names (e.g. "email.Contact") to their
 * generated JS wrapper functions. Populated by actions.gen.js at module load
 * time, consumed by DOMBinder when resolving action directive payloads.
 */
const actionFunctionRegistry = new Map<string, (...args: unknown[]) => ActionBuilder>();

/**
 * Registers an action function in the global registry.
 *
 * Called by the generated actions.gen.js to make action wrapper functions
 * discoverable by the DOMBinder for template-bound action calls
 * (e.g. p-on:submit="email.Contact($form)").
 *
 * @param name - The Go action name (e.g. "email.Contact").
 * @param actionFactory - The generated wrapper function that returns an ActionBuilder.
 */
export function registerActionFunction(name: string, actionFactory: (...args: unknown[]) => ActionBuilder): void {
    actionFunctionRegistry.set(name, actionFactory);
}

/**
 * Looks up an action function by its Go action name.
 *
 * @param name - The Go action name (e.g. "email.Contact").
 * @returns The wrapper function, or undefined if not registered.
 */
export function getActionFunction(name: string): ((...args: unknown[]) => ActionBuilder) | undefined {
    return actionFunctionRegistry.get(name);
}

/** Result of a single action within a batch. */
export interface BatchActionResult<T = unknown> {
    /** Action name. */
    name: string;
    /** HTTP status code. */
    status: number;
    /** Response data if successful. */
    data?: T;
    /** Error message if failed. */
    error?: string;
    /** Error code if failed. */
    code?: string;
}

/** Response from a batch action request. */
export interface BatchActionResponse<T extends unknown[]> {
    /** Results for each action in order. */
    results: { [K in keyof T]: BatchActionResult<T[K]> };
    /** True if all actions succeeded. */
    success: boolean;
}

/**
 * Executes multiple actions in a single HTTP request.
 *
 * Uses "continue all, report failures" strategy: all actions execute,
 * failures are reported in results.
 *
 * @param actions - ActionBuilders to execute.
 * @returns Promise resolving to batch response with all results.
 */
export async function batch<T extends unknown[]>(
    ...actions: { [K in keyof T]: ActionDescriptor<T[K]> }
): Promise<BatchActionResponse<T>> {
    const response = await fetch('/_piko/actions/_batch', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        credentials: 'same-origin',
        body: JSON.stringify({
            actions: actions.map(a => ({
                name: a.action,
                args: a.args?.reduce<Record<number, unknown>>((acc, val, i) => ({...acc, [i]: val}), {}) ?? {}
            }))
        })
    });

    if (!response.ok) {
        throw createActionError(
            response.status,
            `Batch request failed with status ${response.status}`
        );
    }

    return response.json() as Promise<BatchActionResponse<T>>;
}
