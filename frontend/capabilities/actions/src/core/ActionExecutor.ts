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

import {
    isActionDescriptor,
    createActionError,
    type ActionDescriptor,
    type ActionError,
    type RetryConfig,
    type RetryStreamConfig
} from '@/pk/action';
import {readSSEStream} from './SSEStreamReader';
import {getCSRFTokenFromMeta, getCSRFEphemeralFromMeta} from './CSRFUtils';
import {applyLoadingIndicator, removeLoadingIndicator} from '@/pk/loadingState';
import {
    ActionsHookEvent as HookEvent,
    type HookManagerLike as HookManager,
    type FormStateManagerLike as FormStateManager,
    type HelperRegistryLike as HelperRegistry
} from '@/coreServices';

/**
 * Holds data for a helper call from server response.
 */
interface HelperCall {
    /** The name of the helper to execute. */
    name: string;
    /** Optional arguments to pass to the helper. */
    args?: unknown[];
}

/**
 * Holds response data from an action endpoint.
 */
interface ActionResponseData {
    /** HTTP status code from the server. */
    status?: number;
    /** Success or error message from the server. */
    message?: string;
    /** Error message when the action fails. */
    error?: string;
    /** Validation errors keyed by field name. */
    errors?: Record<string, string[]>;
    /** Response payload data. */
    data?: unknown;
    /** Client-side helpers to execute after action completes. */
    _helpers?: HelperCall[];
}

/**
 * Function type for a global error handler invoked on every action error.
 */
export type GlobalActionErrorHandler = (
    error: ActionError,
    descriptor: ActionDescriptor
) => void;

/** HTTP status code for unprocessable entity (validation errors). */
const HTTP_STATUS_UNPROCESSABLE = 422;

/** HTTP status code for forbidden (CSRF errors). */
const HTTP_STATUS_FORBIDDEN = 403;

/** CSRF error code indicating an expired token. */
const CSRF_ERROR_EXPIRED = 'csrf_expired';

/** CSRF error code indicating an invalid token. */
const CSRF_ERROR_INVALID = 'csrf_invalid';

/** Default base delay between retries in milliseconds. */
const DEFAULT_RETRY_BASE_DELAY = 1000;

/** Maximum retry delay in milliseconds (cap for exponential backoff). */
const MAX_RETRY_DELAY = 30000;

/** Default SSE reconnect base delay in milliseconds. */
const DEFAULT_SSE_RECONNECT_DELAY = 3000;

/** Maximum SSE reconnect delay in milliseconds. */
const MAX_SSE_RECONNECT_DELAY = 30000;

/** HTTP status code for request timeout. */
const HTTP_STATUS_TIMEOUT = 408;

/** Minimum HTTP status code for server errors. */
const HTTP_STATUS_SERVER_ERROR = 500;

/** HTTP status code for success (default). */
const HTTP_STATUS_OK = 200;

/** Radix for random string generation (base 36 = 0-9 + a-z). */
const RANDOM_STRING_RADIX = 36;

/** Slice start index for random string generation. */
const RANDOM_STRING_SLICE_START = 2;

/** Slice end index for random string generation. */
const RANDOM_STRING_SLICE_END = 9;

/** Debounce timers keyed by action name and element. */
const debounceTimers = new Map<string, ReturnType<typeof setTimeout>>();

/** Global error handler, if registered. */
let globalErrorHandler: GlobalActionErrorHandler | null = null;

/** Optional HookManager for analytics events. */
let hookManager: HookManager | null = null;

/** Optional FormStateManager for dirty-state tracking. */
let formStateManager: FormStateManager | null = null;

/** Optional HelperRegistry for executing server-response helpers. */
let helperRegistry: HelperRegistry | null = null;

/**
 * Registers optional dependencies for the ActionExecutor.
 *
 * Called by the actions capability entry point after services are injected.
 *
 * @param deps - The dependencies to register.
 */
export function setActionExecutorDependencies(deps: {
    hookManager?: HookManager;
    formStateManager?: FormStateManager;
    helperRegistry?: HelperRegistry;
}): void {
    if (deps.hookManager) { hookManager = deps.hookManager; }
    if (deps.formStateManager) { formStateManager = deps.formStateManager; }
    if (deps.helperRegistry) { helperRegistry = deps.helperRegistry; }
}

/**
 * Resolves CSRF tokens, checking element data attributes first, then meta tags.
 *
 * Element-level tokens take precedence because partials may have fresher
 * tokens than the page-level meta tags after a partial refresh.
 *
 * @param element - Optional element to check for data-attribute tokens.
 * @returns The action token and ephemeral token.
 */
function getCSRFTokens(element?: HTMLElement): { actionToken: string | null; ephemeralToken: string | null } {
    const actionToken = element?.getAttribute('data-csrf-action-token')
        ?? getCSRFTokenFromMeta();
    const ephemeralToken = element?.getAttribute('data-csrf-ephemeral-token')
        ?? getCSRFEphemeralFromMeta();
    return {actionToken, ephemeralToken};
}

/**
 * Checks if a response indicates a CSRF error requiring recovery.
 *
 * @param status - The HTTP status code.
 * @param responseData - The parsed response data.
 * @returns True if the response is a CSRF error.
 */
function isCSRFError(status: number, responseData: ActionResponseData): boolean {
    return status === HTTP_STATUS_FORBIDDEN &&
        (responseData.error === CSRF_ERROR_EXPIRED || responseData.error === CSRF_ERROR_INVALID);
}

