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

/** Hook event type constants for framework lifecycle events. */
export const HookEvent = {
    /** Fired when the framework is fully initialised. */
    FRAMEWORK_READY: 'framework:ready',

    /** Fired on page view (initial load and each navigation). */
    PAGE_VIEW: 'page:view',

    /** Fired before SPA navigation begins. */
    NAVIGATION_START: 'navigation:start',

    /** Fired after SPA navigation completes successfully. */
    NAVIGATION_COMPLETE: 'navigation:complete',

    /** Fired when navigation fails. */
    NAVIGATION_ERROR: 'navigation:error',

    /** Fired when a server action is triggered. */
    ACTION_START: 'action:start',

    /** Fired when a server action completes. */
    ACTION_COMPLETE: 'action:complete',

    /** Fired when a modal opens. */
    MODAL_OPEN: 'modal:open',

    /** Fired when a modal closes. */
    MODAL_CLOSE: 'modal:close',

    /** Fired after a partial render completes. */
    PARTIAL_RENDER: 'partial:render',

    /** Fired when a form becomes dirty (has unsaved changes). */
    FORM_DIRTY: 'form:dirty',

    /** Fired when a form becomes clean (changes saved or reset). */
    FORM_CLEAN: 'form:clean',

    /** Fired when network connection is restored. */
    NETWORK_ONLINE: 'network:online',

    /** Fired when network connection is lost. */
    NETWORK_OFFLINE: 'network:offline',

    /** Fired when an error occurs. */
    ERROR: 'error',

    /** Fired when user code requests a custom analytics event via piko.analytics.track(). */
    ANALYTICS_TRACK: 'analytics:track',
} as const;

/** Union type of all hook event string values. */
export type HookEventType = typeof HookEvent[keyof typeof HookEvent];

/** Payload for the framework:ready event. */
export interface FrameworkReadyPayload {
    /** Framework version string. */
    version: string;
    /** Time taken to load the framework in milliseconds. */
    loadTime: number;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for the page:view event. */
export interface PageViewPayload {
    /** Current page URL. */
    url: string;
    /** Page title. */
    title: string;
    /** Referring URL. */
    referrer: string;
    /** Whether this is the initial page load. */
    isInitialLoad: boolean;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for navigation:start events. */
export interface NavigationPayload {
    /** Target URL. */
    url: string;
    /** URL of the previous page. */
    previousUrl?: string;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for navigation:complete events. */
export interface NavigationCompletePayload extends NavigationPayload {
    /** Navigation duration in milliseconds. */
    duration: number;
}

/** Payload for navigation:error events. */
export interface NavigationErrorPayload {
    /** Target URL that failed. */
    url: string;
    /** Error message. */
    error: string;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for action:start events. */
export interface ActionPayload {
    /** Action identifier. */
    action: string;
    /** HTTP method used. */
    method: string;
    /** Tag name of the triggering element. */
    elementTag: string;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for action:complete events. */
export interface ActionCompletePayload extends ActionPayload {
    /** Whether the action succeeded. */
    success: boolean;
    /** HTTP status code of the response. */
    statusCode: number;
    /** Action duration in milliseconds. */
    duration: number;
    /** Whether the action failed due to validation. */
    validationFailed?: boolean;
}

/** Payload for modal:open and modal:close events. */
export interface ModalPayload {
    /** Identifier of the modal. */
    modalId?: string;
    /** URL associated with the modal. */
    url?: string;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for partial:render events. */
export interface PartialRenderPayload {
    /** Source URL of the partial. */
    src: string;
    /** Selector for the patch location. */
    patchLocation: string;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for form:dirty and form:clean events. */
export interface FormStatePayload {
    /** Identifier of the form. */
    formId?: string;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for network:online and network:offline events. */
export interface NetworkPayload {
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for error events. */
export interface ErrorPayload {
    /** Error message. */
    message: string;
    /** Error type classification. */
    type: string;
    /** Context in which the error occurred. */
    context: 'navigation' | 'action' | 'render' | 'unknown';
    /** Error stack trace. */
    stack?: string;
    /** URL associated with the error. */
    url?: string;
    /** Unix timestamp of the event. */
    timestamp: number;
}

/** Payload for analytics:track events fired by piko.analytics.track(). */
export interface AnalyticsTrackPayload {
    /** The custom event name (e.g. "purchase", "sign_up", "add_to_cart"). */
    eventName: string;
    /** Key-value parameters to send with the event. */
    params: Record<string, string | number | boolean>;
    /** Unix timestamp of when track() was called. */
    timestamp: number;
}

/** Maps hook event types to their corresponding payload types. */
export interface HookPayloads {
    [HookEvent.FRAMEWORK_READY]: FrameworkReadyPayload;
    [HookEvent.PAGE_VIEW]: PageViewPayload;
    [HookEvent.NAVIGATION_START]: NavigationPayload;
    [HookEvent.NAVIGATION_COMPLETE]: NavigationCompletePayload;
    [HookEvent.NAVIGATION_ERROR]: NavigationErrorPayload;
    [HookEvent.ACTION_START]: ActionPayload;
    [HookEvent.ACTION_COMPLETE]: ActionCompletePayload;
    [HookEvent.MODAL_OPEN]: ModalPayload;
    [HookEvent.MODAL_CLOSE]: ModalPayload;
    [HookEvent.PARTIAL_RENDER]: PartialRenderPayload;
    [HookEvent.FORM_DIRTY]: FormStatePayload;
    [HookEvent.FORM_CLEAN]: FormStatePayload;
    [HookEvent.NETWORK_ONLINE]: NetworkPayload;
    [HookEvent.NETWORK_OFFLINE]: NetworkPayload;
    [HookEvent.ERROR]: ErrorPayload;
    [HookEvent.ANALYTICS_TRACK]: AnalyticsTrackPayload;
}

/** Callback function type for hook subscribers, typed by event. */
export type HookCallback<E extends HookEventType = HookEventType> = (
    payload: E extends keyof HookPayloads ? HookPayloads[E] : unknown
) => void;

/** Registration options for a hook. */
export interface HookOptions {
    /** Hook identifier for debugging and removal. */
    id?: string;
    /** Whether the hook should fire only once then auto-remove. */
    once?: boolean;
    /** Execution priority (higher values execute earlier). Defaults to 0. */
    priority?: number;
}

/** Represents a queued hook registration from the pre-init queue. */
export interface QueuedHook {
    /** The event to subscribe to. */
    event: HookEventType;
    /** The callback to invoke. */
    callback: HookCallback;
    /** Optional registration options. */
    options?: HookOptions;
}

/** Public hooks API exposed on PPFramework.hooks for external consumers. */
export interface HooksAPI {
    /**
     * Registers a hook for an event.
     * @param event - The event to subscribe to.
     * @param callback - The callback to invoke when the event fires.
     * @param options - The optional registration options.
     * @returns An unsubscribe function.
     */
    on<E extends HookEventType>(
        event: E,
        callback: HookCallback<E>,
        options?: HookOptions
    ): () => void;

