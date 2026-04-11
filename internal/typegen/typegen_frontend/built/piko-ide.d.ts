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

/** Cleanup function returned by subscriptions and watchers. */
declare type CleanupFn = () => void;

/** Handle for a cancellable async operation. */
declare interface AbortableOperation<T> {
    /** The promise for the operation. */
    promise: Promise<T>;
    /** Aborts the operation. */
    abort: () => void;
    /** The AbortSignal being used. */
    signal: AbortSignal;
}

/** Options for programmatic navigation. */
declare interface PKNavigateOptions {
    /** Replace current history entry instead of pushing (default: false). */
    replace?: boolean;
    /** Scroll to top after navigation (default: true). */
    scroll?: boolean;
    /** State object to pass to history.pushState/replaceState. */
    state?: unknown;
    /** Skip SPA navigation and do a full page load. */
    fullReload?: boolean;
}

/** Information about the current route. */
declare interface RouteInfo {
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
    /** Get a specific query parameter. */
    getParam(name: string): string | null;
    /** Check if a query parameter exists. */
    hasParam(name: string): boolean;
    /** Get all values for a repeated query parameter. */
    getParams(name: string): string[];
}

/** Navigation guard for intercepting route changes. */
declare interface NavigationGuard {
    /** Called before navigation. Return false to cancel. */
    beforeNavigate?: (to: string, from: string) => boolean | Promise<boolean>;
    /** Called after navigation completes. */
    afterNavigate?: (to: string, from: string) => void;
}

/** Handle for accessing form data. */
declare interface FormDataHandle {
    /** Convert form data to a plain object. */
    toObject(): Record<string, unknown>;
    /** Get the native FormData object. */
    toFormData(): FormData;
    /** Convert form data to JSON string. */
    toJSON(): string;
    /** Get a specific field value. */
    get(key: string): unknown;
    /** Check if a field exists. */
    has(key: string): boolean;
    /** Get all values for a field (useful for multi-select/checkboxes). */
    getAll(key: string): unknown[];
}

/** Validation rule for a single form field. */
declare interface ValidationRule {
    /** Field is required. */
    required?: boolean;
    /** Minimum length for strings. */
    minLength?: number;
    /** Maximum length for strings. */
    maxLength?: number;
    /** Minimum value for numbers. */
    min?: number;
    /** Maximum value for numbers. */
    max?: number;
    /** Regex pattern to match. */
    pattern?: RegExp | string;
    /** Predefined format: 'email', 'url', 'phone', 'date'. */
    format?: 'email' | 'url' | 'phone' | 'date';
    /** Custom validation function. */
    custom?: (value: unknown) => boolean | string;
    /** Custom error message. */
    message?: string;
}

/** Map of field names to validation rules. */
declare interface ValidationRules {
    [field: string]: ValidationRule;
}

/** Result of form validation. */
declare interface ValidationResult {
    /** Whether all fields passed validation. */
    isValid: boolean;
    /** Error messages by field name. */
    errors: Record<string, string[]>;
    /** Focus the first invalid field. */
    focus(): void;
    /** Get errors for a specific field. */
    getErrors(field: string): string[];
    /** Check if a specific field has errors. */
    hasError(field: string): boolean;
}

/** Options for the loading() and withLoading() functions. */
declare interface LoadingOptions {
    /** CSS class to add during loading (default: 'loading'). */
    className?: string;
    /** Text to show during loading (replaces innerText). */
    text?: string;
    /** Disable the element during loading (default: true). */
    disabled?: boolean;
    /** Minimum duration in ms to show loading state (prevents flicker). */
    minDuration?: number;
    /** Called when loading starts. */
    onStart?: () => void;
    /** Called when loading ends (success or error). */
    onEnd?: () => void;
}

