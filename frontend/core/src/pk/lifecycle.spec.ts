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
    onCleanup,
    _runPageCleanup,
    _initCleanupObserver,
    _disconnectCleanupObserver,
    _registerLifecycle,
    _addLifecycleCallback,
    _executeConnected,
    _executeDisconnected,
    _executeBeforeRender,
    _executeAfterRender,
    _executeUpdated,
    _executeConnectedForPartials,
    _hasLifecycleCallbacks,
} from '@/pk/lifecycle';

describe('lifecycle (PK Cleanup Management)', () => {
    let testContainer: HTMLDivElement;

    beforeEach(() => {
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);
    });

    afterEach(() => {
        _disconnectCleanupObserver();
        _runPageCleanup();
        testContainer.remove();
        vi.clearAllMocks();
    });

    describe('onCleanup() - page-level', () => {
        it('should register page-level cleanup when no scope provided', () => {
            const cleanup = vi.fn();
            onCleanup(cleanup);

            expect(cleanup).not.toHaveBeenCalled();

            _runPageCleanup();

            expect(cleanup).toHaveBeenCalledTimes(1);
        });

        it('should run multiple page cleanups in order', () => {
            const order: number[] = [];

            onCleanup(() => order.push(1));
            onCleanup(() => order.push(2));
            onCleanup(() => order.push(3));

            _runPageCleanup();

            expect(order).toEqual([1, 2, 3]);
        });

        it('should clear cleanup queue after running', () => {
            const cleanup = vi.fn();
            onCleanup(cleanup);

            _runPageCleanup();
            _runPageCleanup();

            expect(cleanup).toHaveBeenCalledTimes(1);
        });

        it('should catch errors in cleanup functions', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const errorCleanup = vi.fn(() => {
                throw new Error('Cleanup error');
            });
            const goodCleanup = vi.fn();

            onCleanup(errorCleanup);
            onCleanup(goodCleanup);

            _runPageCleanup();

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in page cleanup'),
                expect.any(Error)
            );
            expect(goodCleanup).toHaveBeenCalled();

            errorSpy.mockRestore();
        });
    });

    describe('onCleanup() - element-scoped', () => {
        it('should register element-scoped cleanup', () => {
            const element = document.createElement('div');
            testContainer.appendChild(element);

            const cleanup = vi.fn();
            onCleanup(cleanup, element);

            expect(cleanup).not.toHaveBeenCalled();
        });

        it('should run element cleanup when element is removed', async () => {
            _initCleanupObserver();

            const element = document.createElement('div');
            testContainer.appendChild(element);

            const cleanup = vi.fn();
            onCleanup(cleanup, element);

            element.remove();

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(cleanup).toHaveBeenCalledTimes(1);
        });

        it('should run cleanup for nested elements when parent is removed', async () => {
            _initCleanupObserver();

            const parent = document.createElement('div');
            const child = document.createElement('span');
            parent.appendChild(child);
            testContainer.appendChild(parent);

            const parentCleanup = vi.fn();
            const childCleanup = vi.fn();

            onCleanup(parentCleanup, parent);
            onCleanup(childCleanup, child);

            parent.remove();

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(parentCleanup).toHaveBeenCalledTimes(1);
            expect(childCleanup).toHaveBeenCalledTimes(1);
        });

        it('should handle multiple cleanups for same element', async () => {
            _initCleanupObserver();

            const element = document.createElement('div');
            testContainer.appendChild(element);

            const cleanup1 = vi.fn();
            const cleanup2 = vi.fn();

            onCleanup(cleanup1, element);
            onCleanup(cleanup2, element);

            element.remove();

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(cleanup1).toHaveBeenCalledTimes(1);
            expect(cleanup2).toHaveBeenCalledTimes(1);
        });

        it('should catch errors in element cleanup', async () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            _initCleanupObserver();

            const element = document.createElement('div');
            testContainer.appendChild(element);

            const errorCleanup = vi.fn(() => {
                throw new Error('Element cleanup error');
            });
            const goodCleanup = vi.fn();

            onCleanup(errorCleanup, element);
            onCleanup(goodCleanup, element);

            element.remove();

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in element cleanup'),
                expect.any(Error)
            );
            expect(goodCleanup).toHaveBeenCalled();

            errorSpy.mockRestore();
        });
    });

    describe('_initCleanupObserver()', () => {
        it('should only initialise observer once', () => {
            _initCleanupObserver();
            _initCleanupObserver();
            _initCleanupObserver();

            const element = document.createElement('div');
            testContainer.appendChild(element);

            const cleanup = vi.fn();
            onCleanup(cleanup, element);
        });
    });

    describe('integration', () => {
        it('should handle mixed page and element cleanups', async () => {
            _initCleanupObserver();

            const element = document.createElement('div');
            testContainer.appendChild(element);

            const pageCleanup = vi.fn();
            const elementCleanup = vi.fn();

            onCleanup(pageCleanup);
            onCleanup(elementCleanup, element);

            element.remove();
            await new Promise(resolve => setTimeout(resolve, 0));

            expect(elementCleanup).toHaveBeenCalledTimes(1);
            expect(pageCleanup).not.toHaveBeenCalled();

            _runPageCleanup();

            expect(pageCleanup).toHaveBeenCalledTimes(1);
        });

        it('should handle real-world cleanup scenario', async () => {
            _initCleanupObserver();

            let intervalCount = 0;
            const intervalId = setInterval(() => intervalCount++, 10);

            onCleanup(() => clearInterval(intervalId));

            await new Promise(resolve => setTimeout(resolve, 50));
            expect(intervalCount).toBeGreaterThan(0);

            const countBeforeCleanup = intervalCount;

            _runPageCleanup();

            await new Promise(resolve => setTimeout(resolve, 50));
            expect(intervalCount).toBe(countBeforeCleanup);
        });
    });
});

