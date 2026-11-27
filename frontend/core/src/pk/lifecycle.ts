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

import type { PikoDebugAPI, PartialDebugInfo, PageScriptDebugInfo } from './debug';

/** A function that performs teardown when a partial or page script is disconnected. */
type CleanupFn = () => void;

/** A callback with no arguments and no return value. */
type VoidCallback = () => void;

/** A callback invoked when a partial's content is updated, optionally receiving context about the change. */
type UpdatedCallback = (context?: unknown) => void;

/**
 * Lifecycle callbacks that can be registered for a PK partial instance.
 *
 * Mirrors the PKC lifecycle API for consistency.
 */
export interface PKLifecycleCallbacks {
    /** Called once when the partial is first connected to the DOM. */
    onConnected?: VoidCallback;
    /** Called when the partial is removed from the DOM. */
    onDisconnected?: VoidCallback;
    /** Called immediately before the partial is re-rendered. */
    onBeforeRender?: VoidCallback;
    /** Called immediately after the partial finishes re-rendering. */
    onAfterRender?: VoidCallback;
    /** Called when the partial's content is updated, with optional context. */
    onUpdated?: UpdatedCallback;
}

/** The callbacks registered for each lifecycle hook, in registration order. */
interface LifecycleCallbackLists {
    /** Callbacks to run when the state connects. */
    onConnected: VoidCallback[];
    /** Callbacks to run when the state disconnects. */
    onDisconnected: VoidCallback[];
    /** Callbacks to run before a re-render. */
    onBeforeRender: VoidCallback[];
    /** Callbacks to run after a re-render. */
    onAfterRender: VoidCallback[];
    /** Callbacks to run when content is updated. */
    onUpdated: UpdatedCallback[];
}

/** The name of a lifecycle hook that takes callbacks. */
export type LifecycleHookName = keyof LifecycleCallbackLists;

/** The type of callback a lifecycle hook takes. */
export type LifecycleHookCallback<H extends LifecycleHookName> = LifecycleCallbackLists[H][number];

/**
 * The lifecycle of one partial instance or one page script instance.
 *
 * A partial's state lives as long as its root element. A page script's state lives for
 * one page visit.
 */
interface LifecycleState {
    /** The registered callbacks of each hook. */
    callbacks: LifecycleCallbackLists;
    /** Whether the onConnected callbacks have run since the state last connected. */
    connectedOnce: boolean;
    /**
     * Whether the state is running its onDisconnected callbacks and cleanups. An
     * onConnected callback added meanwhile waits for the next connection, and a nested
     * connect or disconnect does nothing.
     */
    disconnecting: boolean;
    /** Teardown functions to run, once, when the state disconnects. */
    cleanups: CleanupFn[];
    /**
     * Whether the page visit of a page script has ended. An ended state takes no more
     * callbacks and runs new cleanups straight away. A partial's state never ends.
     */
    ended: boolean;
}

/** A page script's lifecycle state and the element its pk context is scoped to. */
interface PageLifecycleEntry {
    /** The element the page script's pk context is scoped to. */
    scope: Element;
    /** The page script's lifecycle state for the current page visit. */
    state: LifecycleState;
}

/**
 * Registers the lifecycle callbacks of one page script instance.
 *
 * The handle is bound to the instance's own state and scope, so callbacks cannot be
 * added to another instance by mistake.
 */
export interface PageLifecycleHandle {
    /**
     * Adds a callback to a lifecycle hook. Does nothing once the page visit has ended.
     *
     * @param hookName - The lifecycle hook to add to.
     * @param callback - The callback to register.
     */
    add<H extends LifecycleHookName>(hookName: H, callback: LifecycleHookCallback<H>): void;
    /**
     * Adds a cleanup that runs when the page visit ends. Runs it straight away when the
     * visit has already ended.
     *
     * @param cleanupFunction - The cleanup to run.
     */
    addCleanup(cleanupFunction: CleanupFn): void;
}

