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
import { getBehaviour, registerBehaviour } from './registry';
import type { PPBehaviour } from './registry';
import './formAssociated';

interface MockPPElement {
    tagName: string;
    internals: MockElementInternals | undefined;
    refs: Record<string, HTMLElement | undefined>;
    shadowRoot: ShadowRoot | null;
    state: Record<string, unknown> | undefined;
    $$ctx: { $$initialState?: unknown; state?: Record<string, unknown> } | undefined;
    onUpdated: (cb: (changedProps: Set<string>) => void) => void;
    onConnected: (cb: () => void) => void;
    onAfterRender: (cb: () => void) => void;
    getAttribute: (name: string) => string | null;
    localName: string;
    formAssociatedCallback?: (form: HTMLFormElement | null) => void;
    formDisabledCallback?: (disabled: boolean) => void;
    formResetCallback?: () => void;
    formStateRestoreCallback?: (state: string | File | FormData | null, mode: 'restore' | 'autocomplete') => void;
    checkValidity?: () => boolean;
    reportValidity?: () => boolean;
    _updateFormState?: () => void;
    dispatchEvent: (event: Event) => boolean;
    form?: HTMLFormElement | null;
    validity?: ValidityState;
    validationMessage?: string;
    willValidate?: boolean;
    name?: string | null;
    type?: string;
    labels?: NodeList;
}

interface MockElementInternals {
    form: HTMLFormElement | null;
    validity: ValidityState;
    validationMessage: string;
    willValidate: boolean;
    labels: NodeList;
    setFormValue: (value: string | null) => void;
    setValidity: (flags: Partial<ValidityState>, message?: string, anchor?: HTMLElement) => void;
    checkValidity: () => boolean;
    reportValidity: () => boolean;
}