/**
 * Attempts CSRF recovery by refreshing the partial and retrying.
 *
 * For csrf_invalid, reloads the page. For csrf_expired, finds the closest
 * partial, dispatches a refresh event, then retries the action with fresh tokens.
 *
 * @param responseData - The parsed response data containing the error code.
 * @param element - The element that triggered the action.
 * @param retryAction - Callback to retry the action after recovery.
 * @returns True if recovery was initiated.
 */
function attemptCSRFRecovery(
    responseData: ActionResponseData,
    element: HTMLElement,
    retryAction: () => void
): boolean {
    if (responseData.error === CSRF_ERROR_INVALID) {
        window.location.reload();
        return true;
    }

    const partial = element.closest('[partial_src]') as HTMLElement | null;
    if (partial) {
        partial.dispatchEvent(new CustomEvent('refresh-partial', {
            bubbles: false,
            detail: {
                afterMorph: () => {
                    const refreshedEl = partial.querySelector('[data-csrf-action-token]') as HTMLElement | null;
                    if (refreshedEl) {
                        retryAction();
                    } else {
                        console.warn('[ActionExecutor] Could not find element with CSRF token after partial refresh');
                    }
                }
            }
        }));
        return true;
    }

    window.location.reload();
    return true;
}

/**
 * Validates a form using HTML5 constraint validation before submission.
 *
 * Matches native browser behaviour by skipping validation when:
 * - The form has the `novalidate` attribute ({@link https://developer.mozilla.org/en-US/docs/Web/HTML/Element/form#novalidate form.noValidate})
 * - The submitter has the `formnovalidate` attribute ({@link https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Attributes/formnovalidate submitter.formNoValidate})
 *
 * @param element - The element whose closest form to validate.
 * @param event - The originating event, used to check the submitter's formnovalidate.
 * @returns True if validation passes or no form is found, false to abort.
 */
function validateForm(element: HTMLElement, event?: Event): boolean {
    const form = element.closest('form') as HTMLFormElement | null;
    if (!form) { return true; }

    if (form.noValidate) { return true; }

    const submitter = (event as SubmitEvent | undefined)?.submitter as HTMLButtonElement | null;
    if (submitter?.formNoValidate) { return true; }

    return form.reportValidity();
}

/**
 * Clears previous server validation error attributes from form fields.
 *
 * @param form - The form element to clear errors from.
 */
function clearPreviousErrors(form: HTMLElement): void {
    form.querySelectorAll<HTMLElement>('[error]').forEach(el => {
        el.removeAttribute('error');
    });
}

/**
 * Applies server validation errors to matching form fields.
 *
 * Sets an `error` attribute on each field with the concatenated error messages.
 *
 * @param form - The form element containing the fields.
 * @param errors - Validation errors keyed by field name.
 */
function applyServerErrors(form: HTMLElement, errors: Record<string, string[]>): void {
    clearPreviousErrors(form);

    for (const [fieldName, messages] of Object.entries(errors)) {
        const errorMessage = messages.join(', ');
        const fields = form.querySelectorAll<HTMLElement>(`[name="${fieldName}"]`);

        if (fields.length > 0) {
            fields.forEach(field => {
                field.setAttribute('error', errorMessage);
            });
        }
    }
}

/**
 * Registers a global error handler for all action errors.
 *
 * This handler is called for every action error, regardless of whether
 * the action has its own onError callback. Useful for analytics, logging,
 * or global error UI.
 *
 * @param handler - The error handler function.
 * @returns A function to unregister the handler.
 */
export function onActionError(handler: GlobalActionErrorHandler): () => void {
    globalErrorHandler = handler;
    return () => {
        if (globalErrorHandler === handler) {
            globalErrorHandler = null;
        }
    };
}

/**
 * Clears the global error handler.
 */
export function clearGlobalErrorHandler(): void {
    globalErrorHandler = null;
}

/**
 * Shows loading state on target element(s).
 *
 * @param target - The loading target: true for triggering element, string for CSS selector, or HTMLElement.
 * @param element - The triggering element used when target is true.
 */
function showLoading(target: string | HTMLElement | boolean, element: HTMLElement): void {
    if (target === true) {
        applyLoadingIndicator(element);
    } else if (typeof target === 'string') {
        const el = document.querySelector<HTMLElement>(target);
        if (el) {
            applyLoadingIndicator(el);
        }
    } else if (target instanceof HTMLElement) {
        applyLoadingIndicator(target);
    }
}

/**
 * Hides loading state from target element(s).
 *
 * @param target - The loading target: true for triggering element, string for CSS selector, or HTMLElement.
 * @param element - The triggering element used when target is true.
 */
function hideLoading(target: string | HTMLElement | boolean, element: HTMLElement): void {
    if (target === true) {
        removeLoadingIndicator(element);
    } else if (typeof target === 'string') {
        const el = document.querySelector<HTMLElement>(target);
        if (el) {
            removeLoadingIndicator(el);
        }
    } else if (target instanceof HTMLElement) {
        removeLoadingIndicator(target);
    }
}

/**
 * Calculates the delay for a retry attempt using linear or exponential backoff.
 *
 * @param attempt - Current attempt number (0-based).
 * @param config - Retry configuration.
 * @returns Delay in milliseconds.
 */
function calculateRetryDelay(attempt: number, config: RetryConfig): number {
    const backoff = config.backoff ?? 'exponential';

    if (backoff === 'linear') {
        return Math.min(DEFAULT_RETRY_BASE_DELAY * (attempt + 1), MAX_RETRY_DELAY);
    }

    return Math.min(DEFAULT_RETRY_BASE_DELAY * Math.pow(2, attempt), MAX_RETRY_DELAY);
}

