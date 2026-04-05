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

import { describe, it, expect, vi } from 'vitest';
import { makeReactive, type ReactiveContext } from '@/reactivity/makeReactive';

describe('makeReactive', () => {
    describe('basic object reactivity', () => {
        it('should trigger scheduleRender on property set', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ count: 0 }, context);
            state.count = 1;

            expect(scheduleRender).toHaveBeenCalledTimes(1);
        });

        it('should track changed props', () => {
            const changedPropsSet = new Set<string>();
            const context: ReactiveContext = { changedPropsSet };

            const state = makeReactive({ name: 'initial' }, context);
            state.name = 'updated';

            expect(changedPropsSet.has('name')).toBe(true);
        });

        it('should not trigger render for same primitive value', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ count: 5 }, context);
            state.count = 5;

            expect(scheduleRender).not.toHaveBeenCalled();
        });

        it('should trigger render for same object reference (deep change detection)', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const obj = { nested: 'value' };
            const state = makeReactive({ data: obj }, context);
            state.data = obj;

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should handle multiple property changes', () => {
            const scheduleRender = vi.fn();
            const changedPropsSet = new Set<string>();
            const context: ReactiveContext = { scheduleRender, changedPropsSet };

            const state = makeReactive({ a: 1, b: 2, c: 3 }, context);
            state.a = 10;
            state.b = 20;

            expect(scheduleRender).toHaveBeenCalledTimes(2);
            expect(changedPropsSet.has('a')).toBe(true);
            expect(changedPropsSet.has('b')).toBe(true);
        });
    });

    describe('nested object reactivity', () => {
        it('should make nested objects reactive on access', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({
                user: { name: 'John', age: 30 }
            }, context);

            state.user.name = 'Jane';

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should track nested property changes', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({
                profile: { username: 'user1' }
            }, context);

            state.profile.username = 'user2';

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should handle deeply nested objects', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({
                level1: {
                    level2: {
                        level3: { value: 'deep' }
                    }
                }
            }, context);

            state.level1.level2.level3.value = 'changed';

            expect(scheduleRender).toHaveBeenCalled();
        });
    });

    describe('array reactivity', () => {
        it('should trigger render on array mutator methods - push', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ items: [1, 2, 3] }, context);
            state.items.push(4);

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should trigger render on array mutator methods - pop', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ items: [1, 2, 3] }, context);
            state.items.pop();

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should trigger render on array mutator methods - shift', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ items: [1, 2, 3] }, context);
            state.items.shift();

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should trigger render on array mutator methods - unshift', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ items: [1, 2, 3] }, context);
            state.items.unshift(0);

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should trigger render on array mutator methods - splice', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ items: [1, 2, 3] }, context);
            state.items.splice(1, 1, 99);

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should trigger render on array mutator methods - sort', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ items: [3, 1, 2] }, context);
            state.items.sort();

            expect(scheduleRender).toHaveBeenCalled();
            expect(state.items).toEqual([1, 2, 3]);
        });

        it('should trigger render on array mutator methods - reverse', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ items: [1, 2, 3] }, context);
            state.items.reverse();

            expect(scheduleRender).toHaveBeenCalled();
            expect(state.items).toEqual([3, 2, 1]);
        });

        it('should trigger render on direct index assignment', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ items: [1, 2, 3] }, context);
            state.items[0] = 100;

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should track array changes with parent key', () => {
            const changedPropsSet = new Set<string>();
            const context: ReactiveContext = { changedPropsSet };

            const state = makeReactive({ list: [1, 2, 3] }, context);
            state.list.push(4);

            expect(changedPropsSet.has('list')).toBe(true);
        });

        it('should make nested objects in array reactive', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({
                users: [{ name: 'Alice' }, { name: 'Bob' }]
            }, context);

            state.users[0].name = 'Alicia';

            expect(scheduleRender).toHaveBeenCalled();
        });
    });

    describe('edge cases', () => {
        it('should return non-objects as-is', () => {
            const result1 = makeReactive(null as unknown as object);
            const result2 = makeReactive(42 as unknown as object);
            const result3 = makeReactive('string' as unknown as object);

            expect(result1).toBe(null);
            expect(result2).toBe(42);
            expect(result3).toBe('string');
        });

        it('should not wrap DOM Nodes', () => {
            const div = document.createElement('div');
            const context: ReactiveContext = { scheduleRender: vi.fn() };

            const state = makeReactive({ element: div }, context);

            expect(state.element).toBe(div);
            expect(state.element.tagName).toBe('DIV');
        });

        it('should work without context', () => {
            const state = makeReactive({ count: 0 });
            state.count = 1;

            expect(state.count).toBe(1);
        });

        it('should work with context but no scheduleRender', () => {
            const changedPropsSet = new Set<string>();
            const context: ReactiveContext = { changedPropsSet };

            const state = makeReactive({ name: 'test' }, context);
            state.name = 'updated';

            expect(changedPropsSet.has('name')).toBe(true);
        });

        it('should work with context but no changedPropsSet', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const state = makeReactive({ value: 1 }, context);
            state.value = 2;

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should handle top-level arrays', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const arr = makeReactive([1, 2, 3], context, 'items');
            arr.push(4);

            expect(scheduleRender).toHaveBeenCalled();
        });

        it('should preserve array method return values', () => {
            const state = makeReactive({ items: [1, 2, 3] });

            const popped = state.items.pop();
            expect(popped).toBe(3);

            const shifted = state.items.shift();
            expect(shifted).toBe(1);

            const newLength = state.items.push(4, 5);
            expect(newLength).toBe(3);
        });

        it('should handle Symbol keys', () => {
            const scheduleRender = vi.fn();
            const context: ReactiveContext = { scheduleRender };

            const sym = Symbol('test');
            const state = makeReactive({ [sym]: 'value' } as Record<symbol, string>, context);
            state[sym] = 'new value';

            expect(scheduleRender).toHaveBeenCalled();
        });
    });
});