/** The lifecycle hook names, in the order the debug API reports them. */
const LIFECYCLE_HOOK_NAMES: readonly LifecycleHookName[] = [
    'onConnected',
    'onDisconnected',
    'onBeforeRender',
    'onAfterRender',
    'onUpdated',
];

/**
 * The order in which _registerLifecycle adds callbacks. onConnected comes last, so the
 * other callbacks are in place when it runs.
 */
const LEGACY_REGISTRATION_ORDER: readonly LifecycleHookName[] = [
    'onDisconnected',
    'onBeforeRender',
    'onAfterRender',
    'onUpdated',
    'onConnected',
];

/** Names element-scoped cleanups in error logs. */
const ELEMENT_CLEANUP_LABEL = 'element cleanup';

/** Names lifecycle cleanups in error logs. */
const LIFECYCLE_CLEANUP_LABEL = 'lifecycle cleanup';

/**
 * The number of page script lifecycles in one page visit above which a development
 * build warns.
 *
 * Generated page scripts create one lifecycle per script per visit, and the list empties
 * when the visit ends. It only grows past this number when a script creates page
 * contexts in a loop. There is no hard cap, because dropping a lifecycle would silently
 * stop the hooks of a script that needs them.
 */
const PAGE_LIFECYCLE_WARNING_THRESHOLD = 100;

/** Page-level cleanup functions to run on navigation. */
const pageCleanups: CleanupFn[] = [];

/** Element-scoped cleanup functions, keyed by element. */
const elementCleanups = new WeakMap<Element, CleanupFn[]>();

/**
 * Elements whose element-scoped cleanups are running now. A cleanup added to one of them
 * meanwhile runs straight away, as the element is already being torn down.
 */
const elementsCleaningUp = new WeakSet<Element>();

/**
 * Stores lifecycle state per partial element.
 * Uses WeakMap so lifecycle state is GC'd when element is removed.
 */
const partialLifecycleState = new WeakMap<Element, LifecycleState>();

/**
 * Lifecycle states of the page scripts of the page visit currently shown.
 *
 * A page script is scoped to the page root, which SPA navigation keeps. If its
 * lifecycle were keyed by that element, every page would share one state and it would
 * never disconnect. Instead, each page visit gets fresh states here, and the end of the
 * visit disconnects them (see _endPageVisit).
 */
const pageLifecycleEntries: PageLifecycleEntry[] = [];

/** Identifies the page visit currently shown. Replaced each time SPA navigation leaves a page. */
let currentPageVisit: object = {};

/** Whether _endPageVisit is running the callbacks of the visit that is ending. */
let endingPageVisit = false;

/** Whether the development warning about many page lifecycles was given in this visit. */
let warnedAboutPageLifecycles = false;

/**
 * Calls each callback with the same arguments. An error in one callback is logged and
 * does not stop the others.
 *
 * The callbacks are copied first, so a callback added during the pass does not run in
 * the same pass.
 *
 * @param callbacks - The callbacks to call, in order.
 * @param label - Names the callbacks in the error log.
 * @param args - The arguments to pass to each callback.
 */
function invokeEach<A extends unknown[]>(
    callbacks: ReadonlyArray<(...args: A) => void>,
    label: string,
    ...args: A
): void {
    for (const callback of [...callbacks]) {
        try {
            callback(...args);
        } catch (error) {
            console.error(`[pk] Error in ${label}:`, error);
        }
    }
}

/**
 * Creates an empty lifecycle state.
 *
 * @returns A lifecycle state with no callbacks that has not connected yet.
 */
function newLifecycleState(): LifecycleState {
    return {
        callbacks: {
            onConnected: [],
            onDisconnected: [],
            onBeforeRender: [],
            onAfterRender: [],
            onUpdated: [],
        },
        connectedOnce: false,
        disconnecting: false,
        cleanups: [],
        ended: false,
    };
}

/**
 * Returns the lifecycle state for a scope element, creating it if needed.
 *
 * @param scope - The partial's root element.
 * @returns The lifecycle state for the scope.
 */
