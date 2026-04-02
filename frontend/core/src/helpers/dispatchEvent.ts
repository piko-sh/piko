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
 * Dispatches a custom event on the window object. Useful for triggering
 * refreshes or other actions from server responses.
 *
 * @param _element - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is the event name to dispatch.
 * @returns Nothing.
 */
const dispatchEventHelper: PPHelper = (_element: HTMLElement, _event: Event, ...args: string[]) => {
    const eventName = args[0];

    if (!eventName) {
        console.error("The 'dispatchEvent' helper requires an event name as its first argument.");
        return;
    }

    window.dispatchEvent(new CustomEvent(eventName));
};

RegisterHelper('dispatchEvent', dispatchEventHelper);
