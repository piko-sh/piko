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
import { trace, traceLog, traceAsync } from '@/pk/trace';

describe('trace (PK Tracing)', () => {
    beforeEach(() => {
        trace.disable();
        trace.clear();
        trace.enable({ partialReloads: true, events: true, handlers: true, sse: true });
        trace.disable();
        trace.clear();
        vi.spyOn(console, 'log').mockImplementation(() => {});
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe('enable/disable', () => {
        it('should start disabled by default', () => {
            expect(trace.isEnabled()).toBe(false);
        });

        it('should enable tracing', () => {
            trace.enable();
            expect(trace.isEnabled()).toBe(true);
        });

        it('should disable tracing', () => {
            trace.enable();
            trace.disable();
            expect(trace.isEnabled()).toBe(false);
        });

        it('should accept partial config when enabling', () => {
            trace.enable({ partialReloads: false });
            expect(trace.isEnabled()).toBe(true);
        });
    });

    describe('tracePartialReload', () => {
        it('should not record when disabled', () => {
            trace.tracePartialReload('test', 100);
            expect(trace.getEntries()).toHaveLength(0);
        });

        it('should record partial reload when enabled', () => {
            trace.enable();
            trace.tracePartialReload('cart-summary', 42);

            const entries = trace.getEntries();
            expect(entries).toHaveLength(1);
            expect(entries[0].type).toBe('partial');
            expect(entries[0].name).toBe('cart-summary');
            expect(entries[0].duration).toBe(42);
        });

        it('should not record when partialReloads config is false', () => {
            trace.enable({ partialReloads: false });
            trace.tracePartialReload('test', 100);
            expect(trace.getEntries()).toHaveLength(0);
        });

        it('should include metadata when provided', () => {
            trace.enable();
            trace.tracePartialReload('test', 50, { category: 'electronics' });

            const entries = trace.getEntries();
            expect(entries[0].metadata).toEqual({ category: 'electronics' });
        });
    });

    describe('traceEvent', () => {
        it('should not record when disabled', () => {
            trace.traceEvent('cart:updated', 'button');
            expect(trace.getEntries()).toHaveLength(0);
        });

        it('should record event when enabled', () => {
            trace.enable();
            trace.traceEvent('cart:updated', 'button', { count: 3 });

            const entries = trace.getEntries();
            expect(entries).toHaveLength(1);
            expect(entries[0].type).toBe('event');
            expect(entries[0].name).toBe('cart:updated');
            expect(entries[0].metadata).toEqual({ source: 'button', payload: { count: 3 } });
        });

        it('should not record when events config is false', () => {
            trace.enable({ events: false });
            trace.traceEvent('test', 'source');
            expect(trace.getEntries()).toHaveLength(0);
        });
    });

    describe('traceHandler', () => {
        it('should not record when disabled', () => {
            trace.traceHandler('handleClick', 10);
            expect(trace.getEntries()).toHaveLength(0);
        });

        it('should record handler when enabled', () => {
            trace.enable();
            trace.traceHandler('handleClick', 25);

            const entries = trace.getEntries();
            expect(entries).toHaveLength(1);
            expect(entries[0].type).toBe('handler');
            expect(entries[0].name).toBe('handleClick');
            expect(entries[0].duration).toBe(25);
        });

        it('should not record when handlers config is false', () => {
            trace.enable({ handlers: false });
            trace.traceHandler('test', 10);
            expect(trace.getEntries()).toHaveLength(0);
        });
    });

    describe('traceSSE', () => {
        it('should not record when disabled', () => {
            trace.traceSSE('/api/events', 'connect');
            expect(trace.getEntries()).toHaveLength(0);
        });

        it('should record SSE event when enabled', () => {
            trace.enable();
            trace.traceSSE('/api/events', 'message', { id: 123 });

            const entries = trace.getEntries();
            expect(entries).toHaveLength(1);
            expect(entries[0].type).toBe('sse');
            expect(entries[0].name).toBe('/api/events');
            expect(entries[0].metadata).toEqual({ event: 'message', data: { id: 123 } });
        });

        it('should not record when sse config is false', () => {
            trace.enable({ sse: false });
            trace.traceSSE('/api/events', 'connect');
            expect(trace.getEntries()).toHaveLength(0);
        });
    });

    describe('clear', () => {
        it('should clear all entries', () => {
            trace.enable();
            trace.tracePartialReload('test', 100);
            trace.traceEvent('event', 'source');

            expect(trace.getEntries()).toHaveLength(2);

            trace.clear();

            expect(trace.getEntries()).toHaveLength(0);
        });
    });

    describe('getEntries', () => {
        it('should return a copy of entries', () => {
            trace.enable();
            trace.tracePartialReload('test', 100);

            const entries1 = trace.getEntries();
            const entries2 = trace.getEntries();

            expect(entries1).not.toBe(entries2);
            expect(entries1).toEqual(entries2);
        });
    });

    describe('getMetrics', () => {
        it('should return empty object when no entries', () => {
            const metrics = trace.getMetrics();
            expect(metrics).toEqual({});
        });

        it('should aggregate metrics by name', () => {
            trace.enable();
            trace.tracePartialReload('cart', 100);
            trace.tracePartialReload('cart', 200);
            trace.tracePartialReload('cart', 150);

            const metrics = trace.getMetrics();

            expect(metrics['cart']).toBeDefined();
            expect(metrics['cart'].count).toBe(3);
            expect(metrics['cart'].avgDuration).toBe(150);
            expect(metrics['cart'].maxDuration).toBe(200);
            expect(metrics['cart'].minDuration).toBe(100);
        });

        it('should track separate metrics for different names', () => {
            trace.enable();
            trace.tracePartialReload('cart', 100);
            trace.tracePartialReload('header', 50);
            trace.tracePartialReload('cart', 200);

            const metrics = trace.getMetrics();

            expect(metrics['cart'].count).toBe(2);
            expect(metrics['header'].count).toBe(1);
        });

        it('should handle entries without duration', () => {
            trace.enable();
            trace.traceEvent('click', 'button');

            const metrics = trace.getMetrics();

            expect(metrics['click']).toBeDefined();
            expect(metrics['click'].count).toBe(1);
            expect(metrics['click'].avgDuration).toBe(0);
        });
    });

    describe('max entries limit', () => {
        it('should trim old entries when exceeding max', () => {
            trace.enable();

            for (let i = 0; i < 1100; i++) {
                trace.tracePartialReload(`entry-${i}`, i);
            }

            const entries = trace.getEntries();
            expect(entries.length).toBe(1000);
            expect(entries[0].name).toBe('entry-100');
            expect(entries[999].name).toBe('entry-1099');
        });
    });

    describe('traceLog', () => {
        it('should not log when tracing is disabled', () => {
            traceLog('test', { data: 123 });
            expect(trace.getEntries()).toHaveLength(0);
        });

        it('should log when tracing is enabled', () => {
            trace.enable();
            traceLog('custom-event', { value: 'test' });

            const entries = trace.getEntries();
            expect(entries).toHaveLength(1);
            expect(entries[0].type).toBe('event');
            expect(entries[0].name).toBe('custom-event');
        });
    });

    describe('traceAsync', () => {
        beforeEach(() => {
            vi.useFakeTimers();
        });

        afterEach(() => {
            vi.useRealTimers();
        });

        it('should wrap async function and trace duration', async () => {
            trace.enable();

            const asyncFn = vi.fn().mockImplementation(async () => {
                await new Promise(r => setTimeout(r, 100));
                return 'result';
            });

            const traced = traceAsync('myAsyncFn', asyncFn);
            const resultPromise = traced();

            vi.advanceTimersByTime(100);

            const result = await resultPromise;

            expect(result).toBe('result');

            const entries = trace.getEntries();
            expect(entries).toHaveLength(1);
            expect(entries[0].type).toBe('handler');
            expect(entries[0].name).toBe('myAsyncFn');
            expect(entries[0].duration).toBeGreaterThanOrEqual(0);
        });

        it('should trace even if function throws', async () => {
            trace.enable();

            const asyncFn = vi.fn().mockRejectedValue(new Error('test error'));
            const traced = traceAsync('failingFn', asyncFn);

            await expect(traced()).rejects.toThrow('test error');

            const entries = trace.getEntries();
            expect(entries).toHaveLength(1);
            expect(entries[0].name).toBe('failingFn');
        });
    });
});
