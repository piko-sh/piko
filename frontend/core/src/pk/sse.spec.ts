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
import {subscribeToUpdates, createSSESubscription} from '@/pk/sse';

vi.mock('./coordination', () => ({
    reloadPartial: vi.fn().mockResolvedValue(undefined)
}));

import {reloadPartial} from './coordination';

let lastMockInstance: MockEventSource | null = null;

class MockEventSource {
    url: string;
    onopen: ((event: Event) => void) | null = null;
    onerror: (() => void) | null = null;
    onmessage: ((event: MessageEvent) => void) | null = null;
    readyState: number = 0;
    close = vi.fn();

    private listeners: Map<string, ((event: MessageEvent) => void)[]> = new Map();

    constructor(url: string) {
        this.url = url;
        lastMockInstance = this;
    }

    addEventListener(type: string, handler: (event: MessageEvent) => void): void {
        const existing = this.listeners.get(type) ?? [];
        existing.push(handler);
        this.listeners.set(type, existing);
    }

    triggerOpen(): void {
        this.readyState = 1;
        if (this.onopen) {
            this.onopen(new Event('open'));
        }
    }

    triggerError(): void {
        this.readyState = 2;
        if (this.onerror) {
            this.onerror();
        }
    }

    triggerMessage(data: string, eventType: string = 'message'): void {
        const event = new MessageEvent(eventType, {data});
        const handlers = this.listeners.get(eventType) ?? [];
        for (const handler of handlers) {
            handler(event);
        }
        if (this.onmessage) {
            this.onmessage(event);
        }
    }

    triggerTypedEvent(eventType: string, data: string): void {
        const event = new MessageEvent(eventType, {data});
        const handlers = this.listeners.get(eventType) ?? [];
        for (const handler of handlers) {
            handler(event);
        }
    }
}

