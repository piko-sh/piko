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

type DataLayerItem = Record<string, unknown> | unknown[];

let dataLayer: DataLayerItem[];
let gtagCalls: unknown[][];

function resetGlobals(): void {
    dataLayer = [];
    gtagCalls = [];

    Object.defineProperty(window, 'dataLayer', {
        get: () => dataLayer,
        set: (value: DataLayerItem[]) => { dataLayer = value; },
        configurable: true,
    });

    (window as unknown as { gtag: (...args: unknown[]) => void }).gtag = (...args: unknown[]) => {
        gtagCalls.push(args);
    };
}

function injectPageData(data: Record<string, unknown>): HTMLScriptElement {
    const existing = document.getElementById('pk-page-data');
    if (existing) {
        existing.remove();
    }

    const script = document.createElement('script');
    script.id = 'pk-page-data';
    script.type = 'application/json';
    script.textContent = JSON.stringify(data);
    document.body.appendChild(script);
    return script;
}

function removePageData(): void {
    document.getElementById('pk-page-data')?.remove();
}

describe('getPageData', () => {
    beforeEach(resetGlobals);
    afterEach(removePageData);

    it('should return empty object when no pk-page-data element exists', async () => {
        const { getPageData } = await import('./main.testutils');
        expect(getPageData()).toEqual({});
    });

    it('should parse valid JSON from pk-page-data element', async () => {
        const testData = { property_type: 'house', property_price: 500000 };
        injectPageData(testData);

        const { getPageData } = await import('./main.testutils');
        expect(getPageData()).toEqual(testData);
    });

    it('should return empty object when pk-page-data has empty content', async () => {
        const script = document.createElement('script');
        script.id = 'pk-page-data';
        script.type = 'application/json';
        script.textContent = '';
        document.body.appendChild(script);

        const { getPageData } = await import('./main.testutils');
        expect(getPageData()).toEqual({});
    });

    it('should return empty object and warn on invalid JSON', async () => {
        const script = document.createElement('script');
        script.id = 'pk-page-data';
        script.type = 'application/json';
        script.textContent = '{invalid json}';
        document.body.appendChild(script);

        const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
        const { getPageData } = await import('./main.testutils');
        expect(getPageData()).toEqual({});
        expect(warnSpy).toHaveBeenCalledWith('[piko/analytics] Failed to parse pk-page-data JSON');
        warnSpy.mockRestore();
    });
});

describe('dispatchEvent', () => {
    beforeEach(resetGlobals);

    it('should push to dataLayer when GTM is active', async () => {
        const { dispatchEvent } = await import('./main.testutils');
        dispatchEvent(
            { hasGTM: true, hasGA4: false, debugMode: false },
            'page_view',
            { page_path: '/test' },
            'piko_page_view',
            { page_path: '/test' },
        );

        expect(dataLayer).toHaveLength(1);
        expect(dataLayer[0]).toEqual({ event: 'piko_page_view', page_path: '/test' });
        expect(gtagCalls).toHaveLength(0);
    });

    it('should call gtag when GA4 is active', async () => {
        const { dispatchEvent } = await import('./main.testutils');
        dispatchEvent(
            { hasGTM: false, hasGA4: true, debugMode: false },
            'page_view',
            { page_path: '/test' },
            'piko_page_view',
            { page_path: '/test' },
        );

        expect(gtagCalls).toHaveLength(1);
        expect(gtagCalls[0]).toEqual(['event', 'page_view', { page_path: '/test' }]);
        expect(dataLayer).toHaveLength(0);
    });

    it('should dispatch to both when GTM and GA4 are active', async () => {
        const { dispatchEvent } = await import('./main.testutils');
        dispatchEvent(
            { hasGTM: true, hasGA4: true, debugMode: false },
            'navigation',
            { event_category: 'SPA' },
            'piko_navigation',
            { page_path: '/about' },
        );

        expect(gtagCalls).toHaveLength(1);
        expect(dataLayer).toHaveLength(1);
    });

    it('should not dispatch when neither mode is active', async () => {
        const { dispatchEvent } = await import('./main.testutils');
        dispatchEvent(
            { hasGTM: false, hasGA4: false, debugMode: false },
            'page_view',
            { page_path: '/test' },
            'piko_page_view',
            { page_path: '/test' },
        );

        expect(gtagCalls).toHaveLength(0);
        expect(dataLayer).toHaveLength(0);
    });

    it('should log GTM push when debug mode is on', async () => {
        const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
        const { dispatchEvent } = await import('./main.testutils');

        dispatchEvent(
            { hasGTM: true, hasGA4: false, debugMode: true },
            'page_view',
            {},
            'piko_page_view',
            { page_path: '/debug' },
        );

        expect(warnSpy).toHaveBeenCalledWith(
            '[piko/analytics] GTM push: piko_page_view',
            { page_path: '/debug' },
        );
        warnSpy.mockRestore();
    });

    it('should not log GTM push when debug mode is off', async () => {
        const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
        const { dispatchEvent } = await import('./main.testutils');

        dispatchEvent(
            { hasGTM: true, hasGA4: false, debugMode: false },
            'page_view',
            {},
            'piko_page_view',
            { page_path: '/quiet' },
        );

        expect(warnSpy).not.toHaveBeenCalled();
        warnSpy.mockRestore();
    });
});
