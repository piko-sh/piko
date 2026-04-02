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

/** Record type for props objects. */
type PropsRecord = Record<string, unknown>;

/** Length of the 'pe:' event prefix. */
const PREFIX_PE_LENGTH = 3;

/** Length of the 'on' event prefix. */
const PREFIX_ON_LENGTH = 2;

/**
 * Reserved prop names that have special handling.
 */
export const RESERVED_PROPS = ['_k', '_c', '_s', 'class', '_class', 'style', '_style'] as const;

/**
 * Union of reserved prop name literals.
 */
export type ReservedProp = typeof RESERVED_PROPS[number];

/**
 * Checks if a prop name is reserved for special handling.
 *
 * @param propName - The property name to check.
 * @returns True if the property is reserved.
 */
export function isReservedProp(propName: string): propName is ReservedProp {
    return RESERVED_PROPS.includes(propName as ReservedProp);
}

/**
 * Categorises a prop name into its type.
 */
export type PropCategory =
    | { type: 'boolean-attr'; attrName: string }
    | { type: 'event'; eventName: string; prefix: 'on' | 'pe'; listenerOptions?: AddEventListenerOptions }
    | { type: 'ref' }
    | { type: 'reserved' }
    | { type: 'standard' };

/**
 * Parses a raw event name portion (after stripping the on/pe: prefix) into
 * the event name and optional listener options encoded as $-delimited suffixes.
 *
 * @param raw - The raw string after the prefix (e.g. "Click$capture$passive").
 * @returns The event name and optional listener options.
 */
function parseEventPropName(raw: string): { eventName: string; listenerOptions?: AddEventListenerOptions } {
    const delimiterIndex = raw.indexOf('$');
    if (delimiterIndex === -1) {
        return {eventName: raw};
    }
    const eventName = raw.slice(0, delimiterIndex);
    const opts: AddEventListenerOptions = {};
    for (const s of raw.slice(delimiterIndex + 1).split('$')) {
        if (s === 'capture') { opts.capture = true; }
        if (s === 'passive') { opts.passive = true; }
    }
    return {eventName, listenerOptions: opts};
}

/**
 * Categorises a prop name into its handling type.
 *
 * @param propName - The property name to categorise.
 * @returns The category describing how the prop should be handled.
 */
export function categoriseProp(propName: string): PropCategory {
    if (isReservedProp(propName)) {
        return {type: 'reserved'};
    }
    if (propName.startsWith('?')) {
        return {type: 'boolean-attr', attrName: propName.slice(1)};
    }
    if (propName.startsWith('on')) {
        const {eventName, listenerOptions} = parseEventPropName(propName.slice(PREFIX_ON_LENGTH));
        return {type: 'event', eventName: eventName.toLowerCase(), prefix: 'on', listenerOptions};
    }
    if (propName.startsWith('pe:')) {
        const {eventName, listenerOptions} = parseEventPropName(propName.slice(PREFIX_PE_LENGTH));
        return {type: 'event', eventName: eventName.toLowerCase(), prefix: 'pe', listenerOptions};
    }
    if (propName === '_ref') {
        return {type: 'ref'};
    }
    return {type: 'standard'};
}

/**
 * Determines whether a prop should be considered for removal.
 *
 * @param propName - The property name to check.
 * @param oldValue - The previous value of the property.
 * @param newProps - The new props object.
 * @returns True if the prop is absent from newProps or its value has changed.
 */
export function shouldRemoveProp(
    propName: string,
    oldValue: unknown,
    newProps: PropsRecord
): boolean {
    return !(propName in newProps) || oldValue !== newProps[propName];
}

/**
 * Determines whether a prop value needs to be updated.
 *
 * The value prop always returns true because input elements require
 * special DOM property synchronisation.
 *
 * @param propName - The property name to check.
 * @param oldValue - The previous value.
 * @param newValue - The new value.
 * @returns True if the prop should be updated.
 */
export function shouldUpdateProp(
    propName: string,
    oldValue: unknown,
    newValue: unknown
): boolean {
    if (isReservedProp(propName)) {
        return false;
    }
    if (propName === 'value') {
        return true;
    }
    return newValue !== oldValue;
}

/**
 * Parses class data into a string. Handles strings, arrays, and object formats.
 *
 * @param value - The class value (string, array, or object).
 * @returns The parsed class string.
 */
