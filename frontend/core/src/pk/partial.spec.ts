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
import { partial } from '@/pk/partial';

const wrap = (inner: string): string =>
    `<head></head><body><div id="app" data-pageid="test">${inner}</div></body>`;

describe('partial (PK Server Partials)', () => {
    let testContainer: HTMLDivElement;
    let mockFetch: ReturnType<typeof vi.fn>;
    let originalFetch: typeof fetch;

    beforeEach(() => {
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);

        originalFetch = global.fetch;
        mockFetch = vi.fn();
        global.fetch = mockFetch as unknown as typeof fetch;
    });

    afterEach(() => {
        testContainer.remove();
        global.fetch = originalFetch;
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
                text: () => Promise.resolve(wrap('<div><p>Loaded content</p></div>'))
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
                text: () => Promise.resolve(wrap('<div><p>Content</p></div>'))
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

        it('should update element children on success (merge mode)', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap('<div><span>New content</span></div>'))
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

            resolvePromise!(wrap('<div><p>Done</p></div>'));
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
                    text: () => Promise.resolve(wrap('<div><p>First</p></div>'))
                })
                .mockResolvedValueOnce({
                    ok: true,
                    text: () => Promise.resolve(wrap('<div><p>Second</p></div>'))
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
                    text: () => Promise.resolve(wrap(`<div><p>Call ${count}</p></div>`))
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

    describe('#app drilling (regression for the nesting bug)', () => {
        it('should drill into the #app wrapper and morph the partial root, not the wrapper', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<pp-paginate partial_name="root-is-wc" partial_src="/p" total_pages="1"><pp-table>fresh</pp-table></pp-paginate>'
                ))
            });

            const partialEl = document.createElement('pp-paginate');
            partialEl.setAttribute('partial_name', 'root-is-wc');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('total_pages', '3');
            partialEl.innerHTML = '<pp-table>stale</pp-table>';
            testContainer.appendChild(partialEl);

            await partial('root-is-wc').reload();

            expect(document.querySelectorAll('[partial_name="root-is-wc"]')).toHaveLength(1);
            expect(document.querySelector('[partial_name="root-is-wc"]')).toBe(partialEl);
            expect(partialEl.querySelector('pp-paginate')).toBeNull();
            expect(partialEl.querySelector('pp-table')?.textContent).toBe('fresh');
            expect(partialEl.getAttribute('total_pages')).toBe('1');
        });

        it('should fall back to doc.body when response is not wrapped (test-shaped responses)', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<div><span>Unwrapped</span></div>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'unwrapped');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.innerHTML = '<p>Old</p>';
            testContainer.appendChild(partialEl);

            await partial('unwrapped').reload();

            expect(partialEl.querySelector('span')?.textContent).toBe('Unwrapped');
        });
    });

    describe('partial scope chain', () => {
        it('should merge server selfScope onto live parentScopes in merge mode', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="scoped-merge" partial_src="/p" partial="child_new"><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'scoped-merge');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('partial', 'child_old parent_abc');
            testContainer.appendChild(partialEl);

            await partial('scoped-merge').reload();

            expect(partialEl.getAttribute('partial')).toBe('child_new parent_abc');
        });

        it('should merge scope in attrs-only mode too', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="scoped-attrs" partial_src="/p" partial="child_new"><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'scoped-attrs');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('partial', 'child_old parent_abc');
            testContainer.appendChild(partialEl);

            await partial('scoped-attrs').reloadWithOptions({ mode: 'attrs-only' });

            expect(partialEl.getAttribute('partial')).toBe('child_new parent_abc');
        });

        it('should keep parent scope intact when partial is in preserveAttrs', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="scoped-preserve" partial_src="/p" partial="child_new"><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'scoped-preserve');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('partial', 'child_old parent_abc');
            testContainer.appendChild(partialEl);

            await partial('scoped-preserve').reloadWithOptions({
                preserveAttrs: ['partial']
            });

            expect(partialEl.getAttribute('partial')).toBe('child_old parent_abc');
        });
    });

    describe('mode: "merge" (default)', () => {
        it('should overwrite server-emitted root attrs and preserve live-only ones', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="merge-test" partial_src="/p" class="from-server" data-server="x"><span>new</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'merge-test');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('class', 'from-live');
            partialEl.setAttribute('data-client', 'kept');
            partialEl.innerHTML = '<p>old</p>';
            testContainer.appendChild(partialEl);

            await partial('merge-test').reload();

            expect(partialEl.getAttribute('class')).toBe('from-server');
            expect(partialEl.getAttribute('data-server')).toBe('x');
            expect(partialEl.getAttribute('data-client')).toBe('kept');
            expect(partialEl.querySelector('span')?.textContent).toBe('new');
            expect(partialEl.querySelector('p')).toBeNull();
        });

        it('should set a root attr to empty when the server explicitly emits it as empty', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="merge-empty" partial_src="/p" class=""><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'merge-empty');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('class', 'old-value');
            testContainer.appendChild(partialEl);

            await partial('merge-empty').reload();

            expect(partialEl.getAttribute('class')).toBe('');
        });
    });

    describe('mode: "replace"', () => {
        it('should remove live-only root attrs', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="replace-test" partial_src="/p" class="from-server"><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'replace-test');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('class', 'from-live');
            partialEl.setAttribute('data-client', 'will-be-removed');
            testContainer.appendChild(partialEl);

            await partial('replace-test').reloadWithOptions({ mode: 'replace' });

            expect(partialEl.getAttribute('class')).toBe('from-server');
            expect(partialEl.hasAttribute('data-client')).toBe(false);
        });

        it('should still honour preserveAttrs even in replace mode', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="replace-preserve" partial_src="/p" class="from-server"><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'replace-preserve');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('class', 'from-live');
            partialEl.setAttribute('data-client', 'kept');
            testContainer.appendChild(partialEl);

            await partial('replace-preserve').reloadWithOptions({
                mode: 'replace',
                preserveAttrs: ['class', 'data-client']
            });

            expect(partialEl.getAttribute('class')).toBe('from-live');
            expect(partialEl.getAttribute('data-client')).toBe('kept');
        });
    });

    describe('mode: "children-only"', () => {
        it('should leave root attrs untouched and morph children', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="children-only-test" partial_src="/p" class="from-server"><span>new</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'children-only-test');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('class', 'from-live');
            partialEl.innerHTML = '<p>old</p>';
            testContainer.appendChild(partialEl);

            await partial('children-only-test').reloadWithOptions({ mode: 'children-only' });

            expect(partialEl.getAttribute('class')).toBe('from-live');
            expect(partialEl.querySelector('span')?.textContent).toBe('new');
            expect(partialEl.querySelector('p')).toBeNull();
        });
    });

    describe('mode: "attrs-only"', () => {
        it('should refresh root attrs and leave children alone', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="attrs-only-test" partial_src="/p" class="from-server"><span>ignored</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'attrs-only-test');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('class', 'from-live');
            partialEl.innerHTML = '<p>kept</p>';
            testContainer.appendChild(partialEl);

            await partial('attrs-only-test').reloadWithOptions({ mode: 'attrs-only' });

            expect(partialEl.getAttribute('class')).toBe('from-server');
            expect(partialEl.querySelector('p')?.textContent).toBe('kept');
            expect(partialEl.querySelector('span')).toBeNull();
        });
    });

    describe('preserveAttrs', () => {
        it('should never modify named root attrs in merge mode', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="preserve-test" partial_src="/p" class="from-server" data-x="server"><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'preserve-test');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('class', 'from-live');
            partialEl.setAttribute('data-x', 'live');
            testContainer.appendChild(partialEl);

            await partial('preserve-test').reloadWithOptions({
                preserveAttrs: ['class']
            });

            expect(partialEl.getAttribute('class')).toBe('from-live');
            expect(partialEl.getAttribute('data-x')).toBe('server');
        });
    });

    describe('ownedAttrs', () => {
        it('should only sync named root attrs and leave all others alone', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="owned-test" partial_src="/p" data-a="from-server" data-b="also-server"><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'owned-test');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.setAttribute('data-a', 'live-a');
            partialEl.setAttribute('data-b', 'live-b');
            partialEl.setAttribute('data-c', 'live-c-only');
            testContainer.appendChild(partialEl);

            await partial('owned-test').reloadWithOptions({
                ownedAttrs: ['data-a']
            });

            expect(partialEl.getAttribute('data-a')).toBe('from-server');
            expect(partialEl.getAttribute('data-b')).toBe('live-b');
            expect(partialEl.getAttribute('data-c')).toBe('live-c-only');
        });
    });

    describe('NEVER_SYNC_ATTRS', () => {
        it('should never copy pk-ev-bound or pk-sync-bound from source to live', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="never-sync" partial_src="/p" pk-ev-bound="x" pk-sync-bound="y"><span>x</span></div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'never-sync');
            partialEl.setAttribute('partial_src', '/p');
            testContainer.appendChild(partialEl);

            await partial('never-sync').reload();

            expect(partialEl.hasAttribute('pk-ev-bound')).toBe(false);
            expect(partialEl.hasAttribute('pk-sync-bound')).toBe(false);
        });
    });

    describe('getNodeKey adoption (shared with RemoteRenderer)', () => {
        it('should reorder keyed children rather than destroy and recreate them', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="keyed-test" partial_src="/p">' +
                    '<li p-key="b">B</li>' +
                    '<li p-key="a">A</li>' +
                    '</div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'keyed-test');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.innerHTML =
                '<li p-key="a">A</li>' +
                '<li p-key="b">B</li>';
            testContainer.appendChild(partialEl);

            const liA = partialEl.querySelector('[p-key="a"]') as HTMLElement & { __id?: number };
            const liB = partialEl.querySelector('[p-key="b"]') as HTMLElement & { __id?: number };
            liA.__id = 1;
            liB.__id = 2;

            await partial('keyed-test').reload();

            const newLis = partialEl.querySelectorAll('li');
            expect(newLis).toHaveLength(2);
            expect((newLis[0] as HTMLElement & { __id?: number }).__id).toBe(2);
            expect((newLis[1] as HTMLElement & { __id?: number }).__id).toBe(1);
        });

        it('should namespace p-key by partial_name to avoid cross-partial collisions', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap(
                    '<div partial_name="namespaced-keys" partial_src="/p">' +
                    '<section partial_name="inner" p-key="r.0">refreshed</section>' +
                    '</div>'
                ))
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'namespaced-keys');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.innerHTML =
                '<section partial_name="inner" p-key="r.0">stale</section>';
            testContainer.appendChild(partialEl);

            const liveInner = partialEl.querySelector('[partial_name="inner"]') as HTMLElement & { __id?: number };
            liveInner.__id = 42;

            await partial('namespaced-keys').reload();

            const afterInner = partialEl.querySelector('[partial_name="inner"]') as HTMLElement & { __id?: number };
            expect(afterInner.__id).toBe(42);
            expect(afterInner.textContent).toBe('refreshed');
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
                text: () => Promise.resolve(wrap('<div><p>With data</p></div>'))
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

        it('should use partial_src for custom base URL with data params', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve(wrap('<div><p>Custom src</p></div>'))
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
                text: () => Promise.resolve(wrap('<div><p>No params</p></div>'))
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

            resolveText!(wrap('<div><p>Done</p></div>'));
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
                text: () => Promise.resolve(wrap('<div><p>Custom</p></div>'))
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
                text: () => Promise.resolve(wrap('<div><p>Params</p></div>'))
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

        it('should warn when #app is empty', async () => {
            mockFetch.mockResolvedValueOnce({
                ok: true,
                text: () => Promise.resolve('<head></head><body><div id="app"></div></body>')
            });

            const partialEl = document.createElement('div');
            partialEl.setAttribute('partial_name', 'empty-app');
            partialEl.setAttribute('partial_src', '/p');
            partialEl.innerHTML = '<p>Original</p>';
            testContainer.appendChild(partialEl);

            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            await partial('empty-app').reload();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('partial "empty-app" received empty or invalid response')
            );
            expect(partialEl.querySelector('p')?.textContent).toBe('Original');

            warnSpy.mockRestore();
        });
    });
});
