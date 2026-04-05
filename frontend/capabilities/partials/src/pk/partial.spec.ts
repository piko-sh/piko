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

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { partial, getOwnedAttributes } from '@/pk/partial';

describe('partial (PK Server Partials)', () => {
    let testContainer: HTMLDivElement;
    let mockFetch: ReturnType<typeof vi.fn>;
    let originalFetch: typeof fetch;

    beforeEach(() => {
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);

        originalFetch = globalThis.fetch;
        mockFetch = vi.fn();
        globalThis.fetch = mockFetch as unknown as typeof fetch;
    });

    afterEach(() => {
        testContainer.remove();
        globalThis.fetch = originalFetch;
        vi.clearAllMocks();
    });

    describe('partial()', () => {
        it('should return a handle with element reference', () => {
            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'test-partial');
            testContainer.appendChild(partialEl);

            const handle = partial('test-partial');

            expect(handle.element).toBe(partialEl);
        });

        it('should return null element when partial not found', () => {
            const handle = partial('nonexistent');

            expect(handle.element).toBeNull();
        });

        it('should select correct partial when multiple exist', () => {
            const partial1 = document.createElement('div');
            partial1.setAttribute('partial_name', 'partial-a');
            testContainer.appendChild(partial1);

            const partial2 = document.createElement('div');
            partial2.setAttribute('partial_name', 'partial-b');
            testContainer.appendChild(partial2);

            expect(partial('partial-a').element).toBe(partial1);
            expect(partial('partial-b').element).toBe(partial2);
        });
    });

    describe('reload()', () => {
        it('should fetch from partial_src URL', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<p>Loaded content</p>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'custom-src');
            partialEl.setAttribute('partial_src', '/api/partials/custom');
            testContainer.appendChild(partialEl);

            await partial('custom-src').reload();

            expect(mockFetch).toHaveBeenCalledWith('/api/partials/custom?_f=true');
        });

        it('should throw when no partial_src is set', async () => {
            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'default-url');
            testContainer.appendChild(partialEl);

            await expect(partial('default-url').reload()).rejects.toThrow(
                'has no partial_src attribute'
            );

            expect(mockFetch).not.toHaveBeenCalled();
        });

        it('should append query params when data provided', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<p>Content</p>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'with-params');
            partialEl.setAttribute('partial_src', '/api/partial');
            testContainer.appendChild(partialEl);

            await partial('with-params').reload({
                id: '123',
                highlight: true,
                count: 5
            });

            const calledUrl = mockFetch.mock.calls[0][0] as string;
            expect(calledUrl).toContain('/api/partial?');
            expect(calledUrl).toContain('id=123');
            expect(calledUrl).toContain('highlight=true');
            expect(calledUrl).toContain('count=5');
            expect(calledUrl).toContain('_f=true');
        });

        it('should update element children on success (morph mode)', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div><span>New content</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'update-test');
            partialEl.setAttribute('partial_src', '/_piko/partials/update-test');
            partialEl.innerHTML = '<p>Old content</p>';
            testContainer.appendChild(partialEl);

            await partial('update-test').reload();

            expect(partialEl.querySelector('span')?.textContent).toBe('New content');
            expect(partialEl.querySelector('p')).toBeNull();
        });

        it('should add pk-loading class during fetch', async () => {
            let resolvePromise: (value: string) => void;
            const textPromise = new Promise<string>(resolve => {
                resolvePromise = resolve;
            });

            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => textPromise
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'loading-test');
            partialEl.setAttribute('partial_src', '/_piko/partials/loading-test');
            testContainer.appendChild(partialEl);

            const reloadPromise = partial('loading-test').reload();

            expect(partialEl.classList.contains('pk-loading')).toBe(true);
            expect(partialEl.getAttribute('aria-busy')).toBe('true');

            resolvePromise!('<p>Done</p>');
            await reloadPromise;

            expect(partialEl.classList.contains('pk-loading')).toBe(false);
            expect(partialEl.hasAttribute('aria-busy')).toBe(false);
        });

        it('should remove loading state on error', async () => {
            mockFetch.mockRejectedValueOnce(new Error('Network error'));

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'error-loading');
            partialEl.setAttribute('partial_src', '/_piko/partials/error-loading');
            testContainer.appendChild(partialEl);

            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            await expect(partial('error-loading').reload()).rejects.toThrow();

            expect(partialEl.classList.contains('pk-loading')).toBe(false);
            expect(partialEl.hasAttribute('aria-busy')).toBe(false);

            errorSpy.mockRestore();
        });

        it('should warn and return early when partial not found', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            await partial('nonexistent').reload();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('partial "nonexistent" not found')
            );
            expect(mockFetch).not.toHaveBeenCalled();

            warnSpy.mockRestore();
        });

        it('should throw on non-ok response', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: false,
                status: 500
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'server-error');
            partialEl.setAttribute('partial_src', '/_piko/partials/server-error');
            testContainer.appendChild(partialEl);

            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            await expect(partial('server-error').reload()).rejects.toThrow(
                'Failed to reload partial: 500'
            );

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Failed to reload partial "server-error"'),
                expect.objectContaining({
                    error: expect.any(Error)
                })
            );

            errorSpy.mockRestore();
        });

        it('should handle network errors', async () => {
            mockFetch.mockRejectedValueOnce(new Error('Network failure'));

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'network-error');
            partialEl.setAttribute('partial_src', '/_piko/partials/network-error');
            testContainer.appendChild(partialEl);

            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            await expect(partial('network-error').reload()).rejects.toThrow('Network failure');

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Failed to reload partial "network-error"'),
                expect.objectContaining({
                    error: expect.any(Error)
                })
            );

            errorSpy.mockRestore();
        });
    });

    describe('integration', () => {
        it('should support sequential reloads', async () => {
            mockFetch
                .mockResolvedValueOnce({
                    ok: true,
                    text: () => Promise.resolve('<div><p>First</p></div>')
                })
                .mockResolvedValueOnce({
                    ok: true,
                    text: () => Promise.resolve('<div><p>Second</p></div>')
                });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'sequential');
            partialEl.setAttribute('partial_src', '/_piko/partials/sequential');
            testContainer.appendChild(partialEl);

            const handle = partial('sequential');

            await handle.reload();
            expect(partialEl.querySelector('p')?.textContent).toBe('First');

            await handle.reload();
            expect(partialEl.querySelector('p')?.textContent).toBe('Second');
        });

        it('should handle concurrent reloads', async () => {
            let callCount = 0;
            mockFetch.mockImplementation(() => {
                callCount++;
                const count = callCount;
                return Promise.resolve({
                    ok: true,
                    text: () => Promise.resolve(`<div><p>Call ${count}</p></div>`)
                });
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'concurrent');
            partialEl.setAttribute('partial_src', '/_piko/partials/concurrent');
            testContainer.appendChild(partialEl);

            const handle = partial('concurrent');

            await Promise.all([
                handle.reload(),
                handle.reload(),
                handle.reload()
            ]);

            expect(mockFetch).toHaveBeenCalledTimes(3);
            expect(partialEl.querySelector('p')?.textContent).toBe('Call 3');
        });
    });

    describe('getOwnedAttributes()', () => {
        it('should return undefined when pk-own-attrs is not set', () => {
            const el = document.createElement('div');
            testContainer.appendChild(el);

            expect(getOwnedAttributes(el)).toBeUndefined();
        });

        it('should return undefined when pk-own-attrs is empty string', () => {
            const el = document.createElement('div');
            el.setAttribute('pk-own-attrs', '');
            testContainer.appendChild(el);

            expect(getOwnedAttributes(el)).toBeUndefined();
        });

        it('should parse a single attribute name', () => {
            const el = document.createElement('div');
            el.setAttribute('pk-own-attrs', 'class');
            testContainer.appendChild(el);

            expect(getOwnedAttributes(el)).toEqual(['class']);
        });

        it('should parse comma-separated attribute names', () => {
            const el = document.createElement('div');
            el.setAttribute('pk-own-attrs', 'class,style,data-active');
            testContainer.appendChild(el);

            expect(getOwnedAttributes(el)).toEqual(['class', 'style', 'data-active']);
        });

        it('should trim whitespace around attribute names', () => {
            const el = document.createElement('div');
            el.setAttribute('pk-own-attrs', ' class , style , data-active ');
            testContainer.appendChild(el);

            expect(getOwnedAttributes(el)).toEqual(['class', 'style', 'data-active']);
        });

        it('should filter out empty entries from trailing commas', () => {
            const el = document.createElement('div');
            el.setAttribute('pk-own-attrs', 'class,,style,');
            testContainer.appendChild(el);

            expect(getOwnedAttributes(el)).toEqual(['class', 'style']);
        });
    });

    describe('detectRefreshLevel (via reload behaviour)', () => {
        it('should use level 0 (children-only morph) by default', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div><span>Level 0</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'level-0');
            partialEl.setAttribute('partial_src', '/_piko/partials/level-0');
            partialEl.innerHTML = '<p>Old</p>';
            testContainer.appendChild(partialEl);

            await partial('level-0').reload();

            expect(partialEl.querySelector('span')?.textContent).toBe('Level 0');
            expect(partialEl.querySelector('p')).toBeNull();
        });

        it('should detect level 1 when pk-refresh-root is present', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div class="updated"><span>Level 1</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'level-1');
            partialEl.setAttribute('partial_src', '/_piko/partials/level-1');
            partialEl.setAttribute('pk-refresh-root', '');
            partialEl.innerHTML = '<p>Old</p>';
            testContainer.appendChild(partialEl);

            await partial('level-1').reload();

            expect(partialEl.querySelector('span')?.textContent).toBe('Level 1');
        });

        it('should detect level 2 when pk-own-attrs is present', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div class="new-class" data-x="ignored"><span>Level 2</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'level-2');
            partialEl.setAttribute('partial_src', '/_piko/partials/level-2');
            partialEl.setAttribute('pk-own-attrs', 'class');
            partialEl.setAttribute('class', 'old-class');
            partialEl.setAttribute('data-x', 'kept');
            partialEl.innerHTML = '<p>Old</p>';
            testContainer.appendChild(partialEl);

            await partial('level-2').reload();

            expect(partialEl.querySelector('span')?.textContent).toBe('Level 2');
            expect(partialEl.getAttribute('class')).toBe('new-class');
            expect(partialEl.getAttribute('data-x')).toBe('kept');
        });

        it('should detect level 3 when pk-no-refresh-attrs is present', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div class="server-class"><span>Level 3</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'level-3');
            partialEl.setAttribute('partial_src', '/_piko/partials/level-3');
            partialEl.setAttribute('pk-no-refresh-attrs', 'class');
            partialEl.setAttribute('class', 'preserved-class');
            partialEl.innerHTML = '<p>Old</p>';
            testContainer.appendChild(partialEl);

            await partial('level-3').reload();

            expect(partialEl.querySelector('span')?.textContent).toBe('Level 3');
            expect(partialEl.getAttribute('class')).toBe('preserved-class');
        });

        it('should prioritise pk-no-refresh-attrs over pk-own-attrs', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div class="new"><span>Priority</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'priority-test');
            partialEl.setAttribute('partial_src', '/_piko/partials/priority-test');
            partialEl.setAttribute('pk-no-refresh-attrs', 'class');
            partialEl.setAttribute('pk-own-attrs', 'class');
            partialEl.setAttribute('class', 'original');
            testContainer.appendChild(partialEl);

            await partial('priority-test').reload();

            expect(partialEl.getAttribute('class')).toBe('original');
        });
    });

    describe('reloadWithOptions()', () => {
        it('should warn and return early when partial not found', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            await partial('missing').reloadWithOptions({});

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('partial "missing" not found')
            );
            expect(mockFetch).not.toHaveBeenCalled();

            warnSpy.mockRestore();
        });

        it('should pass data query params to the fetch URL', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div><p>With data</p></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'opts-data');
            partialEl.setAttribute('partial_src', '/_piko/partials/opts-data');
            testContainer.appendChild(partialEl);

            await partial('opts-data').reloadWithOptions({
                data: { page: '2', sort: 'name' }
            });

            const calledUrl = mockFetch.mock.calls[0][0] as string;
            expect(calledUrl).toContain('/_piko/partials/opts-data?');
            expect(calledUrl).toContain('page=2');
            expect(calledUrl).toContain('sort=name');
            expect(calledUrl).toContain('_f=true');
        });

        it('should override refresh level when level option provided', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div class="forced"><span>Forced level</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'opts-level');
            partialEl.setAttribute('partial_src', '/_piko/partials/opts-level');
            partialEl.innerHTML = '<p>Old</p>';
            testContainer.appendChild(partialEl);

            await partial('opts-level').reloadWithOptions({ level: 1 });

            expect(partialEl.querySelector('span')?.textContent).toBe('Forced level');
        });

        it('should override owned attributes when ownedAttrs option provided', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div class="new" title="new-title" data-x="new-x"><span>Custom owned</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'opts-owned');
            partialEl.setAttribute('partial_src', '/_piko/partials/opts-owned');
            partialEl.setAttribute('pk-own-attrs', 'class');
            partialEl.setAttribute('class', 'old');
            partialEl.setAttribute('title', 'old-title');
            partialEl.setAttribute('data-x', 'old-x');
            testContainer.appendChild(partialEl);

            await partial('opts-owned').reloadWithOptions({
                level: 2,
                ownedAttrs: ['title']
            });

            expect(partialEl.querySelector('span')?.textContent).toBe('Custom owned');
            expect(partialEl.getAttribute('title')).toBe('new-title');
            expect(partialEl.getAttribute('class')).toBe('old');
            expect(partialEl.getAttribute('data-x')).toBe('old-x');
        });

        it('should use partial_src for custom base URL with data params', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div><p>Custom src</p></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'opts-src');
            partialEl.setAttribute('partial_src', '/api/v2/widgets');
            testContainer.appendChild(partialEl);

            await partial('opts-src').reloadWithOptions({
                data: { filter: 'active' }
            });

            const calledUrl = mockFetch.mock.calls[0][0] as string;
            expect(calledUrl).toContain('/api/v2/widgets?');
            expect(calledUrl).toContain('filter=active');
            expect(calledUrl).toContain('_f=true');
        });

        it('should fetch without query string when no data provided', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div><p>No params</p></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'opts-nodata');
            partialEl.setAttribute('partial_src', '/api/content');
            testContainer.appendChild(partialEl);

            await partial('opts-nodata').reloadWithOptions({});

            expect(mockFetch).toHaveBeenCalledWith('/api/content?_f=true');
        });

        it('should add and remove loading state during reloadWithOptions', async () => {
            let resolveText: (value: string) => void;
            const textPromise = new Promise<string>(resolve => {
                resolveText = resolve;
            });

            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => textPromise
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'opts-loading');
            partialEl.setAttribute('partial_src', '/_piko/partials/opts-loading');
            testContainer.appendChild(partialEl);

            const reloadPromise = partial('opts-loading').reloadWithOptions({});

            expect(partialEl.classList.contains('pk-loading')).toBe(true);
            expect(partialEl.getAttribute('aria-busy')).toBe('true');

            resolveText!('<div><p>Done</p></div>');
            await reloadPromise;

            expect(partialEl.classList.contains('pk-loading')).toBe(false);
            expect(partialEl.hasAttribute('aria-busy')).toBe(false);
        });

        it('should throw and clean up on fetch error', async () => {
            mockFetch.mockRejectedValueOnce(new Error('Options fetch failure'));

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'opts-error');
            partialEl.setAttribute('partial_src', '/_piko/partials/opts-error');
            testContainer.appendChild(partialEl);

            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            await expect(
                partial('opts-error').reloadWithOptions({ data: { id: '1' } })
            ).rejects.toThrow('Options fetch failure');

            expect(partialEl.classList.contains('pk-loading')).toBe(false);
            expect(partialEl.hasAttribute('aria-busy')).toBe(false);

            errorSpy.mockRestore();
        });
    });

    describe('partial_src (custom base URL)', () => {
        it('should use partial_src as the complete fetch URL', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div><p>Custom</p></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'src-only');
            partialEl.setAttribute('partial_src', 'https://cdn.example.com/partials/widget');
            testContainer.appendChild(partialEl);

            await partial('src-only').reload();

            expect(mockFetch).toHaveBeenCalledWith('https://cdn.example.com/partials/widget?_f=true');
        });

        it('should append query params to partial_src', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div><p>Params</p></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'src-params');
            partialEl.setAttribute('partial_src', '/custom/endpoint');
            testContainer.appendChild(partialEl);

            await partial('src-params').reload({ key: 'value', num: 42 });

            const calledUrl = mockFetch.mock.calls[0][0] as string;
            expect(calledUrl).toBe('/custom/endpoint?key=value&num=42&_f=true');
        });

        it('should throw when no partial_src is set on element', async () => {
            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'fallback-name');
            testContainer.appendChild(partialEl);

            await expect(partial('fallback-name').reload()).rejects.toThrow(
                'has no partial_src attribute'
            );

            expect(mockFetch).not.toHaveBeenCalled();
        });
    });

    describe('empty or invalid response', () => {
        it('should warn when response parses to empty content', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'empty-response');
            partialEl.setAttribute('partial_src', '/_piko/partials/empty-response');
            partialEl.innerHTML = '<p>Original</p>';
            testContainer.appendChild(partialEl);

            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            await partial('empty-response').reload();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('partial "empty-response" received empty or invalid response')
            );
            expect(partialEl.querySelector('p')?.textContent).toBe('Original');

            warnSpy.mockRestore();
        });
    });
});
