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

import type {HelperRegistry, PPHelper} from './HelperRegistry';
import {findClosestMatch, getGlobalPageContext} from './PageContext';
import {isActionDescriptor, getActionFunction} from '@/pk/action';
import {handleAction} from '@/core/ActionExecutor';
import {formData as createFormData} from '@/pk/form';

const BASE64_BLOCK_SIZE = 4;

/** Represents an argument passed to an action. */
export interface ActionArg {
    /** Argument type: 's' for static, 'v' for variable, 'e' for $event, 'f' for $form. */
    t: 's' | 'v' | 'e' | 'f';
    /** Argument value. Optional for 'e'/'f' types which are injected at runtime. */
    v?: unknown;
}

/** Payload for an action binding. */
interface ActionPayload {
    /** Function name to invoke. */
    f: string;
    /** Arguments to pass to the function. */
    a?: ActionArg[];
}

/**
 * Resolves action arguments, injecting the event and form objects where placeholders appear.
 * Implements position-aware injection based on $event and $form placeholders.
 * @param args - The action arguments from the payload.
 * @param event - The browser event to inject for 'e' type arguments.
 * @param element - The element that triggered the event, used for form ancestor lookup.
 * @returns An array of resolved values with event and form injected at correct positions.
 */
function resolveArgsWithEvent(args: ActionArg[], event: Event, element?: HTMLElement): unknown[] {
    return args.map(a => {
        if (a.t === 'e') {
            return event;
        }
        if (a.t === 'f') {
            if (!element) {
                console.warn('[DOMBinder] $form used but no element context available');
                return null;
            }
            const form = element.closest('form');
            if (!form) {
                console.warn('[DOMBinder] $form used but no form ancestor found for element:', element);
                return null;
            }
            return createFormData(form);
        }
        return a.v;
    });
}

/**
 * Resolves action arguments for registered action functions from actions.gen.js.
 * Converts $form to a plain object via .toObject() because generated action functions
 * expect typed input objects rather than FormDataHandle.
 * @param args - The action arguments from the payload.
 * @param event - The browser event to inject for 'e' type arguments.
 * @param element - The element that triggered the event, used for form ancestor lookup.
 * @returns An array of resolved values with form data converted to plain objects.
 */
function resolveArgsForAction(args: ActionArg[], event: Event, element?: HTMLElement): unknown[] {
    return args.map(a => {
        if (a.t === 'e') {
            return event;
        }
        if (a.t === 'f') {
            if (!element) {
                console.warn('[DOMBinder] $form used but no element context available');
                return {};
            }
            const form = element.closest('form');
            if (!form) {
                console.warn('[DOMBinder] $form used but no form ancestor found for element:', element);
                return {};
            }
            return createFormData(form).toObject();
        }
        return a.v;
    });
}

/** Options for opening a modal dialogue. */
export interface OpenModalOptions {
    /** CSS selector for the modal template. */
    selector: string;
    /** Key-value parameters to pass to the modal. */
    params: Map<string, string>;
    /** Modal dialogue title. */
    title: string;
    /** Modal dialogue message content. */
    message: string;
    /** Label for the cancel button. */
    cancelLabel: string;
    /** Label for the confirm button. */
    confirmLabel: string;
    /** Action to execute on confirmation. */
    confirmAction: string;
    /** Element that triggered the modal. */
    element: HTMLElement;
}

/** Callbacks invoked by the DOMBinder for navigation and modal events. */
export interface DOMBinderCallbacks {
    /**
     * Called when a piko:a link is clicked.
     * @param url - The navigation URL.
     * @param event - The mouse event.
     */
    onNavigate: (url: string, event: MouseEvent) => void;

    /**
     * Called when a modal should be opened.
     * @param options - The modal configuration options.
     */
    onOpenModal: (options: OpenModalOptions) => void;
}

/** Parses and binds event handlers to DOM elements. */
export interface DOMBinder {
    /**
     * Binds all handlers (links and actions) within a root element.
     * @param root - The root element to bind within.
     */
    bind(root: HTMLElement): void;

    /**
     * Binds only piko:a link handlers within a root element.
     * @param root - The root element to bind links within.
     */
    bindLinks(root: HTMLElement): void;