function getOrCreateState(scope: Element): LifecycleState {
    let state = partialLifecycleState.get(scope);
    if (!state) {
        state = newLifecycleState();
        partialLifecycleState.set(scope, state);
    }
    return state;
}

/**
 * Runs a state's onConnected callbacks, once per connection. Nothing runs while the state
 * disconnects.
 *
 * @param state - The lifecycle state to connect.
 */
function runConnected(state: LifecycleState): void {
    if (state.disconnecting || state.connectedOnce) {
        return;
    }
    state.connectedOnce = true;
    invokeEach(state.callbacks.onConnected, 'onConnected');
}

/**
 * Runs a state's onDisconnected callbacks and cleanups, and lets it connect again.
 *
 * The state counts as connected until its cleanups have run. An onConnected callback
 * added by one of these callbacks does not run now. It runs at the next connection. A
 * disconnect started by one of these callbacks does nothing, so no callback runs twice.
 *
 * @param state - The lifecycle state to disconnect.
 */
function runDisconnected(state: LifecycleState): void {
    if (state.disconnecting) {
        return;
    }
    state.disconnecting = true;
    try {
        invokeEach(state.callbacks.onDisconnected, 'onDisconnected');
        invokeEach(state.cleanups.splice(0), LIFECYCLE_CLEANUP_LABEL);
    } finally {
        state.disconnecting = false;
        state.connectedOnce = false;
    }
}

/**
 * Connects a state after an onConnected callback was added to it while its scope is
 * connected. If the state has already connected, only the new callback runs. While the
 * state disconnects, nothing runs.
 *
 * @param state - The lifecycle state the callback was added to.
 * @param callback - The onConnected callback that was added.
 */
function connectAddedCallback(state: LifecycleState, callback: VoidCallback): void {
    if (state.disconnecting) {
        return;
    }
    if (state.connectedOnce) {
        invokeEach([callback], 'onConnected');
        return;
    }
    runConnected(state);
}

/**
 * Adds a callback to a partial's lifecycle state. An onConnected callback added while
 * the partial is in the DOM connects the state.
 *
 * @param state - The partial's lifecycle state.
 * @param scope - The partial's root element.
 * @param hookName - The lifecycle hook to add to.
 * @param callback - The callback to register.
 */
function addPartialCallback<H extends LifecycleHookName>(
    state: LifecycleState,
    scope: Element,
    hookName: H,
    callback: LifecycleHookCallback<H>,
): void {
    state.callbacks[hookName].push(callback);
    if (hookName === 'onConnected' && scope.isConnected) {
        connectAddedCallback(state, callback);
    }
}

/**
 * Adds a callback to a page script's lifecycle state. Nothing is added once the page
 * visit has ended.
 *
 * A page script is connected for the whole of its page visit, so an onConnected callback
 * connects the state whether or not the scope element is in the DOM.
 *
 * @param state - The page script's lifecycle state.
 * @param hookName - The lifecycle hook to add to.
 * @param callback - The callback to register.
 */
function addPageCallback<H extends LifecycleHookName>(
    state: LifecycleState,
    hookName: H,
    callback: LifecycleHookCallback<H>,
): void {
    if (state.ended) {
        return;
    }
    state.callbacks[hookName].push(callback);
    if (hookName === 'onConnected') {
        connectAddedCallback(state, callback);
    }
}

/**
 * Adds a cleanup to a lifecycle state. A state whose page visit has ended runs the
 * cleanup straight away.
 *
 * @param state - The lifecycle state to add to.
 * @param cleanupFunction - The cleanup to run when the state disconnects.
 */
function addCleanup(state: LifecycleState, cleanupFunction: CleanupFn): void {
    if (state.ended) {
        invokeEach([cleanupFunction], LIFECYCLE_CLEANUP_LABEL);
        return;
    }
    state.cleanups.push(cleanupFunction);
}

