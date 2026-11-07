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

/**
 * Process lifecycle and timing helpers, kept free of VS Code so they can be
 * unit tested.
 */

/**
 * The subset of a child process needed to wait for and force its exit.
 */
export interface ExitableProcess {
    /** The exit code once the process has exited, otherwise null. */
    readonly exitCode: number | null;
    /** The terminating signal once the process has exited by signal, otherwise null. */
    readonly signalCode: NodeJS.Signals | null;
    /** Registers a one-shot exit listener. */
    once(event: 'exit', listener: () => void): unknown;
    /** Removes an exit listener. */
    removeListener(event: 'exit', listener: () => void): unknown;
    /** Sends a signal to the process. */
    kill(signal?: NodeJS.Signals): boolean;
}

/**
 * Reports whether a process has already exited.
 *
 * @param child - The process.
 * @returns True once the process has an exit code or signal.
 */
export function hasExited(child: ExitableProcess): boolean {
    return child.exitCode !== null || child.signalCode !== null;
}

/**
 * Waits for a process to exit, up to a timeout.
 *
 * @param child - The process to wait for.
 * @param timeoutMs - The maximum time to wait.
 * @returns True when the process exited within the timeout.
 */
export function waitForExit(child: ExitableProcess, timeoutMs: number): Promise<boolean> {
    if (hasExited(child)) {
        return Promise.resolve(true);
    }
    return new Promise(resolve => {
        const onExit = (): void => {
            clearTimeout(timer);
            resolve(true);
        };
        const timer = setTimeout(() => {
            child.removeListener('exit', onExit);
            resolve(hasExited(child));
        }, timeoutMs);
        child.once('exit', onExit);
    });
}

/**
 * Stops a process gracefully: waits for it to exit on its own, then sends
 * SIGTERM, and only sends SIGKILL if it is still running after a second wait.
 *
 * @param child - The process to stop.
 * @param graceMs - How long to wait at each stage.
 * @returns How the process ended: `exited`, `terminated` or `killed`.
 */
export async function stopProcess(child: ExitableProcess, graceMs: number): Promise<'exited' | 'terminated' | 'killed'> {
    if (await waitForExit(child, graceMs)) {
        return 'exited';
    }
    child.kill('SIGTERM');
    if (await waitForExit(child, graceMs)) {
        return 'terminated';
    }
    child.kill('SIGKILL');
    return 'killed';
}

/**
 * Wraps a function so bursts of calls run it once, after a quiet period.
 *
 * @param fn - The function to run.
 * @param delayMs - The quiet period.
 * @returns The debounced function and a cancel hook.
 */
export function debounce<Args extends unknown[]>(
    fn: (...args: Args) => void,
    delayMs: number
): { call: (...args: Args) => void; cancel: () => void } {
    let timer: ReturnType<typeof setTimeout> | undefined;
    return {
        call: (...args: Args) => {
            if (timer !== undefined) {
                clearTimeout(timer);
            }
            timer = setTimeout(() => {
                timer = undefined;
                fn(...args);
            }, delayMs);
        },
        cancel: () => {
            if (timer !== undefined) {
                clearTimeout(timer);
                timer = undefined;
            }
        },
    };
}
