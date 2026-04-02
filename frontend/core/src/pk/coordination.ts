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

import {partial, type PartialHandle} from './partial';
import {debounce} from './utils';

/** Base delay in milliseconds for exponential backoff retry. */
const RETRY_BACKOFF_BASE_MS = 1000;

/** Configuration options for reloadPartial(). */
export interface ReloadOptions {
    /** Arguments to pass to the partial endpoint. */
    args?: Record<string, unknown>;
    /** Auto-toggle pk-loading class (default: true). */
    loading?: boolean;
    /** Show optimistic UI before server responds. */
    optimistic?: unknown;
    /** Called on successful reload. */
    onSuccess?: (html: string) => void;
    /** Called on reload error. */
    onError?: (error: Error) => void;
    /** Number of retry attempts (default: 0). */
    retry?: number;
    /** Debounce in milliseconds. */
    debounce?: number;
}

/** Configuration options for reloadGroup(). */
export interface ReloadGroupOptions {
    /** Arguments to pass to all partials. */
    args?: Record<string, unknown>;
    /** Reload mode: parallel (default) or sequential. */
    mode?: 'sequential' | 'parallel';
    /** Shares the same args across all partials. */
    shareArgs?: boolean;
    /** Auto-toggle pk-loading class. */
    loading?: boolean;
    /** Progress callback. */
    onProgress?: (completed: number, total: number) => void;
}

/** Configuration options for autoRefresh(). */
export interface AutoRefreshOptions {
    /** Interval between refreshes in milliseconds. */
    interval: number;
    /** Condition function; skips refresh if it returns false. */
    when?: () => boolean;
    /** Error handling: 'retry' (default) or 'stop'. */
    onError?: 'retry' | 'stop';
    /** Maximum retry attempts before stopping (default: 3). */
    maxRetries?: number;
}

/** Node in a cascade reload tree. */
export interface CascadeNode {
    /** Partial name. */
    name: string;
    /** Child partials to reload after this one. */
    children?: CascadeNode[];
}

/** Configuration options for reloadCascade(). */
export interface CascadeOptions {
    /** Arguments to pass to all partials. */
    args?: Record<string, unknown>;
    /** Called when a node completes. */
    onNodeComplete?: (name: string) => void;
}

/** Registry of debounced reload functions, keyed by partial name. */
const debounceRegistry = new Map<string, ReturnType<typeof debounce>>();

/**
 * Returns a promise that resolves after the specified number of milliseconds.
 *
 * @param ms - Duration in milliseconds.
 * @returns Promise that resolves after the delay.
 */
function delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Converts an args record with arbitrary values to one containing only string,
 * number, and boolean values suitable for passing to the partial reload endpoint.
 *
 * @param args - The record to convert.
 * @returns The converted record, or undefined if args is falsy.
 */
function toStringRecord(args: Record<string, unknown> | undefined): Record<string, string | number | boolean> | undefined {
    if (!args) {
        return undefined;
    }
    const result: Record<string, string | number | boolean> = {};
    for (const [key, value] of Object.entries(args)) {
        if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
            result[key] = value;
        } else if (value !== null && value !== undefined) {
            result[key] = String(value);
        }
    }
    return result;
}

/**
 * Sanitises an HTML string by parsing it through a template element
 * and stripping script tags and inline event handlers.
 *
 * @param html - The raw HTML string to sanitise.
 * @returns The sanitised HTML string.
 */
function sanitiseHTML(html: string): string {
    const template = document.createElement('template');
    template.innerHTML = html;
    template.content.querySelectorAll('script').forEach(s => s.remove());
    template.content.querySelectorAll('*').forEach(el => {
        for (const attr of Array.from(el.attributes)) {
            if (attr.name.startsWith('on')) {
                el.removeAttribute(attr.name);
            }
        }
    });
    const container = document.createElement('div');
    container.appendChild(template.content.cloneNode(true));
    return container.innerHTML;
}

/**
 * Applies an optimistic update to an element's DOM before the server responds.
 *
 * Accepts either an HTML string (replaces innerHTML) or an object with granular
 * mutation fields such as innerHTML, className, addClass, and removeClass.
 * HTML content is sanitised to prevent XSS by stripping script tags and
 * inline event handlers.
 *
 * @param element - The element to update.
 * @param optimistic - HTML string or mutation descriptor object.
 */
function applyOptimisticUpdate(element: HTMLElement, optimistic: unknown): void {
    if (typeof optimistic === 'string') {
        element.innerHTML = sanitiseHTML(optimistic);
    } else if (typeof optimistic === 'object' && optimistic !== null) {
        const opts = optimistic as Record<string, unknown>;
        if (typeof opts.innerHTML === 'string') {
            element.innerHTML = sanitiseHTML(opts.innerHTML);
        }
        if (typeof opts.className === 'string') {
            element.className = opts.className;
        }
        if (typeof opts.addClass === 'string') {
            element.classList.add(opts.addClass);
        }
        if (typeof opts.removeClass === 'string') {
            element.classList.remove(opts.removeClass);
        }
    }
}

/**
 * Executes a partial reload with retry logic and exponential backoff.
 *
 * Manages loading states, optimistic updates, and success/error callbacks.
 *
 * @param handle - The partial handle to reload.
 * @param options - Reload options (excluding retry and debounce, which are handled by the caller).
 * @param retriesLeft - Number of retry attempts remaining.
 * @param maxRetries - Total retry attempts configured (used for backoff calculation).
 */
