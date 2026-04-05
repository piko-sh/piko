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
import {
    handleAction,
    onActionError,
    clearGlobalErrorHandler,
    clearAllDebounceTimers,
    callServerActionDirect
} from '@/core/ActionExecutor';
import {action} from '@/pk/action';
import * as CSRFUtils from '@/core/CSRFUtils';

describe('ActionExecutor', () => {
    let element: HTMLElement;
    let fetchSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
        element = document.createElement('button');
        document.body.appendChild(element);

        vi.spyOn(CSRFUtils, 'getCSRFTokenFromMeta').mockReturnValue('test-action-token');
        vi.spyOn(CSRFUtils, 'getCSRFEphemeralFromMeta').mockReturnValue('test-ephemeral-token');
        vi.spyOn(CSRFUtils, 'getCSRFTokensFromMeta').mockReturnValue({
            actionToken: 'test-action-token',
            ephemeralToken: 'test-ephemeral-token'
        });

        fetchSpy = vi.spyOn(globalThis, 'fetch');
    });

    afterEach(() => {
        document.body.innerHTML = '';
        vi.restoreAllMocks();
        clearGlobalErrorHandler();
        clearAllDebounceTimers();
    });

    describe('handleAction()', () => {
        describe('optimistic updates', () => {
            it('should call optimistic callback immediately before fetch', async () => {
                const optimisticFn = vi.fn();
                const callOrder: string[] = [];

                fetchSpy.mockImplementation(async () => {
                    callOrder.push('fetch');
                    return new Response(JSON.stringify({status: 200}), {status: 200});
                });

                const descriptor = action('testAction')
                    .setOptimistic(() => {
                        callOrder.push('optimistic');
                        optimisticFn();
                    });

                await handleAction(descriptor, element);

                expect(optimisticFn).toHaveBeenCalledOnce();
                expect(callOrder).toEqual(['optimistic', 'fetch']);
            });

            it('should continue if optimistic callback throws', async () => {
                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('testAction')
                    .setOptimistic(() => {
                        throw new Error('Optimistic failed');
                    });

                await handleAction(descriptor, element);

                expect(consoleSpy).toHaveBeenCalledWith(
                    '[ActionExecutor] Optimistic update failed:',
                    expect.any(Error)
                );
                expect(fetchSpy).toHaveBeenCalled();

                consoleSpy.mockRestore();
            });
        });

        describe('server communication', () => {
            it('should make POST request to action endpoint by default', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction');
                await handleAction(descriptor, element);

                expect(fetchSpy).toHaveBeenCalledWith(
                    '/_piko/actions/myAction',
                    expect.objectContaining({
                        method: 'POST',
                        credentials: 'same-origin'
                    })
                );
            });

            it('should use specified HTTP method', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction').setMethod('PUT');
                await handleAction(descriptor, element);

                expect(fetchSpy).toHaveBeenCalledWith(
                    expect.any(String),
                    expect.objectContaining({method: 'PUT'})
                );
            });

            it('should include CSRF tokens in request', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction');
                await handleAction(descriptor, element);

                const [, options] = fetchSpy.mock.calls[0];
                expect(options.headers['X-CSRF-Action-Token']).toBe('test-action-token');

                const body = JSON.parse(options.body as string);
                expect(body._csrf_ephemeral_token).toBe('test-ephemeral-token');
            });

            it('should include args in request body', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction', 'arg1', 123, {key: 'value'});
                await handleAction(descriptor, element);

                const [, options] = fetchSpy.mock.calls[0];
                const body = JSON.parse(options.body as string);

                expect(body.args).toEqual({
                    '0': 'arg1',
                    '1': 123,
                    '2': {key: 'value'}
                });
            });
        });

        describe('success callback', () => {
            it('should call onSuccess with response data', async () => {
                const responseData = {data: {id: 123, name: 'Test'}};
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify(responseData), {status: 200})
                );

                const onSuccessFn = vi.fn();
                const descriptor = action('myAction').setOnSuccess(onSuccessFn);

                await handleAction(descriptor, element);

                expect(onSuccessFn).toHaveBeenCalledWith({id: 123, name: 'Test'});
            });

            it('should chain actions when onSuccess returns an action descriptor', async () => {
                fetchSpy
                    .mockResolvedValueOnce(new Response(JSON.stringify({status: 200}), {status: 200}))
                    .mockResolvedValueOnce(new Response(JSON.stringify({status: 200}), {status: 200}));

                const secondSuccess = vi.fn();
                const descriptor = action('firstAction')
                    .setOnSuccess(() => {
                        return action('secondAction').setOnSuccess(secondSuccess);
                    });

                await handleAction(descriptor, element);

                expect(fetchSpy).toHaveBeenCalledTimes(2);
                expect(fetchSpy).toHaveBeenNthCalledWith(1, '/_piko/actions/firstAction', expect.anything());
                expect(fetchSpy).toHaveBeenNthCalledWith(2, '/_piko/actions/secondAction', expect.anything());
                expect(secondSuccess).toHaveBeenCalled();
            });

            it('should handle onSuccess callback errors gracefully', async () => {
                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction')
                    .setOnSuccess(() => {
                        throw new Error('Callback failed');
                    });

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(consoleSpy).toHaveBeenCalledWith(
                    '[ActionExecutor] onSuccess callback failed:',
                    expect.any(Error)
                );

                consoleSpy.mockRestore();
            });
        });

        describe('error callback', () => {
            it('should call onError when server returns error', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Server error'}), {status: 500})
                );

                const onErrorFn = vi.fn();
                const descriptor = action('myAction').setOnError(onErrorFn);

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(onErrorFn).toHaveBeenCalledWith(
                    expect.objectContaining({
                        status: 500,
                        message: 'Server error'
                    })
                );
            });

            it('should include validation errors for 422 responses', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({
                        message: 'Validation failed',
                        errors: {email: ['Invalid email'], password: ['Too short']}
                    }), {status: 422})
                );

                const onErrorFn = vi.fn();
                const descriptor = action('myAction').setOnError(onErrorFn);

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(onErrorFn).toHaveBeenCalledWith(
                    expect.objectContaining({
                        status: 422,
                        validationErrors: {email: ['Invalid email'], password: ['Too short']}
                    })
                );
            });

            it('should call onError for network errors', async () => {
                fetchSpy.mockRejectedValueOnce(new Error('Network error'));

                const onErrorFn = vi.fn();
                const descriptor = action('myAction').setOnError(onErrorFn);

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(onErrorFn).toHaveBeenCalled();
            });

            it('should log error to console when no onError provided', async () => {
                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Failed'}), {status: 500})
                );

                const descriptor = action('myAction');

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(consoleSpy).toHaveBeenCalledWith(
                    '[ActionExecutor] Action failed:',
                    expect.anything()
                );

                consoleSpy.mockRestore();
            });
        });

        describe('complete callback', () => {
            it('should call onComplete after success', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const onCompleteFn = vi.fn();
                const descriptor = action('myAction').setOnComplete(onCompleteFn);

                await handleAction(descriptor, element);

                expect(onCompleteFn).toHaveBeenCalledOnce();
            });

            it('should call onComplete after error', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Failed'}), {status: 500})
                );

                const onCompleteFn = vi.fn();
                const descriptor = action('myAction')
                    .setOnError(() => {})
                    .setOnComplete(onCompleteFn);

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(onCompleteFn).toHaveBeenCalledOnce();
            });

            it('should handle onComplete callback errors gracefully', async () => {
                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction')
                    .setOnComplete(() => {
                        throw new Error('Complete failed');
                    });

                await handleAction(descriptor, element);

                expect(consoleSpy).toHaveBeenCalledWith(
                    '[ActionExecutor] onComplete callback failed:',
                    expect.any(Error)
                );

                consoleSpy.mockRestore();
            });
        });

        describe('loading state', () => {
            it('should add loading class to element when loading=true', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction').setLoading(true);

                const promise = handleAction(descriptor, element);

                expect(element.classList.contains('pk-loading')).toBe(true);
                expect(element.getAttribute('aria-busy')).toBe('true');

                await promise;

                expect(element.classList.contains('pk-loading')).toBe(false);
                expect(element.getAttribute('aria-busy')).toBeNull();
            });

            it('should add loading class to selector target', async () => {
                const container = document.createElement('div');
                container.className = 'loading-target';
                document.body.appendChild(container);

                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction').setLoading('.loading-target');

                const promise = handleAction(descriptor, element);

                expect(container.classList.contains('pk-loading')).toBe(true);

                await promise;

                expect(container.classList.contains('pk-loading')).toBe(false);
            });

            it('should add loading class to specific HTMLElement', async () => {
                const loadingEl = document.createElement('div');
                document.body.appendChild(loadingEl);

                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('myAction').setLoading(loadingEl);

                const promise = handleAction(descriptor, element);

                expect(loadingEl.classList.contains('pk-loading')).toBe(true);

                await promise;

                expect(loadingEl.classList.contains('pk-loading')).toBe(false);
            });

            it('should remove loading state even on error', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Failed'}), {status: 500})
                );

                const descriptor = action('myAction')
                    .setLoading(true)
                    .setOnError(() => {});

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(element.classList.contains('pk-loading')).toBe(false);
            });
        });

        describe('real-world scenarios', () => {
            it('should support optimistic UI with rollback on error', async () => {
                let likeCount = 0;

                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Failed'}), {status: 500})
                );

                const descriptor = action('likePost', 123)
                    .setOptimistic(() => {
                        likeCount++;
                    })
                    .setOnError(() => {
                        likeCount--;
                    });

                expect(likeCount).toBe(0);

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(likeCount).toBe(0);
            });

            it('should support complete flow with loading', async () => {
                const events: string[] = [];

                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({data: {result: 'success'}}), {status: 200})
                );

                const descriptor = action('submitForm')
                    .setMethod('POST')
                    .setOptimistic(() => events.push('optimistic'))
                    .setLoading(true)
                    .setOnSuccess((response) => {
                        events.push(`success:${(response as {result: string}).result}`);
                    })
                    .setOnComplete(() => events.push('complete'));

                await handleAction(descriptor, element);

                expect(events).toEqual(['optimistic', 'success:success', 'complete']);
            });
        });

        describe('debounce', () => {
            it('should delay action execution by debounce time', async () => {
                vi.useFakeTimers();

                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('searchAction').setDebounce(300);

                const promise = handleAction(descriptor, element);

                expect(fetchSpy).not.toHaveBeenCalled();

                vi.advanceTimersByTime(200);
                expect(fetchSpy).not.toHaveBeenCalled();

                vi.advanceTimersByTime(150);
                await promise;

                expect(fetchSpy).toHaveBeenCalledOnce();

                vi.useRealTimers();
            });

            it('should reset timer on subsequent calls', async () => {
                vi.useFakeTimers();

                fetchSpy.mockResolvedValue(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('searchAction').setDebounce(300);

                handleAction(descriptor, element);

                vi.advanceTimersByTime(200);
                expect(fetchSpy).not.toHaveBeenCalled();

                const promise = handleAction(descriptor, element);

                vi.advanceTimersByTime(200);
                expect(fetchSpy).not.toHaveBeenCalled();

                vi.advanceTimersByTime(150);
                await promise;

                expect(fetchSpy).toHaveBeenCalledOnce();

                vi.useRealTimers();
            });

            it('should clear debounce timer with clearAllDebounceTimers()', async () => {
                vi.useFakeTimers();

                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('searchAction').setDebounce(300);

                handleAction(descriptor, element);

                vi.advanceTimersByTime(100);
                expect(fetchSpy).not.toHaveBeenCalled();

                clearAllDebounceTimers();

                vi.advanceTimersByTime(500);

                expect(fetchSpy).not.toHaveBeenCalled();

                vi.useRealTimers();
            });

            it('should execute immediately when debounce is 0 or not set', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

                const descriptor = action('instantAction');

                await handleAction(descriptor, element);

                expect(fetchSpy).toHaveBeenCalledOnce();
            });

            it('should use different debounce keys for different elements', async () => {
                vi.useFakeTimers();

                const element2 = document.createElement('button');
                document.body.appendChild(element2);

                fetchSpy
                    .mockResolvedValueOnce(new Response(JSON.stringify({status: 200}), {status: 200}))
                    .mockResolvedValueOnce(new Response(JSON.stringify({status: 200}), {status: 200}));

                const descriptor = action('searchAction').setDebounce(300);

                const promise1 = handleAction(descriptor, element);

                const promise2 = handleAction(descriptor, element2);

                vi.advanceTimersByTime(350);
                await Promise.all([promise1, promise2]);

                expect(fetchSpy).toHaveBeenCalledTimes(2);

                vi.useRealTimers();
            });
        });

        describe('retry', () => {
            it('should retry on 5xx errors', async () => {
                vi.useFakeTimers();

                fetchSpy
                    .mockResolvedValueOnce(new Response(JSON.stringify({message: 'Server error'}), {status: 500}))
                    .mockResolvedValueOnce(new Response(JSON.stringify({message: 'Server error'}), {status: 500}))
                    .mockResolvedValueOnce(new Response(JSON.stringify({status: 200}), {status: 200}));

                const onSuccessFn = vi.fn();
                const descriptor = action('retryAction')
                    .setRetry(3, 'exponential')
                    .setOnSuccess(onSuccessFn);

                const promise = handleAction(descriptor, element);

                await vi.advanceTimersByTimeAsync(0);

                await vi.advanceTimersByTimeAsync(1000);

                await vi.advanceTimersByTimeAsync(2000);

                await promise;

                expect(fetchSpy).toHaveBeenCalledTimes(3);
                expect(onSuccessFn).toHaveBeenCalled();

                vi.useRealTimers();
            });

            it('should not retry on 4xx errors', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Bad request'}), {status: 400})
                );

                const onErrorFn = vi.fn();
                const descriptor = action('retryAction')
                    .setRetry(3)
                    .setOnError(onErrorFn);

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(fetchSpy).toHaveBeenCalledOnce();
                expect(onErrorFn).toHaveBeenCalledWith(
                    expect.objectContaining({status: 400})
                );
            });

            it('should retry on network errors (status 0)', async () => {
                vi.useFakeTimers();

                fetchSpy
                    .mockRejectedValueOnce(new Error('Network error'))
                    .mockResolvedValueOnce(new Response(JSON.stringify({status: 200}), {status: 200}));

                const onSuccessFn = vi.fn();
                const descriptor = action('retryAction')
                    .setRetry(2)
                    .setOnSuccess(onSuccessFn);

                const promise = handleAction(descriptor, element);

                await vi.advanceTimersByTimeAsync(1100);

                await promise;

                expect(fetchSpy).toHaveBeenCalledTimes(2);
                expect(onSuccessFn).toHaveBeenCalled();

                vi.useRealTimers();
            });

            it('should respect maximum retry attempts', async () => {
                vi.useFakeTimers();

                fetchSpy.mockImplementation(async () => {
                    return new Response(JSON.stringify({message: 'Server error'}), {status: 500});
                });

                const onErrorFn = vi.fn();
                const descriptor = action('retryAction')
                    .setRetry(3)
                    .setOnError(onErrorFn);

                const promise = handleAction(descriptor, element);
                const rejection = expect(promise).rejects.toThrow();

                await vi.advanceTimersByTimeAsync(10000);

                await rejection;

                expect(fetchSpy).toHaveBeenCalledTimes(3);
                expect(onErrorFn).toHaveBeenCalled();

                vi.useRealTimers();
            });

            it('should use exponential backoff by default', async () => {
                vi.useFakeTimers();

                const delays: number[] = [];
                let lastCallTime = Date.now();
                fetchSpy.mockImplementation(async () => {
                    const now = Date.now();
                    if (fetchSpy.mock.calls.length > 1) {
                        delays.push(now - lastCallTime);
                    }
                    lastCallTime = now;
                    return new Response(JSON.stringify({message: 'Server error'}), {status: 500});
                });

                const descriptor = action('retryAction')
                    .setRetry(4)
                    .setOnError(() => {});

                const promise = handleAction(descriptor, element);
                const rejection = expect(promise).rejects.toThrow();

                await vi.advanceTimersByTimeAsync(1100);
                await vi.advanceTimersByTimeAsync(2100);
                await vi.advanceTimersByTimeAsync(4100);

                await rejection;

                expect(delays[0]).toBeGreaterThanOrEqual(1000);
                expect(delays[1]).toBeGreaterThanOrEqual(2000);
                expect(delays[2]).toBeGreaterThanOrEqual(4000);

                vi.useRealTimers();
            });

            it('should use linear backoff when specified', async () => {
                vi.useFakeTimers();

                const delays: number[] = [];
                let lastCallTime = Date.now();

                fetchSpy.mockImplementation(async () => {
                    const now = Date.now();
                    if (fetchSpy.mock.calls.length > 1) {
                        delays.push(now - lastCallTime);
                    }
                    lastCallTime = now;
                    return new Response(JSON.stringify({message: 'Server error'}), {status: 500});
                });

                const descriptor = action('retryAction')
                    .setRetry(4, 'linear')
                    .setOnError(() => {});

                const promise = handleAction(descriptor, element);
                const rejection = expect(promise).rejects.toThrow();

                await vi.advanceTimersByTimeAsync(1100);
                await vi.advanceTimersByTimeAsync(2100);
                await vi.advanceTimersByTimeAsync(3100);

                await rejection;

                expect(delays[0]).toBeGreaterThanOrEqual(1000);
                expect(delays[0]).toBeLessThan(1500);
                expect(delays[1]).toBeGreaterThanOrEqual(2000);
                expect(delays[1]).toBeLessThan(2500);
                expect(delays[2]).toBeGreaterThanOrEqual(3000);
                expect(delays[2]).toBeLessThan(3500);

                vi.useRealTimers();
            });
        });

        describe('global error handler', () => {
            it('should call global error handler on action error', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Server error'}), {status: 500})
                );

                const globalHandler = vi.fn();
                onActionError(globalHandler);

                const descriptor = action('failAction').setOnError(() => {});

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(globalHandler).toHaveBeenCalledWith(
                    expect.objectContaining({status: 500}),
                    expect.objectContaining({action: 'failAction'})
                );
            });

            it('should call global handler before local onError', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Server error'}), {status: 500})
                );

                const callOrder: string[] = [];

                onActionError(() => callOrder.push('global'));

                const descriptor = action('failAction')
                    .setOnError(() => callOrder.push('local'));

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(callOrder).toEqual(['global', 'local']);
            });

            it('should allow unsubscribing from global error handler', async () => {
                fetchSpy.mockResolvedValue(
                    new Response(JSON.stringify({message: 'Server error'}), {status: 500})
                );

                const globalHandler = vi.fn();
                const unsubscribe = onActionError(globalHandler);

                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

                await expect(handleAction(action('fail1').setOnError(() => {}), element)).rejects.toThrow();
                expect(globalHandler).toHaveBeenCalledTimes(1);

                unsubscribe();

                await expect(handleAction(action('fail2').setOnError(() => {}), element)).rejects.toThrow();
                expect(globalHandler).toHaveBeenCalledTimes(1);

                consoleSpy.mockRestore();
            });

            it('should not unsubscribe different handler', async () => {
                fetchSpy.mockResolvedValue(
                    new Response(JSON.stringify({message: 'Server error'}), {status: 500})
                );

                const handler1 = vi.fn();
                const handler2 = vi.fn();

                const unsubscribe1 = onActionError(handler1);
                onActionError(handler2);

                unsubscribe1();

                await expect(handleAction(action('fail').setOnError(() => {}), element)).rejects.toThrow();
                expect(handler2).toHaveBeenCalled();
            });

            it('should use clearGlobalErrorHandler() to clear handler', async () => {
                fetchSpy.mockResolvedValue(
                    new Response(JSON.stringify({message: 'Server error'}), {status: 500})
                );

                const globalHandler = vi.fn();
                onActionError(globalHandler);

                clearGlobalErrorHandler();

                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
                await expect(handleAction(action('fail'), element)).rejects.toThrow();

                expect(globalHandler).not.toHaveBeenCalled();
                expect(consoleSpy).toHaveBeenCalledWith(
                    '[ActionExecutor] Action failed:',
                    expect.anything()
                );

                consoleSpy.mockRestore();
            });

            it('should continue even if global handler throws', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Server error'}), {status: 500})
                );

                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

                onActionError(() => {
                    throw new Error('Handler crashed');
                });

                const localHandler = vi.fn();
                const descriptor = action('failAction').setOnError(localHandler);

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(consoleSpy).toHaveBeenCalledWith(
                    '[ActionExecutor] Global error handler failed:',
                    expect.any(Error)
                );

                expect(localHandler).toHaveBeenCalled();

                consoleSpy.mockRestore();
            });

            it('should suppress default console.error when global handler is set', async () => {
                fetchSpy.mockResolvedValueOnce(
                    new Response(JSON.stringify({message: 'Server error'}), {status: 500})
                );

                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

                const globalHandler = vi.fn();
                onActionError(globalHandler);

                const descriptor = action('failAction');

                await expect(handleAction(descriptor, element)).rejects.toThrow();

                expect(globalHandler).toHaveBeenCalled();

                expect(consoleSpy).not.toHaveBeenCalledWith(
                    '[ActionExecutor] Action failed:',
                    expect.anything()
                );

                consoleSpy.mockRestore();
            });
        });
    });

    describe('ActionBuilder method aliases', () => {
        it('withOptimistic() should be alias for setOptimistic()', () => {
            const mockOptimisticHandler = vi.fn();
            const builder = action('test');

            const result = builder.withOptimistic(mockOptimisticHandler);

            expect(result).toBe(builder);
            expect(builder.optimistic).toBe(mockOptimisticHandler);
        });

        it('withOnSuccess() should be alias for setOnSuccess()', () => {
            const mockSuccessHandler = vi.fn();
            const builder = action('test');

            const result = builder.withOnSuccess(mockSuccessHandler);

            expect(result).toBe(builder);
            expect(builder.onSuccess).toBe(mockSuccessHandler);
        });

        it('withOnError() should be alias for setOnError()', () => {
            const mockErrorHandler = vi.fn();
            const builder = action('test');

            const result = builder.withOnError(mockErrorHandler);

            expect(result).toBe(builder);
            expect(builder.onError).toBe(mockErrorHandler);
        });

        it('withOnComplete() should be alias for setOnComplete()', () => {
            const mockCompleteHandler = vi.fn();
            const builder = action('test');

            const result = builder.withOnComplete(mockCompleteHandler);

            expect(result).toBe(builder);
            expect(builder.onComplete).toBe(mockCompleteHandler);
        });

        it('withLoading() should be alias for setLoading()', () => {
            const builder = action('test');

            builder.withLoading(true);
            expect(builder.loading).toBe(true);

            builder.withLoading('#element');
            expect(builder.loading).toBe('#element');
        });

        it('withDebounce() should be alias for setDebounce()', () => {
            const builder = action('test');

            const result = builder.withDebounce(500);

            expect(result).toBe(builder);
            expect(builder.debounce).toBe(500);
        });

        it('withRetry() should be alias for setRetry()', () => {
            const builder = action('test');

            builder.withRetry(3, 'linear');

            expect(builder.retry).toEqual({attempts: 3, backoff: 'linear'});
        });

        it('withMethod() should be alias for setMethod()', () => {
            const builder = action('test');

            const result = builder.withMethod('DELETE');

            expect(result).toBe(builder);
            expect(builder.method).toBe('DELETE');
        });

        it('should allow chaining all aliases together', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const events: string[] = [];

            const descriptor = action('chainTest')
                .withMethod('PUT')
                .withOptimistic(() => { events.push('optimistic'); })
                .withLoading(true)
                .withOnSuccess(() => { events.push('success'); })
                .withOnError(() => { events.push('error'); })
                .withOnComplete(() => { events.push('complete'); });

            expect(descriptor.method).toBe('PUT');
            expect(descriptor.loading).toBe(true);

            await handleAction(descriptor, element);

            expect(events).toEqual(['optimistic', 'success', 'complete']);
        });
    });

    describe('callServerActionDirect() SSE', () => {
        function createSSEResponse(sseText: string): Response {
            const stream = new ReadableStream({
                start(controller) {
                    controller.enqueue(new TextEncoder().encode(sseText));
                    controller.close();
                }
            });
            return new Response(stream, {
                status: 200,
                headers: {'Content-Type': 'text/event-stream'}
            });
        }

        it('should deliver progress events and resolve with complete data', async () => {
            const sseText =
                'event: progress\ndata: {"percent":50}\n\n' +
                'event: progress\ndata: {"percent":100}\n\n' +
                'event: complete\ndata: {"result":"done"}\n\n';

            fetchSpy.mockResolvedValueOnce(createSSEResponse(sseText));

            const progressEvents: Array<{data: unknown; eventType: string}> = [];
            const onProgress = vi.fn((data: unknown, eventType: string) => {
                progressEvents.push({data, eventType});
            });

            const result = await callServerActionDirect('testSSE', [], 'POST', {onProgress});

            expect(onProgress).toHaveBeenCalledTimes(2);
            expect(progressEvents[0]).toEqual({data: {percent: 50}, eventType: 'progress'});
            expect(progressEvents[1]).toEqual({data: {percent: 100}, eventType: 'progress'});
            expect(result.data).toEqual({result: 'done'});
        });

        it('should parse SSE events with id fields correctly', async () => {
            const sseText =
                'id: 1\nevent: update\ndata: {"v":1}\n\n' +
                'id: 2\nevent: update\ndata: {"v":2}\n\n' +
                'event: complete\ndata: "ok"\n\n';

            fetchSpy.mockResolvedValueOnce(createSSEResponse(sseText));

            const progressEvents: Array<{data: unknown; eventType: string}> = [];
            const onProgress = vi.fn((data: unknown, eventType: string) => {
                progressEvents.push({data, eventType});
            });

            const result = await callServerActionDirect('testSSE', [], 'POST', {onProgress});

            expect(onProgress).toHaveBeenCalledTimes(2);
            expect(progressEvents[0]).toEqual({data: {v: 1}, eventType: 'update'});
            expect(progressEvents[1]).toEqual({data: {v: 2}, eventType: 'update'});
            expect(result.data).toBe('ok');
        });

        it('should reject on SSE error event', async () => {
            const sseText =
                'event: progress\ndata: {"percent":50}\n\n' +
                'event: error\ndata: {"message":"Something went wrong"}\n\n';

            fetchSpy.mockResolvedValueOnce(createSSEResponse(sseText));

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('testSSE', [], 'POST', {onProgress})
            ).rejects.toMatchObject({
                status: 0,
                message: 'Something went wrong'
            });
        });

        it('should reject when SSE stream ends without complete event', async () => {
            const sseText =
                'event: progress\ndata: {"percent":50}\n\n' +
                'event: progress\ndata: {"percent":75}\n\n';

            fetchSpy.mockResolvedValueOnce(createSSEResponse(sseText));

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('testSSE', [], 'POST', {onProgress})
            ).rejects.toMatchObject({
                message: 'SSE stream ended without completion'
            });
        });
    });

    describe('callServerActionDirect() SSE with retryStream', () => {
        function createSSEResponse(sseText: string): Response {
            const stream = new ReadableStream({
                start(controller) {
                    controller.enqueue(new TextEncoder().encode(sseText));
                    controller.close();
                }
            });
            return new Response(stream, {
                status: 200,
                headers: {'Content-Type': 'text/event-stream'}
            });
        }

        it('should reconnect on connection drop and resolve on second attempt', async () => {
            const sseText1 =
                'event: progress\ndata: {"step":1}\n\n' +
                'event: progress\ndata: {"step":2}\n\n';

            const sseText2 =
                'event: progress\ndata: {"step":3}\n\n' +
                'event: complete\ndata: {"result":"finished"}\n\n';

            fetchSpy
                .mockResolvedValueOnce(createSSEResponse(sseText1))
                .mockResolvedValueOnce(createSSEResponse(sseText2));

            const progressEvents: Array<{data: unknown; eventType: string}> = [];
            const onProgress = vi.fn((data: unknown, eventType: string) => {
                progressEvents.push({data, eventType});
            });

            const result = await callServerActionDirect('testSSE', [], 'POST', {
                onProgress,
                retryStream: {maxReconnects: 3, baseDelay: 1}
            });

            expect(onProgress).toHaveBeenCalledTimes(3);
            expect(progressEvents[0]).toEqual({data: {step: 1}, eventType: 'progress'});
            expect(progressEvents[1]).toEqual({data: {step: 2}, eventType: 'progress'});
            expect(progressEvents[2]).toEqual({data: {step: 3}, eventType: 'progress'});
            expect(result.data).toEqual({result: 'finished'});
            expect(fetchSpy).toHaveBeenCalledTimes(2);
        });

        it('should send Last-Event-ID header on reconnect', async () => {
            const sseText1 =
                'id: 5\nevent: update\ndata: 1\n\n';

            const sseText2 =
                'event: complete\ndata: "done"\n\n';

            fetchSpy
                .mockResolvedValueOnce(createSSEResponse(sseText1))
                .mockResolvedValueOnce(createSSEResponse(sseText2));

            const onProgress = vi.fn();

            await callServerActionDirect('testSSE', [], 'POST', {
                onProgress,
                retryStream: {maxReconnects: 3, baseDelay: 1}
            });

            expect(fetchSpy).toHaveBeenCalledTimes(2);

            const secondCallArgs = fetchSpy.mock.calls[1];
            const secondCallOptions = secondCallArgs[1] as RequestInit;
            const headers = secondCallOptions.headers as Record<string, string>;
            expect(headers['Last-Event-ID']).toBe('5');
        });

        it('should respect maxReconnects limit', async () => {
            const sseTextDrop =
                'event: progress\ndata: {"step":1}\n\n';

            fetchSpy.mockResolvedValue(createSSEResponse(sseTextDrop));

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('testSSE', [], 'POST', {
                    onProgress,
                    retryStream: {maxReconnects: 2, baseDelay: 1}
                })
            ).rejects.toMatchObject({
                message: expect.stringContaining('after 2 reconnection attempts')
            });

            expect(fetchSpy).toHaveBeenCalledTimes(3);
        });

        it('should not retry on server error event', async () => {
            const sseText =
                'event: error\ndata: {"message":"Server failure"}\n\n';

            fetchSpy.mockResolvedValueOnce(createSSEResponse(sseText));

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('testSSE', [], 'POST', {
                    onProgress,
                    retryStream: {maxReconnects: 3, baseDelay: 1}
                })
            ).rejects.toMatchObject({
                message: 'Server failure'
            });

            expect(fetchSpy).toHaveBeenCalledTimes(1);
        });

        it('should not retry on HTTP 4xx error', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({message: 'Bad request'}), {status: 400})
            );

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('testSSE', [], 'POST', {
                    onProgress,
                    retryStream: {maxReconnects: 3, baseDelay: 1}
                })
            ).rejects.toMatchObject({
                status: 400
            });

            expect(fetchSpy).toHaveBeenCalledTimes(1);
        });

        it('should call onDisconnect and onReconnect callbacks', async () => {
            const sseText1 =
                'event: progress\ndata: 1\n\n';

            const sseText2 =
                'event: complete\ndata: "ok"\n\n';

            fetchSpy
                .mockResolvedValueOnce(createSSEResponse(sseText1))
                .mockResolvedValueOnce(createSSEResponse(sseText2));

            const onDisconnect = vi.fn();
            const onReconnect = vi.fn();
            const onProgress = vi.fn();

            await callServerActionDirect('testSSE', [], 'POST', {
                onProgress,
                retryStream: {
                    maxReconnects: 3,
                    baseDelay: 1,
                    onDisconnect,
                    onReconnect
                }
            });

            expect(onDisconnect).toHaveBeenCalledOnce();
            expect(onReconnect).toHaveBeenCalledOnce();
            expect(onReconnect).toHaveBeenCalledWith(1);
        });

        it('should not retry on cancellation via AbortController', async () => {
            const controller = new AbortController();
            controller.abort();

            fetchSpy.mockImplementation(async (_url: string, init: RequestInit) => {
                if (init.signal?.aborted) {
                    throw new DOMException('The operation was aborted.', 'AbortError');
                }
                return createSSEResponse('event: complete\ndata: "ok"\n\n');
            });

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('testSSE', [], 'POST', {
                    onProgress,
                    signal: controller.signal,
                    retryStream: {maxReconnects: 3, baseDelay: 1}
                })
            ).rejects.toMatchObject({
                message: 'Request cancelled'
            });

            expect(fetchSpy).toHaveBeenCalledTimes(1);
        });
    });

    describe('CSRF recovery paths', () => {
        it('should reload page on csrf_invalid error', async () => {
            const reloadSpy = vi.fn();
            Object.defineProperty(window, 'location', {
                value: {reload: reloadSpy},
                writable: true,
                configurable: true
            });

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({
                    message: 'CSRF token invalid',
                    error: 'csrf_invalid'
                }), {status: 403})
            );

            const descriptor = action('csrfAction').setOnError(() => {});

            await handleAction(descriptor, element);

            expect(reloadSpy).toHaveBeenCalledOnce();
        });

        it('should dispatch refresh-partial event on csrf_expired when inside a partial', async () => {
            const partial = document.createElement('div');
            partial.setAttribute('partial_src', '/my-partial');
            partial.appendChild(element);
            document.body.appendChild(partial);

            let refreshEventFired = false;
            partial.addEventListener('refresh-partial', ((e: CustomEvent) => {
                refreshEventFired = true;
                const freshEl = document.createElement('input');
                freshEl.setAttribute('data-csrf-action-token', 'fresh-token');
                partial.appendChild(freshEl);
                e.detail.afterMorph();
            }) as EventListener);

            fetchSpy
                .mockResolvedValueOnce(
                    new Response(JSON.stringify({
                        message: 'CSRF token expired',
                        error: 'csrf_expired',
                        data: 'csrf_expired'
                    }), {status: 403})
                )
                .mockResolvedValueOnce(
                    new Response(JSON.stringify({status: 200}), {status: 200})
                );

            const descriptor = action('csrfAction');

            await handleAction(descriptor, element);

            expect(refreshEventFired).toBe(true);
        });

        it('should reload page when csrf_expired and no partial ancestor exists', async () => {
            const reloadSpy = vi.fn();
            Object.defineProperty(window, 'location', {
                value: {reload: reloadSpy},
                writable: true,
                configurable: true
            });

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({
                    message: 'CSRF token expired',
                    error: 'csrf_expired',
                    data: 'csrf_expired'
                }), {status: 403})
            );

            const descriptor = action('csrfAction').setOnError(() => {});

            await handleAction(descriptor, element);

            expect(reloadSpy).toHaveBeenCalledOnce();
        });

        it('should warn when no element with CSRF token found after partial refresh', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const partial = document.createElement('div');
            partial.setAttribute('partial_src', '/my-partial');
            partial.appendChild(element);
            document.body.appendChild(partial);

            partial.addEventListener('refresh-partial', ((e: CustomEvent) => {
                e.detail.afterMorph();
            }) as EventListener);

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({
                    message: 'CSRF token expired',
                    error: 'csrf_expired',
                    data: 'csrf_expired'
                }), {status: 403})
            );

            const descriptor = action('csrfAction');

            await handleAction(descriptor, element);

            expect(warnSpy).toHaveBeenCalledWith(
                '[ActionExecutor] Could not find element with CSRF token after partial refresh'
            );

            warnSpy.mockRestore();
        });
    });

    describe('validation 422 path with form field errors', () => {
        it('should apply error attributes to matching form fields', async () => {
            const form = document.createElement('form');
            const emailInput = document.createElement('input');
            emailInput.setAttribute('name', 'email');
            const passwordInput = document.createElement('input');
            passwordInput.setAttribute('name', 'password');
            form.appendChild(emailInput);
            form.appendChild(passwordInput);
            form.appendChild(element);
            document.body.appendChild(form);

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({
                    message: 'Validation failed',
                    errors: {email: ['Invalid email address'], password: ['Too short', 'Must contain a number']}
                }), {status: 422})
            );

            const descriptor = action('submitForm').setOnError(() => {});

            await expect(handleAction(descriptor, element)).rejects.toThrow();

            expect(emailInput.getAttribute('error')).toBe('Invalid email address');
            expect(passwordInput.getAttribute('error')).toBe('Too short, Must contain a number');
        });

        it('should clear previous error attributes before applying new ones', async () => {
            const form = document.createElement('form');
            const emailInput = document.createElement('input');
            emailInput.setAttribute('name', 'email');
            emailInput.setAttribute('error', 'Old error');
            const nameInput = document.createElement('input');
            nameInput.setAttribute('name', 'username');
            nameInput.setAttribute('error', 'Old name error');
            form.appendChild(emailInput);
            form.appendChild(nameInput);
            form.appendChild(element);
            document.body.appendChild(form);

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({
                    message: 'Validation failed',
                    errors: {email: ['New email error']}
                }), {status: 422})
            );

            const descriptor = action('submitForm').setOnError(() => {});

            await expect(handleAction(descriptor, element)).rejects.toThrow();

            expect(emailInput.getAttribute('error')).toBe('New email error');
            expect(nameInput.hasAttribute('error')).toBe(false);
        });
    });

    describe('form validation failure', () => {
        it('should abort action when HTML5 form validation fails', async () => {
            const form = document.createElement('form');
            const requiredInput = document.createElement('input');
            requiredInput.setAttribute('type', 'text');
            requiredInput.required = true;
            requiredInput.value = '';
            form.appendChild(requiredInput);
            form.appendChild(element);
            document.body.appendChild(form);

            vi.spyOn(form, 'reportValidity').mockReturnValue(false);

            const onSuccessFn = vi.fn();
            const descriptor = action('submitForm').setOnSuccess(onSuccessFn);

            await handleAction(descriptor, element);

            expect(fetchSpy).not.toHaveBeenCalled();
            expect(onSuccessFn).not.toHaveBeenCalled();
        });

        it('should proceed when form validation passes', async () => {
            const form = document.createElement('form');
            const input = document.createElement('input');
            input.setAttribute('type', 'text');
            input.required = true;
            input.value = 'filled';
            form.appendChild(input);
            form.appendChild(element);
            document.body.appendChild(form);

            vi.spyOn(form, 'reportValidity').mockReturnValue(true);

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const onSuccessFn = vi.fn();
            const descriptor = action('submitForm').setOnSuccess(onSuccessFn);

            await handleAction(descriptor, element);

            expect(fetchSpy).toHaveBeenCalledOnce();
            expect(onSuccessFn).toHaveBeenCalled();
        });

        it('should skip validation when submitter has formnovalidate', async () => {
            const form = document.createElement('form');
            const requiredInput = document.createElement('input');
            requiredInput.setAttribute('type', 'text');
            requiredInput.required = true;
            requiredInput.value = '';
            form.appendChild(requiredInput);
            form.appendChild(element);
            document.body.appendChild(form);

            const reportValiditySpy = vi.spyOn(form, 'reportValidity').mockReturnValue(false);

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const onSuccessFn = vi.fn();
            const descriptor = action('submitForm').setOnSuccess(onSuccessFn);

            const submitter = document.createElement('button');
            submitter.type = 'submit';
            submitter.formNoValidate = true;
            const submitEvent = new Event('submit') as any;
            submitEvent.submitter = submitter;

            await handleAction(descriptor, element, submitEvent);

            expect(reportValiditySpy).not.toHaveBeenCalled();
            expect(fetchSpy).toHaveBeenCalledOnce();
            expect(onSuccessFn).toHaveBeenCalled();
        });

        it('should skip validation when form has novalidate attribute', async () => {
            const form = document.createElement('form');
            form.noValidate = true;
            const requiredInput = document.createElement('input');
            requiredInput.setAttribute('type', 'text');
            requiredInput.required = true;
            requiredInput.value = '';
            form.appendChild(requiredInput);
            form.appendChild(element);
            document.body.appendChild(form);

            const reportValiditySpy = vi.spyOn(form, 'reportValidity').mockReturnValue(false);

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const onSuccessFn = vi.fn();
            const descriptor = action('submitForm').setOnSuccess(onSuccessFn);

            await handleAction(descriptor, element);

            expect(reportValiditySpy).not.toHaveBeenCalled();
            expect(fetchSpy).toHaveBeenCalledOnce();
            expect(onSuccessFn).toHaveBeenCalled();
        });

        it('should proceed when element is not inside a form', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const onSuccessFn = vi.fn();
            const descriptor = action('noFormAction').setOnSuccess(onSuccessFn);

            await handleAction(descriptor, element);

            expect(fetchSpy).toHaveBeenCalledOnce();
            expect(onSuccessFn).toHaveBeenCalled();
        });
    });

    describe('file upload via FormData', () => {
        it('should send FormData when args contain a File', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200, data: 'uploaded'}), {status: 200})
            );

            const file = new File(['hello'], 'test.txt', {type: 'text/plain'});
            const descriptor = action('uploadFile', {file, name: 'document'});

            await handleAction(descriptor, element);

            const [, options] = fetchSpy.mock.calls[0];
            expect(options.body).toBeInstanceOf(FormData);
            const formData = options.body as FormData;
            expect(formData.get('file')).toBeInstanceOf(File);
            expect(formData.get('name')).toBe('document');
        });

        it('should send FormData when args contain a Blob', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200, data: 'uploaded'}), {status: 200})
            );

            const blob = new Blob(['binary data'], {type: 'application/octet-stream'});
            const descriptor = action('uploadBlob', {data: blob, label: 'myBlob'});

            await handleAction(descriptor, element);

            const [, options] = fetchSpy.mock.calls[0];
            expect(options.body).toBeInstanceOf(FormData);
            const formData = options.body as FormData;
            expect(formData.get('data')).toBeInstanceOf(Blob);
            expect(formData.get('label')).toBe('myBlob');
        });

        it('should send JSON when args contain no files', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const descriptor = action('noFileAction', {name: 'test', count: 42});

            await handleAction(descriptor, element);

            const [, options] = fetchSpy.mock.calls[0];
            expect(typeof options.body).toBe('string');
            expect(options.headers['Content-Type']).toBe('application/json');
        });

        it('should include ephemeral token in FormData', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const file = new File(['data'], 'doc.pdf', {type: 'application/pdf'});
            const descriptor = action('uploadWithCSRF', {file});

            await handleAction(descriptor, element);

            const [, options] = fetchSpy.mock.calls[0];
            const formData = options.body as FormData;
            expect(formData.get('_csrf_ephemeral_token')).toBe('test-ephemeral-token');
        });

        it('should not include Content-Type header when sending FormData (browser sets boundary)', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const file = new File(['data'], 'photo.jpg', {type: 'image/jpeg'});
            const descriptor = action('uploadFile', {file});

            await handleAction(descriptor, element);

            const [, options] = fetchSpy.mock.calls[0];
            expect(options.headers['Content-Type']).toBeUndefined();
        });
    });

    describe('SSE transport fallback', () => {
        it('should fall back to JSON parsing when response is not SSE content-type', async () => {
            const jsonResponse = {data: {result: 'json-fallback'}, status: 200};
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify(jsonResponse), {
                    status: 200,
                    headers: {'Content-Type': 'application/json'}
                })
            );

            const onProgress = vi.fn();

            const result = await callServerActionDirect('testFallback', [], 'POST', {onProgress});

            expect(onProgress).not.toHaveBeenCalled();
            expect(result.data).toEqual(jsonResponse);
        });

        it('should send Accept: text/event-stream header when onProgress is provided', async () => {
            const sseText = 'event: complete\ndata: {"done":true}\n\n';
            const stream = new ReadableStream({
                start(controller) {
                    controller.enqueue(new TextEncoder().encode(sseText));
                    controller.close();
                }
            });

            fetchSpy.mockResolvedValueOnce(new Response(stream, {
                status: 200,
                headers: {'Content-Type': 'text/event-stream'}
            }));

            const onProgress = vi.fn();

            await callServerActionDirect('testSSEHeaders', [], 'POST', {onProgress});

            const [, options] = fetchSpy.mock.calls[0];
            expect(options.headers['Accept']).toBe('text/event-stream');
        });
    });

    describe('SSE reconnect delay calculation', () => {
        it('should use linear backoff delays when reconnecting SSE streams', async () => {
            const sseTextDrop = 'event: progress\ndata: {"step":1}\n\n';

            function createSSEResponse(sseText: string): Response {
                const stream = new ReadableStream({
                    start(controller) {
                        controller.enqueue(new TextEncoder().encode(sseText));
                        controller.close();
                    }
                });
                return new Response(stream, {
                    status: 200,
                    headers: {'Content-Type': 'text/event-stream'}
                });
            }

            const sseTextComplete = 'event: complete\ndata: "done"\n\n';

            fetchSpy
                .mockResolvedValueOnce(createSSEResponse(sseTextDrop))
                .mockResolvedValueOnce(createSSEResponse(sseTextComplete));

            const onProgress = vi.fn();
            const onDisconnect = vi.fn();
            const onReconnect = vi.fn();

            const result = await callServerActionDirect('testSSE', [], 'POST', {
                onProgress,
                retryStream: {
                    maxReconnects: 3,
                    baseDelay: 1,
                    backoff: 'linear',
                    onDisconnect,
                    onReconnect
                }
            });

            expect(result.data).toBe('done');
            expect(onDisconnect).toHaveBeenCalledOnce();
            expect(onReconnect).toHaveBeenCalledWith(1);
        });

        it('should use exponential backoff delays when specified', async () => {
            const sseTextDrop = 'event: progress\ndata: {"step":1}\n\n';

            function createSSEResponse(sseText: string): Response {
                const stream = new ReadableStream({
                    start(controller) {
                        controller.enqueue(new TextEncoder().encode(sseText));
                        controller.close();
                    }
                });
                return new Response(stream, {
                    status: 200,
                    headers: {'Content-Type': 'text/event-stream'}
                });
            }

            const sseTextComplete = 'event: complete\ndata: "done"\n\n';

            fetchSpy
                .mockResolvedValueOnce(createSSEResponse(sseTextDrop))
                .mockResolvedValueOnce(createSSEResponse(sseTextComplete));

            const onProgress = vi.fn();

            const result = await callServerActionDirect('testSSE', [], 'POST', {
                onProgress,
                retryStream: {
                    maxReconnects: 3,
                    baseDelay: 1,
                    backoff: 'exponential'
                }
            });

            expect(result.data).toBe('done');
            expect(fetchSpy).toHaveBeenCalledTimes(2);
        });
    });

    describe('callServerActionDirect() with suppressHelpers', () => {
        it('should not execute helpers when suppressHelpers is true', async () => {
            const responseData = {
                status: 200,
                data: {result: 'test'},
                _helpers: [{name: 'redirect', args: ['/some-page']}]
            };

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify(responseData), {status: 200})
            );

            const result = await callServerActionDirect('testAction', [], 'POST', {
                suppressHelpers: true
            });

            expect(result.data).toEqual({result: 'test'});
            expect(result.helpers).toEqual([{name: 'redirect', args: ['/some-page']}]);
        });

        it('should return response data without helpers field when no helpers in response', async () => {
            const responseData = {status: 200, data: {value: 42}};

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify(responseData), {status: 200})
            );

            const result = await callServerActionDirect('testAction', [], 'POST');

            expect(result.data).toEqual({value: 42});
            expect(result.status).toBe(200);
        });
    });

    describe('callServerActionDirect() with SSE progress', () => {
        function createSSEResponse(sseText: string): Response {
            const stream = new ReadableStream({
                start(controller) {
                    controller.enqueue(new TextEncoder().encode(sseText));
                    controller.close();
                }
            });
            return new Response(stream, {
                status: 200,
                headers: {'Content-Type': 'text/event-stream'}
            });
        }

        it('should use SSE transport when onProgress is provided', async () => {
            const sseText =
                'event: progress\ndata: {"step":1}\n\n' +
                'event: complete\ndata: {"result":"done"}\n\n';

            fetchSpy.mockResolvedValueOnce(createSSEResponse(sseText));

            const progressEvents: unknown[] = [];
            const onProgress = vi.fn((data: unknown) => {
                progressEvents.push(data);
            });

            const result = await callServerActionDirect('sseAction', ['arg1'], 'POST', {
                onProgress
            });

            expect(result.data).toEqual({result: 'done'});
            expect(result.status).toBe(200);
            expect(onProgress).toHaveBeenCalledOnce();
            expect(progressEvents[0]).toEqual({step: 1});
        });

        it('should use SSE with retryStream when both onProgress and retryStream are provided', async () => {
            const sseText1 = 'event: progress\ndata: {"step":1}\n\n';
            const sseText2 = 'event: complete\ndata: "ok"\n\n';

            fetchSpy
                .mockResolvedValueOnce(createSSEResponse(sseText1))
                .mockResolvedValueOnce(createSSEResponse(sseText2));

            const onProgress = vi.fn();

            const result = await callServerActionDirect('sseAction', [], 'POST', {
                onProgress,
                retryStream: {maxReconnects: 2, baseDelay: 1}
            });

            expect(result.data).toBe('ok');
            expect(fetchSpy).toHaveBeenCalledTimes(2);
        });

        it('should reject on SSE error from callServerActionDirect', async () => {
            const sseText = 'event: error\ndata: {"message":"Server exploded"}\n\n';

            fetchSpy.mockResolvedValueOnce(createSSEResponse(sseText));

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('sseAction', [], 'POST', {onProgress})
            ).rejects.toMatchObject({
                message: 'Server exploded'
            });
        });
    });

    describe('calculateRetryDelay behaviour through handleAction', () => {
        it('should cap exponential backoff at max retry delay', async () => {
            vi.useFakeTimers();

            const delays: number[] = [];
            let lastCallTime = Date.now();

            fetchSpy.mockImplementation(async () => {
                const now = Date.now();
                if (fetchSpy.mock.calls.length > 1) {
                    delays.push(now - lastCallTime);
                }
                lastCallTime = now;
                return new Response(JSON.stringify({message: 'Server error'}), {status: 500});
            });

            const descriptor = action('retryAction')
                .setRetry(6, 'exponential')
                .setOnError(() => {});

            const promise = handleAction(descriptor, element);
            const rejection = expect(promise).rejects.toThrow();

            await vi.advanceTimersByTimeAsync(1100);
            await vi.advanceTimersByTimeAsync(2100);
            await vi.advanceTimersByTimeAsync(4100);
            await vi.advanceTimersByTimeAsync(8100);
            await vi.advanceTimersByTimeAsync(16100);

            await rejection;

            expect(fetchSpy).toHaveBeenCalledTimes(6);
            expect(delays[0]).toBeGreaterThanOrEqual(1000);
            for (const d of delays) {
                expect(d).toBeLessThanOrEqual(31000);
            }

            vi.useRealTimers();
        });

        it('should cap linear backoff at max retry delay', async () => {
            vi.useFakeTimers();

            const delays: number[] = [];
            let lastCallTime = Date.now();

            fetchSpy.mockImplementation(async () => {
                const now = Date.now();
                if (fetchSpy.mock.calls.length > 1) {
                    delays.push(now - lastCallTime);
                }
                lastCallTime = now;
                return new Response(JSON.stringify({message: 'Server error'}), {status: 500});
            });

            const descriptor = action('retryAction')
                .setRetry(4, 'linear')
                .setOnError(() => {});

            const promise = handleAction(descriptor, element);
            const rejection = expect(promise).rejects.toThrow();

            await vi.advanceTimersByTimeAsync(1100);
            await vi.advanceTimersByTimeAsync(2100);
            await vi.advanceTimersByTimeAsync(3100);

            await rejection;

            expect(fetchSpy).toHaveBeenCalledTimes(4);
            expect(delays[0]).toBeGreaterThanOrEqual(1000);
            expect(delays[0]).toBeLessThan(1500);
            expect(delays[1]).toBeGreaterThanOrEqual(2000);
            expect(delays[1]).toBeLessThan(2500);
            expect(delays[2]).toBeGreaterThanOrEqual(3000);

            vi.useRealTimers();
        });
    });

    describe('race condition: SSE reconnect interrupted by abort', () => {
        function createSSEResponse(sseText: string): Response {
            const stream = new ReadableStream({
                start(controller) {
                    controller.enqueue(new TextEncoder().encode(sseText));
                    controller.close();
                }
            });
            return new Response(stream, {
                status: 200,
                headers: {'Content-Type': 'text/event-stream'}
            });
        }

        it('should abort SSE reconnection when signal fires during reconnect delay', async () => {
            const controller = new AbortController();

            const sseTextDrop = 'event: progress\ndata: {"step":1}\n\n';

            let fetchCallCount = 0;
            fetchSpy.mockImplementation(async () => {
                fetchCallCount++;
                if (fetchCallCount === 1) {
                    return createSSEResponse(sseTextDrop);
                }
                return createSSEResponse('event: complete\ndata: "ok"\n\n');
            });

            const onProgress = vi.fn();
            const onDisconnect = vi.fn(() => {
                controller.abort();
            });

            await expect(
                callServerActionDirect('sseAction', [], 'POST', {
                    onProgress,
                    signal: controller.signal,
                    retryStream: {
                        maxReconnects: 3,
                        baseDelay: 50,
                        onDisconnect
                    }
                })
            ).rejects.toMatchObject({
                message: 'Request cancelled'
            });

            expect(onDisconnect).toHaveBeenCalledOnce();
            expect(fetchCallCount).toBe(1);
        });

        it('should abort SSE when signal is already aborted', async () => {
            const controller = new AbortController();
            controller.abort();

            fetchSpy.mockImplementation(async (_url: string, init: RequestInit) => {
                if (init.signal?.aborted) {
                    throw new DOMException('The operation was aborted.', 'AbortError');
                }
                return createSSEResponse('event: complete\ndata: "ok"\n\n');
            });

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('sseAction', [], 'POST', {
                    onProgress,
                    signal: controller.signal
                })
            ).rejects.toMatchObject({
                message: 'Request cancelled'
            });
        });
    });

    describe('handleAction with SSE via onProgress descriptor', () => {
        function createSSEResponse(sseText: string): Response {
            const stream = new ReadableStream({
                start(controller) {
                    controller.enqueue(new TextEncoder().encode(sseText));
                    controller.close();
                }
            });
            return new Response(stream, {
                status: 200,
                headers: {'Content-Type': 'text/event-stream'}
            });
        }

        it('should use SSE transport when descriptor has onProgress', async () => {
            const sseText =
                'event: progress\ndata: {"percent":50}\n\n' +
                'event: complete\ndata: {"result":"done"}\n\n';

            fetchSpy.mockResolvedValueOnce(createSSEResponse(sseText));

            const progressEvents: unknown[] = [];
            const onSuccessFn = vi.fn();

            const descriptor = action('sseAction')
                .setOnProgress((data: unknown) => {
                    progressEvents.push(data);
                })
                .setOnSuccess(onSuccessFn);

            await handleAction(descriptor, element);

            expect(progressEvents).toEqual([{percent: 50}]);
            expect(onSuccessFn).toHaveBeenCalled();
            const [, options] = fetchSpy.mock.calls[0];
            expect(options.headers['Accept']).toBe('text/event-stream');
        });

        it('should use SSE with retryStream when descriptor has both onProgress and retryStream', async () => {
            const sseText1 = 'event: progress\ndata: 1\n\n';
            const sseText2 =
                'event: progress\ndata: 2\n\n' +
                'event: complete\ndata: "finished"\n\n';

            fetchSpy
                .mockResolvedValueOnce(createSSEResponse(sseText1))
                .mockResolvedValueOnce(createSSEResponse(sseText2));

            const progressData: unknown[] = [];
            const onSuccessFn = vi.fn();

            const descriptor = action('sseRetryAction')
                .setOnProgress((data: unknown) => {
                    progressData.push(data);
                })
                .setRetryStream({maxReconnects: 3, baseDelay: 1})
                .setOnSuccess(onSuccessFn);

            await handleAction(descriptor, element);

            expect(progressData).toEqual([1, 2]);
            expect(onSuccessFn).toHaveBeenCalled();
            expect(fetchSpy).toHaveBeenCalledTimes(2);
        });
    });

    describe('callServerActionDirect() error handling', () => {
        it('should throw ActionError on server 500 response', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({message: 'Internal Server Error'}), {status: 500})
            );

            await expect(
                callServerActionDirect('failAction', [])
            ).rejects.toMatchObject({
                status: 500,
                message: 'Internal Server Error'
            });
        });

        it('should throw ActionError when response is not valid JSON', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response('not json', {status: 200, headers: {'Content-Type': 'text/plain'}})
            );

            await expect(
                callServerActionDirect('badResponse', [])
            ).rejects.toMatchObject({
                message: 'Failed to parse server response'
            });
        });

        it('should throw ActionError with validation errors on 422', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({
                    message: 'Validation failed',
                    errors: {email: ['Required']}
                }), {status: 422})
            );

            await expect(
                callServerActionDirect('validateAction', [])
            ).rejects.toMatchObject({
                status: 422,
                message: 'Validation failed',
                validationErrors: {email: ['Required']}
            });
        });

        it('should handle network error as ActionError', async () => {
            fetchSpy.mockRejectedValueOnce(new TypeError('Failed to fetch'));

            await expect(
                callServerActionDirect('networkFail', [])
            ).rejects.toThrow();
        });
    });

    describe('callServerActionDirect() method and args', () => {
        it('should default to POST method', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200, data: null}), {status: 200})
            );

            await callServerActionDirect('testDefault', []);

            const [, options] = fetchSpy.mock.calls[0];
            expect(options.method).toBe('POST');
        });

        it('should use specified HTTP method', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200, data: null}), {status: 200})
            );

            await callServerActionDirect('testDelete', [], 'DELETE');

            const [, options] = fetchSpy.mock.calls[0];
            expect(options.method).toBe('DELETE');
        });

        it('should pass args in request body', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200, data: null}), {status: 200})
            );

            await callServerActionDirect('testArgs', ['hello', 42]);

            const [, options] = fetchSpy.mock.calls[0];
            const body = JSON.parse(options.body as string);
            expect(body.args).toEqual({'0': 'hello', '1': 42});
        });
    });

    describe('request timeout', () => {
        it('should abort request after timeout and throw timeout error via callServerActionDirect', async () => {
            vi.useFakeTimers();

            fetchSpy.mockImplementation((_url: string, init: RequestInit) => {
                return new Promise((_resolve, reject) => {
                    if (init.signal) {
                        init.signal.addEventListener('abort', () => {
                            reject(new DOMException('The operation was aborted.', 'AbortError'));
                        });
                    }
                });
            });

            const promise = callServerActionDirect('slowAction', [], 'POST', {timeout: 5000});
            const rejection = expect(promise).rejects.toMatchObject({
                status: 408,
                message: 'Request timeout'
            });

            await vi.advanceTimersByTimeAsync(5100);

            await rejection;

            vi.useRealTimers();
        });
    });

    describe('SSE error response parsing', () => {
        it('should handle non-JSON error response from SSE endpoint', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response('Unauthorized', {status: 401, headers: {'Content-Type': 'text/plain'}})
            );

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('sseAction', [], 'POST', {onProgress})
            ).rejects.toMatchObject({
                status: 401,
                message: expect.stringContaining('401')
            });
        });

        it('should handle JSON error response from SSE endpoint', async () => {
            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({message: 'Forbidden', error: 'not_allowed'}), {status: 403})
            );

            const onProgress = vi.fn();

            await expect(
                callServerActionDirect('sseAction', [], 'POST', {onProgress})
            ).rejects.toMatchObject({
                status: 403,
                message: 'Forbidden'
            });
        });
    });

    describe('element-level CSRF tokens', () => {
        it('should use data-attribute CSRF tokens when present on element', async () => {
            element.setAttribute('data-csrf-action-token', 'element-action-token');
            element.setAttribute('data-csrf-ephemeral-token', 'element-ephemeral-token');

            fetchSpy.mockResolvedValueOnce(
                new Response(JSON.stringify({status: 200}), {status: 200})
            );

            const descriptor = action('csrfAction');
            await handleAction(descriptor, element);

            const [, options] = fetchSpy.mock.calls[0];
            expect(options.headers['X-CSRF-Action-Token']).toBe('element-action-token');

            const body = JSON.parse(options.body as string);
            expect(body._csrf_ephemeral_token).toBe('element-ephemeral-token');
        });
    });
});