/**
 * Returns the lifecycle states whose render hooks run when an element re-renders. These
 * are the element's own partial state, then every page script whose scope contains the
 * element.
 * A page script is scoped to the page root, so its render hooks run whenever a partial
 * reload re-renders an element on the page. RemoteRenderer calls the render hooks for
 * the element it patches when that element carries a `partial` attribute, which can be a
 * partial's root or any element a partial rendered.
 *
 * @param renderedElement - The element being re-rendered.
 * @returns The states, partial first.
 */
function statesRenderedWith(renderedElement: Element): LifecycleState[] {
    const states = pageLifecycleEntries
        .filter(entry => entry.scope.contains(renderedElement))
        .map(entry => entry.state);
    const partialState = partialLifecycleState.get(renderedElement);
    return partialState ? [partialState, ...states] : states;
}

/**
 * Creates the lifecycle state of a page script instance and adds it to the current page
 * visit.
 *
 * The state is born ended, so it never connects, when it is created while a visit is
 * ending, or for a visit that is not the one shown now. A page script module is loaded
 * once and shared by every visit to its page, so code from an old visit, such as a timer
 * that calls the module's `__reinit__`, can still run. The generated module passes the
 * visit it was started for, which stops that code from connecting on the page shown now.
 *
 * With no visit, the state joins the visit shown now.
 *
 * @param scope - The element the page script's pk context is scoped to.
 * @param visit - The page visit the state is for, from _currentPageVisit.
 * @returns The new lifecycle state.
 */
function createPageLifecycleState(scope: Element, visit: object | undefined): LifecycleState {
    const state = newLifecycleState();
    if (endingPageVisit || (visit !== undefined && visit !== currentPageVisit)) {
        state.ended = true;
        return state;
    }
    warnIfScopeDetached(scope);
    pageLifecycleEntries.push({scope, state});
    warnIfManyPageLifecycles();
    return state;
}

/**
 * Warns, in development builds, when a page script is scoped to an element that is not
 * in the DOM. The page script still connects, because its lifecycle follows the page
 * visit, but its refs find nothing until the element is added.
 *
 * @param scope - The element the page script's pk context is scoped to.
 */
function warnIfScopeDetached(scope: Element): void {
    if (!__DEV__ || scope.isConnected) {
        return;
    }
    console.warn('[pk] A page script is scoped to an element that is not in the DOM:', scope);
}

/**
 * Warns once per page visit, in development builds, when the visit holds an unusual
 * number of page script lifecycles.
 */
function warnIfManyPageLifecycles(): void {
    if (!__DEV__ || warnedAboutPageLifecycles || pageLifecycleEntries.length <= PAGE_LIFECYCLE_WARNING_THRESHOLD) {
        return;
    }
    warnedAboutPageLifecycles = true;
    console.warn(
        '[pk] The page holds many page script lifecycles. A script may be creating page contexts in a loop:',
        pageLifecycleEntries.length,
    );
}

/**
 * Returns the token of the page visit currently shown.
 *
 * Generated page scripts key their instance by this token, so each visit to a page,
 * including a return through SPA navigation, creates a fresh instance. They also pass it
 * to `_createPagePKContext`, so an instance made for an old visit never connects.
 *
 * @returns An object that stays the same until SPA navigation leaves the page.
 */
export function _currentPageVisit(): object {
    return currentPageVisit;
}

/**
 * Creates the lifecycle of a page script instance for a page visit.
 *
 * The instance is connected for the whole visit, so an onConnected callback runs straight
 * away. The end of the page visit runs onDisconnected and the cleanups. When the visit is
 * not the one shown now, the lifecycle has already ended. Its callbacks never run and a
 * cleanup runs straight away.
 *
 * @param scope - The element the page script's pk context is scoped to.
 * @param visit - The page visit the instance is for, from _currentPageVisit. Leave it
 *     out to use the visit shown now.
 * @returns A handle that registers callbacks on the new lifecycle.
 */
