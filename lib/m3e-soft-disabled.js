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
 * Soft-disabled pattern for M3E components.
 *
 * A soft-disabled element looks disabled but remains focusable and visible
 * to screen readers. This follows W3C ARIA guidance for keyboard-accessible
 * disabled controls:
 * https://www.w3.org/WAI/ARIA/apg/practices/keyboard-interface/#kbd_disabled_controls
 *
 * Usage in a PKC <script> block:
 *
 *   import { setupSoftDisabled } from 'piko.sh/piko/lib/m3e-soft-disabled.js';
 *
 *   const state = {
 *       disabled: false as boolean,
 *       soft_disabled: false as boolean,
 *   };
 *
 *   const softDisabled = setupSoftDisabled({
 *       host: this,
 *       getState: () => state,
 *       interactiveSelector: '.button',
 *   });
 *
 * The module intercepts clicks on the host when soft-disabled is active and
 * manages aria-disabled on the interactive element.
 *
 * @module m3e-soft-disabled
 */

/**
 * Checks whether a component is effectively disabled (either hard or soft).
 *
 * @param {Object} state - The component's reactive state.
 * @returns {boolean}
 */
export function isEffectivelyDisabled(state) {
    return !!(state.disabled || state.soft_disabled);
}

/**
 * Sets up soft-disabled behaviour on a component.
 *
 * @param {Object} options
 * @param {HTMLElement} options.host
 *   The custom element host.
 * @param {() => Object} options.getState
 *   Returns the component's reactive state object. Must have `disabled` and
 *   `soft_disabled` properties.
 * @param {string} [options.interactiveSelector]
 *   CSS selector for the interactive element inside the shadow DOM that should
 *   receive aria-disabled. If omitted, aria-disabled is set on the host itself.
 * @returns {SoftDisabledController}
 */
export function setupSoftDisabled(options) {
    const { host, getState, interactiveSelector } = options;

    function getInteractiveElement() {
        if (!interactiveSelector) return host;
        return host.shadowRoot?.querySelector(interactiveSelector) || host;
    }

    function syncAriaDisabled() {
        const state = getState();
        const el = getInteractiveElement();
        if (state.soft_disabled) {
            el.setAttribute('aria-disabled', 'true');
            el.removeAttribute('disabled');
        } else if (state.disabled) {
            el.setAttribute('aria-disabled', 'true');
        } else {
            el.removeAttribute('aria-disabled');
        }
    }

    const clickHandler = (event) => {
        const state = getState();
        if (state.soft_disabled) {
            event.stopImmediatePropagation();
            event.preventDefault();
        }
    };

    const keydownHandler = (event) => {
        const state = getState();
        if (state.soft_disabled && (event.key === 'Enter' || event.key === ' ')) {
            event.stopImmediatePropagation();
            event.preventDefault();
        }
    };

    host.addEventListener('click', clickHandler, true);
    host.addEventListener('keydown', keydownHandler, true);

    return {
        syncAriaDisabled,
        isEffectivelyDisabled() {
            return isEffectivelyDisabled(getState());
        },
        destroy() {
            host.removeEventListener('click', clickHandler, true);
            host.removeEventListener('keydown', keydownHandler, true);
        },
    };
}
