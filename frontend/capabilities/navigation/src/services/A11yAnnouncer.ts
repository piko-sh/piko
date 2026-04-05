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

/** DOM identifier for the announcer element. */
const ANNOUNCER_ID = 'ppf-a11y-announcer';

/** Represents the priority level for an accessibility announcement. */
export type AnnouncementPriority = 'polite' | 'assertive';

/** Provides accessibility announcements to screen readers via a hidden aria-live region. */
export interface A11yAnnouncer {
    /**
     * Announces a message to screen readers.
     * @param message - The message to announce.
     * @param priority - The priority level: 'polite' waits for silence, 'assertive' interrupts.
     */
    announce(message: string, priority?: AnnouncementPriority): void;

    /**
     * Announces a navigation event to screen readers.
     * @param pageTitle - The title of the new page.
     */
    announceNavigation(pageTitle: string): void;

    /** Announces a loading state to screen readers. */
    announceLoading(): void;

    /**
     * Announces an error message assertively to screen readers.
     * @param errorMessage - The error message to announce.
     */
    announceError(errorMessage: string): void;

    /** Removes the announcer element from the DOM. */
    destroy(): void;
}

/**
 * Creates a visually hidden announcer element that remains accessible to screen readers.
 * @returns The created HTML element configured with aria-live attributes and hidden styles.
 */
function createAnnouncerElement(): HTMLElement {
    const element = document.createElement('div');
    element.id = ANNOUNCER_ID;
    element.setAttribute('role', 'status');
    element.setAttribute('aria-live', 'polite');
    element.setAttribute('aria-atomic', 'true');

    element.style.cssText = [
        'position:absolute',
        'width:1px',
        'height:1px',
        'padding:0',
        'margin:-1px',
        'overflow:hidden',
        'clip:rect(0,0,0,0)',
        'white-space:nowrap',
        'border:0',
    ].join(';');

    return element;
}

/**
 * Creates an A11yAnnouncer for delivering screen reader announcements.
 * Reuses an existing announcer element if one is already present in the DOM.
 * @returns A new A11yAnnouncer instance.
 */
export function createA11yAnnouncer(): A11yAnnouncer {
    let element = document.getElementById(ANNOUNCER_ID);
    if (!element) {
        element = createAnnouncerElement();
        document.body.appendChild(element);
    }

    const announce = (message: string, priority: AnnouncementPriority = 'polite'): void => {
        if (!element) {
            return;
        }

        element.setAttribute('aria-live', priority);

        element.textContent = '';
        requestAnimationFrame(() => {
            if (element) {
                element.textContent = message;
            }
        });
    };

    return {
        announce,

        announceNavigation(pageTitle: string): void {
            announce(`Navigated to ${pageTitle}`);
        },

        announceLoading(): void {
            announce('Loading page');
        },

        announceError(errorMessage: string): void {
            announce(errorMessage, 'assertive');
        },

        destroy(): void {
            element?.remove();
            element = null;
        }
    };
}
