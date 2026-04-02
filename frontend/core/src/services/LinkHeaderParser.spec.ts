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

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createLinkHeaderParser, type LinkHeaderParser } from './LinkHeaderParser';

describe('LinkHeaderParser', () => {
    let parser: LinkHeaderParser;

    beforeEach(() => {
        parser = createLinkHeaderParser();
    });

    afterEach(() => {
        const testLinks = document.head.querySelectorAll('link[data-test-link]');
        testLinks.forEach(link => link.remove());

        const parserLinks = document.head.querySelectorAll('link');
        parserLinks.forEach(link => {
            if (link.href.includes('/assets/') ||
                link.href.includes('/font') ||
                link.href.includes('/script') ||
                link.href.includes('/api/') ||
                link.href.includes('/css/') ||
                link.href.includes('/js/') ||
                link.href.includes('/norel') ||
                link.href.includes('/style') ||
                link.href.includes('/existing')) {
                link.remove();
            }
        });
    });

    const countLinks = (hrefPattern: string): number => {
        const links = document.head.querySelectorAll('link');
        return Array.from(links).filter(l => l.href.includes(hrefPattern)).length;
    };

    const getLink = (hrefPattern: string): HTMLLinkElement | undefined => {
        const links = document.head.querySelectorAll('link');
        return Array.from(links).find(l => l.href.includes(hrefPattern)) as HTMLLinkElement | undefined;
    };

    describe('parseAndApply()', () => {
        it('should do nothing for empty link header', () => {
            const before = document.head.querySelectorAll('link').length;
            parser.parseAndApply('');
            const after = document.head.querySelectorAll('link').length;
            expect(after).toBe(before);
        });

        it('should parse a single preload link', () => {
            parser.parseAndApply('</assets/style.css>; rel=preload; as=style');

            const link = getLink('/assets/style.css');
            expect(link).toBeDefined();
            expect(link!.rel).toBe('preload');
            expect(link!.getAttribute('as')).toBe('style');
        });

        it('should parse multiple comma-separated links', () => {
            parser.parseAndApply('</font.woff2>; rel=preload; as=font, </script.js>; rel=preload; as=script');

            expect(countLinks('/font.woff2')).toBe(1);
            expect(countLinks('/script.js')).toBe(1);

            const fontLink = getLink('/font.woff2');
            expect(fontLink!.rel).toBe('preload');
            expect(fontLink!.getAttribute('as')).toBe('font');

            const scriptLink = getLink('/script.js');
            expect(scriptLink!.rel).toBe('preload');
            expect(scriptLink!.getAttribute('as')).toBe('script');
        });

        it('should handle crossorigin attribute without value', () => {
            parser.parseAndApply('</font-cross.woff2>; rel=preload; as=font; crossorigin');

            const link = getLink('/font-cross.woff2');
            expect(link).toBeDefined();
            expect(link!.crossOrigin).toBe('');
        });

        it('should handle crossorigin attribute with value', () => {
            parser.parseAndApply('</api/data>; rel=preconnect; crossorigin=use-credentials');

            const link = getLink('/api/data');
            expect(link).toBeDefined();
            expect(link!.crossOrigin).toBe('use-credentials');
        });

        it('should handle type attribute', () => {
            parser.parseAndApply('</font-typed.woff2>; rel=preload; as=font; type=font/woff2');

            const link = getLink('/font-typed.woff2');
            expect(link).toBeDefined();
            expect(link!.type).toBe('font/woff2');
        });

        it('should skip links with malformed URL (no angle brackets)', () => {
            const before = document.head.querySelectorAll('link').length;
            parser.parseAndApply('/no-brackets.css; rel=stylesheet');
            const after = document.head.querySelectorAll('link').length;
            expect(after).toBe(before);
        });

        it('should skip duplicate links that already exist in document', () => {
            const existingLink = document.createElement('link');
            existingLink.href = '/existing.css';
            document.head.appendChild(existingLink);

            const before = countLinks('/existing.css');
            expect(before).toBe(1);

            parser.parseAndApply('</existing.css>; rel=stylesheet');

            expect(countLinks('/existing.css')).toBe(1);
        });

        it('should handle quoted parameter values', () => {
            parser.parseAndApply('</style-quoted.css>; rel=stylesheet; type="text/css"');

            const link = getLink('/style-quoted.css');
            expect(link).toBeDefined();
            expect(link!.rel).toBe('stylesheet');
            expect(link!.type).toBe('text/css');
        });

        it('should handle links without rel attribute', () => {
            parser.parseAndApply('</norel.css>');

            const link = getLink('/norel.css');
            expect(link).toBeDefined();
            expect(link!.rel).toBe('');
        });

        it('should handle complex real-world Link headers', () => {
            parser.parseAndApply(
                '</fonts/inter.woff2>; rel=preload; as=font; type=font/woff2; crossorigin, ' +
                '</css/main.css>; rel=preload; as=style, ' +
                '</js/app.js>; rel=modulepreload'
            );

            const fontLink = getLink('/fonts/inter.woff2');
            expect(fontLink).toBeDefined();
            expect(fontLink!.rel).toBe('preload');
            expect(fontLink!.getAttribute('as')).toBe('font');
            expect(fontLink!.type).toBe('font/woff2');
            expect(fontLink!.crossOrigin).toBe('');

            const cssLink = getLink('/css/main.css');
            expect(cssLink).toBeDefined();
            expect(cssLink!.rel).toBe('preload');
            expect(cssLink!.getAttribute('as')).toBe('style');

            const jsLink = getLink('/js/app.js');
            expect(jsLink).toBeDefined();
            expect(jsLink!.rel).toBe('modulepreload');
        });

        it('should handle crossorigin=true as empty string', () => {
            parser.parseAndApply('</font-true.woff2>; crossorigin=true');

            const link = getLink('/font-true.woff2');
            expect(link).toBeDefined();
            expect(link!.crossOrigin).toBe('');
        });
    });
});
