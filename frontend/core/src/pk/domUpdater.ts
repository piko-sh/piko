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
 * Shared callback for re-binding the DOM after a partial morph.
 *
 * PPFramework registers its bindDOM function here during init. The partial
 * reload code calls it after morphing new content to re-bind p-on handlers.
 */

/** The currently registered DOM rebind callback, or null if not yet registered. */
let _onDOMUpdated: ((root: HTMLElement) => void) | null = null;

/**
 * Registers the DOM rebind callback. Called by PPFramework during init.
 *
 * @param callback - The function to call when the DOM needs rebinding.
 */
export function registerDOMUpdater(callback: (root: HTMLElement) => void): void {
    _onDOMUpdated = callback;
}

/**
 * Calls the registered DOM rebind callback on the given root element.
 * This re-binds p-on event handlers after a partial morph.
 *
 * @param root - The root element whose subtree needs rebinding.
 */
export function notifyDOMUpdated(root: HTMLElement): void {
    if (_onDOMUpdated) {
        _onDOMUpdated(root);
    }
}
