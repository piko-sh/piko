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
import {
    initAutoRefreshObserver,
    stopAllAutoRefreshers,
    getActiveRefresherCount
} from '@/pk/autoRefreshObserver';

vi.mock('./coordination', () => ({
    autoRefresh: vi.fn(() => vi.fn())
}));

import {autoRefresh} from './coordination';

const mockedAutoRefresh = vi.mocked(autoRefresh);

async function flushObserver(): Promise<void> {
    await new Promise(resolve => setTimeout(resolve, 0));
}

function createAutoRefreshElement(
    interval: string,
    partialName?: string,
    opts?: {when?: string; onError?: string}
): HTMLDivElement {
    const el = document.createElement('div');
    el.setAttribute('data-auto-refresh', interval);
    if (partialName !== undefined) {
        el.setAttribute('data-partial', partialName);
    }
    if (opts?.when) {
        el.setAttribute('data-auto-refresh-when', opts.when);
    }
    if (opts?.onError) {
        el.setAttribute('data-auto-refresh-on-error', opts.onError);
    }
    return el;
}

describe('autoRefreshObserver', () => {
    let testContainer: HTMLDivElement;

    beforeEach(() => {
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);
        vi.clearAllMocks();
        mockedAutoRefresh.mockImplementation(() => vi.fn());
    });

    afterEach(async () => {
        stopAllAutoRefreshers();

        testContainer.remove();
        await flushObserver();
    });

    describe('initAutoRefreshObserver() - initialisation', () => {
        it('should process existing [data-auto-refresh] elements', () => {
            const el = createAutoRefreshElement('5000', 'my-partial');
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledTimes(1);
            expect(getActiveRefresherCount()).toBe(1);
        });

        it('should call autoRefresh with correct interval, partial name, and default options', () => {
            const el = createAutoRefreshElement('3000', 'dashboard');
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledWith('dashboard', {
                interval: 3000,
                when: undefined,
                onError: 'retry'
            });
        });

        it('should report zero active refreshers before any init', () => {
            expect(getActiveRefresherCount()).toBe(0);
        });
    });

    describe('dynamic elements via MutationObserver', () => {
        it('should start autoRefresh when an element is added after init', async () => {
            initAutoRefreshObserver();

            const el = createAutoRefreshElement('2000', 'live-feed');
            testContainer.appendChild(el);

            await flushObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledWith('live-feed', expect.objectContaining({
                interval: 2000
            }));
            expect(getActiveRefresherCount()).toBe(1);
        });

        it('should process nested elements inside an appended parent', async () => {
            initAutoRefreshObserver();

            const parent = document.createElement('div');
            const child = createAutoRefreshElement('1000', 'nested-partial');
            parent.appendChild(child);
            testContainer.appendChild(parent);

            await flushObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledWith('nested-partial', expect.objectContaining({
                interval: 1000
            }));
            expect(getActiveRefresherCount()).toBe(1);
        });

        it('should ignore non-HTMLElement nodes (e.g. text nodes)', async () => {
            initAutoRefreshObserver();

            const textNode = document.createTextNode('just some text');
            testContainer.appendChild(textNode);

            await flushObserver();

            expect(mockedAutoRefresh).not.toHaveBeenCalled();
            expect(getActiveRefresherCount()).toBe(0);
        });
    });

    describe('validation', () => {
        it('should warn and skip when interval is NaN', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const el = createAutoRefreshElement('not-a-number', 'broken');
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Invalid auto-refresh interval'),
                expect.anything()
            );
            expect(mockedAutoRefresh).not.toHaveBeenCalled();
            expect(getActiveRefresherCount()).toBe(0);

            warnSpy.mockRestore();
        });

        it('should warn and skip when interval is zero', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const el = createAutoRefreshElement('0', 'zero-interval');
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Invalid auto-refresh interval'),
                expect.anything()
            );
            expect(mockedAutoRefresh).not.toHaveBeenCalled();

            warnSpy.mockRestore();
        });

        it('should warn and skip when interval is negative', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const el = createAutoRefreshElement('-5', 'negative-interval');
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Invalid auto-refresh interval'),
                expect.anything()
            );
            expect(mockedAutoRefresh).not.toHaveBeenCalled();

            warnSpy.mockRestore();
        });

        it('should skip elements missing the data-partial attribute', () => {
            const el = createAutoRefreshElement('5000');
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(mockedAutoRefresh).not.toHaveBeenCalled();
            expect(getActiveRefresherCount()).toBe(0);
        });
    });

    describe('when conditions', () => {
        it('should pass a visibility-checking function when data-auto-refresh-when="visible"', () => {
            const el = createAutoRefreshElement('4000', 'vis-partial', {when: 'visible'});
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledTimes(1);
            const callArgs = mockedAutoRefresh.mock.calls[0];
            const whenFn = callArgs[1].when;
            expect(whenFn).toBeTypeOf('function');

            Object.defineProperty(document, 'visibilityState', {value: 'visible', configurable: true});
            expect(whenFn!()).toBe(true);

            Object.defineProperty(document, 'visibilityState', {value: 'hidden', configurable: true});
            expect(whenFn!()).toBe(false);

            Object.defineProperty(document, 'visibilityState', {value: 'visible', configurable: true});
        });

        it('should pass a focus-checking function when data-auto-refresh-when="focus"', () => {
            const el = createAutoRefreshElement('4000', 'focus-partial', {when: 'focus'});
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledTimes(1);
            const callArgs = mockedAutoRefresh.mock.calls[0];
            const whenFn = callArgs[1].when;
            expect(whenFn).toBeTypeOf('function');

            const hasFocusSpy = vi.spyOn(document, 'hasFocus');

            hasFocusSpy.mockReturnValue(true);
            expect(whenFn!()).toBe(true);

            hasFocusSpy.mockReturnValue(false);
            expect(whenFn!()).toBe(false);

            hasFocusSpy.mockRestore();
        });

        it('should not pass a when function when no when attribute is set', () => {
            const el = createAutoRefreshElement('4000', 'no-when');
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            const callArgs = mockedAutoRefresh.mock.calls[0];
            expect(callArgs[1].when).toBeUndefined();
        });
    });

    describe('error handling options', () => {
        it('should pass onError: "stop" when data-auto-refresh-on-error="stop"', () => {
            const el = createAutoRefreshElement('3000', 'err-stop', {onError: 'stop'});
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledWith('err-stop', expect.objectContaining({
                onError: 'stop'
            }));
        });

        it('should default to onError: "retry" when no on-error attribute is set', () => {
            const el = createAutoRefreshElement('3000', 'err-default');
            testContainer.appendChild(el);

            initAutoRefreshObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledWith('err-default', expect.objectContaining({
                onError: 'retry'
            }));
        });
    });

    describe('state management', () => {
        it('should not process the same element twice (duplicate prevention)', () => {
            const el = createAutoRefreshElement('5000', 'dup-partial');
            testContainer.appendChild(el);

            initAutoRefreshObserver();
            initAutoRefreshObserver();

            expect(mockedAutoRefresh).toHaveBeenCalledTimes(1);
            expect(getActiveRefresherCount()).toBe(1);
        });

        it('should return the correct active refresher count', () => {
            const el1 = createAutoRefreshElement('1000', 'count-a');
            const el2 = createAutoRefreshElement('2000', 'count-b');
            testContainer.appendChild(el1);
            testContainer.appendChild(el2);

            initAutoRefreshObserver();

            expect(getActiveRefresherCount()).toBe(2);
        });

        it('should call cleanup and decrement count when an element is removed', async () => {
            const cleanupFn = vi.fn();
            mockedAutoRefresh.mockReturnValueOnce(cleanupFn);

            const el = createAutoRefreshElement('5000', 'removable');
            testContainer.appendChild(el);

            initAutoRefreshObserver();
            expect(getActiveRefresherCount()).toBe(1);

            el.remove();
            await flushObserver();

            expect(cleanupFn).toHaveBeenCalledTimes(1);
            expect(getActiveRefresherCount()).toBe(0);
        });

        it('should clean up all descendant refreshers when a parent is removed', async () => {
            const cleanupA = vi.fn();
            const cleanupB = vi.fn();
            mockedAutoRefresh
                .mockReturnValueOnce(cleanupA)
                .mockReturnValueOnce(cleanupB);

            const parent = document.createElement('div');
            const childA = createAutoRefreshElement('1000', 'child-a');
            const childB = createAutoRefreshElement('2000', 'child-b');
            parent.appendChild(childA);
            parent.appendChild(childB);
            testContainer.appendChild(parent);

            initAutoRefreshObserver();
            expect(getActiveRefresherCount()).toBe(2);

            parent.remove();
            await flushObserver();

            expect(cleanupA).toHaveBeenCalledTimes(1);
            expect(cleanupB).toHaveBeenCalledTimes(1);
            expect(getActiveRefresherCount()).toBe(0);
        });

        it('should stop all auto-refreshers via stopAllAutoRefreshers()', () => {
            const cleanupA = vi.fn();
            const cleanupB = vi.fn();
            mockedAutoRefresh
                .mockReturnValueOnce(cleanupA)
                .mockReturnValueOnce(cleanupB);

            const el1 = createAutoRefreshElement('1000', 'stop-a');
            const el2 = createAutoRefreshElement('2000', 'stop-b');
            testContainer.appendChild(el1);
            testContainer.appendChild(el2);

            initAutoRefreshObserver();
            expect(getActiveRefresherCount()).toBe(2);

            stopAllAutoRefreshers();

            expect(cleanupA).toHaveBeenCalledTimes(1);
            expect(cleanupB).toHaveBeenCalledTimes(1);
            expect(getActiveRefresherCount()).toBe(0);
        });
    });

    describe('race conditions', () => {
        it('should handle adding then immediately removing an element before microtask flushes', async () => {
            const cleanupFn = vi.fn();
            mockedAutoRefresh.mockReturnValue(cleanupFn);

            initAutoRefreshObserver();

            const el = createAutoRefreshElement('3000', 'ephemeral');
            testContainer.appendChild(el);
            el.remove();

            await flushObserver();

            expect(getActiveRefresherCount()).toBe(0);
        });

        it('should handle multiple mutations in a single batch (add and remove different elements)', async () => {
            const cleanupExisting = vi.fn();
            const cleanupNew = vi.fn();
            mockedAutoRefresh
                .mockReturnValueOnce(cleanupExisting)
                .mockReturnValueOnce(cleanupNew);

            const existing = createAutoRefreshElement('1000', 'existing');
            testContainer.appendChild(existing);

            initAutoRefreshObserver();
            expect(getActiveRefresherCount()).toBe(1);

            const fresh = createAutoRefreshElement('2000', 'fresh');
            existing.remove();
            testContainer.appendChild(fresh);

            await flushObserver();

            expect(cleanupExisting).toHaveBeenCalledTimes(1);
            expect(getActiveRefresherCount()).toBe(1);
        });
    });
});
