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
 * Redirects the browser to a given URL. When the second argument is the
 * string "true", the current history entry is replaced instead of pushing
 * a new one. The navigation is deferred via queueMicrotask so that any
 * in-flight DOM updates complete first.
 *
 * @param _element - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is the target URL, the second optionally "true" to replace history.
 * @returns Nothing.
 */
const redirectHelper: PPHelper = (_element: HTMLElement, _event: Event, ...args: string[]) => {
    const url = args[0];
    const replace = args[1] === 'true';

    if (!url) {
        console.error("The 'redirect' helper requires a URL string as its first argument.");
        return;
    }

    queueMicrotask(() => {
        if (replace) {
            window.location.replace(url);
        } else {
            window.location.assign(url);
        }
    });
};

RegisterHelper('redirect', redirectHelper);
