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

import type {FetchClient} from '@/core/FetchClient';
import type {LoaderUI} from '@/services/LoaderUI';
import type {ErrorDisplay} from '@/services/ErrorDisplay';
import type {FormStateManager} from '@/services/FormStateManager';
import type {A11yAnnouncer} from '@/services/A11yAnnouncer';
import {HookEvent, type HookManager} from '@/services/HookManager';
import {type WindowOperations, type DOMOperations, browserWindowOperations, browserDOMOperations} from '@/core/BrowserAPIs';
import {addFragmentQuery, isSameDomain} from '@/core/URLUtils';

/**
 * Options for navigation operations.
 */
export interface NavigateOptions {
    /** Whether to replace the current history entry instead of pushing. */
    replaceHistory?: boolean;
    /** Whether this navigation is triggered by browser back/forward. */
    isPopState?: boolean;
    /** Scroll position to restore (used by popstate navigation). */
    restoreScrollY?: number;
    /** Callback invoked before navigation begins. */
    beforeNavigate?: (url: string) => void;
    /** Callback invoked after navigation completes. */
    afterNavigate?: (url: string) => void;
}

/**
 * State stored in browser history for scroll restoration.
 */
export interface HistoryState {
    /** Vertical scroll position in pixels. */
    scrollY: number;
}

/**
 * Options passed to onPageLoad for scroll restoration.
 */
export interface PageLoadScrollOptions {
    /** Scroll position to restore for back/forward navigation. */
    restoreScrollY?: number;
    /** Hash/anchor to scroll to for new navigation. */
    hash?: string;
}

/**
 * Global configuration for the router.
 */
export interface RouterConfig {
    /** Global callback invoked before each navigation. */
    beforeNavigate?: (url: string) => void;
    /** Global callback invoked after each navigation. */
    afterNavigate?: (url: string) => void;
}

/**
 * Dependencies for creating a Router.
 */
export interface RouterDependencies {
    /** HTTP client for fetching pages. */
    fetchClient: FetchClient;
    /** Loading indicator UI. */
    loader: LoaderUI;
    /** Error display service. */
    errorDisplay: ErrorDisplay;
    /** Callback invoked when a page is loaded and ready for DOM insertion. May return a Promise when the DOM update is deferred (e.g. View Transitions). */
    onPageLoad: (doc: Document, targetUrl: string, scrollOptions: PageLoadScrollOptions) => void | Promise<void>;
    /** Window operations implementation. Defaults to browser window. */
    windowOps?: WindowOperations;
    /** DOM operations implementation. Defaults to browser document. */
    domOps?: DOMOperations;
    /** Hook manager for analytics events. */
    hookManager?: HookManager;
    /** Form state manager for dirty form checking. */
    formStateManager?: FormStateManager;
    /** Accessibility announcer for screen readers. */
    a11yAnnouncer?: A11yAnnouncer;
}

/**
 * SPA router with history management and scroll restoration.
 */
export interface Router {
    /** Navigates to a URL using SPA navigation. */
    navigateTo(targetUrl: string, event?: Event, options?: NavigateOptions): Promise<void>;

    /** Returns whether a navigation is currently in progress. */
    isNavigating(): boolean;

    /** Sets global navigation callbacks. */
    setConfig(config: RouterConfig): void;

    /** Cleans up event listeners and releases resources. */
    destroy(): void;
}

/** Maximum progress value for the loading indicator. */
const PROGRESS_MAX = 100;

/**
 * Invokes a navigation callback, catching and logging any errors it throws.
 *
 * @param callback - The callback to invoke, or undefined to skip.
 * @param url - The URL to pass to the callback.
 */
function safeInvokeCallback(callback: ((url: string) => void) | undefined, url: string): void {
    if (!callback) {
        return;
    }
    try {
        callback(url);
    } catch (error) {
        console.warn('[Router] Error in navigation callback:', {
            url,
            callback: callback.name || 'anonymous',
            error
        });
    }
}

/**
 * Mutable state shared across navigation operations.
 */
