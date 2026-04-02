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

import type {PropType} from '@/element/types';
import {shouldReflectProperty, type PropTypeRegistry} from '@/element/services/PropTypeRegistry';

/**
 * Manages two-way synchronisation between HTML attributes and component state.
 */
export interface AttributeSyncService {
    /** Applies an HTML attribute value to component state. */
    applyAttributeToState(propertyName: string, attributeValue: string | null): void;

    /** Reflects a state property value to an HTML attribute. */
    reflectStateToAttribute(propertyName: string, propertyValue: unknown): void;

    /** Handles attributeChangedCallback from the component. */
    handleAttributeChanged(
        attributeName: string,
        oldValue: string | null,
        newValue: string | null
    ): void;

    /** Translates a raw attribute string value to a typed value. */
    translateAttributeValue(
        typeHint: PropType,
        attributeValue: string | null,
        propertyName: string,
        isNullable?: boolean
    ): unknown;

    /** Syncs all current attributes to state during initialisation. */
    syncAllAttributesToState(): void;

    /** Reflects all state properties to attributes during initialisation. */
    reflectAllStateToAttributes(): void;

    /** Returns whether currently applying to state to prevent loops. */
    isApplyingToState(): boolean;

    /** Returns whether currently reflecting to attribute to prevent loops. */
    isReflectingToAttribute(): boolean;

    /** Returns whether currently in initialisation phase. */
    isInitialising(): boolean;

    /** Sets the initialisation phase flag. */
    setInitialising(value: boolean): void;
}

/**
 * Options for creating an AttributeSyncService.
 */
export interface AttributeSyncServiceOptions {
    /** The host custom element. */
    host: HTMLElement;

    /** Property type registry for type information. */
    propTypeRegistry: PropTypeRegistry;

    /** Function to get the current reactive state. */
    getState: () => Record<string, unknown> | undefined;

    /** Function to get default property values. */
    getDefaults: () => Record<string, unknown>;
}

/**
 * Parses a number from a raw attribute string value.
 *
 * @param attributeValue - The raw attribute string.
 * @param defaultValue - The fallback value if parsing fails.
 * @returns The parsed number, or the default if the string is not a valid number.
 */
function parseNumberAttribute(attributeValue: string, defaultValue: unknown): number {
    const num = parseFloat(attributeValue);
    if (isNaN(num)) {
        return typeof defaultValue === "number" ? defaultValue : 0;
    }
    return num;
}

/**
 * Parses a JSON string from a raw attribute value.
 *
 * @param attributeValue - The raw attribute string containing JSON.
 * @param typeHint - The expected property type, used to determine fallback on parse failure.
 * @param defaultValue - The fallback value if parsing fails and no default is defined.
 * @returns The parsed value, or a fallback (default, empty array, or null).
 */
function parseJsonAttribute(
    attributeValue: string,
    typeHint: PropType,
    defaultValue: unknown
): unknown {
    try {
        return JSON.parse(attributeValue) as unknown;
    } catch {
        if (defaultValue !== undefined) {
            return defaultValue;
        }
        return typeHint === "array" ? [] : null;
    }
}

/**
 * Translates a raw attribute string to a typed value based on the property type hint.
 *
 * @param typeHint - The target property type.
 * @param attributeValue - The raw attribute string, or null if the attribute was removed.
 * @param defaultValue - The fallback value when the attribute is absent.
 * @param isNullable - Whether the property accepts null values.
 * @returns The coerced typed value.
 */
function translateValue(
    typeHint: PropType,
    attributeValue: string | null,
    defaultValue: unknown,
    isNullable: boolean
): unknown {
    if (attributeValue === null) {
        if (typeHint === "boolean") {
            return false;
        }
        return isNullable ? null : defaultValue;
    }

    switch (typeHint) {
        case "boolean":
            return attributeValue !== "false";
        case "number":
            return parseNumberAttribute(attributeValue, defaultValue);
        case "string":
            return attributeValue;
        case "array":
        case "object":
        case "json":
            return parseJsonAttribute(attributeValue, typeHint, defaultValue);
        default:
            return attributeValue;
    }
}

/**
 * Converts a property value to a string suitable for HTML attribute reflection.
 *
 * @param propType - The property type controlling serialisation format.
 * @param propertyValue - The value to convert.
 * @returns The string representation for the attribute.
 */
function valueToAttributeString(propType: PropType, propertyValue: unknown): string {
    if (propType === "boolean") {
        return propertyValue === true ? "true" : "false";
    }
    if (propType === "json" || propType === "object" || propType === "array") {
        return JSON.stringify(propertyValue);
    }
    return String(propertyValue);
}

/**
 * Tracks synchronisation direction flags to prevent circular updates.
 */
