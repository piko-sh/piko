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

import {describe, expect, it, vi} from 'vitest';
import {PPElement, type PropTypeDefinition} from '@/element';
import {makeReactive} from '@/reactivity';
import {dom, type VirtualNode} from '@/vdom';

class ResettableField extends PPElement {
    static override formAssociated = true;
    static override propTypes: Record<string, PropTypeDefinition> = {value: {type: 'string', reflectToAttribute: true}};
}
customElements.define('test-workflow-reset-field', ResettableField);

describe('form-associated reset', () => {
    it('registers the native reaction on the prototype and restores the last external default', async () => {
        const field = new ResettableField();
        const baseline = {value: 'declared'};
        field.setAttribute('value', 'from markup');
        field.init({state: makeReactive(baseline, {scheduleRender: () => field.scheduleRender()}), $$initialState: baseline});
        const sync = vi.fn();
        field._updateFormState = sync;
        const nativeReset = ResettableField.prototype.formResetCallback;
        expect(nativeReset).toBeTypeOf('function');
        field.state!.value = 'user edit';
        nativeReset.call(field);
        expect(field.state!.value).toBe('from markup');
        expect(sync).not.toHaveBeenCalled();
        await Promise.resolve();
        expect(sync).toHaveBeenCalledOnce();

        field.setAttribute('value', 'new default');
        field.state!.value = 'another edit';
        nativeReset.call(field);
        expect(field.state!.value).toBe('new default');
    });

    it('handles an uninitialised field and an empty default', () => {
        const field = new ResettableField();
        expect(() => field.formResetCallback()).not.toThrow();
        field.init({state: {}});
        field.state!.value = 'entered';
        field.formResetCallback();
        expect(field.state!.value).toBe('');
    });
});

describe('form-associated disabled and restore reactions', () => {
    const initialise = (): ResettableField => {
        const field = new ResettableField();
        const baseline = {value: 'declared', disabled: false};
        field.init({state: makeReactive(baseline, {scheduleRender: () => field.scheduleRender()}), $$initialState: baseline});
        return field;
    };

    it('registers the disabled and restore reactions on the prototype', () => {
        expect(ResettableField.prototype.formDisabledCallback).toBeTypeOf('function');
        expect(ResettableField.prototype.formStateRestoreCallback).toBeTypeOf('function');
        expect(Object.hasOwn(new ResettableField(), 'formDisabledCallback')).toBe(false);
    });

    it('keeps the owning fieldset disabled state out of the own prop', () => {
        const field = initialise();

        ResettableField.prototype.formDisabledCallback.call(field, true);
        expect(field.state!.disabled).toBe(false);
        expect(field.formDisabled).toBe(true);
        expect(field.effectiveDisabled).toBe(true);

        ResettableField.prototype.formDisabledCallback.call(field, false);
        expect(field.formDisabled).toBe(false);
        expect(field.effectiveDisabled).toBe(false);

        field.state!.disabled = true;
        expect(field.effectiveDisabled).toBe(true);
    });

    it('restores a saved value and synchronises the form state on the next microtask', async () => {
        const field = initialise();
        const sync = vi.fn();
        field._updateFormState = sync;

        ResettableField.prototype.formStateRestoreCallback.call(field, 'restored', 'restore');
        expect(field.state!.value).toBe('restored');
        expect(sync).not.toHaveBeenCalled();
        await Promise.resolve();
        expect(sync).toHaveBeenCalledOnce();

        ResettableField.prototype.formStateRestoreCallback.call(field, 'suggested', 'autocomplete');
        expect(field.state!.value).toBe('suggested');
    });

    it('ignores reactions before initialisation', () => {
        const field = new ResettableField();

        expect(() => field.formDisabledCallback(true)).not.toThrow();
        expect(() => field.formStateRestoreCallback('value', 'restore')).not.toThrow();
    });
});

class DisableableField extends PPElement {
    static override formAssociated = true;
    static override propTypes: Record<string, PropTypeDefinition> = {disabled: {type: 'boolean', reflectToAttribute: true}};

    override renderVDOM(): VirtualNode {
        return dom.el('input', 'disableable-root', {'?disabled': this.effectiveDisabled});
    }
}
customElements.define('test-workflow-disableable-field', DisableableField);

describe('form-associated field inside a fieldset', () => {
    it('never reflects an inherited disabled state onto the host', async () => {
        const fieldset = document.createElement('fieldset');
        fieldset.disabled = true;
        const field = new DisableableField();
        const baseline = {disabled: false};
        field.init({state: makeReactive(baseline, {scheduleRender: () => field.scheduleRender()}), $$initialState: baseline});
        fieldset.append(field);
        document.body.append(fieldset);
        const input = (): HTMLInputElement | null | undefined => field.shadowRoot?.querySelector('input');
        const render = vi.spyOn(field, 'scheduleRender');

        field.formDisabledCallback(fieldset.disabled);
        field.render();
        await Promise.resolve();
        expect(field.effectiveDisabled).toBe(true);
        expect(field.hasAttribute('disabled')).toBe(false);
        expect(input()?.disabled).toBe(true);
        expect(render).toHaveBeenCalledOnce();

        fieldset.disabled = false;
        field.formDisabledCallback(fieldset.disabled);
        field.render();
        await Promise.resolve();
        expect(field.effectiveDisabled).toBe(false);
        expect(field.hasAttribute('disabled')).toBe(false);
        expect(input()?.disabled).toBe(false);

        fieldset.disabled = true;
        field.formDisabledCallback(fieldset.disabled);
        field.formDisabledCallback(fieldset.disabled);
        expect(render).toHaveBeenCalledTimes(3);

        expect(field.hasAttribute('disabled')).toBe(false);
        fieldset.remove();
    });
});
