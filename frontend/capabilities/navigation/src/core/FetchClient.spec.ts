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

import {describe, it, expect, vi, beforeEach} from 'vitest';

import {type HTTPOperations} from '@/core/BrowserAPIs';
import {createFetchClient, type FetchClient} from '@/core/FetchClient';

function mockResponse(options: {
    ok?: boolean;
    status?: number;
    text?: string;
    headers?: Record<string, string>;
    body?: ReadableStream<Uint8Array> | null;
}): Response {
    const {
        ok = true,
        status = ok ? 200 : 500,
        text = '',
        headers = {},
        body = null
    } = options;

    return {
        ok,
        status,
        headers: new Headers(headers),
        text: vi.fn().mockResolvedValue(text),
        body,
        json: vi.fn(),
        clone: vi.fn(),
        bodyUsed: false,
        arrayBuffer: vi.fn(),
        blob: vi.fn(),
        formData: vi.fn(),
        redirected: false,
        statusText: ok ? 'OK' : 'Internal Server Error',
        type: 'basic' as ResponseType,
        url: ''
    } as unknown as Response;
}

function createChunkedStream(chunks: Uint8Array[]): ReadableStream<Uint8Array> {
    return new ReadableStream({
        start(controller) {
            for (const chunk of chunks) {
                controller.enqueue(chunk);
            }
            controller.close();
        }
    });
}