export function _createPageLifecycle(scope: Element, visit?: object): PageLifecycleHandle {
    const state = createPageLifecycleState(scope, visit);
    return {
        add: (hookName, callback) => {
            addPageCallback(state, hookName, callback);
        },
        addCleanup: (cleanupFunction) => {
            addCleanup(state, cleanupFunction);
        },
    };
}

/**
 * Ends the current page visit. It fires onDisconnected and the cleanups of every page
 * script instance of the visit, then starts a new visit, so the next page's scripts
 * create fresh instances.
 *
 * Every ending state is marked as ended before any callback runs. A callback that runs
 * late, or that registers more callbacks, cannot bring a state back. A page context
 * created by one of these callbacks is born ended.
 *
 * Called by PPFramework when SPA navigation leaves a page, before the DOM is swapped.
 */
export function _endPageVisit(): void {
    const ending = pageLifecycleEntries.splice(0);
    for (const entry of ending) {
        entry.state.ended = true;
    }

    const wasEnding = endingPageVisit;
    endingPageVisit = true;
    try {
        for (const entry of ending) {
            runDisconnected(entry.state);
        }
    } finally {
        endingPageVisit = wasEnding;
        currentPageVisit = {};
        warnedAboutPageLifecycles = false;
    }
}

/**
 * Registers lifecycle callbacks for a partial instance.
 *
 * Called by generated PK JavaScript when lifecycle functions are detected.
 * Kept for backward compatibility with already-compiled PK JS modules.
 *
 * Each callback is added as `_addLifecycleCallback` adds it, so an onConnected callback
 * registered after the partial has connected runs straight away, on its own.
 *
 * @param scope - The partial's root element (element with [partial] attribute).
 * @param callbacks - Lifecycle callback functions.
 */
export function _registerLifecycle(scope: Element, callbacks: PKLifecycleCallbacks): void {
    const state = getOrCreateState(scope);

    for (const hookName of LEGACY_REGISTRATION_ORDER) {
        const callback = callbacks[hookName];
        if (callback) {
            addPartialCallback(state, scope, hookName, callback);
        }
    }
}

/**
 * Adds a single lifecycle callback for a partial instance.
 *
 * Used by the `pk` context object for additive lifecycle registration.
 * Multiple calls to the same hook register multiple callbacks that all
 * fire in order. An onConnected callback added after the partial has
 * connected runs straight away, on its own.
 *
 * @param scope - The partial's root element.
 * @param hookName - The lifecycle hook name.
 * @param callback - The callback to register.
 */
export function _addLifecycleCallback<H extends LifecycleHookName>(
    scope: Element,
    hookName: H,
    callback: LifecycleHookCallback<H>,
): void {
    addPartialCallback(getOrCreateState(scope), scope, hookName, callback);
}

/**
 * Executes the onConnected lifecycle for a partial element.
 *
 * Only fires once per element lifecycle (reset on disconnection).
 *
 * @param scope - The partial's root element.
 */
export function _executeConnected(scope: Element): void {
    const state = partialLifecycleState.get(scope);
    if (state) {
        runConnected(state);
    }
}

/**
 * Executes the onDisconnected lifecycle for a partial element.
 *
 * Resets connectedOnce so onConnected can fire again if re-mounted.
 *
 * @param scope - The partial's root element.
 */
export function _executeDisconnected(scope: Element): void {
    const state = partialLifecycleState.get(scope);
    if (state) {
        runDisconnected(state);
    }
}

/**
 * Executes the onBeforeRender lifecycle for a partial element.
 *
 * Called before a partial reload/re-render. Also runs onBeforeRender of the page
 * scripts whose scope contains the element.
 *
 * @param scope - The element being re-rendered, usually a partial's root.
 */
export function _executeBeforeRender(scope: Element): void {
    for (const state of statesRenderedWith(scope)) {
        invokeEach(state.callbacks.onBeforeRender, 'onBeforeRender');
    }
}

