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
 * Creates a closure that synchronises form value and validity state.
 *
 * Validation follows a two-tier priority:
 *
 * 1. If the component exposes a `customValidity()` method, it is called first.
 *    A non-null result sets explicit validity; a null result clears validity.
 * 2. Otherwise, a native form element is located (explicit `nativeInput` ref
 *    takes precedence over a shadow DOM query) and its validity is forwarded
 *    to ElementInternals. The native element also serves as the focus anchor
 *    when delegatesFocus is enabled on the shadow root.
 *
 * @param component - The PPElement component to observe.
 * @param internals - The ElementInternals instance attached to the component.
 * @returns A parameterless function that updates the form state.
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
 * Attaches the four standard form lifecycle callbacks to the component.
 *
 * Each callback is dynamically assigned because PPElement does not
 * declare them in its type definition. The callbacks handle form
 * association (no-op), disabled-state propagation, form reset
 * (restores the initial value), and browser state restoration.
 *
 * @param component - The PPElement component to augment.
 * @param updateFormState - The closure that synchronises form value and validity.
 */
function attachFormLifecycleCallbacks(component: PPElement, updateFormState: () => void): void {
    component.formAssociatedCallback = (_form: HTMLFormElement | null) => {
    };

    component.formDisabledCallback = (disabled: boolean) => {
        if (component.state) {
            component.state.disabled = disabled;
        }
    };

    component.formResetCallback = () => {
        if (component.state && component.$$ctx?.$$initialState) {
            const initialValue = component.$$ctx.state.value;
            component.state.value = initialValue ?? '';
            queueMicrotask(updateFormState);
        }
    };

    component.formStateRestoreCallback = (state: string | File | FormData | null, _mode: 'restore' | 'autocomplete') => {
        if (component.state) {
            component.state.value = state;
            queueMicrotask(updateFormState);
        }
    };
}

/**
 * Defines standard form-related properties and methods on the component.
 *
 * Adds enumerable getters for `form`, `validity`, `validationMessage`,
 * `willValidate`, `name`, `type`, and `labels`, as well as
 * `checkValidity()` and `reportValidity()` methods, all delegating
 * to ElementInternals.
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
 * Applies form-associated behaviour to a PPElement component.
 *
 * Sets up value synchronisation, validation forwarding, form lifecycle
 * callbacks, and standard form properties. The behaviour triggers a
 * form-state update whenever value, required, pattern, min, or max
 * properties change, as well as on connect and after each render.
 *
 * Requires that `ElementInternals` has already been attached to the
 * component (i.e. the class must declare `static formAssociated = true`).
 *
 * @param component - The PPElement component to augment.
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

    attachFormLifecycleCallbacks(component, updateFormState);
    defineFormProperties(component, internals);
};

registerBehaviour("form", formAssociatedBehaviour);