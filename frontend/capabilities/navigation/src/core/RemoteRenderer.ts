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

import fragmentMorpher from '@/core/fragmentMorpher';
import {
    type DOMOperations,
    type WindowOperations,
    type HTTPOperations,
    browserDOMOperations,
    browserWindowOperations,
    browserHTTPOperations
} from '@/core/BrowserAPIs';
import {buildRemoteUrl} from '@/core/URLUtils';
import {
    NavigationHookEvent as HookEvent,
    type HookManagerLike as HookManager,
    type ModuleLoaderLike as ModuleLoader,
    type SpriteSheetManagerLike as SpriteSheetManager,
    type LinkHeaderParserLike as LinkHeaderParser,
    type PageContextLike
} from '@/coreServices';

/** Bit shift amount for simple hash calculation. */
const HASH_SHIFT = 5;

/** Radix for converting hash to string. */
const HASH_RADIX = 36;

/**
 * Computes a simple numeric hash of a string and returns it in base-36.
 *
 * @param str - The string to hash.
 * @returns The base-36 encoded hash.
 */
function sha1(str: string): string {
    let h = 0;
    const length = str.length;
    for (let i = 0; i < length; i++) {
        h = ((h << HASH_SHIFT) - h + str.charCodeAt(i)) | 0;
    }
    return h.toString(HASH_RADIX);
}

/** Primitive value types accepted in form data, including null/undefined for unset entries. */
type FormDataValue = string | number | boolean | null | undefined;

/** Plain object of form data values. */
type FormDataObject = Record<string, FormDataValue | FormDataValue[]>;

/** Accepted input types for form data conversion. */
type FormDataInput = FormData | URLSearchParams | Map<string, FormDataValue | FormDataValue[]> | FormDataObject;

/**
 * Converts various form data input types into URLSearchParams.
 *
 * @param input - The form data to convert.
 * @returns URLSearchParams, or undefined if the input type is unrecognised.
 */
function buildFormData(input: FormDataInput): URLSearchParams | undefined {
    if (input instanceof URLSearchParams) {
        return input;
    }

    const params = new URLSearchParams();

    if (input instanceof FormData) {
        for (const [key, value] of input.entries()) {
            if (typeof value === 'string') {
                params.append(key, value);
            }
        }
        return params;
    }

    if (input instanceof Map || typeof input === 'object') {
        const entries =
            input instanceof Map ? input.entries() : Object.entries(input);

        for (const [key, value] of entries) {
            if (Array.isArray(value)) {
                value.forEach(v => {
                    if (v !== null && v !== undefined) {
                        params.append(key, String(v));
                    }
                });
            } else if (value !== null && value !== undefined) {
                params.append(key, String(value));
            }
        }
        return params;
    }

    console.warn('RemoteRenderer: `options.formData` was provided but is not a recognised type. Ignoring.');
    return undefined;
}

/**
 * Inserts page-scoped style blocks from a parsed document into the live document head.
 *
 * @param parsedDoc - The parsed document containing style blocks.
 * @param domOps - DOM operations implementation.
 */
function processStyleBlocks(parsedDoc: Document, domOps: DOMOperations): void {
    const styleBlocks = parsedDoc.querySelectorAll<HTMLStyleElement>('style[pk-page]');
    styleBlocks.forEach(srcStyleEl => {
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Node.textContent is string | null per DOM spec
        const cssText = srcStyleEl.textContent ?? '';
        if (!cssText.trim()) {
            return;
        }

        const pageId = (parsedDoc.querySelector('#app') as HTMLElement | null)?.dataset.pageid;
        const styleKey = pageId ?? sha1(cssText);

        if (domOps.getHead().querySelector(`style[data-pk-style-key="${styleKey}"]`)) {
            return;
        }

        const newStyleEl = domOps.createElement('style');
        newStyleEl.setAttribute('pk-page', '');
        newStyleEl.setAttribute('data-pk-style-key', styleKey);
        newStyleEl.textContent = cssText;
        domOps.getHead().appendChild(newStyleEl);
    });
}

/**
 * Returns a stable key for a DOM node using data-stable-id, p-key, or element id.
 *
 * @param node - The node to extract a key from.
 * @returns The key string, or null if no key is available.
 */
function getNodeKey(node: Node): string | null {
    if (node.nodeType !== 1) {
        return null;
    }
    const el = node as HTMLElement;
    return el.dataset.stableId ?? el.getAttribute('p-key') ?? (el.id || null);
}

