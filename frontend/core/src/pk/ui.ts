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

/** Configuration options for the loading function. */
export interface LoadingOptions {
    /** CSS class to add during loading (default: 'loading'). */
    className?: string;
    /** Text to show during loading (replaces innerText). */
    text?: string;
    /** Disables the element during loading (default: true). */
    disabled?: boolean;
    /** Minimum duration in ms to show loading state (prevents flicker). */
    minDuration?: number;
    /** Called when loading starts. */
    onStart?: () => void;
    /** Called when loading ends (success or error). */
    onEnd?: () => void;
}

/** Configuration options for the withRetry function. */
export interface RetryOptions {
    /** Number of retry attempts (default: 3). */
    attempts?: number;
    /** Backoff strategy: 'linear' or 'exponential' (default: 'exponential'). */
    backoff?: 'linear' | 'exponential';
    /** Initial delay in ms (default: 1000). */
    delay?: number;
    /** Maximum delay in ms (default: 30000). */
    maxDelay?: number;
    /** Called before each retry attempt. */
    onRetry?: (attempt: number, error: Error) => void;
    /** Condition to check if retry should happen (default: always retry). */
    shouldRetry?: (error: Error) => boolean;
}

/** Result of a successful retry operation. */
export interface RetryResult<T> {
    /** The successful result. */
    data: T;
    /** Number of attempts it took. */
    attempts: number;
}

/**
 * Resolves a target to an HTMLElement.
 *
 * @param target - CSS selector, p-ref name, or HTMLElement.
 * @returns The resolved element, or null if not found.
 */
function resolveElement(target: string | HTMLElement): HTMLElement | null {
    if (target instanceof HTMLElement) {
        return target;
    }

    const byRef = document.querySelector(`[p-ref="${target}"]`);
    if (byRef instanceof HTMLElement) {
        return byRef;
    }

    const bySelector = document.querySelector(target);
    if (bySelector instanceof HTMLElement) {
        return bySelector;
    }

    return null;
}

/**
 * Calculates delay for a retry attempt.
 *
 * @param attempt - Current attempt number.
 * @param options - Backoff configuration.
 * @returns Delay in milliseconds.
 */
function calculateDelay(
    attempt: number,
    options: Required<Pick<RetryOptions, 'backoff' | 'delay' | 'maxDelay'>>
): number {
    let calculatedDelay: number;

    if (options.backoff === 'linear') {
        calculatedDelay = options.delay * attempt;
    } else {
        calculatedDelay = options.delay * Math.pow(2, attempt - 1);
    }

    return Math.min(calculatedDelay, options.maxDelay);
}

/**
 * Sleeps for a specified duration.
 *
 * @param ms - Duration in milliseconds.
 * @returns Promise that resolves after the duration.
 */
function sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}

/** Union of HTML element types that support the disabled property. */
type DisableableElement = HTMLButtonElement | HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;

/**
 * Checks if an element supports the disabled property.
 *
 * @param el - Element to check.
 * @returns True if the element supports the disabled property.
 */
function isDisableableElement(el: HTMLElement): el is DisableableElement {
    return 'disabled' in el;
}

/**
 * Captures the original state of an element before applying loading state.
 *
 * @param element - Element to capture state from.
 * @returns Object containing the original text and disabled state.
 */
function captureOriginalState(element: HTMLElement): { text: string; disabled: boolean } {
    return {
        text: element.innerText,
        disabled: isDisableableElement(element) ? element.disabled : false
    };
}

/**
 * Applies loading state to an element.
 *
 * @param element - Element to apply state to.
 * @param className - CSS class to add.
 * @param text - Optional text to set.
 * @param disabled - Whether to disable the element.
 */
function applyLoadingState(
    element: HTMLElement,
    className: string,
    text: string | undefined,
    disabled: boolean
): void {
    element.classList.add(className);
    if (text !== undefined) {
        element.innerText = text;
    }
    if (disabled && isDisableableElement(element)) {
        element.disabled = true;
    }
}

/**
 * Restores the original state of an element after loading completes.
 *
 * @param element - Element to restore.
 * @param className - CSS class to remove.
 * @param original - Original state to restore.
 * @param textWasSet - Whether text was overridden during loading.
 * @param disabled - Whether the element was disabled during loading.
 */
function restoreOriginalState(
    element: HTMLElement,
    className: string,
    original: { text: string; disabled: boolean },
    textWasSet: boolean,
    disabled: boolean
): void {
    element.classList.remove(className);
    if (textWasSet) {
        element.innerText = original.text;
    }
    if (disabled && isDisableableElement(element)) {
        element.disabled = original.disabled;
    }
}