    /**
     * Binds only action handlers (p-on, p-event, p-modal) within a root element.
     * @param root - The root element to bind actions within.
     */
    bindActions(root: HTMLElement): void;
}

/** Event handler function type. */
type EventHandlerFunc = (event: Event) => void;

/** Result of creating an action handler, containing the event name, handler function, and listener options. */
type HandlerResult = { eventName: string; handlerFunc: EventHandlerFunc | null; listenerOptions?: AddEventListenerOptions };

/** Attribute marker indicating an element's events have been bound. */
const BOUND_MARKER = 'pk-ev-bound';

/**
 * Gets a trimmed attribute value from an element.
 * @param el - The element to read the attribute from.
 * @param attr - The attribute name.
 * @returns The trimmed attribute value, or an empty string if absent.
 */
function getAttrTrimmed(el: HTMLElement, attr: string): string {
    return el.getAttribute(attr)?.trim() ?? '';
}

/** Prefix for modal parameter attributes. */
const P_MODAL_PARAM_PREFIX = 'p-modal-param:';

/** Length of the 'p-on:' prefix. */
const P_ON_PREFIX_LEN = 5;

/** Length of the 'p-event:' prefix. */
const P_EVENT_PREFIX_LEN = 8;

/**
 * URI schemes that should be handled natively by the browser or OS.
 * These trigger platform-specific actions and should not be intercepted by SPA navigation.
 */
const NATIVE_URI_SCHEMES: readonly string[] = [
    'tel:',
    'mailto:',
    'sms:',
    'geo:',
    'webcal:',
    'facetime:',
    'facetime-audio:',
    'skype:',
    'whatsapp:',
    'viber:',
    'maps:',
    'comgooglemaps:'
];

/** URI schemes that should be blocked entirely for security reasons. */
// eslint-disable-next-line no-script-url -- These are blocked schemes, not executed
const BLOCKED_URI_SCHEMES: readonly string[] = ['javascript:', 'data:', 'blob:', 'file:'];

/**
 * Checks whether a href uses a native URI scheme.
 * @param href - The href value to check.
 * @returns True if the href starts with a native scheme.
 */
function isNativeScheme(href: string): boolean {
    const lowerHref = href.toLowerCase();
    return NATIVE_URI_SCHEMES.some(scheme => lowerHref.startsWith(scheme));
}

/**
 * Checks whether a href uses a blocked URI scheme.
 * @param href - The href value to check.
 * @returns True if the href starts with a blocked scheme.
 */
function isBlockedScheme(href: string): boolean {
    const lowerHref = href.toLowerCase();
    return BLOCKED_URI_SCHEMES.some(scheme => lowerHref.startsWith(scheme));
}

/**
 * Walks up the DOM to find the closest partial's hashed ID.
 * @param el - The starting element.
 * @returns The partial ID, or undefined if the element is not within a partial.
 */
function findPartialScope(el: HTMLElement): string | undefined {
    let current: HTMLElement | null = el;
    while (current) {
        const partialId = current.getAttribute('partial');
        if (partialId) {
            return partialId;
        }
        current = current.parentElement;
    }
    return undefined;
}

/**
 * Parses a function reference for explicit scope syntax.
 * The "@partial-name.functionName" prefix indicates a cross-partial call that broadcasts
 * to all instances; a plain "functionName" uses implicit scoping.
 * @param ref - The function reference string.
 * @returns An object containing the optional partial name and the function name.
 */
function parseFunctionReference(ref: string): { partialName: string | null; fnName: string } {
    if (ref.startsWith('@')) {
        const dotIndex = ref.indexOf('.');
        if (dotIndex > 1) {
            return {
                partialName: ref.slice(1, dotIndex),
                fnName: ref.slice(dotIndex + 1)
            };
        }
    }
    return {partialName: null, fnName: ref};
}

/**
 * Collects modal parameters from an element's p-modal-param: attributes.
 * @param el - The element to collect parameters from.
 * @returns A map of parameter names to their values.
 */