describe('lifecycle (PK Lifecycle Callbacks)', () => {
    let testContainer: HTMLDivElement;

    beforeEach(() => {
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);
    });

    afterEach(() => {
        _disconnectCleanupObserver();
        _runPageCleanup();
        testContainer.remove();
        vi.clearAllMocks();
    });

    describe('_registerLifecycle()', () => {
        it('should register all five callback types', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'test-1');

            const callbacks = {
                onConnected: vi.fn(),
                onDisconnected: vi.fn(),
                onBeforeRender: vi.fn(),
                onAfterRender: vi.fn(),
                onUpdated: vi.fn(),
            };

            _registerLifecycle(scope, callbacks);

            expect(_hasLifecycleCallbacks(scope)).toBe(true);
        });

        it('should auto-execute onConnected when element is already in the DOM', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'test-2');
            testContainer.appendChild(scope);

            const onConnected = vi.fn();
            _registerLifecycle(scope, { onConnected });

            expect(onConnected).toHaveBeenCalledTimes(1);
        });

        it('should not auto-execute onConnected when element is not in the DOM', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'test-3');

            const onConnected = vi.fn();
            _registerLifecycle(scope, { onConnected });

            expect(onConnected).not.toHaveBeenCalled();
        });

        it('should allow partial update of callbacks on the same scope', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'test-4');

            const onConnected = vi.fn();
            const onDisconnected = vi.fn();

            _registerLifecycle(scope, { onConnected });
            _registerLifecycle(scope, { onDisconnected });

            expect(_hasLifecycleCallbacks(scope)).toBe(true);

            _executeDisconnected(scope);
            expect(onDisconnected).toHaveBeenCalledTimes(1);
        });

        it('should not re-fire onConnected if already connected once', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'test-5');
            testContainer.appendChild(scope);

            const onConnected = vi.fn();
            _registerLifecycle(scope, { onConnected });

            expect(onConnected).toHaveBeenCalledTimes(1);

            _registerLifecycle(scope, { onDisconnected: vi.fn() });
            expect(onConnected).toHaveBeenCalledTimes(1);
        });
    });

    describe('_executeConnected()', () => {
        it('should fire the onConnected callback when registered', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'conn-1');

            const onConnected = vi.fn();
            _registerLifecycle(scope, { onConnected });

            _executeConnected(scope);

            expect(onConnected).toHaveBeenCalledTimes(1);
        });

        it('should only fire onConnected once per lifecycle', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'conn-2');

            const onConnected = vi.fn();
            _registerLifecycle(scope, { onConnected });

            _executeConnected(scope);
            _executeConnected(scope);
            _executeConnected(scope);

            expect(onConnected).toHaveBeenCalledTimes(1);
        });

        it('should no-op on unregistered scope', () => {
            const scope = document.createElement('div');

            expect(() => _executeConnected(scope)).not.toThrow();
        });

        it('should catch and log errors thrown in the onConnected callback', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'conn-err');

            const onConnected = vi.fn(() => {
                throw new Error('connected boom');
            });
            _registerLifecycle(scope, { onConnected });

            _executeConnected(scope);

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in onConnected'),
                expect.any(Error),
            );
            errorSpy.mockRestore();
        });
    });

    describe('_executeDisconnected()', () => {
        it('should fire the onDisconnected callback when registered', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'disc-1');

            const onDisconnected = vi.fn();
            _registerLifecycle(scope, { onDisconnected });

            _executeDisconnected(scope);

            expect(onDisconnected).toHaveBeenCalledTimes(1);
        });

        it('should no-op on unregistered scope', () => {
            const scope = document.createElement('div');

            expect(() => _executeDisconnected(scope)).not.toThrow();
        });

        it('should reset connectedOnce flag allowing re-connect', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'disc-2');

            const onConnected = vi.fn();
            _registerLifecycle(scope, { onConnected });

            _executeConnected(scope);
            expect(onConnected).toHaveBeenCalledTimes(1);

            _executeDisconnected(scope);

            _executeConnected(scope);
            expect(onConnected).toHaveBeenCalledTimes(2);
        });

        it('should run partial cleanups on disconnect', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'disc-3');

            const onConnected = vi.fn();
            _registerLifecycle(scope, { onConnected });
            _executeConnected(scope);

            const onDisconnected = vi.fn();
            _registerLifecycle(scope, { onDisconnected });

            _executeDisconnected(scope);

            expect(onDisconnected).toHaveBeenCalledTimes(1);
        });

        it('should catch and log errors thrown in onDisconnected callback', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'disc-err');

            const onDisconnected = vi.fn(() => {
                throw new Error('disconnected boom');
            });
            _registerLifecycle(scope, { onDisconnected });

            _executeDisconnected(scope);

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in onDisconnected'),
                expect.any(Error),
            );
            errorSpy.mockRestore();
        });
    });

    describe('_executeBeforeRender()', () => {
        it('should fire onBeforeRender callback when registered', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'br-1');

            const onBeforeRender = vi.fn();
            _registerLifecycle(scope, { onBeforeRender });

            _executeBeforeRender(scope);

            expect(onBeforeRender).toHaveBeenCalledTimes(1);
        });

        it('should no-op on unregistered scope', () => {
            const scope = document.createElement('div');

            expect(() => _executeBeforeRender(scope)).not.toThrow();
        });

        it('should no-op when onBeforeRender is not in the callbacks', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'br-2');

            _registerLifecycle(scope, { onConnected: vi.fn() });

            expect(() => _executeBeforeRender(scope)).not.toThrow();
        });

        it('should catch and log errors thrown in onBeforeRender callback', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'br-err');

            const onBeforeRender = vi.fn(() => {
                throw new Error('before render boom');
            });
            _registerLifecycle(scope, { onBeforeRender });

            _executeBeforeRender(scope);

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in onBeforeRender'),
                expect.any(Error),
            );
            errorSpy.mockRestore();
        });
    });

    describe('_executeAfterRender()', () => {
        it('should fire onAfterRender callback when registered', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'ar-1');

            const onAfterRender = vi.fn();
            _registerLifecycle(scope, { onAfterRender });

            _executeAfterRender(scope);

            expect(onAfterRender).toHaveBeenCalledTimes(1);
        });

        it('should no-op on unregistered scope', () => {
            const scope = document.createElement('div');

            expect(() => _executeAfterRender(scope)).not.toThrow();
        });

        it('should no-op when onAfterRender is not in the callbacks', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'ar-2');

            _registerLifecycle(scope, { onConnected: vi.fn() });

            expect(() => _executeAfterRender(scope)).not.toThrow();
        });

        it('should catch and log errors thrown in onAfterRender callback', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'ar-err');

            const onAfterRender = vi.fn(() => {
                throw new Error('after render boom');
            });
            _registerLifecycle(scope, { onAfterRender });

            _executeAfterRender(scope);

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in onAfterRender'),
                expect.any(Error),
            );
            errorSpy.mockRestore();
        });
    });

    describe('_executeUpdated()', () => {
        it('should fire onUpdated callback when registered', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'upd-1');

            const onUpdated = vi.fn();
            _registerLifecycle(scope, { onUpdated });

            _executeUpdated(scope);

            expect(onUpdated).toHaveBeenCalledTimes(1);
        });

        it('should pass context to the onUpdated callback', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'upd-2');

            const onUpdated = vi.fn();
            _registerLifecycle(scope, { onUpdated });

            const context = { reason: 'data-change', field: 'name' };
            _executeUpdated(scope, context);

            expect(onUpdated).toHaveBeenCalledWith(context);
        });

        it('should no-op on unregistered scope', () => {
            const scope = document.createElement('div');

            expect(() => _executeUpdated(scope)).not.toThrow();
        });

        it('should no-op when onUpdated is not in the callbacks', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'upd-3');

            _registerLifecycle(scope, { onConnected: vi.fn() });

            expect(() => _executeUpdated(scope, 'some context')).not.toThrow();
        });

        it('should catch and log errors thrown in onUpdated callback', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'upd-err');

            const onUpdated = vi.fn(() => {
                throw new Error('updated boom');
            });
            _registerLifecycle(scope, { onUpdated });

            _executeUpdated(scope);

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in onUpdated'),
                expect.any(Error),
            );
            errorSpy.mockRestore();
        });
    });

    describe('_executeConnectedForPartials()', () => {
        it('should execute onConnected for partials within a container', () => {
            const container = document.createElement('div');
            testContainer.appendChild(container);

            const partial1 = document.createElement('div');
            partial1.setAttribute('partial', 'fp-1');
            container.appendChild(partial1);

            const partial2 = document.createElement('div');
            partial2.setAttribute('partial', 'fp-2');
            container.appendChild(partial2);

            const onConnected1 = vi.fn();
            const onConnected2 = vi.fn();

            _registerLifecycle(partial1, { onConnected: onConnected1 });
            _registerLifecycle(partial2, { onConnected: onConnected2 });

            _executeDisconnected(partial1);
            _executeDisconnected(partial2);
            onConnected1.mockClear();
            onConnected2.mockClear();

            _executeConnectedForPartials(container);

            expect(onConnected1).toHaveBeenCalledTimes(1);
            expect(onConnected2).toHaveBeenCalledTimes(1);
        });

        it('should skip partials that are already connected', () => {
            const container = document.createElement('div');
            testContainer.appendChild(container);

            const partial = document.createElement('div');
            partial.setAttribute('partial', 'fp-3');
            container.appendChild(partial);

            const onConnected = vi.fn();
            _registerLifecycle(partial, { onConnected });

            expect(onConnected).toHaveBeenCalledTimes(1);

            onConnected.mockClear();
            _executeConnectedForPartials(container);

            expect(onConnected).not.toHaveBeenCalled();
        });

        it('should skip elements without registered lifecycle', () => {
            const container = document.createElement('div');
            testContainer.appendChild(container);

            const partial = document.createElement('div');
            partial.setAttribute('partial', 'fp-4');
            container.appendChild(partial);

            expect(() => _executeConnectedForPartials(container)).not.toThrow();
        });
    });

    describe('_hasLifecycleCallbacks()', () => {
        it('should return true when lifecycle is registered', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'has-1');

            _registerLifecycle(scope, { onConnected: vi.fn() });

            expect(_hasLifecycleCallbacks(scope)).toBe(true);
        });

        it('should return false when lifecycle is not registered', () => {
            const scope = document.createElement('div');

            expect(_hasLifecycleCallbacks(scope)).toBe(false);
        });
    });

    describe('runElementCleanups (via MutationObserver)', () => {
        it('should fire onDisconnected for partial elements in a removed subtree', async () => {
            _initCleanupObserver();

            const partial = document.createElement('div');
            partial.setAttribute('partial', 'rec-1');
            testContainer.appendChild(partial);

            const onDisconnected = vi.fn();
            _registerLifecycle(partial, { onDisconnected });

            partial.remove();
            await new Promise(resolve => setTimeout(resolve, 0));

            expect(onDisconnected).toHaveBeenCalledTimes(1);
        });

        it('should fire onDisconnected for nested partial elements when parent is removed', async () => {
            _initCleanupObserver();

            const parent = document.createElement('div');
            testContainer.appendChild(parent);

            const child = document.createElement('div');
            child.setAttribute('partial', 'rec-2');
            parent.appendChild(child);

            const onDisconnected = vi.fn();
            _registerLifecycle(child, { onDisconnected });

            parent.remove();
            await new Promise(resolve => setTimeout(resolve, 0));

            expect(onDisconnected).toHaveBeenCalledTimes(1);
        });

        it('should fire both element-scoped cleanups and onDisconnected for partials', async () => {
            _initCleanupObserver();

            const partial = document.createElement('div');
            partial.setAttribute('partial', 'rec-3');
            testContainer.appendChild(partial);

            const onDisconnected = vi.fn();
            const elementCleanup = vi.fn();

            _registerLifecycle(partial, { onDisconnected });
            onCleanup(elementCleanup, partial);

            partial.remove();
            await new Promise(resolve => setTimeout(resolve, 0));

            expect(onDisconnected).toHaveBeenCalledTimes(1);
            expect(elementCleanup).toHaveBeenCalledTimes(1);
        });

        it('should handle errors in child element cleanups gracefully', async () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            _initCleanupObserver();

            const parent = document.createElement('div');
            const child = document.createElement('span');
            parent.appendChild(child);
            testContainer.appendChild(parent);

            const errorCleanup = vi.fn(() => {
                throw new Error('child cleanup boom');
            });
            const goodCleanup = vi.fn();

            onCleanup(errorCleanup, child);
            onCleanup(goodCleanup, child);

            parent.remove();
            await new Promise(resolve => setTimeout(resolve, 0));

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in element cleanup'),
                expect.any(Error),
            );
            expect(goodCleanup).toHaveBeenCalled();
            errorSpy.mockRestore();
        });
    });

    describe('createDebugAPI (via window.__pikoDebug)', () => {
        it('should be available on the window object', () => {
            expect(window.__pikoDebug).toBeDefined();
        });

        it('isAvailable() should return true', () => {
            expect(window.__pikoDebug!.isAvailable()).toBe(true);
        });

        describe('getPartialInfo()', () => {
            it('should return correct info for a registered partial', () => {
                const scope = document.createElement('div');
                scope.setAttribute('partial', 'dbg-id-1');
                scope.setAttribute('partial_name', 'MyPartial');

                const onConnected = vi.fn();
                const onDisconnected = vi.fn();
                _registerLifecycle(scope, { onConnected, onDisconnected });

                const info = window.__pikoDebug!.getPartialInfo(scope);

                expect(info.exists).toBe(true);
                expect(info.partialId).toBe('dbg-id-1');
                expect(info.partialName).toBe('MyPartial');
                expect(info.registeredCallbacks).toContain('onConnected');
                expect(info.registeredCallbacks).toContain('onDisconnected');
            });

            it('should return default info for an unregistered element', () => {
                const scope = document.createElement('div');

                const info = window.__pikoDebug!.getPartialInfo(scope);

                expect(info.exists).toBe(false);
                expect(info.isConnected).toBe(false);
                expect(info.connectedOnce).toBe(false);
                expect(info.registeredCallbacks).toEqual([]);
                expect(info.cleanupCount).toBe(0);
            });

            it('should reflect connected state after executeConnected', () => {
                const scope = document.createElement('div');
                scope.setAttribute('partial', 'dbg-conn');

                _registerLifecycle(scope, { onConnected: vi.fn() });
                _executeConnected(scope);

                const info = window.__pikoDebug!.getPartialInfo(scope);
                expect(info.isConnected).toBe(true);
                expect(info.connectedOnce).toBe(true);
            });

            it('should use data-partial-name and data-partial attributes as fallback', () => {
                const scope = document.createElement('div');
                scope.setAttribute('data-partial', 'dp-id');
                scope.setAttribute('data-partial-name', 'DataPartial');

                _registerLifecycle(scope, { onConnected: vi.fn() });

                const info = window.__pikoDebug!.getPartialInfo(scope);
                expect(info.partialId).toBe('dp-id');
                expect(info.partialName).toBe('DataPartial');
            });

            it('should include element-scoped cleanups in cleanupCount', () => {
                const scope = document.createElement('div');
                scope.setAttribute('partial', 'dbg-clean');

                _registerLifecycle(scope, { onConnected: vi.fn() });
                onCleanup(vi.fn(), scope);
                onCleanup(vi.fn(), scope);

                const info = window.__pikoDebug!.getPartialInfo(scope);
                expect(info.cleanupCount).toBe(2);
            });
        });

        describe('isConnected()', () => {
            it('should return false for an unregistered element', () => {
                const scope = document.createElement('div');
                expect(window.__pikoDebug!.isConnected(scope)).toBe(false);
            });

            it('should return true after executeConnected', () => {
                const scope = document.createElement('div');
                scope.setAttribute('partial', 'ic-1');
                _registerLifecycle(scope, { onConnected: vi.fn() });
                _executeConnected(scope);

                expect(window.__pikoDebug!.isConnected(scope)).toBe(true);
            });

            it('should return false after executeDisconnected', () => {
                const scope = document.createElement('div');
                scope.setAttribute('partial', 'ic-2');
                _registerLifecycle(scope, { onConnected: vi.fn() });
                _executeConnected(scope);
                _executeDisconnected(scope);

                expect(window.__pikoDebug!.isConnected(scope)).toBe(false);
            });
        });

        describe('getCleanupCount()', () => {
            it('should return 0 for an unregistered element', () => {
                const scope = document.createElement('div');
                expect(window.__pikoDebug!.getCleanupCount(scope)).toBe(0);
            });

            it('should count element-scoped cleanups', () => {
                const scope = document.createElement('div');
                scope.setAttribute('partial', 'cc-1');

                _registerLifecycle(scope, { onConnected: vi.fn() });
                onCleanup(vi.fn(), scope);
                onCleanup(vi.fn(), scope);
                onCleanup(vi.fn(), scope);

                expect(window.__pikoDebug!.getCleanupCount(scope)).toBe(3);
            });
        });

        describe('getRegisteredCallbacks()', () => {
            it('should return empty array for an unregistered element', () => {
                const scope = document.createElement('div');
                expect(window.__pikoDebug!.getRegisteredCallbacks(scope)).toEqual([]);
            });

            it('should return names of all registered callbacks', () => {
                const scope = document.createElement('div');
                scope.setAttribute('partial', 'gcb-1');

                _registerLifecycle(scope, {
                    onConnected: vi.fn(),
                    onBeforeRender: vi.fn(),
                    onAfterRender: vi.fn(),
                    onUpdated: vi.fn(),
                });

                const names = window.__pikoDebug!.getRegisteredCallbacks(scope);
                expect(names).toContain('onConnected');
                expect(names).toContain('onBeforeRender');
                expect(names).toContain('onAfterRender');
                expect(names).toContain('onUpdated');
                expect(names).not.toContain('onDisconnected');
            });
        });

        describe('getAllConnectedPartials()', () => {
            it('should return only connected partials in the DOM', () => {
                const partial1 = document.createElement('div');
                partial1.setAttribute('partial', 'acp-1');
                testContainer.appendChild(partial1);

                const partial2 = document.createElement('div');
                partial2.setAttribute('partial', 'acp-2');
                testContainer.appendChild(partial2);

                const onConnected1 = vi.fn();
                const onConnected2 = vi.fn();

                _registerLifecycle(partial1, { onConnected: onConnected1 });
                _registerLifecycle(partial2, { onConnected: onConnected2 });

                const connected = window.__pikoDebug!.getAllConnectedPartials();
                expect(connected).toContain(partial1);
                expect(connected).toContain(partial2);
            });

            it('should not include disconnected partials', () => {
                const partial = document.createElement('div');
                partial.setAttribute('partial', 'acp-3');
                testContainer.appendChild(partial);

                _registerLifecycle(partial, { onConnected: vi.fn() });
                _executeDisconnected(partial);

                const connected = window.__pikoDebug!.getAllConnectedPartials();
                expect(connected).not.toContain(partial);
            });
        });
    });
});

