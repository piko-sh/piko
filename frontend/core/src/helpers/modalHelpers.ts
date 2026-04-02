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

/** Extends HTMLElement with optional modal lifecycle methods. */
interface ModalElement extends HTMLElement {
    /** Closes the modal. */
    close?: () => void;
    /** Updates the modal content. */
    update?: () => void;
    /** Navigates to a new URL within the modal. */
    navigate?: (options: { src: string; params: Record<string, string> }) => void;
}

/**
 * Opens a modal using the selector and metadata provided via data attributes
 * on the triggering element.
 *
 * @param element - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @returns Nothing.
 */
const showModalHelper: PPHelper = (element: HTMLElement, _event: Event) => {
    const selector = element.dataset.modalSelector;
    if (!selector) {
        console.warn('helpers.showModal() requires a "data-modal-selector" attribute.', element);
        return;
    }

    PPFramework.openModalIfAvailable({
        selector,
        params: new Map(),
        title: element.dataset.modalTitle ?? '',
        message: element.dataset.modalMessage ?? '',
        cancelLabel: element.dataset.modalCancelMessage ?? '',
        triggerElement: element
    }).catch((err: unknown) => {
        console.error('showModalHelper: Failed to open modal:', err);
    });
};

RegisterHelper('showModal', showModalHelper);

/**
 * Closes a modal identified by name or the closest ancestor modal of the
 * triggering element.
 *
 * @param triggerElement - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is an optional modal name.
 * @returns Nothing.
 */
const closeModalHelper: PPHelper = (triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
    const modalName = args[0];
    let modalToClose: ModalElement | null;

    if (modalName) {
        const selector = `[modal="${modalName}"]`;
        modalToClose = document.querySelector(selector);

        if (!modalToClose) {
            console.warn(`closeModalHelper: Could not find any modal with selector: ${selector}`);
            return;
        }
    } else {
        modalToClose = triggerElement.closest('[modal]');

        if (!modalToClose) {
            console.warn(`closeModalHelper: The triggering element is not inside a [modal].`, {triggerElement});
            return;
        }
    }

    if (typeof modalToClose.close === 'function') {
        modalToClose.close();
    } else {
        console.error(`The found modal does not have a public 'close()' method.`, {modalToClose});
    }
};

RegisterHelper('closeModal', closeModalHelper);

/**
 * Refreshes the content of a modal identified by name or the closest ancestor
 * modal of the triggering element.
 *
 * @param triggerElement - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is an optional modal name.
 * @returns Nothing.
 */
const updateModalHelper: PPHelper = (triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
    const modalName = args[0];
    let modalToUpdate: ModalElement | null;

    if (modalName) {
        const selector = `[modal="${modalName}"]`;
        modalToUpdate = document.querySelector(selector);

        if (!modalToUpdate) {
            console.warn(`updateModalHelper: Could not find any modal with selector: ${selector}`);
            return;
        }
    } else {
        modalToUpdate = triggerElement.closest('[modal]');

        if (!modalToUpdate) {
            console.warn(`updateModalHelper: The triggering element is not inside a [modal].`, {triggerElement});
            return;
        }
    }

    if (typeof modalToUpdate.update === 'function') {
        modalToUpdate.update();
    } else {
        console.error(`The found modal does not have a public 'update()' method.`, {modalToUpdate});
    }
};

RegisterHelper('updateModal', updateModalHelper);

/**
 * Navigates a modal to a new source URL by setting its src attribute.
 * The first argument is the modal CSS selector and the second is the full
 * URL (including any query string).
 *
 * @param _triggerElement - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is the modal selector, the second is the source URL.
 * @returns Nothing.
 */
const modalNavigateHelper: PPHelper = (_triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
    const modalSelector = args[0];
    const srcWithParams = args[1];

    if (!modalSelector || !srcWithParams) {
        console.error('modalNavigate helper requires modalSelector and src as arguments');
        return;
    }

    const modal = document.querySelector(modalSelector);

    if (!modal) {
        console.warn(`modalNavigateHelper: Could not find modal with selector: ${modalSelector}`);
        return;
    }

    modal.setAttribute('src', srcWithParams);
};

RegisterHelper('modalNavigate', modalNavigateHelper);
