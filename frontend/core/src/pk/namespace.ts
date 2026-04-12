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

import {partial as _partial} from './partial';
import {bus as _bus} from './bus';

import {
    navigate as _navigate,
    goBack as _goBack,
    goForward as _goForward,
    go as _go,
    currentRoute as _currentRoute,
    buildUrl as _buildUrl,
    updateQuery as _updateQuery,
    registerNavigationGuard as _registerNavigationGuard,
    matchPath as _matchPath,
    extractParams as _extractParams,
} from './navigation';

import {
    formData as _formData,
    validate as _validate,
    resetForm as _resetForm,
    setFormValues as _setFormValues,
} from './form';

import {
    loading as _loading,
    withLoading as _withLoading,
    withRetry as _withRetry,
    debounceAsync as _debounceAsync,
    throttleAsync as _throttleAsync,
} from './ui';

import {
    dispatch as _dispatch,
    listen as _listen,
    listenOnce as _listenOnce,
    waitForEvent as _waitForEvent,
} from './events';

import {debounce as _debounce, throttle as _throttle} from './utils';

import {
    reloadPartial as _reloadPartial,
    reloadGroup as _reloadGroup,
    autoRefresh as _autoRefresh,
    reloadCascade as _reloadCascade,
} from './coordination';

import {
    subscribeToUpdates as _subscribeToUpdates,
    createSSESubscription as _createSSESubscription,
} from './sse';

import {
    whenVisible as _whenVisible,
    withAbortSignal as _withAbortSignal,
    timeout as _timeout,
    poll as _poll,
    watchMutations as _watchMutations,
    whenIdle as _whenIdle,
    nextFrame as _nextFrame,
    waitFrames as _waitFrames,
    deferred as _deferred,
    once as _once,
} from './advanced';

import {
    trace as _trace,
    traceLog as _traceLog,
    traceAsync as _traceAsync,
} from './trace';

import {
    initAutoRefreshObserver as _initAutoRefreshObserver,
    stopAllAutoRefreshers as _stopAllAutoRefreshers,
    getActiveRefresherCount as _getActiveRefresherCount,
} from './autoRefreshObserver';

import {getGlobalPageContext as _getGlobalPageContext} from '../services/PageContext';

import {PPFramework as _PPFramework} from '../core/PPFramework';
import type {PPHelper} from '../services';
import type {ModalRequestOptions} from '../core/ModalManager';
import type {RemoteRenderOptions} from '../core/RemoteRenderer';
import {HookEvent as _HookEvent, type HookEventType, type HookCallback, type HookOptions} from '../services/HookManager';

export type {PartialHandle, PartialReloadOptions} from './partial';
export type {NavigateOptions, RouteInfo, NavigationGuard} from './navigation';
export type {FormDataHandle, ValidationRule, ValidationRules, ValidationResult} from './form';
export type {LoadingOptions, RetryOptions, RetryResult} from './ui';
export type {DispatchOptions, EventListener} from './events';
export type {ReloadOptions, ReloadGroupOptions, AutoRefreshOptions, CascadeNode, CascadeOptions} from './coordination';
export type {SSEOptions, SSESubscription} from './sse';
export type {WhenVisibleOptions, AbortableOperation, PollingOptions, MutationWatchOptions} from './advanced';
export type {TraceConfig, TraceEntry, TraceMetrics} from './trace';

/** Registered ready callbacks, or null once the framework has finished initialising. */
let _readyCallbacks: (() => void)[] | null = [];

/** Whether the framework has finished initialising and is ready for use. */
let _isReady = false;

/** Public API namespace providing access to all Piko client-side utilities. */
export namespace piko {

    /**
     * Gets a handle for a server-side partial by name or element.
     *
     * When given a string, looks up the partial by its partial_name attribute.
     * When given an Element, uses it directly (must have partial_src for reload).
     *
     * @param nameOrElement - The partial name or a partial root element.
     * @returns A handle for reloading the partial.
     */
    export const partial = _partial;

    /**
     * Event bus for cross-component communication.
     *
     * Provides a simple pub/sub mechanism for decoupled communication
     * between different parts of your application.
     */
    export const bus = _bus;

    /** Navigation and routing utilities. */
    export namespace nav {
        /**
         * Navigates to a new URL using SPA navigation.
         */
        export const navigate = _navigate;

        /**
         * Navigates back in history.
         */
        export const back = _goBack;

        /**
         * Navigates forward in history.
         */
        export const forward = _goForward;

        /**
         * Navigates to a specific point in history.
         */
        export const go = _go;

        /**
         * Gets information about the current route.
         */
        export const current = _currentRoute;

        /**
         * Builds a URL with query parameters.
         */
        export const buildUrl = _buildUrl;

