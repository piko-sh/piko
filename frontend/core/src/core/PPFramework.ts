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

import {_runPageCleanup, _executeConnectedForPartials} from '@/pk';
import {registerDOMUpdater as _registerDOMUpdater} from '@/pk/domUpdater';
import {getGlobalPageContext} from '@/services/PageContext';
import {
    createDOMBinder,
    createErrorDisplay,
    createHelperRegistry,
    createLinkHeaderParser,
    createLoaderUI,
    createModuleLoader,
    createSpriteSheetManager,
    createSyncPartialManager,
    type DOMBinder,
    type ErrorDisplay,
    type HelperRegistry,
    type LinkHeaderParser,
    type LoaderUI,
    type ModuleLoader,
    type SpriteSheetManager,
    type SyncPartialManager,
    initModuleLoaderFromPage,
    type OpenModalOptions,
    type PPHelper
} from '@/services';
import {createHookManager, HookEvent, type HookManager, type HooksAPI} from '@/services/HookManager';
import {createNetworkStatus, type NetworkStatus} from '@/services/NetworkStatus';
import {createA11yAnnouncer, type A11yAnnouncer} from '@/services/A11yAnnouncer';
import {createFormStateManager, type FormStateManager} from '@/services/FormStateManager';
import {createFetchClient, type FetchClient, type FetchResult} from '@/core/FetchClient';
import {createRouter, type NavigateOptions, type PageLoadScrollOptions, type Router} from '@/core/Router';
import fragmentMorpher, {type NodeKey} from '@/core/fragmentMorpher';
import {handleAction} from '@/core/ActionExecutor';
import {getActionFunction} from '@/pk/action';
import {createRemoteRenderer, type PatchTarget, type RemoteRenderer, type RemoteRenderOptions} from '@/core/RemoteRenderer';
import {createModalManager, type ModalManager, type ModalRequestOptions} from '@/core/ModalManager';
import {addFragmentQuery, buildRemoteUrl, isSameDomain} from '@/core/URLUtils';

export type {PPHelper} from '@/services';
export type {NavigateOptions, RemoteRenderOptions, PatchTarget, FetchResult};

/**
 * Scans the DOM for partial elements and registers their name-to-ID mappings
 * in PageContext for scoped function resolution and cross-partial calls.
 *
 * @param doc - The document or fragment to scan.
 */
function registerPartialInstancesFromDOM(doc: Document | DocumentFragment): void {
    const partials = doc.querySelectorAll('[partial][data-partial-name]');
    const pageContext = getGlobalPageContext();

    for (const el of partials) {
        const partialId = el.getAttribute('partial');
        const partialName = el.getAttribute('data-partial-name');
        if (partialId && partialName) {
            pageContext.registerPartialInstance(partialName, partialId);
        }
    }
}

/**
 * Configuration options for initialising PPFramework.
 */
export interface PPFrameworkOptions {
    /** Colour for the loading indicator bar. */
    loaderColour?: string;
    /** Callback invoked before each navigation. */
    beforeNavigate?: (targetUrl: string) => void;
    /** Callback invoked after each navigation completes. */
    afterNavigate?: (targetUrl: string) => void;
}

/** Global helper registry shared by all framework components. */
const globalHelperRegistry = createHelperRegistry();

/**
 * Registers a helper function for use in templates.
 *
 * Helpers are callable from p-on:* attributes in HTML templates.
 *
 * @param name - The name used to reference the helper in templates.
 * @param setupFunction - The helper function implementation.
 */
export function RegisterHelper(name: string, setupFunction: PPHelper): void {
    globalHelperRegistry.register(name, setupFunction);
}

/**
 * Returns the global helper registry for internal use by ActionExecutor.
 *
 * @returns The global helper registry instance.
 */
export function getGlobalHelperRegistry(): typeof globalHelperRegistry {
    return globalHelperRegistry;
}

/**
 * Internal representation of the PPFramework singleton instance.
 */
interface PPFrameworkInstance {
    /** Set of loaded module script URLs (backwards compatibility). */
    loadedModuleScripts: Set<string>;
    /** Whether a navigation is currently in progress (backwards compatibility). */
    navigating: boolean;
    /** The loader bar element (backwards compatibility). */
    loaderElement: HTMLDivElement | null;
    /** The current AbortController (backwards compatibility). */
    currentAbortController: AbortController | null;
    /** Global configuration options. */
    globalConfig: PPFrameworkOptions;