interface SyncState {
    /** Whether a value is currently being applied from attribute to state. */
    applyingToState: boolean;
    /** Whether a value is currently being reflected from state to attribute. */
    reflectingToAttribute: boolean;
    /** Whether the component is in its initialisation phase. */
    initialising: boolean;
}

/**
 * Applies a raw attribute value to the component state after type coercion.
 *
 * @param propertyName - The property name to update.
 * @param attributeValue - The raw attribute string, or null if removed.
 * @param options - The service configuration options.
 * @param syncState - The synchronisation direction flags.
 */
function applyToState(
    propertyName: string,
    attributeValue: string | null,
    options: AttributeSyncServiceOptions,
    syncState: SyncState
): void {
    const {propTypeRegistry, getState, getDefaults} = options;
    const propDef = propTypeRegistry.get(propertyName);
    const state = getState();
    if (!propDef || !state) {
        return;
    }

    syncState.applyingToState = true;
    try {
        const defaultValue = getDefaults()[propertyName];
        const typedValue = translateValue(
            propDef.type ?? "any",
            attributeValue,
            defaultValue,
            propDef.nullable ?? false
        );
        if (state[propertyName] !== typedValue) {
            state[propertyName] = typedValue;
        }
    } finally {
        syncState.applyingToState = false;
    }
}

/**
 * Reflects a state property value to the corresponding HTML attribute.
 *
 * Uses toggleAttribute for booleans where attribute presence (not value) matters.
 *
 * @param propertyName - The property name to reflect.
 * @param propertyValue - The current property value.
 * @param options - The service configuration options.
 * @param syncState - The synchronisation direction flags.
 */
function reflectToAttribute(
    propertyName: string,
    propertyValue: unknown,
    options: AttributeSyncServiceOptions,
    syncState: SyncState
): void {
    if (syncState.applyingToState && !syncState.initialising) {
        return;
    }

    const {host, propTypeRegistry} = options;
    const propDef = propTypeRegistry.get(propertyName);
    if (!shouldReflectProperty(propDef)) {
        return;
    }

    syncState.reflectingToAttribute = true;
    try {
        const attributeName = propTypeRegistry.propertyToAttributeName(propertyName);
        const propType = propDef?.type ?? "any";

        if (propertyValue === null || propertyValue === undefined) {
            if (host.hasAttribute(attributeName)) {
                host.removeAttribute(attributeName);
            }
        } else if (propType === "boolean") {
            host.toggleAttribute(attributeName, propertyValue === true);
        } else {
            const stringValue = valueToAttributeString(propType, propertyValue);
            if (host.getAttribute(attributeName) !== stringValue) {
                host.setAttribute(attributeName, stringValue);
            }
        }
    } finally {
        syncState.reflectingToAttribute = false;
    }
}

/**
 * Creates an AttributeSyncService for managing two-way attribute/state synchronisation.
 *
 * @param options - Configuration options for the service.
 * @returns A new AttributeSyncService instance.
 */
export function createAttributeSyncService(
    options: AttributeSyncServiceOptions
): AttributeSyncService {
    const syncState: SyncState = {applyingToState: false, reflectingToAttribute: false, initialising: true};

    return {
        applyAttributeToState(propertyName: string, attributeValue: string | null): void {
            applyToState(propertyName, attributeValue, options, syncState);
        },

        reflectStateToAttribute(propertyName: string, propertyValue: unknown): void {
            reflectToAttribute(propertyName, propertyValue, options, syncState);
        },

        handleAttributeChanged(attributeName: string, _oldValue: string | null, newValue: string | null): void {
            if (syncState.reflectingToAttribute || !options.getState() || syncState.initialising) {
                return;
            }
            const propertyName = options.propTypeRegistry.attributeToPropertyName(attributeName);
            if (options.propTypeRegistry.get(propertyName)) {
                this.applyAttributeToState(propertyName, newValue);
            }
        },

        translateAttributeValue(typeHint: PropType, attributeValue: string | null, propertyName: string, isNullable = false): unknown {
            return translateValue(typeHint, attributeValue, options.getDefaults()[propertyName], isNullable);
        },

        syncAllAttributesToState(): void {
            const propNames = options.propTypeRegistry.getPropertyNames();
            for (const attribute of Array.from(options.host.attributes)) {
                const propertyName = options.propTypeRegistry.attributeToPropertyName(attribute.name);
                if (propNames.includes(propertyName)) {
                    this.applyAttributeToState(propertyName, attribute.value);
                }
            }
        },

        reflectAllStateToAttributes(): void {
            const state = options.getState();
            if (!state) {
                return;
            }
            for (const propName in state) {
                this.reflectStateToAttribute(propName, state[propName]);
            }
        },

        isApplyingToState: () => syncState.applyingToState,
        isReflectingToAttribute: () => syncState.reflectingToAttribute,
        isInitialising: () => syncState.initialising,
        setInitialising: (value: boolean) => {
            syncState.initialising = value;
        },
    };
}
