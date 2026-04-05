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

/** Array methods that mutate the array and should trigger reactivity. */
const arrayMutatorMethods = ['push', 'pop', 'shift', 'unshift', 'splice', 'sort', 'reverse'];

/** Object type that can be used as a proxy target. */
type ReactiveTarget = Record<string | symbol, unknown>;

/**
 * Context for reactive proxies to communicate with the component.
 *
 * Provides the hooks that a reactive proxy uses to notify the owning component
 * of state mutations.
 */
export interface ReactiveContext {
    /** Callback to schedule a re-render when state changes. */
    scheduleRender?: () => void;
    /** Set of property names that have changed since the last render. */
    changedPropsSet?: Set<string>;
}

/**
 * Makes an object reactive by wrapping it in an ES Proxy.
 *
 * Primitive values, `null`, and DOM {@link Node} instances are returned
 * unchanged because they either cannot be proxied or because their native
 * methods break when accessed through a proxy. Arrays are wrapped with
 * mutator-method interception via {@link createArrayProxy}; plain objects
 * are wrapped via {@link createObjectProxy}.
 *
 * @param target - The object to make reactive.
 * @param context - The reactive context for render scheduling.
 * @param parentProp - The parent property name for tracking nested changes.
 * @returns A reactive proxy wrapping the object, or the original value if it cannot be proxied.
 */
export function makeReactive<T extends object>(
    target: T | null,
    context?: ReactiveContext,
    parentProp?: string
): T {
    if (typeof target !== 'object' || target === null) {
        return target as unknown as T;
    }

    if (target instanceof Node) {
        return target;
    }

    if (Array.isArray(target)) {
        return createArrayProxy(target, context, parentProp) as T;
    }

    return createObjectProxy(target, context) as T;
}

/**
 * Creates a reactive proxy for an array.
 *
 * Intercepts the standard array mutator methods (push, pop, splice, etc.) so
 * that each mutation records the owning property in the changed-props set and
 * schedules a re-render. Nested objects and arrays retrieved via the `get` trap
 * are recursively wrapped, excluding DOM {@link Node} instances.
 *
 * @param arr - The source array to proxy.
 * @param context - The reactive context for render scheduling.
 * @param parentProp - The property name of the array on the parent object.
 * @returns A proxied copy of the array with reactive traps.
 */
function createArrayProxy<T extends unknown[]>(
    arr: T,
    context: ReactiveContext | undefined,
    parentProp: string | undefined
): T {
    return new Proxy(arr, {
        get(target, prop, receiver) {
            const value = Reflect.get(target, prop, receiver) as unknown;
            if (typeof prop === 'string' && arrayMutatorMethods.includes(prop) && typeof value === 'function') {
                return function (...args: unknown[]) {
                    const result: unknown = (value as (...args: unknown[]) => unknown).apply(target, args);
                    if (context?.changedPropsSet && parentProp) {
                        context.changedPropsSet.add(parentProp);
                    }
                    if (context?.scheduleRender) {
                        context.scheduleRender();
                    }
                    return result;
                };
            }
            if (typeof value === 'object' && value !== null && !(value instanceof Node)) {
                return makeReactive(value as object, context, parentProp);
            }
            return value;
        },
        set(target, prop, value) {
            (target as ReactiveTarget)[prop] = value as unknown;
            if (context?.changedPropsSet && parentProp) {
                context.changedPropsSet.add(parentProp);
            }
            if (context?.scheduleRender) {
                context.scheduleRender();
            }
            return true;
        }
    }) as T;
}

/**
 * Creates a reactive proxy for a plain object.
 *
 * The `get` trap recursively wraps nested objects and arrays, excluding DOM
 * {@link Node} instances. The `set` trap performs an equality short-circuit
 * for primitive values to avoid unnecessary renders, then records the changed
 * property and schedules a re-render through the provided context.
 *
 * @param target - The source object to proxy.
 * @param context - The reactive context for render scheduling.
 * @returns A proxied copy of the object with reactive traps.
 */
function createObjectProxy<T extends object>(
    target: T,
    context: ReactiveContext | undefined
): T {
    return new Proxy(target, {
        get(proxyTarget, prop, receiver) {
            const value = Reflect.get(proxyTarget, prop, receiver) as unknown;
            if (typeof value === 'object' && value !== null && !(value instanceof Node)) {
                const propKey = typeof prop === 'string' ? prop : String(prop);
                return makeReactive(value as object, context, propKey);
            }
            return value;
        },
        set(proxyTarget, prop, value) {
            const oldVal = (proxyTarget as ReactiveTarget)[prop];
            if (oldVal === value && typeof value !== "object") {
                return true;
            }
            (proxyTarget as ReactiveTarget)[prop] = value as unknown;
            if (context?.changedPropsSet && typeof prop === "string") {
                context.changedPropsSet.add(prop);
            }
            if (context?.scheduleRender) {
                context.scheduleRender();
            }
            return true;
        },
    }) as T;
}
