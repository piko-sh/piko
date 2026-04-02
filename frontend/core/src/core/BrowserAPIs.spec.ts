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

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
    browserDOMOperations,
    browserWindowOperations,
    browserHTTPOperations,
    browserAPIs,
    createBrowserAPIs,
    type DOMOperations,
    type WindowOperations,
    type HTTPOperations,
    type BrowserAPIs
} from './BrowserAPIs';

describe('BrowserAPIs', () => {
    describe('browserDOMOperations', () => {
        it('should create an element', () => {
            const div = browserDOMOperations.createElement('div');
            expect(div).toBeInstanceOf(HTMLDivElement);
        });

        it('should create a typed element', () => {
            const input = browserDOMOperations.createElement('input');
            expect(input).toBeInstanceOf(HTMLInputElement);
        });

        it('should create a custom element', () => {
            const custom = browserDOMOperations.createElement('my-custom-element');
            expect(custom).toBeInstanceOf(HTMLElement);
        });

        it('should create a text node', () => {
            const text = browserDOMOperations.createTextNode('Hello');
            expect(text).toBeInstanceOf(Text);
            expect(text.textContent).toBe('Hello');
        });

        it('should create a comment node', () => {
            const comment = browserDOMOperations.createComment('My comment');
            expect(comment).toBeInstanceOf(Comment);
            expect(comment.textContent).toBe('My comment');
        });

        it('should create a document fragment', () => {
            const fragment = browserDOMOperations.createDocumentFragment();
            expect(fragment).toBeInstanceOf(DocumentFragment);
        });

        it('should query selector on document', () => {
            const div = document.createElement('div');
            div.id = 'test-query-selector';
            document.body.appendChild(div);

            const result = browserDOMOperations.querySelector('#test-query-selector');
            expect(result).toBe(div);

            div.remove();
        });

        it('should query selector all on document', () => {
            const div1 = document.createElement('div');
            const div2 = document.createElement('div');
            div1.className = 'test-query-all';
            div2.className = 'test-query-all';
            document.body.appendChild(div1);
            document.body.appendChild(div2);

            const results = browserDOMOperations.querySelectorAll('.test-query-all');
            expect(results).toHaveLength(2);

            div1.remove();
            div2.remove();
        });

        it('should get element by ID', () => {
            const div = document.createElement('div');
            div.id = 'test-get-by-id';
            document.body.appendChild(div);

            const result = browserDOMOperations.getElementById('test-get-by-id');
            expect(result).toBe(div);

            div.remove();
        });

        it('should return null for non-existent ID', () => {
            const result = browserDOMOperations.getElementById('non-existent-id');
            expect(result).toBeNull();
        });

        it('should get document head', () => {
            const head = browserDOMOperations.getHead();
            expect(head).toBe(document.head);
        });

        it('should get active element', () => {
            const input = document.createElement('input');
            document.body.appendChild(input);
            input.focus();

            const result = browserDOMOperations.getActiveElement();
            expect(result).toBe(input);

            input.remove();
        });

        it('should parse HTML string to document', () => {
            const html = '<html><head><title>Test</title></head><body><div id="app">Content</div></body></html>';
            const doc = browserDOMOperations.parseHTML(html);

            expect(doc).toBeInstanceOf(Document);
            expect(doc.getElementById('app')?.textContent).toBe('Content');
            expect(doc.title).toBe('Test');
        });

        it('should handle partial HTML fragments', () => {
            const html = '<div class="fragment">Fragment content</div>';
            const doc = browserDOMOperations.parseHTML(html);

            expect(doc.querySelector('.fragment')?.textContent).toBe('Fragment content');
        });
    });

    describe('browserWindowOperations', () => {
        beforeEach(() => {
        });

        it('should get location', () => {
            const location = browserWindowOperations.getLocation();
            expect(location).toBe(window.location);
        });

        it('should get location origin', () => {
            const origin = browserWindowOperations.getLocationOrigin();
            expect(origin).toBe(window.location.origin);
        });

        it('should get location href', () => {
            const href = browserWindowOperations.getLocationHref();
            expect(href).toBe(window.location.href);
        });

        it('should push state to history', () => {
            const pushStateSpy = vi.spyOn(window.history, 'pushState');

            browserWindowOperations.historyPushState({ page: 1 }, '', '/new-url');

            expect(pushStateSpy).toHaveBeenCalledWith({ page: 1 }, '', '/new-url');

            pushStateSpy.mockRestore();
        });

        it('should replace state in history', () => {
            const replaceStateSpy = vi.spyOn(window.history, 'replaceState');

            browserWindowOperations.historyReplaceState({ page: 2 }, '', '/replaced-url');

            expect(replaceStateSpy).toHaveBeenCalledWith({ page: 2 }, '', '/replaced-url');

            replaceStateSpy.mockRestore();
        });

        it('should add event listener to window', () => {
            const addEventListenerSpy = vi.spyOn(window, 'addEventListener');
            const handler = () => {};

            browserWindowOperations.addEventListener('click', handler);

            expect(addEventListenerSpy).toHaveBeenCalledWith('click', handler);

            addEventListenerSpy.mockRestore();
        });

        it('should remove event listener from window', () => {
            const removeEventListenerSpy = vi.spyOn(window, 'removeEventListener');
            const handler = () => {};

            browserWindowOperations.removeEventListener('click', handler);

            expect(removeEventListenerSpy).toHaveBeenCalledWith('click', handler);

            removeEventListenerSpy.mockRestore();
        });

        it('should have setLocationHref method', () => {
            expect(typeof browserWindowOperations.setLocationHref).toBe('function');
        });

        it('should have locationReload method', () => {
            expect(typeof browserWindowOperations.locationReload).toBe('function');
        });

        it('should get history state', () => {
            window.history.pushState({testData: 'value'}, '', '/test-state');

            const state = browserWindowOperations.getHistoryState();
            expect(state).toEqual({testData: 'value'});

            window.history.back();
        });

        it('should get scroll Y position', () => {
            const scrollY = browserWindowOperations.getScrollY();
            expect(typeof scrollY).toBe('number');
            expect(scrollY).toBe(window.scrollY);
        });

        it('should scroll to position', () => {
            const scrollToSpy = vi.spyOn(window, 'scrollTo');

            browserWindowOperations.scrollTo(0, 100);

            expect(scrollToSpy).toHaveBeenCalledWith(0, 100);

            scrollToSpy.mockRestore();
        });
    });

    describe('browserHTTPOperations', () => {
        beforeEach(() => {
            vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('OK')));
        });

        afterEach(() => {
            vi.unstubAllGlobals();
        });

        it('should call global fetch', async () => {
            await browserHTTPOperations.fetch('/api/test');

            expect(fetch).toHaveBeenCalledWith('/api/test', undefined);
        });

        it('should pass fetch options', async () => {
            const options = {
                method: 'POST',
                body: JSON.stringify({ data: 'test' }),
                headers: { 'Content-Type': 'application/json' }
            };

            await browserHTTPOperations.fetch('/api/test', options);

            expect(fetch).toHaveBeenCalledWith('/api/test', options);
        });

        it('should return the fetch response', async () => {
            const mockResponse = new Response('Test response');
            vi.mocked(fetch).mockResolvedValue(mockResponse);

            const result = await browserHTTPOperations.fetch('/api/test');

            expect(result).toBe(mockResponse);
        });
    });

    describe('browserAPIs', () => {
        it('should provide all browser API operations', () => {
            expect(browserAPIs.dom).toBe(browserDOMOperations);
            expect(browserAPIs.window).toBe(browserWindowOperations);
            expect(browserAPIs.http).toBe(browserHTTPOperations);
        });

        it('should be of type BrowserAPIs', () => {
            const apis: BrowserAPIs = browserAPIs;
            expect(apis.dom).toBeDefined();
            expect(apis.window).toBeDefined();
            expect(apis.http).toBeDefined();
        });
    });

    describe('createBrowserAPIs()', () => {
        it('should return default browser APIs when no overrides', () => {
            const apis = createBrowserAPIs();

            expect(apis.dom).toBe(browserDOMOperations);
            expect(apis.window).toBe(browserWindowOperations);
            expect(apis.http).toBe(browserHTTPOperations);
        });

        it('should allow overriding dom operations', () => {
            const mockDOM: DOMOperations = {
                createElement: vi.fn(),
                createTextNode: vi.fn(),
                createComment: vi.fn(),
                createDocumentFragment: vi.fn(),
                querySelector: vi.fn(),
                querySelectorAll: vi.fn(),
                getElementById: vi.fn(),
                getHead: vi.fn(),
                getActiveElement: vi.fn(),
                parseHTML: vi.fn()
            };

            const apis = createBrowserAPIs({ dom: mockDOM });

            expect(apis.dom).toBe(mockDOM);
            expect(apis.window).toBe(browserWindowOperations);
            expect(apis.http).toBe(browserHTTPOperations);
        });

        it('should allow overriding window operations', () => {
            const mockWindow: WindowOperations = {
                getLocation: vi.fn(),
                getLocationOrigin: vi.fn(),
                getLocationHref: vi.fn(),
                setLocationHref: vi.fn(),
                locationReload: vi.fn(),
                historyPushState: vi.fn(),
                historyReplaceState: vi.fn(),
                getHistoryState: vi.fn(),
                addEventListener: vi.fn(),
                removeEventListener: vi.fn(),
                getScrollY: vi.fn(),
                scrollTo: vi.fn(),
                setScrollRestoration: vi.fn(),
                getScrollRestoration: vi.fn()
            };

            const apis = createBrowserAPIs({ window: mockWindow });

            expect(apis.dom).toBe(browserDOMOperations);
            expect(apis.window).toBe(mockWindow);
            expect(apis.http).toBe(browserHTTPOperations);
        });

        it('should allow overriding http operations', () => {
            const mockHTTP: HTTPOperations = {
                fetch: vi.fn()
            };

            const apis = createBrowserAPIs({ http: mockHTTP });

            expect(apis.dom).toBe(browserDOMOperations);
            expect(apis.window).toBe(browserWindowOperations);
            expect(apis.http).toBe(mockHTTP);
        });

        it('should allow overriding multiple operations', () => {
            const mockDOM: DOMOperations = {
                createElement: vi.fn(),
                createTextNode: vi.fn(),
                createComment: vi.fn(),
                createDocumentFragment: vi.fn(),
                querySelector: vi.fn(),
                querySelectorAll: vi.fn(),
                getElementById: vi.fn(),
                getHead: vi.fn(),
                getActiveElement: vi.fn(),
                parseHTML: vi.fn()
            };

            const mockHTTP: HTTPOperations = {
                fetch: vi.fn()
            };

            const apis = createBrowserAPIs({ dom: mockDOM, http: mockHTTP });

            expect(apis.dom).toBe(mockDOM);
            expect(apis.window).toBe(browserWindowOperations);
            expect(apis.http).toBe(mockHTTP);
        });
    });
});