export function parseClassData(value: unknown): string {
    let classes = '';

    if (typeof value === 'string' && value) {
        classes += ` ${value}`;
    } else if (Array.isArray(value)) {
        for (const item of value) {
            const nestedClass = parseClassData(item);
            if (nestedClass) {
                classes += ` ${nestedClass}`;
            }
        }
    } else if (typeof value === 'object' && value !== null) {
        const objValue = value as Record<string, unknown>;
        for (const key in objValue) {
            if (Object.prototype.hasOwnProperty.call(objValue, key) && objValue[key]) {
                classes += ` ${key}`;
            }
        }
    }

    return classes.trim();
}

/**
 * Parses style data into a CSS string. Handles strings and object formats.
 *
 * @param styleValue - The style value (string or object).
 * @returns The parsed CSS string.
 */
export function parseStyleData(styleValue: unknown): string {
    if (!styleValue) {
        return '';
    }
    if (typeof styleValue === 'string') {
        return styleValue;
    }
    if (typeof styleValue === 'object') {
        const entries = Object.entries(styleValue as Record<string, unknown>)
            .filter(([, value]) => value != null);

        if (entries.length === 0) {
            return '';
        }

        return `${entries
            .map(([key, value]) => {
                const cssKey = key.replace(/([A-Z])/g, '-$1').toLowerCase();
                return `${cssKey}: ${value}`;
            })
            .join('; ')};`;
    }
    return '';
}

/**
 * Combines static and dynamic class values.
 *
 * @param staticClass - The static class string.
 * @param dynamicClass - The dynamic class string.
 * @returns The combined class string.
 */
export function combineClasses(staticClass: string, dynamicClass: string): string {
    const sClass = staticClass.trim();
    const dClass = dynamicClass.trim();

    if (!sClass && !dClass) {
        return '';
    }
    if (!sClass) {
        return dClass;
    }
    if (!dClass) {
        return sClass;
    }
    return `${sClass} ${dClass}`;
}

/**
 * Combines static and dynamic style values.
 *
 * @param staticStyle - The static style string.
 * @param dynamicStyle - The dynamic style string.
 * @returns The combined style string.
 */
export function combineStyles(staticStyle: string, dynamicStyle: string): string {
    const sStyle = staticStyle.replace(/;+\s*$/, '').trim();
    const dStyle = dynamicStyle.replace(/^;+|;+\s*$/g, '').trim();

    if (!sStyle && !dStyle) {
        return '';
    }
    if (!sStyle) {
        return `${dStyle};`;
    }
    if (!dStyle) {
        return `${sStyle};`;
    }
    return `${sStyle}; ${dStyle};`;
}

/**
 * Result of computing final class and style values.
 */
export interface ClassStyleResult {
    /** The computed class attribute value. */
    finalClass: string;
    /** The computed style attribute value. */
    finalStyle: string;
    /** Whether the element should be visible. */
    shouldShow: boolean;
}

/**
 * Computes the final class and style values from props.
 *
 * @param props - The props object containing class and style data.
 * @returns The computed class, style, and visibility values.
 */
export function computeClassStyle(props: PropsRecord): ClassStyleResult {
    const staticClassValue = (props.class ?? '') as string;
    const dynamicClassValue = props._class ?? null;
    const staticStyleValue = (props.style ?? '') as string;
    const dynamicStyleValue = props._style ?? null;

    const finalClass = combineClasses(
        staticClassValue,
        parseClassData(dynamicClassValue)
    ).trim();

    const finalStyle = combineStyles(
        staticStyleValue,
        parseStyleData(dynamicStyleValue)
    ).trim();

    const shouldShow = props._s !== false;

    return {finalClass, finalStyle, shouldShow};
}

/**
 * Represents a single prop change operation.
 */
export type PropChangeOperation =
    | { type: 'remove-attr'; attrName: string }
    | { type: 'remove-boolean-attr'; attrName: string }
    | { type: 'remove-event'; eventName: string; handler: unknown; listenerOptions?: AddEventListenerOptions }
    | { type: 'remove-ref'; refName: string }
    | { type: 'set-boolean-attr'; attrName: string; value: boolean }
    | { type: 'set-event'; eventName: string; oldHandler: unknown; newHandler: unknown; listenerOptions?: AddEventListenerOptions }
    | { type: 'set-ref'; oldRefName: string | null; newRefName: string }
    | { type: 'set-value'; value: string }
    | { type: 'set-attr'; attrName: string; value: string }
    | { type: 'remove-null-attr'; attrName: string };

