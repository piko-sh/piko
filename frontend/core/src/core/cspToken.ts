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

/** Selects the elements whose loading a Content Security Policy token controls. */
const TOKEN_CONTROLLED_SELECTOR = 'script, style, link[rel~="modulepreload"]';

/** The page token of each document, recorded when the runtime starts. */
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
    const script = doc.querySelector<HTMLScriptElement>('script[nonce]');
    if (!script) {
        return '';
    }
    const token = script.nonce ?? '';
    if (token !== '') {
        return token;
    }
    return script.getAttribute('nonce') ?? '';
}

/**
 * Records the page's Content Security Policy token while the document still holds only
 * the elements the server sent. Later reads use this token, so an element the page
 * inserts afterwards, which may carry another token, can never become the page token.
 *
 * Call it once when the runtime starts.
 *
 * @param doc - The document to read from.
 */
export function capturePageCSPToken(doc: Document = document): void {
    capturedPageTokens.set(doc, readTokenFromScripts(doc));
}

/**
 * Returns the Content Security Policy token the server gave this document. This is the
 * token recorded when the runtime started, or, before that, the token of the document's
 * first script element that has one.
 *
 * @param doc - The document to read from.
 * @returns The token, or an empty string when the page has none.
 */
export function readPageCSPToken(doc: Document = document): string {
    return capturedPageTokens.get(doc) ?? readTokenFromScripts(doc);
}

/**
 * Sets the given token on an element and removes any token attribute it arrived with.
 *
 * @param element - The element to give the token to.
 * @param token - The page token.
 */
function setToken(element: HTMLElement | SVGElement, token: string): void {
    element.removeAttribute('nonce');
    element.nonce = token;
}

/**
 * Gives an element the runtime inserts, such as a style, script or preload link, the
 * page's Content Security Policy token, so a token-based policy lets it load. An element
 * copied from a fetched document would otherwise carry no token, or the token of that
 * other response. A token the element arrived with is never kept. Its attribute is
 * removed, so it can neither load the element nor later be read as the page's token.
 *
 * @param element - The element to give the token to.
 * @param doc - The live document the element goes into. An element copied from a fetched
 * document still belongs to that document until it is inserted, so its own document
 * cannot be used.
 * @returns The same element, for chaining.
 */
export function applyPageCSPToken<T extends HTMLElement | SVGElement>(element: T, doc: Document = document): T {
    setToken(element, readPageCSPToken(doc));
    return element;
}

/**
 * Gives every script, style and module preload element in a subtree the runtime is about
 * to insert the page's Content Security Policy token, in place of any token it arrived
 * with. The page token is read once for the whole subtree.
 *
 * @param root - The root of the subtree, which may itself be such an element.
 * @param doc - The live document the subtree goes into.
 */
export function applyPageCSPTokenToTree(root: Node, doc: Document = document): void {
    if (root.nodeType !== Node.ELEMENT_NODE && root.nodeType !== Node.DOCUMENT_FRAGMENT_NODE) {
        return;
    }
    const token = readPageCSPToken(doc);
    if (root.nodeType === Node.ELEMENT_NODE && (root as Element).matches(TOKEN_CONTROLLED_SELECTOR)) {
        setToken(root as HTMLElement | SVGElement, token);
    }
    for (const element of (root as ParentNode).querySelectorAll<HTMLElement | SVGElement>(TOKEN_CONTROLLED_SELECTOR)) {
        setToken(element, token);
    }
}

/**
 * Parses HTML into a fragment owned by the live document, with every script, style and
 * module preload element in it carrying the page's Content Security Policy token. The
 * result can be inserted without any element ever being live with another token.
 *
 * @param html - The HTML to parse.
 * @param doc - The live document the fragment goes into.
 * @returns The parsed fragment.
 */
export function parseWithPageCSPToken(html: string, doc: Document = document): DocumentFragment {
    const template = doc.createElement('template');
    template.innerHTML = html;
    const fragment = doc.importNode(template.content, true);
    applyPageCSPTokenToTree(fragment, doc);
    return fragment;
}

/**
 * Copies nodes into a fragment owned by the live document, with every script, style and
 * module preload element in the copies carrying the page's Content Security Policy token.
 *
 * @param nodes - The nodes to copy, often from a fetched document.
 * @param doc - The live document the fragment goes into.
 * @returns The fragment holding the copies.
 */
export function cloneWithPageCSPToken(nodes: Iterable<Node>, doc: Document = document): DocumentFragment {
    const fragment = doc.createDocumentFragment();
    for (const node of nodes) {
        fragment.appendChild(doc.importNode(node, true));
    }
    applyPageCSPTokenToTree(fragment, doc);
    return fragment;
}
