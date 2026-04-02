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
 * Roving-tabindex keyboard navigation controller for M3E compound components.
 *
 * Implements the WAI-ARIA roving tabindex pattern: exactly one item in a
 * collection has tabIndex=0 (focusable via Tab); all others have tabIndex=-1.
 * Arrow keys move focus between items; Home/End jump to first/last.
 *
 * Usage in a PKC <script> block:
 *
 *   import { createKeyboardNav } from 'piko.sh/piko/lib/m3e-keyboard-nav.js';
 *
 *   const nav = createKeyboardNav({
 *       getItems: () => this.querySelectorAll('m3e-tab'),
 *       isDisabled: (item) => item.hasAttribute('disabled'),
 *       orientation: 'horizontal',
 *       wrap: true,
 *   });
 *
 *   function onConnected() {
 *       this.addEventListener('keydown', nav.handleKeydown);
 *       nav.syncTabIndices();
 *   }
 *
 *   function onDisconnected() {
 *       this.removeEventListener('keydown', nav.handleKeydown);
 *   }
 *
 * @module m3e-keyboard-nav
 */

/**
 * Returns the computed text direction for a given element.
 *
 * @param {HTMLElement} el
 * @returns {boolean} true if the element's text direction is right-to-left.
 */
function isRtl(el) {
    return getComputedStyle(el).direction === 'rtl';
}

/**
 * Creates a keyboard navigation controller.
 *
 * @param {Object} options
 * @param {() => NodeListOf<HTMLElement> | HTMLElement[]} options.getItems
 *   Returns the navigable items in DOM order.
 * @param {(item: HTMLElement) => boolean} [options.isDisabled]
 *   Returns true if the item should be skipped during navigation.
 *   Defaults to checking the `disabled` attribute.
 * @param {'horizontal' | 'vertical' | 'both'} [options.orientation]
 *   Axis of arrow-key navigation. Defaults to 'vertical'.
 * @param {boolean} [options.wrap]
 *   Whether navigation wraps from last to first and vice versa.
 *   Defaults to true.
 * @param {boolean} [options.autoActivate]
 *   Whether focusing an item also activates it (fires a click).
 *   Defaults to false.
 * @param {(item: HTMLElement) => string} [options.getTypeaheadText]
 *   Returns the text used for typeahead matching. Defaults to textContent.
 * @param {boolean} [options.typeahead]
 *   Whether to enable typeahead (type a character to jump to matching item).
 *   Defaults to false.
 * @returns {KeyboardNav}
 */