/**
 * Computes which props need to be removed.
 *
 * @param oldProps - The previous props object.
 * @param newProps - The new props object.
 * @returns An array of removal operations.
 */
export function computePropsToRemove(
    oldProps: PropsRecord,
    newProps: PropsRecord
): PropChangeOperation[] {
    const operations: PropChangeOperation[] = [];

    for (const propName in oldProps) {
        if (!shouldRemoveProp(propName, oldProps[propName], newProps)) {
            continue;
        }

        const category = categoriseProp(propName);

        switch (category.type) {
            case 'boolean-attr':
                operations.push({type: 'remove-boolean-attr', attrName: category.attrName});
                break;
            case 'event':
                operations.push({
                    type: 'remove-event',
                    eventName: category.eventName,
                    handler: oldProps[propName],
                    listenerOptions: category.listenerOptions
                });
                break;
            case 'ref':
                if (oldProps[propName]) {
                    operations.push({type: 'remove-ref', refName: oldProps[propName] as string});
                }
                break;
            case 'reserved':
                break;
            case 'standard':
                operations.push({type: 'remove-attr', attrName: propName});
                break;
        }
    }

    return operations;
}

/**
 * Determines the attribute value string for a prop.
 *
 * Returns null for null, undefined, or false values. Objects are
 * serialised as JSON.
 *
 * @param value - The prop value to convert.
 * @returns The string representation, or null if the attribute should be removed.
 */
export function computeAttrValue(value: unknown): string | null {
    if (value == null || value === false) {
        return null;
    }
    if (typeof value === 'object') {
        try {
            return JSON.stringify(value);
        } catch {
            return String(value);
        }
    }
    return String(value);
}

/**
 * Computes which props need to be added or updated.
 *
 * @param oldProps - The previous props object.
 * @param newProps - The new props object.
 * @returns An array of update operations.
 */
export function computePropsToUpdate(
    oldProps: PropsRecord,
    newProps: PropsRecord
): PropChangeOperation[] {
    const operations: PropChangeOperation[] = [];

    for (const propName in newProps) {
        const newValue = newProps[propName];
        const oldValue = oldProps[propName];

        if (!shouldUpdateProp(propName, oldValue, newValue)) {
            continue;
        }

        const category = categoriseProp(propName);

        switch (category.type) {
            case 'boolean-attr':
                operations.push({
                    type: 'set-boolean-attr',
                    attrName: category.attrName,
                    value: Boolean(newValue)
                });
                break;
            case 'event':
                if (oldValue !== newValue) {
                    operations.push({
                        type: 'set-event',
                        eventName: category.eventName,
                        oldHandler: oldValue,
                        newHandler: newValue,
                        listenerOptions: category.listenerOptions
                    });
                }
                break;
            case 'ref':
                if (newValue && typeof newValue === 'string') {
                    operations.push({
                        type: 'set-ref',
                        oldRefName: oldValue as string | null,
                        newRefName: newValue
                    });
                }
                break;
            case 'reserved':
                break;
            case 'standard': {
                if (newValue == null || newValue === false) {
                    operations.push({type: 'remove-null-attr', attrName: propName});
                    break;
                }
                const attrValue = computeAttrValue(newValue);
                if (attrValue !== null) {
                    operations.push({type: 'set-attr', attrName: propName, value: attrValue});
                }
                break;
            }
        }
    }

    return operations;
}

/**
 * Full prop diff result containing all operations needed to update an element.
 */
export interface PropDiffResult {
    /** Operations to remove old props. */
    removals: PropChangeOperation[];
    /** Operations to add or update new props. */
    updates: PropChangeOperation[];
    /** Computed class and style values. */
    classStyle: ClassStyleResult;
}

/**
 * Computes the full diff between old and new props.
 *
 * @param oldProps - The old props object.
 * @param newProps - The new props object.
 * @returns The diff result with removal and update operations.
 */
export function computePropDiff(
    oldProps: PropsRecord,
    newProps: PropsRecord
): PropDiffResult {
    return {
        removals: computePropsToRemove(oldProps, newProps),
        updates: computePropsToUpdate(oldProps, newProps),
        classStyle: computeClassStyle(newProps)
    };
}