/**
 * Wraps a promise with visual loading state.
 *
 * Applies a CSS class and optionally disables the element while the promise is pending.
 *
 * @param target - CSS selector, p-ref name, or element to apply loading state to.
 * @param promise - Promise to await.
 * @param options - Loading state options.
 * @returns The resolved promise value.
 */
export async function loading<T>(
    target: string | HTMLElement,
    promise: Promise<T>,
    options: LoadingOptions = {}
): Promise<T> {
    const {className = 'loading', text, disabled = true, minDuration = 0, onStart, onEnd} = options;

    const element = resolveElement(target);
    if (!element) {
        console.warn(`[pk] loading: target "${target}" not found`);
        return promise;
    }

    const originalState = captureOriginalState(element);
    applyLoadingState(element, className, text, disabled);
    onStart?.();

    const startTime = Date.now();

    try {
        const result = await promise;

        const elapsed = Date.now() - startTime;
        if (elapsed < minDuration) {
            await sleep(minDuration - elapsed);
        }

        return result;
    } finally {
        restoreOriginalState(element, className, originalState, text !== undefined, disabled);
        onEnd?.();
    }
}

/**
 * Wraps an async function with retry logic.
 *
 * Automatically retries failed operations with configurable backoff.
 *
 * @param operation - Async function to retry.
 * @param options - Retry configuration.
 * @returns The successful result.
 * @throws The last error if all attempts fail.
 */
export async function withRetry<T>(
    operation: () => Promise<T>,
    options: RetryOptions = {}
): Promise<T> {
    const {
        attempts = 3,
        backoff = 'exponential',
        delay = 1000,
        maxDelay = 30000,
        onRetry,
        shouldRetry = () => true
    } = options;

    let lastError: Error | null = null;

    for (let attempt = 1; attempt <= attempts; attempt++) {
        try {
            return await operation();
        } catch (error) {
            lastError = error instanceof Error ? error : new Error(String(error));

            if (attempt === attempts || !shouldRetry(lastError)) {
                throw lastError;
            }

            onRetry?.(attempt, lastError);

            const waitTime = calculateDelay(attempt, {backoff, delay, maxDelay});
            await sleep(waitTime);
        }
    }

    throw lastError ?? new Error('Retry failed');
}

/**
 * Shows a loading indicator on an element while a function executes.
 *
 * Similar to `loading()` but accepts a function instead of a promise,
 * allowing for more complex operations.
 *
 * @param target - Element selector, p-ref name, or HTMLElement.
 * @param operation - Async function to execute.
 * @param options - Loading state options.
 * @returns The function result.
 */
export async function withLoading<T>(
    target: string | HTMLElement,
    operation: () => Promise<T>,
    options: LoadingOptions = {}
): Promise<T> {
    return loading(target, operation(), options);
}

/**
 * Creates a debounced version of a function.
 *
 * The function will only execute after the specified delay has passed
 * without any new calls.
 *
 * @param handler - Function to debounce.
 * @param delay - Delay in milliseconds.
 * @returns Debounced function with cancel method.
 */
export function debounceAsync<T extends unknown[], R>(
    handler: (...args: T) => Promise<R>,
    delay: number
): ((...args: T) => Promise<R>) & { cancel: () => void } {
    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    let pendingResolve: ((value: R) => void) | null = null;
    let pendingReject: ((error: Error) => void) | null = null;

    const debounced = (...args: T): Promise<R> => {
        if (timeoutId !== null) {
            clearTimeout(timeoutId);
        }

        return new Promise<R>((resolve, reject) => {
            pendingResolve = resolve;
            pendingReject = reject;

            timeoutId = setTimeout(() => {
                void (async () => {
                    timeoutId = null;
                    try {
                        const result = await handler(...args);
                        pendingResolve?.(result);
                    } catch (error) {
                        pendingReject?.(error instanceof Error ? error : new Error(String(error)));
                    }
                })();
            }, delay);
        });
    };

    debounced.cancel = (): void => {
        if (timeoutId !== null) {
            clearTimeout(timeoutId);
            timeoutId = null;
        }
    };

    return debounced;
}

/**
 * Creates a throttled version of an async function.
 *
 * The function will execute immediately on the first call, then ignore
 * subsequent calls until the delay has passed.
 *
 * @param handler - Function to throttle.
 * @param delay - Minimum delay between executions in milliseconds.
 * @returns Throttled function.
 */
export function throttleAsync<T extends unknown[], R>(
    handler: (...args: T) => Promise<R>,
    delay: number
): (...args: T) => Promise<R | undefined> {
    let lastCall = 0;
    let pendingPromise: Promise<R> | null = null;

    return async (...args: T): Promise<R | undefined> => {
        const now = Date.now();

        if (now - lastCall >= delay) {
            lastCall = now;
            pendingPromise = handler(...args);
            return pendingPromise;
        }

        if (pendingPromise) {
            return pendingPromise;
        }

        return undefined;
    };
}
