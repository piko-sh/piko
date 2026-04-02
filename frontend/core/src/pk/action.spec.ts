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
import {
    action,
    isActionDescriptor,
    createActionError,
    createActionBuilder,
    registerActionFunction,
    getActionFunction,
    batch,
    ActionBuilder,
    type ActionDescriptor,
    type RetryStreamConfig
} from '@/pk/action';

vi.mock('@/core/ActionExecutor', () => ({
    callServerActionDirect: vi.fn()
}));

describe('action (PK Action Descriptor)', () => {

    describe('action() helper function', () => {
        it('should create an ActionBuilder with action name', () => {
            const result = action('testAction');

            expect(result).toBeInstanceOf(ActionBuilder);
            expect(result.action).toBe('testAction');
        });

        it('should create an ActionBuilder with arguments', () => {
            const result = action('testAction', 'arg1', 123, {key: 'value'});

            expect(result.args).toEqual(['arg1', 123, {key: 'value'}]);
        });

        it('should create an ActionBuilder with no arguments by default', () => {
            const result = action('testAction');

            expect(result.args).toEqual([]);
        });

        it('should support generic type parameter', () => {
            interface Order {
                id: string;
            }

            const result = action<Order>('createOrder');
            expect(result).toBeInstanceOf(ActionBuilder);
        });
    });

    describe('ActionBuilder', () => {
        describe('constructor and getters', () => {
            it('should store action name and args', () => {
                const builder = new ActionBuilder('myAction', ['a', 'b']);

                expect(builder.action).toBe('myAction');
                expect(builder.args).toEqual(['a', 'b']);
            });

            it('should have undefined for unset optional properties', () => {
                const builder = new ActionBuilder('myAction', []);

                expect(builder.method).toBeUndefined();
                expect(builder.optimistic).toBeUndefined();
                expect(builder.onSuccess).toBeUndefined();
                expect(builder.onError).toBeUndefined();
                expect(builder.onComplete).toBeUndefined();
                expect(builder.loading).toBeUndefined();
                expect(builder.debounce).toBeUndefined();
                expect(builder.retry).toBeUndefined();
            });
        });

        describe('fluent setters', () => {
            it('should set method and return this', () => {
                const builder = action('test');
                const result = builder.setMethod('PUT');

                expect(result).toBe(builder);
                expect(builder.method).toBe('PUT');
            });

            it('should set optimistic callback', () => {
                const mockOptimisticHandler = vi.fn();
                const builder = action('test').setOptimistic(mockOptimisticHandler);

                expect(builder.optimistic).toBe(mockOptimisticHandler);
            });

            it('should set onSuccess callback', () => {
                const mockSuccessHandler = vi.fn();
                const builder = action('test').setOnSuccess(mockSuccessHandler);

                expect(builder.onSuccess).toBe(mockSuccessHandler);
            });

            it('should set onError callback', () => {
                const mockErrorHandler = vi.fn();
                const builder = action('test').setOnError(mockErrorHandler);

                expect(builder.onError).toBe(mockErrorHandler);
            });

            it('should set onComplete callback', () => {
                const mockCompleteHandler = vi.fn();
                const builder = action('test').setOnComplete(mockCompleteHandler);

                expect(builder.onComplete).toBe(mockCompleteHandler);
            });

            it('should set loading with boolean', () => {
                const builder = action('test').setLoading(true);

                expect(builder.loading).toBe(true);
            });

            it('should set loading with string selector', () => {
                const builder = action('test').setLoading('.loading-container');

                expect(builder.loading).toBe('.loading-container');
            });

            it('should set loading with HTMLElement', () => {
                const element = document.createElement('div');
                const builder = action('test').setLoading(element);

                expect(builder.loading).toBe(element);
            });

            it('should set debounce', () => {
                const builder = action('test').setDebounce(300);

                expect(builder.debounce).toBe(300);
            });

            it('should set retry with attempts only', () => {
                const builder = action('test').setRetry(3);

                expect(builder.retry).toEqual({attempts: 3, backoff: undefined});
            });

            it('should set retry with attempts and backoff', () => {
                const builder = action('test').setRetry(5, 'exponential');

                expect(builder.retry).toEqual({attempts: 5, backoff: 'exponential'});
            });

            it('should support method chaining', () => {
                const optimisticFn = vi.fn();
                const successFn = vi.fn();
                const errorFn = vi.fn();
                const completeFn = vi.fn();

                const builder = action('chainTest', 'arg1')
                    .setMethod('POST')
                    .setOptimistic(optimisticFn)
                    .setOnSuccess(successFn)
                    .setOnError(errorFn)
                    .setOnComplete(completeFn)
                    .setLoading(true)
                    .setDebounce(100)
                    .setRetry(2, 'linear');

                expect(builder.action).toBe('chainTest');
                expect(builder.args).toEqual(['arg1']);
                expect(builder.method).toBe('POST');
                expect(builder.optimistic).toBe(optimisticFn);
                expect(builder.onSuccess).toBe(successFn);
                expect(builder.onError).toBe(errorFn);
                expect(builder.onComplete).toBe(completeFn);
                expect(builder.loading).toBe(true);
                expect(builder.debounce).toBe(100);
                expect(builder.retry).toEqual({attempts: 2, backoff: 'linear'});
            });
        });

        describe('build()', () => {
            it('should return a plain ActionDescriptor object', () => {
                const builder = action('buildTest', 'x', 'y')
                    .setMethod('DELETE')
                    .setLoading(true);

                const descriptor = builder.build();

                expect(descriptor).not.toBeInstanceOf(ActionBuilder);
                expect(descriptor.action).toBe('buildTest');
                expect(descriptor.args).toEqual(['x', 'y']);
                expect(descriptor.method).toBe('DELETE');
                expect(descriptor.loading).toBe(true);
            });

            it('should include all set properties', () => {
                const mockHandler = vi.fn();
                const descriptor = action('test')
                    .setOptimistic(mockHandler)
                    .setOnSuccess(mockHandler)
                    .setOnError(mockHandler)
                    .setOnComplete(mockHandler)
                    .build();

                expect(descriptor.optimistic).toBe(mockHandler);
                expect(descriptor.onSuccess).toBe(mockHandler);
                expect(descriptor.onError).toBe(mockHandler);
                expect(descriptor.onComplete).toBe(mockHandler);
            });
        });

        describe('ActionDescriptor interface implementation', () => {
            it('should be usable as ActionDescriptor without calling build()', () => {
                const builder = action('interfaceTest', 123);

                const descriptor: ActionDescriptor = builder;

                expect(descriptor.action).toBe('interfaceTest');
                expect(descriptor.args).toEqual([123]);
            });
        });
    });

    describe('isActionDescriptor()', () => {
        it('should return true for ActionBuilder', () => {
            const builder = action('test');

            expect(isActionDescriptor(builder)).toBe(true);
        });

        it('should return true for built ActionDescriptor', () => {
            const descriptor = action('test').build();

            expect(isActionDescriptor(descriptor)).toBe(true);
        });

        it('should return true for plain object with action property', () => {
            const plain = {action: 'plainAction', args: []};

            expect(isActionDescriptor(plain)).toBe(true);
        });

        it('should return true for object with only action property', () => {
            const minimal = {action: 'minimal'};

            expect(isActionDescriptor(minimal)).toBe(true);
        });

        it('should return false for null', () => {
            expect(isActionDescriptor(null)).toBe(false);
        });

        it('should return false for undefined', () => {
            expect(isActionDescriptor(undefined)).toBe(false);
        });

        it('should return false for string', () => {
            expect(isActionDescriptor('action')).toBe(false);
        });

        it('should return false for number', () => {
            expect(isActionDescriptor(42)).toBe(false);
        });

        it('should return false for array', () => {
            expect(isActionDescriptor(['action'])).toBe(false);
        });

        it('should return false for empty object', () => {
            expect(isActionDescriptor({})).toBe(false);
        });

        it('should return false for object with non-string action', () => {
            expect(isActionDescriptor({action: 123})).toBe(false);
            expect(isActionDescriptor({action: null})).toBe(false);
            expect(isActionDescriptor({action: {}})).toBe(false);
        });

        it('should return false for function', () => {
            expect(isActionDescriptor(() => 'action')).toBe(false);
        });
    });

    describe('createActionError()', () => {
        it('should create error with basic properties', () => {
            const error = createActionError(500, 'Server Error');

            expect(error.status).toBe(500);
            expect(error.message).toBe('Server Error');
            expect(error.validationErrors).toBeUndefined();
            expect(error.data).toBeUndefined();
        });

        it('should create error with validation errors', () => {
            const validationErrors = {
                email: ['Invalid email format'],
                password: ['Too short', 'Must contain number']
            };
            const error = createActionError(422, 'Validation Failed', validationErrors);

            expect(error.validationErrors).toEqual(validationErrors);
        });

        it('should create error with data', () => {
            const data = {code: 'RATE_LIMITED', retryAfter: 60};
            const error = createActionError(429, 'Too Many Requests', undefined, data);

            expect(error.data).toEqual(data);
        });

        describe('computed properties', () => {
            it('should return isNetworkError true for status 0', () => {
                const error = createActionError(0, 'Network Error');

                expect(error.isNetworkError).toBe(true);
                expect(error.isValidationError).toBe(false);
                expect(error.isAuthError).toBe(false);
            });

            it('should return isNetworkError false for non-zero status', () => {
                const error = createActionError(500, 'Server Error');

                expect(error.isNetworkError).toBe(false);
            });

            it('should return isValidationError true for 422 with validation errors', () => {
                const error = createActionError(422, 'Validation Failed', {field: ['error']});

                expect(error.isValidationError).toBe(true);
                expect(error.isNetworkError).toBe(false);
                expect(error.isAuthError).toBe(false);
            });

            it('should return isValidationError false for 422 without validation errors', () => {
                const error = createActionError(422, 'Unprocessable');

                expect(error.isValidationError).toBe(false);
            });

            it('should return isAuthError true for 401', () => {
                const error = createActionError(401, 'Unauthorized');

                expect(error.isAuthError).toBe(true);
                expect(error.isNetworkError).toBe(false);
                expect(error.isValidationError).toBe(false);
            });

            it('should return isAuthError true for 403', () => {
                const error = createActionError(403, 'Forbidden');

                expect(error.isAuthError).toBe(true);
            });

            it('should return isAuthError false for other statuses', () => {
                expect(createActionError(400, 'Bad Request').isAuthError).toBe(false);
                expect(createActionError(404, 'Not Found').isAuthError).toBe(false);
                expect(createActionError(500, 'Server Error').isAuthError).toBe(false);
            });
        });
    });

    describe('real-world usage patterns', () => {
        it('should support optimistic UI with rollback', () => {
            let likeCount = 0;
            let isLiked = false;

            const likeAction = action('likePost', 123)
                .setOptimistic(() => {
                    likeCount++;
                    isLiked = true;
                })
                .setOnError(() => {
                    likeCount--;
                    isLiked = false;
                });

            likeAction.optimistic?.();
            expect(likeCount).toBe(1);
            expect(isLiked).toBe(true);

            likeAction.onError?.(createActionError(500, 'Failed'));
            expect(likeCount).toBe(0);
            expect(isLiked).toBe(false);
        });

        it('should support action chaining', () => {
            const actions: string[] = [];

            const firstAction = action('first')
                .setOnSuccess(() => {
                    actions.push('first completed');
                    return action('second').setOnSuccess(() => {
                        actions.push('second completed');
                    });
                });

            const nextAction = firstAction.onSuccess?.({});
            expect(actions).toEqual(['first completed']);
            expect(isActionDescriptor(nextAction)).toBe(true);

            (nextAction as ActionDescriptor).onSuccess?.({});
            expect(actions).toEqual(['first completed', 'second completed']);
        });
    });

    describe('ActionBuilder additional setters', () => {
        it('should set timeout and return this', () => {
            const builder = action('test');
            const result = builder.setTimeout(5000);

            expect(result).toBe(builder);
            expect(builder.timeout).toBe(5000);
        });

        it('should set signal and return this', () => {
            const controller = new AbortController();
            const builder = action('test');
            const result = builder.setSignal(controller.signal);

            expect(result).toBe(builder);
            expect(builder.signal).toBe(controller.signal);
        });

        it('should set onProgress callback and return this', () => {
            const mockProgressHandler = vi.fn();
            const builder = action('test');
            const result = builder.setOnProgress(mockProgressHandler);

            expect(result).toBe(builder);
            expect(builder.onProgress).toBe(mockProgressHandler);
        });

        it('should set retryStream config and return this', () => {
            const config: RetryStreamConfig = {
                maxReconnects: 10,
                backoff: 'exponential',
                baseDelay: 1000,
                maxDelay: 60000
            };
            const builder = action('test');
            const result = builder.setRetryStream(config);

            expect(result).toBe(builder);
            expect(builder.retryStream).toEqual(config);
        });

        it('should set suppressHelpers flag and return this', () => {
            const builder = action('test');
            const result = builder.suppressHelpers();

            expect(result).toBe(builder);
            expect(builder.shouldSuppressHelpers).toBe(true);
        });

        it('should have shouldSuppressHelpers as false by default', () => {
            const builder = action('test');

            expect(builder.shouldSuppressHelpers).toBe(false);
        });

        it('should have undefined for timeout, signal, onProgress, retryStream when unset', () => {
            const builder = new ActionBuilder('myAction', []);

            expect(builder.timeout).toBeUndefined();
            expect(builder.signal).toBeUndefined();
            expect(builder.onProgress).toBeUndefined();
            expect(builder.retryStream).toBeUndefined();
        });
    });

    describe('ActionBuilder build() with all fields', () => {
        it('should include timeout, signal, onProgress, retryStream, and shouldSuppressHelpers in build output', () => {
            const controller = new AbortController();
            const progressFn = vi.fn();
            const streamConfig: RetryStreamConfig = {maxReconnects: 5, backoff: 'linear'};

            const descriptor = action('fullBuild', 'a')
                .setMethod('PUT')
                .setTimeout(3000)
                .setSignal(controller.signal)
                .setOnProgress(progressFn)
                .setRetryStream(streamConfig)
                .suppressHelpers()
                .build();

            expect(descriptor.action).toBe('fullBuild');
            expect(descriptor.args).toEqual(['a']);
            expect(descriptor.method).toBe('PUT');
            expect(descriptor.timeout).toBe(3000);
            expect(descriptor.signal).toBe(controller.signal);
            expect(descriptor.onProgress).toBe(progressFn);
            expect(descriptor.retryStream).toEqual(streamConfig);
            expect(descriptor.shouldSuppressHelpers).toBe(true);
        });

        it('should have shouldSuppressHelpers as undefined (not false) when suppressHelpers is not called', () => {
            const descriptor = action('test').build();

            expect(descriptor.shouldSuppressHelpers).toBeUndefined();
        });
    });

    describe('ActionBuilder with* alias methods', () => {
        it('withMethod should delegate to setMethod and return this', () => {
            const builder = action('test');
            const result = builder.withMethod('PATCH');

            expect(result).toBe(builder);
            expect(builder.method).toBe('PATCH');
        });

        it('withOptimistic should delegate to setOptimistic and return this', () => {
            const mockOptimisticHandler = vi.fn();
            const builder = action('test');
            const result = builder.withOptimistic(mockOptimisticHandler);

            expect(result).toBe(builder);
            expect(builder.optimistic).toBe(mockOptimisticHandler);
        });

        it('withOnSuccess should delegate to setOnSuccess and return this', () => {
            const mockSuccessHandler = vi.fn();
            const builder = action('test');
            const result = builder.withOnSuccess(mockSuccessHandler);

            expect(result).toBe(builder);
            expect(builder.onSuccess).toBe(mockSuccessHandler);
        });

        it('withOnError should delegate to setOnError and return this', () => {
            const mockErrorHandler = vi.fn();
            const builder = action('test');
            const result = builder.withOnError(mockErrorHandler);

            expect(result).toBe(builder);
            expect(builder.onError).toBe(mockErrorHandler);
        });

        it('withOnComplete should delegate to setOnComplete and return this', () => {
            const mockCompleteHandler = vi.fn();
            const builder = action('test');
            const result = builder.withOnComplete(mockCompleteHandler);

            expect(result).toBe(builder);
            expect(builder.onComplete).toBe(mockCompleteHandler);
        });

        it('withLoading should delegate to setLoading and return this', () => {
            const builder = action('test');
            const result = builder.withLoading('#spinner');

            expect(result).toBe(builder);
            expect(builder.loading).toBe('#spinner');
        });

        it('withDebounce should delegate to setDebounce and return this', () => {
            const builder = action('test');
            const result = builder.withDebounce(250);

            expect(result).toBe(builder);
            expect(builder.debounce).toBe(250);
        });

        it('withRetry should delegate to setRetry and return this', () => {
            const builder = action('test');
            const result = builder.withRetry(4, 'exponential');

            expect(result).toBe(builder);
            expect(builder.retry).toEqual({attempts: 4, backoff: 'exponential'});
        });

        it('withTimeout should delegate to setTimeout and return this', () => {
            const builder = action('test');
            const result = builder.withTimeout(8000);

            expect(result).toBe(builder);
            expect(builder.timeout).toBe(8000);
        });

        it('withSignal should delegate to setSignal and return this', () => {
            const controller = new AbortController();
            const builder = action('test');
            const result = builder.withSignal(controller.signal);

            expect(result).toBe(builder);
            expect(builder.signal).toBe(controller.signal);
        });

        it('withOnProgress should delegate to setOnProgress and return this', () => {
            const mockProgressHandler = vi.fn();
            const builder = action('test');
            const result = builder.withOnProgress(mockProgressHandler);

            expect(result).toBe(builder);
            expect(builder.onProgress).toBe(mockProgressHandler);
        });

        it('withRetryStream should delegate to setRetryStream and return this', () => {
            const config: RetryStreamConfig = {maxReconnects: Infinity};
            const builder = action('test');
            const result = builder.withRetryStream(config);

            expect(result).toBe(builder);
            expect(builder.retryStream).toEqual(config);
        });

        it('should support full chaining with with* aliases', () => {
            const mockHandler = vi.fn();
            const controller = new AbortController();
            const streamConfig: RetryStreamConfig = {maxReconnects: 3};

            const builder = action('chainWithAliases')
                .withMethod('DELETE')
                .withOptimistic(mockHandler)
                .withOnSuccess(mockHandler)
                .withOnError(mockHandler)
                .withOnComplete(mockHandler)
                .withLoading(true)
                .withDebounce(200)
                .withRetry(2, 'linear')
                .withTimeout(5000)
                .withSignal(controller.signal)
                .withOnProgress(mockHandler)
                .withRetryStream(streamConfig);

            expect(builder.action).toBe('chainWithAliases');
            expect(builder.method).toBe('DELETE');
            expect(builder.timeout).toBe(5000);
            expect(builder.signal).toBe(controller.signal);
            expect(builder.onProgress).toBe(mockHandler);
            expect(builder.retryStream).toEqual(streamConfig);
        });
    });

    describe('createActionBuilder()', () => {
        it('should create a builder with args wrapped in an array', () => {
            const builder = createActionBuilder('media.Search', {query: 'cats', limit: 10});

            expect(builder).toBeInstanceOf(ActionBuilder);
            expect(builder.action).toBe('media.Search');
            expect(builder.args).toEqual([{query: 'cats', limit: 10}]);
        });

        it('should convert args with toObject() method', () => {
            const argsWithToObject = {
                query: 'dogs',
                toObject(): Record<string, unknown> {
                    return {query: this.query, converted: true};
                }
            };

            const builder = createActionBuilder('media.Search', argsWithToObject);

            expect(builder.args).toEqual([{query: 'dogs', converted: true}]);
        });

        it('should pass through args without toObject() as-is', () => {
            const plainArgs = {name: 'test', value: 42};

            const builder = createActionBuilder('email.Contact', plainArgs);

            expect(builder.args).toEqual([{name: 'test', value: 42}]);
        });
    });

    describe('action function registry', () => {
        it('should register and retrieve an action function', () => {
            const actionFactory = (...args: unknown[]) => action('email.Contact', ...args);
            registerActionFunction('email.Contact', actionFactory);

            const retrieved = getActionFunction('email.Contact');

            expect(retrieved).toBe(actionFactory);
        });

        it('should return undefined for an unregistered action function', () => {
            const retrieved = getActionFunction('nonexistent.Action');

            expect(retrieved).toBeUndefined();
        });
    });

    describe('ActionBuilder.call()', () => {
        let mockCallServerActionDirect: ReturnType<typeof vi.fn>;

        beforeEach(async () => {
            const mod = await import('@/core/ActionExecutor');
            mockCallServerActionDirect = vi.mocked(mod.callServerActionDirect);
            mockCallServerActionDirect.mockReset();
        });

        it('should delegate to callServerActionDirect with correct arguments', async () => {
            mockCallServerActionDirect.mockResolvedValue({data: {id: 1}, status: 200});

            const result = await action('test.Action', 'arg1')
                .setMethod('PUT')
                .setTimeout(5000)
                .call();

            expect(mockCallServerActionDirect).toHaveBeenCalledWith(
                'test.Action',
                ['arg1'],
                'PUT',
                {
                    timeout: 5000,
                    signal: undefined,
                    suppressHelpers: undefined,
                    onProgress: undefined,
                    retryStream: undefined
                }
            );
            expect(result).toEqual({id: 1});
        });

        it('should default to POST when no method is set', async () => {
            mockCallServerActionDirect.mockResolvedValue({data: 'ok', status: 200});

            await action('test.Default').call();

            expect(mockCallServerActionDirect).toHaveBeenCalledWith(
                'test.Default',
                [],
                'POST',
                expect.objectContaining({})
            );
        });

        it('should pass suppressHelpers when suppressHelpers() is called', async () => {
            mockCallServerActionDirect.mockResolvedValue({data: null, status: 200});

            await action('test.Suppress')
                .suppressHelpers()
                .call();

            expect(mockCallServerActionDirect).toHaveBeenCalledWith(
                'test.Suppress',
                [],
                'POST',
                expect.objectContaining({suppressHelpers: true})
            );
        });

        it('should pass onProgress and retryStream options', async () => {
            const progressFn = vi.fn();
            const streamConfig: RetryStreamConfig = {maxReconnects: 5};
            mockCallServerActionDirect.mockResolvedValue({data: 'streamed', status: 200});

            await action('test.Stream')
                .setOnProgress(progressFn)
                .setRetryStream(streamConfig)
                .call();

            expect(mockCallServerActionDirect).toHaveBeenCalledWith(
                'test.Stream',
                [],
                'POST',
                expect.objectContaining({
                    onProgress: progressFn,
                    retryStream: streamConfig
                })
            );
        });

        it('should return the data field from the response', async () => {
            mockCallServerActionDirect.mockResolvedValue({
                data: {items: [1, 2, 3], total: 3},
                status: 200
            });

            const result = await action<{items: number[]; total: number}>('test.List').call();

            expect(result).toEqual({items: [1, 2, 3], total: 3});
        });
    });

    describe('batch()', () => {
        let originalFetch: typeof globalThis.fetch;

        beforeEach(() => {
            originalFetch = globalThis.fetch;
        });

        afterEach(() => {
            globalThis.fetch = originalFetch;
        });

        it('should send correct JSON body structure with indexed args', async () => {
            let capturedBody: string | undefined;
            globalThis.fetch = vi.fn().mockImplementation(async (_url: string, init: RequestInit) => {
                capturedBody = init.body as string;
                return new Response(JSON.stringify({
                    results: [
                        {name: 'user.Create', status: 200, data: {id: 1}},
                        {name: 'email.Send', status: 200, data: {sent: true}}
                    ],
                    success: true
                }), {status: 200, headers: {'Content-Type': 'application/json'}});
            });

            const action1 = action('user.Create', 'Alice', 30);
            const action2 = action('email.Send', 'hello@example.com');

            const result = await batch(action1, action2);

            expect(globalThis.fetch).toHaveBeenCalledWith(
                '/_piko/actions/_batch',
                expect.objectContaining({
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    credentials: 'same-origin'
                })
            );

            const parsed = JSON.parse(capturedBody!);
            expect(parsed.actions).toEqual([
                {name: 'user.Create', args: {0: 'Alice', 1: 30}},
                {name: 'email.Send', args: {0: 'hello@example.com'}}
            ]);

            expect(result.success).toBe(true);
            expect(result.results).toHaveLength(2);
        });

        it('should handle actions with no args', async () => {
            let capturedBody: string | undefined;
            globalThis.fetch = vi.fn().mockImplementation(async (_url: string, init: RequestInit) => {
                capturedBody = init.body as string;
                return new Response(JSON.stringify({
                    results: [{name: 'ping', status: 200, data: 'pong'}],
                    success: true
                }), {status: 200, headers: {'Content-Type': 'application/json'}});
            });

            await batch(action('ping'));

            const parsed = JSON.parse(capturedBody!);
            expect(parsed.actions).toEqual([
                {name: 'ping', args: {}}
            ]);
        });

        it('should throw ActionError when HTTP response is not ok', async () => {
            globalThis.fetch = vi.fn().mockResolvedValue(
                new Response('Internal Server Error', {status: 500})
            );

            await expect(batch(action('failing.Action')))
                .rejects
                .toMatchObject({
                    status: 500,
                    message: 'Batch request failed with status 500'
                });
        });
    });
});
