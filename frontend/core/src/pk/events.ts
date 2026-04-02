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

/** Options for dispatching a custom event. */
export interface DispatchOptions {
    /** Whether the event bubbles up through the DOM (default: true). */
    bubbles?: boolean;
    /** Whether the event can cross shadow DOM boundary (default: true). */
    composed?: boolean;
}

/** Callback type for custom event listeners. */
export type EventListener = (event: CustomEvent) => void;

/**
 * Resolves a target specification to an Element.
 *
 * Supports '*' for document, p-ref attribute lookup, CSS selector
 * matching, and direct HTMLElement references.
 *
 * @param target - Target specification string or element.
 * @returns The resolved element, document, or null if not found.
 */
function resolveTarget(target: string | HTMLElement | '*'): Element | Document | null {
    if (target === '*') {
        return document;
    }

    if (target instanceof HTMLElement) {
        return target;
    }

    const byRef = document.querySelector(`[p-ref="${target}"]`);
    if (byRef) {
        return byRef;
    }

    return document.querySelector(target);
}

/**
 * Dispatches a custom event to a target element.
 *
 * Supports targeting by p-ref name, CSS selector, or HTMLElement reference.
 *
 * @param target - Target specification (p-ref name, selector, or element).
 * @param eventName - Event name (e.g., 'item:added', 'cart:updated').
 * @param detail - Event detail payload.
 * @param options - Event options (bubbles, composed).
 */
export function dispatch(
    target: string | HTMLElement,
    eventName: string,
    detail?: unknown,
    options?: DispatchOptions
): void {
    const el = resolveTarget(target);

    if (!el) {
        console.warn(`[pk] dispatch: target "${target}" not found`);
        return;
    }

    const event = new CustomEvent(eventName, {
        detail,
        bubbles: options?.bubbles ?? true,
        composed: options?.composed ?? true
    });

    el.dispatchEvent(event);
}

/**
 * Listens for custom events on a target element.
 *
 * Supports targeting by '*' for document-wide listening, p-ref name,
 * CSS selector, or HTMLElement reference.
 *
 * @param target - Target specification ('*' for document, p-ref name, selector, or element).
 * @param eventName - Event name to listen for.
 * @param callback - Event handler function.
 * @returns Unsubscribe function to remove the listener.
 */
export function listen(
    target: string | HTMLElement | '*',
    eventName: string,
    callback: EventListener
): () => void {
    const el = resolveTarget(target);

    if (!el) {
        console.warn(`[pk] listen: target "${target}" not found`);
        return () => {
        };
    }

    const handler = (e: Event): void => {
        callback(e as CustomEvent);
    };

    el.addEventListener(eventName, handler);

    return () => {
        el.removeEventListener(eventName, handler);
    };
}

/**
 * Listens for an event once, then automatically unsubscribes.
 *
 * @param target - Target specification.
 * @param eventName - Event name to listen for.
 * @param callback - Event handler function.
 * @returns Unsubscribe function (useful if you want to cancel before the event fires).
 */
export function listenOnce(
    target: string | HTMLElement | '*',
    eventName: string,
    callback: EventListener
): () => void {
    const unsubscribe = listen(target, eventName, (event) => {
        unsubscribe();
        callback(event);
    });

    return () => {
        unsubscribe();
    };
}

/**
 * Creates a promise that resolves when an event is received.
 *
 * Useful for async/await patterns with events.
 *
 * @param target - Target specification.
 * @param eventName - Event name to wait for.
 * @param timeout - Optional timeout in ms (rejects if exceeded).
 * @returns Promise that resolves with the event detail.
 */
export function waitForEvent<T = unknown>(
    target: string | HTMLElement | '*',
    eventName: string,
    timeout?: number
): Promise<T> {
    return new Promise((resolve, reject) => {
        let timeoutId: ReturnType<typeof setTimeout> | null = null;

        const unsubscribe = listenOnce(target, eventName, (event) => {
            if (timeoutId) {
                clearTimeout(timeoutId);
            }
            resolve(event.detail as T);
        });

        if (timeout !== undefined && timeout > 0) {
            timeoutId = setTimeout(() => {
                unsubscribe();
                reject(new Error(`Timeout waiting for event "${eventName}"`));
            }, timeout);
        }
    });
}