    /** Initialises the framework with the given options. */
    init(options?: PPFrameworkOptions): void;

    /** Navigates to a URL using SPA navigation. */
    navigateTo(targetUrl: string, evt?: Event, options?: NavigateOptions): Promise<void>;

    /** Fetches and renders remote HTML content. */
    remoteRender(options: RemoteRenderOptions): Promise<void>;

    /**
     * Dispatches a server action from compiled template event handlers.
     *
     * @param actionName - The registered action name (e.g., "contact.send").
     * @param element - The element that triggered the event.
     * @param event - The original DOM event.
     */
    dispatchAction(
        actionName: string,
        element: HTMLElement,
        event?: Event
    ): void;

    /** Builds a URL with query parameters for remote rendering. */
    buildRemoteUrl(base: string, args: Record<string, string | number>): string;

    /** Adds the fragment query parameter to a URL. */
    addFragmentQuery(urlValue: string): string;

    /** Checks if a location or anchor element is on the same domain. */
    isSameDomain(loc: Location | HTMLAnchorElement): boolean;

    /**
     * Transforms an asset source path by prepending the asset serve path.
     *
     * @param src - The asset source path.
     * @param moduleName - Optional module name for resolving @/ aliases.
     * @returns The resolved asset URL.
     */
    assetSrc(src: string, moduleName?: string): string;

    /** Toggles the loading indicator visibility. */
    toggleLoader(isVisible: boolean): void;

    /** Updates the progress bar to the given percentage. */
    updateProgressBar(percentValue: number): void;

    /** Displays an error message to the user. */
    displayError(message: string): void;

    /** Creates a new loader indicator with the given colour. */
    createLoaderIndicator(color: string): void;

    /** Loads module scripts from the given document. */
    loadModuleScripts(doc: Document): void;

    /** Patches an HTML string into the DOM at the given CSS selector. */
    patchPartial(htmlString: string, cssSelector: string): void;

    /** Opens a modal if available, dispatching a fallback event if not found. */
    openModalIfAvailable(options: ModalRequestOptions): Promise<void>;

    /** Executes a helper action from a p-on attribute string. */
    executeHelper(event: Event, actionString: string, element: HTMLElement): void;

    /** Executes a server helper by name with the given arguments. */
    executeServerHelper(name: string, args: unknown[], triggerElement: HTMLElement, event?: Event): void;

    /** Hooks API for analytics integrations. */
    hooks: HooksAPI;

    /** Internal hook emitter for framework-level event dispatch. */
    emitHook: HookManager['emit'];

    /** Registers a helper function for extensions. */
    registerHelper(name: string, helper: PPHelper): void;

    /** Whether the browser is currently online. */
    readonly isOnline: boolean;

    /**
     * Returns the module configuration for the given module name.
     *
     * @param moduleName - The module to retrieve configuration for.
     * @returns The configuration object, or null if not found.
     */
    getModuleConfig<T = unknown>(moduleName: string): T | null;
}

/**
 * Loads page scripts declared via meta[name="pk-script"] elements.
 *
 * @param doc - The document to scan for page script meta tags.
 */
function loadPageScripts(doc: Document | DocumentFragment): void {
    const pageScriptMetas = doc.querySelectorAll('meta[name="pk-script"]');
    for (const meta of pageScriptMetas) {
        const scriptUrl = meta.getAttribute('content');
        const partialName = meta.getAttribute('data-partial-name');
        if (scriptUrl) {
            void getGlobalPageContext().loadModule(scriptUrl, partialName ?? undefined).catch((err: unknown) => {
                console.error('[PPFramework] Failed to load page script:', err);
            });
        }
    }
}

/** Tracks widget script URLs that have already been loaded. */
const loadedWidgetScripts = new Set<string>();

/**
 * Loads classic (non-module) widget scripts declared via
 * meta[name="pk-widget-script"] tags. Scripts already loaded are skipped.
 * After all scripts finish loading, dispatches a piko:widgetinit event so
 * init scripts can re-scan the DOM for new widget instances.
 *
 * @param doc - The document to scan for widget script meta tags.
 */
