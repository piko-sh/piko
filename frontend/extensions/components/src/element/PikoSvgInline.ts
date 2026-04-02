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

import {replaceElementWithTracking} from '@/vdom/renderer';

/** Cache of fetched SVG content keyed by URL. */
const svgCache = new Map<string, string>();

/** In-flight fetch promises to prevent duplicate requests for the same URL. */
const fetchPromises = new Map<string, Promise<string>>();

/**
 * Custom element that fetches and inlines SVG content for CSS styling.
 */
export class PikoSvgInline extends HTMLElement {
    /** The current SVG source URL. */
    private _src = '';
    /** Controller for cancelling in-flight fetch requests. */
    private _abortController: AbortController | null = null;

    /** Returns the list of attributes to observe for changes. */
    static get observedAttributes(): string[] {
        return ['src'];
    }

    /** Returns the current SVG source URL. */
    get src(): string {
        return this._src;
    }

    /**
     * Sets the SVG source URL and triggers a reload.
     *
     * @param value - The new source URL.
     */
    set src(value: string) {
        if (this._src !== value) {
            this._src = value;
            this.setAttribute('src', value);
            void this._loadSvg();
        }
    }

    /** Called when the element is inserted into the DOM. */
    connectedCallback(): void {
        const srcAttr = this.getAttribute('src');
        if (srcAttr && srcAttr !== this._src) {
            this._src = srcAttr;
            void this._loadSvg();
        }
    }

    /** Called when the element is removed from the DOM. */
    disconnectedCallback(): void {
        this._abortController?.abort();
        this._abortController = null;
    }

    /**
     * Called when an observed attribute changes.
     *
     * @param name - The attribute name.
     * @param oldValue - The previous attribute value.
     * @param newValue - The new attribute value.
     */
    attributeChangedCallback(name: string, oldValue: string | null, newValue: string | null): void {
        if (name === 'src' && newValue !== oldValue && newValue !== this._src) {
            this._src = newValue ?? '';
            void this._loadSvg();
        }
    }

    /**
     * Fetches and inlines the SVG from the current source URL.
     *
     * Aborts any previous in-flight request before starting a new one.
     */
    private async _loadSvg(): Promise<void> {
        const src = this._src;
        if (!src) {
            return;
        }

        this._abortController?.abort();
        this._abortController = new AbortController();
        const signal = this._abortController.signal;

        try {
            const svgContent = await this._fetchSvg(src, signal);
            if (signal.aborted) {
                return;
            }

            this._inlineSvg(svgContent);
        } catch (err) {
            if (signal.aborted) {
                return;
            }
            console.warn(`PikoSvgInline: Failed to load SVG from ${src}`, err);
            this.innerHTML = `<!-- piko-svg-inline error: failed to load ${src} -->`;
        }
    }

    /**
     * Fetches SVG content with caching and deduplication.
     *
     * Returns cached content if available. Deduplicates concurrent requests
     * for the same URL.
     *
     * @param src - The SVG source URL.
     * @param signal - The abort signal for cancellation.
     * @returns The SVG content string.
     */
    private async _fetchSvg(src: string, signal: AbortSignal): Promise<string> {
        const cached = svgCache.get(src);
        if (cached) {
            return cached;
        }

        const existingPromise = fetchPromises.get(src);
        if (existingPromise) {
            return existingPromise;
        }

        const fetchPromise = this._doFetch(src, signal);
        fetchPromises.set(src, fetchPromise);

        try {
            const result = await fetchPromise;
            svgCache.set(src, result);
            return result;
        } finally {
            fetchPromises.delete(src);
        }
    }

    /**
     * Performs the HTTP fetch and validates the response is SVG.
     *
     * @param src - The SVG source URL.
     * @param signal - The abort signal for cancellation.
     * @returns The validated SVG content string.
     */
    private async _doFetch(src: string, signal: AbortSignal): Promise<string> {
        const response = await fetch(src, {signal});
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        const text = await response.text();

        if (!text.includes('<svg')) {
            throw new Error('Response is not valid SVG');
        }

        return text;
    }

    /**
     * Parses SVG content and replaces this element with the inline SVG.
     *
     * Copies attributes from the host element to the SVG, merging classes
     * and preserving SVG-specific attributes like viewBox. Registers the
     * replacement with the VDOM renderer and watches the src attribute
     * for changes.
     *
     * @param svgContent - The raw SVG markup string.
     */
    private _inlineSvg(svgContent: string): void {
        const parser = new DOMParser();
        const doc = parser.parseFromString(svgContent, 'image/svg+xml');
        const svgElement = doc.querySelector('svg');

        if (!svgElement) {
            console.warn('PikoSvgInline: No SVG element found in response');
            return;
        }

        for (const attr of Array.from(this.attributes)) {
            if (attr.name === 'src') {
                continue;
            }
            if (attr.name === 'class') {
                const existingClass = svgElement.getAttribute('class') ?? '';
                const mergedClass = existingClass
                    ? `${existingClass} ${attr.value}`
                    : attr.value;
                svgElement.setAttribute('class', mergedClass);
            } else if (!svgElement.hasAttribute(attr.name)) {
                svgElement.setAttribute(attr.name, attr.value);
            }
        }

        replaceElementWithTracking(this, svgElement, { watchProps: ['src'] });
    }
}

/**
 * Registers the piko-svg-inline custom element.
 *
 * This function is safe to call multiple times; it only registers once.
 */
export function registerPikoSvgInline(): void {
    if (!customElements.get('piko-svg-inline')) {
        customElements.define('piko-svg-inline', PikoSvgInline);
    }
}
