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
import { createRefs, refs } from '@/pk/refs';

describe('refs (PK Scoped Refs)', () => {
    let testContainer: HTMLDivElement;

    beforeEach(() => {
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);
    });

    afterEach(() => {
        testContainer.remove();
        vi.clearAllMocks();
    });

    describe('createRefs()', () => {
        it('should return element with matching p-ref attribute', () => {
            const refEl = document.createElement('button');
            refEl.setAttribute('p-ref', 'myButton');
            testContainer.appendChild(refEl);

            const scopedRefs = createRefs(testContainer);

            expect(scopedRefs.myButton).toBe(refEl);
        });

        it('should return null and warn for non-existent ref', () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const scopedRefs = createRefs(testContainer);

            const result = scopedRefs.nonexistent;

            expect(result).toBeNull();
            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('ref "nonexistent" not found')
            );

            warnSpy.mockRestore();
        });

        it('should scope queries to provided element', () => {
            const outsideEl = document.createElement('div');
            outsideEl.setAttribute('p-ref', 'outsideRef');
            document.body.appendChild(outsideEl);

            const insideEl = document.createElement('div');
            insideEl.setAttribute('p-ref', 'insideRef');
            testContainer.appendChild(insideEl);

            const scopedRefs = createRefs(testContainer);

            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            expect(scopedRefs.insideRef).toBe(insideEl);
            expect(scopedRefs.outsideRef).toBeNull();

            outsideEl.remove();
            warnSpy.mockRestore();
        });

        it('should lazily query elements', () => {
            const scopedRefs = createRefs(testContainer);

            const lateEl = document.createElement('input');
            lateEl.setAttribute('p-ref', 'lateAddition');
            testContainer.appendChild(lateEl);

            expect(scopedRefs.lateAddition).toBe(lateEl);
        });

        it('should handle multiple refs in same scope', () => {
            const input = document.createElement('input');
            input.setAttribute('p-ref', 'emailInput');
            testContainer.appendChild(input);

            const button = document.createElement('button');
            button.setAttribute('p-ref', 'submitBtn');
            testContainer.appendChild(button);

            const form = document.createElement('form');
            form.setAttribute('p-ref', 'mainForm');
            testContainer.appendChild(form);

            const scopedRefs = createRefs(testContainer);

            expect(scopedRefs.emailInput).toBe(input);
            expect(scopedRefs.submitBtn).toBe(button);
            expect(scopedRefs.mainForm).toBe(form);
        });

        it('should handle nested refs', () => {
            const outer = document.createElement('div');
            outer.setAttribute('p-ref', 'outer');
            testContainer.appendChild(outer);

            const inner = document.createElement('span');
            inner.setAttribute('p-ref', 'inner');
            outer.appendChild(inner);

            const scopedRefs = createRefs(testContainer);

            expect(scopedRefs.outer).toBe(outer);
            expect(scopedRefs.inner).toBe(inner);
        });

        it('should return first matching element if duplicates exist', () => {
            const first = document.createElement('div');
            first.setAttribute('p-ref', 'duplicate');
            first.id = 'first';
            testContainer.appendChild(first);

            const second = document.createElement('div');
            second.setAttribute('p-ref', 'duplicate');
            second.id = 'second';
            testContainer.appendChild(second);

            const scopedRefs = createRefs(testContainer);

            expect(scopedRefs.duplicate).toBe(first);
        });

        it('should handle special proxy traps correctly', () => {
            const scopedRefs = createRefs(testContainer);

            expect((scopedRefs as Record<string, unknown>).then).toBeUndefined();

            const symKey = Symbol('test');
            expect((scopedRefs as Record<symbol, unknown>)[symKey]).toBeUndefined();
        });
    });

    describe('refs (global)', () => {
        it('should be scoped to document.body', () => {
            const bodyEl = document.createElement('div');
            bodyEl.setAttribute('p-ref', 'bodyLevelRef');
            document.body.appendChild(bodyEl);

            expect(refs.bodyLevelRef).toBe(bodyEl);

            bodyEl.remove();
        });

        it('should find refs anywhere in body', () => {
            const deepEl = document.createElement('span');
            deepEl.setAttribute('p-ref', 'deepRef');
            testContainer.appendChild(deepEl);

            expect(refs.deepRef).toBe(deepEl);
        });
    });

    describe('integration', () => {
        it('should work with dynamically created and destroyed elements', () => {
            const scopedRefs = createRefs(testContainer);
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            expect(scopedRefs.dynamic).toBeNull();
            warnSpy.mockClear();

            const dynamicEl = document.createElement('div');
            dynamicEl.setAttribute('p-ref', 'dynamic');
            testContainer.appendChild(dynamicEl);

            expect(scopedRefs.dynamic).toBe(dynamicEl);

            dynamicEl.remove();

            expect(scopedRefs.dynamic).toBeNull();

            warnSpy.mockRestore();
        });

        it('should support common use case of form refs', () => {
            testContainer.innerHTML = `
                <form>
                    <input type="text" p-ref="username" />
                    <input type="email" p-ref="email" />
                    <input type="password" p-ref="password" />
                    <button type="submit" p-ref="submitBtn">Submit</button>
                </form>
            `;

            const formRefs = createRefs(testContainer);

            expect(formRefs.username).toBeInstanceOf(HTMLInputElement);
            expect(formRefs.email).toBeInstanceOf(HTMLInputElement);
            expect(formRefs.password).toBeInstanceOf(HTMLInputElement);
            expect(formRefs.submitBtn).toBeInstanceOf(HTMLButtonElement);

            (formRefs.username as HTMLInputElement).value = 'testuser';
            expect((formRefs.username as HTMLInputElement).value).toBe('testuser');
        });
    });
});
