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

import {afterEach, describe, expect, it} from 'vitest';
import {createModuleLoader} from '@/services/ModuleLoader';
import {addPageTokenScript} from '../../test/pageTokenScript';

describe('ModuleLoader', () => {
    afterEach(() => {
        document.querySelectorAll('[data-token-test], script[src="/next.js"]').forEach(element => element.remove());
    });

    it.each([
        {name: 'gives an inserted module script the page CSP token', pageToken: 'page-token', wantToken: 'page-token'},
        {name: 'inserts a module script without a token when the page has none', pageToken: '', wantToken: ''},
    ])('$name', ({pageToken, wantToken}) => {
        if (pageToken !== '') {
            addPageTokenScript(pageToken).setAttribute('data-token-test', '');
        }
        const fetched = new DOMParser().parseFromString(
            '<script type="module" src="/next.js" nonce="token-of-another-response"></script>', 'text/html');

        createModuleLoader().loadFromDocument(fetched);

        const inserted = document.body.querySelector<HTMLScriptElement>('script[src="/next.js"]');
        expect(inserted).not.toBeNull();
        expect(inserted!.nonce).toBe(wantToken);
    });
});