/**
 * Executes the onAfterRender lifecycle for a partial element.
 *
 * Called after a partial reload/re-render completes. Also runs onAfterRender of the page
 * scripts whose scope contains the element.
 *
 * @param scope - The element that was re-rendered, usually a partial's root.
 */
export function _executeAfterRender(scope: Element): void {
    for (const state of statesRenderedWith(scope)) {
        invokeEach(state.callbacks.onAfterRender, 'onAfterRender');
    }
}

/**
 * Executes the onUpdated lifecycle for a partial element.
 *
 * Called after partial content is updated. Also runs onUpdated of the page scripts whose
 * scope contains the element, with the same context.
 *
 * @param scope - The element that was re-rendered, usually a partial's root.
 * @param context - Optional context about what was updated.
 */
export function _executeUpdated(scope: Element, context?: unknown): void {
    for (const state of statesRenderedWith(scope)) {
        invokeEach(state.callbacks.onUpdated, 'onUpdated', context);
    }
}

/**
 * Executes onConnected for all partial elements within a container.
 *
 * Called after DOM patching to ensure new partials receive lifecycle events.
 *
 * @param container - The container element or document to scan.
 */
export function _executeConnectedForPartials(container: Element | Document): void {
    const partials = container.querySelectorAll('[partial]');
    for (const partial of partials) {
        _executeConnected(partial);
    }
}

/**
 * Checks whether a partial has lifecycle callbacks registered.
 *
 * @param scope - The partial's root element.
 * @returns True if lifecycle callbacks are registered.
 */
export function _hasLifecycleCallbacks(scope: Element): boolean {
    return partialLifecycleState.has(scope);
}

/**
 * Registers a cleanup function to be called when navigating away from the page
 * or when an element is removed from the DOM. If the element's cleanups are already
 * running, a new cleanup for it runs straight away.
 *
 * @param cleanupFunction - Cleanup function to call.
 * @param scope - Optional element to scope the cleanup to. If provided, the cleanup
 *                runs when this element is removed from the DOM. If omitted, the
 *                cleanup runs on page navigation.
 */
export function onCleanup(cleanupFunction: CleanupFn, scope?: Element): void {
    if (scope && elementsCleaningUp.has(scope)) {
        invokeEach([cleanupFunction], ELEMENT_CLEANUP_LABEL);
        return;
    }
    if (scope) {
        let scopeCleanups = elementCleanups.get(scope);
        if (!scopeCleanups) {
            scopeCleanups = [];
            elementCleanups.set(scope, scopeCleanups);
        }
        scopeCleanups.push(cleanupFunction);
    } else {
        pageCleanups.push(cleanupFunction);
    }
}

/**
 * Runs all page-level cleanup functions.
 *
 * Called by PPFramework on navigation.
 */
export function _runPageCleanup(): void {
    invokeEach(pageCleanups.splice(0), 'page cleanup');
}

/** Active MutationObserver for element removal cleanup, or null if not initialised. */
let cleanupObserver: MutationObserver | null = null;

/**
 * Initialises the MutationObserver that watches for element removal
 * and triggers associated cleanup functions.
 */
export function _initCleanupObserver(): void {
    if (cleanupObserver) {
        return;
    }

    cleanupObserver = new MutationObserver((mutations) => {
        for (const mutation of mutations) {
            for (const node of mutation.removedNodes) {
                if (typeof Element !== 'undefined' && node instanceof Element) {
                    runElementCleanups(node);
                }
            }
        }
    });

    cleanupObserver.observe(document.body, {
        childList: true,
        subtree: true
    });
}

/**
 * Disconnects the cleanup observer. Used for testing.
 */
export function _disconnectCleanupObserver(): void {
    if (cleanupObserver) {
        cleanupObserver.disconnect();
        cleanupObserver = null;
    }
}

/**
 * Runs the element-scoped cleanups of one element, once.
 *
 * @param element - The element whose cleanups run.
 */
