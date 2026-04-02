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

import type {PropTypeDefinition} from "../types";

/**
 * Manages property type definitions and attribute name derivation.
 */
export interface PropTypeRegistry {
    /** Gets the prop type definition for a property. */
    get(propertyName: string): PropTypeDefinition | undefined;

    /** Gets all registered property names. */
    getPropertyNames(): string[];

    /** Derives the list of observed attribute names from prop types. */
    deriveObservedAttributes(): string[];

    /** Gets the default value for a property, evaluating factory functions to produce fresh values. */
    getDefaultValue(propertyName: string): unknown;

    /** Returns whether a property should reflect to attributes. */
    shouldReflect(propertyName: string): boolean;

    /** Converts a property name to attribute name (camelCase to kebab-case). */
    propertyToAttributeName(propertyName: string): string;

    /** Converts an attribute name to property name (kebab-case to camelCase). */
    attributeToPropertyName(attributeName: string): string;
}

/**
 * Options for creating a PropTypeRegistry.
 */
export interface PropTypeRegistryOptions {
    /** Property type definitions map. */
    propTypes: Record<string, PropTypeDefinition> | undefined;
}

/**
 * Converts a property name to attribute name (camelCase to kebab-case).
 *
 * @param propertyName - The camelCase property name.
 * @returns The kebab-case attribute name.
 */
export function propertyToAttributeName(propertyName: string): string {
    return propertyName.replace(/[A-Z]/g, (match) => `-${match.toLowerCase()}`);
}

/**
 * Converts an attribute name to a property name (kebab-case to camelCase).
 *
 * Attempts to match against known propTypes first, then falls back to standard conversion.
 *
 * @param attributeName - The kebab-case attribute name.
 * @param propTypes - The registered property type definitions to match against.
 * @returns The camelCase property name.
 */
function attributeToPropertyNameFn(
    attributeName: string,
    propTypes: Partial<Record<string, PropTypeDefinition>>
): string {
    for (const propName in propTypes) {
        if (propertyToAttributeName(propName) === attributeName) {
            return propName;
        }
    }
    return attributeName.replace(/-([a-z])/g, (_match, letter: string) => letter.toUpperCase());
}

/**
 * Checks whether a property should reflect its value to an HTML attribute.
 *
 * Primitive types (string, number, boolean) reflect by default.
 *
 * @param propDef - The property type definition to check.
 * @returns Whether the property should reflect to an attribute.
 */
export function shouldReflectProperty(propDef: PropTypeDefinition | undefined): boolean {
    if (!propDef) {
        return false;
    }
    if (propDef.reflectToAttribute === true) {
        return true;
    }
    if (propDef.reflectToAttribute === false) {
        return false;
    }
    return propDef.type === "string" || propDef.type === "number" || propDef.type === "boolean";
}

/**
 * Creates a PropTypeRegistry for managing property type definitions.
 *
 * @param options - Configuration options including prop type definitions.
 * @returns A new PropTypeRegistry instance.
 */
export function createPropTypeRegistry(options: PropTypeRegistryOptions): PropTypeRegistry {
    const propTypes: Partial<Record<string, PropTypeDefinition>> = options.propTypes ?? {};

    return {
        get(propertyName: string): PropTypeDefinition | undefined {
            return propTypes[propertyName];
        },

        getPropertyNames(): string[] {
            return Object.keys(propTypes);
        },

        deriveObservedAttributes(): string[] {
            const attributesToObserve: string[] = [];
            for (const propName in propTypes) {
                const propDef = propTypes[propName];
                if (shouldReflectProperty(propDef)) {
                    attributesToObserve.push(propertyToAttributeName(propName));
                }
            }
            return attributesToObserve;
        },

        getDefaultValue(propertyName: string): unknown {
            const propDef = propTypes[propertyName];
            if (propDef?.default === undefined) {
                return undefined;
            }
            return typeof propDef.default === "function"
                ? (propDef.default as () => unknown)()
                : propDef.default;
        },

        shouldReflect(propertyName: string): boolean {
            return shouldReflectProperty(propTypes[propertyName]);
        },

        propertyToAttributeName,

        attributeToPropertyName(attributeName: string): string {
            return attributeToPropertyNameFn(attributeName, propTypes);
        },
    };
}
