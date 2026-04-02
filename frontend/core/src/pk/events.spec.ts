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
import { dispatch, listen, listenOnce, waitForEvent } from '@/pk/events';

describe('events (PK Event Helpers)', () => {
    let container: HTMLDivElement;

    beforeEach(() => {
        container = document.createElement('div');
        document.body.appendChild(container);
        vi.spyOn(console, 'warn').mockImplementation(() => {});
    });

    afterEach(() => {
        container.remove();
        vi.restoreAllMocks();
    });

    describe('dispatch', () => {
        it('should dispatch event to element by p-ref', () => {
            container.innerHTML = '<div p-ref="test-element"></div>';
            const element = container.querySelector('[p-ref="test-element"]')!;
            const handler = vi.fn();

            element.addEventListener('test-event', handler);

            dispatch('test-element', 'test-event', { data: 123 });

            expect(handler).toHaveBeenCalledTimes(1);
            expect((handler.mock.calls[0][0] as CustomEvent).detail).toEqual({ data: 123 });
        });

        it('should dispatch event to element by CSS selector', () => {
            container.innerHTML = '<div id="my-element"></div>';
            const element = container.querySelector('#my-element')!;
            const handler = vi.fn();

            element.addEventListener('click', handler);

            dispatch('#my-element', 'click');

            expect(handler).toHaveBeenCalledTimes(1);
        });

        it('should dispatch event to HTMLElement directly', () => {
            container.innerHTML = '<div></div>';
            const element = container.querySelector('div')!;
            const handler = vi.fn();

            element.addEventListener('custom', handler);

            dispatch(element, 'custom', { value: 'test' });

            expect(handler).toHaveBeenCalledTimes(1);
        });

        it('should prefer p-ref over CSS selector', () => {
            container.innerHTML = '<div p-ref="button" id="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;
            const handler = vi.fn();

            element.addEventListener('click', handler);

            dispatch('button', 'click');

            expect(handler).toHaveBeenCalledTimes(1);
        });

        it('should warn when target not found', () => {
            dispatch('non-existent', 'click');

            expect(console.warn).toHaveBeenCalledWith(
                expect.stringContaining('target "non-existent" not found')
            );
        });

        it('should bubble by default', () => {
            container.innerHTML = '<div id="parent"><button id="child"></button></div>';
            const parent = container.querySelector('#parent')!;
            const child = container.querySelector('#child')!;
            const parentHandler = vi.fn();

            parent.addEventListener('custom', parentHandler);

            dispatch(child as HTMLElement, 'custom');

            expect(parentHandler).toHaveBeenCalledTimes(1);
        });

        it('should respect bubbles: false option', () => {
            container.innerHTML = '<div id="parent"><button id="child"></button></div>';
            const parent = container.querySelector('#parent')!;
            const child = container.querySelector('#child')!;
            const parentHandler = vi.fn();

            parent.addEventListener('custom', parentHandler);

            dispatch(child as HTMLElement, 'custom', null, { bubbles: false });

            expect(parentHandler).not.toHaveBeenCalled();
        });

        it('should set composed: true by default', () => {
            container.innerHTML = '<div id="target"></div>';
            const element = container.querySelector('#target')!;
            const handler = vi.fn();

            element.addEventListener('custom', (e) => {
                handler((e as CustomEvent).composed);
            });

            dispatch('#target', 'custom');

            expect(handler).toHaveBeenCalledWith(true);
        });
    });

    describe('listen', () => {
        it('should listen for events on element by p-ref', () => {
            container.innerHTML = '<div p-ref="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;
            const handler = vi.fn();

            listen('button', 'click', handler);

            element.dispatchEvent(new CustomEvent('click'));

            expect(handler).toHaveBeenCalledTimes(1);
        });

        it('should listen for events on document with *', () => {
            const handler = vi.fn();

            listen('*', 'global-event', handler);

            document.dispatchEvent(new CustomEvent('global-event'));

            expect(handler).toHaveBeenCalledTimes(1);
        });

        it('should return unsubscribe function', () => {
            container.innerHTML = '<div p-ref="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;
            const handler = vi.fn();

            const unsubscribe = listen('button', 'click', handler);

            element.dispatchEvent(new CustomEvent('click'));
            expect(handler).toHaveBeenCalledTimes(1);

            unsubscribe();

            element.dispatchEvent(new CustomEvent('click'));
            expect(handler).toHaveBeenCalledTimes(1);
        });

        it('should pass CustomEvent to handler', () => {
            container.innerHTML = '<div p-ref="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;
            const handler = vi.fn();

            listen('button', 'custom', handler);

            element.dispatchEvent(new CustomEvent('custom', { detail: { id: 42 } }));

            expect(handler).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: 'custom',
                    detail: { id: 42 }
                })
            );
        });

        it('should warn when target not found', () => {
            const handler = vi.fn();
            const unsubscribe = listen('non-existent', 'click', handler);

            expect(console.warn).toHaveBeenCalled();
            expect(typeof unsubscribe).toBe('function');
        });

        it('should return no-op unsubscribe when target not found', () => {
            const handler = vi.fn();
            const unsubscribe = listen('non-existent', 'click', handler);

            expect(() => unsubscribe()).not.toThrow();
        });

        it('should work with HTMLElement directly', () => {
            container.innerHTML = '<div></div>';
            const element = container.querySelector('div')!;
            const handler = vi.fn();

            listen(element, 'test', handler);

            element.dispatchEvent(new CustomEvent('test'));

            expect(handler).toHaveBeenCalledTimes(1);
        });
    });

    describe('listenOnce', () => {
        it('should only trigger handler once', () => {
            container.innerHTML = '<div p-ref="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;
            const handler = vi.fn();

            listenOnce('button', 'click', handler);

            element.dispatchEvent(new CustomEvent('click'));
            element.dispatchEvent(new CustomEvent('click'));
            element.dispatchEvent(new CustomEvent('click'));

            expect(handler).toHaveBeenCalledTimes(1);
        });

        it('should return unsubscribe function for cancellation', () => {
            container.innerHTML = '<div p-ref="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;
            const handler = vi.fn();

            const unsubscribe = listenOnce('button', 'click', handler);

            unsubscribe();

            element.dispatchEvent(new CustomEvent('click'));

            expect(handler).not.toHaveBeenCalled();
        });

        it('should work with document (*)', () => {
            const handler = vi.fn();

            listenOnce('*', 'once-event', handler);

            document.dispatchEvent(new CustomEvent('once-event'));
            document.dispatchEvent(new CustomEvent('once-event'));

            expect(handler).toHaveBeenCalledTimes(1);
        });
    });

    describe('waitForEvent', () => {
        beforeEach(() => {
            vi.useFakeTimers();
        });

        afterEach(() => {
            vi.useRealTimers();
        });

        it('should resolve when event is received', async () => {
            container.innerHTML = '<div p-ref="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;

            const promise = waitForEvent<{ id: number }>('button', 'complete');

            element.dispatchEvent(new CustomEvent('complete', { detail: { id: 42 } }));

            const result = await promise;

            expect(result).toEqual({ id: 42 });
        });

        it('should timeout if event not received', async () => {
            container.innerHTML = '<div p-ref="button"></div>';

            const promise = waitForEvent('button', 'complete', 1000);

            vi.advanceTimersByTime(1000);

            await expect(promise).rejects.toThrow('Timeout waiting for event "complete"');
        });

        it('should not timeout if event received before timeout', async () => {
            container.innerHTML = '<div p-ref="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;

            const promise = waitForEvent('button', 'complete', 1000);

            vi.advanceTimersByTime(500);

            element.dispatchEvent(new CustomEvent('complete', { detail: 'done' }));

            const result = await promise;

            expect(result).toBe('done');
        });

        it('should work without timeout', async () => {
            container.innerHTML = '<div p-ref="button"></div>';
            const element = container.querySelector('[p-ref="button"]')!;

            const promise = waitForEvent('button', 'complete');

            element.dispatchEvent(new CustomEvent('complete', { detail: 'no timeout' }));

            const result = await promise;

            expect(result).toBe('no timeout');
        });

        it('should work with document (*)', async () => {
            const promise = waitForEvent('*', 'global-complete');

            document.dispatchEvent(new CustomEvent('global-complete', { detail: 'global' }));

            const result = await promise;

            expect(result).toBe('global');
        });
    });
});