    /**
     * Registers a hook that fires only once then auto-removes.
     * @param event - The event to subscribe to.
     * @param callback - The callback to invoke when the event fires.
     * @param options - The optional registration options (excluding once).
     * @returns An unsubscribe function.
     */
    once<E extends HookEventType>(
        event: E,
        callback: HookCallback<E>,
        options?: Omit<HookOptions, 'once'>
    ): () => void;

    /**
     * Removes a specific hook by its identifier.
     * @param event - The event the hook is registered for.
     * @param id - The hook identifier.
     */
    off(event: HookEventType, id: string): void;

    /**
     * Removes all hooks for an event, or all hooks if no event is specified.
     * @param event - An optional event to clear hooks for.
     */
    clear(event?: HookEventType): void;

    /** Whether the framework is ready. */
    readonly ready: boolean;

    /** Available hook event constants. */
    readonly events: typeof HookEvent;
}

/** Manages hook registration, emission, and lifecycle for analytics and external integrations. */
export interface HookManager {
    /** Public API to expose on PPFramework.hooks. */
    api: HooksAPI;

    /**
     * Emits a hook event to all registered subscribers.
     * @param event - The event to emit.
     * @param payload - The typed payload for the event.
     */
    emit<E extends HookEventType>(event: E, payload: HookPayloads[E]): void;

    /** Processes queued hooks from window.__PP_HOOKS_QUEUE__ after initialisation. */
    processQueue(): void;

