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

/** Configuration options for whenVisible(). */
export interface WhenVisibleOptions {
    /** IntersectionObserver threshold (0-1, default: 0). */
    threshold?: number | number[];
    /** Root element for intersection (default: viewport). */
    root?: Element | null;
    /** Root margin (default: '0px'). */
    rootMargin?: string;
    /** Only trigger once, then disconnect (default: true). */
    once?: boolean;
}

/** Handle for a cancellable async operation. */
export interface AbortableOperation<T> {
    /** The promise for the operation. */
    promise: Promise<T>;
    /** Aborts the operation. */
    abort: () => void;
    /** The AbortSignal being used. */
    signal: AbortSignal;
}

/** Configuration options for poll(). */
export interface PollingOptions {
    /** Polling interval in milliseconds. */
    interval: number;
    /** Stops polling when this returns true. */
    until?: () => boolean | Promise<boolean>;
    /** Maximum number of polls (default: unlimited). */
    maxAttempts?: number;
    /** Called on each poll with the result. */
    onPoll?: (result: unknown, attempt: number) => void;
    /** Called when polling stops. */
    onStop?: (reason: 'condition' | 'maxAttempts' | 'manual') => void;
}

/** Configuration options for watchMutations(). */
export interface MutationWatchOptions {
    /** Watches for child additions/removals (default: true). */
    childList?: boolean;
    /** Watches attribute changes (default: false). */
    attributes?: boolean;
    /** Watches text content changes (default: false). */
    characterData?: boolean;
    /** Watches all descendants, not just direct children (default: false). */
    subtree?: boolean;
    /** Only watches specific attributes. */
    attributeFilter?: string[];
}

/** Default timeout in ms for requestIdleCallback fallback (when not supported). */
const DEFAULT_IDLE_CALLBACK_TIMEOUT_MS = 50;

/**
 * Resolves a target to an Element.
 *
 * @param target - CSS selector, p-ref name, or Element.
 * @returns The resolved element, or null if not found.
 */
function resolveElement(target: string | Element): Element | null {
    if (target instanceof Element) {
        return target;
    }

    const byRef = document.querySelector(`[p-ref="${target}"]`);
    if (byRef) {
        return byRef;
    }

    return document.querySelector(target);
}

/**
 * Executes a callback when an element becomes visible in the viewport.
 *
 * Uses IntersectionObserver for efficient visibility detection.
 *
 * @param target - Element selector, p-ref name, or Element.
 * @param callback - Function to call when visible.
 * @param options - IntersectionObserver options.
 * @returns Function to stop watching.
 */
export function whenVisible(
    target: string | Element,
    callback: (entry: IntersectionObserverEntry) => void,
    options: WhenVisibleOptions = {}
): () => void {
    const {
        threshold = 0,
        root = null,
        rootMargin = '0px',
        once: triggerOnce = true
    } = options;

    const element = resolveElement(target);
    if (!element) {
        console.warn(`[pk] whenVisible: target "${target}" not found`);
        return () => {
        };
    }

    const observer = new IntersectionObserver(
        (entries) => {
            for (const entry of entries) {
                if (entry.isIntersecting) {
                    callback(entry);
                    if (triggerOnce) {
                        observer.disconnect();
                    }
                } else if (!triggerOnce) {
                    callback(entry);
                }
            }
        },
        {threshold, root, rootMargin}
    );

    observer.observe(element);

    return () => {
        observer.disconnect();
    };
}

/**
 * Creates a cancellable async operation using AbortController.
 *
 * @param operation - Async function that receives an AbortSignal.
 * @returns Object with promise, abort function, and signal.
 */
export function withAbortSignal<T>(
    operation: (signal: AbortSignal) => Promise<T>
): AbortableOperation<T> {
    const controller = new AbortController();

    return {
        promise: operation(controller.signal),
        abort: () => controller.abort(),
        signal: controller.signal
    };
}

/**
 * Creates a timeout that returns a promise.
 *
 * @param ms - Timeout duration in milliseconds.
 * @returns Object with promise and cancel function.
 */
export function timeout(ms: number): { promise: Promise<void>; cancel: () => void } {
    let timeoutId: ReturnType<typeof setTimeout> | null = null;
    let rejectFn: ((reason: Error) => void) | null = null;

    const promise = new Promise<void>((resolve, reject) => {
        rejectFn = reject;
        timeoutId = setTimeout(resolve, ms);
    });

    return {
        promise,
        cancel: () => {
            if (timeoutId !== null) {
                clearTimeout(timeoutId);
                timeoutId = null;
                rejectFn?.(new Error('Timeout cancelled'));
            }
        }
    };
}

