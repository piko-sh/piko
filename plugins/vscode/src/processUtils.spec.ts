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

import {EventEmitter} from 'events';
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {debounce, ExitableProcess, hasExited, stopProcess, waitForExit} from './processUtils';

class FakeProcess extends EventEmitter implements ExitableProcess {
    exitCode: number | null = null;
    signalCode: NodeJS.Signals | null = null;
    readonly signals: Array<NodeJS.Signals | undefined> = [];

    constructor(private readonly exitOn?: NodeJS.Signals) {
        super();
    }

    kill(signal?: NodeJS.Signals): boolean {
        this.signals.push(signal);
        if (signal === this.exitOn) {
            this.exit(null, signal ?? null);
        }
        return true;
    }

    exit(code: number | null, signal: NodeJS.Signals | null = null): void {
        this.exitCode = code;
        this.signalCode = signal;
        this.emit('exit');
    }
}

describe('processUtils', () => {
    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
    });

    it('reports exited processes', () => {
        const child = new FakeProcess();
        expect(hasExited(child)).toBe(false);
        child.exit(0);
        expect(hasExited(child)).toBe(true);
    });

    it('resolves immediately for an exited process', async () => {
        const child = new FakeProcess();
        child.exit(0);

        await expect(waitForExit(child, 1000)).resolves.toBe(true);
    });

    it('resolves true when the process exits in time', async () => {
        const child = new FakeProcess();
        const result = waitForExit(child, 1000);
        child.exit(0);

        await expect(result).resolves.toBe(true);
        expect(child.listenerCount('exit')).toBe(0);
    });

    it('resolves false on timeout and removes its listener', async () => {
        const child = new FakeProcess();
        const result = waitForExit(child, 1000);
        await vi.advanceTimersByTimeAsync(1000);

        await expect(result).resolves.toBe(false);
        expect(child.listenerCount('exit')).toBe(0);
    });

    it('does not signal a process that exits on its own', async () => {
        const child = new FakeProcess();
        const result = stopProcess(child, 1000);
        child.exit(0);

        await expect(result).resolves.toBe('exited');
        expect(child.signals).toEqual([]);
    });

    it('sends SIGTERM after the grace period', async () => {
        const child = new FakeProcess('SIGTERM');
        const result = stopProcess(child, 1000);
        await vi.advanceTimersByTimeAsync(1000);

        await expect(result).resolves.toBe('terminated');
        expect(child.signals).toEqual(['SIGTERM']);
    });

    it('only sends SIGKILL when SIGTERM is ignored', async () => {
        const child = new FakeProcess();
        const result = stopProcess(child, 1000);
        await vi.advanceTimersByTimeAsync(2000);

        await expect(result).resolves.toBe('killed');
        expect(child.signals).toEqual(['SIGTERM', 'SIGKILL']);
    });

    it('debounces bursts into one call with the latest arguments', () => {
        const fn = vi.fn();
        const debounced = debounce(fn, 100);

        debounced.call(1);
        debounced.call(2);
        vi.advanceTimersByTime(99);
        expect(fn).not.toHaveBeenCalled();
        vi.advanceTimersByTime(1);
        expect(fn).toHaveBeenCalledTimes(1);
        expect(fn).toHaveBeenCalledWith(2);
    });

    it('cancels a pending debounced call', () => {
        const fn = vi.fn();
        const debounced = debounce(fn, 100);

        debounced.call();
        debounced.cancel();
        debounced.cancel();
        vi.advanceTimersByTime(200);
        expect(fn).not.toHaveBeenCalled();
    });
});