export function createKeyboardNav(options) {
    const {
        getItems,
        isDisabled = (item) => item.hasAttribute('disabled') || item.getAttribute('aria-disabled') === 'true',
        orientation = 'vertical',
        wrap = true,
        autoActivate = false,
        getTypeaheadText = (item) => (item.textContent || '').trim().toLowerCase(),
        typeahead = false,
    } = options;

    let typeaheadBuffer = '';
    let typeaheadTimeout = 0;

    function getActivatableItems() {
        const all = Array.from(getItems());
        return all.filter((item) => !isDisabled(item));
    }

    function getActiveItem() {
        const all = Array.from(getItems());
        return all.find((item) => item.tabIndex === 0) || null;
    }

    function activateItem(item) {
        const all = Array.from(getItems());
        for (const el of all) {
            el.tabIndex = el === item ? 0 : -1;
        }
        item.focus();
        if (autoActivate) {
            item.click();
        }
    }

    function focusItem(item) {
        const all = Array.from(getItems());
        for (const el of all) {
            el.tabIndex = el === item ? 0 : -1;
        }
        item.focus();
    }

    function navigateToNext(current, direction) {
        const activatable = getActivatableItems();
        if (activatable.length === 0) return;

        const currentIndex = current ? activatable.indexOf(current) : -1;
        let nextIndex;

        if (direction === 'first') {
            nextIndex = 0;
        } else if (direction === 'last') {
            nextIndex = activatable.length - 1;
        } else if (direction === 'next') {
            if (currentIndex === -1) {
                nextIndex = 0;
            } else {
                nextIndex = currentIndex + 1;
                if (nextIndex >= activatable.length) {
                    nextIndex = wrap ? 0 : activatable.length - 1;
                }
            }
        } else {
            if (currentIndex === -1) {
                nextIndex = activatable.length - 1;
            } else {
                nextIndex = currentIndex - 1;
                if (nextIndex < 0) {
                    nextIndex = wrap ? activatable.length - 1 : 0;
                }
            }
        }

        const target = activatable[nextIndex];
        if (target) {
            if (autoActivate) {
                activateItem(target);
            } else {
                focusItem(target);
            }
        }
    }

    function isForwardKey(key, el) {
        const rtl = isRtl(el);
        if (orientation === 'horizontal' || orientation === 'both') {
            if (key === 'ArrowRight') return !rtl;
            if (key === 'ArrowLeft') return rtl;
        }
        if (orientation === 'vertical' || orientation === 'both') {
            if (key === 'ArrowDown') return true;
        }
        return false;
    }

    function isBackwardKey(key, el) {
        const rtl = isRtl(el);
        if (orientation === 'horizontal' || orientation === 'both') {
            if (key === 'ArrowLeft') return !rtl;
            if (key === 'ArrowRight') return rtl;
        }
        if (orientation === 'vertical' || orientation === 'both') {
            if (key === 'ArrowUp') return true;
        }
        return false;
    }

    function isNavigationKey(key) {
        return ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(key);
    }

    function handleTypeahead(event) {
        if (!typeahead) return false;
        if (event.key.length !== 1 || event.ctrlKey || event.metaKey || event.altKey) return false;

        event.preventDefault();
        const char = event.key.toLowerCase();

        clearTimeout(typeaheadTimeout);
        typeaheadBuffer += char;
        typeaheadTimeout = setTimeout(() => { typeaheadBuffer = ''; }, 500);

        const activatable = getActivatableItems();
        const current = getActiveItem();
        const currentIndex = current ? activatable.indexOf(current) : -1;

        const startIndex = typeaheadBuffer.length === 1 ? currentIndex + 1 : currentIndex;

        for (let i = 0; i < activatable.length; i++) {
            const index = (startIndex + i) % activatable.length;
            const text = getTypeaheadText(activatable[index]);
            if (text.startsWith(typeaheadBuffer)) {
                focusItem(activatable[index]);
                return true;
            }
        }

        return true;
    }

    /** @type {(event: KeyboardEvent) => void} */
    const handleKeydown = (event) => {
        const key = event.key;

        if (!isNavigationKey(key)) {
            handleTypeahead(event);
            return;
        }

        const target = event.currentTarget;
        const current = getActiveItem();

        if (key === 'Home') {
            event.preventDefault();
            navigateToNext(current, 'first');
            return;
        }

        if (key === 'End') {
            event.preventDefault();
            navigateToNext(current, 'last');
            return;
        }

        if (isForwardKey(key, target)) {
            event.preventDefault();
            navigateToNext(current, 'next');
            return;
        }

        if (isBackwardKey(key, target)) {
            event.preventDefault();
            navigateToNext(current, 'prev');
        }
    };

    function syncTabIndices() {
        const all = Array.from(getItems());
        const activatable = all.filter((item) => !isDisabled(item));

        const existing = all.find((item) => item.tabIndex === 0 && !isDisabled(item));

        if (existing) {
            for (const el of all) {
                el.tabIndex = el === existing ? 0 : -1;
            }
            return;
        }

        const first = activatable[0];
        for (const el of all) {
            el.tabIndex = el === first ? 0 : -1;
        }
    }

    function activateByIndex(index) {
        const activatable = getActivatableItems();
        if (index >= 0 && index < activatable.length) {
            const all = Array.from(getItems());
            for (const el of all) {
                el.tabIndex = el === activatable[index] ? 0 : -1;
            }
        }
    }

    function activateElement(element) {
        const all = Array.from(getItems());
        for (const el of all) {
            el.tabIndex = el === element ? 0 : -1;
        }
    }

    function handleFocusout(event) {
        if (event.currentTarget.contains(event.relatedTarget)) {
            return;
        }
        const active = getActiveItem();
        if (!active) {
            syncTabIndices();
        }
    }

    return {
        handleKeydown,
        syncTabIndices,
        activateByIndex,
        activateElement,
        getActiveItem,
        handleFocusout,
    };
}
