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

import {describe, it, expect, vi} from 'vitest';

import {readSSEStream} from '@/core/SSEStreamReader';

function createSSEStream(text: string): ReadableStream<Uint8Array> {
    return new ReadableStream({
        start(controller) {
            controller.enqueue(new TextEncoder().encode(text));
            controller.close();
        }
    });
}

describe('SSEStreamReader', () => {
    describe('readSSEStream()', () => {
        it('resolves with complete event data', async () => {
            const onEvent = vi.fn();
            const stream = createSSEStream(
                'event: progress\ndata: {"step":1}\n\nevent: complete\ndata: {"result":"ok"}\n\n'
            );

            const result = await readSSEStream(stream, {onEvent});

            expect(result).toEqual({result: 'ok'});
            expect(onEvent).toHaveBeenCalledTimes(1);
            expect(onEvent).toHaveBeenCalledWith({step: 1}, 'progress');
        });

        it('uses default "message" event type when no event line', async () => {
            const onEvent = vi.fn();
            const stream = createSSEStream(
                'data: "hello"\n\nevent: complete\ndata: null\n\n'
            );

            await readSSEStream(stream, {onEvent});

            expect(onEvent).toHaveBeenCalledWith('hello', 'message');
        });

        it('ignores comment-only blocks (heartbeats)', async () => {
            const onEvent = vi.fn();
            const stream = createSSEStream(
                ': heartbeat\n\nevent: complete\ndata: true\n\n'
            );

            const result = await readSSEStream(stream, {onEvent});

            expect(onEvent).not.toHaveBeenCalled();
            expect(result).toBe(true);
        });

        it('throws on error event', async () => {
            const onEvent = vi.fn();
            const stream = createSSEStream(
                'event: error\ndata: {"message":"something broke"}\n\n'
            );

            await expect(readSSEStream(stream, {onEvent}))
                .rejects.toMatchObject({message: 'something broke'});
        });

        it('throws when stream ends without complete', async () => {
            const onEvent = vi.fn();
            const stream = createSSEStream(
                'event: progress\ndata: 1\n\n'
            );

            await expect(readSSEStream(stream, {onEvent}))
                .rejects.toMatchObject({message: 'SSE stream ended without completion'});
        });

        it('parses id: field and calls onEventId', async () => {
            const onEvent = vi.fn();
            const onEventId = vi.fn();
            const stream = createSSEStream(
                'id: 42\nevent: update\ndata: "test"\n\nevent: complete\ndata: null\n\n'
            );

            await readSSEStream(stream, {onEvent, onEventId});

            expect(onEventId).toHaveBeenCalledWith('42');
            expect(onEvent).toHaveBeenCalledWith('test', 'update');
        });

        it('does not call onEventId when not provided', async () => {
            const onEvent = vi.fn();
            const stream = createSSEStream(
                'id: 1\nevent: update\ndata: 1\n\nevent: complete\ndata: null\n\n'
            );

            await expect(readSSEStream(stream, {onEvent})).resolves.toBeNull();
        });

        it('handles multiple events with incrementing IDs', async () => {
            const onEvent = vi.fn();
            const onEventId = vi.fn();
            const stream = createSSEStream(
                'id: 1\nevent: update\ndata: "a"\n\n' +
                'id: 2\nevent: update\ndata: "b"\n\n' +
                'id: 3\nevent: update\ndata: "c"\n\n' +
                'event: complete\ndata: null\n\n'
            );

            await readSSEStream(stream, {onEvent, onEventId});

            expect(onEventId).toHaveBeenCalledTimes(3);
            expect(onEventId.mock.calls[0][0]).toBe('1');
            expect(onEventId.mock.calls[1][0]).toBe('2');
            expect(onEventId.mock.calls[2][0]).toBe('3');
        });

        it('ignores empty blocks', async () => {
            const onEvent = vi.fn();
            const stream = createSSEStream(
                '\n\nevent: complete\ndata: "done"\n\n'
            );

            const result = await readSSEStream(stream, {onEvent});

            expect(result).toBe('done');
            expect(onEvent).not.toHaveBeenCalled();
        });

        it('handles data that is not valid JSON as raw string', async () => {
            const onEvent = vi.fn();
            const stream = createSSEStream(
                'data: not-json\n\nevent: complete\ndata: null\n\n'
            );

            await readSSEStream(stream, {onEvent});

            expect(onEvent).toHaveBeenCalledWith('not-json', 'message');
        });
    });
});
