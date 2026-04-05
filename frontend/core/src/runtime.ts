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

import {PPFramework, RegisterHelper} from '@/core/PPFramework';
import {_initCleanupObserver} from '@/pk/lifecycle';
import {getGlobalPageContext} from '@/services/PageContext';
import {_registerCapability} from '@/core/CapabilityRegistry';
import {registerActionFunction} from '@/pk/actionRegistry';

import {bus as _bus} from '@/pk/bus';
import {
    navigate as _navigate, goBack as _goBack, goForward as _goForward,
    go as _go, currentRoute as _currentRoute, buildUrl as _buildUrl,
    updateQuery as _updateQuery, registerNavigationGuard as _registerNavigationGuard,
    matchPath as _matchPath, extractParams as _extractParams,
} from '@/pk/navigation';
import {loading as _loading, withLoading as _withLoading, withRetry as _withRetry, debounceAsync as _debounceAsync, throttleAsync as _throttleAsync} from '@/pk/ui';
import {dispatch as _dispatch, listen as _listen, listenOnce as _listenOnce, waitForEvent as _waitForEvent} from '@/pk/events';
import {debounce as _debounce, throttle as _throttle} from '@/pk/utils';
import {whenVisible as _whenVisible, withAbortSignal as _withAbortSignal, timeout as _timeout, poll as _poll, watchMutations as _watchMutations, whenIdle as _whenIdle, nextFrame as _nextFrame, waitFrames as _waitFrames, deferred as _deferred, once as _once} from '@/pk/advanced';
import {trace as _trace, traceLog as _traceLog} from '@/pk/trace';

/** State handed off from shim.ts via window.__pikoShimData__. */
interface ShimData {
    /** Pre-registered hook listeners keyed by event name. */
    hookListeners: Map<string, Set<(payload: unknown) => void>>;
    /** Pre-registered helper functions keyed by name. */
    helpers: Map<string, (...args: unknown[]) => void>;
    /** Pre-registered capability implementations keyed by name. */
    capabilities: Map<string, unknown>;
    /** Globally exported page functions keyed by name. */
    globalExports: Map<string, (...args: unknown[]) => unknown>;
    /** Registered action factories keyed by name. */
    actionRegistry: Map<string, (...args: unknown[]) => unknown>;
}

/**
 * Initialises PPFramework and replays any state accumulated in
 * window.__pikoShimData__.
 *
 * Replays helpers, hook listeners, capabilities, page exports, and action
 * factories into PPFramework's services. Detaches the pre-existing link
 * click handlers before calling PPFramework.init() so the DOMBinder can
 * re-scan without double-binding, then upgrades window.piko with the
 * complete namespace API.
 */
function upgradeFromShim(): void {
    const shimData = (window as unknown as { __pikoShimData__?: ShimData }).__pikoShimData__;

    _initCleanupObserver();

    if (shimData) {
        shimData.helpers.forEach((fn, name) => {
            RegisterHelper(name, fn as (el: HTMLElement, event: Event, ...args: string[]) => void | Promise<void>);
        });

        shimData.capabilities.forEach((impl, name) => {
            _registerCapability(name, impl);
        });

        shimData.actionRegistry.forEach((factory, name) => {
            registerActionFunction(name, factory as (...args: unknown[]) => { action: string; args?: unknown[] });
        });
    }

    removeShimLinkHandlers();

    PPFramework.init();

    if (shimData) {
        shimData.hookListeners.forEach((listeners, event) => {
            listeners.forEach(cb => {
                PPFramework.hooks.on(event as never, cb as never);
            });
        });

        const pageContext = getGlobalPageContext();
        shimData.globalExports.forEach((fn, name) => {
            pageContext.setExports({ [name]: fn });
        });
    }

    upgradePikoNamespace();
}

/**
 * Detaches any pre-existing click handler from piko:a anchors so the
 * DOMBinder's subsequent scan does not leave two listeners attached.
 */
function removeShimLinkHandlers(): void {
    document.querySelectorAll<HTMLAnchorElement>('a[piko\\:a]').forEach(link => {
        const nav = (link as unknown as { __pkNav?: EventListener }).__pkNav;
        if (nav) {
            link.removeEventListener('click', nav);
            delete (link as unknown as { __pkNav?: EventListener }).__pkNav;
        }
    });
}

/**
 * Replaces window.piko with the complete namespace API backed by PPFramework.
 *
 * Rebinds every entry, including hooks, registerHelper, and _registerCapability,
 * so that all live calls route through PPFramework's services.
 */
function upgradePikoNamespace(): void {
    const piko = (window as unknown as { piko: Record<string, unknown> }).piko;
    piko.bus = _bus;
    piko.nav = {
        navigate: _navigate, back: _goBack, forward: _goForward, go: _go,
        current: _currentRoute, buildUrl: _buildUrl, updateQuery: _updateQuery,
        guard: _registerNavigationGuard, matchPath: _matchPath, extractParams: _extractParams,
    };
    piko.ui = { loading: _loading, withLoading: _withLoading, withRetry: _withRetry };
    piko.event = { dispatch: _dispatch, listen: _listen, listenOnce: _listenOnce, waitFor: _waitForEvent };
    piko.timing = {
        debounce: _debounce, throttle: _throttle,
        debounceAsync: _debounceAsync, throttleAsync: _throttleAsync,
        timeout: _timeout, poll: _poll, nextFrame: _nextFrame, waitFrames: _waitFrames,
    };
    piko.util = {
        whenVisible: _whenVisible, withAbortSignal: _withAbortSignal,
        watchMutations: _watchMutations, whenIdle: _whenIdle,
        deferred: _deferred, once: _once,
    };
    piko.trace = {
        enable: _trace.enable, disable: _trace.disable, isEnabled: _trace.isEnabled,
        clear: _trace.clear, getEntries: _trace.getEntries, getMetrics: _trace.getMetrics,
        log: _traceLog,
    };
    piko.network = { isOnline: () => PPFramework.isOnline };
    piko.loader = {
        toggle: (visible: boolean) => PPFramework.toggleLoader(visible),
        progress: (percent: number) => PPFramework.updateProgressBar(percent),
        error: (message: string) => PPFramework.displayError(message),
        create: (colour: string) => PPFramework.createLoaderIndicator(colour),
    };
    piko.context = { get: getGlobalPageContext };
    piko.hooks = PPFramework.hooks;
    piko.registerHelper = (name: string, fn: unknown) => RegisterHelper(name, fn as (el: HTMLElement, event: Event, ...args: string[]) => void | Promise<void>);
    piko._registerCapability = _registerCapability;
    piko.getModuleConfig = PPFramework.getModuleConfig;
    piko._emitHook = (event: string, payload: unknown) => PPFramework._emitHook(event as never, payload as never);
}

upgradeFromShim();
