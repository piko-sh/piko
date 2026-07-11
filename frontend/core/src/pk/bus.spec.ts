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

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { bus } from '@/pk/bus';

describe('bus (PK Event Bus)', () => {
    beforeEach(() => {
        bus.off();
    });

    describe('on()', () => {
        it('should register a listener and call it on emit', () => {
            const handler = vi.fn();
            bus.on('test-event', handler);

            bus.emit('test-event', { data: 'value' });

            expect(handler).toHaveBeenCalledTimes(1);
            expect(handler).toHaveBeenCalledWith({ data: 'value' });
        });

        it('should allow multiple listeners for same event', () => {
            const handler1 = vi.fn();
            const handler2 = vi.fn();

            bus.on('multi-event', handler1);
            bus.on('multi-event', handler2);

            bus.emit('multi-event', 'payload');

            expect(handler1).toHaveBeenCalledWith('payload');
            expect(handler2).toHaveBeenCalledWith('payload');
        });

        it('should return an unsubscribe function', () => {
            const handler = vi.fn();
            const unsubscribe = bus.on('unsub-event', handler);

            bus.emit('unsub-event');
            expect(handler).toHaveBeenCalledTimes(1);

            unsubscribe();

            bus.emit('unsub-event');
            expect(handler).toHaveBeenCalledTimes(1);
        });

        it('should handle unsubscribe when called multiple times', () => {
            const handler = vi.fn();
            const unsubscribe = bus.on('safe-unsub', handler);

            unsubscribe();
            unsubscribe();

            bus.emit('safe-unsub');
            expect(handler).not.toHaveBeenCalled();
        });
    });

    describe('emit()', () => {
        it('should do nothing when no listeners exist', () => {
            expect(() => bus.emit('nonexistent-event', 'data')).not.toThrow();
        });

        it('should pass undefined when no data provided', () => {
            const handler = vi.fn();
            bus.on('no-data-event', handler);

            bus.emit('no-data-event');

            expect(handler).toHaveBeenCalledWith(undefined);
        });

        it('should catch and log errors from handlers', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const errorHandler = vi.fn(() => {
                throw new Error('Handler error');
            });
            const goodHandler = vi.fn();

            bus.on('error-event', errorHandler);
            bus.on('error-event', goodHandler);

            bus.emit('error-event', 'data');

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in bus handler for "error-event"'),
                expect.any(Error)
            );
            expect(goodHandler).toHaveBeenCalledWith('data');

            errorSpy.mockRestore();
        });

        it('should pass complex data types', () => {
            const handler = vi.fn();
            bus.on('complex-event', handler);

            const complexData = {
                items: [1, 2, 3],
                nested: { value: 'test' },
                fn: () => {}
            };

            bus.emit('complex-event', complexData);

            expect(handler).toHaveBeenCalledWith(complexData);
        });
    });

    describe('once()', () => {
        it('should only call handler once', () => {
            const handler = vi.fn();
            bus.once('once-event', handler);

            bus.emit('once-event', 'first');
            bus.emit('once-event', 'second');
            bus.emit('once-event', 'third');

            expect(handler).toHaveBeenCalledTimes(1);
            expect(handler).toHaveBeenCalledWith('first');
        });

        it('should return unsubscribe function that works before emit', () => {
            const handler = vi.fn();
            const unsubscribe = bus.once('once-unsub', handler);

            unsubscribe();
            bus.emit('once-unsub', 'data');

            expect(handler).not.toHaveBeenCalled();
        });

        it('should auto-unsubscribe even if handler throws', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const handler = vi.fn(() => {
                throw new Error('Once handler error');
            });

            bus.once('once-error', handler);
            bus.emit('once-error');
            bus.emit('once-error');

            expect(handler).toHaveBeenCalledTimes(1);
            errorSpy.mockRestore();
        });
    });

    describe('off()', () => {
        it('should remove all listeners for a specific event', () => {
            const handler1 = vi.fn();
            const handler2 = vi.fn();

            bus.on('remove-event', handler1);
            bus.on('remove-event', handler2);

            bus.off('remove-event');

            bus.emit('remove-event');

            expect(handler1).not.toHaveBeenCalled();
            expect(handler2).not.toHaveBeenCalled();
        });

        it('should remove all listeners when no event specified', () => {
            const handler1 = vi.fn();
            const handler2 = vi.fn();

            bus.on('event-a', handler1);
            bus.on('event-b', handler2);

            bus.off();

            bus.emit('event-a');
            bus.emit('event-b');

            expect(handler1).not.toHaveBeenCalled();
            expect(handler2).not.toHaveBeenCalled();
        });

        it('should not throw when removing non-existent event', () => {
            expect(() => bus.off('nonexistent')).not.toThrow();
        });
    });

    describe('reactive payload snapshotting', () => {
        const REACTIVE_RAW = Symbol.for('piko.reactivity.rawTarget');

        function makeReactiveLike<T extends object>(target: T): T {
            return new Proxy(target, {
                get(proxyTarget, prop, receiver) {
                    if (prop === REACTIVE_RAW) {
                        return proxyTarget;
                    }
                    const value = Reflect.get(proxyTarget, prop, receiver) as unknown;
                    if (value !== null && typeof value === 'object') {
                        return makeReactiveLike(value as object);
                    }
                    return value;
                }
            }) as T;
        }

        it('should deliver reactive state to handlers as plain cloneable data', () => {
            const state = makeReactiveLike({
                config: { title: 'Piko', tags: ['a', 'b'], nested: { level: 2 } }
            });

            expect(() => structuredClone(state.config)).toThrow();

            let received: unknown;
            bus.on('reactive-event', data => {
                received = (data as { config: unknown }).config;
            });

            bus.emit('reactive-event', { config: state.config });

            expect(() => structuredClone(received)).not.toThrow();
            expect(received).toEqual({ title: 'Piko', tags: ['a', 'b'], nested: { level: 2 } });
        });

        it('should deliver a snapshot decoupled from later state mutations', () => {
            const state = makeReactiveLike({ config: { count: 1 } });

            let received: { count: number } | undefined;
            bus.on('snapshot-event', data => {
                received = (data as { config: { count: number } }).config;
            });

            bus.emit('snapshot-event', { config: state.config });
            state.config.count = 99;

            expect(received?.count).toBe(1);
        });

        it('should unwrap cyclic reactive state without overflowing the stack', () => {
            const raw: Record<string, unknown> = { name: 'root' };
            raw.self = raw;
            const state = makeReactiveLike(raw);

            let received: { node: { name: string; self: unknown } } | undefined;
            bus.on('cyclic-event', data => {
                received = data as { node: { name: string; self: unknown } };
            });

            expect(() => bus.emit('cyclic-event', { node: state })).not.toThrow();
            expect(received?.node.name).toBe('root');
            expect(received?.node.self).toBe(received?.node);
        });

        it('should surface a throwing handler without blocking siblings for a reactive payload', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const throwingHandler = vi.fn(() => {
                throw new Error('Handler error');
            });
            const siblingHandler = vi.fn();
            const state = makeReactiveLike({ config: { ok: true } });

            bus.on('reactive-error-event', throwingHandler);
            bus.on('reactive-error-event', siblingHandler);

            bus.emit('reactive-error-event', { config: state.config });

            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error in bus handler for "reactive-error-event"'),
                expect.any(Error)
            );
            expect(siblingHandler).toHaveBeenCalledTimes(1);
            expect(() => structuredClone(siblingHandler.mock.calls[0][0])).not.toThrow();

            errorSpy.mockRestore();
        });

        it('should leave host objects such as Blobs intact in the payload', () => {
            const blob = new Blob(['hello'], { type: 'text/plain' });

            let received: { file: Blob } | undefined;
            bus.on('host-object-event', data => {
                received = data as { file: Blob };
            });

            bus.emit('host-object-event', { file: blob });

            expect(received?.file).toBe(blob);
        });

        it('should still reach every handler when a payload getter throws', () => {
            const errorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const payload = {
                get boom(): never {
                    throw new Error('getter error');
                }
            };
            const handler = vi.fn();
            bus.on('throwing-getter-event', handler);

            expect(() => bus.emit('throwing-getter-event', payload)).not.toThrow();
            expect(handler).toHaveBeenCalledTimes(1);
            expect(errorSpy).toHaveBeenCalledWith(
                expect.stringContaining('Error snapshotting payload for "throwing-getter-event"'),
                expect.any(Error)
            );

            errorSpy.mockRestore();
        });

        it('should deep-copy Date and Map so a later mutation does not reach delivered payloads', () => {
            const date = new Date(2026, 0, 1);
            const map = new Map<string, number>([['a', 1]]);

            let received: { when: Date; counts: Map<string, number> } | undefined;
            bus.on('mutable-host-event', data => {
                received = data as { when: Date; counts: Map<string, number> };
            });

            bus.emit('mutable-host-event', { when: date, counts: map });

            date.setFullYear(1999);
            map.set('a', 99);

            expect(received?.when).not.toBe(date);
            expect(received?.when.getFullYear()).toBe(2026);
            expect(received?.counts).not.toBe(map);
            expect(received?.counts.get('a')).toBe(1);
        });
    });

    describe('integration', () => {
        it('should support multiple events independently', () => {
            const handlerA = vi.fn();
            const handlerB = vi.fn();

            bus.on('event-a', handlerA);
            bus.on('event-b', handlerB);

            bus.emit('event-a', 'a-data');

            expect(handlerA).toHaveBeenCalledWith('a-data');
            expect(handlerB).not.toHaveBeenCalled();
        });

        it('should handle rapid emit/subscribe cycles', () => {
            const results: number[] = [];

            for (let i = 0; i < 10; i++) {
                const unsubscribe = bus.on('rapid-event', (data) => {
                    results.push(data as number);
                });
                bus.emit('rapid-event', i);
                unsubscribe();
            }

            expect(results).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 8, 9]);
        });
    });
});
