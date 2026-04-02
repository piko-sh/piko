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

import {type PPHelper, RegisterHelper} from '@/core';

/**
 * Locates a form element by an optional CSS selector, falling back to the
 * closest ancestor form of the triggering element.
 *
 * @param element - The HTML element that triggered the helper.
 * @param helperName - The name of the calling helper, used in warning messages.
 * @param formSelector - An optional CSS selector identifying the target form.
 * @returns The matched HTMLFormElement, or null if none is found.
 */
const findForm = (element: HTMLElement, helperName: string, formSelector?: string): HTMLFormElement | null => {
    if (formSelector) {
        const form = document.querySelector<HTMLFormElement>(formSelector);

        if (!form) {
            console.error(`PPFramework Helper '${helperName}' failed: Could not find any form matching the selector "${formSelector}".`, {
                triggeringElement: element,
            });
            return null;
        }
        return form;
    }

    const parentForm = element.closest<HTMLFormElement>('form');

    if (!parentForm) {
        console.warn(`helpers.${helperName}() was used without a selector, but no parent form could be found.`, {
            triggeringElement: element,
        });
        return null;
    }

    return parentForm;
};

/**
 * Submits a form identified by an optional CSS selector or the closest
 * ancestor form. Uses requestSubmit to trigger validation.
 *
 * @param element - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is an optional form CSS selector.
 * @returns Nothing.
 */
const submitFormHelper: PPHelper = (element: HTMLElement, _event: Event, ...args: string[]) => {
    const formSelector = args[0];
    const form = findForm(element, 'submitForm', formSelector);
    form?.requestSubmit();
};

/**
 * Resets a form identified by an optional CSS selector or the closest
 * ancestor form.
 *
 * @param element - The HTML element that triggered the helper.
 * @param _event - The original DOM event.
 * @param args - The variadic string arguments; the first is an optional form CSS selector.
 * @returns Nothing.
 */
const resetFormHelper: PPHelper = (element: HTMLElement, _event: Event, ...args: string[]) => {
    const formSelector = args[0];
    const form = findForm(element, 'resetForm', formSelector);
    form?.reset();
};

RegisterHelper('submitForm', submitFormHelper);
RegisterHelper('submitModalForm', submitFormHelper);
RegisterHelper('resetForm', resetFormHelper);