describe('FetchClient', () => {
    let mockHttp: HTTPOperations;
    let client: FetchClient;

    beforeEach(() => {
        mockHttp = {
            fetch: vi.fn()
        };
        client = createFetchClient({http: mockHttp});
    });

    describe('get()', () => {
        it('returns [true, text] on successful response', async () => {
            const response = mockResponse({ok: true, text: 'hello world'});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const result = await client.get('/api/data');

            expect(result).toEqual([true, 'hello world']);
            expect(mockHttp.fetch).toHaveBeenCalledWith('/api/data', {
                method: 'GET',
                credentials: 'same-origin',
                signal: expect.any(AbortSignal)
            });
        });

        it('returns [false, null] on non-ok response', async () => {
            const response = mockResponse({ok: false, status: 404});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const result = await client.get('/api/missing');

            expect(result).toEqual([false, null]);
            expect(response.text).not.toHaveBeenCalled();
        });

        it('returns [false, null] on network error', async () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            vi.mocked(mockHttp.fetch).mockRejectedValue(new TypeError('Failed to fetch'));

            const result = await client.get('/api/data');

            expect(result).toEqual([false, null]);
            expect(consoleSpy).toHaveBeenCalledWith('FetchClient.get error:', expect.any(TypeError));
            consoleSpy.mockRestore();
        });

        it('re-throws AbortError instead of catching it', async () => {
            const abortError = new DOMException('The operation was aborted.', 'AbortError');
            vi.mocked(mockHttp.fetch).mockRejectedValue(abortError);

            await expect(client.get('/api/data')).rejects.toThrow('The operation was aborted.');
        });

        it('returns [false, null] for non-abort DOMException', async () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const otherError = new DOMException('Some other error', 'NotAllowedError');
            vi.mocked(mockHttp.fetch).mockRejectedValue(otherError);

            const result = await client.get('/api/data');

            expect(result).toEqual([false, null]);
            expect(consoleSpy).toHaveBeenCalled();
            consoleSpy.mockRestore();
        });

        it('returns [false, null] for non-DOMException error objects', async () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            vi.mocked(mockHttp.fetch).mockRejectedValue('string error');

            const result = await client.get('/api/data');

            expect(result).toEqual([false, null]);
            consoleSpy.mockRestore();
        });

        it('sets the abort controller before each request', async () => {
            const response = mockResponse({ok: true, text: 'ok'});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            expect(client.getController()).toBeNull();
            const promise = client.get('/api/data');
            expect(client.getController()).toBeInstanceOf(AbortController);
            await promise;
        });
    });

    describe('get() with progress', () => {
        it('streams body and reports progress when Content-Length is present', async () => {
            const encoder = new TextEncoder();
            const chunk1 = encoder.encode('Hello');
            const chunk2 = encoder.encode(' World');
            const totalLength = chunk1.length + chunk2.length;

            const stream = createChunkedStream([chunk1, chunk2]);
            const response = mockResponse({
                ok: true,
                headers: {'Content-Length': String(totalLength)},
                body: stream
            });
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const onProgress = vi.fn();
            const result = await client.get('/api/large', {onProgress});

            expect(result).toEqual([true, 'Hello World']);
            expect(onProgress).toHaveBeenCalledTimes(2);
            expect(onProgress).toHaveBeenCalledWith(chunk1.length, totalLength);
            expect(onProgress).toHaveBeenCalledWith(totalLength, totalLength);
            expect(response.text).not.toHaveBeenCalled();
        });

        it('falls back to response.text() when Content-Length header is missing', async () => {
            const response = mockResponse({ok: true, text: 'fallback text'});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const onProgress = vi.fn();
            const result = await client.get('/api/data', {onProgress});

            expect(result).toEqual([true, 'fallback text']);
            expect(response.text).toHaveBeenCalled();
            expect(onProgress).not.toHaveBeenCalled();
        });

        it('falls back to response.text() when Content-Length is zero', async () => {
            const response = mockResponse({
                ok: true,
                text: 'body text',
                headers: {'Content-Length': '0'}
            });
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const onProgress = vi.fn();
            const result = await client.get('/api/data', {onProgress});

            expect(result).toEqual([true, 'body text']);
            expect(response.text).toHaveBeenCalled();
        });

        it('falls back to response.text() when Content-Length is not a number', async () => {
            const response = mockResponse({
                ok: true,
                text: 'body text',
                headers: {'Content-Length': 'not-a-number'}
            });
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const onProgress = vi.fn();
            const result = await client.get('/api/data', {onProgress});

            expect(result).toEqual([true, 'body text']);
            expect(response.text).toHaveBeenCalled();
        });

        it('falls back to response.text() when Content-Length is negative', async () => {
            const response = mockResponse({
                ok: true,
                text: 'body text',
                headers: {'Content-Length': '-5'}
            });
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const onProgress = vi.fn();
            const result = await client.get('/api/data', {onProgress});

            expect(result).toEqual([true, 'body text']);
            expect(response.text).toHaveBeenCalled();
        });

        it('falls back to response.text() when response body is null', async () => {
            const response = mockResponse({
                ok: true,
                text: 'body text',
                headers: {'Content-Length': '100'},
                body: null
            });
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const onProgress = vi.fn();
            const result = await client.get('/api/data', {onProgress});

            expect(result).toEqual([true, 'body text']);
            expect(response.text).toHaveBeenCalled();
        });

        it('handles empty chunks during streaming', async () => {
            const encoder = new TextEncoder();
            const chunk = encoder.encode('data');

            let readCount = 0;
            const stream = new ReadableStream<Uint8Array>({
                pull(controller) {
                    readCount++;
                    if (readCount === 1) {
                        controller.enqueue(chunk);
                    } else {
                        controller.close();
                    }
                }
            });

            const response = mockResponse({
                ok: true,
                headers: {'Content-Length': String(chunk.length)},
                body: stream
            });
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const onProgress = vi.fn();
            const result = await client.get('/api/data', {onProgress});

            expect(result).toEqual([true, 'data']);
            expect(onProgress).toHaveBeenCalledWith(chunk.length, chunk.length);
        });

        it('does not call onProgress when no progress callback is provided', async () => {
            const response = mockResponse({ok: true, text: 'no progress'});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const result = await client.get('/api/data');

            expect(result).toEqual([true, 'no progress']);
            expect(response.text).toHaveBeenCalled();
        });
    });

    describe('post()', () => {
        it('sends a POST request and returns the response', async () => {
            const response = mockResponse({ok: true, text: '{"id": 1}'});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const result = await client.post('/api/create', 'body-data');

            expect(result).toBe(response);
            expect(mockHttp.fetch).toHaveBeenCalledWith('/api/create', {
                method: 'POST',
                headers: {},
                body: 'body-data',
                credentials: 'same-origin',
                signal: expect.any(AbortSignal)
            });
        });

        it('sends custom headers with the request', async () => {
            const response = mockResponse({ok: true});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            const headers = {'Content-Type': 'application/json', 'X-Custom': 'value'};
            await client.post('/api/create', '{"data": true}', headers);

            expect(mockHttp.fetch).toHaveBeenCalledWith('/api/create', {
                method: 'POST',
                headers,
                body: '{"data": true}',
                credentials: 'same-origin',
                signal: expect.any(AbortSignal)
            });
        });

        it('sends POST with no body when body is omitted', async () => {
            const response = mockResponse({ok: true});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            await client.post('/api/trigger');

            expect(mockHttp.fetch).toHaveBeenCalledWith('/api/trigger', {
                method: 'POST',
                headers: {},
                body: undefined,
                credentials: 'same-origin',
                signal: expect.any(AbortSignal)
            });
        });

        it('propagates errors from the fetch call', async () => {
            vi.mocked(mockHttp.fetch).mockRejectedValue(new Error('Network failure'));

            await expect(client.post('/api/create', 'data')).rejects.toThrow('Network failure');
        });

        it('sets a new controller for each post request', async () => {
            const response = mockResponse({ok: true});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            await client.post('/api/first');
            const firstController = client.getController();

            await client.post('/api/second');
            const secondController = client.getController();

            expect(firstController).toBeInstanceOf(AbortController);
            expect(secondController).toBeInstanceOf(AbortController);
            expect(firstController).not.toBe(secondController);
        });
    });

    describe('abort()', () => {
        it('aborts the current controller and sets it to null', async () => {
            const response = mockResponse({ok: true, text: 'ok'});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            await client.get('/api/data');
            const controller = client.getController();
            expect(controller).toBeInstanceOf(AbortController);

            client.abort();

            expect(client.getController()).toBeNull();
            expect(controller!.signal.aborted).toBe(true);
        });

        it('does nothing when no request is in flight', () => {
            expect(client.getController()).toBeNull();
            client.abort();
            expect(client.getController()).toBeNull();
        });

        it('causes an in-flight get() to throw AbortError', async () => {
            vi.mocked(mockHttp.fetch).mockImplementation((_url, init) => {
                return new Promise((_resolve, reject) => {
                    init?.signal?.addEventListener('abort', () => {
                        reject(new DOMException('The operation was aborted.', 'AbortError'));
                    });
                });
            });

            const promise = client.get('/api/slow');

            client.abort();

            await expect(promise).rejects.toThrow('The operation was aborted.');
        });
    });

    describe('getController()', () => {
        it('returns null before any request', () => {
            expect(client.getController()).toBeNull();
        });

        it('returns the current AbortController during a request', async () => {
            const response = mockResponse({ok: true, text: 'ok'});
            vi.mocked(mockHttp.fetch).mockResolvedValue(response);

            await client.get('/api/data');

            const controller = client.getController();
            expect(controller).toBeInstanceOf(AbortController);
            expect(controller!.signal.aborted).toBe(false);
        });
    });

    describe('createFetchClient()', () => {
        it('creates a client with default dependencies when none provided', () => {
            const defaultClient = createFetchClient();
            expect(defaultClient).toBeDefined();
            expect(defaultClient.get).toBeTypeOf('function');
            expect(defaultClient.post).toBeTypeOf('function');
            expect(defaultClient.abort).toBeTypeOf('function');
            expect(defaultClient.getController).toBeTypeOf('function');
        });

        it('creates a client with empty deps object', () => {
            const emptyDepsClient = createFetchClient({});
            expect(emptyDepsClient).toBeDefined();
            expect(emptyDepsClient.get).toBeTypeOf('function');
        });
    });
});