function loadWidgetScripts(doc: Document | DocumentFragment): void {
    const metas = doc.querySelectorAll('meta[name="pk-widget-script"]');
    if (metas.length === 0) {
        return;
    }

    let pendingCount = 0;

    function onScriptReady(): void {
        pendingCount--;
        if (pendingCount <= 0) {
            document.dispatchEvent(new Event('piko:widgetinit'));
        }
    }

    for (const meta of metas) {
        const src = meta.getAttribute('content');
        if (!src || loadedWidgetScripts.has(src)) {
            continue;
        }
        loadedWidgetScripts.add(src);
        pendingCount++;

        const script = document.createElement('script');
        script.src = src;
        script.async = true;
        script.defer = true;
        script.onload = onScriptReady;
        script.onerror = onScriptReady;
        document.head.appendChild(script);
    }

    if (pendingCount === 0) {
        document.dispatchEvent(new Event('piko:widgetinit'));
    }
}

/**
 * Bundles all services used by the framework, including both eagerly and
 * lazily initialised dependencies.
 */
interface FrameworkServices {
    /** The hook manager for analytics events. */
    hookManager: HookManager;
    /** The sprite sheet manager for SVG merging. */
    spriteSheetManager: SpriteSheetManager;
    /** The module loader for ES module scripts. */
    moduleLoader: ModuleLoader;
    /** The link header parser for preload hints. */
    linkHeaderParser: LinkHeaderParser;
    /** The modal manager for dialog handling. */
    modalManager: ModalManager;
    /** The global helper registry. */
    helperRegistry: HelperRegistry;
    /** The network status monitor (lazy). */
    networkStatus: NetworkStatus | null;
    /** The accessibility announcer (lazy). */
    a11yAnnouncer: A11yAnnouncer | null;
    /** The form state manager (lazy). */
    formStateManager: FormStateManager | null;
    /** The loader UI for navigation progress (lazy). */
    loader: LoaderUI | null;
    /** The error display for user-facing messages (lazy). */
    errorDisplay: ErrorDisplay | null;
    /** The fetch client for HTTP requests (lazy). */
    fetchClient: FetchClient | null;
    /** The SPA router (lazy). */
    router: Router | null;
    /** The remote renderer for partial updates (lazy). */
    remoteRenderer: RemoteRenderer | null;
    /** The DOM binder for event delegation (lazy). */
    domBinder: DOMBinder | null;
    /** The sync partial manager for auto-refreshing partials (lazy). */
    syncPartialManager: SyncPartialManager | null;
    /** The global configuration options. */
    globalConfig: PPFrameworkOptions;
    /** The cached module configuration from the DOM. */
    moduleConfigCache: Record<string, unknown> | null;
}

/**
 * Stable-key extractor for SPA navigation morphing.
 *
 * Matches data-stable-id, then p-key, then id. Mirrors RemoteRenderer so that
 * full-page navigations and server-driven fragment patches use the same
 * identity rules, preserving scroll, focus, and stateful subtrees across
 * navigation when keys agree.
 */
function navigationNodeKey(node: Node): NodeKey {
    if (node.nodeType !== Node.ELEMENT_NODE) {
        return null;
    }
    const el = node as HTMLElement;
    return el.dataset.stableId ?? el.getAttribute('p-key') ?? (el.id || null);
}

/**
 * Scrolls to an anchor element synchronously.
 *
 * @param hash - The hash fragment (e.g., "#section").
 */
function scrollToAnchor(hash: string): void {
    if (!hash || hash === '#') {
        return;
    }

    const elementId = hash.slice(1);
    const element = document.getElementById(elementId);

    if (element) {
        element.scrollIntoView({behavior: 'instant', block: 'start'});
    }
}

/**
 * Handle scroll position restoration synchronously after DOM replacement.
 *
 * @param scrollOptions - The options controlling scroll behaviour.
 */
function handleScrollPosition(scrollOptions: PageLoadScrollOptions): void {
    if (scrollOptions.restoreScrollY !== undefined) {
        window.scrollTo({top: scrollOptions.restoreScrollY, behavior: 'instant'});
    } else if (scrollOptions.hash) {
        scrollToAnchor(scrollOptions.hash);
    } else {
        window.scrollTo({top: 0, behavior: 'instant'});
    }
}

