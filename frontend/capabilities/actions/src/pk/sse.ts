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

/** Reloads a partial by name via the piko namespace (cross-capability). */
function reloadPartial(name: string): void {
    if (typeof window !== 'undefined' && window.piko?.partials) {
        void window.piko.partials.reload(name);
    }
}

/** Maximum backoff multiplier for reconnection delay. */
const MAX_RECONNECT_BACKOFF_MULTIPLIER = 5;

/** Configuration options for SSE subscriptions. */
export interface SSEOptions {
    /** SSE endpoint URL. */
    url: string;
    /** Custom message handler (overrides default partial reload). */
    onMessage?: (data: unknown) => void;
    /** Error handling: 'reconnect' (default) or 'stop'. */
    onError?: 'reconnect' | 'stop';
    /** Reconnection delay in ms (default: 3000). */
    reconnectDelay?: number;
    /** Maximum reconnection attempts (default: 10). */
    maxReconnects?: number;
    /** Event types to listen for (default: ['message']). */
    eventTypes?: string[];
    /** Called when connection opens. */
    onOpen?: () => void;
    /** Called when connection closes. */
    onClose?: () => void;
}

/** Handle for an active SSE subscription. */
export interface SSESubscription {
    /** Stops listening and closes the connection. */
    unsubscribe: () => void;
    /** Current connection state. */
    readonly state: 'connecting' | 'open' | 'closed' | 'error';
    /** Number of reconnection attempts. */
    readonly reconnectCount: number;
}

/** State for an SSE connection. */
interface SSEConnectionState {
    /** The underlying EventSource instance. */
    eventSource: EventSource | null;
    /** Number of reconnection attempts made. */
    reconnectCount: number;
    /** Pending reconnection timeout ID. */
    reconnectTimeout: ReturnType<typeof setTimeout> | null;
    /** Whether the connection has been stopped. */
    stopped: boolean;
}

/**
 * Parses SSE data, attempting JSON deserialisation.
 *
 * @param data - Raw data from the SSE message.
 * @returns Parsed JSON object, or the original string if parsing fails.
 */
function parseSSEData(data: unknown): unknown {
    if (typeof data === 'string') {
        try {
            return JSON.parse(data) as unknown;
        } catch {}
    }
    return data;
}

/**
 * Creates a message handler for SSE events.
 *
 * @param state - Connection state.
 * @param name - Partial name to reload on message.
 * @param onMessage - Optional custom message handler.
 * @returns Event handler function.
 */
function createMessageHandler(
    state: SSEConnectionState,
    name: string,
    onMessage?: (data: unknown) => void
): (event: MessageEvent) => void {
    return (event: MessageEvent): void => {
        if (state.stopped) {
            return;
        }
        const data = parseSSEData(event.data);
        if (onMessage) {
            onMessage(data);
        } else {
            void reloadPartial(name);
        }
    };
}

/**
 * Creates an error handler for SSE connections with reconnection logic.
 *
 * @param state - Connection state.
 * @param url - SSE endpoint URL.
 * @param onError - Error handling strategy.
 * @param reconnectDelay - Base reconnection delay in ms.
 * @param maxReconnects - Maximum reconnection attempts.
 * @param onClose - Optional close callback.
 * @param connect - Function to re-establish the connection.
 * @returns Error handler function.
 */
function createErrorHandler(
    state: SSEConnectionState,
    url: string,
    onError: 'reconnect' | 'stop',
    reconnectDelay: number,
    maxReconnects: number,
    onClose: (() => void) | undefined,
    connect: () => void
): () => void {
    return (): void => {
        if (state.stopped) {
            return;
        }

        state.eventSource?.close();
        state.eventSource = null;

        if (onError === 'stop') {
            state.stopped = true;
            console.warn(`[pk] SSE connection to "${url}" failed, stopping`);
            onClose?.();
            return;
        }

        if (state.reconnectCount < maxReconnects) {
            state.reconnectCount++;
            const delay = reconnectDelay * Math.min(state.reconnectCount, MAX_RECONNECT_BACKOFF_MULTIPLIER);
            console.warn(`[pk] SSE connection to "${url}" lost, reconnecting in ${delay}ms (attempt ${state.reconnectCount}/${maxReconnects})`);

            state.reconnectTimeout = setTimeout(() => {
                if (!state.stopped) {
                    connect();
                }
            }, delay);
        } else {
            state.stopped = true;
            console.error(`[pk] SSE connection to "${url}" failed after ${maxReconnects} attempts`);
            onClose?.();
        }
    };
}

/**
 * Subscribes to Server-Sent Events for a partial.
 *
 * When the server sends an event, the partial is automatically reloaded.
 * Alternatively, provide a custom onMessage handler to process events.
 *
 * @param name - Partial name to reload on message.
 * @param options - SSE configuration options.
 * @returns Cleanup function to stop listening.
 */
export function subscribeToUpdates(name: string, options: SSEOptions): () => void {
    const {
        url,
        onMessage,
        onError = 'reconnect',
        reconnectDelay = 3000,
        maxReconnects = 10,
        eventTypes = ['message'],
        onOpen,
        onClose
    } = options;

    const state: SSEConnectionState = {
        eventSource: null,
        reconnectCount: 0,
        reconnectTimeout: null,
        stopped: false
    };

    const handleMessage = createMessageHandler(state, name, onMessage);

    const connect = (): void => {
        if (state.stopped) {
            return;
        }

        try {
            state.eventSource = new EventSource(url);
            state.eventSource.onopen = () => {
                state.reconnectCount = 0;
                onOpen?.();
            };
            state.eventSource.onerror = createErrorHandler(
                state, url, onError, reconnectDelay, maxReconnects, onClose, connect
            );

            for (const eventType of eventTypes) {
                state.eventSource.addEventListener(eventType, handleMessage);
            }
        } catch (error) {
            console.error(`[pk] Failed to create SSE connection to "${url}":`, {error});
            state.eventSource = null;
        }
    };

    connect();

    return () => {
        state.stopped = true;
        if (state.reconnectTimeout) {
            clearTimeout(state.reconnectTimeout);
        }
        state.eventSource?.close();
        onClose?.();
    };
}

/**
 * Creates an SSE subscription with detailed state tracking.
 *
 * Use this when you need to monitor connection state or access reconnect count.
 *
 * @param name - Partial name to reload on message.
 * @param options - SSE configuration options.
 * @returns An SSESubscription with state tracking.
 */
export function createSSESubscription(name: string, options: SSEOptions): SSESubscription {
    let state: 'connecting' | 'open' | 'closed' | 'error' = 'connecting';
    let reconnectCount = 0;

    const unsubscribe = subscribeToUpdates(name, {
        ...options,
        onOpen: () => {
            state = 'open';
            reconnectCount = 0;
            options.onOpen?.();
        },
        onClose: () => {
            state = 'closed';
            options.onClose?.();
        },
        onError: options.onError
    });

    return {
        unsubscribe,
        get state() {
            return state;
        },
        get reconnectCount() {
            return reconnectCount;
        }
    };
}