function collectModalParams(el: HTMLElement): Map<string, string> {
    const params = new Map<string, string>();
    for (const {name, value} of Array.from(el.attributes)) {
        if (name.startsWith(P_MODAL_PARAM_PREFIX)) {
            const paramName = name.slice(P_MODAL_PARAM_PREFIX.length).trim();
            params.set(paramName, value.trim());
        }
    }
    return params;
}

/** Context object passed to modifier handlers during action dispatch. */
interface ModifierContext {
    /** The parsed action payload. */
    payload: ActionPayload;
    /** The resolved action arguments. */
    resolvedArgs: ActionArg[];
    /** The element that triggered the event. */
    el: HTMLElement;
    /** The browser event. */
    event: Event;
    /** HTTP method for the action. */
    method: string;
    /** Registry for looking up helper functions. */
    helperRegistry: HelperRegistry;
    /** Callbacks for navigation and modals. */
    callbacks: DOMBinderCallbacks;
    /** Whether this is a custom event (p-event) rather than a standard event (p-on). */
    isCustomEvent: boolean;
    /** The name of the event being handled. */
    eventName: string;
}

/**
 * Executes a registered helper function with appropriate argument conversion.
 * Helpers receive (el, event, ...stringArgs) where $event becomes event.type
 * and $form becomes JSON-serialised form data.
 * @param helper - The helper function to execute.
 * @param ctx - The modifier context.
 */
function executeHelper(helper: PPHelper, ctx: ModifierContext): void {
    const args = ctx.resolvedArgs.map(a => {
        if (a.t === 'e') {
            return ctx.event.type;
        }
        if (a.t === 'f') {
            const form = ctx.el.closest('form');
            if (!form) {
                return '';
            }
            return createFormData(form).toJSON();
        }
        return String(a.v);
    });
    try {
        const result: unknown = helper(ctx.el, ctx.event, ...args);
        if (result instanceof Promise) {
            result.catch((err: unknown) => {
                console.error('[DOMBinder] Async helper execution failed:', err);
            });
        }
    } catch (err) {
        console.error('[DOMBinder] Helper execution failed:', err);
    }
}

/**
 * Checks a function return value for an ActionDescriptor and dispatches it.
 * Handles both synchronous returns and Promises, since the Go code generator wraps
 * all exported .pk functions as async.
 * @param result - The return value to check.
 * @param el - The element that triggered the action.
 * @param event - The browser event.
 * @param errorPrefix - The prefix for error log messages.
 * @param fnName - The function name for error reporting.
 */
function dispatchIfActionDescriptor(
    result: unknown,
    el: HTMLElement,
    event: Event,
    errorPrefix: string,
    fnName: string
): void {
    if (isActionDescriptor(result)) {
        handleAction(result, el, event).catch((err: unknown) => {
            console.error(`${errorPrefix} Action execution failed for "${fnName}":`, err);
        });
        return;
    }

    if (result instanceof Promise) {
        result.then((resolved: unknown) => {
            if (isActionDescriptor(resolved)) {
                return handleAction(resolved, el, event);
            }
            return undefined;
        }).catch((err: unknown) => {
            console.error(`${errorPrefix} Async action execution failed for "${fnName}":`, err);
        });
    }
}

/**
 * Executes a page function with resolved arguments and dispatches any returned ActionDescriptor.
 * @param pageFunction - The page function to execute.
 * @param fnName - The function name for error reporting.
 * @param ctx - The modifier context.
 * @param errorPrefix - The prefix for error log messages.
 */
function executePageFunction(
    pageFunction: ((...args: unknown[]) => unknown) | undefined,
    fnName: string,
    ctx: ModifierContext,
    errorPrefix: string
): void {
    if (!pageFunction) {
        return;
    }
    try {
        const args = resolveArgsWithEvent(ctx.resolvedArgs, ctx.event, ctx.el);
        const result = pageFunction(ctx.event, ...args);
        dispatchIfActionDescriptor(result, ctx.el, ctx.event, errorPrefix, fnName);
    } catch (error) {
        console.error(`${errorPrefix} Error in page handler "${fnName}":`, error);
    }
}

