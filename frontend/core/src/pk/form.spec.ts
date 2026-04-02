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

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { formData, validate, resetForm, setFormValues } from '@/pk/form';

describe('form (PK Form Helpers)', () => {
    let container: HTMLDivElement;

    beforeEach(() => {
        container = document.createElement('div');
        document.body.appendChild(container);
        vi.spyOn(console, 'warn').mockImplementation(() => {});
    });

    afterEach(() => {
        container.remove();
        vi.restoreAllMocks();
    });

    function createForm(html: string): HTMLFormElement {
        container.innerHTML = html;
        return container.querySelector('form')!;
    }

    describe('formData', () => {
        it('should extract simple form values', () => {
            createForm(`
                <form id="test-form">
                    <input name="name" value="John" />
                    <input name="email" value="john@example.com" />
                </form>
            `);

            const data = formData('#test-form');
            const obj = data.toObject();

            expect(obj.name).toBe('John');
            expect(obj.email).toBe('john@example.com');
        });

        it('should handle checkbox values', () => {
            createForm(`
                <form id="test-form">
                    <input type="checkbox" name="agree" value="yes" checked />
                    <input type="checkbox" name="newsletter" value="yes" />
                </form>
            `);

            const data = formData('#test-form');
            const obj = data.toObject();

            expect(obj.agree).toBe('yes');
            expect(obj.newsletter).toBeUndefined();
        });

        it('should handle radio button values', () => {
            createForm(`
                <form id="test-form">
                    <input type="radio" name="color" value="red" />
                    <input type="radio" name="color" value="blue" checked />
                    <input type="radio" name="color" value="green" />
                </form>
            `);

            const data = formData('#test-form');
            const obj = data.toObject();

            expect(obj.color).toBe('blue');
        });

        it('should handle select values', () => {
            createForm(`
                <form id="test-form">
                    <select name="country">
                        <option value="us">USA</option>
                        <option value="uk" selected>UK</option>
                        <option value="ca">Canada</option>
                    </select>
                </form>
            `);

            const data = formData('#test-form');
            expect(data.get('country')).toBe('uk');
        });

        it('should handle array notation for multiple values', () => {
            createForm(`
                <form id="test-form">
                    <input type="checkbox" name="items[]" value="a" checked />
                    <input type="checkbox" name="items[]" value="b" checked />
                    <input type="checkbox" name="items[]" value="c" />
                </form>
            `);

            const data = formData('#test-form');
            const obj = data.toObject();

            expect(obj.items).toEqual(['a', 'b']);
        });

        it('should handle textarea values', () => {
            createForm(`
                <form id="test-form">
                    <textarea name="message">Hello World</textarea>
                </form>
            `);

            const data = formData('#test-form');
            expect(data.get('message')).toBe('Hello World');
        });

        it('should return empty FormDataHandle for invalid selector', () => {
            const data = formData('#non-existent');

            expect(data.toObject()).toEqual({});
            expect(console.warn).toHaveBeenCalled();
        });

        it('should work with HTMLFormElement directly', () => {
            const form = createForm(`
                <form>
                    <input name="test" value="direct" />
                </form>
            `);

            const data = formData(form);
            expect(data.get('test')).toBe('direct');
        });

        describe('FormDataHandle methods', () => {
            it('toJSON should return JSON string', () => {
                createForm(`
                    <form id="test-form">
                        <input name="name" value="John" />
                    </form>
                `);

                const data = formData('#test-form');
                expect(data.toJSON()).toBe('{"name":"John"}');
            });

            it('toFormData should return native FormData', () => {
                createForm(`
                    <form id="test-form">
                        <input name="test" value="value" />
                    </form>
                `);

                const data = formData('#test-form');
                const fd = data.toFormData();

                expect(fd).toBeInstanceOf(FormData);
                expect(fd.get('test')).toBe('value');
            });

            it('has should check if field exists', () => {
                createForm(`
                    <form id="test-form">
                        <input name="exists" value="yes" />
                    </form>
                `);

                const data = formData('#test-form');

                expect(data.has('exists')).toBe(true);
                expect(data.has('missing')).toBe(false);
            });

            it('getAll should return all values for a field', () => {
                createForm(`
                    <form id="test-form">
                        <input type="checkbox" name="items[]" value="a" checked />
                        <input type="checkbox" name="items[]" value="b" checked />
                    </form>
                `);

                const data = formData('#test-form');
                expect(data.getAll('items')).toEqual(['a', 'b']);
            });

            it('getAll should return single value in array', () => {
                createForm(`
                    <form id="test-form">
                        <input name="single" value="value" />
                    </form>
                `);

                const data = formData('#test-form');
                expect(data.getAll('single')).toEqual(['value']);
            });

            it('getAll should return empty array for missing field', () => {
                createForm(`<form id="test-form"></form>`);

                const data = formData('#test-form');
                expect(data.getAll('missing')).toEqual([]);
            });
        });
    });

    describe('validate', () => {
        describe('required rule', () => {
            it('should fail when required field is empty', () => {
                createForm(`
                    <form id="test-form">
                        <input name="name" value="" />
                    </form>
                `);

                const result = validate('#test-form', {
                    name: { required: true }
                });

                expect(result.isValid).toBe(false);
                expect(result.hasError('name')).toBe(true);
                expect(result.getErrors('name')).toContain('This field is required');
            });

            it('should pass when required field has value', () => {
                createForm(`
                    <form id="test-form">
                        <input name="name" value="John" />
                    </form>
                `);

                const result = validate('#test-form', {
                    name: { required: true }
                });

                expect(result.isValid).toBe(true);
                expect(result.hasError('name')).toBe(false);
            });

            it('should fail for whitespace-only value', () => {
                createForm(`
                    <form id="test-form">
                        <input name="name" value="   " />
                    </form>
                `);

                const result = validate('#test-form', {
                    name: { required: true }
                });

                expect(result.isValid).toBe(false);
            });
        });

        describe('minLength/maxLength rules', () => {
            it('should fail when below minLength', () => {
                createForm(`
                    <form id="test-form">
                        <input name="username" value="ab" />
                    </form>
                `);

                const result = validate('#test-form', {
                    username: { minLength: 3 }
                });

                expect(result.isValid).toBe(false);
                expect(result.getErrors('username')[0]).toContain('too short');
            });

            it('should pass when at or above minLength', () => {
                createForm(`
                    <form id="test-form">
                        <input name="username" value="abc" />
                    </form>
                `);

                const result = validate('#test-form', {
                    username: { minLength: 3 }
                });

                expect(result.isValid).toBe(true);
            });

            it('should fail when above maxLength', () => {
                createForm(`
                    <form id="test-form">
                        <input name="code" value="12345" />
                    </form>
                `);

                const result = validate('#test-form', {
                    code: { maxLength: 4 }
                });

                expect(result.isValid).toBe(false);
                expect(result.getErrors('code')[0]).toContain('too long');
            });

            it('should skip length validation for empty non-required field', () => {
                createForm(`
                    <form id="test-form">
                        <input name="optional" value="" />
                    </form>
                `);

                const result = validate('#test-form', {
                    optional: { minLength: 5 }
                });

                expect(result.isValid).toBe(true);
            });
        });

        describe('min/max rules', () => {
            it('should fail when number below min', () => {
                createForm(`
                    <form id="test-form">
                        <input name="age" type="number" value="15" />
                    </form>
                `);

                const result = validate('#test-form', {
                    age: { min: 18 }
                });

                expect(result.isValid).toBe(false);
                expect(result.getErrors('age')[0]).toContain('too small');
            });

            it('should pass when number at or above min', () => {
                createForm(`
                    <form id="test-form">
                        <input name="age" type="number" value="18" />
                    </form>
                `);

                const result = validate('#test-form', {
                    age: { min: 18 }
                });

                expect(result.isValid).toBe(true);
            });

            it('should fail when number above max', () => {
                createForm(`
                    <form id="test-form">
                        <input name="quantity" type="number" value="101" />
                    </form>
                `);

                const result = validate('#test-form', {
                    quantity: { max: 100 }
                });

                expect(result.isValid).toBe(false);
            });
        });

        describe('pattern rule', () => {
            it('should validate against regex pattern', () => {
                createForm(`
                    <form id="test-form">
                        <input name="code" value="ABC123" />
                    </form>
                `);

                const result = validate('#test-form', {
                    code: { pattern: /^[A-Z]{3}[0-9]{3}$/ }
                });

                expect(result.isValid).toBe(true);
            });

            it('should fail for non-matching pattern', () => {
                createForm(`
                    <form id="test-form">
                        <input name="code" value="abc123" />
                    </form>
                `);

                const result = validate('#test-form', {
                    code: { pattern: /^[A-Z]{3}[0-9]{3}$/ }
                });

                expect(result.isValid).toBe(false);
            });

            it('should accept string pattern', () => {
                createForm(`
                    <form id="test-form">
                        <input name="code" value="ABC" />
                    </form>
                `);

                const result = validate('#test-form', {
                    code: { pattern: '^[A-Z]+$' }
                });

                expect(result.isValid).toBe(true);
            });
        });

        describe('format rule', () => {
            it('should validate email format', () => {
                createForm(`
                    <form id="test-form">
                        <input name="email" value="test@example.com" />
                    </form>
                `);

                const result = validate('#test-form', {
                    email: { format: 'email' }
                });

                expect(result.isValid).toBe(true);
            });

            it('should fail for invalid email', () => {
                createForm(`
                    <form id="test-form">
                        <input name="email" value="not-an-email" />
                    </form>
                `);

                const result = validate('#test-form', {
                    email: { format: 'email' }
                });

                expect(result.isValid).toBe(false);
                expect(result.getErrors('email')[0]).toContain('Invalid email');
            });

            it('should validate URL format', () => {
                createForm(`
                    <form id="test-form">
                        <input name="website" value="https://example.com/path" />
                    </form>
                `);

                const result = validate('#test-form', {
                    website: { format: 'url' }
                });

                expect(result.isValid).toBe(true);
            });

            it('should validate phone format', () => {
                createForm(`
                    <form id="test-form">
                        <input name="phone" value="+1-555-123-4567" />
                    </form>
                `);

                const result = validate('#test-form', {
                    phone: { format: 'phone' }
                });

                expect(result.isValid).toBe(true);
            });

            it('should validate date format (YYYY-MM-DD)', () => {
                createForm(`
                    <form id="test-form">
                        <input name="birthdate" value="1990-05-15" />
                    </form>
                `);

                const result = validate('#test-form', {
                    birthdate: { format: 'date' }
                });

                expect(result.isValid).toBe(true);
            });
        });

        describe('custom rule', () => {
            it('should use custom validation function', () => {
                createForm(`
                    <form id="test-form">
                        <input name="password" value="weakpass" />
                    </form>
                `);

                const result = validate('#test-form', {
                    password: {
                        custom: (value) => {
                            const str = String(value);
                            return str.length >= 8 && /[A-Z]/.test(str);
                        }
                    }
                });

                expect(result.isValid).toBe(false);
            });

            it('should use custom error message from function', () => {
                createForm(`
                    <form id="test-form">
                        <input name="password" value="weak" />
                    </form>
                `);

                const result = validate('#test-form', {
                    password: {
                        custom: (value) => {
                            if (String(value).length < 8) {
                                return 'Password must be at least 8 characters';
                            }
                            return true;
                        }
                    }
                });

                expect(result.getErrors('password')).toContain('Password must be at least 8 characters');
            });
        });

        describe('custom message', () => {
            it('should use custom error message', () => {
                createForm(`
                    <form id="test-form">
                        <input name="name" value="" />
                    </form>
                `);

                const result = validate('#test-form', {
                    name: { required: true, message: 'Please enter your name' }
                });

                expect(result.getErrors('name')).toContain('Please enter your name');
            });
        });

        describe('focus method', () => {
            it('should focus first invalid field', () => {
                const form = createForm(`
                    <form id="test-form">
                        <input name="valid" value="has value" />
                        <input name="invalid" value="" id="invalid-field" />
                    </form>
                `);

                const invalidField = form.querySelector('#invalid-field') as HTMLInputElement;
                const focusSpy = vi.spyOn(invalidField, 'focus');

                const result = validate('#test-form', {
                    valid: { required: true },
                    invalid: { required: true }
                });

                result.focus();

                expect(focusSpy).toHaveBeenCalled();
            });
        });
    });

    describe('resetForm', () => {
        it('should reset form to initial state', () => {
            const form = createForm(`
                <form id="test-form">
                    <input name="name" value="" />
                </form>
            `);

            const input = form.querySelector('input')!;
            input.value = 'Modified';

            resetForm('#test-form');

            expect(input.value).toBe('');
        });

        it('should work with HTMLFormElement directly', () => {
            const form = createForm(`
                <form>
                    <input name="test" value="" />
                </form>
            `);

            const input = form.querySelector('input')!;
            input.value = 'Changed';

            resetForm(form);

            expect(input.value).toBe('');
        });
    });

    describe('setFormValues', () => {
        it('should set text input values', () => {
            const form = createForm(`
                <form id="test-form">
                    <input name="name" value="" />
                    <input name="email" value="" />
                </form>
            `);

            setFormValues('#test-form', {
                name: 'John Doe',
                email: 'john@example.com'
            });

            expect((form.querySelector('[name="name"]') as HTMLInputElement).value).toBe('John Doe');
            expect((form.querySelector('[name="email"]') as HTMLInputElement).value).toBe('john@example.com');
        });

        it('should set checkbox values', () => {
            const form = createForm(`
                <form id="test-form">
                    <input type="checkbox" name="agree" value="yes" />
                </form>
            `);

            setFormValues('#test-form', { agree: true });

            expect((form.querySelector('[name="agree"]') as HTMLInputElement).checked).toBe(true);
        });

        it('should set radio button values', () => {
            const form = createForm(`
                <form id="test-form">
                    <input type="radio" name="color" value="red" />
                    <input type="radio" name="color" value="blue" />
                    <input type="radio" name="color" value="green" />
                </form>
            `);

            setFormValues('#test-form', { color: 'blue' });

            const radios = form.querySelectorAll('[name="color"]') as NodeListOf<HTMLInputElement>;
            expect(radios[0].checked).toBe(false);
            expect(radios[1].checked).toBe(true);
            expect(radios[2].checked).toBe(false);
        });

        it('should set select values', () => {
            const form = createForm(`
                <form id="test-form">
                    <select name="country">
                        <option value="us">USA</option>
                        <option value="uk">UK</option>
                        <option value="ca">Canada</option>
                    </select>
                </form>
            `);

            setFormValues('#test-form', { country: 'ca' });

            expect((form.querySelector('select') as HTMLSelectElement).value).toBe('ca');
        });

        it('should set multi-select values', () => {
            const form = createForm(`
                <form id="test-form">
                    <select name="tags" multiple>
                        <option value="a">A</option>
                        <option value="b">B</option>
                        <option value="c">C</option>
                    </select>
                </form>
            `);

            setFormValues('#test-form', { tags: ['a', 'c'] });

            const select = form.querySelector('select') as HTMLSelectElement;
            expect(select.options[0].selected).toBe(true);
            expect(select.options[1].selected).toBe(false);
            expect(select.options[2].selected).toBe(true);
        });

        it('should set textarea values', () => {
            const form = createForm(`
                <form id="test-form">
                    <textarea name="message"></textarea>
                </form>
            `);

            setFormValues('#test-form', { message: 'Hello World' });

            expect((form.querySelector('textarea') as HTMLTextAreaElement).value).toBe('Hello World');
        });

        it('should handle null values gracefully', () => {
            const form = createForm(`
                <form id="test-form">
                    <input name="field" value="original" />
                </form>
            `);

            setFormValues('#test-form', { field: null as unknown as string });

            expect((form.querySelector('input') as HTMLInputElement).value).toBe('');
        });
    });
});
