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

import {PPFramework, type PPHelper, RegisterHelper} from '@/core';

/** Extends HTMLElement with an optional reload method. */
interface ReloadableElement extends HTMLElement {
    /** Reloads the element's content. */
    reload?: () => void;
}

/**
 * Reloads a partial element identified by CSS selector. If the element exposes
 * a reload() method it is called directly; otherwise, if the element has
 * "partial" and "src" attributes, a remote render is performed as a fallback.
 *
 * @param _triggerElement - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is the CSS selector of the partial to reload.
 * @returns Nothing.
 */
const reloadPartialHelper: PPHelper = (_triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
    const selector = args[0];
    if (!selector) {
        console.error('reloadPartial helper requires a CSS selector string as its first argument.');
        return;
    }

    const partialToReload: ReloadableElement | null = document.querySelector(selector);

    if (!partialToReload) {
        console.warn(`reloadPartial helper: Could not find an element with the selector "${selector}".`);
        return;
    }

    if (typeof partialToReload.reload === 'function') {
        partialToReload.reload();
    } else if (partialToReload.hasAttribute("partial") && partialToReload.hasAttribute("src")) {
        const src = partialToReload.getAttribute("src");
        if (src) {
            PPFramework.remoteRender({
                src,
                args: {},
                querySelector: 'div',
                patchLocation: partialToReload
            }).catch((err: unknown) => {
                console.error(`reloadPartialHelper: Failed to reload partial "${selector}":`, err);
            });
        }
    } else {
        console.error(`The element matching "${selector}" does not have a public 'reload()' method. Ensure you've exposed it on the component (e.g., this.reload = ...).`);
    }
};

RegisterHelper('reloadPartial', reloadPartialHelper);