/**
 * Waits for the specified delay.
 *
 * @param ms - The delay duration in milliseconds.
 * @returns A promise that resolves after the delay.
 */
function delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Executes a server action with retry support.
 *
 * @param actionName - The action name to execute.
 * @param args - Arguments to pass to the action.
 * @param method - HTTP method to use.
 * @param actionToken - CSRF action token for the request.
 * @param ephemeralToken - CSRF ephemeral token for the request.
 * @param retryConfig - Optional retry configuration.
 * @param options - Optional timeout and abort signal.
 * @returns The response data from the server.
 */
async function executeWithRetry(
    actionName: string,
    args: unknown[],
    method: string,
    actionToken: string | null,
    ephemeralToken: string | null,
    retryConfig?: RetryConfig,
    options?: ExecuteOptions
): Promise<unknown> {
    const maxAttempts = retryConfig?.attempts ?? 1;
    let lastError: ActionError | null = null;

    for (let attempt = 0; attempt < maxAttempts; attempt++) {
        try {
            return await executeServerAction(actionName, args, method, actionToken, ephemeralToken, options);
        } catch (error) {
            if (error instanceof Error && !('status' in error)) {
                lastError = createActionError(0, error.message);
            } else {
                lastError = error as ActionError;
            }

            const isTimeout = lastError.status === HTTP_STATUS_TIMEOUT;
            const isCancelled = lastError.status === 0 && lastError.message === 'Request cancelled';
            const isRetryable = (lastError.status === 0 || lastError.status >= HTTP_STATUS_SERVER_ERROR || isTimeout) && !isCancelled;

            if (!isRetryable || attempt >= maxAttempts - 1) {
                throw lastError;
            }

            const retryDelay = calculateRetryDelay(attempt, retryConfig ?? {attempts: maxAttempts});
            await delay(retryDelay);
        }
    }

    throw lastError;
}

/**
 * Holds options for server action execution.
 */
interface ExecuteOptions {
    /** Request timeout in milliseconds. */
    timeout?: number;
    /** AbortSignal for external cancellation. */
    signal?: AbortSignal;
}

/**
 * Marshal action arguments and the ephemeral token into a request body.
 *
 * Detect File/Blob values and return FormData when files are present,
 * otherwise return a JSON string with the appropriate Content-Type header.
 *
 * @param args - The action arguments to marshal.
 * @param ephemeralToken - The CSRF ephemeral token to include.
 * @returns An object containing the body and any extra headers.
 */
function buildActionBody(
    args: unknown[],
    ephemeralToken: string | null
): {body: BodyInit; headers: HeadersInit} {
    const headers: HeadersInit = {};
    const bodyData: Record<string, unknown> = {};

    if (args.length > 0) {
        if (args.length === 1 && typeof args[0] === 'object' && args[0] !== null) {
            Object.assign(bodyData, args[0] as Record<string, unknown>);
        } else {
            bodyData['args'] = args
                .map((v, i) => ({[i]: v}))
                .reduce((acc, b) => ({...acc, ...b}), {});
        }
    }

    if (ephemeralToken) {
        bodyData['_csrf_ephemeral_token'] = ephemeralToken;
    }

    const hasFiles = Object.values(bodyData).some(
        v => v instanceof File || v instanceof Blob
    );

    if (hasFiles) {
        const formData = new FormData();
        for (const [key, value] of Object.entries(bodyData)) {
            if (value instanceof File) {
                formData.append(key, value, value.name);
            } else if (value instanceof Blob) {
                formData.append(key, value);
            } else if (value !== undefined && value !== null) {
                formData.append(key, String(value));
            }
        }
        return {body: formData, headers};
    }

    headers['Content-Type'] = 'application/json';
    return {body: JSON.stringify(bodyData), headers};
}

/**
 * Create an AbortController linked to an optional external signal and timeout.
 *
 * @param options - The execution options containing a signal and timeout.
 * @returns An object containing the controller and timeout identifier.
 */
function createRequestAbortController(options?: ExecuteOptions): {
    controller: AbortController;
    timeoutId: ReturnType<typeof setTimeout> | undefined;
} {
    const controller = new AbortController();
    let timeoutId: ReturnType<typeof setTimeout> | undefined;

    if (options?.signal) {
        if (options.signal.aborted) {
            controller.abort();
        } else {
            options.signal.addEventListener('abort', () => controller.abort());
        }
    }

    if (options?.timeout && options.timeout > 0) {
        timeoutId = setTimeout(() => controller.abort(), options.timeout);
    }

    return {controller, timeoutId};
}

/**
 * Parse a fetch Response as JSON and throw an ActionError when the status is not OK.
 *
 * @param response - The fetch Response to parse.
 * @returns The parsed action response data.
 * @throws ActionError when the response indicates a failure.
 */
async function parseActionResponse(response: Response): Promise<ActionResponseData> {
    let responseData: ActionResponseData;
    try {
        responseData = (await response.json()) as ActionResponseData;
    } catch {
        throw createActionError(
            response.status,
            'Failed to parse server response',
            undefined,
            undefined
        );
    }

    if (!response.ok) {
        const validationErrors = response.status === HTTP_STATUS_UNPROCESSABLE
            ? responseData.errors
            : undefined;

        throw createActionError(
            response.status,
            responseData.message ?? responseData.error ?? `Action failed with status ${response.status}`,
            validationErrors,
            responseData.error ?? responseData.data,
            responseData._helpers
        );
    }

    return responseData;
}