function runCleanupsOf(element: Element): void {
    const cleanups = elementCleanups.get(element);
    if (!cleanups) {
        return;
    }
    elementCleanups.delete(element);
    elementsCleaningUp.add(element);
    try {
        invokeEach(cleanups, ELEMENT_CLEANUP_LABEL);
    } finally {
        elementsCleaningUp.delete(element);
    }
}

/**
 * Runs cleanup functions and lifecycle events for an element and all its descendants.
 *
 * Triggers onDisconnected for any partial elements, then executes registered
 * element-scoped cleanups.
 *
 * @param element - The element being removed.
 */
function runElementCleanups(element: Element): void {
    executeDisconnectedForRemovedPartials(element);

    runCleanupsOf(element);
    for (const child of element.querySelectorAll('*')) {
        runCleanupsOf(child);
    }
}

/**
 * Fires onDisconnected for any partial elements within the removed subtree,
 * including the element itself if it carries the `[partial]` attribute.
 *
 * @param element - The root element of the removed subtree.
 */
function executeDisconnectedForRemovedPartials(element: Element): void {
    if (element.hasAttribute('partial')) {
        _executeDisconnected(element);
    }

    const partials = element.querySelectorAll('[partial]');
    for (const partial of partials) {
        _executeDisconnected(partial);
    }
}

/**
 * Returns the names of the hooks that have callbacks registered.
 *
 * @param callbacks - The lifecycle callbacks to inspect.
 * @returns Array of callback names.
 */
function getCallbackNames(callbacks: LifecycleCallbackLists): string[] {
    return LIFECYCLE_HOOK_NAMES.filter(hookName => callbacks[hookName].length > 0);
}

/**
 * Returns whether a partial element's lifecycle has connected.
 *
 * @param element - The partial's root element.
 * @returns True when the partial's onConnected callbacks have run.
 */
function isPartialConnected(element: Element): boolean {
    return partialLifecycleState.get(element)?.connectedOnce ?? false;
}

/**
 * Creates the debug inspection API for E2E testing.
 *
 * Provides read-only access to internal lifecycle state.
 *
 * @returns The debug API object.
 */
function createDebugAPI(): PikoDebugAPI {
    return {
        getPartialInfo(element: Element): PartialDebugInfo {
            const state = partialLifecycleState.get(element);
            const cleanups = elementCleanups.get(element);

            return {
                exists: state !== undefined,
                partialName: element.getAttribute('partial_name') ?? element.getAttribute('data-partial-name'),
                partialId: element.getAttribute('partial') ?? element.getAttribute('data-partial'),
                isConnected: isPartialConnected(element),
                connectedOnce: state?.connectedOnce ?? false,
                registeredCallbacks: state ? getCallbackNames(state.callbacks) : [],
                cleanupCount: (state?.cleanups.length ?? 0) + (cleanups?.length ?? 0),
            };
        },

        isConnected(element: Element): boolean {
            return isPartialConnected(element);
        },

        getCleanupCount(element: Element): number {
            const state = partialLifecycleState.get(element);
            const cleanups = elementCleanups.get(element);
            return (state?.cleanups.length ?? 0) + (cleanups?.length ?? 0);
        },

        getRegisteredCallbacks(element: Element): string[] {
            const state = partialLifecycleState.get(element);
            return state ? getCallbackNames(state.callbacks) : [];
        },

        getAllConnectedPartials(): Element[] {
            const partials = document.querySelectorAll('[partial], [data-partial]');
            return Array.from(partials).filter(isPartialConnected);
        },

        getPageScripts(): PageScriptDebugInfo[] {
            return pageLifecycleEntries.map(({state}) => ({
                connectedOnce: state.connectedOnce,
                registeredCallbacks: getCallbackNames(state.callbacks),
                cleanupCount: state.cleanups.length,
            }));
        },

        isAvailable(): boolean {
            return true;
        },
    };
}

if (typeof window !== 'undefined') {
    window.__pikoDebug = createDebugAPI();
}
