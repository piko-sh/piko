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
import { debounce, throttle } from '@/pk/utils';

describe('utils (PK Utilities)', () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    describe('debounce()', () => {
        it('should delay function execution', () => {
            const mockCallback = vi.fn();
            const debouncedFn = debounce(mockCallback, 100);

            debouncedFn();

            expect(mockCallback).not.toHaveBeenCalled();

            vi.advanceTimersByTime(100);

            expect(mockCallback).toHaveBeenCalledTimes(1);
        });

        it('should reset timer on subsequent calls', () => {
            const mockCallback = vi.fn();
            const debouncedFn = debounce(mockCallback, 100);

            debouncedFn();
            vi.advanceTimersByTime(50);

            debouncedFn();
            vi.advanceTimersByTime(50);

            expect(mockCallback).not.toHaveBeenCalled();

            vi.advanceTimersByTime(50);

            expect(mockCallback).toHaveBeenCalledTimes(1);
        });

        it('should pass arguments to the debounced function', () => {
            const mockCallback = vi.fn();
            const debouncedFn = debounce(mockCallback, 100);

            debouncedFn('arg1', 'arg2', 123);

            vi.advanceTimersByTime(100);

            expect(mockCallback).toHaveBeenCalledWith('arg1', 'arg2', 123);
        });

        it('should use latest arguments when called multiple times', () => {
            const mockCallback = vi.fn();
            const debouncedFn = debounce(mockCallback, 100);

            debouncedFn('first');
            debouncedFn('second');
            debouncedFn('third');

            vi.advanceTimersByTime(100);

            expect(mockCallback).toHaveBeenCalledTimes(1);
            expect(mockCallback).toHaveBeenCalledWith('third');
        });

        it('should work with different delay values', () => {
            const mockCallback = vi.fn();
            const shortDebounce = debounce(mockCallback, 50);
            const longDebounce = debounce(mockCallback, 200);

            shortDebounce();
            vi.advanceTimersByTime(50);
            expect(mockCallback).toHaveBeenCalledTimes(1);

            mockCallback.mockClear();

            longDebounce();
            vi.advanceTimersByTime(100);
            expect(mockCallback).not.toHaveBeenCalled();

            vi.advanceTimersByTime(100);
            expect(mockCallback).toHaveBeenCalledTimes(1);
        });

        it('should handle zero delay', () => {
            const mockCallback = vi.fn();
            const debouncedFn = debounce(mockCallback, 0);

            debouncedFn('immediate');

            vi.advanceTimersByTime(0);

            expect(mockCallback).toHaveBeenCalledWith('immediate');
        });

        it('should handle rapid successive calls', () => {
            const mockCallback = vi.fn();
            const debouncedFn = debounce(mockCallback, 100);

            for (let i = 0; i < 10; i++) {
                debouncedFn(`char-${i}`);
                vi.advanceTimersByTime(20);
            }

            expect(mockCallback).not.toHaveBeenCalled();

            vi.advanceTimersByTime(100);

            expect(mockCallback).toHaveBeenCalledTimes(1);
            expect(mockCallback).toHaveBeenCalledWith('char-9');
        });
    });

    describe('throttle()', () => {
        it('should execute immediately on first call', () => {
            const mockCallback = vi.fn();
            const throttledFn = throttle(mockCallback, 100);

            throttledFn();

            expect(mockCallback).toHaveBeenCalledTimes(1);
        });

        it('should not execute again within throttle period', () => {
            const mockCallback = vi.fn();
            const throttledFn = throttle(mockCallback, 100);

            throttledFn();
            throttledFn();
            throttledFn();

            expect(mockCallback).toHaveBeenCalledTimes(1);
        });

        it('should execute again after throttle period', () => {
            const mockCallback = vi.fn();
            const throttledFn = throttle(mockCallback, 100);

            throttledFn();
            expect(mockCallback).toHaveBeenCalledTimes(1);

            vi.advanceTimersByTime(100);

            throttledFn();
            expect(mockCallback).toHaveBeenCalledTimes(2);
        });

        it('should pass arguments to the throttled function', () => {
            const mockCallback = vi.fn();
            const throttledFn = throttle(mockCallback, 100);

            throttledFn('arg1', 'arg2');

            expect(mockCallback).toHaveBeenCalledWith('arg1', 'arg2');
        });

        it('should use arguments from executed call, not ignored calls', () => {
            const mockCallback = vi.fn();
            const throttledFn = throttle(mockCallback, 100);

            throttledFn('first');
            throttledFn('second');
            throttledFn('third');

            expect(mockCallback).toHaveBeenCalledTimes(1);
            expect(mockCallback).toHaveBeenCalledWith('first');

            vi.advanceTimersByTime(100);

            throttledFn('fourth');
            expect(mockCallback).toHaveBeenCalledTimes(2);
            expect(mockCallback).toHaveBeenLastCalledWith('fourth');
        });

        it('should allow precise throttle intervals', () => {
            const mockCallback = vi.fn();
            const throttledFn = throttle(mockCallback, 100);

            throttledFn();
            vi.advanceTimersByTime(99);

            throttledFn();
            expect(mockCallback).toHaveBeenCalledTimes(1);

            vi.advanceTimersByTime(1);

            throttledFn();
            expect(mockCallback).toHaveBeenCalledTimes(2);
        });

        it('should handle zero throttle period', () => {
            const mockCallback = vi.fn();
            const throttledFn = throttle(mockCallback, 0);

            throttledFn('first');
            throttledFn('second');
            throttledFn('third');

            expect(mockCallback).toHaveBeenCalledTimes(3);
        });

        it('should work with scroll event pattern', () => {
            const mockCallback = vi.fn();
            const throttledFn = throttle(mockCallback, 100);

            for (let i = 0; i < 20; i++) {
                throttledFn();
                vi.advanceTimersByTime(16);
            }

            expect(mockCallback.mock.calls.length).toBeGreaterThanOrEqual(3);
            expect(mockCallback.mock.calls.length).toBeLessThanOrEqual(4);
        });
    });

    describe('integration', () => {
        it('should handle debounce and throttle independently', () => {
            const debounceFn = vi.fn();
            const throttleFn = vi.fn();

            const debounced = debounce(debounceFn, 100);
            const throttled = throttle(throttleFn, 100);

            debounced('debounce');
            throttled('throttle');

            expect(debounceFn).not.toHaveBeenCalled();
            expect(throttleFn).toHaveBeenCalledWith('throttle');

            vi.advanceTimersByTime(100);

            expect(debounceFn).toHaveBeenCalledWith('debounce');
        });

        it('should work with real-world search pattern', () => {
            const searchFn = vi.fn();
            const debouncedSearch = debounce(searchFn, 300);

            debouncedSearch('t');
            vi.advanceTimersByTime(50);
            debouncedSearch('te');
            vi.advanceTimersByTime(50);
            debouncedSearch('tes');
            vi.advanceTimersByTime(50);
            debouncedSearch('test');

            expect(searchFn).not.toHaveBeenCalled();

            vi.advanceTimersByTime(300);

            expect(searchFn).toHaveBeenCalledTimes(1);
            expect(searchFn).toHaveBeenCalledWith('test');
        });

        it('should work with real-world resize pattern', () => {
            const resizeFn = vi.fn();
            const throttledResize = throttle(resizeFn, 100);

            for (let i = 0; i < 50; i++) {
                throttledResize();
                vi.advanceTimersByTime(10);
            }

            expect(resizeFn.mock.calls.length).toBeGreaterThanOrEqual(5);
            expect(resizeFn.mock.calls.length).toBeLessThanOrEqual(6);
        });
    });
});
