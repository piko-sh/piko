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

/** Signature for registered helper functions. */
export type PPHelper = (
    element: HTMLElement,
    event: Event,
    ...args: string[]
) => void | Promise<void>;

/** Discriminated union of all hook event type strings. */
export type HookEventType =
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
    | 'error';

/** Payload emitted when the framework finishes initialisation. */
export interface FrameworkReadyPayload {
    /** Semantic version of the framework. */
    version: string;
    /** Time in milliseconds the framework took to initialise. */
    loadTime: number;
}

/** Payload emitted on each page view. */
export interface PageViewPayload {
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
export interface NavigationPayload {
    /** Destination URL. */
    url: string;
    /** URL the user is navigating away from. */
    fromUrl: string;
    /** What triggered the navigation. */
    trigger: 'link' | 'popstate' | 'programmatic';
}

/** Payload emitted when a navigation completes successfully. */
export interface NavigationCompletePayload {
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
export interface NavigationErrorPayload {
    /** URL that failed to load. */
    url: string;
    /** URL the user navigated away from. */
    fromUrl: string;
    /** Human-readable error description. */
    error: string;
}

/** Payload emitted when a server action starts. */
export interface ActionPayload {
    /** Name of the server action being invoked. */
    actionName: string;
    /** Element that triggered the action. */
    element: HTMLElement;
    /** Serialised form data when the action originates from a form. */
    formData?: Record<string, unknown>;
}

/** Payload emitted when a server action completes. */
export interface ActionCompletePayload {
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
export interface ModalPayload {
    /** Identifier of the modal element. */
    modalId: string;
    /** Element that triggered the modal, if any. */
    trigger?: HTMLElement;
}

/** Payload emitted when a partial is rendered. */
export interface PartialRenderPayload {
    /** CSS selector of the partial container. */
    selector: string;
    /** Source URL the partial was fetched from. */
    src: string;
    /** Render duration in milliseconds. */
    duration: number;
}

/** Payload emitted when a form's dirty state changes. */
export interface FormStatePayload {
    /** Identifier of the form, when available. */
    formId?: string;
    /** Reference to the form element. */
    formElement: HTMLFormElement;
}

/** Payload emitted when the network status changes. */
export interface NetworkPayload {
    /** Whether the browser is currently online. */
    online: boolean;
    /** Unix timestamp in milliseconds when the change occurred. */
    timestamp: number;
}

/** Payload emitted when an error occurs. */
export interface ErrorPayload {
    /** Category of the error. */
    type: 'navigation' | 'action' | 'render' | 'network' | 'unknown';
    /** Human-readable error description. */
    message: string;
    /** URL associated with the error, when applicable. */
    url?: string;
    /** Stack trace, when available. */
    stack?: string;
}

/** Callback invoked when a hook event fires. */
export type HookCallback<T = unknown> = (payload: T) => void;

/** Hooks API exposed by `piko.hooks`. */
export interface HooksAPI {
    /**
     * Registers a callback for the given event type.
     *
     * @param event - Hook event to listen for.
     * @param callback - Function invoked when the event fires.
     */
    on<T = unknown>(event: HookEventType, callback: HookCallback<T>): void;

    /**
     * Removes a previously registered callback.
     *
     * @param event - Hook event to stop listening for.
     * @param callback - Specific callback to remove. Omit to remove all listeners for the event.
     */
    off(event: HookEventType, callback?: HookCallback): void;

    /**
     * Emits a hook event with the given payload.
     *
     * @param event - Hook event to emit.
     * @param payload - Data to pass to registered callbacks.
     */
    emit<T = unknown>(event: HookEventType, payload: T): void;
}

/** Analytics module configuration (Google Analytics GA4 integration). */
export interface AnalyticsModuleConfig {
    /** GA4 measurement IDs (e.g., "G-XXXXXXXXXX"). */
    trackingIds: string[];
    /** Whether to enable verbose console logging for debugging. */
    debugMode?: boolean;
    /** Whether to anonymise user IP addresses before sending to GA. */
    anonymizeIp?: boolean;
    /** Whether to disable automatic page-view tracking on navigation. */
    disablePageView?: boolean;
}

/** Modals module configuration. */
export interface ModalsModuleConfig {
    /** Whether to prevent closing modals with the Escape key (default: false, i.e. Escape closes). */
    disableCloseOnEscape?: boolean;
    /** Whether to prevent closing modals by clicking the backdrop (default: false, i.e. backdrop closes). */
    disableCloseOnBackdrop?: boolean;
}

/** Toasts module configuration. */
export interface ToastsModuleConfig {
    /** Default display duration in milliseconds (default: 5000). */
    defaultDuration?: number;
    /** Toast position: "top-right", "top-left", "bottom-right", "bottom-left", "top-center", "bottom-center". */
    position?: string;
    /** Maximum number of visible toasts (default: 5). */
    maxVisible?: number;
}

/** Options for requesting a modal via `piko.modal.open`. */
export interface ModalRequestOptions {
    /** CSS selector identifying the modal element. */
    selector: string;
    /** Key-value parameters to pass to the modal. */
    params?: Map<string, string>;
    /** Title text to display in the modal header. */
    title?: string;
    /** Body message to display in the modal content. */
    message?: string;
    /** Label for the cancel button. */
    cancelLabel?: string;
    /** Label for the confirm button. */
    confirmLabel?: string;
    /** Server action name to invoke on confirmation. */
    confirmAction?: string;
    /** Element that triggered the modal. */
    triggerElement: HTMLElement;
    /** Custom event name dispatched as a fallback when no modal element is found. */
    fallbackEventName?: string;
}

/** Response from a direct server action call. */
export interface ActionResponse<T = unknown> {
    /** The typed response data from the action. */
    data: T;
    /** HTTP status code from the server. */
    status: number;
    /** Optional message from the server. */
    message?: string;
    /** Helper calls from the server response (available when suppressHelpers is used). */
    helpers?: { name: string; args?: unknown[] }[];
}

/** Piko global namespace interface available on `window.piko`. */
export interface PikoNamespace {
    /** Hooks API for analytics and integrations. */
    readonly hooks: {
        /**
         * Registers a callback for the given hook event.
         *
         * @param event - Hook event to listen for.
         * @param callback - Function invoked when the event fires.
         * @param options - Optional listener configuration.
         * @returns Unsubscribe function that removes the listener.
         */
        on<T = unknown>(event: HookEventType, callback: HookCallback<T>, options?: { id?: string }): () => void;

        /**
         * Registers a one-shot callback that is automatically removed after the first invocation.
         *
         * @param event - Hook event to listen for.
         * @param callback - Function invoked once when the event fires.
         * @param options - Optional listener configuration.
         * @returns Unsubscribe function that removes the listener.
         */
        once<T = unknown>(event: HookEventType, callback: HookCallback<T>, options?: Omit<{ id?: string }, 'once'>): () => void;

        /**
         * Removes a listener by its identifier.
         *
         * @param event - Hook event the listener was registered for.
         * @param id - Identifier of the listener to remove.
         */
        off(event: HookEventType, id: string): void;

        /**
         * Removes all listeners for the given event, or all listeners if no event is specified.
         *
         * @param event - Hook event to clear. Omit to clear all events.
         */
        clear(event?: HookEventType): void;
    };

