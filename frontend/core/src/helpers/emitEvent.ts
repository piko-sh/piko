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
 * Emits a bubbling custom event from the triggering element. The event is
 * composed so it crosses shadow DOM boundaries.
 *
 * @param element - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic arguments; the first is the event name, the second is optional detail data.
 * @returns Nothing.
 */
const emitEventHelper: PPHelper = (element: HTMLElement, _event: Event, ...args: unknown[]) => {
    const eventName = args[0];
    if (!eventName || typeof eventName !== 'string') {
        console.error("The 'emitEvent' helper requires a non-empty event name string as its first argument.", {
            triggeringElement: element
        });
        return;
    }

    const finalOptions: CustomEventInit = {
        bubbles: true,
        composed: true,
        detail: args[1]
    };

    element.dispatchEvent(new CustomEvent(eventName, finalOptions));
};

RegisterHelper('emitEvent', emitEventHelper);