    /** Marks the framework as ready. */
    setReady(): void;
}

/** Extends the Window interface with the pre-init hooks queue. */
interface WindowWithHooksQueue extends Window {
    __PP_HOOKS_QUEUE__?: QueuedHook[];
}

/** Tracks the internal state of a registered hook. */
interface RegisteredHook {
    /** Unique identifier for this hook. */
    id: string;
    /** The callback function. */
    callback: HookCallback;
    /** Whether the hook should auto-remove after firing. */
    once: boolean;
    /** Execution priority (higher values execute earlier). */
    priority: number;
}

/**
 * Build the public hooks API object exposed on PPFramework.hooks.
 * @param registerHook - The function that registers a hook for an event.
 * @param removeHook - The function that removes a hook by its identifier.
 * @param clearHooks - The function that clears hooks for an event, or all hooks.
 * @param getIsReady - The getter that returns whether the framework is ready.
 * @param hookEventRef - The hook event constants object.
 * @returns A fully constructed HooksAPI.
 */
function buildHooksAPI(
    registerHook: <E extends HookEventType>(
        event: E,
        callback: HookCallback<E>,
        options?: HookOptions
    ) => () => void,
    removeHook: (event: HookEventType, id: string) => void,
    clearHooks: (event?: HookEventType) => void,
    getIsReady: () => boolean,
    hookEventRef: typeof HookEvent
): HooksAPI {
    return {
        on<E extends HookEventType>(
            event: E,
            callback: HookCallback<E>,
            options?: HookOptions
        ): () => void {
            return registerHook(event, callback, options);
        },

        once<E extends HookEventType>(
            event: E,
            callback: HookCallback<E>,
            options?: Omit<HookOptions, 'once'>
        ): () => void {
            return registerHook(event, callback, {...options, once: true});
        },

        off(event: HookEventType, id: string): void {
            removeHook(event, id);
        },

        clear(event?: HookEventType): void {
            clearHooks(event);
        },

        get ready(): boolean {
            return getIsReady();
        },

        events: hookEventRef,
    };
}

/**
 * Process the pre-init hook queue from window.__PP_HOOKS_QUEUE__.
 * @param registerHook - The function that registers a hook for an event.
 */
function processHookQueue(
    registerHook: <E extends HookEventType>(
        event: E,
        callback: HookCallback<E>,
        options?: HookOptions
    ) => () => void
): void {
    const windowWithQueue = window as WindowWithHooksQueue;
    const queue = windowWithQueue.__PP_HOOKS_QUEUE__;
    if (!Array.isArray(queue)) {
        return;
    }

    for (const queued of queue) {
        registerHook(queued.event, queued.callback, queued.options);
    }

    windowWithQueue.__PP_HOOKS_QUEUE__ = [];
}

/**
 * Create a HookManager for analytics and external integrations.
 * Support pre-init queue pattern via window.__PP_HOOKS_QUEUE__.
 * @returns A new HookManager instance.
 */
export function createHookManager(): HookManager {
    const hooks = new Map<HookEventType, RegisteredHook[]>();
    let isReady = false, hookIdCounter = 0;

    /**
     * Generates a unique hook identifier.
     * @returns A string identifier.
     */
    const generateId = (): string => `hook_${++hookIdCounter}`;

    /**
     * Comparator that sorts hooks by descending priority.
     * @param a - The first hook.
     * @param b - The second hook.
     * @returns A negative number if a has higher priority.
     */
    const sortByPriority = (a: RegisteredHook, b: RegisteredHook): number =>
        b.priority - a.priority;

    /**
     * Registers a hook for an event.
     * @param event - The event to subscribe to.
     * @param callback - The callback to invoke.
     * @param options - The optional registration options.
     * @returns An unsubscribe function.
     */
    const registerHook = <E extends HookEventType>(
        event: E,
        callback: HookCallback<E>,
        options: HookOptions = {}
    ): (() => void) => {
        const id = options.id ?? generateId();
        const hook: RegisteredHook = {
            id,
            callback: callback as HookCallback,
            once: options.once ?? false,
            priority: options.priority ?? 0,
        };

        const eventHooks = hooks.get(event) ?? [];
        if (!hooks.has(event)) {
            hooks.set(event, eventHooks);
        }
        eventHooks.push(hook);
        eventHooks.sort(sortByPriority);

        return () => removeHook(event, id);
    };

    /**
     * Removes a hook by its identifier.
     * @param event - The event the hook is registered for.
     * @param id - The hook identifier.
     */
    const removeHook = (event: HookEventType, id: string): void => {
        const eventHooks = hooks.get(event);
        if (!eventHooks) {
            return;
        }
        const index = eventHooks.findIndex(h => h.id === id);
        if (index !== -1) {
            eventHooks.splice(index, 1);
        }
    };

    /**
     * Emits a hook event, invoking all registered callbacks and removing one-time hooks.
     * @param event - The event to emit.
     * @param payload - The typed payload.
     */
    const emit = <E extends HookEventType>(
        event: E,
        payload: HookPayloads[E]
    ): void => {
        const eventHooks = hooks.get(event);
        if (!eventHooks) {
            return;
        }

        const toRemove: string[] = [];

        for (const hook of eventHooks) {
            try {
                hook.callback(payload);
                if (hook.once) {
                    toRemove.push(hook.id);
                }
            } catch (error) {
                console.error(
                    `HookManager: Error in hook "${hook.id}" for event "${event}":`,
                    error
                );
            }
        }

        for (const id of toRemove) {
            removeHook(event, id);
        }
    };

    const api = buildHooksAPI(
        registerHook,
        removeHook,
        (event?: HookEventType) => { if (event) { hooks.delete(event); } else { hooks.clear(); } },
        () => isReady,
        HookEvent
    );

    return {
        api,
        emit,
        processQueue(): void {
            processHookQueue(registerHook);
        },
        setReady(): void {
            isReady = true;
        },
    };
}