describe('lifecycle (_addLifecycleCallback - additive registration)', () => {
    let testContainer: HTMLDivElement;

    beforeEach(() => {
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);
    });

    afterEach(() => {
        _disconnectCleanupObserver();
        _runPageCleanup();
        testContainer.remove();
        vi.clearAllMocks();
    });

    describe('_addLifecycleCallback()', () => {
        it('should register a lifecycle state for a new scope', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-1');

            _addLifecycleCallback(scope, 'onConnected', vi.fn());

            expect(_hasLifecycleCallbacks(scope)).toBe(true);
        });

        it('should fire multiple onConnected callbacks in order', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-2');
            const order: number[] = [];

            _addLifecycleCallback(scope, 'onConnected', () => order.push(1));
            _addLifecycleCallback(scope, 'onConnected', () => order.push(2));
            _addLifecycleCallback(scope, 'onConnected', () => order.push(3));

            _executeConnected(scope);

            expect(order).toEqual([1, 2, 3]);
        });

        it('should fire multiple onDisconnected callbacks in order', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-3');
            const order: number[] = [];

            _addLifecycleCallback(scope, 'onDisconnected', () => order.push(1));
            _addLifecycleCallback(scope, 'onDisconnected', () => order.push(2));

            _executeDisconnected(scope);

            expect(order).toEqual([1, 2]);
        });

        it('should fire multiple onBeforeRender callbacks in order', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-4');
            const order: number[] = [];

            _addLifecycleCallback(scope, 'onBeforeRender', () => order.push(1));
            _addLifecycleCallback(scope, 'onBeforeRender', () => order.push(2));

            _executeBeforeRender(scope);

            expect(order).toEqual([1, 2]);
        });

        it('should fire multiple onAfterRender callbacks in order', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-5');
            const order: number[] = [];

            _addLifecycleCallback(scope, 'onAfterRender', () => order.push(1));
            _addLifecycleCallback(scope, 'onAfterRender', () => order.push(2));

            _executeAfterRender(scope);

            expect(order).toEqual([1, 2]);
        });

        it('should fire multiple onUpdated callbacks with context in order', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-6');
            const received: unknown[] = [];

            _addLifecycleCallback(scope, 'onUpdated', (ctx?: unknown) => received.push(['a', ctx]));
            _addLifecycleCallback(scope, 'onUpdated', (ctx?: unknown) => received.push(['b', ctx]));

            const context = { reason: 'test' };
            _executeUpdated(scope, context);

            expect(received).toEqual([['a', context], ['b', context]]);
        });

        it('should auto-execute onConnected when element is already in the DOM', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-7');
            testContainer.appendChild(scope);

            const onConnected = vi.fn();
            _addLifecycleCallback(scope, 'onConnected', onConnected);

            expect(onConnected).toHaveBeenCalledTimes(1);
        });

        it('should not auto-execute onConnected when element is not in the DOM', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-8');

            const onConnected = vi.fn();
            _addLifecycleCallback(scope, 'onConnected', onConnected);

            expect(onConnected).not.toHaveBeenCalled();
        });

        it('should not auto-execute onConnected if already connected once', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-9');
            testContainer.appendChild(scope);

            const first = vi.fn();
            _addLifecycleCallback(scope, 'onConnected', first);
            expect(first).toHaveBeenCalledTimes(1);

            const second = vi.fn();
            _addLifecycleCallback(scope, 'onConnected', second);
            expect(second).not.toHaveBeenCalled();
        });

        it('should coexist with _registerLifecycle callbacks', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-10');
            const order: string[] = [];

            _registerLifecycle(scope, { onConnected: () => order.push('register') });
            _addLifecycleCallback(scope, 'onConnected', () => order.push('add'));

            _executeConnected(scope);

            expect(order).toEqual(['register', 'add']);
        });

        it('should not auto-execute for non-onConnected hooks', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-11');
            testContainer.appendChild(scope);

            const onDisconnected = vi.fn();
            _addLifecycleCallback(scope, 'onDisconnected', onDisconnected);

            expect(onDisconnected).not.toHaveBeenCalled();
        });

        it('should continue executing remaining callbacks when one throws', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-err');

            const good1 = vi.fn();
            const bad = vi.fn(() => { throw new Error('boom'); });
            const good2 = vi.fn();

            _addLifecycleCallback(scope, 'onConnected', good1);
            _addLifecycleCallback(scope, 'onConnected', bad);
            _addLifecycleCallback(scope, 'onConnected', good2);

            _executeConnected(scope);

            expect(good1).toHaveBeenCalledTimes(1);
            expect(bad).toHaveBeenCalledTimes(1);
            expect(good2).toHaveBeenCalledTimes(1);
            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in onConnected'),
                expect.any(Error),
            );
            errorSpy.mockRestore();
        });

        it('should report registered additive callbacks in debug API', () => {
            const scope = document.createElement('div');
            scope.setAttribute('partial', 'add-dbg');

            _addLifecycleCallback(scope, 'onConnected', vi.fn());
            _addLifecycleCallback(scope, 'onBeforeRender', vi.fn());

            const names = window.__pikoDebug!.getRegisteredCallbacks(scope);
            expect(names).toContain('onConnected');
            expect(names).toContain('onBeforeRender');
            expect(names).not.toContain('onDisconnected');
        });
    });
});