/**
 * Transforms relative p-key values in the source element tree to absolute keys
 * by replacing the source root prefix with the target container's key.
 *
 * Example:
 * - Target container has p-key="r.0:1:3:0:3:0"
 * - Source root has p-key="r.0"
 * - Source child has p-key="r.0:0:1"
 * - After transform: child has p-key="r.0:1:3:0:3:0:0:1"
 *
 * @param sourceEl - The root of the source tree.
 * @param targetEl - The target container element.
 */
function transformRelativeKeys(sourceEl: Element, targetEl: Element): void {
    const targetKey = targetEl.getAttribute('p-key');
    const sourceKey = sourceEl.getAttribute('p-key');

    if (!targetKey || !sourceKey) {
        return;
    }

    const elementsWithKeys = [sourceEl, ...Array.from(sourceEl.querySelectorAll('[p-key]'))];

    for (const el of elementsWithKeys) {
        const currentKey = el.getAttribute('p-key');
        if (!currentKey) {
            continue;
        }

        if (currentKey === sourceKey) {
            el.setAttribute('p-key', targetKey);
        } else if (currentKey.startsWith(`${sourceKey}:`)) {
            const suffix = currentKey.slice(sourceKey.length);
            el.setAttribute('p-key', targetKey + suffix);
        }
    }
}

/**
 * Syncs specified attributes from the source element onto the patch target.
 *
 * @param target - The patch target with attribute sync configuration.
 * @param sourceEl - The source element to copy attributes from.
 */
function syncPatchAttributes(target: PatchTarget, sourceEl: Element): void {
    if (!target.patchAttributes || !target.patchLocation) {
        return;
    }
    for (const attr of Array.from(sourceEl.attributes)) {
        const shouldSync = target.patchAttributes.includes(attr.name);
        const isDifferent = target.patchLocation.getAttribute(attr.name) !== attr.value;
        if (shouldSync && isDifferent) {
            target.patchLocation.setAttribute(attr.name, attr.value);
        }
    }
}

/**
 * Target specification for patching remote content into the DOM.
 */
export interface PatchTarget {
    /** CSS selector to find the source element in fetched HTML. */
    querySelector?: string;
    /** DOM element where content will be patched. */
    patchLocation?: HTMLElement;
    /** Whether to patch only children, not the root element. */
    childrenOnly?: boolean;
    /** Method for applying the patch: 'replace' or 'morph'. */
    patchMethod?: 'replace' | 'morph';
    /** Attributes to sync from source to target. */
    patchAttributes?: string[];
    /** Preserve parent CSS scopes in partial attribute during morph (Level 1+). */
    preservePartialScopes?: boolean;
    /** For Level 2: only update these attributes, preserve all others. */
    ownedAttributes?: string[];
}

/**
 * Options for fetching and rendering remote HTML content.
 */
export interface RemoteRenderOptions {
    /** URL of the remote content to fetch. */
    src: string;
    /** Query parameters to append to the URL. */
    args?: Record<string, string | number>;
    /** Form data to send as POST body. */
    formData?: FormData | URLSearchParams | Map<string, FormDataValue | FormDataValue[]> | FormDataObject;
    /** Default patch method for all targets. */
    patchMethod?: 'replace' | 'morph';
    /** Default attributes to sync for all targets. */
    patchAttributes?: string[];
    /** Default childrenOnly setting for all targets. */
    childrenOnly?: boolean;
    /** Multiple patch targets for the fetched content. */
    targets?: PatchTarget[];
    /** CSS selector for single-target rendering. */
    querySelector?: string;
    /** DOM element for single-target rendering. */
    patchLocation?: HTMLElement;
    /** Preserve parent CSS scopes in partial attribute during morph (Level 1+). */
    preservePartialScopes?: boolean;
    /** For Level 2: only update these attributes, preserve all others. */
    ownedAttributes?: string[];
}

/**
 * Dependencies for creating a RemoteRenderer.
 */
export interface RemoteRendererDependencies {
    /** Module loader for loading scripts from fetched content. */
    moduleLoader: ModuleLoader;
    /** Sprite sheet manager for merging SVG sprites. */
    spriteSheetManager: SpriteSheetManager;
    /** Parser for Link headers from responses. */
    linkHeaderParser: LinkHeaderParser;
    /** Callback invoked after DOM is updated. */
    onDOMUpdated: (root: HTMLElement) => void;
    /** DOM operations implementation. Defaults to browser document. */
    domOps?: DOMOperations;
    /** Window operations implementation. Defaults to browser window. */
    windowOps?: WindowOperations;
    /** HTTP operations implementation. Defaults to browser fetch. */
    http?: HTTPOperations;
    /** Hook manager for analytics events. */
    hookManager?: HookManager;
    /** Page context for module loading and partial registration. */
    getPageContext?: () => PageContextLike;
    /** Fires beforeRender lifecycle callbacks for a partial. */
    executeBeforeRender?: (partialId: string | HTMLElement) => void;
    /** Fires afterRender lifecycle callbacks for a partial. */
    executeAfterRender?: (partialId: string | HTMLElement) => void;
    /** Fires updated lifecycle callbacks for a partial. */
    executeUpdated?: (partialId: string | HTMLElement, options?: Record<string, unknown>) => void;
    /** Fires connected lifecycle callbacks for partials in the given root. */
    executeConnectedForPartials?: (root: Element) => void;
}

