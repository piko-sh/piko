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
import { createErrorDisplay, type ErrorDisplay } from './ErrorDisplay';

describe('ErrorDisplay', () => {
    let errorDisplay: ErrorDisplay;

    beforeEach(() => {
        vi.useFakeTimers();
    });

    afterEach(() => {
        const errorEl = document.getElementById('ppf-error-message');
        if (errorEl) {
            errorEl.remove();
        }
        vi.useRealTimers();
    });

    describe('createErrorDisplay()', () => {
        it('should create an ErrorDisplay with default options', () => {
            errorDisplay = createErrorDisplay();
            expect(errorDisplay).toBeDefined();
            expect(typeof errorDisplay.show).toBe('function');
            expect(typeof errorDisplay.clear).toBe('function');
        });

        it('should accept custom displayMs option', () => {
            errorDisplay = createErrorDisplay({ displayMs: 1000 });
            errorDisplay.show('Quick error');

            const errorEl = document.getElementById('ppf-error-message');
            expect(errorEl).not.toBeNull();

            vi.advanceTimersByTime(999);
            expect(document.getElementById('ppf-error-message')).not.toBeNull();

            vi.advanceTimersByTime(2);
            expect(document.getElementById('ppf-error-message')).toBeNull();
        });

        it('should accept custom container option', () => {
            const container = document.createElement('div');
            container.id = 'custom-container';
            document.body.appendChild(container);

            errorDisplay = createErrorDisplay({ container });
            errorDisplay.show('Container error');

            expect(container.querySelector('#ppf-error-message')).not.toBeNull();

            container.remove();
        });
    });

    describe('show()', () => {
        beforeEach(() => {
            errorDisplay = createErrorDisplay();
        });

        it('should create and display an error element', () => {
            errorDisplay.show('Test error message');

            const errorEl = document.getElementById('ppf-error-message');
            expect(errorEl).not.toBeNull();
            expect(errorEl!.textContent).toBe('Test error message');
        });

        it('should style the error element correctly', () => {
            errorDisplay.show('Styled error');

            const errorEl = document.getElementById('ppf-error-message');
            expect(errorEl).not.toBeNull();
            expect(errorEl!.style.position).toBe('fixed');
            expect(errorEl!.style.background).toBe('rgb(244, 67, 54)');
            expect(errorEl!.style.color).toBe('rgb(255, 255, 255)');
            expect(errorEl!.style.zIndex).toBe('99999');
        });

        it('should auto-remove error after default timeout (5000ms)', () => {
            errorDisplay.show('Auto-remove error');

            expect(document.getElementById('ppf-error-message')).not.toBeNull();

            vi.advanceTimersByTime(4999);
            expect(document.getElementById('ppf-error-message')).not.toBeNull();

            vi.advanceTimersByTime(2);
            expect(document.getElementById('ppf-error-message')).toBeNull();
        });

        it('should reuse existing error element if present', () => {
            errorDisplay.show('First error');
            const firstEl = document.getElementById('ppf-error-message');

            errorDisplay.show('Second error');
            const secondEl = document.getElementById('ppf-error-message');

            expect(secondEl).toBe(firstEl);
            expect(secondEl!.textContent).toBe('Second error');
        });

        it('should clear previous timeout when showing new error with fresh element', () => {
            errorDisplay.show('Error 1');
            vi.advanceTimersByTime(3000);

            errorDisplay.clear();
            errorDisplay.show('Error 2');
            vi.advanceTimersByTime(4999);

            expect(document.getElementById('ppf-error-message')).not.toBeNull();

            vi.advanceTimersByTime(2);
            expect(document.getElementById('ppf-error-message')).toBeNull();
        });
    });

    describe('clear()', () => {
        beforeEach(() => {
            errorDisplay = createErrorDisplay();
        });

        it('should remove the error element immediately', () => {
            errorDisplay.show('Error to clear');
            expect(document.getElementById('ppf-error-message')).not.toBeNull();

            errorDisplay.clear();
            expect(document.getElementById('ppf-error-message')).toBeNull();
        });

        it('should clear the timeout', () => {
            errorDisplay.show('Error with timeout');
            errorDisplay.clear();

            vi.advanceTimersByTime(10000);
            expect(document.getElementById('ppf-error-message')).toBeNull();
        });

        it('should do nothing if no error is displayed', () => {
            expect(() => errorDisplay.clear()).not.toThrow();
        });

        it('should handle clearing when element was already removed externally', () => {
            errorDisplay.show('External removal');

            const el = document.getElementById('ppf-error-message');
            el?.remove();

            expect(() => errorDisplay.clear()).not.toThrow();
        });

        it('should allow showing new error after clear', () => {
            errorDisplay.show('First');
            errorDisplay.clear();
            errorDisplay.show('After clear');

            const el = document.getElementById('ppf-error-message');
            expect(el).not.toBeNull();
            expect(el!.textContent).toBe('After clear');
        });
    });

    describe('multiple instances', () => {
        it('should share the same DOM element by ID', () => {
            const display1 = createErrorDisplay();
            const display2 = createErrorDisplay();

            display1.show('From display 1');
            display2.show('From display 2');

            const elements = document.querySelectorAll('#ppf-error-message');
            expect(elements).toHaveLength(1);
            expect(elements[0].textContent).toBe('From display 2');
        });
    });
});