/** Options for the withRetry() function. */
declare interface RetryOptions {
    /** Number of retry attempts (default: 3). */
    attempts?: number;
    /** Backoff strategy (default: 'exponential'). */
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

/** Options for event dispatch. */
declare interface DispatchOptions {
    /** Whether the event bubbles up through the DOM (default: true). */
    bubbles?: boolean;
    /** Whether the event can cross shadow DOM boundary (default: true). */
    composed?: boolean;
}

/** Event listener callback type. */
declare type PKEventListener = (event: CustomEvent) => void;

/** Partial refresh level. */
declare type RefreshLevel = 0 | 1 | 2 | 3;

/** Options for reloadPartial(). */
declare interface ReloadOptions {
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

/** Options for reloadGroup(). */
declare interface ReloadGroupOptions {
    /** Arguments to pass to all partials. */
    args?: Record<string, unknown>;
    /** Reload mode (default: 'parallel'). */
    mode?: 'sequential' | 'parallel';
    /** Share same args across all partials. */
    shareArgs?: boolean;
    /** Auto-toggle pk-loading class. */
    loading?: boolean;
    /** Progress callback. */
    onProgress?: (completed: number, total: number) => void;
}

/** Node in a cascade reload tree. */
declare interface CascadeNode {
    /** Partial name. */
    name: string;
    /** Child partials to reload after this one. */
    children?: CascadeNode[];
}

/** Options for reloadCascade(). */
declare interface CascadeOptions {
    /** Arguments to pass to all partials. */
    args?: Record<string, unknown>;
    /** Called when a node completes. */
    onNodeComplete?: (name: string) => void;
}

/** Options for autoRefresh(). */
declare interface AutoRefreshOptions {
    /** Interval between refreshes in milliseconds. */
    interval: number;
    /** Condition function - skip refresh if returns false. */
    when?: () => boolean;
    /** Error handling (default: 'retry'). */
    onError?: 'retry' | 'stop';
    /** Maximum retry attempts before stopping (default: 3). */
    maxRetries?: number;
}

/** Handle for interacting with a server-side partial. */
declare interface PartialHandle {
    /** The container element for this partial. */
    element: HTMLElement | null;
    /** Reload the partial from the server. */
    reload(data?: Record<string, string | number | boolean>): Promise<void>;
    /** Reload with fine-grained options. */
    reloadWithOptions(options: PartialReloadOptions): Promise<void>;
}

/** Options for PartialHandle.reloadWithOptions(). */
declare interface PartialReloadOptions {
    /** Query parameters to pass to the server. */
    data?: Record<string, string | number | boolean>;
    /** Override the detected refresh level. */
    level?: RefreshLevel;
    /** Override owned attributes for Level 2 (comma-separated). */
    ownedAttrs?: string[];
}

/** Options for SSE subscriptions. */
declare interface SSEOptions {
    /** SSE endpoint URL. */
    url: string;
    /** Custom message handler (overrides default partial reload). */
    onMessage?: (data: unknown) => void;
    /** Error handling (default: 'reconnect'). */
    onError?: 'reconnect' | 'stop';
    /** Reconnection delay in ms (default: 3000). */
    reconnectDelay?: number;
    /** Maximum reconnection attempts (default: 10). */
    maxReconnects?: number;
    /** Event types to listen for (default: ['message']). */
    eventTypes?: string[];
    /** Called when connection opens. */
    onOpen?: () => void;
    /** Called when connection closes. */
    onClose?: () => void;
}

/** Handle for an active SSE subscription. */
declare interface SSESubscription {
    /** Stops listening and closes the connection. */
    unsubscribe: () => void;
    /** Current connection state. */
    readonly state: 'connecting' | 'open' | 'closed' | 'error';
    /** Number of reconnection attempts. */
    readonly reconnectCount: number;
}

/** Options for poll(). */
declare interface PollingOptions {
    /** Polling interval in milliseconds. */
    interval: number;
    /** Stop polling when this returns true. */
    until?: () => boolean | Promise<boolean>;
    /** Maximum number of polls (default: unlimited). */
    maxAttempts?: number;
    /** Called on each poll with the result. */
    onPoll?: (result: unknown, attempt: number) => void;
    /** Called when polling stops. */
    onStop?: (reason: 'condition' | 'maxAttempts' | 'manual') => void;
}

/** Options for whenVisible(). */
declare interface WhenVisibleOptions {
    /** IntersectionObserver threshold (0-1, default: 0). */
    threshold?: number | number[];
    /** Root element for intersection (default: viewport). */
    root?: Element | null;
    /** Root margin (default: '0px'). */
    rootMargin?: string;
    /** Only trigger once, then disconnect (default: true). */
    once?: boolean;
}

/** Options for watchMutations(). */
declare interface MutationWatchOptions {
    /** Watch for child additions/removals (default: true). */
    childList?: boolean;
    /** Watch attribute changes (default: false). */
    attributes?: boolean;
    /** Watch text content changes (default: false). */
    characterData?: boolean;
    /** Watch all descendants, not just direct children (default: false). */
    subtree?: boolean;
    /** Only watch specific attributes. */
    attributeFilter?: string[];
}

/** Configuration options for the tracer. */
declare interface TraceConfig {
    /** Trace partial reloads (default: true). */
    partialReloads: boolean;
    /** Trace event emissions and listeners (default: true). */
    events: boolean;
    /** Trace handler executions (default: true). */
    handlers: boolean;
    /** Trace SSE connections (default: true). */
    sse: boolean;
}

/** A single trace log entry. */
declare interface TraceEntry {
    /** Entry type. */
    type: 'partial' | 'event' | 'handler' | 'sse';
    /** Entry name/identifier. */
    name: string;
    /** Duration in milliseconds (if applicable). */
    duration?: number;
    /** Timestamp when the entry was created. */
    timestamp: number;
    /** Additional metadata. */
    metadata?: Record<string, unknown>;
}

/** Aggregated metrics for a traced item. */
declare interface TraceMetrics {
    /** Number of times this item was traced. */
    count: number;
    /** Average duration in milliseconds. */
    avgDuration: number;
    /** Maximum duration in milliseconds. */
    maxDuration: number;
    /** Minimum duration in milliseconds. */
    minDuration: number;
}

/** Page context for accessing exported functions from PK script modules. */
declare interface PageContext {
    /** Gets an exported function by name. */
    getFunction(name: string): ((...args: unknown[]) => unknown) | undefined;
    /** Checks if a function is exported. */
    hasFunction(name: string): boolean;
    /** Gets all exported function names. */
    getExportedFunctions(): string[];
}

/** Discriminated union of all hook event type strings. */
declare type HookEventType =
    | 'framework:ready'
    | 'page:view'
    | 'navigation:start'
    | 'navigation:complete'
    | 'navigation:error'
    | 'action:start'
    | 'action:complete'
    | 'modal:open'
    | 'modal:close'
    | 'partial:render'
    | 'form:dirty'
    | 'form:clean'
    | 'network:online'
    | 'network:offline'
    | 'error'
    | 'analytics:track';

/** Payload emitted when custom analytics tracking is requested via piko.analytics.track(). */
declare interface AnalyticsTrackPayload {
    /** The custom event name (e.g. "purchase", "sign_up"). */
    eventName: string;
    /** Key-value parameters for the event. */
    params: Record<string, string | number | boolean>;
    /** Unix timestamp in milliseconds. */
    timestamp: number;
}

/** Callback invoked when a hook event fires. */
declare type HookCallback<T = unknown> = (payload: T) => void;

/** Payload emitted when the framework finishes initialisation. */
declare interface FrameworkReadyPayload {
    /** Semantic version of the framework. */
    version: string;
    /** Time in milliseconds the framework took to initialise. */
    loadTime: number;
}

/** Payload emitted on each page view. */
declare interface PageViewPayload {
    /** Current page URL. */
    url: string;
    /** Document title at the time of the view. */
    title: string;
    /** Referrer URL. */
    referrer: string;
    /** Whether this is the first page load rather than a SPA navigation. */
    isInitialLoad: boolean;
    /** Unix timestamp in milliseconds. */
    timestamp: number;
}

/** Payload emitted when a navigation starts. */
declare interface NavigationStartPayload {
    /** Destination URL. */
    url: string;
    /** URL the user is navigating away from. */
    fromUrl: string;
    /** What triggered the navigation. */
    trigger: 'link' | 'popstate' | 'programmatic';
}

/** Payload emitted when a navigation completes successfully. */
declare interface NavigationCompletePayload {
    /** Destination URL. */
    url: string;
    /** URL the user navigated away from. */
    fromUrl: string;
    /** What triggered the navigation. */
    trigger: 'link' | 'popstate' | 'programmatic';
    /** Navigation duration in milliseconds. */
    duration: number;
}

/** Payload emitted when a navigation fails. */
declare interface NavigationErrorPayload {
    /** URL that failed to load. */
    url: string;
    /** URL the user navigated away from. */
    fromUrl: string;
    /** Human-readable error description. */
    error: string;
}

/** Payload emitted when a server action starts. */
declare interface ActionStartPayload {
    /** Name of the server action being invoked. */
    actionName: string;
    /** Element that triggered the action. */
    element: HTMLElement;
    /** Serialised form data when the action originates from a form. */
    formData?: Record<string, unknown>;
}

/** Payload emitted when a server action completes. */
declare interface ActionCompletePayload {
    /** Name of the completed server action. */
    actionName: string;
    /** Element that triggered the action. */
    element: HTMLElement;
    /** Whether the action succeeded. */
    success: boolean;
    /** Action duration in milliseconds. */
    duration: number;
    /** Server response metadata, present on successful requests. */
    response?: {
        /** HTTP status code. */
        status: number;
        /** Optional server message. */
        message?: string;
    };
}

/** Payload emitted when a modal opens or closes. */
declare interface ModalHookPayload {
    /** Identifier of the modal element. */
    modalId: string;
    /** Element that triggered the modal, if any. */
    trigger?: HTMLElement;
}

/** Payload emitted when a partial is rendered. */
declare interface PartialRenderPayload {
    /** CSS selector of the partial container. */
    selector: string;
    /** Source URL the partial was fetched from. */
    src: string;
    /** Render duration in milliseconds. */
    duration: number;
}

/** Payload emitted when a form's dirty state changes. */
declare interface FormStatePayload {
    /** Identifier of the form, when available. */
    formId?: string;
    /** Reference to the form element. */
    formElement: HTMLFormElement;
}

/** Payload emitted when the network status changes. */
declare interface NetworkPayload {
    /** Whether the browser is currently online. */
    online: boolean;
    /** Unix timestamp in milliseconds when the change occurred. */
    timestamp: number;
}

/** Payload emitted when an error occurs. */
declare interface ErrorHookPayload {
    /** Category of the error. */
    type: 'navigation' | 'action' | 'render' | 'network' | 'unknown';
    /** Human-readable error description. */
    message: string;
    /** URL associated with the error, when applicable. */
    url?: string;
    /** Stack trace, when available. */
    stack?: string;
}

declare global {
    namespace piko {
        /**
         * Register a callback to run when the framework is ready.
         * If the framework is already initialised the callback fires immediately.
         * Multiple callbacks can be registered and they execute in registration order.
         * @param callback - Function to invoke once the framework is ready.
         */
        const ready: (callback: () => void) => void;

        /**
         * Get a handle for interacting with a server-side partial.
         * @param name - The partial name.
         */
        const partial: (name: string) => PartialHandle;

        /** Event bus for pub/sub communication between partials. */
        const bus: {
            /** Emit an event to all listeners. */
            emit(event: string, data?: unknown): void;
            /** Subscribe to an event. Returns an unsubscribe function. */
            on(event: string, handler: (data: unknown) => void): () => void;
            /** Subscribe to an event, firing the handler only once. */
            once(event: string, handler: (data: unknown) => void): () => void;
            /** Remove listeners. If event is omitted, removes all listeners. */
            off(event?: string): void;
        };

        /** Programmatic navigation utilities for SPA routing. */
        export namespace nav {
            /**
             * Navigate to a URL via SPA navigation or full page load.
             * @param url - Target URL.
             * @param options - Navigation options.
             */
            const navigate: (url: string, options?: PKNavigateOptions) => Promise<void>;

            /** Navigate back in history. */
            const back: () => void;

            /** Navigate forward in history. */
            const forward: () => void;

            /**
             * Navigate by delta in history stack.
             * @param delta - Number of steps (negative = back).
             */
            const go: (delta: number) => void;

            /** Get information about the current route. */
            const current: () => RouteInfo;

            /**
             * Build a URL with path, query parameters, and hash.
             * @param path - URL path.
             * @param params - Query parameters.
             * @param hash - Hash fragment.
             */
            const buildUrl: (path: string, params?: Record<string, string | number | boolean | null | undefined>, hash?: string) => string;

            /**
             * Update query parameters without a full navigation.
             * @param params - Parameters to update (null removes a parameter).
             * @param options - Navigation options.
             */
            const updateQuery: (params: Record<string, string | null | undefined>, options?: PKNavigateOptions) => Promise<void>;

            /**
             * Register a navigation guard to intercept route changes.
             * @param guard - Guard with beforeNavigate/afterNavigate callbacks.
             * @returns Unregister function.
             */
            const guard: (guard: NavigationGuard) => () => void;

            /**
             * Check if the current path matches a pattern.
             * @param pattern - Path pattern with :param placeholders.
             */
            const matchPath: (pattern: string) => boolean;

            /**
             * Extract parameters from the current path using a pattern.
             * @param pattern - Path pattern with :param placeholders.
             * @returns Parameter map, or null if no match.
             */
            const extractParams: (pattern: string) => Record<string, string> | null;
        }

        /** Form data access and client-side validation. */
        export namespace form {
            /**
             * Get a handle for reading form data.
             * @param selector - CSS selector or form element.
             */
            const data: (selector: string | HTMLFormElement) => FormDataHandle;

            /**
             * Validate a form against rules.
             * @param selector - CSS selector or form element.
             * @param rules - Validation rules per field.
             */
            const validate: (selector: string | HTMLFormElement, rules?: ValidationRules) => ValidationResult;

            /**
             * Reset a form to its default values.
             * @param selector - CSS selector or form element.
             */
            const reset: (selector: string | HTMLFormElement) => void;

            /**
             * Set form field values programmatically.
             * @param selector - CSS selector or form element.
             * @param values - Map of field names to values.
             */
            const setValues: (selector: string | HTMLFormElement, values: Record<string, unknown>) => void;
        }

        /** UI state management - loading indicators and retry logic. */
        export namespace ui {
            /**
             * Show loading state on an element while a promise resolves.
             * @param target - CSS selector or element.
             * @param promise - Promise to track.
             * @param options - Loading options.
             */
            const loading: <T>(target: string | HTMLElement, promise: Promise<T>, options?: LoadingOptions) => Promise<T>;

            /**
             * Show loading state while executing an async function.
             * @param target - CSS selector or element.
             * @param fn - Async function to execute.
             * @param options - Loading options.
             */
            const withLoading: <T>(target: string | HTMLElement, fn: () => Promise<T>, options?: LoadingOptions) => Promise<T>;

            /**
             * Retry an async function with configurable backoff.
             * @param fn - Async function to retry.
             * @param options - Retry options.
             */
            const withRetry: <T>(fn: () => Promise<T>, options?: RetryOptions) => Promise<T>;
        }

        /** Cross-component event communication using CustomEvents. */
        export namespace event {
            /**
             * Dispatch a custom event to a target element.
             * @param target - CSS selector, element, or ref name.
             * @param eventName - Event name.
             * @param detail - Event detail data.
             * @param options - Dispatch options.
             */
            const dispatch: (target: string | HTMLElement, eventName: string, detail?: unknown, options?: DispatchOptions) => void;

            /**
             * Listen for custom events on a target element.
             * @param target - CSS selector, element, ref name, or '*' for document.
             * @param eventName - Event name.
             * @param callback - Event handler.
             * @returns Unsubscribe function.
             */
            const listen: (target: string | HTMLElement | '*', eventName: string, callback: PKEventListener) => () => void;

            /**
             * Listen for a custom event once, then auto-unsubscribe.
             * @param target - CSS selector, element, ref name, or '*' for document.
             * @param eventName - Event name.
             * @param callback - Event handler.
             * @returns Unsubscribe function.
             */
            const listenOnce: (target: string | HTMLElement | '*', eventName: string, callback: PKEventListener) => () => void;

            /**
             * Wait for a custom event, returning a promise.
             * @param target - CSS selector, element, ref name, or '*' for document.
             * @param eventName - Event name.
             * @param timeout - Timeout in ms (rejects on timeout).
             */
            const waitFor: <T = unknown>(target: string | HTMLElement | '*', eventName: string, timeout?: number) => Promise<T>;
        }

        /** Coordinated partial reloads, auto-refresh, and cascade updates. */
        export namespace partials {
            /**
             * Reload a single partial from the server.
             * @param name - Partial name.
             * @param options - Reload options.
             */
            const reload: (name: string, options?: ReloadOptions) => Promise<void>;

            /**
             * Reload multiple partials in parallel or sequentially.
             * @param names - Array of partial names.
             * @param options - Group reload options.
             */
            const reloadGroup: (names: string[], options?: ReloadGroupOptions) => Promise<void>;

            /**
             * Reload partials in a cascading tree structure.
             * @param tree - Root node of the cascade tree.
             * @param options - Cascade options.
             */
            const reloadCascade: (tree: CascadeNode, options?: CascadeOptions) => Promise<void>;

            /**
             * Auto-refresh a partial at regular intervals.
             * @param name - Partial name.
             * @param options - Auto-refresh options.
             * @returns Stop function.
             */
            const autoRefresh: (name: string, options: AutoRefreshOptions) => () => void;
        }

        /** Server-Sent Events for real-time server push updates. */
        export namespace sse {
            /**
             * Subscribe to SSE updates for a partial.
             * @param name - Partial name to reload on updates.
             * @param options - SSE options.
             * @returns Unsubscribe function.
             */
            const subscribe: (name: string, options: SSEOptions) => () => void;

            /**
             * Create a managed SSE subscription with state tracking.
             * @param name - Partial name to reload on updates.
             * @param options - SSE options.
             * @returns Subscription handle.
             */
            const create: (name: string, options: SSEOptions) => SSESubscription;
        }

        /** Rate limiting, debouncing, throttling, and scheduling utilities. */
        export namespace timing {
            /**
             * Create a debounced version of a function.
             * @param fn - Function to debounce.
             * @param ms - Delay in milliseconds.
             */
            const debounce: <T extends (...args: unknown[]) => unknown>(fn: T, ms: number) => (...args: Parameters<T>) => void;

            /**
             * Create a throttled version of a function.
             * @param fn - Function to throttle.
             * @param ms - Minimum interval in milliseconds.
             */
            const throttle: <T extends (...args: unknown[]) => unknown>(fn: T, ms: number) => (...args: Parameters<T>) => void;

            /**
             * Create a debounced version of an async function with cancellation.
             * @param fn - Async function to debounce.
             * @param delay - Delay in milliseconds.
             */
            const debounceAsync: <T extends unknown[], R>(fn: (...args: T) => Promise<R>, delay: number) => ((...args: T) => Promise<R>) & { cancel: () => void };

            /**
             * Create a throttled version of an async function.
             * @param fn - Async function to throttle.
             * @param delay - Minimum interval in milliseconds.
             */
            const throttleAsync: <T extends unknown[], R>(fn: (...args: T) => Promise<R>, delay: number) => (...args: T) => Promise<R | undefined>;

            /**
             * Create a cancellable timeout promise.
             * @param ms - Delay in milliseconds.
             */
            const timeout: (ms: number) => { promise: Promise<void>; cancel: () => void };

            /**
             * Poll a function at regular intervals until a condition is met.
             * @param fn - Function to call on each poll.
             * @param options - Polling options.
             * @returns Stop function.
             */
            const poll: <T>(fn: () => T | Promise<T>, options: PollingOptions) => () => void;

            /**
             * Wait for the next animation frame.
             * @returns Promise resolving with the frame timestamp.
             */
            const nextFrame: () => Promise<number>;

            /**
             * Wait for a specified number of animation frames.
             * @param count - Number of frames to wait.
             */
            const waitFrames: (count: number) => Promise<void>;
        }

        /** Advanced utilities - visibility detection, abort signals, mutations. */
        export namespace util {
            /**
             * Execute a callback when an element becomes visible in the viewport.
             * @param target - CSS selector or element.
             * @param callback - Called when element is visible.
             * @param options - Visibility options.
             * @returns Stop watching function.
             */
            const whenVisible: (target: string | Element, callback: (entry: IntersectionObserverEntry) => void, options?: WhenVisibleOptions) => () => void;

            /**
             * Execute an async function with an automatically managed AbortSignal.
             * @param fn - Async function receiving an AbortSignal.
             * @returns Handle with promise, abort(), and signal.
             */
            const withAbortSignal: <T>(fn: (signal: AbortSignal) => Promise<T>) => AbortableOperation<T>;

            /**
             * Watch for DOM mutations on a target element.
             * @param target - CSS selector or element.
             * @param callback - Called with mutation records.
             * @param options - Mutation observer options.
             * @returns Stop watching function.
             */
            const watchMutations: (target: string | Element, callback: (mutations: MutationRecord[]) => void, options?: MutationWatchOptions) => () => void;

            /**
             * Schedule a function to run during browser idle time.
             * @param fn - Function to run when idle.
             * @param options - Idle request options.
             * @returns Cancel function.
             */
            const whenIdle: (fn: (deadline?: IdleDeadline) => void, options?: IdleRequestOptions) => () => void;

            /**
             * Create a deferred promise with external resolve/reject.
             * @returns Object with promise, resolve, and reject.
             */
            const deferred: <T>() => { promise: Promise<T>; resolve: (value: T) => void; reject: (reason?: unknown) => void };

            /**
             * Create a function that only executes once, caching the result.
             * @param fn - Function to execute once.
             * @returns Cached function.
             */
            const once: <T>(fn: () => T) => () => T;
        }

        /** Opt-in tracing for debugging partial reloads, events, and handlers. */
        export namespace trace {
            /**
             * Enable tracing with optional configuration.
             * @param config - Trace configuration.
             */
            const enable: (config?: Partial<TraceConfig>) => void;

            /** Disable tracing. */
            const disable: () => void;

            /** Check if tracing is currently enabled. */
            const isEnabled: () => boolean;

            /** Clear all trace entries. */
            const clear: () => void;

            /** Get all trace entries. */
            const getEntries: () => TraceEntry[];

            /** Get aggregated metrics by name. */
            const getMetrics: () => Record<string, TraceMetrics>;

            /**
             * Log a custom trace entry.
             * @param name - Entry name/identifier.
             * @param data - Additional data to attach.
             */
            const log: (name: string, data?: unknown) => void;

            /**
             * Trace an async operation, recording its duration.
             * @param name - Operation name.
             * @param fn - Async function to trace.
             */
            function async<T>(name: string, fn: () => Promise<T>): Promise<T>;
        }

        /** Declarative auto-refresh based on pk-auto-refresh attributes. */
        export namespace autoRefreshObserver {
            /** Initialise the observer to watch for pk-auto-refresh attributes. */
            const init: () => void;

            /** Stop all active auto-refreshers. */
            const stopAll: () => void;

            /** Get the number of currently active auto-refreshers. */
            const getActiveCount: () => number;
        }

        /** Access to the page's exported functions and module context. */
        export namespace context {
            /** Get the global page context. */
            const get: () => PageContext;
        }

        /** Lifecycle hooks for analytics, integrations, and side effects. */
        export namespace hooks {
            /**
             * Register a callback for the given hook event.
             * @param event - Hook event to listen for.
             * @param callback - Function invoked when the event fires.
             * @param options - Optional listener configuration.
             * @returns Unsubscribe function that removes the listener.
             */
            const on: <T = unknown>(event: HookEventType, callback: HookCallback<T>, options?: { id?: string }) => () => void;

            /**
             * Register a one-shot callback that is automatically removed after the first invocation.
             * @param event - Hook event to listen for.
             * @param callback - Function invoked once when the event fires.
             * @param options - Optional listener configuration.
             * @returns Unsubscribe function that removes the listener.
             */
            const once: <T = unknown>(event: HookEventType, callback: HookCallback<T>, options?: { id?: string }) => () => void;

            /**
             * Remove a listener by its identifier.
             * @param event - Hook event the listener was registered for.
             * @param id - Identifier of the listener to remove.
             */
            const off: (event: HookEventType, id: string) => void;

            /**
             * Remove all listeners for the given event, or all listeners if no event is specified.
             * @param event - Hook event to clear. Omit to clear all events.
             */
            const clear: (event?: HookEventType) => void;
        }

        /** Custom analytics event tracking. */
        export namespace analytics {
            /**
             * Send a custom analytics event to GA4 and/or GTM.
             * If the analytics extension is not loaded, the call is silently ignored.
             *
             * @param eventName - The event name (e.g. "purchase", "sign_up", "add_to_cart").
             * @param params - Optional key-value parameters for the event.
             *
             * @example
             * piko.analytics.track('purchase', {
             *     transaction_id: 'T12345',
             *     value: 99.99,
             *     currency: 'GBP',
             * });
             */
            const track: (eventName: string, params?: Record<string, string | number | boolean>) => void;
        }
    }
}