/**
 * Execute a server action via fetch.
 *
 * @param actionName - The action name to execute.
 * @param args - The arguments to pass to the action.
 * @param method - The HTTP method to use.
 * @param actionToken - The CSRF action token for the request.
 * @param ephemeralToken - The CSRF ephemeral token for the request.
 * @param options - Optional timeout and abort signal.
 * @returns The response data from the server.
 */
async function executeServerAction(
    actionName: string,
    args: unknown[],
    method: string,
    actionToken: string | null,
    ephemeralToken: string | null,
    options?: ExecuteOptions
): Promise<unknown> {
    const {body, headers} = buildActionBody(args, ephemeralToken);

    if (actionToken) {
        (headers as Record<string, string>)['X-CSRF-Action-Token'] = actionToken;
    }

    const {controller, timeoutId} = createRequestAbortController(options);

    try {
        const response = await fetch(`/_piko/actions/${actionName}`, {
            method,
            headers,
            credentials: 'same-origin',
            body,
            signal: controller.signal
        });

        return await parseActionResponse(response);
    } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
            const isTimeout = options?.timeout && !options.signal?.aborted;
            throw createActionError(
                isTimeout ? HTTP_STATUS_TIMEOUT : 0,
                isTimeout ? 'Request timeout' : 'Request cancelled',
                undefined,
                undefined
            );
        }
        throw error;
    } finally {
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
    }
}

/**
 * Generates a unique key for debouncing based on action and element.
 *
 * @param actionName - The action name.
 * @param element - The element triggering the action.
 * @returns A unique key combining action name and element identifier.
 */
function getDebounceKey(actionName: string, element: HTMLElement): string {
    element.dataset.ppActionId ??= `action-${Date.now()}-${Math.random().toString(RANDOM_STRING_RADIX).slice(RANDOM_STRING_SLICE_START, RANDOM_STRING_SLICE_END)}`;
    return `${actionName}:${element.dataset.ppActionId}`;
}

/**
 * Clears any pending debounced action for the given key.
 *
 * @param key - The debounce key to clear.
 */
function clearDebounce(key: string): void {
    const timer = debounceTimers.get(key);
    if (timer) {
        clearTimeout(timer);
        debounceTimers.delete(key);
    }
}

/**
 * Clears all debounce timers.
 */
export function clearAllDebounceTimers(): void {
    for (const timer of debounceTimers.values()) {
        clearTimeout(timer);
    }
    debounceTimers.clear();
}

/**
 * Executes helpers from the response, awaiting any that return promises.
 *
 * @param helpers - The helper calls to execute.
 * @param element - The element context for helper execution.
 * @param event - Optional event that triggered the action.
 */
async function executeHelpers(
    helpers: HelperCall[],
    element: HTMLElement,
    event?: Event
): Promise<void> {
    if (!helperRegistry) {
        return;
    }

    const errors: Error[] = [];

    for (const helper of helpers) {
        try {
            const args = (helper.args ?? []).map(a => String(a));
            await helperRegistry.execute(helper.name, element, event as Event, args);
        } catch (error) {
            console.error(`[ActionExecutor] Helper "${helper.name}" failed:`, error);
            errors.push(error as Error);
        }
    }

    if (errors.length > 0) {
        throw new AggregateError(errors, `${errors.length} helper(s) failed`);
    }
}

/**
 * Handles the success callback and any chained action.
 *
 * @param descriptor - The action descriptor containing the callback.
 * @param response - The response data from the server.
 * @param element - The element context for chained actions.
 * @param event - Optional event that triggered the action.
 */
async function invokeSuccessCallback(
    descriptor: ActionDescriptor,
    response: ActionResponseData,
    element: HTMLElement,
    event?: Event
): Promise<void> {
    if (!descriptor.onSuccess) {
        return;
    }

    try {
        const data = response.data ?? response;
        const next = descriptor.onSuccess(data);

        if (isActionDescriptor(next)) {
            await handleAction(next, element, event);
        }
    } catch (error) {
        console.error('[ActionExecutor] onSuccess callback failed:', error);
        throw error;
    }
}

/**
 * Validate the enclosing form and emit the ACTION_START hook.
 *
 * Clear previous validation errors, run HTML5 validation, and emit
 * an ACTION_COMPLETE hook with validationFailed when the form is invalid.
 *
 * @param descriptor - The action descriptor being executed.
 * @param element - The element triggering the action.
 * @param form - The enclosing form element, or null if none.
 * @param actionStartTime - The timestamp when the action started.
 * @param event - The originating event, forwarded to validateForm for formnovalidate support.
 * @returns False if validation failed and the caller should abort.
 */
function validateAndEmitStart(
    descriptor: ActionDescriptor,
    element: HTMLElement,
    form: HTMLFormElement | null,
    actionStartTime: number,
    event?: Event
): boolean {
    if (form) {
        clearPreviousErrors(form);
        if (!validateForm(element, event)) {
            hookManager?.emit(HookEvent.ACTION_COMPLETE, {
                action: descriptor.action,
                method: descriptor.method ?? 'POST',
                elementTag: element.tagName.toLowerCase(),
                success: false,
                statusCode: 0,
                duration: Date.now() - actionStartTime,
                timestamp: Date.now(),
                validationFailed: true
            });
            return false;
        }
    }

    hookManager?.emit(HookEvent.ACTION_START, {
        action: descriptor.action,
        method: descriptor.method ?? 'POST',
        elementTag: element.tagName.toLowerCase(),
        timestamp: actionStartTime
    });

    return true;
}