interface NavigationState {
    /** Whether a navigation is currently in progress. */
    navigating: boolean;
    /** Global navigation callbacks. */
    globalConfig: RouterConfig;
}

/**
 * Resolved dependencies used by navigation helpers.
 */
interface NavigationDeps {
    /** HTTP client for fetching pages. */
    fetchClient: FetchClient;
    /** Loading indicator UI. */
    loader: LoaderUI;
    /** Error display service. */
    errorDisplay: ErrorDisplay;
    /** Callback invoked when a page is loaded and ready for DOM insertion. May return a Promise when the DOM update is deferred (e.g. View Transitions). */
    onPageLoad: (doc: Document, targetUrl: string, scrollOptions: PageLoadScrollOptions) => void | Promise<void>;
    /** Window operations implementation. */
    windowOps: WindowOperations;
    /** DOM operations implementation. */
    domOps: DOMOperations;
    /** Hook manager for analytics events. */
    hookManager?: HookManager;
    /** Form state manager for dirty form checking. */
    formStateManager?: FormStateManager;
    /** Accessibility announcer for screen readers. */
    a11yAnnouncer?: A11yAnnouncer;
}

/**
 * Context for a single navigation operation.
 */
interface NavigationContext {
    /** The URL being navigated to. */
    targetUrl: string;
    /** The URL before navigation started. */
    previousUrl: string;
    /** Timestamp when the navigation began. */
    startTime: number;
    /** Whether this is a popstate (back/forward) navigation. */
    isPopNavigation: boolean;
    /** Original navigation options. */
    options: NavigateOptions;
}

/**
 * Displays an error, announces it for accessibility, emits a hook event,
 * and falls back to a full page load.
 *
 * @param deps - Navigation dependencies.
 * @param targetUrl - The URL that failed to load.
 * @param errorMessage - Technical error message for the hook event.
 * @param displayMessage - User-facing message. Defaults to a generic navigation failure.
 */
function emitNavigationError(
    deps: NavigationDeps,
    targetUrl: string,
    errorMessage: string,
    displayMessage?: string
): void {
    deps.errorDisplay.show(displayMessage ?? `Navigation to ${targetUrl} failed. Loading full page...`);
    deps.a11yAnnouncer?.announceError('Navigation failed');
    deps.hookManager?.emit(HookEvent.NAVIGATION_ERROR, {
        url: targetUrl,
        error: errorMessage,
        timestamp: Date.now()
    });
    deps.windowOps.setLocationHref(targetUrl);
}

/**
 * Saves the current scroll position and pushes or replaces the history entry.
 *
 * @param windowOps - Window operations implementation.
 * @param targetUrl - The URL to set in the history entry.
 * @param replaceHistory - Whether to replace instead of push.
 */
function updateHistoryState(
    windowOps: WindowOperations,
    targetUrl: string,
    replaceHistory: boolean | undefined
): void {
    const currentState: HistoryState = {scrollY: windowOps.getScrollY()};
    windowOps.historyReplaceState(currentState, '', windowOps.getLocationHref());

    const newState: HistoryState = {scrollY: 0};
    if (replaceHistory) {
        windowOps.historyReplaceState(newState, '', targetUrl);
    } else {
        windowOps.historyPushState(newState, '', targetUrl);
    }
}

/**
 * Announces the completed navigation for accessibility and emits analytics hooks.
 *
 * @param deps - Navigation dependencies.
 * @param ctx - The current navigation context.
 * @param pageTitle - The title of the loaded page.
 */
function emitNavigationSuccess(
    deps: NavigationDeps,
    ctx: NavigationContext,
    pageTitle: string
): void {
    const duration = Date.now() - ctx.startTime;

    deps.a11yAnnouncer?.announceNavigation(pageTitle);

    deps.hookManager?.emit(HookEvent.NAVIGATION_COMPLETE, {
        url: ctx.targetUrl,
        previousUrl: ctx.previousUrl,
        timestamp: Date.now(),
        duration
    });

    deps.hookManager?.emit(HookEvent.PAGE_VIEW, {
        url: ctx.targetUrl,
        title: pageTitle,
        referrer: ctx.previousUrl,
        isInitialLoad: false,
        timestamp: Date.now()
    });

    deps.formStateManager?.scanAndTrackForms();
}

