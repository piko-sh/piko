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
 * Creates a debounced version of a function that delays execution
 * until after the specified milliseconds have elapsed since the last call.
 *
 * @param handler - Function to debounce.
 * @param ms - Delay in milliseconds.
 * @returns Debounced function.
 */
export function debounce<T extends (...args: unknown[]) => unknown>(
    handler: T,
    ms: number
): (...args: Parameters<T>) => void {
    let timeoutId: ReturnType<typeof setTimeout>;

    return (...args: Parameters<T>) => {
        clearTimeout(timeoutId);
        timeoutId = setTimeout(() => handler(...args), ms);
    };
}

/**
 * Creates a throttled version of a function that only executes
 * at most once per specified milliseconds.
 *
 * @param handler - Function to throttle.
 * @param ms - Minimum interval in milliseconds.
 * @returns Throttled function.
 */
export function throttle<T extends (...args: unknown[]) => unknown>(
    handler: T,
    ms: number
): (...args: Parameters<T>) => void {
    let lastCall = 0;

    return (...args: Parameters<T>) => {
        const now = Date.now();
        if (now - lastCall >= ms) {
            lastCall = now;
            handler(...args);
        }
    };
}
