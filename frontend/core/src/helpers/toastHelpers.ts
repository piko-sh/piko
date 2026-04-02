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

import {type PPHelper, RegisterHelper} from '@/core';

/**
 * Displays a toast notification by dispatching a "pk-show-toast" custom event
 * on the document. The variant defaults to "info" and the duration defaults
 * to 5000 milliseconds.
 *
 * @param _triggerElement - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is the message, the second is the variant, and the third is the duration in milliseconds.
 * @returns Nothing.
 */
const showToastHelper: PPHelper = (_triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
    const message = args[0];
    const variant = args[1] ?? 'info';
    const durationStr = args[2] ?? '5000';
    const duration = parseInt(durationStr);

    if (!message) {
        console.warn("showToast helper called without a message.");
        return;
    }

    document.dispatchEvent(new CustomEvent('pk-show-toast', {
        detail: {
            message,
            variant,
            duration,
        }
    }));
};

RegisterHelper('showToast', showToastHelper);