    /**
     * Registers a helper function by name.
     *
     * @param name - Unique helper name used in `pk-action` attributes.
     * @param helper - Function to invoke when the helper is called.
     */
    registerHelper(name: string, helper: PPHelper): void;

    /**
     * Retrieves the configuration object for a frontend module.
     *
     * @param moduleName - Name of the module whose configuration to retrieve.
     * @returns The module configuration, or `null` if no configuration exists.
     */
    getModuleConfig<T = unknown>(moduleName: string): T | null;

    /**
     * Registers a callback to run when the framework is ready.
     *
     * If the framework is already initialised the callback fires immediately.
     * Multiple callbacks can be registered and they execute in registration order.
     *
     * @param callback - Function to invoke once the framework is ready.
     */
    ready(callback: () => void): void;

    /** Navigation utilities. */
    readonly nav: {
        /**
         * Navigates to the given URL.
         *
         * @param url - Destination URL.
         * @param options - Navigation options.
         */
        navigate(url: string, options?: { replace?: boolean; scroll?: boolean }): Promise<void>;

        /**
         * Navigates to a URL, optionally preventing the default event behaviour.
         *
         * @param url - Destination URL.
         * @param event - Originating DOM event to prevent, if any.
         */
        navigateTo(url: string, event?: Event): void;
    };

    /** Modal management. */
    readonly modal: {
        /**
         * Opens a modal with the given options.
         *
         * @param options - Modal configuration.
         */
        open(options: ModalRequestOptions): Promise<void>;
    };

    /** Loader UI control. */
    readonly loader: {
        /**
         * Shows or hides the global loader.
         *
         * @param visible - Whether the loader should be visible.
         */
        toggle(visible: boolean): void;

        /**
         * Sets the loader progress percentage.
         *
         * @param percent - Progress value between 0 and 100.
         */
        progress(percent: number): void;

        /**
         * Displays an error message in the loader.
         *
         * @param message - Error message to display.
         */
        error(message: string): void;
    };

    /** Network status. */
    readonly network: {
        /**
         * Returns whether the browser is currently online.
         *
         * @returns `true` when the browser reports an active network connection.
         */
        isOnline(): boolean;
    };

    /** Server action dispatch. */
    readonly actions: {
        /**
         * Dispatches a server action.
         *
         * @param actionName - Name of the server action to invoke.
         * @param element - Element that triggered the action.
         * @param event - Originating DOM event, if any.
         */
        dispatch(actionName: string, element: HTMLElement, event?: Event): void;
    };

    /** Helper execution. */
    readonly helpers: {
        /**
         * Executes a helper action string.
         *
         * @param event - Originating DOM event.
         * @param actionString - Helper action expression to evaluate.
         * @param element - Element the helper is bound to.
         */
        execute(event: Event, actionString: string, element: HTMLElement): void;
    };

    /** Asset path resolution. */
    readonly assets: {
        /**
         * Resolves an asset path relative to a module.
         *
         * @param src - Asset source path.
         * @param moduleName - Module name for scoped resolution.
         * @returns Fully resolved asset URL.
         */
        resolve(src: string, moduleName?: string): string;
    };

    /** Partial reload coordination. */
    readonly partials: {
        /**
         * Reloads a named partial.
         *
         * @param name - Partial name to reload.
         * @param options - Reload options.
         */
        reload(name: string, options?: unknown): Promise<void>;

        /**
         * Renders a partial with the given options.
         *
         * @param options - Render configuration.
         */
        render(options: unknown): Promise<void>;
    };
}

declare global {
    interface Window {
        /** Global Piko namespace. */
        piko: PikoNamespace;
        /** Pre-init hook queue for registering hooks before the framework loads. */
        __PP_HOOKS_QUEUE__?: Array<{
            event: HookEventType;
            callback: HookCallback;
        }>;
    }
}

export {};
