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
 * Manages form-associated custom element support.
 */
export interface FormAssociation {
    /** Gets the ElementInternals object for form integration. */
    getInternals(): ElementInternals | undefined;

    /** Returns whether form association is enabled for this element. */
    isFormAssociated(): boolean;
}

/**
 * Options for creating a FormAssociation.
 */
export interface FormAssociationOptions {
    /** The host custom element. */
    host: HTMLElement;

    /** Whether form association is enabled for this element. */
    formAssociated: boolean;
}

/**
 * Creates a FormAssociation for form-associated custom element support.
 *
 * @param options - Configuration options including the host and form association flag.
 * @returns A new FormAssociation instance.
 */
export function createFormAssociation(options: FormAssociationOptions): FormAssociation {
    let internals: ElementInternals | undefined;

    if (options.formAssociated) {
        internals = options.host.attachInternals();
    }

    return {
        getInternals(): ElementInternals | undefined {
            return internals;
        },

        isFormAssociated(): boolean {
            return options.formAssociated;
        },
    };
}