/**
 * Dependencies required by the DOM update function.
 */
interface DOMUpdateDeps {
    /** The callback that binds event handlers and observers to a root element. */
    bindDOM: (root: HTMLElement) => void;
    /** The module loader for loading ES module scripts. */
    moduleLoader: ModuleLoader;
}

/**
 * Perform the DOM update operations synchronously during page navigation.
 *
 * @param deps - The service dependencies for the update.
 * @param parsedDocument - The parsed document from the server.
 * @param oldAppRoot - The existing #app element.
 * @param newAppRoot - The new #app element from the parsed document.
 * @param scrollOptions - The options controlling scroll restoration.
 */
function performDOMUpdate(
    deps: DOMUpdateDeps,
    parsedDocument: Document,
    oldAppRoot: Element,
    newAppRoot: Element,
    scrollOptions: PageLoadScrollOptions
): void {
    _runPageCleanup();

    getGlobalPageContext().clear();

    if (scrollOptions.morph === 'none') {
        oldAppRoot.innerHTML = newAppRoot.innerHTML;
    } else {
        fragmentMorpher(oldAppRoot as HTMLElement, newAppRoot, {
            childrenOnly: true,
            getNodeKey: navigationNodeKey
        });
    }

    handleScrollPosition(scrollOptions);

    deps.bindDOM(oldAppRoot as HTMLElement);

    deps.moduleLoader.loadFromDocument(parsedDocument);

    loadPageScripts(parsedDocument);

    registerPartialInstancesFromDOM(parsedDocument);

    _executeConnectedForPartials(oldAppRoot);

    loadWidgetScripts(parsedDocument);

    const newPageStyle = parsedDocument.querySelector('style[pk-page]');
    const oldPageStyle = document.head.querySelector('style[pk-page]');

    if (oldPageStyle) {
        oldPageStyle.remove();
    }

    if (newPageStyle) {
        const clonedStyle = newPageStyle.cloneNode(true) as HTMLStyleElement;
        document.head.appendChild(clonedStyle);
    }

    const newTitle = parsedDocument.querySelector('title');
    if (newTitle) {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Node.textContent is string | null per DOM spec
        document.title = (newTitle.textContent ?? '').trim();
    }
}

/**
 * Dependencies required by the page load handler.
 */
interface PageLoadDeps {
    /** The sprite sheet manager for merging SVG sprites. */
    spriteSheetManager: SpriteSheetManager;
    /** The callback that binds event handlers and observers to a root element. */
    bindDOM: (root: HTMLElement) => void;
    /** The module loader for loading ES module scripts. */
    moduleLoader: ModuleLoader;
}

/**
 * Handle a page load after SPA navigation by merging sprites,
 * replacing DOM content, and triggering View Transitions if available.
 *
 * @param deps - The service dependencies for the page load.
 * @param parsedDocument - The parsed document from the server.
 * @param _targetUrl - The URL that was navigated to.
 * @param scrollOptions - The options controlling scroll restoration.
 */
async function handlePageLoad(
    deps: PageLoadDeps,
    parsedDocument: Document,
    _targetUrl: string,
    scrollOptions: PageLoadScrollOptions
): Promise<void> {
    const newSpriteSheet = parsedDocument.getElementById('sprite') as SVGSVGElement | null;
    deps.spriteSheetManager.merge(newSpriteSheet);

    const newAppRoot = parsedDocument.querySelector('#app');
    const oldAppRoot = document.querySelector('#app');

    if (!oldAppRoot || !newAppRoot) {
        return;
    }

    const domUpdateDeps: DOMUpdateDeps = {bindDOM: deps.bindDOM, moduleLoader: deps.moduleLoader};

    if ('startViewTransition' in document && typeof document.startViewTransition === 'function') {
        const transition = document.startViewTransition(() => {
            performDOMUpdate(domUpdateDeps, parsedDocument, oldAppRoot, newAppRoot, scrollOptions);
        });
        await transition.updateCallbackDone;
    } else {
        performDOMUpdate(domUpdateDeps, parsedDocument, oldAppRoot, newAppRoot, scrollOptions);
    }
}

