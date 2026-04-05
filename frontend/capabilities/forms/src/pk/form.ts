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

/** Handle for accessing and serialising form data. */
export interface FormDataHandle {
    /** Converts form data to a plain object. */
    toObject(): Record<string, unknown>;
    /** Returns the native FormData object. */
    toFormData(): FormData;
    /** Converts form data to a JSON string. */
    toJSON(): string;
    /** Returns the value for a specific field. */
    get(key: string): unknown;
    /** Checks whether a field exists. */
    has(key: string): boolean;
    /** Returns all values for a field (useful for multi-select/checkboxes). */
    getAll(key: string): unknown[];
}

/** Validation rule for a single form field. */
export interface ValidationRule {
    /** Field is required. */
    required?: boolean;
    /** Minimum length for strings. */
    minLength?: number;
    /** Maximum length for strings. */
    maxLength?: number;
    /** Minimum value for numbers. */
    min?: number;
    /** Maximum value for numbers. */
    max?: number;
    /** Regex pattern to match. */
    pattern?: RegExp | string;
    /** Predefined format: 'email', 'url', 'phone', 'date'. */
    format?: 'email' | 'url' | 'phone' | 'date';
    /** Custom validation function. */
    custom?: (value: unknown) => boolean | string;
    /** Custom error message. */
    message?: string;
}

/** Mapping of field names to their validation rules. */
export interface ValidationRules {
    /** Validation rule for the named field. */
    [field: string]: ValidationRule;
}

/** Result of form validation. */
export interface ValidationResult {
    /** Whether all fields passed validation. */
    isValid: boolean;
    /** Error messages by field name. */
    errors: Record<string, string[]>;
    /** Focuses the first invalid field. */
    focus(): void;
    /** Returns errors for a specific field. */
    getErrors(field: string): string[];
    /** Checks whether a specific field has errors. */
    hasError(field: string): boolean;
}

