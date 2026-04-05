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

/** Supported property types for attribute coercion. */
export type PropType = "string" | "number" | "boolean" | "array" | "object" | "json" | "any";

/** Defines the type, default value, and reflection behaviour for a component property. */
export interface PropTypeDefinition {
    /** The property type for attribute coercion. */
    type?: PropType;
    /** Default value for the property. */
    default?: unknown;
    /** Whether to reflect the property value to an HTML attribute. */
    reflectToAttribute?: boolean;
    /** For array types, the type of items in the array. */
    itemType?: string;
    /** For map types, the key and value type definitions. */
    mapDefinition?: { keyType: string; valueType: string };
    /** Whether the property can be null. */
    nullable?: boolean;
}

/** Context object passed to PPElement.init(). */
export interface StateContext {
    /** The component's reactive state object. */
    state: Record<string, unknown>;
    /** Copy of the initial state for form reset. */
    $$initialState?: Record<string, unknown>;
    /** Additional arbitrary properties. */
    [key: string]: unknown;
}

/** Custom validity result returned by a component's customValidity() method. */
export interface CustomValidityResult {
    /** The ValidityState flags to set (e.g., { valueMissing: true }). */
    validity: ValidityStateFlags;
    /** The validation message to show to the user. */
    message: string;
    /** Optional element to focus when validation fails. */
    anchor?: HTMLElement;
}

/** Callback that accepts no arguments and returns nothing. */
export type VoidCallback = () => void;

/** Callback invoked after properties change, receiving the set of changed property names. */
export type UpdatedCallback = (changedProperties: Set<string>) => void;

/** Callback invoked when slot content changes, receiving the currently assigned elements. */
export type SlotChangeCallback = (elements: Element[]) => void;
