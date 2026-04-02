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

import {HookEvent, type HookManager} from './HookManager';

/** Callback function type for network status change events. */
export type NetworkStatusCallback = () => void;

/** Monitors network connectivity and provides a visual offline indicator. */
export interface NetworkStatus {
    /** Current online status. */
    readonly isOnline: boolean;

    /**
     * Subscribes to network status changes.
     * @param event - The network event to subscribe to.
     * @param callback - The callback to invoke on status change.
     * @returns An unsubscribe function.
     */
    on(event: 'online' | 'offline', callback: NetworkStatusCallback): () => void;

    /** Cleans up event listeners and removes the offline banner from the DOM. */
    destroy(): void;
}

/** Dependencies for creating a NetworkStatus service. */
export interface NetworkStatusDependencies {
    /** Hook manager for emitting network status events. */
    hookManager: HookManager;
}

/** DOM identifier for the offline status banner. */
const OFFLINE_BANNER_ID = 'ppf-offline-banner';

/**
 * Creates the offline indicator banner element with an assertive aria-live attribute.
 * @returns The created banner element.
 */
function createOfflineBanner(): HTMLElement {
    const banner = document.createElement('div');
    banner.id = OFFLINE_BANNER_ID;
    banner.setAttribute('role', 'alert');
    banner.setAttribute('aria-live', 'assertive');
    banner.textContent = 'You are offline. Some features may be unavailable.';
    banner.style.cssText = [
        'position:fixed',
        'top:0',
        'left:0',
        'right:0',
        'padding:8px 16px',
        'background:#f44336',
        'color:white',
        'text-align:center',
        'font-size:14px',
        'z-index:10000',
        'display:none',
    ].join(';');
    return banner;
}

/**
 * Creates a NetworkStatus service for monitoring network connectivity.
 * Displays a visual banner when the browser goes offline and emits hook events for analytics.
 * @param deps - The dependencies including the hook manager.
 * @returns A new NetworkStatus instance.
 */
export function createNetworkStatus(deps: NetworkStatusDependencies): NetworkStatus {
    const {hookManager} = deps;

    let online = navigator.onLine;
    const listeners = new Map<'online' | 'offline', Set<NetworkStatusCallback>>();
    listeners.set('online', new Set());
    listeners.set('offline', new Set());

    let banner = document.getElementById(OFFLINE_BANNER_ID);
    if (!banner) {
        banner = createOfflineBanner();
        document.body.appendChild(banner);
    }

    if (!online) {
        banner.style.display = 'block';
    }

    /**
     * Notifies all registered listeners for the given event type.
     * @param event - The network event type to notify for.
     */
    const notifyListeners = (event: 'online' | 'offline'): void => {
        const callbacks = listeners.get(event);
        if (!callbacks) {
            return;
        }
        for (const callback of callbacks) {
            try {
                callback();
            } catch (error) {
                console.error(`NetworkStatus: Error in ${event} listener:`, error);
            }
        }
    };

    /** Handles the browser online event by hiding the banner and emitting the hook. */
    const handleOnline = (): void => {
        online = true;
        banner.style.display = 'none';
        hookManager.emit(HookEvent.NETWORK_ONLINE, {timestamp: Date.now()});
        notifyListeners('online');
    };

    /** Handles the browser offline event by showing the banner and emitting the hook. */
    const handleOffline = (): void => {
        online = false;
        banner.style.display = 'block';
        hookManager.emit(HookEvent.NETWORK_OFFLINE, {timestamp: Date.now()});
        notifyListeners('offline');
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return {
        get isOnline(): boolean {
            return online;
        },

        on(event: 'online' | 'offline', callback: NetworkStatusCallback): () => void {
            const callbacks = listeners.get(event);
            if (callbacks) {
                callbacks.add(callback);
            }
            return () => {
                callbacks?.delete(callback);
            };
        },

        destroy(): void {
            window.removeEventListener('online', handleOnline);
            window.removeEventListener('offline', handleOffline);
            banner.remove();
            listeners.clear();
        }
    };
}