/** Regular expressions for built-in format validation rules. */
const PATTERNS: Record<string, RegExp> = {
    email: /^[^\s@]+@[^\s@]+\.[^\s@]+$/,
    url: /^https?:\/\/[^\s/$.?#].[^\s]*$/i,
    phone: /^[+]?[(]?[0-9]{1,4}[)]?[-\s./0-9]*$/,
    date: /^\d{4}-\d{2}-\d{2}$/
};

/** Default human-readable error messages for each validation constraint. */
const DEFAULT_MESSAGES: Record<string, string> = {
    required: 'This field is required',
    minLength: 'Value is too short',
    maxLength: 'Value is too long',
    min: 'Value is too small',
    max: 'Value is too large',
    pattern: 'Invalid format',
    email: 'Invalid email address',
    url: 'Invalid URL',
    phone: 'Invalid phone number',
    date: 'Invalid date format (YYYY-MM-DD)'
};

/**
 * Resolves a selector to a form element.
 *
 * @param selector - CSS selector or HTMLFormElement.
 * @returns The resolved form element, or null if not found.
 */
function resolveForm(selector: string | HTMLFormElement): HTMLFormElement | null {
    if (selector instanceof HTMLFormElement) {
        return selector;
    }

    const element = document.querySelector(selector);
    if (element instanceof HTMLFormElement) {
        return element;
    }

    console.warn(`[pk] formData: "${selector}" is not a form element`);
    return null;
}

/**
 * Converts FormData to a plain object, handling array fields.
 *
 * @param fd - The FormData to convert.
 * @returns A plain object representation.
 */
function formDataToObject(fd: FormData): Record<string, unknown> {
    const result: Record<string, unknown> = {};

    fd.forEach((value, key) => {
        const arrayMatch = key.match(/^(.+)\[\]$/);
        const actualKey = arrayMatch ? arrayMatch[1] : key;

        if (arrayMatch || result[actualKey] !== undefined) {
            const existing = result[actualKey];
            if (Array.isArray(existing)) {
                existing.push(value);
            } else if (existing !== undefined) {
                result[actualKey] = [existing, value];
            } else {
                result[actualKey] = [value];
            }
        } else {
            result[actualKey] = value;
        }
    });

    return result;
}

/**
 * Converts a value to a string for validation.
 *
 * @param value - The value to convert.
 * @returns The string representation, or empty string for null/undefined.
 */
function getStringValue(value: unknown): string {
    if (value === null || value === undefined) {
        return '';
    }
    return String(value);
}

/**
 * Validates length constraints against minLength and maxLength rules.
 *
 * @param strValue - The string value to validate.
 * @param rule - The validation rule containing length constraints.
 * @returns Error message string, or null if valid.
 */
function validateLength(strValue: string, rule: ValidationRule): string | null {
    if (rule.minLength !== undefined && strValue.length < rule.minLength) {
        return rule.message ?? `${DEFAULT_MESSAGES.minLength} (minimum ${rule.minLength} characters)`;
    }
    if (rule.maxLength !== undefined && strValue.length > rule.maxLength) {
        return rule.message ?? `${DEFAULT_MESSAGES.maxLength} (maximum ${rule.maxLength} characters)`;
    }
    return null;
}

/**
 * Validates numeric range constraints against min and max rules.
 *
 * @param value - The value to validate.
 * @param rule - The validation rule containing range constraints.
 * @returns Error message string, or null if valid.
 */
function validateNumericRange(value: unknown, rule: ValidationRule): string | null {
    const numValue = Number(value);
    if (isNaN(numValue)) {
        return null;
    }
    if (rule.min !== undefined && numValue < rule.min) {
        return rule.message ?? `${DEFAULT_MESSAGES.min} (minimum ${rule.min})`;
    }
    if (rule.max !== undefined && numValue > rule.max) {
        return rule.message ?? `${DEFAULT_MESSAGES.max} (maximum ${rule.max})`;
    }
    return null;
}

/**
 * Validates the value against a regex pattern constraint.
 *
 * @param strValue - The string value to validate.
 * @param rule - The validation rule containing the pattern.
 * @returns Error message string, or null if valid.
 */
function validatePattern(strValue: string, rule: ValidationRule): string | null {
    if (rule.pattern === undefined) {
        return null;
    }
    const pattern = typeof rule.pattern === 'string' ? new RegExp(rule.pattern) : rule.pattern;
    if (!pattern.test(strValue)) {
        return rule.message ?? DEFAULT_MESSAGES.pattern;
    }
    return null;
}

/**
 * Validates the value against a predefined format (email, url, phone, date).
 *
 * @param strValue - The string value to validate.
 * @param rule - The validation rule containing the format.
 * @returns Error message string, or null if valid.
 */
function validateFormat(strValue: string, rule: ValidationRule): string | null {
    if (rule.format === undefined) {
        return null;
    }
    const formatPattern = PATTERNS[rule.format];
    if (!formatPattern.test(strValue)) {
        return rule.message ?? DEFAULT_MESSAGES[rule.format];
    }
    return null;
}

/**
 * Validates the value against a custom validation function.
 *
 * @param value - The value to validate.
 * @param rule - The validation rule containing the custom function.
 * @returns Error message string, or null if valid.
 */
function validateCustom(value: unknown, rule: ValidationRule): string | null {
    if (rule.custom === undefined) {
        return null;
    }
    const customResult = rule.custom(value);
    if (customResult === false) {
        return rule.message ?? 'Validation failed';
    }
    if (typeof customResult === 'string') {
        return customResult;
    }
    return null;
}

/**
 * Validates a single field value against a validation rule.
 *
 * @param value - The field value to validate.
 * @param rule - The validation rule to apply.
 * @returns Array of error messages (empty if valid).
 */
function validateField(value: unknown, rule: ValidationRule): string[] {
    const errors: string[] = [];
    const strValue = getStringValue(value);
    const isEmpty = strValue.trim() === '';

    if (rule.required && isEmpty) {
        errors.push(rule.message ?? DEFAULT_MESSAGES.required);
        return errors;
    }

    if (isEmpty) {
        return errors;
    }

    const validationResults = [
        validateLength(strValue, rule),
        validateNumericRange(value, rule),
        validatePattern(strValue, rule),
        validateFormat(strValue, rule),
        validateCustom(value, rule)
    ];

    for (const error of validationResults) {
        if (error !== null) {
            errors.push(error);
        }
    }

    return errors;
}

/**
 * Creates a FormDataHandle for easy form data access.
 *
 * @param selector - Form selector or HTMLFormElement.
 * @returns FormDataHandle for accessing form data.
 */
export function formData(selector: string | HTMLFormElement): FormDataHandle {
    const form = resolveForm(selector);
    const fd = form ? new FormData(form) : new FormData();
    const formObject = formDataToObject(fd);

    return {
        toObject(): Record<string, unknown> {
            return {...formObject};
        },

        toFormData(): FormData {
            return form ? new FormData(form) : new FormData();
        },

        toJSON(): string {
            return JSON.stringify(formObject);
        },

        get(key: string): unknown {
            return formObject[key];
        },

        has(key: string): boolean {
            return key in formObject;
        },

        getAll(key: string): unknown[] {
            const value = formObject[key];
            if (Array.isArray(value)) {
                return value;
            }
            if (value !== undefined) {
                return [value];
            }
            return [];
        }
    };
}

/**
 * Validates a form against a set of rules.
 *
 * @param selector - Form selector or HTMLFormElement.
 * @param rules - Validation rules by field name.
 * @returns ValidationResult with isValid, errors, and helper methods.
 */
export function validate(
    selector: string | HTMLFormElement,
    rules: ValidationRules = {}
): ValidationResult {
    const form = resolveForm(selector);
    const data = formData(selector);
    const formObject = data.toObject();
    const errors: Record<string, string[]> = {};
    let firstInvalidField: string | null = null;

    for (const [field, rule] of Object.entries(rules)) {
        const fieldErrors = validateField(formObject[field], rule);
        if (fieldErrors.length > 0) {
            errors[field] = fieldErrors;
            firstInvalidField ??= field;
        }
    }

    const isValid = Object.keys(errors).length === 0;

    return {
        isValid,
        errors,

        focus(): void {
            if (!form || !firstInvalidField) {
                return;
            }

            const field = form.elements.namedItem(firstInvalidField);
            if (field instanceof HTMLElement && 'focus' in field) {
                (field as HTMLElement).focus();
            }
        },

        getErrors(field: string): string[] {
            return errors[field] ?? [];
        },

        hasError(field: string): boolean {
            return field in errors && errors[field].length > 0;
        }
    };
}

/**
 * Resets a form to its initial state.
 *
 * @param selector - Form selector or HTMLFormElement.
 */
export function resetForm(selector: string | HTMLFormElement): void {
    const form = resolveForm(selector);
    if (form) {
        form.reset();
    }
}

/**
 * Sets the value on a single HTMLInputElement.
 *
 * Handles checkbox checked state and text/number input values.
 *
 * @param input - The input element.
 * @param value - The value to set.
 */
function setInputValue(input: HTMLInputElement, value: unknown): void {
    if (input.type === 'checkbox') {
        input.checked = Boolean(value);
    } else if (input.type !== 'file') {
        input.value = String(value ?? '');
    }
}

/**
 * Sets the value on elements in a RadioNodeList (checkboxes/radios with same name).
 *
 * @param nodeList - The RadioNodeList containing the elements.
 * @param value - The value or array of values to match.
 */
function setRadioNodeListValue(nodeList: RadioNodeList, value: unknown): void {
    for (const element of Array.from(nodeList)) {
        if (!(element instanceof HTMLInputElement)) {
            continue;
        }
        if (element.type === 'checkbox') {
            element.checked = Array.isArray(value)
                ? value.includes(element.value)
                : element.value === String(value);
        } else if (element.type === 'radio') {
            element.checked = element.value === String(value);
        }
    }
}

/**
 * Sets the value on a select element.
 *
 * Handles both single and multiple select elements.
 *
 * @param select - The select element.
 * @param value - The value or array of values to select.
 */
function setSelectValue(select: HTMLSelectElement, value: unknown): void {
    if (select.multiple && Array.isArray(value)) {
        for (const option of Array.from(select.options)) {
            option.selected = value.includes(option.value);
        }
    } else {
        select.value = String(value ?? '');
    }
}

/**
 * Sets form field values programmatically.
 *
 * @param selector - Form selector or HTMLFormElement.
 * @param values - Object with field names and values to set.
 */
export function setFormValues(
    selector: string | HTMLFormElement,
    values: Record<string, unknown>
): void {
    const form = resolveForm(selector);
    if (!form) {
        return;
    }

    for (const [name, value] of Object.entries(values)) {
        const elements = form.elements.namedItem(name);

        if (!elements) {
            continue;
        }

        if (elements instanceof RadioNodeList) {
            setRadioNodeListValue(elements, value);
            continue;
        }

        if (elements instanceof HTMLInputElement) {
            setInputValue(elements, value);
        } else if (elements instanceof HTMLSelectElement) {
            setSelectValue(elements, value);
        } else if (elements instanceof HTMLTextAreaElement) {
            elements.value = String(value ?? '');
        }
    }
}
