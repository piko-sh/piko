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

import {callServerActionDirect, marshalActionArgs, type DirectCallResponse} from '@/core/ActionExecutor';
import {getCSRFTokensFromMeta} from '@/core/CSRFUtils';
import {runtimeVersionGate} from '@/core/RuntimeVersion';

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
    /** Selects the delay strategy and defaults to exponential. */
    backoff?: RetryBackoff;
}

/** Configures automatic SSE reconnection. */
export interface RetryStreamConfig {
    /** Maximum reconnection attempts (Infinity for long-lived streams). */
    maxReconnects: number;
    /** Selects the delay strategy and defaults to linear. */
    backoff?: RetryBackoff;
    /** Sets the initial delay in milliseconds and defaults to 3000. */
    baseDelay?: number;
    /** Caps the delay in milliseconds and defaults to 30000. */
    maxDelay?: number;
    /**
     * Lists the HTTP statuses that trigger reconnection alongside transport failures. The
     * default statuses are 408, 425, 429, 500, 502, 503 and 504.
     */
    retryableStatuses?: number[];
    /**
     * Rereads CSRF tokens from the DOM before each reconnect. The option defaults to true
     * and uses the tokens already present in the page.
     */
    refreshTokensOnReconnect?: boolean;
    /** Called when SSE connection drops (before reconnect attempt). */
    onDisconnect?: () => void;
    /** Called when reconnection succeeds. */
    onReconnect?: (attemptNumber: number) => void;
    /** Receives each stream failure and whether a reconnect will follow. */
    onError?: (error: unknown, willReconnect: boolean) => void;
}

/** Error returned when an action fails. */
export interface ActionError {
    /** HTTP status code (0 for network errors). */
    status: number;
    /** Error message. */
    message: string;
    /** Field-level validation errors. */
    validationErrors?: Record<string, string | string[]>;
    /** Raw response data. */
    data?: unknown;
    /** Server-response helpers attached to error responses. */
    _helpers?: Array<{ name: string; args?: unknown[] }>;
}

/**
 * Classifies an action failure as a network, validation or authentication error when
 * applicable.
 *
 * @param status - The HTTP status or zero for a network failure.
 * @param message - A message describing the failure.
 * @param validationErrors - Optional field errors.
 * @param data - Optional response data.
 * @param helpers - Optional server-response helpers.
 * @returns The failure details and category flags.
 */