describe('formAssociated behaviour', () => {
    let mockComponent: MockPPElement;
    let mockInternals: MockElementInternals;
    let onUpdatedCallbacks: ((changedProps: Set<string>) => void)[];
    let onConnectedCallbacks: (() => void)[];
    let onAfterRenderCallbacks: (() => void)[];
    let formBehaviour: PPBehaviour;

    beforeEach(() => {
        vi.useFakeTimers();

        onUpdatedCallbacks = [];
        onConnectedCallbacks = [];
        onAfterRenderCallbacks = [];

        const mockValidityState: ValidityState = {
            badInput: false,
            customError: false,
            patternMismatch: false,
            rangeOverflow: false,
            rangeUnderflow: false,
            stepMismatch: false,
            tooLong: false,
            tooShort: false,
            typeMismatch: false,
            valid: true,
            valueMissing: false
        };

        mockInternals = {
            form: null,
            validity: mockValidityState,
            validationMessage: '',
            willValidate: true,
            labels: document.querySelectorAll('label'),
            setFormValue: vi.fn(),
            setValidity: vi.fn(),
            checkValidity: vi.fn().mockReturnValue(true),
            reportValidity: vi.fn().mockReturnValue(true)
        };

        mockComponent = {
            tagName: 'PP-TEST-INPUT',
            internals: mockInternals,
            refs: {},
            shadowRoot: null,
            state: { value: 'test-value' },
            $$ctx: { state: { value: 'initial-value' } },
            onUpdated: (cb) => onUpdatedCallbacks.push(cb),
            onConnected: (cb) => onConnectedCallbacks.push(cb),
            onAfterRender: (cb) => onAfterRenderCallbacks.push(cb),
            getAttribute: vi.fn().mockReturnValue('test-name'),
            localName: 'pp-test-input',
            dispatchEvent: vi.fn().mockReturnValue(true)
        };

        formBehaviour = getBehaviour('form')!;
        expect(formBehaviour).toBeDefined();
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.restoreAllMocks();
    });

    describe('registration', () => {
        it('should be registered as "form" behaviour', () => {
            expect(getBehaviour('form')).toBeDefined();
        });
    });

    describe('initialisation', () => {
        it('should log error if internals is not attached', () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            mockComponent.internals = undefined;

            formBehaviour(mockComponent as unknown as never);

            expect(consoleSpy).toHaveBeenCalledWith(
                expect.stringContaining('Form behaviour enabled on PP-TEST-INPUT, but \'internals\' was not attached')
            );
            consoleSpy.mockRestore();
        });

        it('should register lifecycle callbacks', () => {
            formBehaviour(mockComponent as unknown as never);

            expect(onUpdatedCallbacks).toHaveLength(1);
            expect(onConnectedCallbacks).toHaveLength(1);
            expect(onAfterRenderCallbacks).toHaveLength(1);
        });

        it('should add form lifecycle methods to component', () => {
            formBehaviour(mockComponent as unknown as never);

            expect(mockComponent.formAssociatedCallback).toBeDefined();
            expect(mockComponent.formDisabledCallback).toBeDefined();
            expect(mockComponent.formResetCallback).toBeDefined();
            expect(mockComponent.formStateRestoreCallback).toBeDefined();
        });

        it('should add validation methods to component', () => {
            formBehaviour(mockComponent as unknown as never);

            expect(mockComponent.checkValidity).toBeDefined();
            expect(mockComponent.reportValidity).toBeDefined();
            expect(mockComponent._updateFormState).toBeDefined();
        });

        it('should define form-related properties', () => {
            formBehaviour(mockComponent as unknown as never);

            expect(mockComponent.form).toBe(mockInternals.form);
            expect(mockComponent.validity).toBe(mockInternals.validity);
            expect(mockComponent.validationMessage).toBe(mockInternals.validationMessage);
            expect(mockComponent.willValidate).toBe(mockInternals.willValidate);
            expect(mockComponent.name).toBe('test-name');
            expect(mockComponent.type).toBe('pp-test-input');
            expect(mockComponent.labels).toBe(mockInternals.labels);
        });
    });

    describe('onUpdated callback', () => {
        it('should call _updateFormState when value changes', () => {
            formBehaviour(mockComponent as unknown as never);
            mockInternals.setFormValue = vi.fn();

            onUpdatedCallbacks[0](new Set(['value']));

            expect(mockInternals.setFormValue).toHaveBeenCalled();
        });

        it('should call _updateFormState when required changes', () => {
            formBehaviour(mockComponent as unknown as never);
            mockInternals.setFormValue = vi.fn();

            onUpdatedCallbacks[0](new Set(['required']));

            expect(mockInternals.setFormValue).toHaveBeenCalled();
        });

        it('should call _updateFormState when pattern changes', () => {
            formBehaviour(mockComponent as unknown as never);
            mockInternals.setFormValue = vi.fn();

            onUpdatedCallbacks[0](new Set(['pattern']));

            expect(mockInternals.setFormValue).toHaveBeenCalled();
        });

        it('should call _updateFormState when min changes', () => {
            formBehaviour(mockComponent as unknown as never);
            mockInternals.setFormValue = vi.fn();

            onUpdatedCallbacks[0](new Set(['min']));

            expect(mockInternals.setFormValue).toHaveBeenCalled();
        });

        it('should call _updateFormState when max changes', () => {
            formBehaviour(mockComponent as unknown as never);
            mockInternals.setFormValue = vi.fn();

            onUpdatedCallbacks[0](new Set(['max']));

            expect(mockInternals.setFormValue).toHaveBeenCalled();
        });

        it('should not call _updateFormState for unrelated prop changes', () => {
            formBehaviour(mockComponent as unknown as never);
            mockInternals.setFormValue = vi.fn();

            onUpdatedCallbacks[0](new Set(['className', 'disabled']));

            expect(mockInternals.setFormValue).not.toHaveBeenCalled();
        });
    });

    describe('onConnected callback', () => {
        it('should call _updateFormState on next animation frame', () => {
            formBehaviour(mockComponent as unknown as never);

            onConnectedCallbacks[0]();

            expect(mockInternals.setFormValue).not.toHaveBeenCalled();

            vi.advanceTimersByTime(16);

            expect(mockInternals.setFormValue).toHaveBeenCalledWith('test-value');
        });
    });

    describe('onAfterRender callback', () => {
        it('should call _updateFormState on next animation frame', () => {
            formBehaviour(mockComponent as unknown as never);

            onAfterRenderCallbacks[0]();

            vi.advanceTimersByTime(16);

            expect(mockInternals.setFormValue).toHaveBeenCalled();
        });
    });

    describe('_updateFormState()', () => {
        it('should set form value from state', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent._updateFormState!();

            expect(mockInternals.setFormValue).toHaveBeenCalledWith('test-value');
        });

        it('should set form value to null when state.value is null', () => {
            mockComponent.state!.value = null;
            formBehaviour(mockComponent as unknown as never);

            mockComponent._updateFormState!();

            expect(mockInternals.setFormValue).toHaveBeenCalledWith(null);
        });

        it('should set validity from native input in refs', () => {
            const mockInput = document.createElement('input');
            mockInput.required = true;
            mockInput.value = '';
            mockComponent.refs.nativeInput = mockInput;
            formBehaviour(mockComponent as unknown as never);

            mockComponent._updateFormState!();

            expect(mockInternals.setValidity).toHaveBeenCalled();
        });

        it('should query shadowRoot for input if refs.nativeInput is not set', () => {
            const mockShadowRoot = document.createElement('div') as unknown as ShadowRoot;
            const mockInput = document.createElement('input');
            mockShadowRoot.appendChild(mockInput);
            (mockShadowRoot as unknown as { querySelector: typeof document.querySelector }).querySelector = (selector: string) => {
                if (selector.includes('input')) return mockInput;
                return null;
            };
            mockComponent.shadowRoot = mockShadowRoot;
            formBehaviour(mockComponent as unknown as never);

            mockComponent._updateFormState!();

            expect(mockInternals.setValidity).toHaveBeenCalled();
        });

        it('should set empty validity when no native element found', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent._updateFormState!();

            expect(mockInternals.setValidity).toHaveBeenCalledWith({});
        });

        it('should dispatch composed input event on component when value changes', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent._updateFormState!();

            expect(mockComponent.dispatchEvent).toHaveBeenCalledTimes(1);
            const event = (mockComponent.dispatchEvent as ReturnType<typeof vi.fn>).mock.calls[0][0] as Event;
            expect(event.type).toBe('input');
            expect(event.bubbles).toBe(true);
            expect(event.composed).toBe(true);
        });

        it('should not dispatch event when value has not changed', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent._updateFormState!();
            mockComponent._updateFormState!();

            expect(mockComponent.dispatchEvent).toHaveBeenCalledTimes(1);
        });

        it('should dispatch event when value changes back', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent.state!.value = 'a';
            mockComponent._updateFormState!();

            mockComponent.state!.value = 'b';
            mockComponent._updateFormState!();

            expect(mockComponent.dispatchEvent).toHaveBeenCalledTimes(2);
        });
    });

    describe('formDisabledCallback()', () => {
        it('should set disabled state when called with true', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent.formDisabledCallback!(true);

            expect(mockComponent.state!.disabled).toBe(true);
        });

        it('should clear disabled state when called with false', () => {
            mockComponent.state!.disabled = true;
            formBehaviour(mockComponent as unknown as never);

            mockComponent.formDisabledCallback!(false);

            expect(mockComponent.state!.disabled).toBe(false);
        });
    });

    describe('formResetCallback()', () => {
        it('should reset value to initial state', () => {
            mockComponent.state!.value = 'modified-value';
            mockComponent.$$ctx = { $$initialState: { value: 'initial-value' }, state: { value: 'initial-value' } };
            formBehaviour(mockComponent as unknown as never);

            mockComponent.formResetCallback!();

            expect(mockComponent.state!.value).toBe('initial-value');
        });

        it('should call _updateFormState after reset', () => {
            mockComponent.$$ctx = { $$initialState: { value: 'initial-value' }, state: { value: 'initial-value' } };
            formBehaviour(mockComponent as unknown as never);
            mockInternals.setFormValue = vi.fn();

            mockComponent.formResetCallback!();
            vi.advanceTimersByTime(16);

            expect(mockInternals.setFormValue).toHaveBeenCalled();
        });

        it('should set empty value if initialState value is undefined', () => {
            mockComponent.$$ctx = { $$initialState: {}, state: {} };
            formBehaviour(mockComponent as unknown as never);

            mockComponent.formResetCallback!();

            expect(mockComponent.state!.value).toBe('');
        });

        it('should do nothing if state is undefined', () => {
            mockComponent.state = undefined;
            formBehaviour(mockComponent as unknown as never);

            expect(() => mockComponent.formResetCallback!()).not.toThrow();
        });
    });

    describe('formStateRestoreCallback()', () => {
        it('should restore state value from provided state', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent.formStateRestoreCallback!('restored-value', 'restore');

            expect(mockComponent.state!.value).toBe('restored-value');
        });

        it('should handle autocomplete mode', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent.formStateRestoreCallback!('autocomplete-value', 'autocomplete');

            expect(mockComponent.state!.value).toBe('autocomplete-value');
        });

        it('should call _updateFormState after restore', () => {
            formBehaviour(mockComponent as unknown as never);

            mockComponent.formStateRestoreCallback!('test', 'restore');
            vi.advanceTimersByTime(16);

            expect(mockInternals.setFormValue).toHaveBeenCalled();
        });

        it('should do nothing if state is undefined', () => {
            mockComponent.state = undefined;
            formBehaviour(mockComponent as unknown as never);

            expect(() => mockComponent.formStateRestoreCallback!('value', 'restore')).not.toThrow();
        });
    });

    describe('formAssociatedCallback()', () => {
        it('should be a no-op (just for completeness)', () => {
            formBehaviour(mockComponent as unknown as never);

            expect(() => mockComponent.formAssociatedCallback!(null)).not.toThrow();
            expect(() => mockComponent.formAssociatedCallback!(document.createElement('form'))).not.toThrow();
        });
    });

    describe('checkValidity() and reportValidity()', () => {
        it('should delegate checkValidity to internals', () => {
            formBehaviour(mockComponent as unknown as never);

            const result = mockComponent.checkValidity!();

            expect(mockInternals.checkValidity).toHaveBeenCalled();
            expect(result).toBe(true);
        });

        it('should delegate reportValidity to internals', () => {
            formBehaviour(mockComponent as unknown as never);

            const result = mockComponent.reportValidity!();

            expect(mockInternals.reportValidity).toHaveBeenCalled();
            expect(result).toBe(true);
        });
    });
});

describe('behaviour registry', () => {
    it('should warn when overwriting a behaviour', () => {
        const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

        const testBehaviour: PPBehaviour = () => {};
        registerBehaviour('test-overwrite', testBehaviour);
        registerBehaviour('test-overwrite', testBehaviour);

        expect(consoleSpy).toHaveBeenCalledWith('PPBehaviour: Overwriting already registered behaviour "test-overwrite".');

        consoleSpy.mockRestore();
    });

    it('should return undefined for unregistered behaviour', () => {
        expect(getBehaviour('nonexistent-behaviour')).toBeUndefined();
    });
});
