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
import { loading, withRetry, withLoading, debounceAsync, throttleAsync } from '@/pk/ui';

describe('ui (PK UI Helpers)', () => {
    let container: HTMLDivElement;

    beforeEach(() => {
        container = document.createElement('div');
        document.body.appendChild(container);
        vi.useFakeTimers();
        vi.spyOn(console, 'warn').mockImplementation(() => {});
    });

    afterEach(() => {
        container.remove();
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    describe('loading', () => {
        it('should add loading class during operation', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';
            const button = container.querySelector('#btn') as HTMLButtonElement;

            const promise = new Promise<string>(resolve => {
                setTimeout(() => resolve('done'), 100);
            });

            const loadingPromise = loading('#btn', promise);

            expect(button.classList.contains('loading')).toBe(true);

            vi.advanceTimersByTime(100);
            await loadingPromise;

            expect(button.classList.contains('loading')).toBe(false);
        });

        it('should use custom className', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';
            const button = container.querySelector('#btn') as HTMLButtonElement;

            const promise = Promise.resolve('done');
            await loading('#btn', promise, { className: 'is-loading' });

            expect(button.classList.contains('loading')).toBe(false);
        });

        it('should replace text during loading', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';
            const button = container.querySelector('#btn') as HTMLButtonElement;

            const promise = new Promise<string>(resolve => {
                setTimeout(() => resolve('done'), 100);
            });

            const loadingPromise = loading('#btn', promise, { text: 'Loading...' });

            expect(button.innerText).toBe('Loading...');

            vi.advanceTimersByTime(100);
            await loadingPromise;

            expect(button.classList.contains('loading')).toBe(false);
        });

        it('should disable button during loading', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';
            const button = container.querySelector('#btn') as HTMLButtonElement;

            const promise = new Promise<string>(resolve => {
                setTimeout(() => resolve('done'), 100);
            });

            const loadingPromise = loading('#btn', promise);

            expect(button.disabled).toBe(true);

            vi.advanceTimersByTime(100);
            await loadingPromise;

            expect(button.disabled).toBe(false);
        });

        it('should respect disabled: false option', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';
            const button = container.querySelector('#btn') as HTMLButtonElement;

            const promise = Promise.resolve('done');
            await loading('#btn', promise, { disabled: false });

            expect(button.disabled).toBe(false);
        });

        it('should respect minDuration option', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';
            const button = container.querySelector('#btn') as HTMLButtonElement;

            const promise = new Promise<string>(resolve => {
                setTimeout(() => resolve('done'), 10);
            });

            const loadingPromise = loading('#btn', promise, { minDuration: 500 });

            vi.advanceTimersByTime(10);

            expect(button.classList.contains('loading')).toBe(true);

            vi.advanceTimersByTime(490);
            await loadingPromise;

            expect(button.classList.contains('loading')).toBe(false);
        });

        it('should call onStart and onEnd callbacks', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';

            const onStart = vi.fn();
            const onEnd = vi.fn();

            const promise = Promise.resolve('done');
            await loading('#btn', promise, { onStart, onEnd });

            expect(onStart).toHaveBeenCalledTimes(1);
            expect(onEnd).toHaveBeenCalledTimes(1);
        });

        it('should call onEnd even if promise rejects', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';

            const onEnd = vi.fn();

            const promise = Promise.reject(new Error('fail'));
            await expect(loading('#btn', promise, { onEnd })).rejects.toThrow('fail');

            expect(onEnd).toHaveBeenCalledTimes(1);
        });

        it('should restore original disabled state', async () => {
            container.innerHTML = '<button id="btn" disabled>Submit</button>';
            const button = container.querySelector('#btn') as HTMLButtonElement;

            const promise = Promise.resolve('done');
            await loading('#btn', promise);

            expect(button.disabled).toBe(true);
        });

        it('should work with p-ref selector', async () => {
            container.innerHTML = '<button p-ref="submit-btn">Submit</button>';

            const promise = Promise.resolve('done');
            await loading('submit-btn', promise);

            expect(console.warn).not.toHaveBeenCalled();
        });

        it('should warn and return promise if target not found', async () => {
            const promise = Promise.resolve('done');
            const result = await loading('#non-existent', promise);

            expect(console.warn).toHaveBeenCalled();
            expect(result).toBe('done');
        });

        it('should work with HTMLElement directly', async () => {
            container.innerHTML = '<button>Submit</button>';
            const button = container.querySelector('button')!;

            const promise = Promise.resolve('done');
            await loading(button, promise);

            expect(button.classList.contains('loading')).toBe(false);
        });
    });

    describe('withRetry', () => {
        it('should return result on first successful attempt', async () => {
            const mockOperation = vi.fn().mockResolvedValue('success');

            const result = await withRetry(mockOperation);

            expect(result).toBe('success');
            expect(mockOperation).toHaveBeenCalledTimes(1);
        });

        it('should retry on failure', async () => {
            const mockOperation = vi.fn()
                .mockRejectedValueOnce(new Error('fail 1'))
                .mockResolvedValueOnce('success');

            const resultPromise = withRetry(mockOperation, { delay: 100 });

            await vi.advanceTimersByTimeAsync(0);

            await vi.advanceTimersByTimeAsync(100);

            const result = await resultPromise;

            expect(result).toBe('success');
            expect(mockOperation).toHaveBeenCalledTimes(2);
        });

        it('should throw after max attempts', async () => {
            const mockOperation = vi.fn().mockRejectedValue(new Error('always fails'));

            const resultPromise = withRetry(mockOperation, { attempts: 3, delay: 100 });
            resultPromise.catch(() => {});

            await vi.advanceTimersByTimeAsync(0);
            await vi.advanceTimersByTimeAsync(100);
            await vi.advanceTimersByTimeAsync(200);

            await expect(resultPromise).rejects.toThrow('always fails');
            expect(mockOperation).toHaveBeenCalledTimes(3);
        });

        it('should use exponential backoff by default', async () => {
            const mockOperation = vi.fn()
                .mockRejectedValueOnce(new Error('fail'))
                .mockRejectedValueOnce(new Error('fail'))
                .mockResolvedValueOnce('success');

            const resultPromise = withRetry(mockOperation, { delay: 100 });

            await vi.advanceTimersByTimeAsync(0);
            await vi.advanceTimersByTimeAsync(100);
            await vi.advanceTimersByTimeAsync(200);

            await resultPromise;

            expect(mockOperation).toHaveBeenCalledTimes(3);
        });

        it('should use linear backoff when specified', async () => {
            const mockOperation = vi.fn()
                .mockRejectedValueOnce(new Error('fail'))
                .mockRejectedValueOnce(new Error('fail'))
                .mockResolvedValueOnce('success');

            const resultPromise = withRetry(mockOperation, { backoff: 'linear', delay: 100 });

            await vi.advanceTimersByTimeAsync(0);
            await vi.advanceTimersByTimeAsync(100);
            await vi.advanceTimersByTimeAsync(200);

            await resultPromise;

            expect(mockOperation).toHaveBeenCalledTimes(3);
        });

        it('should respect maxDelay option', async () => {
            const mockOperation = vi.fn()
                .mockRejectedValueOnce(new Error('fail'))
                .mockRejectedValueOnce(new Error('fail'))
                .mockRejectedValueOnce(new Error('fail'))
                .mockResolvedValueOnce('success');

            const resultPromise = withRetry(mockOperation, {
                delay: 1000,
                maxDelay: 1500,
                attempts: 4
            });

            await vi.advanceTimersByTimeAsync(0);
            await vi.advanceTimersByTimeAsync(1000);
            await vi.advanceTimersByTimeAsync(1500);
            await vi.advanceTimersByTimeAsync(1500);

            await resultPromise;

            expect(mockOperation).toHaveBeenCalledTimes(4);
        });

        it('should call onRetry callback', async () => {
            const onRetry = vi.fn();
            const mockOperation = vi.fn()
                .mockRejectedValueOnce(new Error('fail 1'))
                .mockRejectedValueOnce(new Error('fail 2'))
                .mockResolvedValueOnce('success');

            const resultPromise = withRetry(mockOperation, { onRetry, delay: 100 });

            await vi.advanceTimersByTimeAsync(0);
            await vi.advanceTimersByTimeAsync(100);
            await vi.advanceTimersByTimeAsync(200);

            await resultPromise;

            expect(onRetry).toHaveBeenCalledTimes(2);
            expect(onRetry).toHaveBeenNthCalledWith(1, 1, expect.any(Error));
            expect(onRetry).toHaveBeenNthCalledWith(2, 2, expect.any(Error));
        });

        it('should respect shouldRetry predicate', async () => {
            const mockOperation = vi.fn()
                .mockRejectedValueOnce(new Error('retriable'))
                .mockRejectedValueOnce(new Error('fatal'));

            const resultPromise = withRetry(mockOperation, {
                delay: 100,
                shouldRetry: (error) => error.message !== 'fatal'
            });
            resultPromise.catch(() => {});

            await vi.advanceTimersByTimeAsync(0);
            await vi.advanceTimersByTimeAsync(100);

            await expect(resultPromise).rejects.toThrow('fatal');
            expect(mockOperation).toHaveBeenCalledTimes(2);
        });
    });

    describe('withLoading', () => {
        it('should wrap function and show loading', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';
            const button = container.querySelector('#btn') as HTMLButtonElement;

            const mockOperation = vi.fn().mockResolvedValue('result');
            const result = await withLoading('#btn', mockOperation);

            expect(result).toBe('result');
            expect(mockOperation).toHaveBeenCalledTimes(1);
            expect(button.classList.contains('loading')).toBe(false);
        });

        it('should pass options to loading', async () => {
            container.innerHTML = '<button id="btn">Submit</button>';

            const onStart = vi.fn();
            await withLoading('#btn', async () => 'done', { onStart });

            expect(onStart).toHaveBeenCalled();
        });
    });

    describe('debounceAsync', () => {
        it('should delay function execution', async () => {
            const mockAsyncCallback = vi.fn().mockResolvedValue('result');
            const debounced = debounceAsync(mockAsyncCallback, 100);

            const promise = debounced('arg');

            expect(mockAsyncCallback).not.toHaveBeenCalled();

            vi.advanceTimersByTime(100);

            const result = await promise;

            expect(mockAsyncCallback).toHaveBeenCalledWith('arg');
            expect(result).toBe('result');
        });

        it('should reset timer on subsequent calls', async () => {
            const mockAsyncCallback = vi.fn().mockResolvedValue('result');
            const debounced = debounceAsync(mockAsyncCallback, 100);

            debounced('first');
            vi.advanceTimersByTime(50);

            debounced('second');
            vi.advanceTimersByTime(50);

            expect(mockAsyncCallback).not.toHaveBeenCalled();

            vi.advanceTimersByTime(50);

            expect(mockAsyncCallback).toHaveBeenCalledWith('second');
            expect(mockAsyncCallback).toHaveBeenCalledTimes(1);
        });

        it('should have cancel method', () => {
            const mockAsyncCallback = vi.fn().mockResolvedValue('result');
            const debounced = debounceAsync(mockAsyncCallback, 100);

            debounced('arg');
            debounced.cancel();

            vi.advanceTimersByTime(100);

            expect(mockAsyncCallback).not.toHaveBeenCalled();
        });

        it('should handle rejection', async () => {
            const mockAsyncCallback = vi.fn().mockRejectedValue(new Error('fail'));
            const debounced = debounceAsync(mockAsyncCallback, 100);

            const promise = debounced();

            vi.advanceTimersByTime(100);

            await expect(promise).rejects.toThrow('fail');
        });
    });

    describe('throttleAsync', () => {
        it('should execute immediately on first call', async () => {
            const mockAsyncCallback = vi.fn().mockResolvedValue('result');
            const throttled = throttleAsync(mockAsyncCallback, 100);

            const result = await throttled('arg');

            expect(mockAsyncCallback).toHaveBeenCalledWith('arg');
            expect(result).toBe('result');
        });

        it('should ignore calls within throttle period', async () => {
            const mockAsyncCallback = vi.fn().mockResolvedValue('result');
            const throttled = throttleAsync(mockAsyncCallback, 100);

            await throttled('first');
            const secondResult = await throttled('second');
            const thirdResult = await throttled('third');

            expect(mockAsyncCallback).toHaveBeenCalledTimes(1);
            expect(mockAsyncCallback).toHaveBeenCalledWith('first');
            expect(secondResult).toBe('result');
            expect(thirdResult).toBe('result');
        });

        it('should allow execution after throttle period', async () => {
            const mockAsyncCallback = vi.fn().mockResolvedValue('result');
            const throttled = throttleAsync(mockAsyncCallback, 100);

            await throttled('first');

            vi.advanceTimersByTime(100);

            await throttled('second');

            expect(mockAsyncCallback).toHaveBeenCalledTimes(2);
            expect(mockAsyncCallback).toHaveBeenNthCalledWith(1, 'first');
            expect(mockAsyncCallback).toHaveBeenNthCalledWith(2, 'second');
        });

        it('should return cached result when within throttle period after first call completes', async () => {
            const mockAsyncCallback = vi.fn().mockImplementation(async () => {
                await new Promise(r => setTimeout(r, 50));
                return 'result';
            });
            const throttled = throttleAsync(mockAsyncCallback, 100);

            const firstPromise = throttled('first');
            vi.advanceTimersByTime(50);
            await firstPromise;

            const secondResult = await throttled('second');
            expect(secondResult).toBe('result');
            expect(mockAsyncCallback).toHaveBeenCalledTimes(1);
        });
    });
});
