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

import {createActionError} from '@/pk/action';

/** Callbacks for SSE stream events. */
export interface SSEStreamCallbacks {
    /** Called for each non-terminal event (everything except complete/error). */
    onEvent: (data: unknown, eventType: string) => void;
    /** Called when an event with an id: field is received, used for Last-Event-ID tracking. */
    onEventId?: (id: string) => void;
}

/** SSE block delimiter. */
const BLOCK_DELIMITER = '\n\n';

/** Default event type when no event: line is present. */
const DEFAULT_EVENT_TYPE = 'message';

/** Terminal event type signalling successful completion. */
const COMPLETE_EVENT_TYPE = 'complete';

/** Terminal event type signalling an error. */
const ERROR_EVENT_TYPE = 'error';

/** Prefix for SSE event type lines. */
const EVENT_LINE_PREFIX = 'event: ';

/** Prefix for SSE data lines. */
const DATA_LINE_PREFIX = 'data: ';

/** Prefix for SSE event ID lines. */
const ID_LINE_PREFIX = 'id: ';

/**
 * Parses a single SSE text block into an event type and data.
 *
 * If the data field cannot be parsed as JSON, the raw string is kept as-is.
 *
 * @param block - The raw text block delimited by double newlines.
 * @returns The parsed event with type, data, and optional id, or null for empty/comment blocks.
 */
function parseSSEBlock(block: string): { eventType: string; data: unknown; id?: string } | null {
    const trimmed = block.trim();
    if (!trimmed || trimmed.startsWith(':')) {
        return null;
    }

    let eventType = '';
    let rawData = '';
    let id: string | undefined;

    for (const line of trimmed.split('\n')) {
        if (line.startsWith(EVENT_LINE_PREFIX)) {
            eventType = line.substring(EVENT_LINE_PREFIX.length);
        } else if (line.startsWith(DATA_LINE_PREFIX)) {
            rawData = line.substring(DATA_LINE_PREFIX.length);
        } else if (line.startsWith(ID_LINE_PREFIX)) {
            id = line.substring(ID_LINE_PREFIX.length);
        }
    }

    if (!eventType && !rawData) {
        return null;
    }

    if (!eventType) {
        eventType = DEFAULT_EVENT_TYPE;
    }

    let data: unknown = rawData;
    try {
        data = JSON.parse(rawData);
    } catch {}

    return {eventType, data, id};
}

/** Holds the data from a terminal "complete" SSE event. */
interface SSECompleteResult {
    /** The parsed data payload from the complete event. */
    completeData: unknown;
}

/**
 * Processes a single SSE block by routing it to the appropriate callback.
 *
 * Detects terminal events (complete/error) and either returns a complete result
 * or throws an ActionError for error events.
 *
 * @param block - The raw SSE text block to process.
 * @param callbacks - The stream event callbacks.
 * @returns The complete result if this was a terminal complete event, or null otherwise.
 * @throws ActionError when the block contains an error event.
 */
function processSSEBlock(
    block: string,
    callbacks: SSEStreamCallbacks,
): SSECompleteResult | null {
    const parsed = parseSSEBlock(block);
    if (!parsed) {
        return null;
    }

    if (parsed.eventType === COMPLETE_EVENT_TYPE) {
        return {completeData: parsed.data};
    }

    if (parsed.eventType === ERROR_EVENT_TYPE) {
        const errorData = parsed.data as Record<string, unknown> | undefined;
        const message = typeof errorData?.message === 'string'
            ? errorData.message
            : 'SSE stream error';
        throw createActionError(0, message, undefined, parsed.data);
    }

    callbacks.onEvent(parsed.data, parsed.eventType);

    if (parsed.id && callbacks.onEventId) {
        callbacks.onEventId(parsed.id);
    }

    return null;
}

/**
 * Consumes the readable stream, processes SSE blocks, and collects the
 * completion result.
 *
 * @param reader - The locked stream reader to consume chunks from.
 * @param callbacks - The stream event callbacks.
 * @returns The complete event data and whether a complete event was received.
 * @throws ActionError when an error event is encountered in the stream.
 */
async function consumeSSEStream(
    reader: ReadableStreamDefaultReader<Uint8Array>,
    callbacks: SSEStreamCallbacks,
): Promise<{completeData: unknown; receivedComplete: boolean}> {
    const decoder = new TextDecoder();
    let buffer = '';
    let completeData: unknown = undefined;
    let receivedComplete = false;

    for (;;) {
        const {done, value} = await reader.read();
        if (done) {
            break;
        }

        buffer += decoder.decode(value, {stream: true});
        const parts = buffer.split(BLOCK_DELIMITER);
        buffer = parts.pop() ?? '';

        for (const part of parts) {
            const result = processSSEBlock(part, callbacks);
            if (!result) { continue; }
            completeData = result.completeData;
            receivedComplete = true;
        }
    }

    return {completeData, receivedComplete};
}

/**
 * Reads an SSE stream, routing events to callbacks.
 *
 * The returned promise resolves with the data from the "complete" event,
 * or rejects with an ActionError on "error" events, stream failures,
 * or if the stream ends without a "complete" event. The reader lock is
 * released in the finally block even if it was already released.
 *
 * @param body - The ReadableStream from the fetch response.
 * @param callbacks - The stream event callbacks.
 * @param _signal - Optional AbortSignal for cancellation.
 * @returns The data payload from the terminal complete event.
 * @throws ActionError when the stream contains an error event or ends unexpectedly.
 */
export async function readSSEStream(
    body: ReadableStream<Uint8Array>,
    callbacks: SSEStreamCallbacks,
    _signal?: AbortSignal,
): Promise<unknown> {
    const reader = body.getReader();

    try {
        const {completeData, receivedComplete} = await consumeSSEStream(reader, callbacks);

        if (!receivedComplete) {
            throw createActionError(0, 'SSE stream ended without completion');
        }

        return completeData;
    } catch (error) {
        if (error instanceof DOMException && error.name === 'AbortError') {
            throw createActionError(0, 'Request cancelled');
        }
        if (error !== null && typeof error === 'object' && 'status' in error) {
            throw error;
        }
        const message = error instanceof Error ? error.message : 'SSE connection lost';
        throw createActionError(0, message);
    } finally {
        try {
            reader.releaseLock();
        } catch {}
    }
}