/**
 * Dispatch the server request using SSE or standard HTTP transport.
 *
 * Choose SSE transport when the descriptor has an onProgress callback,
 * with optional retry-stream support. Otherwise, fall back to the standard
 * fetch-with-retry path.
 *
 * @param descriptor - The action descriptor containing transport options.
 * @param element - The element triggering the action.
 * @returns The action response data from the server.
 */
async function executeServerRequest(
    descriptor: ActionDescriptor,
    element: HTMLElement
): Promise<ActionResponseData> {
    const {actionToken, ephemeralToken} = getCSRFTokens(element);

    if (descriptor.onProgress) {
        const sseOptions: ExecuteOptions = {timeout: descriptor.timeout, signal: descriptor.signal};
        let data: unknown;

        if (descriptor.retryStream) {
            data = await executeServerActionSSEWithRetry({
                actionName: descriptor.action,
                args: descriptor.args ?? [],
                method: descriptor.method ?? 'POST',
                actionToken,
                ephemeralToken,
                onProgress: descriptor.onProgress,
                retryConfig: descriptor.retryStream,
                options: sseOptions
            });
        } else {
            data = await executeServerActionSSE({
                actionName: descriptor.action,
                args: descriptor.args ?? [],
                method: descriptor.method ?? 'POST',
                actionToken,
                ephemeralToken,
                onProgress: descriptor.onProgress,
                options: sseOptions
            });
        }

        return {data, status: HTTP_STATUS_OK};
    }

    return await executeWithRetry(
        descriptor.action,
        descriptor.args ?? [],
        descriptor.method ?? 'POST',
        actionToken,
        ephemeralToken,
        descriptor.retry,
        {timeout: descriptor.timeout, signal: descriptor.signal}
    ) as ActionResponseData;
}

/**
 * Handle a successful action response.
 *
 * Execute server-response helpers, mark the form clean, emit the
 * ACTION_COMPLETE hook, and invoke the success callback.
 *
 * @param descriptor - The action descriptor containing callbacks.
 * @param response - The action response data from the server.
 * @param element - The element that triggered the action.
 * @param event - The original event, or undefined.
 * @param form - The enclosing form element, or null if none.
 * @param actionStartTime - The timestamp when the action started.
 */
async function handleActionSuccess(
    descriptor: ActionDescriptor,
    response: ActionResponseData,
    element: HTMLElement,
    event: Event | undefined,
    form: HTMLFormElement | null,
    actionStartTime: number
): Promise<void> {
    if (!descriptor.shouldSuppressHelpers && response._helpers && response._helpers.length > 0) {
        await executeHelpers(response._helpers, element, event);
    }

    if (form && formStateManager) {
        formStateManager.markFormClean(form);
    }

    hookManager?.emit(HookEvent.ACTION_COMPLETE, {
        action: descriptor.action,
        method: descriptor.method ?? 'POST',
        elementTag: element.tagName.toLowerCase(),
        success: true,
        statusCode: response.status ?? HTTP_STATUS_OK,
        duration: Date.now() - actionStartTime,
        timestamp: Date.now()
    });

    await invokeSuccessCallback(descriptor, response, element, event);
}

/**
 * Invokes the global and descriptor-level error handlers for a failed action.
 *
 * @param actionError - The normalised action error.
 * @param descriptor - The action descriptor containing the onError callback.
 * @param rawError - The original caught error for fallback logging.
 */
function invokeErrorHandlers(actionError: ActionError, descriptor: ActionDescriptor, rawError: unknown): void {
    if (globalErrorHandler) {
        try {
            globalErrorHandler(actionError, descriptor);
        } catch (handlerError) {
            console.error('[ActionExecutor] Global error handler failed:', handlerError);
        }
    }

    if (descriptor.onError) {
        try {
            descriptor.onError(actionError);
        } catch (callbackError) {
            console.error('[ActionExecutor] onError callback failed:', callbackError);
        }
    } else if (!globalErrorHandler) {
        console.error('[ActionExecutor] Action failed:', rawError);
    }
}

/**
 * Handle a failed action by attempting CSRF recovery, applying validation
 * errors, emitting the failure hook, and invoking error handlers.
 *
 * @param descriptor - The action descriptor containing error callbacks.
 * @param error - The caught error value.
 * @param element - The element that triggered the action.
 * @param event - The original event, or undefined.
 * @param form - The enclosing form element, or null if none.
 * @param actionStartTime - The timestamp when the action started.
 * @returns True if CSRF recovery was initiated and the caller should return early.
 */
async function handleActionFailure(
    descriptor: ActionDescriptor,
    error: unknown,
    element: HTMLElement,
    event: Event | undefined,
    form: HTMLFormElement | null,
    actionStartTime: number
): Promise<boolean> {
    const actionError = error as ActionError;

    const errorCode = typeof actionError.data === 'string' ? actionError.data : undefined;
    if (isCSRFError(actionError.status, {error: errorCode})) {
        const recovered = attemptCSRFRecovery(
            {error: errorCode},
            element,
            () => { void executeAction(descriptor, element, event); }
        );
        if (recovered) { return true; }
    }

    if (actionError._helpers && actionError._helpers.length > 0) {
        await executeHelpers(actionError._helpers as HelperCall[], element, event);
    }

    if (actionError.status === HTTP_STATUS_UNPROCESSABLE && actionError.validationErrors && form) {
        applyServerErrors(form, actionError.validationErrors);
    }

    hookManager?.emit(HookEvent.ACTION_COMPLETE, {
        action: descriptor.action,
        method: descriptor.method ?? 'POST',
        elementTag: element.tagName.toLowerCase(),
        success: false,
        statusCode: actionError.status,
        duration: Date.now() - actionStartTime,
        timestamp: Date.now()
    });

    invokeErrorHandlers(actionError, descriptor, error);

    return false;
}

