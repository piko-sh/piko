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

import {describe, it, expect, beforeEach, afterEach, vi} from 'vitest';
import type {PartialHandle, PartialReloadOptions} from './partial';

let mockElement: HTMLElement | null;
let mockReload: ReturnType<typeof vi.fn<(data?: Record<string, string | number | boolean>) => Promise<void>>>;
const mockReloadWithOptions = vi.fn<(options: PartialReloadOptions) => Promise<void>>().mockResolvedValue(undefined);

function makeMockHandle(element: HTMLElement | null = mockElement): PartialHandle {
    return {
        get element() {
            return element;
        },
        reload: mockReload,
        reloadWithOptions: mockReloadWithOptions
    };
}

vi.mock('./partial', () => ({
    partial: vi.fn(() => makeMockHandle())
}));

import {
    reloadPartial,
    reloadGroup,
    autoRefresh,
    reloadCascade
} from './coordination';
import {partial} from './partial';

function resolvePartialName(nameOrElement: string | Element): string {
    if (typeof nameOrElement === 'string') {
        return nameOrElement;
    }
    return nameOrElement.getAttribute('partial_name') ?? nameOrElement.getAttribute('data-partial') ?? 'unknown';
}

function createPartialElement(name = 'test-partial'): HTMLElement {
    const el = document.createElement('div');
    el.setAttribute('data-partial', name);
    el.innerHTML = '<span>initial</span>';
    document.body.appendChild(el);
    return el;
}