async function executeReload(
    handle: PartialHandle,
    options: Omit<ReloadOptions, 'retry' | 'debounce'>,
    retriesLeft: number,
    maxRetries: number
): Promise<void> {
    const {args, loading = true, optimistic, onSuccess, onError} = options;

    try {
        if (optimistic !== undefined && handle.element) {
            applyOptimisticUpdate(handle.element, optimistic);
        }

        if (loading && handle.element) {
            handle.element.classList.add('pk-loading');
            handle.element.setAttribute('aria-busy', 'true');
        }

        await handle.reload(toStringRecord(args));

        onSuccess?.(handle.element?.innerHTML ?? '');
    } catch (error) {
        if (retriesLeft > 0) {
            const backoffDelay = Math.pow(2, maxRetries - retriesLeft) * RETRY_BACKOFF_BASE_MS;
            await delay(backoffDelay);
            return executeReload(handle, options, retriesLeft - 1, maxRetries);
        }

        console.error(`[pk] Failed to reload partial after ${maxRetries} retries:`, {
            name: handle.element?.getAttribute('data-partial'),
            error
        });
        onError?.(error as Error);
        throw error;
    } finally {
        if (loading && handle.element) {
            handle.element.classList.remove('pk-loading');
            handle.element.removeAttribute('aria-busy');
        }
    }
}

/**
 * Reloads a single partial with enhanced options.
 *
 * @param nameOrElement - The partial name (matches partial_name attribute) or a partial root element.
 * @param options - Reload options.
 */
export async function reloadPartial(nameOrElement: string | Element, options: ReloadOptions = {}): Promise<void> {
    const handle = partial(nameOrElement);
    const name = typeof nameOrElement === 'string'
        ? nameOrElement
        : (nameOrElement as HTMLElement).getAttribute('partial_name') ?? 'unknown';

    if (!handle.element) {
        console.warn(`[pk] reloadPartial: partial "${name}" not found`);
        return;
    }

    const {retry = 0, debounce: debounceMs, ...reloadOpts} = options;

    if (debounceMs && debounceMs > 0) {
        let debouncedFn = debounceRegistry.get(name);
        if (!debouncedFn) {
            debouncedFn = debounce(async () => {
                await executeReload(handle, reloadOpts, retry, retry);
            }, debounceMs);
            debounceRegistry.set(name, debouncedFn);
        }
        debouncedFn();
        return;
    }

    return executeReload(handle, reloadOpts, retry, retry);
}

/**
 * Reloads multiple partials in parallel or sequential mode.
 *
 * @param names - Array of partial names to reload.
 * @param options - Group reload options.
 */
export async function reloadGroup(names: string[], options: ReloadGroupOptions = {}): Promise<void> {
    const {mode = 'parallel', args, loading, onProgress} = options;

    if (mode === 'parallel') {
        const promises = names.map((name, index) =>
            reloadPartial(name, {args, loading})
                .then(() => {
                    onProgress?.(index + 1, names.length);
                })
        );
        await Promise.all(promises);
    } else {
        for (let i = 0; i < names.length; i++) {
            await reloadPartial(names[i], {args, loading});
            onProgress?.(i + 1, names.length);
        }
    }
}

/**
 * Sets up automatic refresh for a partial at a specified interval.
 *
 * @param name - The partial name.
 * @param options - Auto-refresh configuration.
 * @returns Cleanup function to stop the auto-refresh.
 */
export function autoRefresh(name: string, options: AutoRefreshOptions): () => void {
    const {interval, when, onError = 'retry', maxRetries = 3} = options;

    let intervalId: ReturnType<typeof setInterval> | null = null;
    let retryCount = 0;
    let stopped = false;

    const refresh = async (): Promise<void> => {
        if (stopped) {
            return;
        }

        if (when && !when()) {
            return;
        }

        try {
            await reloadPartial(name);
            retryCount = 0;
        } catch (error) {
            retryCount++;
            console.warn(`[pk] autoRefresh "${name}" failed (attempt ${retryCount}/${maxRetries}):`, error);

            if (onError === 'stop' || retryCount >= maxRetries) {
                if (intervalId !== null) {
                    clearInterval(intervalId);
                    intervalId = null;
                }
                stopped = true;
                console.warn(`[pk] autoRefresh "${name}" stopped after ${retryCount} failures`);
            }
        }
    };

    intervalId = setInterval(() => void refresh(), interval);

    return () => {
        stopped = true;
        if (intervalId !== null) {
            clearInterval(intervalId);
            intervalId = null;
        }
    };
}

/**
 * Reloads partials in dependency order (parent first, then children).
 *
 * @param tree - The cascade tree root node.
 * @param options - Cascade reload options.
 */
export async function reloadCascade(tree: CascadeNode, options: CascadeOptions = {}): Promise<void> {
    const {args, onNodeComplete} = options;

    await reloadPartial(tree.name, {args});
    onNodeComplete?.(tree.name);

    if (tree.children && tree.children.length > 0) {
        await Promise.all(
            tree.children.map(child => reloadCascade(child, options))
        );
    }
}
