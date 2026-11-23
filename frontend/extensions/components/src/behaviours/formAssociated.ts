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

import {PPElement} from '@/element';
import {registerBehaviour} from '@/behaviours/registry';

/** Native HTML form elements that support validation. */
type NativeFormElement = HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;

/**
 * Builds a callback that synchronises form value and validity. Custom validity takes
 * precedence, and a null result clears validity. Native validation checks the nativeInput
 * ref before looking in the shadow DOM.
 *
 * @param component - The component that supplies the value and validation.
 * @param internals - The object that stores the form value and validity.
 * @returns A callback that emits an input event when the reported value changes.
 */
function createUpdateFormState(component: PPElement, internals: ElementInternals): () => void {
    let lastReportedValue: string | null | undefined;

    return () => {
        const value = (component.state?.value as string | null) ?? null;
        internals.setFormValue(value);

        const customValidityFn = component.customValidity;
        if (typeof customValidityFn === 'function') {
            const result = customValidityFn();
            if (result) {
                internals.setValidity(result.validity, result.message, result.anchor);
            } else {
                internals.setValidity({});
            }
        } else {
            const nativeInputRef = component.refs.nativeInput as NativeFormElement | undefined;
            const nativeEl = nativeInputRef ?? component.shadowRoot?.querySelector('input, select, textarea') as
                NativeFormElement | null;

            if (nativeEl) {
                internals.setValidity(nativeEl.validity, nativeEl.validationMessage || "Please validate", nativeEl);
            } else {
                internals.setValidity({});
            }
        }

        if (value !== lastReportedValue) {
            lastReportedValue = value;
            component.dispatchEvent(new Event('input', {bubbles: true, composed: true}));
        }
    };
}

/**
 * Defines standard form-related properties and methods on the component.
 *
 * @param component - The PPElement component to augment.
 * @param internals - The ElementInternals instance for the component.
 */
function defineFormProperties(component: PPElement, internals: ElementInternals): void {
    Object.defineProperties(component, {
        form: {get: () => internals.form, enumerable: true},
        validity: {get: () => internals.validity, enumerable: true},
        validationMessage: {get: () => internals.validationMessage, enumerable: true},
        willValidate: {get: () => internals.willValidate, enumerable: true},
        name: {get: () => component.getAttribute('name'), enumerable: true},
        type: {get: () => component.localName, enumerable: true},
        labels: {get: () => internals.labels, enumerable: true},
    });

    component.checkValidity = () => internals.checkValidity();
    component.reportValidity = () => internals.reportValidity();
}

/**
 * Exposes form properties and synchronises form state. The component class must declare
 * static formAssociated = true to provide ElementInternals.
 *
 * @param component - The component to make form-associated.
 */
const formAssociatedBehaviour = (component: PPElement) => {
    const internals = component.internals;
    if (!internals) {
        console.error(
            `Form behaviour enabled on ${component.tagName}, but 'internals' was not attached. ` +
            `Ensure the compiler is setting 'static formAssociated = true' on the component class.`
        );
        return;
    }

    const updateFormState = createUpdateFormState(component, internals);

    component._updateFormState = updateFormState;

    component.onUpdated((changedProps: Set<string>) => {
        if (
            changedProps.has('value') ||
            changedProps.has('required') ||
            changedProps.has('pattern') ||
            changedProps.has('min') ||
            changedProps.has('max')
        ) {
            updateFormState();
        }
    });

    component.onConnected(updateFormState);
    component.onAfterRender(updateFormState);

    defineFormProperties(component, internals);
};

registerBehaviour("form", formAssociatedBehaviour);