/**
 * Initialise all lazy framework services and store them in the services bag.
 *
 * @param services - The mutable services bag to populate.
 * @param options - The framework configuration options.
 * @param instance - The framework instance for callback wiring.
 */
function initFrameworkServices(
    services: FrameworkServices,
    options: PPFrameworkOptions,
    instance: PPFrameworkInstance
): void {
    services.globalConfig = options;

    services.loader = createLoaderUI({colour: options.loaderColour ?? '#29e'});
    services.errorDisplay = createErrorDisplay();

    services.networkStatus = createNetworkStatus({hookManager: services.hookManager});
    services.a11yAnnouncer = createA11yAnnouncer();
    services.formStateManager = createFormStateManager({hookManager: services.hookManager});

    services.fetchClient = createFetchClient();

    const bindDOM = (root: HTMLElement): void => {
        services.domBinder?.bind(root);
        services.syncPartialManager?.bind(root);
    };

    _registerDOMUpdater(bindDOM);

    services.remoteRenderer = createRemoteRenderer({
        moduleLoader: services.moduleLoader,
        spriteSheetManager: services.spriteSheetManager,
        linkHeaderParser: services.linkHeaderParser,
        onDOMUpdated: bindDOM,
        hookManager: services.hookManager
    });

    const pageLoadDeps: PageLoadDeps = {
        spriteSheetManager: services.spriteSheetManager,
        bindDOM,
        moduleLoader: services.moduleLoader
    };

    services.router = createRouter({
        fetchClient: services.fetchClient,
        loader: services.loader,
        errorDisplay: services.errorDisplay,
        onPageLoad: (doc, url, scroll) => handlePageLoad(pageLoadDeps, doc, url, scroll),
        hookManager: services.hookManager,
        formStateManager: services.formStateManager,
        a11yAnnouncer: services.a11yAnnouncer
    });

    services.router.setConfig({
        beforeNavigate: options.beforeNavigate,
        afterNavigate: options.afterNavigate
    });

    services.domBinder = createDOMBinder(services.helperRegistry, {
        onNavigate: (url, _event, linkOptions) => {
            void instance.navigateTo(url, undefined, {morph: linkOptions.morph});
        },
        onOpenModal: (opts: OpenModalOptions) => {
            void services.modalManager.openIfAvailable({
                selector: opts.selector,
                params: opts.params,
                title: opts.title,
                message: opts.message,
                cancelLabel: opts.cancelLabel,
                confirmLabel: opts.confirmLabel,
                confirmAction: opts.confirmAction,
                triggerElement: opts.element
            });
        }
    });

    services.syncPartialManager = createSyncPartialManager({
        onRemoteRender: (renderOptions) => instance.remoteRender(renderOptions)
    });
}

/**
 * Perform initial DOM setup after services are created.
 *
 * @param services - The framework services bag.
 */
function initFrameworkDOM(services: FrameworkServices): void {
    services.spriteSheetManager.ensureExists();

    initModuleLoaderFromPage(services.moduleLoader);

    loadPageScripts(document);

    loadWidgetScripts(document);

    registerPartialInstancesFromDOM(document);

    _executeConnectedForPartials(document);

    const appRoot = document.querySelector('#app') as HTMLElement | null;
    if (appRoot) {
        services.domBinder?.bind(appRoot);
        services.syncPartialManager?.bind(appRoot);
    }

    services.formStateManager?.scanAndTrackForms();

    services.hookManager.processQueue();

    services.hookManager.emit(HookEvent.FRAMEWORK_READY, {
        version: '1.0.0',
        loadTime: performance.now(),
        timestamp: Date.now()
    });

    services.hookManager.emit(HookEvent.PAGE_VIEW, {
        url: window.location.href,
        title: document.title,
        referrer: document.referrer,
        isInitialLoad: true,
        timestamp: Date.now()
    });
}

/**
 * Create the initial services bag with eager services populated and
 * lazy services set to null.
 *
 * @returns The framework services bag.
 */
