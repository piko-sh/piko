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

/**
 * Key the reactivity layer uses to expose a proxy's underlying target.
 */
const REACTIVE_RAW = Symbol.for('piko.reactivity.rawTarget');

/** Simple event bus for cross-component communication. */
export const bus = {
    /**
     * Emits an event to all listeners.
     *
     * The payload is snapshotted once before dispatch: reactive proxies are
     * unwrapped and plain objects and arrays are deep-copied, so listeners
     * always receive plain, cloneable data rather than a live reference into
     * component state.
     *
     * @param event - Event name.
     * @param data - Optional data to pass to listeners.
     */
    emit(event: string, data?: unknown): void {
        const handlers = listeners.get(event);
        if (handlers) {
            let payload: unknown;
            try {
                payload = snapshot(data, new WeakMap());
            } catch (error) {
                console.error(`[pk] Error snapshotting payload for "${event}":`, error);
                payload = data;
            }
            handlers.forEach(fn => {
                try {
                    fn(payload);
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



/**
 * Returns the plain target behind a reactive proxy, or the value unchanged when
 * it is not reactive.
 *
 * @param value - The object that may be a reactive proxy.
 * @returns The underlying target when reactive, otherwise the value itself.
 */
function toRaw(value: object): object {
    const raw = (value as Record<symbol, unknown>)[REACTIVE_RAW];
    return raw === undefined ? value : (raw as object);
}

/**
 * Reports whether a value is a plain object (or a reactive proxy wrapping one)
 * that is safe to copy property by property.
 *
 * @param value - The object to inspect.
 * @returns True for plain objects, false for host objects and class instances.
 */
function isPlainObject(value: object): boolean {
    const prototype: unknown = Object.getPrototypeOf(value);
    return prototype === Object.prototype || prototype === null;
}

/**
 * Produces a plain, cloneable snapshot of an emitted payload.
 *
 * @param value - The value to snapshot.
 * @param seen - Map of already-copied raw targets, used to break cycles.
 * @returns A plain snapshot of the value.
 */
function snapshot(value: unknown, seen: WeakMap<object, unknown>): unknown {
    if (value === null || typeof value !== 'object') {
        return value;
    }

    const raw = toRaw(value);
    if (seen.has(raw)) {
        return seen.get(raw);
    }

    if (Array.isArray(raw)) {
        const copy: unknown[] = [];
        seen.set(raw, copy);
        for (let index = 0; index < raw.length; index++) {
            copy[index] = snapshot(raw[index], seen);
        }
        return copy;
    }

    if (raw instanceof Date) {
        return new Date(raw.getTime());
    }

    if (raw instanceof RegExp) {
        return new RegExp(raw.source, raw.flags);
    }

    if (raw instanceof Map) {
        const copy = new Map<unknown, unknown>();
        seen.set(raw, copy);
        for (const [entryKey, entryValue] of raw) {
            copy.set(snapshot(entryKey, seen), snapshot(entryValue, seen));
        }
        return copy;
    }

    if (raw instanceof Set) {
        const copy = new Set<unknown>();
        seen.set(raw, copy);
        for (const entryValue of raw) {
            copy.add(snapshot(entryValue, seen));
        }
        return copy;
    }

    if (isPlainObject(raw)) {
        const copy: Record<string, unknown> = {};
        seen.set(raw, copy);
        for (const key of Object.keys(raw)) {
            copy[key] = snapshot((raw as Record<string, unknown>)[key], seen);
        }
        return copy;
    }

    return raw;
}
