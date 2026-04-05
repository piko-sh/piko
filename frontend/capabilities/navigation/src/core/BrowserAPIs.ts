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
 * Abstraction for DOM operations.
 */
export interface DOMOperations {
    /** Creates an HTML element by tag name. */
    createElement<K extends keyof HTMLElementTagNameMap>(tagName: K): HTMLElementTagNameMap[K];
    createElement(tagName: string): HTMLElement;

    /** Creates a text node. */
    createTextNode(data: string): Text;

    /** Creates a comment node. */
    createComment(data: string): Comment;

    /** Creates a document fragment. */
    createDocumentFragment(): DocumentFragment;

    /** Queries the document for the first element matching a selector. */
    querySelector<E extends Element = Element>(selectors: string): E | null;

    /** Queries the document for all elements matching a selector. */
    querySelectorAll<E extends Element = Element>(selectors: string): NodeListOf<E>;

    /** Returns the element with the given ID. */
    getElementById(elementId: string): HTMLElement | null;

    /** Returns the document head element. */
    getHead(): HTMLHeadElement;

    /** Returns the currently focused element. */
    getActiveElement(): Element | null;

    /** Parses an HTML string into a Document. */
    parseHTML(html: string): Document;
}

/**
 * Abstraction for window and history operations.
 */
export interface WindowOperations {
    /** Returns the current Location object. */
    getLocation(): Location;

    /** Returns the location origin. */
    getLocationOrigin(): string;

    /** Returns the location href. */
    getLocationHref(): string;

    /** Sets the location href, triggering a full page navigation. */
    setLocationHref(href: string): void;

    /** Reloads the current page. */
    locationReload(): void;

    /** Pushes a new entry onto the history stack. */
    historyPushState(data: unknown, unused: string, url: string): void;

    /** Replaces the current history entry. */
    historyReplaceState(data: unknown, unused: string, url: string): void;

    /** Returns the current history state. */
    getHistoryState(): unknown;

    /** Adds an event listener to the window. */
    addEventListener(type: string, listener: EventListenerOrEventListenerObject): void;

    /** Removes an event listener from the window. */
    removeEventListener(type: string, listener: EventListenerOrEventListenerObject): void;

    /** Returns the current vertical scroll position. */
    getScrollY(): number;

    /** Scrolls the window to the specified position. */
    scrollTo(x: number, y: number): void;

    /** Sets the scroll restoration mode. */
    setScrollRestoration(mode: ScrollRestoration): void;

    /** Returns the current scroll restoration mode. */
    getScrollRestoration(): ScrollRestoration;
}

/**
 * Abstraction for HTTP operations.
 */
export interface HTTPOperations {
    /** Fetches a resource from the network. */
    fetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response>;
}

/**
 * Combined browser APIs interface.
 */
export interface BrowserAPIs {
    /** DOM operations for document manipulation. */
    dom: DOMOperations;
    /** Window operations for navigation and history. */
    window: WindowOperations;
    /** HTTP operations for network requests. */
    http: HTTPOperations;
}

/**
 * Production implementation of DOMOperations using real browser APIs.
 */
export const browserDOMOperations: DOMOperations = {
    /** Creates an HTML element by tag name. */
    createElement<K extends keyof HTMLElementTagNameMap>(tagName: K | string): HTMLElementTagNameMap[K] | HTMLElement {
        return document.createElement(tagName);
    },

    /** Creates a text node. */
    createTextNode(data: string): Text {
        return document.createTextNode(data);
    },

    /** Creates a comment node. */
    createComment(data: string): Comment {
        return document.createComment(data);
    },

    /** Creates a document fragment. */
    createDocumentFragment(): DocumentFragment {
        return document.createDocumentFragment();
    },

    /** Queries the document for the first element matching a selector. */
    querySelector<E extends Element = Element>(selectors: string): E | null {
        return document.querySelector<E>(selectors);
    },

    /** Queries the document for all elements matching a selector. */
    querySelectorAll<E extends Element = Element>(selectors: string): NodeListOf<E> {
        return document.querySelectorAll<E>(selectors);
    },

    /** Returns the element with the given ID. */
    getElementById(elementId: string): HTMLElement | null {
        return document.getElementById(elementId);
    },

    /** Returns the document head element. */
    getHead(): HTMLHeadElement {
        return document.head;
    },

    /** Returns the currently focused element. */
    getActiveElement(): Element | null {
        return document.activeElement;
    },

    /** Parses an HTML string into a Document. */
    parseHTML(html: string): Document {
        const parser = new DOMParser();
        return parser.parseFromString(html, 'text/html');
    }
};

/**
 * Production implementation of WindowOperations using real browser APIs.
 */
export const browserWindowOperations: WindowOperations = {
    /** Returns the current Location object. */
    getLocation(): Location {
        return window.location;
    },

    /** Returns the location origin. */
    getLocationOrigin(): string {
        return window.location.origin;
    },

    /** Returns the location href. */
    getLocationHref(): string {
        return window.location.href;
    },

    /** Sets the location href, triggering a full page navigation. */
    setLocationHref(href: string): void {
        window.location.href = href;
    },

    /** Reloads the current page. */
    locationReload(): void {
        window.location.reload();
    },

    /** Pushes a new entry onto the history stack. */
    historyPushState(data: unknown, unused: string, url: string): void {
        window.history.pushState(data, unused, url);
    },

    /** Replaces the current history entry. */
    historyReplaceState(data: unknown, unused: string, url: string): void {
        window.history.replaceState(data, unused, url);
    },

    /** Returns the current history state. */
    getHistoryState(): unknown {
        return window.history.state;
    },

    /** Adds an event listener to the window. */
    addEventListener(type: string, listener: EventListenerOrEventListenerObject): void {
        window.addEventListener(type, listener);
    },

    /** Removes an event listener from the window. */
    removeEventListener(type: string, listener: EventListenerOrEventListenerObject): void {
        window.removeEventListener(type, listener);
    },

    /** Returns the current vertical scroll position. */
    getScrollY(): number {
        return window.scrollY;
    },

    /** Scrolls the window to the specified position. */
    scrollTo(x: number, y: number): void {
        window.scrollTo(x, y);
    },

    /** Sets the scroll restoration mode. */
    setScrollRestoration(mode: ScrollRestoration): void {
        if ('scrollRestoration' in history) {
            history.scrollRestoration = mode;
        }
    },

    /** Returns the current scroll restoration mode. */
    getScrollRestoration(): ScrollRestoration {
        if ('scrollRestoration' in history) {
            return history.scrollRestoration;
        }
        return 'auto';
    }
};

/**
 * Production implementation of HTTPOperations using real browser APIs.
 */
export const browserHTTPOperations: HTTPOperations = {
    /** Fetches a resource from the network. */
    fetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
        return fetch(input, init);
    }
};

/**
 * Default browser APIs instance using real browser APIs.
 */
export const browserAPIs: BrowserAPIs = {
    dom: browserDOMOperations,
    window: browserWindowOperations,
    http: browserHTTPOperations
};

/**
 * Creates a custom BrowserAPIs instance with partial overrides.
 *
 * @param overrides - Partial set of API implementations to override the defaults.
 * @returns A new BrowserAPIs instance with overrides applied.
 */
export function createBrowserAPIs(overrides: Partial<BrowserAPIs> = {}): BrowserAPIs {
    return {
        dom: overrides.dom ?? browserDOMOperations,
        window: overrides.window ?? browserWindowOperations,
        http: overrides.http ?? browserHTTPOperations
    };
}
