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
 * Touch-target expansion utilities for M3E components.
 *
 * Material Design 3 requires a minimum 48dp (48px) touch target for all
 * interactive elements, even when the visual element is smaller (e.g. a
 * 40px button or 32px chip).
 *
 * Two approaches are provided:
 *
 * 1. **CSS class** - add a `.touch-target` element inside the interactive
 *    element; the CSS makes it an invisible 48px-tall hit area:
 *
 *    <button class="button">
 *        <span class="touch-target"></span>
 *        <span class="label">Click me</span>
 *    </button>
 *
 * 2. **Host margin** - when using `touch-target="wrapper"` attribute on the
 *    host, extra vertical margin ensures the component's bounding box is
 *    at least 48px tall without changing its visual height.
 *
 * Usage in a PKC <style> block - just paste the CSS rules directly.
 * This module documents the pattern; components copy the CSS they need.
 *
 * @module m3e-touch-target
 */

/**
 * CSS for the inner touch-target element.
 * Place a `<span class="touch-target"></span>` inside any interactive element.
 */
export const TOUCH_TARGET_CSS = `
.touch-target {
    position: absolute;
    top: 50%;
    left: 0;
    right: 0;
    height: 48px;
    transform: translateY(-50%);
    pointer-events: auto;
}
`;

/**
 * CSS for the host wrapper approach.
 * Uses `:host([touch-target="wrapper"])` to add margin.
 *
 * @param {number} visualHeight - The visual height of the component in px.
 * @returns {string} CSS string.
 */
export function hostWrapperCSS(visualHeight) {
    const margin = Math.max(0, (48 - visualHeight) / 2);
    return `
:host([touch-target="wrapper"]) {
    margin: ${margin}px 0;
}
`;
}

/**
 * CSS to hide the touch target when opted out.
 */
export const TOUCH_TARGET_NONE_CSS = `
:host([touch-target="none"]) .touch-target {
    display: none;
}
`;

/**
 * Full touch-target CSS snippet combining all rules.
 * Pass the component's visual height to get correct wrapper margins.
 *
 * @param {number} [visualHeight=40] - The visual height of the component in px.
 * @returns {string} Complete CSS for touch-target support.
 */
export function touchTargetCSS(visualHeight = 40) {
    return TOUCH_TARGET_CSS + hostWrapperCSS(visualHeight) + TOUCH_TARGET_NONE_CSS;
}
