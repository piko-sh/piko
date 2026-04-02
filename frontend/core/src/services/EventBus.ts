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

/** Callback function type for event bus subscribers. */
export type EventCallback = (data: unknown) => void;

/** Provides a publish/subscribe event system for decoupled communication between components. */
export interface EventBus {
    /**
     * Subscribes to an event.
     * @param event - The event name to subscribe to.
     * @param callback - The callback to invoke when the event fires.
     * @returns An unsubscribe function.
     */
    on(event: string, callback: EventCallback): () => void;

    /**
     * Unsubscribes a callback from an event.
     * @param event - The event name to unsubscribe from.
     * @param callback - The callback to remove.
     */
    off(event: string, callback: EventCallback): void;

    /**
     * Emits an event to all subscribers.
     * @param event - The event name to emit.
     * @param data - An optional data payload to pass to subscribers.
     */
    emit(event: string, data?: unknown): void;

    /**
     * Clears all listeners, or only listeners for a specific event.
     * @param event - An optional event name to clear listeners for.
     */
    clear(event?: string): void;
}

/**
 * Creates a new EventBus instance for publish/subscribe messaging.
 * @returns A new EventBus instance.
 */
export function createEventBus(): EventBus {
    const listeners = new Map<string, Set<EventCallback>>();

    return {
        on(event, callback) {
            if (!listeners.has(event)) {
                listeners.set(event, new Set());
            }
            listeners.get(event)?.add(callback);

            return () => this.off(event, callback);
        },

        off(event, callback) {
            listeners.get(event)?.delete(callback);
        },

        emit(event, data) {
            const eventListeners = listeners.get(event);
            if (!eventListeners) {
                return;
            }

            eventListeners.forEach(cb => {
                try {
                    cb(data);
                } catch (e) {
                    console.error(`EventBus: Error in listener for '${event}':`, e);
                }
            });
        },

        clear(event) {
            if (event) {
                listeners.delete(event);
            } else {
                listeners.clear();
            }
        }
    };
}