/**
 * Handles errors thrown during navigation by logging and falling back.
 *
 * @param deps - Navigation dependencies.
 * @param targetUrl - The URL that caused the error.
 * @param error - The caught error.
 */
function handleNavigationError(
    deps: NavigationDeps,
    targetUrl: string,
    error: unknown
): void {
    if (error instanceof DOMException && error.name === 'AbortError') {
        console.warn('Fetch aborted:', targetUrl);
        return;
    }

    console.error('navigateTo error:', error);
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    emitNavigationError(deps, targetUrl, errorMessage);
}

/**
 * Checks whether navigation should be cancelled due to dirty forms.
 *
 * @param isPopNavigation - Whether this is a popstate navigation (never cancelled).
 * @param formStateManager - The form state manager, if available.
 * @returns True if navigation should be cancelled.
 */
function shouldCancelNavigation(
    isPopNavigation: boolean,
    formStateManager: FormStateManager | undefined
): boolean {
    if (isPopNavigation) {
        return false;
    }
    return Boolean(formStateManager?.hasDirtyForms() && !formStateManager.confirmNavigation());
}

/**
 * Invokes before-navigate callbacks and emits navigation start events.
 *
 * @param deps - Navigation dependencies.
 * @param ctx - The current navigation context.
 * @param localBeforeNavigate - The before-navigate callback for this navigation.
 */
function emitNavigationStart(
    deps: NavigationDeps,
    ctx: NavigationContext,
    localBeforeNavigate: ((url: string) => void) | undefined
): void {
    safeInvokeCallback(localBeforeNavigate, ctx.targetUrl);
    deps.hookManager?.emit(HookEvent.NAVIGATION_START, {
        url: ctx.targetUrl,
        previousUrl: ctx.previousUrl,
        timestamp: ctx.startTime
    });
    deps.a11yAnnouncer?.announceLoading();
}

/**
 * Builds scroll restoration options for a navigation.
 *
 * @param ctx - The current navigation context.
 * @param options - The navigation options containing scroll restoration data.
 * @param windowOps - Window operations for resolving the origin.
 * @returns Scroll options with either a restore position or hash anchor.
 */
function buildScrollOptions(
    ctx: NavigationContext,
    options: NavigateOptions,
    windowOps: WindowOperations
): PageLoadScrollOptions {
    const hash = new URL(ctx.targetUrl, windowOps.getLocationOrigin()).hash;
    return {
        restoreScrollY: ctx.isPopNavigation ? options.restoreScrollY : undefined,
        hash: (!ctx.isPopNavigation || options.restoreScrollY === undefined) ? hash : undefined
    };
}

/**
 * Executes a full SPA navigation: fetches the page, updates the DOM,
 * manages history state, and emits lifecycle events.
 *
 * @param state - Shared mutable navigation state.
 * @param deps - Resolved navigation dependencies.
 * @param targetUrl - The URL to navigate to.
 * @param event - The DOM event that triggered navigation, if any.
 * @param options - Navigation options.
 */
