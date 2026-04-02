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

/// <reference path="../../../shared/types/index.d.ts" />

import { waitForPiko } from '../../../shared/utils';

/**
 * Registers the `showToast` helper with the Piko framework.
 *
 * The helper dispatches a `pk-show-toast` custom event with the message,
 * variant (defaults to `"info"`), and duration (defaults to 5000 ms)
 * provided as positional arguments.
 *
 * @param pk - The Piko namespace instance.
 */
function registerHelpers(pk: typeof window.piko): void {
    pk.registerHelper('showToast', (_triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
        const message = args[0];
        const variant = args[1] ?? 'info';
        const durationStr = args[2] ?? '5000';
        const duration = parseInt(durationStr, 10);

        if (!message) {
            console.warn('showToast helper called without a message.');
            return;
        }

        document.dispatchEvent(new CustomEvent('pk-show-toast', {
            detail: {
                message,
                variant,
                duration,
            }
        }));
    });

    console.debug('[piko/toasts] Extension loaded - helpers: showToast');
}

waitForPiko('toasts')
    .then(registerHelpers)
    .catch((err) => console.error(err.message));

export {};
