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

/** Handler function type for bus events. */
type EventHandler = (data: unknown) => void;

/** Registry of event listeners keyed by event name. */
const listeners = new Map<string, Set<EventHandler>>();

/** Simple event bus for cross-component communication. */
export const bus = {
    /**
     * Emits an event to all listeners.
     *
     * @param event - Event name.
     * @param data - Optional data to pass to listeners.
     */
    emit(event: string, data?: unknown): void {
        const handlers = listeners.get(event);
        if (handlers) {
            handlers.forEach(fn => {
                try {
                    fn(data);
                } catch (error) {
                    console.error(`[pk] Error in bus handler for "${event}":`, error);
                }
            });
        }
    },

    /**
     * Subscribes to an event.
     *
     * @param event - Event name.
     * @param handler - Handler function.
     * @returns Unsubscribe function.
     */
    on(event: string, handler: EventHandler): () => void {
        let eventListeners = listeners.get(event);
        if (!eventListeners) {
            eventListeners = new Set();
            listeners.set(event, eventListeners);
        }
        eventListeners.add(handler);

        return () => {
            listeners.get(event)?.delete(handler);
        };
    },

    /**
     * Subscribes to an event once (auto-unsubscribes after first call).
     *
     * @param event - Event name.
     * @param handler - Handler function.
     * @returns Unsubscribe function (in case you want to cancel before it fires).
     */
    once(event: string, handler: EventHandler): () => void {
        const wrappedHandler = (data: unknown) => {
            listeners.get(event)?.delete(wrappedHandler);
            handler(data);
        };
        return this.on(event, wrappedHandler);
    },

    /**
     * Removes all listeners for an event, or all listeners if no event specified.
     *
     * @param event - Optional event name.
     */
    off(event?: string): void {
        if (event) {
            listeners.delete(event);
        } else {
            listeners.clear();
        }
    }
};