/**
 * Polls a function at regular intervals until a condition is met.
 *
 * @param operation - Function to poll.
 * @param options - Polling configuration.
 * @returns Function to stop polling.
 */
export function poll<T>(
    operation: () => T | Promise<T>,
    options: PollingOptions
): () => void {
    const {
        interval,
        until,
        maxAttempts,
        onPoll,
        onStop
    } = options;

    let stopped = false;
    let attempt = 0;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    const runPoll = async (): Promise<void> => {
        if (stopped) {
            return;
        }

        attempt++;

        try {
            const result = await operation();
            onPoll?.(result, attempt);

            if (until) {
                const shouldStop = await until();
                if (shouldStop) {
                    stopped = true;
                    onStop?.('condition');
                    return;
                }
            }

            if (maxAttempts !== undefined && attempt >= maxAttempts) {
                stopped = true;
                onStop?.('maxAttempts');
                return;
            }

            timeoutId = setTimeout(() => {
                void runPoll();
            }, interval);
        } catch (error) {
            console.error('[pk] poll error:', {
                fn: operation.name || 'anonymous',
                interval,
                stopped,
                error
            });
            if (!stopped) {
                timeoutId = setTimeout(() => {
                    void runPoll();
                }, interval);
            }
        }
    };

    void runPoll();

    return () => {
        stopped = true;
        if (timeoutId !== null) {
            clearTimeout(timeoutId);
            timeoutId = null;
        }
        onStop?.('manual');
    };
}

/**
 * Watches for DOM mutations on an element.
 *
 * @param target - Element selector, p-ref name, or Element.
 * @param callback - Function to call on mutations.
 * @param options - MutationObserver options.
 * @returns Function to stop watching.
 */
export function watchMutations(
    target: string | Element,
    callback: (mutations: MutationRecord[]) => void,
    options: MutationWatchOptions = {}
): () => void {
    const {
        childList = true,
        attributes = false,
        characterData = false,
        subtree = false,
        attributeFilter
    } = options;

    const element = resolveElement(target);
    if (!element) {
        console.warn(`[pk] watchMutations: target "${target}" not found`);
        return () => {
        };
    }

    const observer = new MutationObserver(callback);

    observer.observe(element, {
        childList,
        attributes,
        characterData,
        subtree,
        attributeFilter
    });

    return () => {
        observer.disconnect();
    };
}

/**
 * Runs a function when the document is idle.
 *
 * Uses requestIdleCallback when available, falls back to setTimeout.
 *
 * @param task - Function to run during idle time.
 * @param options - RequestIdleCallback options.
 * @returns Function to cancel.
 */
export function whenIdle(
    task: (deadline?: IdleDeadline) => void,
    options?: IdleRequestOptions
): () => void {
    if ('requestIdleCallback' in window) {
        const id = window.requestIdleCallback(task, options);
        return () => window.cancelIdleCallback(id);
    }

    const timeoutMs = options?.timeout ?? DEFAULT_IDLE_CALLBACK_TIMEOUT_MS;
    const id = setTimeout(() => task(), timeoutMs);
    return () => clearTimeout(id);
}

/**
 * Creates a promise that resolves on the next animation frame.
 *
 * @returns Promise that resolves with the timestamp.
 */
export function nextFrame(): Promise<number> {
    return new Promise(resolve => {
        requestAnimationFrame(resolve);
    });
}

/**
 * Waits for multiple animation frames.
 *
 * @param count - Number of frames to wait.
 * @returns Promise that resolves after the frames.
 */
export async function waitFrames(count: number): Promise<void> {
    for (let i = 0; i < count; i++) {
        await nextFrame();
    }
}

/**
 * Creates a deferred promise that can be resolved/rejected externally.
 *
 * @returns Object with promise, resolve, and reject.
 */
export function deferred<T>(): {
    promise: Promise<T>;
    resolve: (value: T) => void;
    reject: (reason?: unknown) => void;
} {
    let resolve!: (value: T) => void;
    let reject!: (reason?: unknown) => void;

    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });

    return {promise, resolve, reject};
}

/**
 * Executes a function only once, caching the result.
 *
 * @param factory - Function to execute once.
 * @returns Wrapped function that returns the cached result.
 */
export function once<T>(factory: () => T): () => T {
    let called = false;
    let result: T;

    return () => {
        if (!called) {
            called = true;
            result = factory();
        }
        return result;
    };
}