/**
 * Core action execution logic (without debounce wrapper).
 *
 * @param descriptor - The action descriptor to execute.
 * @param element - The element triggering the action.
 * @param event - Optional event that triggered the action.
 */
async function executeAction(
    descriptor: ActionDescriptor,
    element: HTMLElement,
    event?: Event
): Promise<void> {
    const actionStartTime = Date.now();
    const form = element.closest('form') as HTMLFormElement | null;

    if (!validateAndEmitStart(descriptor, element, form, actionStartTime, event)) {
        return;
    }

    if (descriptor.optimistic) {
        try {
            descriptor.optimistic();
        } catch (error) {
            console.error('[ActionExecutor] Optimistic update failed:', error);
        }
    }

    if (descriptor.loading !== undefined) {
        showLoading(descriptor.loading, element);
    }

    try {
        const response = await executeServerRequest(descriptor, element);
        await handleActionSuccess(descriptor, response, element, event, form, actionStartTime);
    } catch (error) {
        const recovered = await handleActionFailure(descriptor, error, element, event, form, actionStartTime);
        if (recovered) { return; }
        throw error;
    } finally {
        if (descriptor.onComplete) {
            try {
                descriptor.onComplete();
            } catch (error) {
                console.error('[ActionExecutor] onComplete callback failed:', error);
            }
        }

        if (descriptor.loading !== undefined) {
            hideLoading(descriptor.loading, element);
        }
    }
}

/**
 * Handles an action descriptor through its full lifecycle.
 *
 * Applies debounce if configured, runs optimistic updates, shows loading state,
 * calls the server action (with retry if configured), processes helpers,
 * invokes success/error callbacks, and always calls onComplete.
 *
 * @param descriptor - The action descriptor to execute.
 * @param element - The triggering element (for loading state).
 * @param event - The original event (optional).
 */
export async function handleAction(
    descriptor: ActionDescriptor,
    element: HTMLElement,
    event?: Event
): Promise<void> {
    if (descriptor.debounce && descriptor.debounce > 0) {
        const debounceKey = getDebounceKey(descriptor.action, element);

        clearDebounce(debounceKey);

        return new Promise<void>((resolve, reject) => {
            const timer = setTimeout(() => {
                debounceTimers.delete(debounceKey);
                executeAction(descriptor, element, event).then(resolve).catch((err: unknown) => {
                    console.error('[ActionExecutor] Debounced action failed:', err);
                    reject(err instanceof Error ? err : new Error(String(err)));
                });
            }, descriptor.debounce);

            debounceTimers.set(debounceKey, timer);
        });
    }

    return executeAction(descriptor, element, event);
}

/**
 * Holds options for direct action calls.
 */
export interface DirectCallOptions {
    /** Request timeout in milliseconds. */
    timeout?: number;
    /** AbortSignal for external cancellation. */
    signal?: AbortSignal;
    /** When true, server-response helpers are not auto-executed. */
    suppressHelpers?: boolean;
    /** Progress callback - when set, uses SSE transport instead of standard HTTP. */
    onProgress?: (data: unknown, eventType: string) => void;
    /** Retry configuration for SSE streams - enables auto-reconnection on connection drop. */
    retryStream?: RetryStreamConfig;
}

/**
 * Holds the response from a direct action call.
 */
export interface DirectCallResponse<T = unknown> {
    /** The typed response data from the action. */
    data: T;
    /** HTTP status code from the server. */
    status: number;
    /** Optional message from the server. */
    message?: string;
    /** Helper calls from the server response (available when suppressHelpers is used). */
    helpers?: HelperCall[];
}

/** Content type for SSE responses. */
const SSE_CONTENT_TYPE = 'text/event-stream';

/** Accept header value to request SSE transport. */
const SSE_ACCEPT_HEADER = 'text/event-stream';

/**
 * Calculates the delay for an SSE reconnection attempt.
 *
 * @param attempt - Current attempt number (0-based).
 * @param config - Retry stream configuration.
 * @returns Delay in milliseconds.
 */
function calculateSSEReconnectDelay(attempt: number, config: RetryStreamConfig): number {
    const baseDelay = config.baseDelay ?? DEFAULT_SSE_RECONNECT_DELAY;
    const maxDelay = config.maxDelay ?? MAX_SSE_RECONNECT_DELAY;
    const backoff = config.backoff ?? 'linear';

    if (backoff === 'linear') {
        return Math.min(baseDelay * (attempt + 1), maxDelay);
    }

    return Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
}

/**
 * Parameters for the internal SSE transport implementation.
 */
interface SSEInternalParams {
    /** The action name to execute. */
    actionName: string;
    /** Arguments to pass to the action. */
    args: unknown[];
    /** HTTP method to use. */
    method: string;
    /** CSRF action token for the request. */
    actionToken: string | null;
    /** CSRF ephemeral token for the request. */
    ephemeralToken: string | null;
    /** Callback invoked for each SSE event. */
    onProgress: (data: unknown, eventType: string) => void;
    /** Optional timeout and abort signal. */
    options?: ExecuteOptions;
    /** Last received event ID for resumption. */
    lastEventId?: string;
    /** Callback invoked when an event with an id field is received. */
    onEventId?: (id: string) => void;
}