/**
 * Executes a registered action function from actions.gen.js.
 * Unlike page functions, action functions do not receive the event as the first argument
 * and receive $form as a plain object via .toObject().
 * @param actionFunction - The action function to execute.
 * @param fnName - The function name for error reporting.
 * @param ctx - The modifier context.
 * @param errorPrefix - The prefix for error log messages.
 */
function executeRegisteredAction(
    actionFunction: (...args: unknown[]) => unknown,
    fnName: string,
    ctx: ModifierContext,
    errorPrefix: string
): void {
    try {
        const args = resolveArgsForAction(ctx.resolvedArgs, ctx.event, ctx.el);
        const result = actionFunction(...args);
        dispatchIfActionDescriptor(result, ctx.el, ctx.event, errorPrefix, fnName);
    } catch (error) {
        console.error(`${errorPrefix} Error in action handler "${fnName}":`, error);
    }
}

/**
 * Handles a custom event (p-event) with no modifier by searching page context,
 * helper registry, and action function registry in order.
 * @param ctx - The modifier context.
 */
function handleCustomEventNoModifier(ctx: ModifierContext): void {
    const pageContext = getGlobalPageContext();
    if (pageContext.hasFunction(ctx.payload.f)) {
        executePageFunction(
            pageContext.getFunction(ctx.payload.f),
            ctx.payload.f,
            ctx,
            '[DOMBinder]'
        );
        return;
    }

    const helper = ctx.helperRegistry.get(ctx.payload.f);
    if (helper) {
        executeHelper(helper, ctx);
        return;
    }

    const actionFn = getActionFunction(ctx.payload.f);
    if (actionFn) {
        executeRegisteredAction(actionFn as (...args: unknown[]) => unknown, ctx.payload.f, ctx, '[DOMBinder]');
        return;
    }

    const available = pageContext.getExportedFunctions();
    const suggestion = findClosestMatch(ctx.payload.f, available);

    let message = `[DOMBinder] Function "${ctx.payload.f}" not found for p-event handler.`;
    message += ` Did you forget to export it?`;
    if (suggestion) {
        message += ` Did you mean "${suggestion}"?`;
    }
    if (available.length > 0) {
        message += ` Available functions: ${available.join(', ')}`;
    }
    console.warn(message);
}

/**
 * Executes a single partial function and dispatches any returned ActionDescriptor.
 * @param partialFunction - The function to execute.
 * @param ctx - The modifier context.
 * @param args - The resolved arguments.
 * @param errorLabel - A label for error reporting.
 */
function executeSinglePartialFunction(
    partialFunction: (...args: unknown[]) => unknown,
    ctx: ModifierContext,
    args: unknown[],
    errorLabel: string
): void {
    try {
        const result = partialFunction(ctx.event, ...args);
        dispatchIfActionDescriptor(result, ctx.el, ctx.event, '[DOMBinder]', errorLabel);
    } catch (error) {
        console.error(`[DOMBinder] Error in ${errorLabel}:`, error);
    }
}

/**
 * Handles an explicit cross-partial call using the "@partial-name.fn()" syntax.
 * Broadcasts to all instances of the named partial.
 * @param ctx - The modifier context.
 * @param explicitPartial - The partial name from the @ prefix.
 * @param fnName - The function name to invoke.
 */
function handleExplicitPartialCall(ctx: ModifierContext, explicitPartial: string, fnName: string): void {
    const pageContext = getGlobalPageContext();
    const fns = pageContext.getFunctionsByPartialName(explicitPartial, fnName);

    if (fns.length > 0) {
        const args = resolveArgsWithEvent(ctx.resolvedArgs, ctx.event, ctx.el);
        const errorLabel = `@${explicitPartial}.${fnName}`;
        fns.forEach(partialFunction => executeSinglePartialFunction(partialFunction, ctx, args, errorLabel));
        return;
    }

    const available = pageContext.getRegisteredPartialNames();
    const suggestion = findClosestMatch(explicitPartial, available);

    let message = `[DOMBinder] Partial "${explicitPartial}" not found or has no function "${fnName}".`;
    if (suggestion) {
        message += ` Did you mean "@${suggestion}"?`;
    }
    if (available.length > 0) {
        message += ` Registered partials: ${available.join(', ')}`;
    }
    console.warn(message);
}