function createInitialServices(): FrameworkServices {
    const hookManager = createHookManager();

    return {
        hookManager,
        spriteSheetManager: createSpriteSheetManager(),
        moduleLoader: createModuleLoader(),
        linkHeaderParser: createLinkHeaderParser(),
        modalManager: createModalManager({hookManager}),
        helperRegistry: globalHelperRegistry,
        networkStatus: null,
        a11yAnnouncer: null,
        formStateManager: null,
        loader: null,
        errorDisplay: null,
        fetchClient: null,
        router: null,
        remoteRenderer: null,
        domBinder: null,
        syncPartialManager: null,
        globalConfig: {},
        moduleConfigCache: null
    };
}

/**
 * Build the framework instance object wired to the given services bag.
 *
 * @param services - The mutable services bag shared by all methods.
 * @returns The framework instance.
 */
function buildFrameworkInstance(services: FrameworkServices): PPFrameworkInstance {
    const instance: PPFrameworkInstance = {
        /** Gets the set of loaded module script URLs. */
        get loadedModuleScripts() { return services.moduleLoader.getLoadedModules(); },
        /** Gets whether a navigation is currently in progress. */
        get navigating() { return services.router?.isNavigating() ?? false; },
        /** No-op setter retained for backwards compatibility. */
        set navigating(_value: boolean) {},
        /** Gets the loader bar element from the DOM. */
        get loaderElement() { return document.getElementById('ppf-loader-bar') as HTMLDivElement | null; },
        /** No-op setter retained for backwards compatibility. */
        set loaderElement(_value: HTMLDivElement | null) {},
        /** Gets the current AbortController from the fetch client. */
        get currentAbortController() { return services.fetchClient?.getController() ?? null; },
        /** No-op setter retained for backwards compatibility. */
        set currentAbortController(_value: AbortController | null) {},
        /** Gets the global configuration options. */
        get globalConfig() { return services.globalConfig; },
        /** Sets the global configuration and updates the router. */
        set globalConfig(value: PPFrameworkOptions) {
            services.globalConfig = value;
            services.router?.setConfig({beforeNavigate: value.beforeNavigate, afterNavigate: value.afterNavigate});
        },
        hooks: services.hookManager.api,
        emitHook: services.hookManager.emit,
        registerHelper: services.helperRegistry.register.bind(services.helperRegistry),
        /** Gets whether the browser is currently online. */
        get isOnline() { return services.networkStatus?.isOnline ?? navigator.onLine; },
        getModuleConfig: <T = unknown>(moduleName: string): T | null => getModuleConfig<T>(services, moduleName),
        init(options: PPFrameworkOptions = {}) {
            initFrameworkServices(services, options, instance);
            initFrameworkDOM(services);
        },
        async navigateTo(targetUrl: string, evt?: Event, options: NavigateOptions = {}): Promise<void> {
            if (!services.router) {
                console.warn('PPFramework: navigateTo called before init()');
                return;
            }
            return services.router.navigateTo(targetUrl, evt, options);
        },
        async remoteRender(options: RemoteRenderOptions): Promise<void> {
            if (!services.remoteRenderer) {
                console.warn('PPFramework: remoteRender called before init()');
                return;
            }
            return services.remoteRenderer.render(options);
        },
        dispatchAction: (actionName: string, element: HTMLElement, event?: Event) => {
            dispatchActionImpl(actionName, element, event);
        },
        buildRemoteUrl,
        addFragmentQuery,
        isSameDomain,
        assetSrc: (src: string, moduleName?: string): string => resolveAssetSrc(src, moduleName),
        createLoaderIndicator(color: string) {
            if (services.loader) { services.loader.destroy(); }
            services.loader = createLoaderUI({colour: color});
        },
        toggleLoader(isVisible: boolean) {
            if (!services.loader) { return; }
            if (isVisible) { services.loader.show(); } else { services.loader.hide(); }
        },
        updateProgressBar(percentValue: number) { services.loader?.setProgress(percentValue); },
        displayError(message: string) { services.errorDisplay?.show(message); },
        loadModuleScripts(doc: Document) { services.moduleLoader.loadFromDocument(doc); },
        patchPartial(htmlString: string, cssSelector: string) { services.remoteRenderer?.patchPartial(htmlString, cssSelector); },
        async openModalIfAvailable(options: ModalRequestOptions): Promise<void> {
            return services.modalManager.openIfAvailable(options);
        },
        executeHelper: (event: Event, actionString: string, element: HTMLElement) => {
            executeHelperImpl(event, actionString, element);
        },
        executeServerHelper(name: string, args: unknown[], triggerElement: HTMLElement, event?: Event) {
            const stringArgs = args.map(a => String(a));
            void services.helperRegistry.execute(name, triggerElement, event as Event, stringArgs);
        }
    };

    return instance;
}