        /**
         * Updates query parameters without full navigation.
         */
        export const updateQuery = _updateQuery;

        /**
         * Registers a navigation guard.
         */
        export const guard = _registerNavigationGuard;

        /**
         * Checks if the current path matches a pattern.
         */
        export const matchPath = _matchPath;

        /**
         * Extracts path parameters from the current URL.
         */
        export const extractParams = _extractParams;

        /**
         * Navigates to a URL with event handling (used by compiler-generated link handlers).
         *
         * Unlike `navigate()`, this accepts a DOM event parameter which is passed
         * to the router for preventDefault handling on link clicks.
         *
         * @param url - The URL to navigate to.
         * @param event - The DOM event (e.g., click event on an anchor).
         */
        export function navigateTo(url: string, event?: Event): void {
            void _PPFramework.navigateTo(url, event);
        }
    }

    /** Form handling and validation utilities. */
    export namespace form {
        /**
         * Creates a FormDataHandle for easy form data access.
         */
        export const data = _formData;

        /**
         * Validates a form against a set of rules.
         */
        export const validate = _validate;

        /**
         * Resets a form to its initial state.
         */
        export const reset = _resetForm;

        /**
         * Sets form field values programmatically.
         */
        export const setValues = _setFormValues;
    }

    /** UI state management utilities for loading states and retry logic. */
    export namespace ui {
        /**
         * Wraps a promise with automatic loading state management.
         */
        export const loading = _loading;

        /**
         * Shows a loading indicator while a function executes.
         */
        export const withLoading = _withLoading;

        /**
         * Wraps an async function with retry logic.
         */
        export const withRetry = _withRetry;
    }

    /** Event dispatching and listening utilities. */
    export namespace event {
        /**
         * Dispatches a custom event to a target element.
         */
        export const dispatch = _dispatch;

        /**
         * Listens for custom events on a target element.
         */
        export const listen = _listen;

        /**
         * Listens for an event once, then automatically unsubscribes.
         */
        export const listenOnce = _listenOnce;

        /**
         * Creates a promise that resolves when an event is received.
         */
        export const waitFor = _waitForEvent;
    }

    /** Partial reload coordination utilities. */
    export namespace partials {
        /**
         * Reloads a single partial with retry and debounce support.
         */
        export const reload = _reloadPartial;

        /**
         * Reloads multiple partials in parallel or sequential order.
         */
        export const reloadGroup = _reloadGroup;

        /**
         * Reloads partials in dependency order (cascade).
         */
        export const reloadCascade = _reloadCascade;

        /**
         * Sets up automatic refresh for a partial at specified intervals.
         */
        export const autoRefresh = _autoRefresh;

        /**
         * Performs a remote render/partial reload with full options.
         *
         * This is the low-level render API that accepts full RemoteRenderOptions.
         * For simple partial reloads, prefer `piko.partials.reload()`.
         *
         * @param options - The remote render options.
         * @returns Promise that resolves when the render completes.
         */
        export function render(options: RemoteRenderOptions): Promise<void> {
            return _PPFramework.remoteRender(options);
        }
    }

    /** Server-Sent Events (SSE) utilities. */
    export namespace sse {
        /**
         * Subscribes to server-sent events.
         */
        export const subscribe = _subscribeToUpdates;

        /**
         * Creates an SSE subscription with state tracking.
         */
        export const create = _createSSESubscription;
    }

    /** Timing utilities for rate limiting and scheduling. */
    export namespace timing {
        /**
         * Creates a debounced version of a function.
         */
        export const debounce = _debounce;

        /**
         * Creates a throttled version of a function.
         */
        export const throttle = _throttle;

        /**
         * Creates a debounced version of an async function.
         */
        export const debounceAsync = _debounceAsync;

        /**
         * Creates a throttled version of an async function.
         */
        export const throttleAsync = _throttleAsync;

        /**
         * Creates a timeout promise that can be cancelled.
         */
        export const timeout = _timeout;

        /**
         * Polls a function at specified intervals.
         */
        export const poll = _poll;

        /**
         * Returns a promise that resolves on the next animation frame.
         */
        export const nextFrame = _nextFrame;

        /**
         * Waits for multiple animation frames.
         */
        export const waitFrames = _waitFrames;
    }

    /** Advanced utilities for visibility, mutations, and async operations. */
    export namespace util {
        /**
         * Executes a callback when an element becomes visible.
         */
        export const whenVisible = _whenVisible;

        /**
         * Creates a cancellable async operation.
         */
        export const withAbortSignal = _withAbortSignal;

        /**
         * Watches for DOM mutations on an element.
         */
        export const watchMutations = _watchMutations;

        /**
         * Executes a function when the browser is idle.
         */
        export const whenIdle = _whenIdle;

        /**
         * Creates a deferred promise with external resolve/reject.
         */
        export const deferred = _deferred;