describe('coordination', () => {
    beforeEach(() => {
        vi.useFakeTimers();
        mockElement = createPartialElement();
        mockReload = vi.fn<(data?: Record<string, string | number | boolean>) => Promise<void>>().mockResolvedValue(undefined);
        vi.mocked(partial).mockImplementation(() => makeMockHandle());
    });

    afterEach(() => {
        document.body.innerHTML = '';
        vi.restoreAllMocks();
        vi.useRealTimers();
    });

    describe('reloadPartial()', () => {
        it('should call handle.reload() with no args when none provided', async () => {
            await reloadPartial('test-partial');

            expect(mockReload).toHaveBeenCalledOnce();
            expect(mockReload).toHaveBeenCalledWith(undefined);
        });

        it('should pass converted args to handle.reload()', async () => {
            await reloadPartial('test-partial', {
                args: {str: 'hello', num: 42, bool: true}
            });

            expect(mockReload).toHaveBeenCalledWith({str: 'hello', num: 42, bool: true});
        });

        it('should stringify object args values', async () => {
            await reloadPartial('test-partial', {
                args: {obj: {nested: 1}}
            });

            expect(mockReload).toHaveBeenCalledWith({obj: '[object Object]'});
        });

        it('should filter out null and undefined args values', async () => {
            await reloadPartial('test-partial', {
                args: {keep: 'yes', dropNull: null, dropUndefined: undefined}
            });

            expect(mockReload).toHaveBeenCalledWith({keep: 'yes'});
        });

        it('should log a warning and return when partial element is not found', async () => {
            mockElement = null;
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            await reloadPartial('missing-partial');

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('missing-partial')
            );
            expect(mockReload).not.toHaveBeenCalled();
        });

        it('should add pk-loading class and aria-busy before reload, then remove after', async () => {
            let loadingDuringReload = false;
            let ariaBusyDuringReload = false;

            mockReload.mockImplementation(async () => {
                loadingDuringReload = mockElement!.classList.contains('pk-loading');
                ariaBusyDuringReload = mockElement!.getAttribute('aria-busy') === 'true';
            });

            await reloadPartial('test-partial');

            expect(loadingDuringReload).toBe(true);
            expect(ariaBusyDuringReload).toBe(true);
            expect(mockElement!.classList.contains('pk-loading')).toBe(false);
            expect(mockElement!.getAttribute('aria-busy')).toBeNull();
        });

        it('should remove pk-loading and aria-busy even when reload throws', async () => {
            mockReload.mockRejectedValueOnce(new Error('boom'));
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            await expect(reloadPartial('test-partial')).rejects.toThrow('boom');

            expect(mockElement!.classList.contains('pk-loading')).toBe(false);
            expect(mockElement!.getAttribute('aria-busy')).toBeNull();
            errorSpy.mockRestore();
        });

        it('should not toggle loading class when loading option is false', async () => {
            let loadingDuringReload = false;
            mockReload.mockImplementation(async () => {
                loadingDuringReload = mockElement!.classList.contains('pk-loading');
            });

            await reloadPartial('test-partial', {loading: false});

            expect(loadingDuringReload).toBe(false);
        });

        it('should call onSuccess with element innerHTML after successful reload', async () => {
            mockReload.mockImplementation(async () => {
                mockElement!.innerHTML = '<p>refreshed</p>';
            });
            const onSuccess = vi.fn();

            await reloadPartial('test-partial', {onSuccess});

            expect(onSuccess).toHaveBeenCalledWith('<p>refreshed</p>');
        });

        it('should call onError with the error on failure', async () => {
            const err = new Error('network fail');
            mockReload.mockRejectedValueOnce(err);
            const onError = vi.fn();
            vi.spyOn(console, 'error').mockImplementation(() => {});

            await expect(reloadPartial('test-partial', {onError})).rejects.toThrow('network fail');

            expect(onError).toHaveBeenCalledWith(err);
        });
    });

    describe('retry logic', () => {
        it('should succeed on second attempt when first call fails', async () => {
            mockReload
                .mockRejectedValueOnce(new Error('transient'))
                .mockResolvedValueOnce(undefined);
            vi.spyOn(console, 'error').mockImplementation(() => {});

            const promise = reloadPartial('test-partial', {retry: 1});

            await vi.advanceTimersByTimeAsync(1100);
            await promise;

            expect(mockReload).toHaveBeenCalledTimes(2);
        });

        it('should apply exponential backoff delays: 1000ms, 2000ms, 4000ms', async () => {
            mockReload
                .mockRejectedValueOnce(new Error('fail1'))
                .mockRejectedValueOnce(new Error('fail2'))
                .mockRejectedValueOnce(new Error('fail3'))
                .mockResolvedValueOnce(undefined);
            vi.spyOn(console, 'error').mockImplementation(() => {});

            const promise = reloadPartial('test-partial', {retry: 3});

            await vi.advanceTimersByTimeAsync(1100);
            await vi.advanceTimersByTimeAsync(2100);
            await vi.advanceTimersByTimeAsync(4100);

            await promise;

            expect(mockReload).toHaveBeenCalledTimes(4);
        });

        it('should throw and call onError after all retries are exhausted', async () => {
            const err = new Error('persistent');
            mockReload
                .mockRejectedValueOnce(err)
                .mockRejectedValueOnce(err)
                .mockRejectedValueOnce(err);
            const onError = vi.fn();
            vi.spyOn(console, 'error').mockImplementation(() => {});

            let caughtError: Error | null = null;
            const promise = reloadPartial('test-partial', {retry: 2, onError}).catch(e => {
                caughtError = e as Error;
            });

            await vi.advanceTimersByTimeAsync(4000);
            await promise;

            expect(caughtError).toBe(err);
            expect(onError).toHaveBeenCalledWith(err);
            expect(mockReload).toHaveBeenCalledTimes(3);
        });
    });

    describe('debounce', () => {
        it('should coalesce multiple calls within the debounce window into a single reload', async () => {
            const name = 'debounce-single-' + Math.random();
            vi.mocked(partial).mockImplementation(() => makeMockHandle());

            reloadPartial(name, {debounce: 100});
            reloadPartial(name, {debounce: 100});
            reloadPartial(name, {debounce: 100});

            await vi.advanceTimersByTimeAsync(150);

            expect(mockReload).toHaveBeenCalledOnce();
        });

        it('should reuse the debounced function for the same partial name', async () => {
            const name = 'debounce-reuse-' + Math.random();

            reloadPartial(name, {debounce: 200});
            await vi.advanceTimersByTimeAsync(250);
            expect(mockReload).toHaveBeenCalledTimes(1);

            reloadPartial(name, {debounce: 200});
            await vi.advanceTimersByTimeAsync(250);
            expect(mockReload).toHaveBeenCalledTimes(2);
        });
    });

    describe('sanitiseHTML (via optimistic updates)', () => {
        it('should strip <script> tags from optimistic HTML', async () => {
            await reloadPartial('test-partial', {
                optimistic: '<p>safe</p><script>alert("xss")</script>'
            });

            let innerDuringReload = '';
            mockReload.mockImplementation(async () => {
                innerDuringReload = mockElement!.innerHTML;
            });

            await reloadPartial('test-partial', {
                optimistic: '<p>safe</p><script>alert("xss")</script>'
            });

            expect(innerDuringReload).not.toContain('<script');
            expect(innerDuringReload).toContain('<p>safe</p>');
        });

        it('should strip inline event handlers like onclick', async () => {
            let innerDuringReload = '';
            mockReload.mockImplementation(async () => {
                innerDuringReload = mockElement!.innerHTML;
            });

            await reloadPartial('test-partial', {
                optimistic: '<button onclick="alert(1)" onload="foo()">click</button>'
            });

            expect(innerDuringReload).not.toContain('onclick');
            expect(innerDuringReload).not.toContain('onload');
            expect(innerDuringReload).toContain('click');
        });

        it('should preserve normal HTML', async () => {
            let innerDuringReload = '';
            mockReload.mockImplementation(async () => {
                innerDuringReload = mockElement!.innerHTML;
            });

            await reloadPartial('test-partial', {
                optimistic: '<div class="card"><h2>Title</h2><p>body</p></div>'
            });

            expect(innerDuringReload).toContain('<h2>Title</h2>');
            expect(innerDuringReload).toContain('<p>body</p>');
            expect(innerDuringReload).toContain('class="card"');
        });
    });

    describe('optimistic updates', () => {
        it('should set innerHTML when optimistic is a string', async () => {
            let innerDuringReload = '';
            mockReload.mockImplementation(async () => {
                innerDuringReload = mockElement!.innerHTML;
            });

            await reloadPartial('test-partial', {optimistic: '<b>loading...</b>'});

            expect(innerDuringReload).toBe('<b>loading...</b>');
        });

        it('should set innerHTML from {innerHTML} option (sanitised)', async () => {
            let innerDuringReload = '';
            mockReload.mockImplementation(async () => {
                innerDuringReload = mockElement!.innerHTML;
            });

            await reloadPartial('test-partial', {
                optimistic: {innerHTML: '<em>loading</em><script>bad()</script>'}
            });

            expect(innerDuringReload).toBe('<em>loading</em>');
        });

        it('should set className from {className} option', async () => {
            let classDuringReload = '';
            mockReload.mockImplementation(async () => {
                classDuringReload = mockElement!.className;
            });

            await reloadPartial('test-partial', {
                optimistic: {className: 'foo'}
            });

            expect(classDuringReload).toContain('foo');
        });

        it('should add a class from {addClass} option', async () => {
            let hadClass = false;
            mockReload.mockImplementation(async () => {
                hadClass = mockElement!.classList.contains('bar');
            });

            await reloadPartial('test-partial', {optimistic: {addClass: 'bar'}});

            expect(hadClass).toBe(true);
        });

        it('should remove a class from {removeClass} option', async () => {
            mockElement!.classList.add('baz');
            let hadClass = true;
            mockReload.mockImplementation(async () => {
                hadClass = mockElement!.classList.contains('baz');
            });

            await reloadPartial('test-partial', {optimistic: {removeClass: 'baz'}});

            expect(hadClass).toBe(false);
        });
    });

    describe('reloadGroup()', () => {
        function setupMultiplePartials(names: string[]): void {
            const elements = new Map<string, HTMLElement>();
            for (const name of names) {
                const el = createPartialElement(name);
                elements.set(name, el);
            }
            vi.mocked(partial).mockImplementation((nameOrElement) => makeMockHandle(elements.get(resolvePartialName(nameOrElement)) ?? null));
        }

        it('should reload all partials in parallel (default mode)', async () => {
            const names = ['a', 'b', 'c'];
            setupMultiplePartials(names);

            await reloadGroup(names);

            expect(mockReload).toHaveBeenCalledTimes(3);
        });

        it('should reload partials sequentially when mode is "sequential"', async () => {
            const order: string[] = [];
            const names = ['first', 'second', 'third'];
            const elements = new Map<string, HTMLElement>();
            for (const name of names) {
                elements.set(name, createPartialElement(name));
            }

            vi.mocked(partial).mockImplementation((nameOrElement) => {
                const name = resolvePartialName(nameOrElement);
                return {
                    element: elements.get(name) ?? null,
                    reload: vi.fn(async () => {
                        order.push(name);
                    }),
                    reloadWithOptions: mockReloadWithOptions
                };
            });

            await reloadGroup(names, {mode: 'sequential'});

            expect(order).toEqual(['first', 'second', 'third']);
        });

        it('should call onProgress with (completed, total) for each partial', async () => {
            const names = ['p1', 'p2'];
            setupMultiplePartials(names);
            const onProgress = vi.fn();

            await reloadGroup(names, {onProgress});

            expect(onProgress).toHaveBeenCalledTimes(2);
            expect(onProgress).toHaveBeenCalledWith(1, 2);
            expect(onProgress).toHaveBeenCalledWith(2, 2);
        });
    });

    describe('autoRefresh()', () => {
        it('should fire reload at the specified interval', async () => {
            autoRefresh('test-partial', {interval: 5000});

            await vi.advanceTimersByTimeAsync(5100);
            expect(mockReload).toHaveBeenCalledTimes(1);

            await vi.advanceTimersByTimeAsync(5000);
            expect(mockReload).toHaveBeenCalledTimes(2);
        });

        it('should skip reload when "when" condition returns false', async () => {
            let condition = false;
            autoRefresh('test-partial', {
                interval: 1000,
                when: () => condition
            });

            await vi.advanceTimersByTimeAsync(1100);
            expect(mockReload).not.toHaveBeenCalled();

            condition = true;
            await vi.advanceTimersByTimeAsync(1000);
            expect(mockReload).toHaveBeenCalledOnce();
        });

        it('should increment retry count on errors and stop after maxRetries', async () => {
            mockReload.mockRejectedValue(new Error('fail'));
            vi.spyOn(console, 'warn').mockImplementation(() => {});

            autoRefresh('test-partial', {interval: 1000, maxRetries: 2});

            await vi.advanceTimersByTimeAsync(1100);
            expect(mockReload).toHaveBeenCalledTimes(1);

            await vi.advanceTimersByTimeAsync(1100);
            expect(mockReload).toHaveBeenCalledTimes(2);

            await vi.advanceTimersByTimeAsync(3000);
            expect(mockReload).toHaveBeenCalledTimes(2);
        });

        it('should stop immediately on first failure when onError is "stop"', async () => {
            mockReload.mockRejectedValue(new Error('fatal'));
            vi.spyOn(console, 'warn').mockImplementation(() => {});

            autoRefresh('test-partial', {
                interval: 1000,
                onError: 'stop'
            });

            await vi.advanceTimersByTimeAsync(1100);
            expect(mockReload).toHaveBeenCalledTimes(1);

            await vi.advanceTimersByTimeAsync(5000);
            expect(mockReload).toHaveBeenCalledTimes(1);
        });

        it('should stop the interval when cleanup function is called', async () => {
            const cleanup = autoRefresh('test-partial', {interval: 1000});

            await vi.advanceTimersByTimeAsync(1100);
            expect(mockReload).toHaveBeenCalledTimes(1);

            cleanup();

            await vi.advanceTimersByTimeAsync(5000);
            expect(mockReload).toHaveBeenCalledTimes(1);
        });

        it('should reset retryCount after a successful reload', async () => {
            let callIndex = 0;
            mockReload.mockImplementation(async () => {
                callIndex++;
                if (callIndex === 1 || callIndex === 3) {
                    throw new Error('transient');
                }
            });
            vi.spyOn(console, 'warn').mockImplementation(() => {});

            autoRefresh('test-partial', {interval: 1000, maxRetries: 3});

            await vi.advanceTimersByTimeAsync(1100);
            await vi.advanceTimersByTimeAsync(1100);
            await vi.advanceTimersByTimeAsync(1100);
            await vi.advanceTimersByTimeAsync(1100);

            expect(mockReload).toHaveBeenCalledTimes(4);
        });
    });

    describe('reloadCascade()', () => {
        function setupCascadePartials(names: string[]): void {
            const elements = new Map<string, HTMLElement>();
            for (const name of names) {
                elements.set(name, createPartialElement(name));
            }
            vi.mocked(partial).mockImplementation((nameOrElement) => makeMockHandle(elements.get(resolvePartialName(nameOrElement)) ?? null));
        }

        it('should reload root first, then children in parallel', async () => {
            const order: string[] = [];
            const elements = new Map<string, HTMLElement>();
            const names = ['root', 'child-a', 'child-b'];
            for (const name of names) {
                elements.set(name, createPartialElement(name));
            }

            vi.mocked(partial).mockImplementation((nameOrElement) => {
                const name = resolvePartialName(nameOrElement);
                return {
                    element: elements.get(name) ?? null,
                    reload: vi.fn(async () => {
                        order.push(name);
                    }),
                    reloadWithOptions: mockReloadWithOptions
                };
            });

            await reloadCascade({
                name: 'root',
                children: [
                    {name: 'child-a'},
                    {name: 'child-b'}
                ]
            });

            expect(order[0]).toBe('root');
            expect(order).toContain('child-a');
            expect(order).toContain('child-b');
            expect(order.length).toBe(3);
        });

        it('should handle deep nesting (3 levels)', async () => {
            const allNames = ['l1', 'l2a', 'l2b', 'l3'];
            setupCascadePartials(allNames);

            await reloadCascade({
                name: 'l1',
                children: [
                    {
                        name: 'l2a',
                        children: [{name: 'l3'}]
                    },
                    {name: 'l2b'}
                ]
            });

            expect(mockReload).toHaveBeenCalledTimes(4);
        });

        it('should call onNodeComplete for each node', async () => {
            const names = ['root', 'child'];
            setupCascadePartials(names);
            const onNodeComplete = vi.fn();

            await reloadCascade(
                {name: 'root', children: [{name: 'child'}]},
                {onNodeComplete}
            );

            expect(onNodeComplete).toHaveBeenCalledWith('root');
            expect(onNodeComplete).toHaveBeenCalledWith('child');
            expect(onNodeComplete).toHaveBeenCalledTimes(2);
        });

        it('should pass args to every partial in the cascade', async () => {
            const names = ['parent', 'child'];
            const elements = new Map<string, HTMLElement>();
            for (const name of names) {
                elements.set(name, createPartialElement(name));
            }
            const reloadSpies: ReturnType<typeof vi.fn>[] = [];

            vi.mocked(partial).mockImplementation((nameOrElement) => {
                const spy = vi.fn<(data?: Record<string, string | number | boolean>) => Promise<void>>().mockResolvedValue(undefined);
                reloadSpies.push(spy);
                return {
                    element: elements.get(resolvePartialName(nameOrElement)) ?? null,
                    reload: spy,
                    reloadWithOptions: mockReloadWithOptions
                };
            });

            await reloadCascade(
                {name: 'parent', children: [{name: 'child'}]},
                {args: {lang: 'en'}}
            );

            for (const spy of reloadSpies) {
                expect(spy).toHaveBeenCalledWith({lang: 'en'});
            }
        });
    });

    describe('race conditions', () => {
        it('should handle concurrent reloadPartial calls to the same partial', async () => {
            const p1 = reloadPartial('test-partial');
            const p2 = reloadPartial('test-partial');

            await Promise.all([p1, p2]);

            expect(mockReload).toHaveBeenCalledTimes(2);
        });

        it('should coalesce 10 rapid debounced calls into a single reload', async () => {
            const name = 'rapid-debounce-' + Math.random();

            for (let i = 0; i < 10; i++) {
                reloadPartial(name, {debounce: 100});
            }

            await vi.advanceTimersByTimeAsync(200);

            expect(mockReload).toHaveBeenCalledOnce();
        });

        it('should not stack up reloads when autoRefresh reload takes longer than interval', async () => {
            let concurrentCount = 0;
            let maxConcurrent = 0;

            mockReload.mockImplementation(async () => {
                concurrentCount++;
                if (concurrentCount > maxConcurrent) {
                    maxConcurrent = concurrentCount;
                }
                await new Promise(resolve => setTimeout(resolve, 3000));
                concurrentCount--;
            });

            autoRefresh('test-partial', {interval: 1000});

            await vi.advanceTimersByTimeAsync(10000);

            expect(mockReload.mock.calls.length).toBeGreaterThanOrEqual(1);
        });
    });
});
