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

import {PPFramework as _PPFramework} from '../core/PPFramework';
import type {NavigateOptions as RouterNavigateOptions} from '../core/Router';

/** Options for the navigate() function. */
export interface NavigateOptions {
    /** Replaces current history entry instead of pushing (default: false). */
    replace?: boolean;
    /** Scrolls to top after navigation (default: true). */
    scroll?: boolean;
    /** State object to pass to history.pushState/replaceState. */
    state?: unknown;
    /** Skips SPA navigation and does a full page load. */
    fullReload?: boolean;
}

/** Information about the current route. */
export interface RouteInfo {
    /** The pathname (e.g., '/products/123'). */
    path: string;
    /** Query parameters as an object. */
    query: Record<string, string>;
    /** Hash fragment without the # (e.g., 'section-1'). */
    hash: string;
    /** Full URL. */
    href: string;
    /** Origin (protocol + host). */
    origin: string;
    /** Returns a specific query parameter value. */
    getParam(name: string): string | null;
    /** Checks whether a query parameter exists. */
    hasParam(name: string): boolean;
    /** Returns all values for a repeated query parameter. */
    getParams(name: string): string[];
}

/** Navigation guard for intercepting navigation events. */
export interface NavigationGuard {
    /** Called before navigation. Returns false to cancel. */
    beforeNavigate?: (to: string, from: string) => boolean | Promise<boolean>;
    /** Called after navigation completes. */
    afterNavigate?: (to: string, from: string) => void;
}

/** Registry of active navigation guards. */
const navigationGuards: NavigationGuard[] = [];

/**
 * Parses a query string into a record mapping each parameter name to its first value.
 * Subsequent values for the same key are ignored; use {@link RouteInfo.getParams} for multi-value access.
 *
 * @param search - The query string to parse (e.g. '?foo=bar&baz=1').
 * @returns Record mapping parameter names to their first values.
 */
function parseQuery(search: string): Record<string, string> {
    const params = new URLSearchParams(search);
    const result: Record<string, string> = {};

    params.forEach((value, key) => {
        if (!(key in result)) {
            result[key] = value;
        }
    });

    return result;
}

/**
 * Returns the framework navigation adapter when available.
 * Accesses PPFramework at call-time only (not module-load time) to avoid circular dependency issues.
 *
 * @returns The navigation adapter, or null if the framework is not initialised.
 */
function getFramework(): {navigate: (url: string, options?: NavigateOptions) => Promise<void>} | null {
    if (typeof _PPFramework.navigateTo === 'function') {
        return {
            navigate: (url: string, options?: NavigateOptions) => {
                const routerOptions: RouterNavigateOptions = {
                    replaceHistory: options?.replace,
                };
                return _PPFramework.navigateTo(url, undefined, routerOptions);
            }
        };
    }
    return null;
}

/**
 * Runs all registered {@link NavigationGuard.beforeNavigate} guards sequentially.
 * Returns false if any guard vetoes the navigation.
 *
 * @param url - The destination URL.
 * @param currentUrl - The URL being navigated away from.
 * @returns Whether navigation is permitted.
 */
async function runBeforeNavigateGuards(url: string, currentUrl: string): Promise<boolean> {
    for (const guard of navigationGuards) {
        if (!guard.beforeNavigate) {
            continue;
        }
        const allowed = await guard.beforeNavigate(url, currentUrl);
        if (!allowed) {
            return false;
        }
    }
    return true;
}

/**
 * Performs a full page reload by setting or replacing the browser location.
 *
 * @param url - The URL to load.
 * @param replace - Whether to replace the current history entry.
 */
function performFullReload(url: string, replace: boolean): void {
    if (replace) {
        window.location.replace(url);
    } else {
        window.location.href = url;
    }
}

/**
 * Navigates using the History API, optionally scrolling to top and dispatching a popstate event.
 *
 * @param url - The URL to push or replace.
 * @param replace - Whether to replace instead of push.
 * @param scroll - Whether to scroll to the top of the page.
 * @param state - State object for the history entry.
 */
function performHistoryNavigation(url: string, replace: boolean, scroll: boolean, state: unknown): void {
    const method = replace ? 'replaceState' : 'pushState';
    window.history[method](state, '', url);

    if (scroll) {
        window.scrollTo(0, 0);
    }

    window.dispatchEvent(new PopStateEvent('popstate', {state}));
}

/**
 * Invokes all registered {@link NavigationGuard.afterNavigate} callbacks.
 *
 * @param url - The destination URL.
 * @param currentUrl - The URL navigated away from.
 */
function runAfterNavigateGuards(url: string, currentUrl: string): void {
    for (const guard of navigationGuards) {
        guard.afterNavigate?.(url, currentUrl);
    }
}

/**
 * Navigates to a URL using the framework or History API.
 *
 * @param url - URL to navigate to.
 * @param options - Navigation options.
 */
