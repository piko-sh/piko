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
import {createNetworkStatus, type NetworkStatus} from './NetworkStatus';
import {createHookManager, HookEvent, type HookManager} from './HookManager';

describe('NetworkStatus', () => {
    let hookManager: HookManager;
    let networkStatus: NetworkStatus;
    let originalOnLine: boolean;

    beforeEach(() => {
        hookManager = createHookManager();
        originalOnLine = navigator.onLine;

        const existingBanner = document.getElementById('ppf-offline-banner');
        if (existingBanner) {
            existingBanner.remove();
        }
    });

    afterEach(() => {
        networkStatus?.destroy();
        Object.defineProperty(navigator, 'onLine', {
            value: originalOnLine,
            writable: true,
            configurable: true
        });

        const banner = document.getElementById('ppf-offline-banner');
        if (banner) {
            banner.remove();
        }
    });

    describe('createNetworkStatus', () => {
        it('should create a network status service', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            expect(networkStatus).toBeDefined();
            expect(typeof networkStatus.isOnline).toBe('boolean');
            expect(typeof networkStatus.on).toBe('function');
            expect(typeof networkStatus.destroy).toBe('function');
        });

        it('should reflect initial online status', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            expect(networkStatus.isOnline).toBe(true);
        });

        it('should reflect initial offline status', () => {
            Object.defineProperty(navigator, 'onLine', {value: false, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            expect(networkStatus.isOnline).toBe(false);
        });

        it('should create offline banner element', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const banner = document.getElementById('ppf-offline-banner');
            expect(banner).toBeTruthy();
            expect(banner?.getAttribute('role')).toBe('alert');
            expect(banner?.getAttribute('aria-live')).toBe('assertive');
        });

        it('should hide banner when online', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const banner = document.getElementById('ppf-offline-banner');
            expect(banner?.style.display).toBe('none');
        });

        it('should show banner when offline', () => {
            Object.defineProperty(navigator, 'onLine', {value: false, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const banner = document.getElementById('ppf-offline-banner');
            expect(banner?.style.display).toBe('block');
        });

        it('should reuse existing banner if present', () => {
            const existingBanner = document.createElement('div');
            existingBanner.id = 'ppf-offline-banner';
            document.body.appendChild(existingBanner);

            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const banners = document.querySelectorAll('#ppf-offline-banner');
            expect(banners.length).toBe(1);
        });
    });

    describe('online/offline events', () => {
        it('should update isOnline when going offline', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            expect(networkStatus.isOnline).toBe(true);

            window.dispatchEvent(new Event('offline'));

            expect(networkStatus.isOnline).toBe(false);
        });

        it('should update isOnline when going online', () => {
            Object.defineProperty(navigator, 'onLine', {value: false, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            expect(networkStatus.isOnline).toBe(false);

            window.dispatchEvent(new Event('online'));

            expect(networkStatus.isOnline).toBe(true);
        });

        it('should show banner when going offline', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const banner = document.getElementById('ppf-offline-banner');
            expect(banner?.style.display).toBe('none');

            window.dispatchEvent(new Event('offline'));

            expect(banner?.style.display).toBe('block');
        });

        it('should hide banner when going online', () => {
            Object.defineProperty(navigator, 'onLine', {value: false, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const banner = document.getElementById('ppf-offline-banner');
            expect(banner?.style.display).toBe('block');

            window.dispatchEvent(new Event('online'));

            expect(banner?.style.display).toBe('none');
        });

        it('should emit NETWORK_OFFLINE hook when going offline', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const callback = vi.fn();
            hookManager.api.on(HookEvent.NETWORK_OFFLINE, callback);

            window.dispatchEvent(new Event('offline'));

            expect(callback).toHaveBeenCalledTimes(1);
            expect(callback).toHaveBeenCalledWith(expect.objectContaining({
                timestamp: expect.any(Number)
            }));
        });

        it('should emit NETWORK_ONLINE hook when going online', () => {
            Object.defineProperty(navigator, 'onLine', {value: false, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const callback = vi.fn();
            hookManager.api.on(HookEvent.NETWORK_ONLINE, callback);

            window.dispatchEvent(new Event('online'));

            expect(callback).toHaveBeenCalledTimes(1);
            expect(callback).toHaveBeenCalledWith(expect.objectContaining({
                timestamp: expect.any(Number)
            }));
        });
    });

    describe('on (subscribe)', () => {
        it('should call online listeners when going online', () => {
            Object.defineProperty(navigator, 'onLine', {value: false, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const callback = vi.fn();
            networkStatus.on('online', callback);

            window.dispatchEvent(new Event('online'));

            expect(callback).toHaveBeenCalledTimes(1);
        });

        it('should call offline listeners when going offline', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const callback = vi.fn();
            networkStatus.on('offline', callback);

            window.dispatchEvent(new Event('offline'));

            expect(callback).toHaveBeenCalledTimes(1);
        });

        it('should return unsubscribe function', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const callback = vi.fn();
            const unsubscribe = networkStatus.on('offline', callback);

            unsubscribe();

            window.dispatchEvent(new Event('offline'));

            expect(callback).not.toHaveBeenCalled();
        });

        it('should allow multiple listeners', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const callback1 = vi.fn();
            const callback2 = vi.fn();

            networkStatus.on('offline', callback1);
            networkStatus.on('offline', callback2);

            window.dispatchEvent(new Event('offline'));

            expect(callback1).toHaveBeenCalledTimes(1);
            expect(callback2).toHaveBeenCalledTimes(1);
        });

        it('should catch and log errors in listeners', () => {
            const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const errorCallback = vi.fn(() => {
                throw new Error('Listener error');
            });
            const normalCallback = vi.fn();

            networkStatus.on('offline', errorCallback);
            networkStatus.on('offline', normalCallback);

            window.dispatchEvent(new Event('offline'));

            expect(errorCallback).toHaveBeenCalled();
            expect(normalCallback).toHaveBeenCalled();
            expect(consoleErrorSpy).toHaveBeenCalled();

            consoleErrorSpy.mockRestore();
        });
    });

    describe('destroy', () => {
        it('should remove event listeners', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const callback = vi.fn();
            networkStatus.on('offline', callback);

            networkStatus.destroy();

            window.dispatchEvent(new Event('offline'));

            expect(callback).not.toHaveBeenCalled();
        });

        it('should remove banner from DOM', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            expect(document.getElementById('ppf-offline-banner')).toBeTruthy();

            networkStatus.destroy();

            expect(document.getElementById('ppf-offline-banner')).toBeNull();
        });

        it('should clear listeners', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const callback = vi.fn();
            networkStatus.on('offline', callback);

            networkStatus.destroy();

            networkStatus = createNetworkStatus({hookManager});

            window.dispatchEvent(new Event('offline'));

            expect(callback).not.toHaveBeenCalled();
        });
    });

    describe('banner styling', () => {
        it('should have correct styling for visibility', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const banner = document.getElementById('ppf-offline-banner');
            expect(banner?.style.position).toBe('fixed');
            expect(banner?.style.top).toBe('0px');
            expect(banner?.style.zIndex).toBe('10000');
        });

        it('should have alert text content', () => {
            Object.defineProperty(navigator, 'onLine', {value: true, writable: true, configurable: true});
            networkStatus = createNetworkStatus({hookManager});

            const banner = document.getElementById('ppf-offline-banner');
            expect(banner?.textContent).toContain('offline');
        });
    });
});