/**
 * Fetches and patches remote HTML fragments into the DOM.
 */
export interface RemoteRenderer {
    /** Fetches remote content and patches it into the DOM. */
    render(options: RemoteRenderOptions): Promise<void>;

    /** Patches an HTML string directly into the DOM at the given selector. */
    patchPartial(htmlString: string, cssSelector: string): void;
}

/**
 * Context required to apply a single patch operation.
 */
/** Lifecycle callbacks for partial render events. */
interface LifecycleCallbacks {
    /** Fires beforeRender lifecycle callbacks for a partial. */
    executeBeforeRender?: (partialId: string | HTMLElement) => void;
    /** Fires afterRender lifecycle callbacks for a partial. */
    executeAfterRender?: (partialId: string | HTMLElement) => void;
    /** Fires updated lifecycle callbacks for a partial. */
    executeUpdated?: (partialId: string | HTMLElement, options?: Record<string, unknown>) => void;
    /** Fires connected lifecycle callbacks for partials in the given root. */
    executeConnectedForPartials?: (root: Element) => void;
}

interface ApplyPatchContext {
    /** The resolved patch target with a guaranteed patchLocation. */
    target: PatchTarget & { patchLocation: HTMLElement };
    /** The source element from the fetched document. */
    sourceEl: Element;
    /** The method to use for patching. */
    patchMethod: 'replace' | 'morph';
    /** Callback invoked after DOM is updated. */
    onDOMUpdated: (root: HTMLElement) => void;
    /** DOM operations implementation. */
    domOps: DOMOperations;
    /** Lifecycle callbacks for partial render events. */
    lifecycle: LifecycleCallbacks;
}

/**
 * Applies a patch from source content to a DOM target using morph or replace.
 *
 * @param ctx - The patch context containing target, source, and dependencies.
 */
function applyPatch(ctx: ApplyPatchContext): void {
    const {target, sourceEl, patchMethod, onDOMUpdated, domOps, lifecycle} = ctx;
    const {patchLocation} = target;

    if (patchLocation.hasAttribute('partial')) {
        lifecycle.executeBeforeRender?.(patchLocation);
    }

    if (patchMethod === 'morph') {
        fragmentMorpher(patchLocation, sourceEl as HTMLElement, {
            childrenOnly: target.childrenOnly,
            preservePartialScopes: target.preservePartialScopes,
            ownedAttributes: target.ownedAttributes,
            getNodeKey,
            onNodeAdded(node) {
                if (node.nodeType === 1) {
                    onDOMUpdated(node as HTMLElement);
                }
                return node;
            },
            onBeforeElUpdated(fromEl, toEl) {
                return !fromEl.isEqualNode(toEl) && fromEl !== domOps.getActiveElement();
            }
        });
    } else {
        patchLocation.innerHTML = '';
        Array.from(sourceEl.children).forEach(child => {
            patchLocation.appendChild(child.cloneNode(true));
        });
        onDOMUpdated(patchLocation);
    }

    syncPatchAttributes(target, sourceEl);

    if (patchLocation.hasAttribute('partial')) {
        lifecycle.executeAfterRender?.(patchLocation);
        lifecycle.executeUpdated?.(patchLocation, {patchMethod});
    }
}

/**
 * Finds a source element within the fetched root, falling back to the root itself.
 *
 * @param rootEl - The root element from the fetched document.
 * @param selector - Optional CSS selector to narrow the search.
 * @returns The matched element, or the root if no selector is given.
 */
function findSourceElement(rootEl: Element, selector: string | undefined): Element | null {
    if (!selector) {
        return rootEl;
    }
    const matched = rootEl.querySelector(selector);
    if (!matched) {
        console.warn(`RemoteRenderer: selector "${selector}" not found`);
    }
    return matched ?? rootEl;
}

