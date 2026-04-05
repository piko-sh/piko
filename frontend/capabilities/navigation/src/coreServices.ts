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

/**
 * Minimal interfaces for core services that the navigation capability
 * depends on. These are satisfied by the real implementations in the
 * core shim and injected at setup time.
 */

/** Emits typed hook events for analytics and lifecycle tracking. */
export interface HookManagerLike {
    /** Emits an event with its payload. */
    emit(event: string, payload: unknown): void;
}

/** Displays user-facing error messages. */
export interface ErrorDisplayLike {
    /** Shows an error message. */
    show(message: string): void;
    /** Clears the current error message. */
    clear(): void;
}

/** Tracks form dirty state and confirms navigation when forms are unsaved. */
export interface FormStateManagerLike {
    /** Whether any tracked forms have unsaved changes. */
    hasDirtyForms(): boolean;
    /** Prompts the user to confirm navigation with dirty forms. */
    confirmNavigation(): boolean;
    /** Scans the DOM and begins tracking all forms. */
    scanAndTrackForms(): void;
    /** Stops tracking all forms. */
    untrackAll(): void;
}

/** Loads ES module scripts and tracks loaded modules. */
export interface ModuleLoaderLike {
    /** Loads module scripts from a parsed document (fire-and-forget). */
    loadFromDocument(doc: Document): void;
    /** Loads module scripts from a parsed document (async with error tracking). */
    loadFromDocumentAsync(doc: Document): Promise<void>;
    /** Returns the set of loaded module script URLs. */
    getLoadedModules(): Set<string>;
    /** Checks whether a specific module URL has been loaded. */
    hasLoaded(url: string): boolean;
}

/** Manages SVG sprite sheets in the document. */
export interface SpriteSheetManagerLike {
    /** Merges new sprites from a fetched document into the page. */
    merge(newSpriteSheet: SVGSVGElement | null): void;
    /** Ensures the sprite sheet container element exists. */
    ensureExists(): void;
}

/** Parses HTTP Link headers and applies preload hints. */
export interface LinkHeaderParserLike {
    /** Parses a Link header value and injects preload elements. */
    parseAndApply(linkHeader: string | null): void;
}

/** Hook event constants used by navigation for analytics. */
export const NavigationHookEvent = {
    NAVIGATION_START: 'navigation:start',
    NAVIGATION_COMPLETE: 'navigation:complete',
    NAVIGATION_ERROR: 'navigation:error',
    PAGE_VIEW: 'page:view',
    PARTIAL_RENDER: 'partial:render',
    FRAMEWORK_READY: 'framework:ready',
} as const;

/** Page context for module loading and partial registration. */
export interface PageContextLike {
    /** Loads a module script and optionally associates it with a partial. */
    loadModule(url: string, partialName?: string): Promise<void>;
    /** Clears all registered modules and exports. */
    clear(): void;
    /** Registers a partial instance mapping. */
    registerPartialInstance(partialName: string, partialId: string): void;
}

/**
 * Services injected by the core shim when the navigation capability loads.
 * Provides access to framework internals needed for routing and rendering.
 */
export interface NavigationCoreServices {
    /** Callback for handling page loads after SPA navigation. */
    onPageLoad: (doc: Document, targetUrl: string, scrollOptions: unknown) => void | Promise<void>;
    /** Hook manager for emitting analytics events. */
    hookManager: HookManagerLike;
    /** Error display for showing navigation errors. */
    errorDisplay: ErrorDisplayLike;
    /** Form state manager for dirty-form checks before navigation. */
    formStateManager: FormStateManagerLike | null;
    /** Module loader for loading page scripts after navigation. */
    moduleLoader: ModuleLoaderLike;
    /** Sprite sheet manager for SVG sprite merging. */
    spriteSheetManager: SpriteSheetManagerLike;
    /** Link header parser for preload hints. */
    linkHeaderParser: LinkHeaderParserLike;
    /** Global page context accessor. */
    getPageContext(): PageContextLike;
    /** Callback to rebind DOMBinder and SyncPartialManager after DOM update. */
    onDOMUpdated(root: HTMLElement): void;
    /** Runs page cleanup before navigation replaces content. */
    runPageCleanup(): void;
    /** Fires connected lifecycle callbacks for partials in the given root. */
    executeConnectedForPartials(root: Element): void;
    /** Fires beforeRender lifecycle callbacks. */
    executeBeforeRender(partialId: string | HTMLElement): void;
    /** Fires afterRender lifecycle callbacks. */
    executeAfterRender(partialId: string | HTMLElement): void;
    /** Fires updated lifecycle callbacks. */
    executeUpdated(partialId: string | HTMLElement, options?: Record<string, unknown>): void;
    /** Adds the fragment query parameter to a URL. */
    addFragmentQuery(url: string): string;
    /** Checks if a location is on the same domain. */
    isSameDomain(loc: Location | HTMLAnchorElement): boolean;
    /** Builds a URL with query parameters for remote rendering. */
    buildRemoteUrl(base: string, args: Record<string, string | number>): string;
}
