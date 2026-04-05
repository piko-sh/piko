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

/** CSS class applied to elements during loading operations. */
const LOADING_CLASS = 'pk-loading';

/** ARIA attribute set during loading for accessibility. */
const ARIA_BUSY_ATTR = 'aria-busy';

/**
 * Applies loading state to an element.
 *
 * Adds the pk-loading CSS class and sets aria-busy for accessibility.
 *
 * @param el - The element to mark as loading.
 */
export function applyLoadingIndicator(el: HTMLElement): void {
    el.classList.add(LOADING_CLASS);
    el.setAttribute(ARIA_BUSY_ATTR, 'true');
}

/**
 * Removes loading state from an element.
 *
 * Removes the pk-loading CSS class and aria-busy attribute.
 *
 * @param el - The element to unmark.
 */
export function removeLoadingIndicator(el: HTMLElement): void {
    el.classList.remove(LOADING_CLASS);
    el.removeAttribute(ARIA_BUSY_ATTR);
}
