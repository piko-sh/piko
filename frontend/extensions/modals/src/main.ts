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
import { createModalManager, type ModalManager } from './ModalManager';

/**
 * Creates a hook manager that bridges modal analytics events onto the
 * framework's hooks bus via `pk._emitHook`. `pk.hooks` is listener-only by
 * design, so trusted extensions use the underscore escape hatch to emit.
 *
 * @param pk - The Piko namespace instance.
 * @returns A hook manager that forwards `emit` calls to `pk._emitHook`.
 */
function createPikoHookManager(pk: typeof window.piko): {emit(event: string, payload: unknown): void} {
    return {
        emit(event: string, payload: unknown): void {
            pk._emitHook(event as Parameters<typeof pk._emitHook>[0], payload);
        }
    };
}

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
 * Detail payload for the pk-open-modal custom event dispatched by DOMBinder.
 */
interface ModalEventDetail {
    /** CSS selector for the modal element. */
    selector: string;
    /** Key-value parameters for the modal. */
    params: Map<string, string>;
    /** Modal title. */
    title: string;
    /** Modal message. */
    message: string;
    /** Cancel button label. */
    cancelLabel: string;
    /** Confirm button label. */
    confirmLabel: string;
    /** Confirm action name. */
    confirmAction: string;
}

/**
 * Registers a listener that opens modals in response to `pk-open-modal` events
 * dispatched by DOMBinder.
 *
 * @param modalManager - Modal manager used to open the requested modal.
 */
function registerPkOpenModalListener(modalManager: ModalManager): void {
    document.addEventListener('pk-open-modal', (event: Event) => {
        const detail = (event as CustomEvent<ModalEventDetail>).detail;
        const triggerElement = event.target as HTMLElement;
        void modalManager.openIfAvailable({
            selector: detail.selector,
            params: detail.params,
            title: detail.title,
            message: detail.message,
            cancelLabel: detail.cancelLabel,
            confirmLabel: detail.confirmLabel,
            confirmAction: detail.confirmAction,
            triggerElement,
        });
    });
}

/**
 * Opens a modal via data attributes on the trigger element.
 *
 * @param element - The element that declares the modal via its dataset.
 * @param modalManager - Modal manager used to open the modal.
 */
function showModalHelper(element: HTMLElement, modalManager: ModalManager): void {
    const selector = element.dataset.modalSelector;
    if (!selector) {
        console.warn('helpers.showModal() requires a "data-modal-selector" attribute.', element);
        return;
    }
    void modalManager.openIfAvailable({
        selector,
        params: new Map(),
        title: element.dataset.modalTitle ?? '',
        message: element.dataset.modalMessage ?? '',
        cancelLabel: element.dataset.modalCancelMessage ?? '',
        triggerElement: element
    });
}

/**
 * Resolves the modal element the helper should act on, either by explicit name
 * or by walking up from the trigger element to the nearest `[modal]` ancestor.
 *
 * @param triggerElement - Element that invoked the helper.
 * @param modalName - Optional explicit modal name supplied as a helper argument.
 * @param helperName - Helper name used for diagnostic messages.
 * @returns The resolved modal element, or null if no suitable modal is found.
 */
function resolveTargetModal(
    triggerElement: HTMLElement,
    modalName: string | undefined,
    helperName: string
): ModalElement | null {
    if (typeof modalName === 'string' && modalName) {
        const selector = `[modal="${modalName}"]`;
        const found = document.querySelector<ModalElement>(selector);
        if (!found) {
            console.warn(`${helperName}: Could not find any modal with selector: ${selector}`);
        }
        return found;
    }
    const ancestor = triggerElement.closest<ModalElement>('[modal]');
    if (!ancestor) {
        console.warn(`${helperName}: The triggering element is not inside a [modal].`, {triggerElement});
    }
    return ancestor;
}

/**
 * Closes a modal identified by name, or the nearest ancestor modal of the
 * triggering element.
 *
 * @param triggerElement - Element that invoked the helper.
 * @param modalName - Optional explicit modal name supplied as a helper argument.
 */
function closeModalHelper(triggerElement: HTMLElement, modalName: string | undefined): void {
    const modalToClose = resolveTargetModal(triggerElement, modalName, 'closeModal');
    if (!modalToClose) {
        return;
    }
    if (typeof modalToClose.close === 'function') {
        modalToClose.close();
    } else {
        console.error(`The found modal does not have a public 'close()' method.`, {modalToClose});
    }
}

/**
 * Requests an update on a modal identified by name, or the nearest ancestor
 * modal of the triggering element.
 *
 * @param triggerElement - Element that invoked the helper.
 * @param modalName - Optional explicit modal name supplied as a helper argument.
 */
function updateModalHelper(triggerElement: HTMLElement, modalName: string | undefined): void {
    const modalToUpdate = resolveTargetModal(triggerElement, modalName, 'updateModal');
    if (!modalToUpdate) {
        return;
    }
    if (typeof modalToUpdate.update === 'function') {
        modalToUpdate.update();
    } else {
        console.error(`The found modal does not have a public 'update()' method.`, {modalToUpdate});
    }
}

/**
 * Reloads a server-side partial identified by CSS selector, falling back to a
 * `pk-reload-partial` custom event when the element exposes no `reload()`.
 *
 * @param selector - CSS selector identifying the partial element.
 */
function reloadPartialHelper(selector: string | undefined): void {
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
}

/**
 * Registers modal and partial helper functions with the Piko framework and
 * listens for `pk-open-modal` events dispatched by DOMBinder.
 *
 * @param pk - The Piko namespace instance.
 */
function registerHelpers(pk: typeof window.piko): void {
    const hookManager = createPikoHookManager(pk);
    const modalManager = createModalManager({hookManager});

    registerPkOpenModalListener(modalManager);

    pk.registerHelper('showModal', (element: HTMLElement) => {
        showModalHelper(element, modalManager);
    });
    pk.registerHelper('closeModal', (triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
        closeModalHelper(triggerElement, args[0]);
    });
    pk.registerHelper('updateModal', (triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
        updateModalHelper(triggerElement, args[0]);
    });
    pk.registerHelper('reloadPartial', (_triggerElement: HTMLElement, _event: Event, ...args: string[]) => {
        reloadPartialHelper(args[0]);
    });
}

waitForPiko('modals')
    .then(registerHelpers)
    .catch((err: unknown) => console.error((err as Error).message));

export {};