/**
 * Creates the PPFramework singleton instance with all services wired together.
 *
 * @returns The framework instance.
 */
function createPPFramework(): PPFrameworkInstance {
    const services = createInitialServices();
    return buildFrameworkInstance(services);
}

/**
 * Retrieve the module configuration for the given module name from the DOM cache.
 *
 * @param services - The framework services bag containing the cache.
 * @param moduleName - The module to retrieve configuration for.
 * @returns The configuration object, or null if not found.
 */
function getModuleConfig<T = unknown>(services: FrameworkServices, moduleName: string): T | null {
    if (services.moduleConfigCache === null) {
        const configEl = document.getElementById('pk-module-config');
        if (configEl?.textContent) {
            try {
                services.moduleConfigCache = JSON.parse(configEl.textContent) as Record<string, unknown>;
            } catch {
                console.warn('[PPFramework] Failed to parse module config JSON');
                services.moduleConfigCache = {};
            }
        } else {
            services.moduleConfigCache = {};
        }
    }
    return (services.moduleConfigCache[moduleName] as T) ?? null;
}

/**
 * Dispatch a server action from compiled template event handlers.
 *
 * @param actionName - The registered action name (e.g., "contact.send").
 * @param element - The element that triggered the event.
 * @param event - The original DOM event.
 */
function dispatchActionImpl(
    actionName: string,
    element: HTMLElement,
    event?: Event
): void {
    if (element.tagName === 'FORM' || (element as HTMLButtonElement).type === 'submit') {
        event?.preventDefault();
    }

    const actionFn = getActionFunction(actionName);
    if (actionFn) {
        const form = element.closest('form') as HTMLFormElement | null;
        let formData: Record<string, unknown> | undefined;
        if (form) {
            const fd = new FormData(form);
            formData = {};
            for (const [key, value] of fd.entries()) {
                formData[key] = value;
            }
        }

        const args = formData ? [formData] : [];
        const result = actionFn(...args);
        handleAction(result, element, event).catch((err: unknown) => {
            console.error(`[PPFramework] dispatchAction failed for "${actionName}":`, err);
        });
        return;
    }

    console.warn(`[PPFramework] Action function "${actionName}" not found in registry.`);
}

/**
 * Execute a helper action from a p-on attribute string.
 *
 * @param event - The DOM event that triggered the helper.
 * @param actionString - The action string to parse and execute.
 * @param element - The element that triggered the helper.
 */
function executeHelperImpl(event: Event, actionString: string, element: HTMLElement): void {
    event.preventDefault();

    const match = actionString.match(/(\w+)(?:\((.*)\))?/);
    if (!match) {
        console.warn(`PPFramework: Could not parse helper action string: "${actionString}"`);
        return;
    }

    const helperName = match[1];
    const paramsStr = (match as (string | undefined)[])[2] ?? '';

    const args = paramsStr
        .split(',')
        .map(p => p.trim())
        .filter(p => p);

    void globalHelperRegistry.execute(helperName, element, event, args);
}

/**
 * Resolve an asset source path by prepending the asset serve path.
 *
 * @param src - The asset source path.
 * @param moduleName - An optional module name for resolving @/ aliases.
 * @returns The resolved asset URL.
 */
function resolveAssetSrc(src: string, moduleName?: string): string {
    if (!src ||
        src.startsWith('http://') ||
        src.startsWith('https://') ||
        src.startsWith('data:') ||
        src.startsWith('/')) {
        return src;
    }
    let resolvedSrc = src;
    if (src.startsWith('@/') && moduleName) {
        resolvedSrc = `${moduleName}/${src.slice(2)}`;
    }
    return `/_piko/assets/${resolvedSrc}`;
}

/** Global PPFramework singleton instance. */
export const PPFramework = createPPFramework();

document.addEventListener('DOMContentLoaded', () => {
});
