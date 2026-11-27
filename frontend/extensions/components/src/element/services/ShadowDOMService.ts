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

/** Reset CSS applied to all shadow DOM components. */
export const RESET_CSS = `*, *::before, *::after { margin:0; padding:0; box-sizing:border-box; } :host { display: block; }`;

/**
 * Manages shadow DOM creation and CSS injection for custom elements.
 */
export interface ShadowDOMService {
    /** Attaches shadow root if not already attached, injecting reset CSS and component CSS. Returns existing if present. */
    ensureShadowRoot(): ShadowRoot;

    /** Gets the shadow root, or null if not attached. */
    getShadowRoot(): ShadowRoot | null;

    /** Returns whether the shadow root is attached. */
    hasShadowRoot(): boolean;
}

/**
 * Options for creating a ShadowDOMService.
 */
export interface ShadowDOMServiceOptions {
    /** The host custom element. */
    host: HTMLElement;

    /** Component-specific CSS from static css getter. */
    componentCSS?: string;

    /** Shadow DOM mode: 'open' or 'closed'. */
    mode?: ShadowRootMode;

    /** Whether to delegate focus to the first focusable element in the shadow DOM. */
    delegatesFocus?: boolean;
}

/**
 * Creates a ShadowDOMService for managing shadow DOM creation and CSS injection.
 *
 * @param options - Configuration options including host and CSS.
 * @returns A new ShadowDOMService instance.
 */
export function createShadowDOMService(options: ShadowDOMServiceOptions): ShadowDOMService {
    const {host, componentCSS, mode = "open", delegatesFocus = false} = options;

    return {
        ensureShadowRoot(): ShadowRoot {
            if (host.shadowRoot) {
                return host.shadowRoot;
            }

            const shadow = host.attachShadow({mode, delegatesFocus, serializable: true});
            const cspToken = readPageCSPToken(host.ownerDocument);

            shadow.appendChild(createStyleElement(RESET_CSS, cspToken));

            if (typeof componentCSS === "string" && componentCSS.trim()) {
                shadow.appendChild(createStyleElement(componentCSS, cspToken));
            }

            return shadow;
        },

        getShadowRoot(): ShadowRoot | null {
            return host.shadowRoot;
        },

        hasShadowRoot(): boolean {
            return host.shadowRoot !== null;
        },
    };
}

/** The page token of each document, recorded when the component runtime starts. */
const capturedPageTokens = new WeakMap<Document, string>();

/**
 * Reads the Content Security Policy token from the first script element that has one.
 * Browsers hide the token attribute's value once the page has loaded, but keep it in the
 * element's `nonce` property.
 *
 * @param doc - The document to read from.
 * @returns The token, or an empty string when the page has none.
 */
function readTokenFromScripts(doc: Document): string {
    const script = doc.querySelector<HTMLScriptElement>("script[nonce]");
    if (!script) {
        return "";
    }
    const token = script.nonce ?? "";
    if (token !== "") {
        return token;
    }
    return script.getAttribute("nonce") ?? "";
}

/**
 * Records the page's Content Security Policy token once, when the component runtime
 * starts. Later reads use this token, so an element the page inserts afterwards, which
 * may carry another token, can never become the page token.
 *
 * @param doc - The page's document.
 */
export function capturePageCSPToken(doc: Document = document): void {
    capturedPageTokens.set(doc, readTokenFromScripts(doc));
}

/**
 * Returns the Content Security Policy token the server gave the page. This is the token
 * recorded when the component runtime started, or, before that, the token of the page's
 * first script element that has one.
 *
 * @param doc - The page's document.
 * @returns The token, or an empty string when the page has none.
 */
function readPageCSPToken(doc: Document): string {
    return capturedPageTokens.get(doc) ?? readTokenFromScripts(doc);
}

/**
 * Creates a style element holding the given CSS, carrying the page's CSP token so a
 * token-based policy applies it.
 *
 * @param css - The CSS text.
 * @param cspToken - The page's CSP token, or empty when it has none.
 * @returns The new style element.
 */
function createStyleElement(css: string, cspToken: string): HTMLStyleElement {
    const style = document.createElement("style");
    if (cspToken !== "") {
        style.nonce = cspToken;
    }
    style.textContent = css;
    return style;
}