async function performNavigation(
    state: NavigationState,
    deps: NavigationDeps,
    targetUrl: string,
    event: Event | undefined,
    options: NavigateOptions
): Promise<void> {
    const {fetchClient, loader, onPageLoad, windowOps, domOps, formStateManager} = deps;

    if (event) {
        event.preventDefault();
    }

    const isPopNavigation = Boolean(options.isPopState);
    if (shouldCancelNavigation(isPopNavigation, formStateManager)) {
        return;
    }

    formStateManager?.untrackAll();

    if (state.navigating) {
        fetchClient.abort();
    }

    state.navigating = true;

    loader.show();

    const ctx: NavigationContext = {
        targetUrl,
        previousUrl: windowOps.getLocationHref(),
        startTime: Date.now(),
        isPopNavigation,
        options
    };

    const localBeforeNavigate = options.beforeNavigate ?? state.globalConfig.beforeNavigate;
    const localAfterNavigate = options.afterNavigate ?? state.globalConfig.afterNavigate;

    emitNavigationStart(deps, ctx, localBeforeNavigate);

    try {
        if (!isPopNavigation) {
            updateHistoryState(windowOps, targetUrl, options.replaceHistory);
        }

        const [success, htmlString] = await fetchClient.get(addFragmentQuery(targetUrl), {
            onProgress: (loaded, total) => loader.setProgress((loaded / total) * PROGRESS_MAX)
        });

        if (!success || !htmlString) {
            emitNavigationError(deps, targetUrl, 'Fetch failed');
            return;
        }

        const parsedDocument = domOps.parseHTML(htmlString);
        if (!parsedDocument.querySelector('#app')) {
            emitNavigationError(deps, targetUrl, 'No #app in response', 'No #app in fragment. Loading full page...');
            return;
        }

        const scrollOptions = buildScrollOptions(ctx, options, windowOps);

        await onPageLoad(parsedDocument, targetUrl, scrollOptions);
        focusMainContent(domOps);
        emitNavigationSuccess(deps, ctx, parsedDocument.title || document.title);
    } catch (error) {
        handleNavigationError(deps, targetUrl, error);
    } finally {
        state.navigating = false;

        loader.hide();
        safeInvokeCallback(localAfterNavigate, targetUrl);
    }
}

/**
 * Focuses the main content area for accessibility after SPA navigation.
 *
 * @param domOps - DOM operations implementation.
 */
function focusMainContent(domOps: DOMOperations): void {
    const mainContent = domOps.querySelector<HTMLElement>('[role="main"], main, #app');

    if (mainContent) {
        const hadTabIndex = mainContent.hasAttribute('tabindex');
        const originalTabIndex = mainContent.getAttribute('tabindex');

        mainContent.setAttribute('tabindex', '-1');
        mainContent.focus({preventScroll: true});

        if (hadTabIndex && originalTabIndex !== null) {
            mainContent.setAttribute('tabindex', originalTabIndex);
        } else if (!hadTabIndex) {
            mainContent.removeAttribute('tabindex');
        }
    }
}

/**
 * Creates a Router instance for SPA navigation.
 *
 * @param deps - Required and optional dependencies for the router.
 * @returns A new Router instance.
 */
export function createRouter(deps: RouterDependencies): Router {
    const windowOps = deps.windowOps ?? browserWindowOperations;
    const domOps = deps.domOps ?? browserDOMOperations;
    const state: NavigationState = {navigating: false, globalConfig: {}};

    windowOps.setScrollRestoration('manual');

    const navDeps: NavigationDeps = {
        fetchClient: deps.fetchClient,
        loader: deps.loader,
        errorDisplay: deps.errorDisplay,
        onPageLoad: deps.onPageLoad,
        windowOps,
        domOps,
        hookManager: deps.hookManager,
        formStateManager: deps.formStateManager,
        a11yAnnouncer: deps.a11yAnnouncer
    };

    const navigateTo = (targetUrl: string, event?: Event, options: NavigateOptions = {}) =>
        performNavigation(state, navDeps, targetUrl, event, options);

    const popstateHandler = () => {
        const location = windowOps.getLocation();
        if (!isSameDomain(location)) {
            windowOps.locationReload();
            return;
        }

        const historyState = windowOps.getHistoryState() as HistoryState | null;
        const restoreScrollY = historyState?.scrollY;

        void navigateTo(windowOps.getLocationHref(), undefined, {
            replaceHistory: true,
            isPopState: true,
            restoreScrollY
        });
    };

    windowOps.addEventListener('popstate', popstateHandler);

    return {
        navigateTo,
        isNavigating: () => state.navigating,
        setConfig: (config: RouterConfig) => {
            state.globalConfig = config;
        },
        destroy: () => windowOps.removeEventListener('popstate', popstateHandler)
    };
}