/**
 * Builds the list of patch targets from render options, including any inline target.
 *
 * @param options - The remote render options.
 * @returns The list of patch targets.
 */
function buildTargetsList(options: RemoteRenderOptions): PatchTarget[] {
    const targets = options.targets ?? [];
    if (options.querySelector ?? options.patchLocation) {
        targets.push({
            querySelector: options.querySelector ?? undefined,
            patchLocation: options.patchLocation ?? undefined,
            patchMethod: options.patchMethod ?? undefined,
            patchAttributes: options.patchAttributes ?? undefined,
            childrenOnly: options.childrenOnly ?? true,
            preservePartialScopes: options.preservePartialScopes ?? undefined,
            ownedAttributes: options.ownedAttributes ?? undefined
        } as PatchTarget);
    }
    return targets;
}

/**
 * Dependencies needed to fetch a remote fragment.
 */
interface FetchContext {
    /** Parser for Link headers from responses. */
    linkHeaderParser: LinkHeaderParser;
    /** HTTP operations implementation. */
    http: HTTPOperations;
    /** Window operations for page reload fallback. */
    windowOps: WindowOperations;
}

/**
 * Fetches a remote HTML fragment, validates the response type, and processes Link headers.
 *
 * @param fullUrl - The fully qualified URL to fetch.
 * @param options - Render options containing optional form data.
 * @param ctx - Fetch dependencies.
 * @returns The HTML string, or null if the fetch failed or response type was unexpected.
 */
async function fetchFragment(fullUrl: string, options: RemoteRenderOptions, ctx: FetchContext): Promise<string | null> {
    const fetchOptions: RequestInit = {method: 'GET'};

    if (options.formData) {
        fetchOptions.method = 'POST';
        const body = buildFormData(options.formData);
        if (body) {
            fetchOptions.body = body.toString();
            fetchOptions.headers = {
                'Content-Type': 'application/x-www-form-urlencoded'
            };
        }
    }

    const response = await ctx.http.fetch(fullUrl, fetchOptions);
    if (!response.ok) {
        console.error('RemoteRenderer: fetch failed', response.status, response.statusText);
        return null;
    }

    const responseType = response.headers.get('X-PP-Response-Support');
    if (responseType !== 'fragment-patch') {
        console.warn(`RemoteRenderer: expected 'fragment-patch' response but got '${responseType ?? 'none'}'. Reloading page.`);
        ctx.windowOps.locationReload();
        return null;
    }

    const linkHeader = response.headers.get('Link');
    if (linkHeader) {
        ctx.linkHeaderParser.parseAndApply(linkHeader);
    }

    return response.text();
}

/**
 * Dependencies for applying render targets to the DOM.
 */
interface ApplyRenderTargetsDeps {
    /** Callback invoked after the DOM is updated. */
    onDOMUpdated: (root: HTMLElement) => void;
    /** DOM operations implementation. */
    domOps: DOMOperations;
    /** Hook manager for analytics events. */
    hookManager?: HookManager;
    /** Lifecycle callbacks for partial render events. */
    lifecycle: LifecycleCallbacks;
}

/**
 * Reload modules that were already cached by the module loader, triggering
 * their page-context callbacks so that newly rendered partials are wired up.
 *
 * @param parsedDoc - The parsed document containing module script elements.
 * @param moduleLoader - The module loader to query and reload through.
 * @param getPageCtx - Optional accessor for the global page context.
 */
async function reloadCachedModules(parsedDoc: Document, moduleLoader: ModuleLoader, getPageCtx?: () => PageContextLike): Promise<void> {
    const moduleScripts = parsedDoc.querySelectorAll('script[type="module"]');
    const cachedScripts: string[] = [];
    for (const scriptEl of Array.from(moduleScripts)) {
        const src = scriptEl.getAttribute('src');
        if (src && moduleLoader.hasLoaded(src)) {
            cachedScripts.push(src);
        }
    }

    await moduleLoader.loadFromDocumentAsync(parsedDoc);

    if (cachedScripts.length > 0 && getPageCtx) {
        const pageContext = getPageCtx();
        for (const src of cachedScripts) {
            void pageContext.loadModule(src).catch((err: unknown) => {
                console.error('[RemoteRenderer] Failed to reload cached module:', err);
            });
        }
    }
}

/**
 * Iterate the patch targets, locate the corresponding source elements in the
 * fetched document, apply patches, and emit lifecycle hooks.
 *
 * @param targets - The list of patch targets to process.
 * @param rootEl - The root element of the fetched document.
 * @param options - The remote render options for this request.
 * @param deps - Shared dependencies from the factory closure.
 */
