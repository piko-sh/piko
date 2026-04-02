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
import {
    whenVisible,
    withAbortSignal,
    timeout,
    poll,
    watchMutations,
    whenIdle,
    nextFrame,
    waitFrames,
    deferred,
    once
} from '@/pk/advanced';

describe('advanced (PK Advanced Helpers)', () => {
    let container: HTMLDivElement;

    beforeEach(() => {
        container = document.createElement('div');
        document.body.appendChild(container);
        vi.spyOn(console, 'warn').mockImplementation(() => {});
        vi.spyOn(console, 'error').mockImplementation(() => {});
    });

    afterEach(() => {
        container.remove();
        vi.restoreAllMocks();
    });

    describe('whenVisible', () => {
        let mockObserverCallback: IntersectionObserverCallback;
        let mockObserve: ReturnType<typeof vi.fn>;
        let mockDisconnect: ReturnType<typeof vi.fn>;

        beforeEach(() => {
            mockObserve = vi.fn();
            mockDisconnect = vi.fn();

            vi.stubGlobal('IntersectionObserver', class {
                constructor(callback: IntersectionObserverCallback) {
                    mockObserverCallback = callback;
                }
                observe = mockObserve;
                disconnect = mockDisconnect;
            });
        });

        afterEach(() => {
            vi.unstubAllGlobals();
        });

        it('should create IntersectionObserver for element', () => {
            container.innerHTML = '<div id="target"></div>';

            whenVisible('#target', () => {});

            expect(mockObserve).toHaveBeenCalled();
        });

        it('should call callback when element becomes visible', () => {
            container.innerHTML = '<div id="target"></div>';
            const callback = vi.fn();

            whenVisible('#target', callback);

            mockObserverCallback(
                [{ isIntersecting: true } as IntersectionObserverEntry],
                {} as IntersectionObserver
            );

            expect(callback).toHaveBeenCalledTimes(1);
        });

        it('should disconnect after first visibility when once is true (default)', () => {
            container.innerHTML = '<div id="target"></div>';

            whenVisible('#target', () => {});

            mockObserverCallback(
                [{ isIntersecting: true } as IntersectionObserverEntry],
                {} as IntersectionObserver
            );

            expect(mockDisconnect).toHaveBeenCalled();
        });

        it('should not disconnect when once is false', () => {
            container.innerHTML = '<div id="target"></div>';

            whenVisible('#target', () => {}, { once: false });

            mockObserverCallback(
                [{ isIntersecting: true } as IntersectionObserverEntry],
                {} as IntersectionObserver
            );

            expect(mockDisconnect).not.toHaveBeenCalled();
        });

        it('should call callback for exit when once is false', () => {
            container.innerHTML = '<div id="target"></div>';
            const callback = vi.fn();

            whenVisible('#target', callback, { once: false });

            mockObserverCallback(
                [{ isIntersecting: true } as IntersectionObserverEntry],
                {} as IntersectionObserver
            );

            mockObserverCallback(
                [{ isIntersecting: false } as IntersectionObserverEntry],
                {} as IntersectionObserver
            );

            expect(callback).toHaveBeenCalledTimes(2);
        });

        it('should return stop function that disconnects', () => {
            container.innerHTML = '<div id="target"></div>';

            const stop = whenVisible('#target', () => {});
            stop();

            expect(mockDisconnect).toHaveBeenCalled();
        });

        it('should warn when target not found', () => {
            whenVisible('#non-existent', () => {});

            expect(console.warn).toHaveBeenCalled();
        });

        it('should work with p-ref', () => {
            container.innerHTML = '<div p-ref="my-element"></div>';

            whenVisible('my-element', () => {});

            expect(mockObserve).toHaveBeenCalled();
        });
    });

    describe('withAbortSignal', () => {
        it('should create abortable operation', () => {
            const result = withAbortSignal((signal) => {
                return new Promise((resolve, reject) => {
                    signal.addEventListener('abort', () => reject(new Error('Aborted')));
                    setTimeout(() => resolve('done'), 100);
                });
            });

            expect(result.promise).toBeInstanceOf(Promise);
            expect(typeof result.abort).toBe('function');
            expect(result.signal).toBeInstanceOf(AbortSignal);
        });

        it('should abort the operation', async () => {
            const { promise, abort } = withAbortSignal((signal) => {
                return new Promise((resolve, reject) => {
                    signal.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')));
                    setTimeout(() => resolve('done'), 1000);
                });
            });

            abort();

            await expect(promise).rejects.toThrow('Aborted');
        });

        it('should complete if not aborted', async () => {
            const { promise } = withAbortSignal(() => Promise.resolve('success'));

            const result = await promise;

            expect(result).toBe('success');
        });
    });

    describe('timeout', () => {
        beforeEach(() => {
            vi.useFakeTimers();
        });

        afterEach(() => {
            vi.useRealTimers();
        });

        it('should resolve after specified time', async () => {
            const { promise } = timeout(1000);

            const spy = vi.fn();
            promise.then(spy);

            expect(spy).not.toHaveBeenCalled();

            vi.advanceTimersByTime(1000);
            await promise;

            expect(spy).toHaveBeenCalled();
        });

        it('should reject when cancelled', async () => {
            const { promise, cancel } = timeout(1000);

            cancel();

            await expect(promise).rejects.toThrow('Timeout cancelled');
        });

        it('should not reject if already resolved', async () => {
            const { promise, cancel } = timeout(100);

            vi.advanceTimersByTime(100);
            await promise;

            expect(() => cancel()).not.toThrow();
        });
    });

    describe('poll', () => {
        beforeEach(() => {
            vi.useFakeTimers();
        });

        afterEach(() => {
            vi.useRealTimers();
        });

        it('should poll at specified interval', async () => {
            const mockPollFunction = vi.fn().mockResolvedValue('result');

            poll(mockPollFunction, { interval: 1000 });

            await vi.advanceTimersByTimeAsync(0);
            expect(mockPollFunction).toHaveBeenCalledTimes(1);

            await vi.advanceTimersByTimeAsync(1000);
            expect(mockPollFunction).toHaveBeenCalledTimes(2);

            await vi.advanceTimersByTimeAsync(1000);
            expect(mockPollFunction).toHaveBeenCalledTimes(3);
        });

        it('should call onPoll with result and attempt', async () => {
            const mockPollFunction = vi.fn().mockResolvedValue('data');
            const onPoll = vi.fn();

            poll(mockPollFunction, { interval: 1000, onPoll });

            await vi.advanceTimersByTimeAsync(0);

            expect(onPoll).toHaveBeenCalledWith('data', 1);
        });

        it('should stop when until condition returns true', async () => {
            const mockPollFunction = vi.fn().mockResolvedValue('done');
            const until = vi.fn().mockReturnValueOnce(false).mockReturnValueOnce(true);
            const onStop = vi.fn();

            poll(mockPollFunction, { interval: 1000, until, onStop });

            await vi.advanceTimersByTimeAsync(0);
            await vi.advanceTimersByTimeAsync(1000);

            expect(onStop).toHaveBeenCalledWith('condition');
        });

        it('should stop after maxAttempts', async () => {
            const mockPollFunction = vi.fn().mockResolvedValue('result');
            const onStop = vi.fn();

            poll(mockPollFunction, { interval: 1000, maxAttempts: 3, onStop });

            await vi.advanceTimersByTimeAsync(0);
            await vi.advanceTimersByTimeAsync(1000);
            await vi.advanceTimersByTimeAsync(1000);

            expect(onStop).toHaveBeenCalledWith('maxAttempts');
            expect(mockPollFunction).toHaveBeenCalledTimes(3);

            await vi.advanceTimersByTimeAsync(1000);
            expect(mockPollFunction).toHaveBeenCalledTimes(3);
        });

        it('should stop when stop function is called', async () => {
            const mockPollFunction = vi.fn().mockResolvedValue('result');
            const onStop = vi.fn();

            const stop = poll(mockPollFunction, { interval: 1000, onStop });

            await vi.advanceTimersByTimeAsync(0);
            stop();

            expect(onStop).toHaveBeenCalledWith('manual');

            await vi.advanceTimersByTimeAsync(1000);
            expect(mockPollFunction).toHaveBeenCalledTimes(1);
        });

        it('should continue polling on error', async () => {
            const mockPollFunction = vi.fn()
                .mockRejectedValueOnce(new Error('fail'))
                .mockResolvedValue('success');

            poll(mockPollFunction, { interval: 1000 });

            await vi.advanceTimersByTimeAsync(0);
            expect(console.error).toHaveBeenCalled();

            await vi.advanceTimersByTimeAsync(1000);
            expect(mockPollFunction).toHaveBeenCalledTimes(2);
        });
    });

    describe('watchMutations', () => {
        let mockObserve: ReturnType<typeof vi.fn>;
        let mockDisconnect: ReturnType<typeof vi.fn>;

        beforeEach(() => {
            mockObserve = vi.fn();
            mockDisconnect = vi.fn();

            vi.stubGlobal('MutationObserver', class {
                observe = mockObserve;
                disconnect = mockDisconnect;
            });
        });

        afterEach(() => {
            vi.unstubAllGlobals();
        });

        it('should create MutationObserver for element', () => {
            container.innerHTML = '<div id="target"></div>';

            watchMutations('#target', () => {});

            expect(mockObserve).toHaveBeenCalled();
        });

        it('should use default options', () => {
            container.innerHTML = '<div id="target"></div>';

            watchMutations('#target', () => {});

            expect(mockObserve).toHaveBeenCalledWith(
                expect.any(Element),
                expect.objectContaining({
                    childList: true,
                    attributes: false,
                    characterData: false,
                    subtree: false
                })
            );
        });

        it('should use custom options', () => {
            container.innerHTML = '<div id="target"></div>';

            watchMutations('#target', () => {}, {
                childList: false,
                attributes: true,
                subtree: true,
                attributeFilter: ['class', 'id']
            });

            expect(mockObserve).toHaveBeenCalledWith(
                expect.any(Element),
                expect.objectContaining({
                    childList: false,
                    attributes: true,
                    subtree: true,
                    attributeFilter: ['class', 'id']
                })
            );
        });

        it('should return stop function that disconnects', () => {
            container.innerHTML = '<div id="target"></div>';

            const stop = watchMutations('#target', () => {});
            stop();

            expect(mockDisconnect).toHaveBeenCalled();
        });

        it('should warn when target not found', () => {
            watchMutations('#non-existent', () => {});

            expect(console.warn).toHaveBeenCalled();
        });
    });

    describe('whenIdle', () => {
        beforeEach(() => {
            vi.useFakeTimers();
        });

        afterEach(() => {
            vi.useRealTimers();
        });

        it('should use requestIdleCallback when available', () => {
            const mockRequestIdleCallback = vi.fn().mockReturnValue(1);
            const mockCancelIdleCallback = vi.fn();

            vi.stubGlobal('requestIdleCallback', mockRequestIdleCallback);
            vi.stubGlobal('cancelIdleCallback', mockCancelIdleCallback);

            const mockIdleCallback = vi.fn();
            whenIdle(mockIdleCallback);

            expect(mockRequestIdleCallback).toHaveBeenCalledWith(mockIdleCallback, undefined);

            vi.unstubAllGlobals();
        });

        it('should return cancel function', () => {
            const mockRequestIdleCallback = vi.fn().mockReturnValue(123);
            const mockCancelIdleCallback = vi.fn();

            vi.stubGlobal('requestIdleCallback', mockRequestIdleCallback);
            vi.stubGlobal('cancelIdleCallback', mockCancelIdleCallback);

            const cancel = whenIdle(() => {});
            cancel();

            expect(mockCancelIdleCallback).toHaveBeenCalledWith(123);

            vi.unstubAllGlobals();
        });

        it('should fallback to setTimeout when requestIdleCallback not available', () => {
            const original = window.requestIdleCallback;
            delete (window as unknown as Record<string, unknown>).requestIdleCallback;

            const mockIdleCallback = vi.fn();
            whenIdle(mockIdleCallback);

            vi.advanceTimersByTime(50);

            expect(mockIdleCallback).toHaveBeenCalled();

            (window as unknown as Record<string, unknown>).requestIdleCallback = original;
        });
    });

    describe('nextFrame', () => {
        it('should return a promise', () => {
            const result = nextFrame();
            expect(result).toBeInstanceOf(Promise);
        });

        it('should resolve with timestamp from requestAnimationFrame', async () => {
            const mockRaf = vi.fn((cb: FrameRequestCallback) => {
                cb(1000);
                return 1;
            });
            vi.stubGlobal('requestAnimationFrame', mockRaf);

            const timestamp = await nextFrame();

            expect(timestamp).toBe(1000);

            vi.unstubAllGlobals();
        });
    });

    describe('waitFrames', () => {
        it('should wait for specified number of frames', async () => {
            let frameCount = 0;
            const mockRaf = vi.fn((cb: FrameRequestCallback) => {
                frameCount++;
                cb(frameCount * 16);
                return frameCount;
            });
            vi.stubGlobal('requestAnimationFrame', mockRaf);

            await waitFrames(3);

            expect(mockRaf).toHaveBeenCalledTimes(3);

            vi.unstubAllGlobals();
        });
    });

    describe('deferred', () => {
        it('should create deferred promise', () => {
            const { promise, resolve, reject } = deferred<string>();

            expect(promise).toBeInstanceOf(Promise);
            expect(typeof resolve).toBe('function');
            expect(typeof reject).toBe('function');
        });

        it('should resolve when resolve is called', async () => {
            const { promise, resolve } = deferred<string>();

            resolve('success');

            const result = await promise;

            expect(result).toBe('success');
        });

        it('should reject when reject is called', async () => {
            const { promise, reject } = deferred<string>();

            reject(new Error('failed'));

            await expect(promise).rejects.toThrow('failed');
        });
    });

    describe('once', () => {
        it('should only execute function once', () => {
            const mockFunction = vi.fn().mockReturnValue('result');
            const onceFn = once(mockFunction);

            onceFn();
            onceFn();
            onceFn();

            expect(mockFunction).toHaveBeenCalledTimes(1);
        });

        it('should return cached result on subsequent calls', () => {
            let counter = 0;
            const incrementer = () => ++counter;
            const onceFn = once(incrementer);

            const result1 = onceFn();
            const result2 = onceFn();
            const result3 = onceFn();

            expect(result1).toBe(1);
            expect(result2).toBe(1);
            expect(result3).toBe(1);
        });

        it('should work with async functions', async () => {
            const mockAsyncFunction = vi.fn().mockResolvedValue('async result');
            const onceFn = once(mockAsyncFunction);

            const result1 = await onceFn();
            const result2 = await onceFn();

            expect(mockAsyncFunction).toHaveBeenCalledTimes(1);
            expect(result1).toBe('async result');
            expect(result2).toBe('async result');
        });

        it('should cache undefined/null results', () => {
            let callCount = 0;
            const returnUndefined = () => {
                callCount++;
                return undefined;
            };
            const onceFn = once(returnUndefined);

            onceFn();
            onceFn();
            onceFn();

            expect(callCount).toBe(1);
        });
    });
});