/**
 * Builds the HTTP headers for an SSE request.
 *
 * @param actionToken - CSRF action token.
 * @param ephemeralToken - CSRF ephemeral token.
 * @param lastEventId - Optional last event ID for resumption.
 * @returns The headers object.
 */
function buildSSEHeaders(
    actionToken: string | null,
    ephemeralToken: string | null,
    lastEventId?: string
): HeadersInit {
    const headers: HeadersInit = {
        'Content-Type': 'application/json',
        'Accept': SSE_ACCEPT_HEADER
    };

    if (actionToken && ephemeralToken) {
        headers['X-CSRF-Action-Token'] = actionToken;
    }

    if (lastEventId) {
        headers['Last-Event-ID'] = lastEventId;
    }

    return headers;
}

/**
 * Builds the URL for an SSE request, appending the ephemeral token as a query parameter.
 *
 * @param actionName - The action name.
 * @param actionToken - CSRF action token.
 * @param ephemeralToken - CSRF ephemeral token.
 * @returns The fully constructed URL.
 */
function buildSSEUrl(actionName: string, actionToken: string | null, ephemeralToken: string | null): string {
    let url = `/_piko/actions/${actionName}`;
    if (actionToken && ephemeralToken) {
        url += `?_csrf_ephemeral_token=${encodeURIComponent(ephemeralToken)}`;
    }
    return url;
}

/**
 * Builds the request body for an SSE request.
 *
 * @param args - The action arguments.
 * @returns The body data object.
 */
function buildSSEBody(args: unknown[]): Record<string, unknown> {
    const bodyData: Record<string, unknown> = {};

    if (args.length > 0) {
        if (args.length === 1 && typeof args[0] === 'object' && args[0] !== null) {
            Object.assign(bodyData, args[0] as Record<string, unknown>);
        } else {
            bodyData['args'] = args
                .map((v, i) => ({[i]: v}))
                .reduce((acc, b) => ({...acc, ...b}), {});
        }
    }

    return bodyData;
}

/**
 * Creates an AbortController linked to optional external signal and timeout.
 *
 * @param options - Optional timeout and abort signal.
 * @returns The controller and timeout ID.
 */
function setupAbortControl(options?: ExecuteOptions): {controller: AbortController; timeoutId: ReturnType<typeof setTimeout> | undefined} {
    const controller = new AbortController();
    let timeoutId: ReturnType<typeof setTimeout> | undefined;

    if (options?.signal) {
        if (options.signal.aborted) {
            controller.abort();
        } else {
            options.signal.addEventListener('abort', () => controller.abort());
        }
    }

    if (options?.timeout && options.timeout > 0) {
        timeoutId = setTimeout(() => controller.abort(), options.timeout);
    }

    return {controller, timeoutId};
}

/**
 * Parses an error response and throws it as an ActionError.
 *
 * @param response - The HTTP response with a non-OK status.
 * @throws ActionError with the parsed response details.
 */
async function throwSSEErrorResponse(response: Response): Promise<never> {
    let responseData: ActionResponseData;
    try {
        responseData = (await response.json()) as ActionResponseData;
    } catch {
        throw createActionError(response.status, `Action failed with status ${response.status}`);
    }

    const validationErrors = response.status === HTTP_STATUS_UNPROCESSABLE
        ? responseData.errors
        : undefined;

    throw createActionError(
        response.status,
        responseData.message ?? responseData.error ?? `Action failed with status ${response.status}`,
        validationErrors,
        responseData.error ?? responseData.data,
        responseData._helpers
    );
}

/**
 * Re-throws an error as an ActionError, converting AbortErrors to timeout or cancellation.
 *
 * @param error - The caught error.
 * @param options - Optional execute options for timeout detection.
 * @throws ActionError.
 */
function rethrowAsActionError(error: unknown, options?: ExecuteOptions): never {
    if (error instanceof DOMException && error.name === 'AbortError') {
        const isTimeout = options?.timeout && !options.signal?.aborted;
        throw createActionError(
            isTimeout ? HTTP_STATUS_TIMEOUT : 0,
            isTimeout ? 'Request timeout' : 'Request cancelled'
        );
    }
    throw error;
}

/**
 * Sends a POST with Accept: text/event-stream. If the server responds with SSE,
 * reads the stream and routes events to the progress callback. Falls back to
 * standard JSON response parsing if the server does not support SSE.
 *
 * @param params - The SSE request parameters.
 * @returns The completion data from the SSE stream or JSON response.
 */
async function executeServerActionSSEInternal(params: SSEInternalParams): Promise<unknown> {
    const {actionName, args, method, actionToken, ephemeralToken, onProgress, options, lastEventId, onEventId} = params;

    const headers = buildSSEHeaders(actionToken, ephemeralToken, lastEventId);
    const url = buildSSEUrl(actionName, actionToken, ephemeralToken);
    const bodyData = buildSSEBody(args);
    const {controller, timeoutId} = setupAbortControl(options);

    try {
        const response = await fetch(url, {
            method,
            headers,
            credentials: 'same-origin',
            body: JSON.stringify(bodyData),
            signal: controller.signal
        });

        if (!response.ok) {
            await throwSSEErrorResponse(response);
        }

        const contentType = response.headers.get('Content-Type') ?? '';

        if (contentType.startsWith(SSE_CONTENT_TYPE) && response.body) {
            return await readSSEStream(response.body, {onEvent: onProgress, onEventId}, controller.signal);
        }

        let responseData: ActionResponseData;
        try {
            responseData = (await response.json()) as ActionResponseData;
        } catch {
            throw createActionError(response.status, 'Failed to parse server response');
        }
        return responseData;
    } catch (error) {
        return rethrowAsActionError(error, options);
    } finally {
        if (timeoutId) {
            clearTimeout(timeoutId);
        }
    }
}