export async function navigate(url: string, options: NavigateOptions = {}): Promise<void> {
    const {replace = false, scroll = true, state = null, fullReload = false} = options;
    const currentUrl = window.location.href;

    const allowed = await runBeforeNavigateGuards(url, currentUrl);
    if (!allowed) {
        return;
    }

    if (fullReload) {
        performFullReload(url, replace);
        return;
    }

    const framework = getFramework();
    if (framework) {
        await framework.navigate(url, {replace, scroll});
    } else {
        performHistoryNavigation(url, replace, scroll, state);
    }

    runAfterNavigateGuards(url, currentUrl);
}

/**
 * Navigates back in history.
 */
export function goBack(): void {
    window.history.back();
}

/**
 * Navigates forward in history.
 */
export function goForward(): void {
    window.history.forward();
}

/**
 * Navigates to a specific point in history.
 *
 * @param delta - Number of steps (negative for back, positive for forward).
 */
export function go(delta: number): void {
    window.history.go(delta);
}

/**
 * Returns information about the current route.
 *
 * @returns RouteInfo object with path, query, hash, and helper methods.
 */
export function currentRoute(): RouteInfo {
    const location = window.location;
    const searchParams = new URLSearchParams(location.search);

    return {
        path: location.pathname,
        query: parseQuery(location.search),
        hash: location.hash.replace(/^#/, ''),
        href: location.href,
        origin: location.origin,

        getParam(name: string): string | null {
            return searchParams.get(name);
        },

        hasParam(name: string): boolean {
            return searchParams.has(name);
        },

        getParams(name: string): string[] {
            return searchParams.getAll(name);
        }
    };
}

/**
 * Builds a URL with query parameters.
 *
 * @param path - Base path.
 * @param params - Query parameters to add.
 * @param hash - Optional hash fragment.
 * @returns Constructed URL string.
 */
export function buildUrl(
    path: string,
    params?: Record<string, string | number | boolean | null | undefined>,
    hash?: string
): string {
    const url = new URL(path, window.location.origin);

    if (params) {
        for (const [key, value] of Object.entries(params)) {
            if (value !== null && value !== undefined) {
                url.searchParams.set(key, String(value));
            }
        }
    }

    if (hash) {
        url.hash = hash;
    }

    return url.pathname + url.search + url.hash;
}

/**
 * Updates query parameters without full navigation.
 *
 * Useful for filters, pagination, etc.
 *
 * @param params - Parameters to update (null to remove).
 * @param options - Navigation options.
 */
export async function updateQuery(
    params: Record<string, string | null | undefined>,
    options: NavigateOptions = {}
): Promise<void> {
    const url = new URL(window.location.href);

    for (const [key, value] of Object.entries(params)) {
        if (value === null || value === undefined) {
            url.searchParams.delete(key);
        } else {
            url.searchParams.set(key, value);
        }
    }

    await navigate(url.pathname + url.search + url.hash, {
        ...options,
        scroll: options.scroll ?? false
    });
}

/**
 * Registers a navigation guard.
 *
 * Guards can intercept navigation before it happens and optionally cancel it.
 *
 * @param guard - Guard object with beforeNavigate and/or afterNavigate callbacks.
 * @returns Function to remove the guard.
 */
export function registerNavigationGuard(guard: NavigationGuard): () => void {
    navigationGuards.push(guard);

    return () => {
        const index = navigationGuards.indexOf(guard);
        if (index > -1) {
            navigationGuards.splice(index, 1);
        }
    };
}

/**
 * Checks whether the current path matches a pattern.
 *
 * @param pattern - Path pattern (supports * wildcard and :param placeholders).
 * @returns True if the current path matches.
 */
export function matchPath(pattern: string): boolean {
    const currentPath = window.location.pathname;

    const regexPattern = pattern
        .replace(/[.+?^${}()|[\]\\]/g, '\\$&')
        .replace(/:([^/]+)/g, '([^/]+)')
        .replace(/\*/g, '.*');

    const regex = new RegExp(`^${regexPattern}$`);
    return regex.test(currentPath);
}

/**
 * Extracts path parameters from the current URL based on a pattern.
 *
 * @param pattern - Path pattern with :param placeholders.
 * @returns Object with extracted parameters, or null if the pattern does not match.
 */
export function extractParams(pattern: string): Record<string, string> | null {
    const currentPath = window.location.pathname;

    const paramNames: string[] = [];
    const regexPattern = pattern
        .replace(/[.+?^${}()|[\]\\]/g, '\\$&')
        .replace(/:([^/]+)/g, (_, name: string) => {
            paramNames.push(name);
            return '([^/]+)';
        })
        .replace(/\*/g, '.*');

    const regex = new RegExp(`^${regexPattern}$`);
    const match = currentPath.match(regex);

    if (!match) {
        return null;
    }

    const params: Record<string, string> = {};
    paramNames.forEach((name, index) => {
        params[name] = match[index + 1];
    });

    return params;
}
