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

import {type HTTPOperations, browserHTTPOperations} from '@/core/BrowserAPIs';

/**
 * Result tuple from a fetch operation: [success, responseText].
 */
export type FetchResult = [ok: boolean, text: string | null];

/**
 * Options for FetchClient requests.
 */
export interface FetchClientOptions {
    /** Callback for progress updates during download. */
    onProgress?: (loaded: number, total: number) => void;
}

/**
 * HTTP client with progress tracking and abort capability.
 */
export interface FetchClient {
    /** Performs a GET request and returns the response as text. */
    get(url: string, options?: FetchClientOptions): Promise<FetchResult>;

    /** Performs a POST request with optional body and headers. */
    post(url: string, body?: BodyInit, headers?: Record<string, string>): Promise<Response>;

    /** Aborts any in-progress request. */
    abort(): void;

    /** Returns the current AbortController for external management. */
    getController(): AbortController | null;
}

/**
 * Reads a response body with progress reporting via chunked streaming.
 *
 * Falls back to standard `response.text()` when the Content-Length header
 * is missing or the body stream is unavailable.
 *
 * @param response - The fetch Response to read.
 * @param onProgress - Callback invoked with bytes loaded and total bytes.
 * @returns The decoded response text.
 */
async function readWithProgress(
    response: Response,
    onProgress: (loaded: number, total: number) => void
): Promise<string> {
    const lenHeader = response.headers.get('Content-Length');
    if (!lenHeader) {
        return response.text();
    }

    const totalSize = parseInt(lenHeader, 10);
    if (isNaN(totalSize) || totalSize <= 0) {
        return response.text();
    }

    const reader = response.body?.getReader();
    if (!reader) {
        return response.text();
    }

    let loaded = 0;
    const chunks: Uint8Array[] = [];

    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- loop control
    while (true) {
        const {done, value} = await reader.read() as {done: boolean; value: Uint8Array | undefined};
        if (done) {
            break;
        }
        if (!value) {
            continue;
        }
        chunks.push(value);
        loaded += value.length;
        onProgress(loaded, totalSize);
    }

    const combined = new Uint8Array(loaded);
    let position = 0;
    for (const chunk of chunks) {
        combined.set(chunk, position);
        position += chunk.length;
    }

    return new TextDecoder('utf-8').decode(combined);
}

/**
 * Dependencies for creating a FetchClient.
 */
export interface FetchClientDependencies {
    /** HTTP operations implementation. Defaults to browser fetch. */
    http?: HTTPOperations;
}

/**
 * Creates a FetchClient instance with optional dependency injection.
 *
 * @param deps - Optional dependencies for testing or customisation.
 * @returns A new FetchClient instance.
 */
export function createFetchClient(deps: FetchClientDependencies = {}): FetchClient {
    const http = deps.http ?? browserHTTPOperations;
    let controller: AbortController | null = null;

    return {
        /**
         * Performs a GET request and returns the response as text.
         *
         * @param url - The URL to fetch.
         * @param options - The optional fetch client options.
         * @returns A tuple of [success, responseText].
         */
        async get(url: string, options: FetchClientOptions = {}): Promise<FetchResult> {
            try {
                controller = new AbortController();
                const response = await http.fetch(url, {
                    method: 'GET',
                    credentials: 'same-origin',
                    signal: controller.signal
                });

                if (!response.ok) {
                    return [false, null];
                }

                let text: string;
                if (options.onProgress) {
                    text = await readWithProgress(response, options.onProgress);
                } else {
                    text = await response.text();
                }

                return [true, text];
            } catch (error) {
                if (error instanceof DOMException && error.name === 'AbortError') {
                    throw error;
                }
                console.error('FetchClient.get error:', error);
                return [false, null];
            }
        },

        /**
         * Performs a POST request with optional body and headers.
         *
         * @param url - The URL to fetch.
         * @param body - The optional request body.
         * @param headers - The optional request headers.
         * @returns The fetch Response.
         */
        async post(url: string, body?: BodyInit, headers: Record<string, string> = {}): Promise<Response> {
            controller = new AbortController();
            return http.fetch(url, {
                method: 'POST',
                headers,
                body,
                credentials: 'same-origin',
                signal: controller.signal
            });
        },

        /** Aborts any in-progress request. */
        abort() {
            controller?.abort();
            controller = null;
        },

        /** Returns the current AbortController for external management. */
        getController() {
            return controller;
        }
    };
}