        /**
         * Creates a function that only executes once.
         */
        export const once = _once;
    }

    /** Debugging and performance tracing utilities. */
    export namespace trace {
        /**
         * Enables tracing with optional configuration.
         */
        export const enable = _trace.enable;

        /**
         * Disables tracing.
         */
        export const disable = _trace.disable;

        /**
         * Checks if tracing is enabled.
         */
        export const isEnabled = _trace.isEnabled;

        /**
         * Clears all trace entries.
         */
        export const clear = _trace.clear;

        /**
         * Gets all trace entries.
         */
        export const getEntries = _trace.getEntries;

        /**
         * Gets aggregated metrics from trace entries.
         */
        export const getMetrics = _trace.getMetrics;

        /**
         * Logs a custom trace entry.
         */
        export const log = _traceLog;

        /**
         * Wraps an async function with timing trace.
         *
         * @param name - Name for the trace entry.
         * @param operation - Async function to trace.
         * @returns Promise resolving to the function result.
         */
        export function async<T>(name: string, operation: () => Promise<T>): Promise<T> {
            return _traceAsync(name, operation)();
        }
    }

    /** Declarative auto-refresh management for data-auto-refresh elements. */
    export namespace autoRefreshObserver {
        /**
         * Initialises the auto-refresh observer.
         */
        export const init = _initAutoRefreshObserver;

        /**
         * Stops all active auto-refreshers.
         */
        export const stopAll = _stopAllAutoRefreshers;

        /**
         * Gets the count of active auto-refreshers.
         */
        export const getActiveCount = _getActiveRefresherCount;
    }

    /** Page context access utilities. */
    export namespace context {
        /**
         * Gets the global page context.
         */
        export const get = _getGlobalPageContext;
    }

    /** Hooks API for analytics and integration event tracking. */
    export namespace hooks {
        /**
         * Registers a hook listener for the given event type.
         *
         * @param hookEvent - The event type to listen for.
         * @param callback - The callback to invoke.
         * @param options - Optional hook configuration.
         * @returns An unsubscribe function.
         */
        export function on<E extends HookEventType>(
            hookEvent: E,
            callback: HookCallback<E>,
            options?: HookOptions
        ): () => void {
            return _PPFramework.hooks.on(hookEvent, callback, options);
        }

        /**
         * Registers a hook listener that fires only once.
         *
         * @param hookEvent - The event type to listen for.
         * @param callback - The callback to invoke.
         * @param options - Optional hook configuration.
         * @returns An unsubscribe function.
         */
        export function once<E extends HookEventType>(
            hookEvent: E,
            callback: HookCallback<E>,
            options?: Omit<HookOptions, 'once'>
        ): () => void {
            return _PPFramework.hooks.once(hookEvent, callback, options);
        }

        /**
         * Removes a specific hook by event and ID.
         *
         * @param hookEvent - The event type.
         * @param id - The hook ID to remove.
         */
        export function off(hookEvent: HookEventType, id: string): void {
            _PPFramework.hooks.off(hookEvent, id);
        }

        /**
         * Removes all hooks for an event, or all hooks if no event specified.
         *
         * @param hookEvent - Optional event type to clear.
         */
        export function clear(hookEvent?: HookEventType): void {
            _PPFramework.hooks.clear(hookEvent);
        }

        /** Available hook event constants (e.g. piko.hooks.events.PAGE_VIEW). */
        export const events = _HookEvent;
    }

    /** Analytics tracking utilities for custom event reporting. */
    export namespace analytics {
        /**
         * Sends a custom analytics event to GA4 and/or GTM.
         *
         * This fires an `analytics:track` hook event. If the analytics
         * extension is loaded, it dispatches the event to the configured
         * providers. If analytics is not loaded, the call is silently
         * ignored (graceful degradation).
         *
         * @param eventName - The event name (e.g. "purchase", "sign_up").
         *   For GA4, use recommended event names where possible.
         * @param params - Optional key-value parameters for the event.
         *
         * @example
         * ```ts
         * piko.analytics.track('purchase', {
         *     transaction_id: 'T12345',
         *     value: 99.99,
         *     currency: 'GBP',
         * });
         * ```
         */
        export function track(eventName: string, params?: Record<string, string | number | boolean>): void {
            _PPFramework.emitHook(_HookEvent.ANALYTICS_TRACK, {
                eventName,
                params: params ?? {},
                timestamp: Date.now(),
            });
        }
    }

    /**
     * Registers a helper function for use in templates.
     *
     * Helpers are callable from p-on:* attributes in HTML templates.
     *
     * @param name - The name used to reference the helper in templates.
     * @param helper - The helper function implementation.
     */
    export function registerHelper(name: string, helper: PPHelper): void {
        _PPFramework.registerHelper(name, helper);
    }