function applyRenderTargets(
    targets: PatchTarget[],
    rootEl: Element,
    options: RemoteRenderOptions,
    deps: ApplyRenderTargetsDeps
): void {
    for (const target of targets) {
        if (!target.patchLocation) {
            console.warn('RemoteRenderer: target has no patchLocation');
            continue;
        }

        const sourceEl = findSourceElement(rootEl, target.querySelector);
        if (!sourceEl) {
            console.warn('RemoteRenderer: no valid element in fetched HTML');
            return;
        }

        transformRelativeKeys(sourceEl, target.patchLocation);

        applyPatch({
            target: {...target, patchLocation: target.patchLocation},
            sourceEl,
            patchMethod: target.patchMethod ?? options.patchMethod ?? 'replace',
            onDOMUpdated: deps.onDOMUpdated,
            domOps: deps.domOps,
            lifecycle: deps.lifecycle
        });

        // eslint-disable-next-line @typescript-eslint/prefer-nullish-coalescing -- Need || for empty string fallback
        const patchLocationId = target.querySelector || target.patchLocation.id || 'unknown';
        deps.hookManager?.emit(HookEvent.PARTIAL_RENDER, {
            src: options.src,
            patchLocation: patchLocationId,
            timestamp: Date.now()
        });

        deps.lifecycle.executeConnectedForPartials?.(target.patchLocation);
    }
}

/**
 * Creates a RemoteRenderer instance for fetching and patching remote content.
 *
 * @param deps - Required dependencies for remote rendering.
 * @returns A new RemoteRenderer instance.
 */
export function createRemoteRenderer(deps: RemoteRendererDependencies): RemoteRenderer {
    const {moduleLoader, spriteSheetManager, linkHeaderParser, onDOMUpdated, hookManager, getPageContext, executeBeforeRender, executeAfterRender, executeUpdated, executeConnectedForPartials} = deps;
    const domOps = deps.domOps ?? browserDOMOperations;
    const windowOps = deps.windowOps ?? browserWindowOperations;
    const http = deps.http ?? browserHTTPOperations;
    const fetchCtx: FetchContext = {linkHeaderParser, http, windowOps};
    const lifecycle: LifecycleCallbacks = {executeBeforeRender, executeAfterRender, executeUpdated, executeConnectedForPartials};
    const renderTargetDeps: ApplyRenderTargetsDeps = {onDOMUpdated, domOps, hookManager, lifecycle};

    /**
     * Fetches remote content and patches it into the DOM at configured targets.
     *
     * @param options - The remote render options specifying source and targets.
     */
    async function render(options: RemoteRenderOptions): Promise<void> {
        const fullUrl = buildRemoteUrl(options.src, options.args ?? {});

        let htmlContent: string | null = null;
        try {
            htmlContent = await fetchFragment(fullUrl, options, fetchCtx);
        } catch (error) {
            console.error('RemoteRenderer: network error:', error);
            return;
        }

        if (!htmlContent) {
            return;
        }

        const parsedDoc = domOps.parseHTML(htmlContent);
        spriteSheetManager.merge(parsedDoc.getElementById('sprite') as SVGSVGElement | null);
        processStyleBlocks(parsedDoc, domOps);

        await reloadCachedModules(parsedDoc, moduleLoader, getPageContext);

        const rootEl = parsedDoc.querySelector('#app') ?? parsedDoc.documentElement;
        const targets = buildTargetsList(options);
        applyRenderTargets(targets, rootEl, options, renderTargetDeps);
    }

    /**
     * Patches an HTML string directly into the DOM at the given CSS selector.
     *
     * @param htmlString - The HTML content to patch.
     * @param cssSelector - The CSS selector to locate both source and target elements.
     */
    function patchPartial(htmlString: string, cssSelector: string): void {
        const doc = domOps.parseHTML(htmlString);
        const newPartialEl = doc.querySelector(cssSelector);
        if (!newPartialEl) {
            return console.warn(`RemoteRenderer: patchPartial - no element found for selector ${cssSelector}`);
        }
        const currentEl = domOps.querySelector(cssSelector);
        if (!currentEl) {
            return console.warn(`RemoteRenderer: patchPartial - no existing element found for selector ${cssSelector}`);
        }
        currentEl.innerHTML = newPartialEl.innerHTML;
        onDOMUpdated(currentEl as HTMLElement);

        hookManager?.emit(HookEvent.PARTIAL_RENDER, {
            src: 'inline',
            patchLocation: cssSelector,
            timestamp: Date.now()
        });
    }

    return {render, patchPartial};
}
