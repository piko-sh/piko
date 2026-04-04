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
 * Resolves a target to a DOM element via p-ref attribute or CSS selector.
 *
 * Checks for a matching p-ref attribute first, then falls back to a
 * standard CSS selector query.
 *
 * @param target - CSS selector, p-ref name, or Element instance.
 * @returns The resolved element, or null if not found.
 */
export function resolveElement(target: string | Element): Element | null {
    if (target instanceof Element) {
        return target;
    }

    return document.querySelector(`[p-ref="${target}"]`)
        ?? document.querySelector(target);
}

/**
 * Resolves a target to an HTMLElement via p-ref attribute or CSS selector.
 *
 * Validates each candidate is an HTMLElement instance before returning,
 * filtering out non-HTMLElement nodes such as SVG elements.
 *
 * @param target - CSS selector, p-ref name, or HTMLElement instance.
 * @returns The resolved HTMLElement, or null if not found.
 */
export function resolveHTMLElement(target: string | HTMLElement): HTMLElement | null {
    if (target instanceof HTMLElement) {
        return target;
    }

    const byRef = document.querySelector(`[p-ref="${target}"]`);
    if (byRef instanceof HTMLElement) {
        return byRef;
    }

    const bySelector = document.querySelector(target);
    if (bySelector instanceof HTMLElement) {
        return bySelector;
    }

    return null;
}