/**
 * Handles an implicit scope call by searching the enclosing partial scope,
 * then global scope, then helper registry, then action function registry.
 * @param ctx - The modifier context.
 * @param fnName - The function name to invoke.
 */
function handleImplicitScopeCall(ctx: ModifierContext, fnName: string): void {
    const pageContext = getGlobalPageContext();
    const partialId = findPartialScope(ctx.el);
    let pageFunction: ((...args: unknown[]) => unknown) | undefined;

    if (partialId) {
        pageFunction = pageContext.getScopedFunction(fnName, partialId);
    }

    pageFunction ??= pageContext.getFunction(fnName);

    if (pageFunction) {
        executePageFunction(pageFunction, fnName, ctx, '[DOMBinder]');
        return;
    }

    const helper = ctx.helperRegistry.get(fnName);
    if (helper) {
        executeHelper(helper, ctx);
        return;
    }

    const actionFn = getActionFunction(fnName);
    if (actionFn) {
        executeRegisteredAction(actionFn as (...args: unknown[]) => unknown, fnName, ctx, '[DOMBinder]');
        return;
    }

    const available = pageContext.getExportedFunctions();
    const suggestion = findClosestMatch(fnName, available);

    let message = `[DOMBinder] Function "${fnName}" not found.`;
    if (partialId) {
        message += ` (searched partial scope and global)`;
    }
    message += ` Did you forget to export it?`;
    if (suggestion) {
        message += ` Did you mean "${suggestion}"?`;
    }
    if (available.length > 0) {
        message += ` Available functions: ${available.join(', ')}`;
    }
    console.warn(message);
}

/**
 * Handles a standard event (p-on) with no modifier by parsing function references
 * and delegating to explicit or implicit scope handling.
 * @param ctx - The modifier context.
 */
function handleNoModifier(ctx: ModifierContext): void {
    const {partialName: explicitPartial, fnName} = parseFunctionReference(ctx.payload.f);

    if (explicitPartial) {
        handleExplicitPartialCall(ctx, explicitPartial, fnName);
    } else {
        handleImplicitScopeCall(ctx, fnName);
    }
}

/**
 * Dispatches an event handler to the appropriate resolution strategy based on event type.
 * @param ctx - The modifier context.
 */
function dispatchHandler(ctx: ModifierContext): void {
    if (ctx.isCustomEvent) {
        handleCustomEventNoModifier(ctx);
        return;
    }
    handleNoModifier(ctx);
}

/**
 * Creates a click handler for opening a modal dialogue.
 * @param el - The element with modal attributes.
 * @param callbacks - The DOMBinder callbacks.
 * @returns An event handler function.
 */
function createModalHandler(el: HTMLElement, callbacks: DOMBinderCallbacks): EventHandlerFunc {
    return () => {
        callbacks.onOpenModal({
            selector: getAttrTrimmed(el, 'p-modal:selector'),
            params: collectModalParams(el),
            title: getAttrTrimmed(el, 'p-modal:title'),
            message: getAttrTrimmed(el, 'p-modal:message'),
            cancelLabel: getAttrTrimmed(el, 'p-modal:cancel_label'),
            confirmLabel: getAttrTrimmed(el, 'p-modal:confirm_label'),
            confirmAction: getAttrTrimmed(el, 'p-modal:confirm_action'),
            element: el
        });
    };
}

/**
 * Converts a URL-safe base64 string (RFC 4648 §5) to standard base64 for atob().
 * The Go server encodes payloads with base64.RawURLEncoding which uses '-' and '_'
 * instead of '+' and '/', and omits padding. atob() requires standard base64.
 */
function urlSafeBase64ToStd(s: string): string {
    let std = s.replace(/-/g, '+').replace(/_/g, '/');
    const pad = (BASE64_BLOCK_SIZE - (std.length % BASE64_BLOCK_SIZE)) % BASE64_BLOCK_SIZE;
    std += '='.repeat(pad);
    return std;
}

/**
 * Parses a base64-encoded action payload from an element attribute.
 * @param encodedPayload - The base64-encoded payload string.
 * @param el - The element for error context.
 * @returns The parsed ActionPayload, or null if decoding fails.
 */