describe('sse (Server-Sent Events)', () => {
    beforeEach(() => {
        lastMockInstance = null;
        vi.useFakeTimers();
        vi.stubGlobal('EventSource', MockEventSource);
        vi.spyOn(console, 'warn').mockImplementation(() => {});
        vi.spyOn(console, 'error').mockImplementation(() => {});
        vi.mocked(reloadPartial).mockClear();
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.restoreAllMocks();
        vi.unstubAllGlobals();
    });

    describe('subscribeToUpdates', () => {
        describe('connection lifecycle', () => {
            it('should create an EventSource with the provided URL', () => {
                subscribeToUpdates('my-partial', {url: '/events'});

                expect(lastMockInstance).not.toBeNull();
                expect(lastMockInstance!.url).toBe('/events');
            });

            it('should fire onOpen callback when connection opens', () => {
                const onOpen = vi.fn();
                subscribeToUpdates('my-partial', {url: '/events', onOpen});

                lastMockInstance!.triggerOpen();

                expect(onOpen).toHaveBeenCalledOnce();
            });

            it('should call reloadPartial with the partial name when a message is received', () => {
                subscribeToUpdates('my-partial', {url: '/events'});
                lastMockInstance!.triggerOpen();

                lastMockInstance!.triggerMessage('{}');

                expect(reloadPartial).toHaveBeenCalledWith('my-partial');
            });

            it('should call reloadPartial when message data is a JSON string', () => {
                subscribeToUpdates('dashboard', {url: '/events'});
                lastMockInstance!.triggerOpen();

                lastMockInstance!.triggerMessage('{"action":"update"}');

                expect(reloadPartial).toHaveBeenCalledWith('dashboard');
            });

            it('should call custom onMessage handler instead of reloadPartial when provided', () => {
                const onMessage = vi.fn();
                subscribeToUpdates('my-partial', {url: '/events', onMessage});
                lastMockInstance!.triggerOpen();

                lastMockInstance!.triggerMessage('{"key":"value"}');

                expect(onMessage).toHaveBeenCalledWith({key: 'value'});
                expect(reloadPartial).not.toHaveBeenCalled();
            });

            it('should pass raw string to onMessage when data is not valid JSON', () => {
                const onMessage = vi.fn();
                subscribeToUpdates('my-partial', {url: '/events', onMessage});
                lastMockInstance!.triggerOpen();

                lastMockInstance!.triggerMessage('plain text');

                expect(onMessage).toHaveBeenCalledWith('plain text');
            });

            it('should register handlers for custom event types', () => {
                const onMessage = vi.fn();
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    onMessage,
                    eventTypes: ['update', 'refresh']
                });
                lastMockInstance!.triggerOpen();

                lastMockInstance!.triggerTypedEvent('update', '{"type":"update"}');

                expect(onMessage).toHaveBeenCalledWith({type: 'update'});
            });

            it('should register handlers for multiple custom event types', () => {
                const onMessage = vi.fn();
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    onMessage,
                    eventTypes: ['update', 'refresh']
                });
                lastMockInstance!.triggerOpen();

                lastMockInstance!.triggerTypedEvent('update', '"first"');
                lastMockInstance!.triggerTypedEvent('refresh', '"second"');

                expect(onMessage).toHaveBeenCalledTimes(2);
                expect(onMessage).toHaveBeenCalledWith('first');
                expect(onMessage).toHaveBeenCalledWith('second');
            });
        });

        describe('error handling', () => {
            it('should stop and fire onClose when onError is "stop"', () => {
                const onClose = vi.fn();
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    onError: 'stop',
                    onClose
                });
                lastMockInstance!.triggerOpen();

                lastMockInstance!.triggerError();

                expect(onClose).toHaveBeenCalledOnce();
                expect(lastMockInstance!.close).toHaveBeenCalled();
            });

            it('should not attempt reconnection when onError is "stop"', () => {
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    onError: 'stop'
                });
                const firstInstance = lastMockInstance!;
                firstInstance.triggerError();

                vi.advanceTimersByTime(10000);

                expect(lastMockInstance).toBe(firstInstance);
            });

            it('should attempt reconnection by default (onError: "reconnect")', () => {
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    reconnectDelay: 1000
                });
                const firstInstance = lastMockInstance!;
                firstInstance.triggerError();

                vi.advanceTimersByTime(1000);

                expect(lastMockInstance).not.toBe(firstInstance);
                expect(lastMockInstance!.url).toBe('/events');
            });

            it('should apply exponential backoff to reconnection delay', () => {
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    reconnectDelay: 1000,
                    maxReconnects: 5
                });

                const firstInstance = lastMockInstance!;
                firstInstance.triggerError();

                vi.advanceTimersByTime(999);
                expect(lastMockInstance).toBe(firstInstance);
                vi.advanceTimersByTime(1);
                const secondInstance = lastMockInstance!;
                expect(secondInstance).not.toBe(firstInstance);

                secondInstance.triggerError();
                vi.advanceTimersByTime(1999);
                expect(lastMockInstance).toBe(secondInstance);
                vi.advanceTimersByTime(1);
                expect(lastMockInstance).not.toBe(secondInstance);
            });

            it('should cap backoff multiplier at MAX_RECONNECT_BACKOFF_MULTIPLIER (5)', () => {
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    reconnectDelay: 1000,
                    maxReconnects: 10
                });

                for (let i = 0; i < 6; i++) {
                    lastMockInstance!.triggerError();
                    const delay = 1000 * Math.min(i + 1, 5);
                    vi.advanceTimersByTime(delay);
                }

                const instanceBeforeError = lastMockInstance!;
                instanceBeforeError.triggerError();
                vi.advanceTimersByTime(4999);
                expect(lastMockInstance).toBe(instanceBeforeError);
                vi.advanceTimersByTime(1);
                expect(lastMockInstance).not.toBe(instanceBeforeError);
            });

            it('should stop and log error when max reconnections exceeded', () => {
                const onClose = vi.fn();
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    reconnectDelay: 100,
                    maxReconnects: 2,
                    onClose
                });

                lastMockInstance!.triggerError();
                vi.advanceTimersByTime(100);

                lastMockInstance!.triggerError();
                vi.advanceTimersByTime(200);

                lastMockInstance!.triggerError();

                expect(console.error).toHaveBeenCalledWith(
                    expect.stringContaining('failed after 2 attempts')
                );
                expect(onClose).toHaveBeenCalledOnce();
            });

            it('should reset reconnection count on successful open after reconnect', () => {
                const onClose = vi.fn();
                subscribeToUpdates('my-partial', {
                    url: '/events',
                    reconnectDelay: 100,
                    maxReconnects: 2,
                    onClose
                });

                lastMockInstance!.triggerError();
                vi.advanceTimersByTime(100);

                lastMockInstance!.triggerOpen();

                lastMockInstance!.triggerError();
                vi.advanceTimersByTime(100);
                lastMockInstance!.triggerError();
                vi.advanceTimersByTime(200);

                lastMockInstance!.triggerError();

                expect(onClose).toHaveBeenCalledOnce();
            });

            it('should not reconnect if stopped before timeout fires', () => {
                const unsubscribe = subscribeToUpdates('my-partial', {
                    url: '/events',
                    reconnectDelay: 5000
                });
                const instanceAfterError = lastMockInstance!;
                instanceAfterError.triggerError();

                unsubscribe();
                vi.advanceTimersByTime(5000);

                expect(lastMockInstance).toBe(instanceAfterError);
            });
        });

        describe('cleanup', () => {
            it('should close EventSource when unsubscribe is called', () => {
                const unsubscribe = subscribeToUpdates('my-partial', {url: '/events'});
                const instance = lastMockInstance!;

                unsubscribe();

                expect(instance.close).toHaveBeenCalled();
            });

            it('should fire onClose callback when unsubscribe is called', () => {
                const onClose = vi.fn();
                const unsubscribe = subscribeToUpdates('my-partial', {
                    url: '/events',
                    onClose
                });

                unsubscribe();

                expect(onClose).toHaveBeenCalledOnce();
            });

            it('should clear pending reconnection timeout on unsubscribe', () => {
                const unsubscribe = subscribeToUpdates('my-partial', {
                    url: '/events',
                    reconnectDelay: 5000
                });
                lastMockInstance!.triggerError();

                const instanceAfterError = lastMockInstance!;
                unsubscribe();

                vi.advanceTimersByTime(10000);

                expect(lastMockInstance).toBe(instanceAfterError);
            });

            it('should not process messages after unsubscribe', () => {
                const onMessage = vi.fn();
                const unsubscribe = subscribeToUpdates('my-partial', {
                    url: '/events',
                    onMessage
                });
                const instance = lastMockInstance!;
                instance.triggerOpen();

                unsubscribe();

                instance.triggerMessage('{"data":"test"}');

                expect(onMessage).not.toHaveBeenCalled();
            });

            it('should not reconnect after unsubscribe even if error fires', () => {
                const unsubscribe = subscribeToUpdates('my-partial', {
                    url: '/events',
                    reconnectDelay: 100
                });
                const instance = lastMockInstance!;

                unsubscribe();

                instance.triggerError();

                vi.advanceTimersByTime(1000);
                expect(lastMockInstance).toBe(instance);
            });
        });

        describe('race conditions', () => {
            it('should handle rapid subscribe/unsubscribe cleanly', () => {
                const onClose = vi.fn();
                const unsubscribe = subscribeToUpdates('my-partial', {
                    url: '/events',
                    onClose
                });
                const instance = lastMockInstance!;

                unsubscribe();

                expect(instance.close).toHaveBeenCalled();
                expect(onClose).toHaveBeenCalledOnce();
            });

            it('should ignore messages received while in stopped state', () => {
                const unsubscribe = subscribeToUpdates('my-partial', {url: '/events'});
                const instance = lastMockInstance!;
                instance.triggerOpen();

                unsubscribe();

                instance.triggerMessage('{"should":"ignore"}');

                expect(reloadPartial).not.toHaveBeenCalled();
            });

            it('should ignore errors received while in stopped state', () => {
                const unsubscribe = subscribeToUpdates('my-partial', {
                    url: '/events',
                    onError: 'stop'
                });
                const instance = lastMockInstance!;
                instance.triggerOpen();

                unsubscribe();

                instance.triggerError();

                expect(console.warn).not.toHaveBeenCalledWith(
                    expect.stringContaining('failed, stopping')
                );
            });
        });

        describe('EventSource constructor failure', () => {
            it('should handle EventSource constructor throwing an error', () => {
                vi.stubGlobal('EventSource', class {
                    constructor() {
                        throw new Error('EventSource not supported');
                    }
                });

                const unsubscribe = subscribeToUpdates('my-partial', {url: '/events'});

                expect(console.error).toHaveBeenCalledWith(
                    expect.stringContaining('Failed to create SSE connection'),
                    expect.any(Object)
                );

                unsubscribe();
            });
        });
    });

    describe('createSSESubscription', () => {
        it('should start in "connecting" state', () => {
            const sub = createSSESubscription('my-partial', {url: '/events'});

            expect(sub.state).toBe('connecting');

            sub.unsubscribe();
        });

        it('should transition to "open" state when connection opens', () => {
            const sub = createSSESubscription('my-partial', {url: '/events'});

            lastMockInstance!.triggerOpen();

            expect(sub.state).toBe('open');

            sub.unsubscribe();
        });

        it('should transition to "closed" state when unsubscribed', () => {
            const sub = createSSESubscription('my-partial', {url: '/events'});
            lastMockInstance!.triggerOpen();

            sub.unsubscribe();

            expect(sub.state).toBe('closed');
        });

        it('should transition to "closed" state when onError is "stop" and error occurs', () => {
            const sub = createSSESubscription('my-partial', {
                url: '/events',
                onError: 'stop'
            });

            lastMockInstance!.triggerError();

            expect(sub.state).toBe('closed');

            sub.unsubscribe();
        });

        it('should expose reconnectCount as 0 initially', () => {
            const sub = createSSESubscription('my-partial', {url: '/events'});

            expect(sub.reconnectCount).toBe(0);

            sub.unsubscribe();
        });

        it('should delegate onOpen callback from options', () => {
            const onOpen = vi.fn();
            const sub = createSSESubscription('my-partial', {
                url: '/events',
                onOpen
            });

            lastMockInstance!.triggerOpen();

            expect(onOpen).toHaveBeenCalledOnce();

            sub.unsubscribe();
        });

        it('should delegate onClose callback from options', () => {
            const onClose = vi.fn();
            const sub = createSSESubscription('my-partial', {
                url: '/events',
                onClose
            });

            sub.unsubscribe();

            expect(onClose).toHaveBeenCalledOnce();
        });

        it('should reset state to "open" and reconnectCount on successful reconnection', () => {
            const sub = createSSESubscription('my-partial', {
                url: '/events',
                reconnectDelay: 100,
                maxReconnects: 5
            });
            lastMockInstance!.triggerOpen();
            expect(sub.state).toBe('open');

            lastMockInstance!.triggerError();
            vi.advanceTimersByTime(100);

            lastMockInstance!.triggerOpen();
            expect(sub.state).toBe('open');
            expect(sub.reconnectCount).toBe(0);

            sub.unsubscribe();
        });
    });
});
