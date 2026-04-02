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

import {describe, it, expect, vi, beforeEach, afterEach} from 'vitest';
import {createHookManager, HookEvent, type HookManager} from './HookManager';

describe('HookManager', () => {
    let hookManager: HookManager;

    beforeEach(() => {
        hookManager = createHookManager();
    });

    describe('createHookManager', () => {
        it('should create a hook manager with api and methods', () => {
            expect(hookManager).toBeDefined();
            expect(hookManager.api).toBeDefined();
            expect(hookManager.emit).toBeDefined();
            expect(hookManager.processQueue).toBeDefined();
            expect(hookManager.setReady).toBeDefined();
        });

        it('should expose HookEvent constants via api.events', () => {
            expect(hookManager.api.events).toBe(HookEvent);
            expect(hookManager.api.events.FRAMEWORK_READY).toBe('framework:ready');
            expect(hookManager.api.events.PAGE_VIEW).toBe('page:view');
        });

        it('should start with ready = false', () => {
            expect(hookManager.api.ready).toBe(false);
        });
    });

    describe('api.on', () => {
        it('should register a hook and call it when event is emitted', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.PAGE_VIEW, callback);

            const payload = {
                url: 'https://example.com',
                title: 'Test Page',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            };
            hookManager.emit(HookEvent.PAGE_VIEW, payload);

            expect(callback).toHaveBeenCalledTimes(1);
            expect(callback).toHaveBeenCalledWith(payload);
        });

        it('should return an unsubscribe function', () => {
            const callback = vi.fn();
            const unsubscribe = hookManager.api.on(HookEvent.PAGE_VIEW, callback);

            expect(typeof unsubscribe).toBe('function');

            unsubscribe();

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(callback).not.toHaveBeenCalled();
        });

        it('should allow multiple hooks for the same event', () => {
            const callback1 = vi.fn();
            const callback2 = vi.fn();

            hookManager.api.on(HookEvent.PAGE_VIEW, callback1);
            hookManager.api.on(HookEvent.PAGE_VIEW, callback2);

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(callback1).toHaveBeenCalledTimes(1);
            expect(callback2).toHaveBeenCalledTimes(1);
        });

        it('should respect priority order (higher priority first)', () => {
            const order: number[] = [];

            hookManager.api.on(HookEvent.PAGE_VIEW, () => order.push(1), {priority: 1});
            hookManager.api.on(HookEvent.PAGE_VIEW, () => order.push(10), {priority: 10});
            hookManager.api.on(HookEvent.PAGE_VIEW, () => order.push(5), {priority: 5});

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(order).toEqual([10, 5, 1]);
        });

        it('should use default priority of 0', () => {
            const order: number[] = [];

            hookManager.api.on(HookEvent.PAGE_VIEW, () => order.push(0));
            hookManager.api.on(HookEvent.PAGE_VIEW, () => order.push(1), {priority: 1});
            hookManager.api.on(HookEvent.PAGE_VIEW, () => order.push(-1), {priority: -1});

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(order).toEqual([1, 0, -1]);
        });

        it('should allow custom hook ID via options', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.PAGE_VIEW, callback, {id: 'my-custom-hook'});

            hookManager.api.off(HookEvent.PAGE_VIEW, 'my-custom-hook');

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(callback).not.toHaveBeenCalled();
        });
    });

    describe('api.once', () => {
        it('should only fire the callback once', () => {
            const callback = vi.fn();
            hookManager.api.once(HookEvent.PAGE_VIEW, callback);

            const payload = {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            };

            hookManager.emit(HookEvent.PAGE_VIEW, payload);
            hookManager.emit(HookEvent.PAGE_VIEW, payload);
            hookManager.emit(HookEvent.PAGE_VIEW, payload);

            expect(callback).toHaveBeenCalledTimes(1);
        });

        it('should return an unsubscribe function', () => {
            const callback = vi.fn();
            const unsubscribe = hookManager.api.once(HookEvent.PAGE_VIEW, callback);

            unsubscribe();

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(callback).not.toHaveBeenCalled();
        });

        it('should respect priority option', () => {
            const order: number[] = [];

            hookManager.api.once(HookEvent.PAGE_VIEW, () => order.push(1), {priority: 1});
            hookManager.api.once(HookEvent.PAGE_VIEW, () => order.push(10), {priority: 10});

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(order).toEqual([10, 1]);
        });
    });

    describe('api.off', () => {
        it('should remove a hook by ID', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.PAGE_VIEW, callback, {id: 'test-hook'});

            hookManager.api.off(HookEvent.PAGE_VIEW, 'test-hook');

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(callback).not.toHaveBeenCalled();
        });

        it('should handle removing non-existent hook gracefully', () => {
            expect(() => {
                hookManager.api.off(HookEvent.PAGE_VIEW, 'non-existent');
            }).not.toThrow();
        });

        it('should handle removing from non-existent event gracefully', () => {
            expect(() => {
                hookManager.api.off(HookEvent.NAVIGATION_START, 'any-id');
            }).not.toThrow();
        });
    });

    describe('api.clear', () => {
        it('should clear all hooks for a specific event', () => {
            const callback1 = vi.fn();
            const callback2 = vi.fn();
            const callback3 = vi.fn();

            hookManager.api.on(HookEvent.PAGE_VIEW, callback1);
            hookManager.api.on(HookEvent.PAGE_VIEW, callback2);
            hookManager.api.on(HookEvent.NAVIGATION_START, callback3);

            hookManager.api.clear(HookEvent.PAGE_VIEW);

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            hookManager.emit(HookEvent.NAVIGATION_START, {
                url: 'https://example.com',
                timestamp: Date.now()
            });

            expect(callback1).not.toHaveBeenCalled();
            expect(callback2).not.toHaveBeenCalled();
            expect(callback3).toHaveBeenCalledTimes(1);
        });

        it('should clear all hooks when no event specified', () => {
            const callback1 = vi.fn();
            const callback2 = vi.fn();

            hookManager.api.on(HookEvent.PAGE_VIEW, callback1);
            hookManager.api.on(HookEvent.NAVIGATION_START, callback2);

            hookManager.api.clear();

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            hookManager.emit(HookEvent.NAVIGATION_START, {
                url: 'https://example.com',
                timestamp: Date.now()
            });

            expect(callback1).not.toHaveBeenCalled();
            expect(callback2).not.toHaveBeenCalled();
        });
    });

    describe('emit', () => {
        it('should call all registered hooks with the payload', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.FRAMEWORK_READY, callback);

            const payload = {version: '1.0.0', loadTime: 100, timestamp: Date.now()};
            hookManager.emit(HookEvent.FRAMEWORK_READY, payload);

            expect(callback).toHaveBeenCalledWith(payload);
        });

        it('should not throw if no hooks registered for event', () => {
            expect(() => {
                hookManager.emit(HookEvent.PAGE_VIEW, {
                    url: 'https://example.com',
                    title: 'Test',
                    referrer: '',
                    isInitialLoad: true,
                    timestamp: Date.now()
                });
            }).not.toThrow();
        });

        it('should catch errors in hook callbacks and continue executing others', () => {
            const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            const callback1 = vi.fn(() => {
                throw new Error('Hook error');
            });
            const callback2 = vi.fn();

            hookManager.api.on(HookEvent.PAGE_VIEW, callback1);
            hookManager.api.on(HookEvent.PAGE_VIEW, callback2);

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(callback1).toHaveBeenCalled();
            expect(callback2).toHaveBeenCalled();
            expect(consoleErrorSpy).toHaveBeenCalled();

            consoleErrorSpy.mockRestore();
        });

        it('should remove once hooks after they fire', () => {
            const callback = vi.fn();
            hookManager.api.once(HookEvent.PAGE_VIEW, callback);

            const payload = {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            };

            hookManager.emit(HookEvent.PAGE_VIEW, payload);
            expect(callback).toHaveBeenCalledTimes(1);

            hookManager.emit(HookEvent.PAGE_VIEW, payload);
            expect(callback).toHaveBeenCalledTimes(1);
        });
    });

    describe('setReady', () => {
        it('should set ready to true', () => {
            expect(hookManager.api.ready).toBe(false);
            hookManager.setReady();
            expect(hookManager.api.ready).toBe(true);
        });
    });

    describe('processQueue', () => {
        let originalQueue: unknown;

        beforeEach(() => {
            originalQueue = (window as unknown as {__PP_HOOKS_QUEUE__?: unknown}).__PP_HOOKS_QUEUE__;
        });

        afterEach(() => {
            if (originalQueue !== undefined) {
                (window as unknown as {__PP_HOOKS_QUEUE__?: unknown}).__PP_HOOKS_QUEUE__ = originalQueue;
            } else {
                delete (window as unknown as {__PP_HOOKS_QUEUE__?: unknown}).__PP_HOOKS_QUEUE__;
            }
        });

        it('should process pre-init queued hooks', () => {
            const callback = vi.fn();

            (window as unknown as {__PP_HOOKS_QUEUE__: unknown[];}).__PP_HOOKS_QUEUE__ = [
                {event: HookEvent.PAGE_VIEW, callback, options: {}}
            ];

            hookManager.processQueue();

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(callback).toHaveBeenCalledTimes(1);
        });

        it('should clear the queue after processing', () => {
            const callback = vi.fn();

            (window as unknown as {__PP_HOOKS_QUEUE__: unknown[];}).__PP_HOOKS_QUEUE__ = [
                {event: HookEvent.PAGE_VIEW, callback, options: {}}
            ];

            hookManager.processQueue();

            expect((window as unknown as {__PP_HOOKS_QUEUE__: unknown[];}).__PP_HOOKS_QUEUE__).toEqual([]);
        });

        it('should handle empty queue', () => {
            (window as unknown as {__PP_HOOKS_QUEUE__: unknown[];}).__PP_HOOKS_QUEUE__ = [];

            expect(() => hookManager.processQueue()).not.toThrow();
        });

        it('should handle missing queue', () => {
            delete (window as unknown as {__PP_HOOKS_QUEUE__?: unknown}).__PP_HOOKS_QUEUE__;

            expect(() => hookManager.processQueue()).not.toThrow();
        });

        it('should handle non-array queue', () => {
            (window as unknown as {__PP_HOOKS_QUEUE__: unknown}).__PP_HOOKS_QUEUE__ = 'not-an-array';

            expect(() => hookManager.processQueue()).not.toThrow();
        });

        it('should register queued hooks with options', () => {
            const callback = vi.fn();

            (window as unknown as {__PP_HOOKS_QUEUE__: unknown[];}).__PP_HOOKS_QUEUE__ = [
                {event: HookEvent.PAGE_VIEW, callback, options: {id: 'queued-hook', priority: 5}}
            ];

            hookManager.processQueue();

            hookManager.api.off(HookEvent.PAGE_VIEW, 'queued-hook');

            hookManager.emit(HookEvent.PAGE_VIEW, {
                url: 'https://example.com',
                title: 'Test',
                referrer: '',
                isInitialLoad: true,
                timestamp: Date.now()
            });

            expect(callback).not.toHaveBeenCalled();
        });
    });

    describe('event types', () => {
        it('should emit navigation:complete with duration', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.NAVIGATION_COMPLETE, callback);

            const payload = {
                url: 'https://example.com',
                previousUrl: 'https://example.com/old',
                timestamp: Date.now(),
                duration: 150
            };
            hookManager.emit(HookEvent.NAVIGATION_COMPLETE, payload);

            expect(callback).toHaveBeenCalledWith(payload);
        });

        it('should emit navigation:error with error message', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.NAVIGATION_ERROR, callback);

            const payload = {
                url: 'https://example.com',
                error: 'Network error',
                timestamp: Date.now()
            };
            hookManager.emit(HookEvent.NAVIGATION_ERROR, payload);

            expect(callback).toHaveBeenCalledWith(payload);
        });

        it('should emit action:complete with success and duration', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.ACTION_COMPLETE, callback);

            const payload = {
                action: '/api/submit',
                method: 'POST',
                elementTag: 'BUTTON',
                timestamp: Date.now(),
                success: true,
                statusCode: 200,
                duration: 50
            };
            hookManager.emit(HookEvent.ACTION_COMPLETE, payload);

            expect(callback).toHaveBeenCalledWith(payload);
        });

        it('should emit modal:open with modal ID', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.MODAL_OPEN, callback);

            const payload = {
                modalId: 'confirm-dialog',
                timestamp: Date.now()
            };
            hookManager.emit(HookEvent.MODAL_OPEN, payload);

            expect(callback).toHaveBeenCalledWith(payload);
        });

        it('should emit error with context', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.ERROR, callback);

            const payload = {
                message: 'Something went wrong',
                type: 'TypeError',
                context: 'navigation' as const,
                stack: 'Error: ...',
                url: 'https://example.com',
                timestamp: Date.now()
            };
            hookManager.emit(HookEvent.ERROR, payload);

            expect(callback).toHaveBeenCalledWith(payload);
        });

        it('should emit form:dirty event', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.FORM_DIRTY, callback);

            const payload = {
                formId: 'login-form',
                timestamp: Date.now()
            };
            hookManager.emit(HookEvent.FORM_DIRTY, payload);

            expect(callback).toHaveBeenCalledWith(payload);
        });

        it('should emit network:online event', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.NETWORK_ONLINE, callback);

            const payload = {timestamp: Date.now()};
            hookManager.emit(HookEvent.NETWORK_ONLINE, payload);

            expect(callback).toHaveBeenCalledWith(payload);
        });
    });
});