function parsePayload(encodedPayload: string, el: HTMLElement): ActionPayload | null {
    try {
        return JSON.parse(atob(urlSafeBase64ToStd(encodedPayload))) as ActionPayload;
    } catch (e) {
        console.error('DOMBinder: Could not decode action payload.', {encodedPayload, element: el, error: e});
        return null;
    }
}

/**
 * Creates an action handler for a p-on or p-event attribute.
 * Parses modifiers (.prevent, .stop, .once, .self, .passive, .capture) from
 * the attribute key and applies them to the event handler. The .prevent and
 * .stop modifiers fire unconditionally before the .once guard to prevent
 * navigation on second clicks. The .passive and .capture modifiers are returned
 * as listener options for addEventListener.
 * @param key - The attribute key suffix containing the event name and modifiers.
 * @param encodedPayload - The base64-encoded action payload.
 * @param isCustomEvent - A flag indicating whether this is a p-event (custom) or p-on (standard) binding.
 * @param el - The element being bound.
 * @param helperRegistry - The helper registry for function lookup.
 * @param callbacks - The DOMBinder callbacks.
 * @returns The event name, handler function, and listener options; or null handler if the key is empty.
 */
function createActionHandler(
    key: string,
    encodedPayload: string,
    isCustomEvent: boolean,
    el: HTMLElement,
    helperRegistry: HelperRegistry,
    callbacks: DOMBinderCallbacks
): HandlerResult {
    const parts = key.split('.');
    const eventName = parts[0].trim();
    const modifiers = new Set(parts.slice(1));

    if (!eventName) {
        return {eventName: '', handlerFunc: null};
    }

    const listenerOptions: AddEventListenerOptions = {};
    if (modifiers.has('capture')) {
        listenerOptions.capture = true;
    }
    if (modifiers.has('passive')) {
        listenerOptions.passive = true;
    }

    let firedOnce = false;

    const handlerFunc = (event: Event) => {
        if (modifiers.has('self') && event.target !== event.currentTarget) {
            return;
        }

        if (modifiers.has('prevent')) {
            event.preventDefault();
        }

        if (modifiers.has('stop')) {
            event.stopPropagation();
        }

        if (modifiers.has('once') && firedOnce) {
            return;
        }

        firedOnce = true;

        const payload = parsePayload(encodedPayload, el);
        if (!payload) {
            return;
        }

        const resolvedArgs = payload.a ?? [];
        const method = (el.getAttribute('data-pk-action-method') ?? 'POST').toUpperCase();

        const ctx: ModifierContext = {
            payload,
            resolvedArgs,
            el,
            event,
            method,
            helperRegistry,
            callbacks,
            isCustomEvent,
            eventName
        };

        dispatchHandler(ctx);
    };

    const hasOptions = listenerOptions.capture === true || listenerOptions.passive === true;
    return {eventName, handlerFunc, listenerOptions: hasOptions ? listenerOptions : undefined};
}

/** Entry in the handler map: handler functions and their shared listener options. */
interface HandlerEntry {
    /** The actual DOM event name. */
    eventName: string;
    /** Handler functions to invoke. */
    funcs: EventHandlerFunc[];
    /** Options to pass to addEventListener. */
    listenerOptions?: AddEventListenerOptions;
}

/**
 * Builds a grouping key for the handler map that includes both event name and
 * listener options, so handlers with different options get separate addEventListener calls.
 * @param eventName - The event name.
 * @param listenerOptions - The listener options.
 * @returns A string key for the handler map.
 */
function handlerGroupKey(eventName: string, listenerOptions?: AddEventListenerOptions): string {
    if (!listenerOptions) {
        return eventName;
    }
    let key = eventName;
    if (listenerOptions.capture) { key += '$capture'; }
    if (listenerOptions.passive) { key += '$passive'; }
    return key;
}

/**
 * Adds an event handler to the handler map, creating the entry if needed.
 * Handlers are grouped by event name + listener options combination.
 * @param handlers - The map of group keys to handler entries.
 * @param eventName - The event name.
 * @param eventHandler - The handler function to add.
 * @param listenerOptions - Optional listener options for addEventListener.
 */
