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

import {afterEach, describe, expect, it, vi} from 'vitest';
import {addPageTokenScript} from '../../test/pageTokenScript';
import {
    applyPageCSPToken,
    applyPageCSPTokenToTree,
    capturePageCSPToken,
    cloneWithPageCSPToken,
    parseWithPageCSPToken,
    readPageCSPToken
} from '@/core/cspToken';

const TOKEN_SCRIPT_SELECTOR = 'script[nonce]';

function createSeparateDocument(): Document {
    return document.implementation.createHTMLDocument('separate');
}

describe('cspToken', () => {
    afterEach(() => {
        vi.restoreAllMocks();
        document.head.replaceChildren();
        document.body.replaceChildren();
    });

    describe('readPageCSPToken', () => {
        it('reads the token a browser has hidden from the attribute', () => {
            addPageTokenScript('page-token');

            expect(readPageCSPToken(document)).toBe('page-token');
        });

        it('reads the token from the attribute when it is not hidden', () => {
            const script = document.createElement('script');
            script.setAttribute('nonce', 'page-token');
            document.head.appendChild(script);

            expect(readPageCSPToken(document)).toBe('page-token');
        });

        it('is empty when the page has no token', () => {
            expect(readPageCSPToken(document)).toBe('');
        });
    });

    describe('capturePageCSPToken', () => {
        it('keeps the token recorded at start when a script with another token is added first', () => {
            const separate = createSeparateDocument();
            addPageTokenScript('page-token', separate.head);
            capturePageCSPToken(separate);

            addPageTokenScript('token-of-another-response', separate.head);

            expect(readPageCSPToken(separate)).toBe('page-token');
        });

        it('keeps an empty token recorded at start when a script with a token is added later', () => {
            const separate = createSeparateDocument();
            capturePageCSPToken(separate);

            addPageTokenScript('token-of-another-response', separate.head);

            expect(readPageCSPToken(separate)).toBe('');
        });

        it('stops searching the document once the token is recorded', () => {
            const separate = createSeparateDocument();
            addPageTokenScript('page-token', separate.head);
            capturePageCSPToken(separate);
            const query = vi.spyOn(separate, 'querySelector');

            readPageCSPToken(separate);

            expect(query).not.toHaveBeenCalled();
        });
    });

    describe('applyPageCSPToken', () => {
        it('gives an inserted element the page token in place of the token it was copied with', () => {
            addPageTokenScript('page-token');
            const style = document.createElement('style');
            style.setAttribute('nonce', 'token-of-another-response');

            const result = applyPageCSPToken(style);

            expect(result.nonce).toBe('page-token');
            expect(result.getAttribute('nonce')).not.toBe('token-of-another-response');
        });

        it('leaves the element alone when the page has no token', () => {
            const style = document.createElement('style');

            expect(applyPageCSPToken(style).nonce).toBe('');
        });
    });

    describe('applyPageCSPTokenToTree', () => {
        it('reads the page token once for the whole subtree', () => {
            addPageTokenScript('page-token');
            const root = document.createElement('div');
            root.innerHTML = '<style></style><script></script><style></style><link rel="modulepreload" href="/m.js">';
            const query = vi.spyOn(document, 'querySelector');

            applyPageCSPTokenToTree(root, document);

            expect(query.mock.calls.filter(([selector]) => selector === TOKEN_SCRIPT_SELECTOR)).toHaveLength(1);
            expect([...root.children].map(element => (element as HTMLElement).nonce)).toEqual(
                ['page-token', 'page-token', 'page-token', 'page-token']);
        });
    });

    describe('parseWithPageCSPToken', () => {
        it('gives every parsed style and script the page token in place of its own', () => {
            addPageTokenScript('page-token');

            const fragment = parseWithPageCSPToken(
                '<p>Text</p><style nonce="token-of-another-response"></style><script nonce="token-of-another-response"></script>');

            const elements = [...fragment.querySelectorAll<HTMLElement>('style, script')];
            expect(elements.map(element => element.nonce)).toEqual(['page-token', 'page-token']);
            expect(elements.some(element => element.getAttribute('nonce') === 'token-of-another-response')).toBe(false);
            expect(fragment.querySelector('p')?.ownerDocument).toBe(document);
        });
    });

    describe('cloneWithPageCSPToken', () => {
        it('copies nodes from another document with the page token on each style', () => {
            addPageTokenScript('page-token');
            const fetched = new DOMParser().parseFromString(
                '<div id="a"><style nonce="token-of-another-response"></style></div>', 'text/html');
            const source = fetched.getElementById('a')!;

            const fragment = cloneWithPageCSPToken([source]);

            expect(fragment.querySelector('style')?.nonce).toBe('page-token');
            expect(source.querySelector('style')?.getAttribute('nonce')).toBe('token-of-another-response');
            expect(fragment.firstElementChild?.ownerDocument).toBe(document);
        });
    });
});
