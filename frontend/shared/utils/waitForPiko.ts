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

/// <reference path="../types/index.d.ts" />

/** Maximum time in milliseconds to wait for the framework to become ready. */
const MAX_WAIT_MS = 5000;
/** Interval in milliseconds between readiness polls. */
const POLL_INTERVAL_MS = 10;

/**
 * Waits for the piko namespace to be available and the framework to be ready.
 *
 * Polls the global `window.piko` object at a fixed interval and resolves once
 * the framework signals readiness. Times out after a configurable maximum wait.
 *
 * @param extensionName - The name of the extension, used in error messages.
 * @returns A promise that resolves with the piko namespace once ready.
 * @throws {Error} When piko does not become available within the timeout.
 */
export function waitForPiko(extensionName: string): Promise<typeof window.piko> {
    return new Promise((resolve, reject) => {
        const startTime = Date.now();

        const check = () => {
            if (typeof window !== 'undefined' && window.piko) {
                window.piko.ready(() => resolve(window.piko));
                return;
            }

            if (Date.now() - startTime > MAX_WAIT_MS) {
                reject(new Error(`[piko/${extensionName}] Timed out waiting for piko core.`));
                return;
            }

            setTimeout(check, POLL_INTERVAL_MS);
        };

        check();
    });
}