function addHandler(
    handlers: Map<string, HandlerEntry>,
    eventName: string,
    eventHandler: EventHandlerFunc,
    listenerOptions?: AddEventListenerOptions
): void {
    const key = handlerGroupKey(eventName, listenerOptions);
    if (!handlers.has(key)) {
        handlers.set(key, {eventName, funcs: [], listenerOptions});
    }
    handlers.get(key)?.funcs.push(eventHandler);
}

/**
 * Creates a DOMBinder for parsing and binding event handlers to DOM elements.
 * Handles piko:a links, p-on:* standard events, p-event:* custom events, and p-modal:* modals.
 * @param helperRegistry - The registry for looking up helper functions.
 * @param callbacks - The callbacks for navigation and modal events.
 * @returns A new DOMBinder instance.
 */
export function createDOMBinder(helperRegistry: HelperRegistry, callbacks: DOMBinderCallbacks): DOMBinder {
    /**
     * Handles click events on piko:a links, blocking dangerous schemes and
     * delegating native schemes to the browser.
     * @param event - The mouse event.
     */
    const onNavigateLinkClick = (event: MouseEvent) => {
        const linkEl = event.currentTarget as HTMLAnchorElement | null;
        const href = linkEl?.getAttribute('href');

        if (!href) {
            return;
        }

        if (isBlockedScheme(href)) {
            event.preventDefault();
            console.warn('DOMBinder: Blocked navigation to dangerous URI scheme:', href);
            return;
        }

        if (isNativeScheme(href)) {
            return;
        }

        event.preventDefault();
        callbacks.onNavigate(href, event);
    };

    /**
     * Binds click handlers to all piko:a anchor elements within a root element.
     * @param rootElement - The root element to scan for links.
     */
    function bindLinks(rootElement: HTMLElement): void {
        rootElement.querySelectorAll<HTMLAnchorElement>('a').forEach(linkEl => {
            if (!linkEl.hasAttribute('piko:a')) {
                return;
            }
            linkEl.removeEventListener('click', onNavigateLinkClick);
            linkEl.addEventListener('click', onNavigateLinkClick);
        });
    }

    /**
     * Binds action handlers (p-on, p-event, p-modal) to all elements within a root element.
     * Skips elements that have already been bound.
     * @param rootElement - The root element to scan for action attributes.
     */
    function bindActions(rootElement: HTMLElement): void {
        rootElement.querySelectorAll<HTMLElement>('*').forEach(el => {
            if (el.hasAttribute(BOUND_MARKER)) {
                return;
            }

            const handlers = new Map<string, HandlerEntry>();
            let hasBound = false;

            for (const {name: attrName, value: attrValue} of Array.from(el.attributes)) {
                let result: HandlerResult | null = null;

                if (attrName.startsWith('p-on:')) {
                    result = createActionHandler(attrName.slice(P_ON_PREFIX_LEN), attrValue, false, el, helperRegistry, callbacks);
                } else if (attrName.startsWith('p-event:')) {
                    result = createActionHandler(attrName.slice(P_EVENT_PREFIX_LEN), attrValue, true, el, helperRegistry, callbacks);
                }

                if (result?.handlerFunc) {
                    addHandler(handlers, result.eventName, result.handlerFunc, result.listenerOptions);
                    hasBound = true;
                }
            }

            if (el.hasAttribute('p-modal:selector')) {
                addHandler(handlers, 'click', createModalHandler(el, callbacks));
                hasBound = true;
            }

            handlers.forEach(({eventName, funcs, listenerOptions}) => {
                el.addEventListener(eventName, event => {
                    for (const handler of funcs) {
                        try {
                            handler(event);
                        } catch (err) {
                            console.error('[DOMBinder] Event handler failed:', err);
                        }
                    }
                }, listenerOptions);
            });

            if (hasBound) {
                el.setAttribute(BOUND_MARKER, 'true');
            }
        });
    }

    return {
        bind: (root: HTMLElement) => {
            bindLinks(root);
            bindActions(root);
        },
        bindLinks,
        bindActions
    };
}