export function createActionError(
    status: number,
    message: string,
    validationErrors?: Record<string, string | string[]>,
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

    /** Selects the HTTP method and defaults to POST. */
    method?: ActionMethod;

    /** Runs immediately before the server call. */
    optimistic?: () => void;

    /** Handles a successful response and may return another action. */
    onSuccess?: (response: T) => void | ActionDescriptor;

    /** Runs when the action fails. */
    onError?: (error: ActionError) => void;

    /** Runs always (success or error). */
    onComplete?: () => void;

    /** Selects an element by CSS selector, by reference or with true for the trigger. */
    loading?: string | HTMLElement | boolean;

    /** Debounce in milliseconds before sending. */
    debounce?: number;

    /** Retry configuration. */
    retry?: RetryConfig;

    /** Sets how many milliseconds to wait before aborting a request. */
    timeout?: number;

    /** External AbortSignal for request cancellation. */
    signal?: AbortSignal;

    /** Whether to suppress automatic helper execution from server response. */
    shouldSuppressHelpers?: boolean;

    /** Progress callback for SSE streaming, called for each intermediate event. */
    onProgress?: (data: unknown, eventType: string) => void;

    /** Configures reconnection for dropped SSE streams. */
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

/** Configures an action and can be used directly as an ActionDescriptor. */
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
     * Sets the callback that runs before the request.
     *
     * @param optimisticUpdate - The callback that applies the optimistic update.
     * @returns The builder for chaining.
     */
    setOptimistic(optimisticUpdate: () => void): this {
        this._optimistic = optimisticUpdate;
        return this;
    }

    /**
     * Sets the callback for a successful response.
     *
     * @param successHandler - Handles the response and may return another action.
     * @returns The builder for chaining.
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
     * Sets the callback that runs after success or failure.
     *
     * @param completeHandler - The callback to run when the action finishes.
     * @returns The builder for chaining.
     */
    setOnComplete(completeHandler: () => void): this {
        this._onComplete = completeHandler;
        return this;
    }

    /**
     * Sets where to show the loading state.
     *
     * @param target - A CSS selector, an element, or true to use the trigger.
     * @returns The builder for chaining.
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
     * Sets when to abort the request.
     *
     * @param ms - The timeout in milliseconds.
     * @returns The builder for chaining.
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
     * Selects SSE transport and sets the callback for intermediate events. Terminal
     * events use the success or error callback.
     *
     * @param progressHandler - The callback that receives intermediate events.
     * @returns The builder for chaining.
     */
    setOnProgress(progressHandler: (data: unknown, eventType: string) => void): this {
        this._onProgress = progressHandler;
        return this;
    }

    /**
     * Configures reconnection when an SSE stream drops. Set the progress callback to
     * select SSE transport. Use Infinity for maxReconnects to allow unlimited retries.
     *
     * @param config - The retry limits, delays and callbacks.
     * @returns The builder for chaining.
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
     * Prevents automatic execution of server-response helpers.
     *
     * @returns The builder for chaining.
     */
    suppressHelpers(): this {
        this._suppressHelpers = true;
        return this;
    }

    /**
     * Creates a descriptor with the current action settings.
     *
     * @returns The configured action.
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
     * Executes the action and runs server-response helpers unless suppressed.
     *
     * @returns A promise for the response data that rejects if the action fails.
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
 * Configures an action from a single object of named arguments.
 *
 * @param name - The server action.
 * @param args - The argument values keyed by name.
 * @returns The builder for the named action.
 */
export function createActionBuilder<T = unknown>(name: string, args: Record<string, unknown>): ActionBuilder<T> {
    if (typeof (args as Record<string, unknown>).toObject === 'function') {
        args = (args as unknown as { toObject(): Record<string, unknown> }).toObject();
    }
    return new ActionBuilder<T>(name, [args]);
}

/** Maps Go action names to generated wrappers. */
const actionFunctionRegistry = new Map<string, (...args: unknown[]) => ActionBuilder>();

/**
 * Stores a generated wrapper for a Go action.
 *
 * @param name - The Go action name.
 * @param actionFactory - The generated function that builds calls to the action.
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
    /** Field-level validation errors, keyed by form field name. */
    errors?: Record<string, string | string[]>;
}

/** Response from a batch action request. */
export interface BatchActionResponse<T extends unknown[]> {
    /** Results for each action in order. */
    results: { [K in keyof T]: BatchActionResult<T[K]> };
    /** True if all actions succeeded. */
    success: boolean;
}

/** Options for a batch action request. */
export interface BatchOptions {
    /** Runs entries concurrently and keeps results in request order. */
    parallel?: boolean;
}

/**
 * Sends multiple actions in one HTTP request. The server runs each action even when
 * another action fails.
 *
 * @param actions - The actions to run.
 * @returns A promise for the ordered results, including individual failures.
 */
export async function batch<T extends unknown[]>(
    ...actions: { [K in keyof T]: ActionDescriptor<T[K]> }
): Promise<BatchActionResponse<T>> {
    return batchWith<T>({}, ...actions);
}

/**
 * Executes multiple actions in a single HTTP request, with batch-level options.
 *
 * @param options - Batch options such as parallel execution.
 * @param actions - ActionBuilders to execute.
 * @returns Promise resolving to batch response with all results.
 */
export async function batchWith<T extends unknown[]>(
    options: BatchOptions,
    ...actions: { [K in keyof T]: ActionDescriptor<T[K]> }
): Promise<BatchActionResponse<T>> {
    const {actionToken, ephemeralToken} = getCSRFTokensFromMeta();

    const headers: Record<string, string> = {'Content-Type': 'application/json'};
    if (actionToken) {
        headers['X-CSRF-Action-Token'] = actionToken;
    }

    const body: Record<string, unknown> = {
        actions: actions.map(a => ({
            name: a.action,
            args: marshalActionArgs(a.args ?? [])
        }))
    };
    if (ephemeralToken) {
        body['_csrf_ephemeral_token'] = ephemeralToken;
    }
    if (options.parallel) {
        body['parallel'] = true;
    }

    const response = await fetch('/_piko/actions/_batch', {
        method: 'POST',
        headers,
        credentials: 'same-origin',
        body: JSON.stringify(body)
    });
    runtimeVersionGate.observe(response.headers);

    if (!response.ok) {
        throw createActionError(
            response.status,
            `Batch request failed with status ${response.status}`
        );
    }

    return response.json() as Promise<BatchActionResponse<T>>;
}