    /**
     * Gets runtime configuration for a frontend module.
     *
     * Configuration is set on the Go server via `piko.WithFrontendModule()`
     * and made available to frontend code via this function.
     *
     * @param moduleName - The module name to get configuration for.
     * @returns The module configuration, or null if not found.
     */
    export function getModuleConfig<T = unknown>(moduleName: string): T | null {
        return _PPFramework.getModuleConfig<T>(moduleName);
    }

    /** Loading indicator control. */
    export namespace loader {
        /**
         * Shows or hides the loading indicator bar.
         *
         * @param visible - Whether the loader should be visible.
         */
        export function toggle(visible: boolean): void {
            _PPFramework.toggleLoader(visible);
        }
        /**
         * Updates the loading progress bar percentage.
         *
         * @param percent - The progress percentage (0-100).
         */
        export function progress(percent: number): void {
            _PPFramework.updateProgressBar(percent);
        }
        /**
         * Displays an error message in the error display area.
         *
         * @param message - The error message to display.
         */
        export function error(message: string): void {
            _PPFramework.displayError(message);
        }
        /**
         * Creates a custom loader indicator with the specified colour.
         *
         * @param color - The CSS colour value for the loader.
         */
        export function create(color: string): void {
            _PPFramework.createLoaderIndicator(color);
        }
    }

    /** Modal management. */
    export namespace modal {
        /**
         * Opens a modal by selector with the given options.
         *
         * @param options - The modal configuration options.
         * @returns Promise that resolves when the modal opens.
         */
        export function open(options: ModalRequestOptions): Promise<void> {
            return _PPFramework.openModalIfAvailable(options);
        }
    }

    /** Network status utilities. */
    export namespace network {
        /**
         * Returns whether the browser is currently online.
         *
         * @returns True if the browser reports online status.
         */
        export function isOnline(): boolean {
            return _PPFramework.isOnline;
        }
    }

    /**
     * Patches a partial's HTML content directly.
     *
     * @param html - The HTML string to patch in.
     * @param selector - The CSS selector for the target element.
     */
    export function patchPartial(html: string, selector: string): void {
        _PPFramework.patchPartial(html, selector);
    }

    /**
     * Registers a callback to run when the framework is ready.
     *
     * If the framework is already initialised the callback fires immediately.
     * Multiple callbacks can be registered and they execute in registration order.
     *
     * @param callback - Function to invoke once the framework is ready.
     */
    export function ready(callback: () => void): void {
        if (_isReady) {
            callback();
            return;
        }
        _readyCallbacks?.push(callback);
    }

    /**
     * Marks the framework as ready. Called by main.ts after init().
     */
    export function _markReady(): void {
        _isReady = true;
        const callbacks = _readyCallbacks;
        _readyCallbacks = null;
        if (callbacks) {
            for (const callback of callbacks) {
                callback();
            }
        }
    }

    /** Server action utilities. */
    export namespace actions {
        /**
         * Dispatches a server action from a compiled template event handler.
         *
         * Collects form data from the element's closest form, builds an
         * ActionDescriptor, and dispatches through the ActionExecutor lifecycle.
         *
         * @param actionName - The registered action name (e.g., "contact.send").
         * @param element - The element that triggered the event.
         * @param domEvent - The original DOM event.
         */
        export function dispatch(actionName: string, element: HTMLElement, domEvent?: Event): void {
            _PPFramework.dispatchAction(actionName, element, domEvent);
        }
    }

    /** Helper execution utilities. */
    export namespace helpers {
        /**
         * Executes a registered helper from a compiled template event handler.
         *
         * Parses the action string to extract the helper name and arguments,
         * then calls the registered helper function.
         *
         * @param domEvent - The DOM event that triggered the helper.
         * @param actionString - The helper call string (e.g., "showToast(Hello)").
         * @param element - The element that triggered the event.
         */
        export function execute(domEvent: Event, actionString: string, element: HTMLElement): void {
            _PPFramework.executeHelper(domEvent, actionString, element);
        }
    }

    /** Asset path resolution utilities. */
    export namespace assets {
        /**
         * Transforms an asset source path by prepending the asset serve path.
         *
         * Paths that are already absolute URLs or start with / are returned
         * unchanged. The optional moduleName resolves @/ aliases.
         *
         * @param src - The asset source path.
         * @param moduleName - Optional module name for @/ alias resolution.
         * @returns The resolved asset URL.
         */
        export function resolve(src: string, moduleName?: string): string {
            return _PPFramework.assetSrc(src, moduleName);
        }
    }
}

/** Exposes the piko namespace globally on window for use in script blocks. */
if (typeof window !== 'undefined') {
    (window as unknown as {piko: typeof piko}).piko = piko;
}
