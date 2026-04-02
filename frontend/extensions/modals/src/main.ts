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

/// <reference path="../../../shared/types/index.d.ts" />

import { waitForPiko } from '../../../shared/utils';

/** HTMLElement augmented with an optional `close` and `update` method for modal elements. */
interface ModalElement extends HTMLElement {
    /** Closes the modal. */
    close?: () => void;
    /** Updates the modal content. */
    update?: () => void;
}

/** HTMLElement augmented with an optional `reload` method for partial elements. */
interface ReloadableElement extends HTMLElement {
    /** Reloads the partial from the server. */
    reload?: () => void;
}

/**
 * Registers modal and partial helper functions with the Piko framework.
 *
 * Registers four helpers: `showModal` (opens a modal by selector from
 * `data-modal-*` attributes), `closeModal` (closes the current or named
 * modal), `updateModal` (triggers an update on the current or named modal),
 * and `reloadPartial` (reloads a server-side partial by CSS selector,
 * falling back to a `pk-reload-partial` custom event).
 *
 * @param pk - The Piko namespace instance.
 */
function registerHelpers(pk: typeof window.piko): void {
    pk.registerHelper('showModal', (element: HTMLElement, _event: Event) => {
        const selector = element.dataset.modalSelector;
        if (!selector) {
            console.warn('helpers.showModal() requires a "data-modal-selector" attribute.', element);
            return;
        }

        void pk.modal.open({
            selector,
            params: new Map(),
            title: element.dataset.modalTitle ?? '',
            message: element.dataset.modalMessage ?? '',
            cancelLabel: element.dataset.modalCancelMessage ?? '',
            triggerElement: element
        });
    });

    pk.registerHelper('closeModal', (triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
        const modalName = args[0];
        let modalToClose: ModalElement | null = null;

        if (typeof modalName === 'string' && modalName) {
            const selector = `[modal="${modalName}"]`;
            modalToClose = document.querySelector(selector);

            if (!modalToClose) {
                console.warn(`closeModal: Could not find any modal with selector: ${selector}`);
                return;
            }
        } else {
            modalToClose = triggerElement.closest('[modal]');

            if (!modalToClose) {
                console.warn(`closeModal: The triggering element is not inside a [modal].`, {triggerElement});
                return;
            }
        }

        if (typeof modalToClose.close === 'function') {
            modalToClose.close();
        } else {
            console.error(`The found modal does not have a public 'close()' method.`, {modalToClose});
        }
    });

    pk.registerHelper('updateModal', (triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
        const modalName = args[0];
        let modalToUpdate: ModalElement | null = null;

        if (typeof modalName === 'string' && modalName) {
            const selector = `[modal="${modalName}"]`;
            modalToUpdate = document.querySelector(selector);

            if (!modalToUpdate) {
                console.warn(`updateModal: Could not find any modal with selector: ${selector}`);
                return;
            }
        } else {
            modalToUpdate = triggerElement.closest('[modal]');

            if (!modalToUpdate) {
                console.warn(`updateModal: The triggering element is not inside a [modal].`, {triggerElement});
                return;
            }
        }

        if (typeof modalToUpdate.update === 'function') {
            modalToUpdate.update();
        } else {
            console.error(`The found modal does not have a public 'update()' method.`, {modalToUpdate});
        }
    });

    pk.registerHelper('reloadPartial', (_triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
        const selector = args[0];
        if (!selector || typeof selector !== 'string') {
            console.error('reloadPartial helper requires a CSS selector string as its first argument.');
            return;
        }

        const partialToReload: ReloadableElement | null = document.querySelector(selector);

        if (!partialToReload) {
            console.warn(`reloadPartial: Could not find an element with the selector "${selector}".`);
            return;
        }

        if (typeof partialToReload.reload === 'function') {
            partialToReload.reload();
        } else if (partialToReload.hasAttribute('partial') && partialToReload.hasAttribute('src')) {
            partialToReload.dispatchEvent(new CustomEvent('pk-reload-partial', {bubbles: true}));
        } else {
            console.error(`The element matching "${selector}" does not have a public 'reload()' method.`);
        }
    });

    console.debug('[piko/modals] Extension loaded - helpers: showModal, closeModal, updateModal, reloadPartial');
}

waitForPiko('modals')
    .then(registerHelpers)
    .catch((err) => console.error(err.message));

export {};
