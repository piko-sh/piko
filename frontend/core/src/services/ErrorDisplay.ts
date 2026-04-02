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

/** Default duration to display error messages in milliseconds. */
const ERROR_DISPLAY_MS = 5000;

/** Displays error messages to the user. */
export interface ErrorDisplay {
    /**
     * Displays an error message. Reuses an existing error element if one is present.
     * @param message - The error message to display.
     */
    show(message: string): void;

    /** Clears any currently displayed error and cancels its auto-dismiss timer. */
    clear(): void;
}

/** Configuration options for creating an ErrorDisplay. */
export interface ErrorDisplayOptions {
    /** Duration to show error in milliseconds. */
    displayMs?: number;

    /** Container element to append error to. */
    container?: HTMLElement;
}

/**
 * Creates an ErrorDisplay for showing error messages with auto-dismiss behaviour.
 * @param options - The optional display configuration.
 * @returns A new ErrorDisplay instance.
 */
export function createErrorDisplay(options: ErrorDisplayOptions = {}): ErrorDisplay {
    const {
        displayMs = ERROR_DISPLAY_MS,
        container = document.body
    } = options;

    let currentElement: HTMLElement | null = null;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    return {
        show(message: string) {
            const existingEl = document.getElementById('ppf-error-message');
            if (existingEl) {
                existingEl.textContent = message;
                return;
            }

            if (timeoutId) {
                clearTimeout(timeoutId);
            }

            const errEl = document.createElement('div');
            errEl.id = 'ppf-error-message';
            errEl.style.cssText = [
                'position:fixed',
                'top:60px',
                'left:50%',
                'transform:translateX(-50%)',
                'background:#f44336',
                'color:#fff',
                'padding:6px 12px',
                'border-radius:4px',
                'font-family:sans-serif',
                'z-index:99999',
            ].join(';');
            errEl.textContent = message;
            container.appendChild(errEl);
            currentElement = errEl;

            timeoutId = setTimeout(() => {
                if (errEl.parentNode) {
                    errEl.parentNode.removeChild(errEl);
                }
                currentElement = null;
                timeoutId = null;
            }, displayMs);
        },

        clear() {
            if (timeoutId) {
                clearTimeout(timeoutId);
                timeoutId = null;
            }
            if (currentElement?.parentNode) {
                currentElement.parentNode.removeChild(currentElement);
                currentElement = null;
            }
        }
    };
}
