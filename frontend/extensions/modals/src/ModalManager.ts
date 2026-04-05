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

/** Minimal hook manager interface for emitting modal events. */
interface HookManagerLike {
    emit(event: string, payload: unknown): void;
}

/** Hook event constants for modal analytics. */
const HookEvent = {
    MODAL_OPEN: 'modal:open',
    MODAL_CLOSE: 'modal:close',
} as const;

type HookManager = HookManagerLike;

/**
 * Dependencies for creating a ModalManager.
 */
export interface ModalManagerDependencies {
    /** Hook manager for analytics events. */
    hookManager?: HookManager;
}

/**
 * Options for opening a modal dialogue.
 */
export interface ModalRequestOptions {
    /** CSS selector for the modal element. */
    selector: string;
    /** Parameters to pass to the modal. */
    params?: Map<string, string>;
    /** Title text for the modal header. */
    title?: string;
    /** Message text for the modal body. */
    message?: string;
    /** Label for the cancel button. */
    cancelLabel?: string;
    /** Label for the confirm button. */
    confirmLabel?: string;
    /** Action to execute when confirmed. */
    confirmAction?: string;
    /** Element that triggered the modal opening. */
    triggerElement: HTMLElement;
    /** Event name to dispatch if the modal is not found. */
    fallbackEventName?: string;
}

/**
 * Manages modal dialogue interactions.
 */
export interface ModalManager {
    /** Opens a modal if available, dispatching a fallback event if not found. */
    openIfAvailable(options: ModalRequestOptions): Promise<void>;
}

/**
 * Creates a ModalManager instance for handling modal dialogues.
 *
 * @param deps - Optional dependencies for analytics integration.
 * @returns A new ModalManager instance.
 */
export function createModalManager(deps: ModalManagerDependencies = {}): ModalManager {
    const {hookManager} = deps;

    return {
        /**
         * Opens a modal if available, dispatching a fallback event if not found.
         *
         * @param options - The modal request options.
         */
        async openIfAvailable(options: ModalRequestOptions): Promise<void> {
            const {
                selector: modalSelector,
                params = new Map(),
                title: modalTitle = '',
                message: modalMessage = '',
                cancelLabel: modalCancelLabel = '',
                confirmLabel: modalConfirmLabel = '',
                confirmAction: modalConfirmAction = '',
                triggerElement,
                fallbackEventName = 'modal-not-found'
            } = options;
            const modalElem = document.querySelector(modalSelector);

            if (!modalElem) {
                console.warn(`ModalManager: Could not find modal "${modalSelector}". Falling back to dispatch event.`);
                triggerElement.dispatchEvent(new CustomEvent(fallbackEventName, {bubbles: true, composed: true}));
                return;
            }

            const modalId = modalElem.id || modalSelector;

            hookManager?.emit(HookEvent.MODAL_OPEN, {
                modalId,
                url: window.location.href,
                timestamp: Date.now()
            });

            const requestFn = (modalElem as unknown as {
                request?: (opts: Record<string, unknown>) => Promise<boolean>
            }).request;

            if (typeof requestFn === 'function') {
                const confirmed = await requestFn({
                    modal_title: modalTitle,
                    message: modalMessage,
                    cancel_label: modalCancelLabel,
                    confirm_label: modalConfirmLabel,
                    confirm_action: modalConfirmAction,
                    params: Object.fromEntries(params.entries())
                });

                hookManager?.emit(HookEvent.MODAL_CLOSE, {
                    modalId,
                    timestamp: Date.now()
                });

                if (confirmed) {
                    triggerElement.dispatchEvent(
                        new CustomEvent('modal-confirmed', {
                            detail: {},
                            bubbles: true,
                            composed: true
                        })
                    );
                } else {
                    triggerElement.dispatchEvent(
                        new CustomEvent('modal-cancelled', {
                            detail: {},
                            bubbles: true,
                            composed: true
                        })
                    );
                }
            } else {
                console.warn(`ModalManager: The modal "${modalSelector}" does not have a request() function. Trying open...`);
                modalElem.setAttribute('open', 'true');
            }
        }
    };
}