/**
 * Executes a server action via SSE transport without retry.
 *
 * @param params - The SSE request parameters.
 * @returns The completion data from the SSE stream.
 */
async function executeServerActionSSE(params: SSEInternalParams): Promise<unknown> {
    return executeServerActionSSEInternal(params);
}

/**
 * Parameters for SSE transport with auto-reconnection.
 */
interface SSERetryParams {
    /** The action name to execute. */
    actionName: string;
    /** Arguments to pass to the action. */
    args: unknown[];
    /** HTTP method to use. */
    method: string;
    /** CSRF action token for the request. */
    actionToken: string | null;
    /** CSRF ephemeral token for the request. */
    ephemeralToken: string | null;
    /** Callback invoked for each SSE event. */
    onProgress: (data: unknown, eventType: string) => void;
    /** Retry stream configuration. */
    retryConfig: RetryStreamConfig;
    /** Optional timeout and abort signal. */
    options?: ExecuteOptions;
}

/**
 * Executes a server action via SSE transport with auto-reconnection.
 *
 * Wraps the internal SSE implementation in a reconnection loop. On transient
 * connection drops, waits with configurable backoff and retries. On application
 * errors or cancellation, throws immediately.
 *
 * Tracks the last received event ID and sends it as Last-Event-ID on reconnect,
 * allowing the server action to skip already-sent events.
 *
 * @param params - The SSE retry parameters.
 * @returns The completion data from the SSE stream.
 */
async function executeServerActionSSEWithRetry(params: SSERetryParams): Promise<unknown> {
    const {actionName, args, method, actionToken, ephemeralToken, onProgress, retryConfig, options} = params;
    let reconnectCount = 0;
    let lastEventId: string | undefined;
    const maxReconnects = retryConfig.maxReconnects;

    for (;;) {
        try {
            let isReconnection = reconnectCount > 0;
            const wrappedOnProgress = (data: unknown, eventType: string): void => {
                if (isReconnection) {
                    reconnectCount = 0;
                    isReconnection = false;
                }
                onProgress(data, eventType);
            };

            const result = await executeServerActionSSEInternal({
                actionName, args, method,
                actionToken, ephemeralToken,
                onProgress: wrappedOnProgress,
                options,
                lastEventId,
                onEventId: (id: string) => { lastEventId = id; }
            });

            return result;
        } catch (error) {
            const actionError = error as ActionError;

            if (actionError.message === 'Request cancelled') {
                throw error;
            }

            if (actionError.data !== undefined) {
                throw error;
            }

            if (actionError.status !== 0) {
                throw error;
            }

            if (reconnectCount >= maxReconnects) {
                throw createActionError(0,
                    `SSE stream failed after ${reconnectCount} reconnection attempts`);
            }

            retryConfig.onDisconnect?.();

            const reconnectDelay = calculateSSEReconnectDelay(reconnectCount, retryConfig);
            await delay(reconnectDelay);

            if (options?.signal?.aborted) {
                throw createActionError(0, 'Request cancelled');
            }

            reconnectCount++;
            retryConfig.onReconnect?.(reconnectCount);
        }
    }
}

/**
 * Executes an action directly and returns the typed response.
 *
 * Used by ActionBuilder.call() for imperative action execution.
 * Helpers are processed automatically before returning.
 *
 * @param actionName - The action name (e.g., 'media.Search').
 * @param args - Arguments to pass to the action.
 * @param method - HTTP method (default: POST).
 * @param options - Optional timeout, abort signal, and progress callback.
 * @returns The action response with typed data.
 */
export async function callServerActionDirect<T = unknown>(
    actionName: string,
    args: unknown[],
    method: string = 'POST',
    options?: DirectCallOptions
): Promise<DirectCallResponse<T>> {
    const {actionToken, ephemeralToken} = getCSRFTokens();

    if (options?.onProgress) {
        let data: unknown;
        const sseOptions: ExecuteOptions = {timeout: options.timeout, signal: options.signal};

        if (options.retryStream) {
            data = await executeServerActionSSEWithRetry({
                actionName, args, method,
                actionToken, ephemeralToken,
                onProgress: options.onProgress,
                retryConfig: options.retryStream,
                options: sseOptions
            });
        } else {
            data = await executeServerActionSSE({
                actionName, args, method,
                actionToken, ephemeralToken,
                onProgress: options.onProgress,
                options: sseOptions
            });
        }

        return {
            data: data as T,
            status: HTTP_STATUS_OK,
            message: undefined,
            helpers: undefined
        };
    }

    const response = await executeServerAction(
        actionName,
        args,
        method,
        actionToken,
        ephemeralToken,
        options
    ) as ActionResponseData;

    const helpers = response._helpers;

    if (!options?.suppressHelpers && helpers && helpers.length > 0) {
        await executeHelpers(helpers, document.body);
    }

    return {
        data: (response.data ?? response) as T,
        status: response.status ?? HTTP_STATUS_OK,
        message: response.message,
        helpers
    };
}
