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

import {describe, it, expect, vi, beforeEach, afterEach} from 'vitest';
import {createFormStateManager, type FormStateManager} from './FormStateManager';
import {createHookManager, HookEvent, type HookManager} from './HookManager';

describe('FormStateManager', () => {
    let hookManager: HookManager;
    let formStateManager: FormStateManager;
    let testContainer: HTMLDivElement;

    beforeEach(() => {
        hookManager = createHookManager();
        testContainer = document.createElement('div');
        document.body.appendChild(testContainer);
    });

    afterEach(() => {
        formStateManager?.destroy();
        testContainer.remove();
    });

    function createForm(options: {
        id?: string;
        action?: string;
        noTrack?: boolean;
        fields?: Array<{name: string; value: string; type?: string}>;
    } = {}): HTMLFormElement {
        const form = document.createElement('form');
        if (options.id) {
            form.id = options.id;
        }
        if (options.action) {
            form.action = options.action;
        }
        if (options.noTrack) {
            form.setAttribute('pk-no-track', '');
        }

        (options.fields ?? []).forEach(field => {
            const input = document.createElement('input');
            input.name = field.name;
            input.value = field.value;
            input.type = field.type ?? 'text';
            form.appendChild(input);
        });

        testContainer.appendChild(form);
        return form;
    }

    describe('createFormStateManager', () => {
        it('should create a form state manager', () => {
            formStateManager = createFormStateManager({hookManager});

            expect(formStateManager).toBeDefined();
            expect(typeof formStateManager.trackForm).toBe('function');
            expect(typeof formStateManager.untrackForm).toBe('function');
            expect(typeof formStateManager.markFormClean).toBe('function');
            expect(typeof formStateManager.hasDirtyForms).toBe('function');
            expect(typeof formStateManager.getDirtyFormIds).toBe('function');
            expect(typeof formStateManager.confirmNavigation).toBe('function');
            expect(typeof formStateManager.scanAndTrackForms).toBe('function');
            expect(typeof formStateManager.destroy).toBe('function');
        });
    });

    describe('trackForm', () => {
        it('should track a form', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            formStateManager.trackForm(form);

            expect(form.getAttribute('data-pk-tracked')).toBe('true');
        });

        it('should not track a form with pk-no-track attribute', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form', noTrack: true});

            formStateManager.trackForm(form);

            expect(form.getAttribute('data-pk-tracked')).toBeNull();
        });

        it('should not track the same form twice', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            formStateManager.trackForm(form);
            formStateManager.trackForm(form);

            expect(form.getAttribute('data-pk-tracked')).toBe('true');
        });

        it('should capture initial form state', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });
    });

    describe('untrackForm', () => {
        it('should untrack a form', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            formStateManager.trackForm(form);
            expect(form.getAttribute('data-pk-tracked')).toBe('true');

            formStateManager.untrackForm(form);
            expect(form.getAttribute('data-pk-tracked')).toBeNull();
        });

        it('should handle untracking an untracked form', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            expect(() => formStateManager.untrackForm(form)).not.toThrow();
        });
    });

    describe('hasDirtyForms', () => {
        it('should return false when no forms are tracked', () => {
            formStateManager = createFormStateManager({hookManager});

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should return false when forms are clean', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should return true when form value has changed', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';

            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should detect changes via input event', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should detect checkbox changes', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.name = 'agree';
            checkbox.checked = false;
            form.appendChild(checkbox);

            formStateManager.trackForm(form);

            checkbox.checked = true;
            checkbox.dispatchEvent(new Event('change', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should detect radio button changes', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            const radio1 = document.createElement('input');
            radio1.type = 'radio';
            radio1.name = 'choice';
            radio1.value = 'a';
            radio1.checked = true;
            form.appendChild(radio1);

            const radio2 = document.createElement('input');
            radio2.type = 'radio';
            radio2.name = 'choice';
            radio2.value = 'b';
            radio2.checked = false;
            form.appendChild(radio2);

            formStateManager.trackForm(form);

            radio2.checked = true;
            radio1.checked = false;
            radio2.dispatchEvent(new Event('change', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(true);
        });
    });

    describe('getDirtyFormIds', () => {
        it('should return empty array when no dirty forms', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            formStateManager.trackForm(form);

            expect(formStateManager.getDirtyFormIds()).toEqual([]);
        });

        it('should return form ID when form is dirty', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';

            expect(formStateManager.getDirtyFormIds()).toEqual(['test-form']);
        });

        it('should generate ID for forms without ID', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                action: '/submit',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';

            const dirtyIds = formStateManager.getDirtyFormIds();
            expect(dirtyIds.length).toBe(1);
            expect(dirtyIds[0]).toMatch(/^form-\d+-/);
        });

        it('should return multiple dirty form IDs', () => {
            formStateManager = createFormStateManager({hookManager});

            const form1 = createForm({
                id: 'form-1',
                fields: [{name: 'field1', value: 'initial1'}]
            });
            const form2 = createForm({
                id: 'form-2',
                fields: [{name: 'field2', value: 'initial2'}]
            });

            formStateManager.trackForm(form1);
            formStateManager.trackForm(form2);

            (form1.querySelector('input') as HTMLInputElement).value = 'changed1';
            (form2.querySelector('input') as HTMLInputElement).value = 'changed2';

            const dirtyIds = formStateManager.getDirtyFormIds();
            expect(dirtyIds).toContain('form-1');
            expect(dirtyIds).toContain('form-2');
        });
    });

    describe('markFormClean', () => {
        it('should mark a dirty form as clean', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';

            expect(formStateManager.hasDirtyForms()).toBe(true);

            formStateManager.markFormClean(form);

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should update initial snapshot to current state', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            formStateManager.markFormClean(form);

            input.value = 'initial';
            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should emit FORM_CLEAN hook when marking dirty form clean', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            const callback = vi.fn();
            hookManager.api.on(HookEvent.FORM_CLEAN, callback);

            formStateManager.markFormClean(form);

            expect(callback).toHaveBeenCalledWith(expect.objectContaining({
                formId: 'test-form',
                timestamp: expect.any(Number)
            }));
        });

        it('should not emit FORM_CLEAN if form was already clean', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            formStateManager.trackForm(form);

            const callback = vi.fn();
            hookManager.api.on(HookEvent.FORM_CLEAN, callback);

            formStateManager.markFormClean(form);

            expect(callback).not.toHaveBeenCalled();
        });

        it('should handle marking untracked form as clean', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form'});

            expect(() => formStateManager.markFormClean(form)).not.toThrow();
        });
    });

    describe('confirmNavigation', () => {
        it('should return true when no dirty forms', () => {
            formStateManager = createFormStateManager({hookManager});

            expect(formStateManager.confirmNavigation()).toBe(true);
        });

        it('should call confirmFn when forms are dirty', () => {
            const confirmFn = vi.fn().mockReturnValue(true);
            formStateManager = createFormStateManager({hookManager, confirmFn});

            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);
            (form.querySelector('input') as HTMLInputElement).value = 'changed';

            formStateManager.confirmNavigation();

            expect(confirmFn).toHaveBeenCalledWith('You have unsaved changes. Leave anyway?');
        });

        it('should return true if user confirms', () => {
            const confirmFn = vi.fn().mockReturnValue(true);
            formStateManager = createFormStateManager({hookManager, confirmFn});

            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);
            (form.querySelector('input') as HTMLInputElement).value = 'changed';

            expect(formStateManager.confirmNavigation()).toBe(true);
        });

        it('should return false if user cancels', () => {
            const confirmFn = vi.fn().mockReturnValue(false);
            formStateManager = createFormStateManager({hookManager, confirmFn});

            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);
            (form.querySelector('input') as HTMLInputElement).value = 'changed';

            expect(formStateManager.confirmNavigation()).toBe(false);
        });
    });

    describe('scanAndTrackForms', () => {
        it('should scan and track all forms in document body', () => {
            formStateManager = createFormStateManager({hookManager});

            const form1 = createForm({id: 'form-1'});
            const form2 = createForm({id: 'form-2'});

            formStateManager.scanAndTrackForms();

            expect(form1.getAttribute('data-pk-tracked')).toBe('true');
            expect(form2.getAttribute('data-pk-tracked')).toBe('true');
        });

        it('should scan and track forms within a specific root', () => {
            formStateManager = createFormStateManager({hookManager});

            const container = document.createElement('div');
            const form = document.createElement('form');
            form.id = 'nested-form';
            container.appendChild(form);
            testContainer.appendChild(container);

            const outsideForm = createForm({id: 'outside-form'});

            formStateManager.scanAndTrackForms(container);

            expect(form.getAttribute('data-pk-tracked')).toBe('true');
            expect(outsideForm.getAttribute('data-pk-tracked')).toBeNull();
        });

        it('should skip forms with pk-no-track', () => {
            formStateManager = createFormStateManager({hookManager});

            const trackedForm = createForm({id: 'tracked-form'});
            const noTrackForm = createForm({id: 'no-track-form', noTrack: true});

            formStateManager.scanAndTrackForms();

            expect(trackedForm.getAttribute('data-pk-tracked')).toBe('true');
            expect(noTrackForm.getAttribute('data-pk-tracked')).toBeNull();
        });

        it('should not double-track already tracked forms', () => {
            formStateManager = createFormStateManager({hookManager});

            const form = createForm({id: 'test-form'});
            formStateManager.trackForm(form);

            formStateManager.scanAndTrackForms();

            expect(form.getAttribute('data-pk-tracked')).toBe('true');
        });

        it('should clean up forms that are no longer in DOM', () => {
            formStateManager = createFormStateManager({hookManager});

            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);
            (form.querySelector('input') as HTMLInputElement).value = 'changed';

            expect(formStateManager.hasDirtyForms()).toBe(true);

            form.remove();
            formStateManager.scanAndTrackForms();

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });
    });

    describe('FORM_DIRTY hook', () => {
        it('should emit FORM_DIRTY when form becomes dirty', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const callback = vi.fn();
            hookManager.api.on(HookEvent.FORM_DIRTY, callback);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(callback).toHaveBeenCalledWith(expect.objectContaining({
                formId: 'test-form',
                timestamp: expect.any(Number)
            }));
        });

        it('should not emit FORM_DIRTY multiple times for same dirty state', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const callback = vi.fn();
            hookManager.api.on(HookEvent.FORM_DIRTY, callback);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));
            input.value = 'changed again';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(callback).toHaveBeenCalledTimes(1);
        });

        it('should emit FORM_CLEAN when form returns to initial state', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const dirtyCallback = vi.fn();
            const cleanCallback = vi.fn();
            hookManager.api.on(HookEvent.FORM_DIRTY, dirtyCallback);
            hookManager.api.on(HookEvent.FORM_CLEAN, cleanCallback);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;

            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            input.value = 'initial';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(dirtyCallback).toHaveBeenCalledTimes(1);
            expect(cleanCallback).toHaveBeenCalledTimes(1);
        });
    });

    describe('form submission', () => {
        it('should mark form clean on submit', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            expect(formStateManager.hasDirtyForms()).toBe(true);

            form.dispatchEvent(new Event('submit', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should not trigger dirty warning after form submission', () => {
            const confirmFn = vi.fn().mockReturnValue(false);
            formStateManager = createFormStateManager({hookManager, confirmFn});

            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);
            (form.querySelector('input') as HTMLInputElement).value = 'changed';

            form.dispatchEvent(new Event('submit', {bubbles: true}));

            expect(formStateManager.confirmNavigation()).toBe(true);
            expect(confirmFn).not.toHaveBeenCalled();
        });

        it('should emit FORM_CLEAN hook on submit when form was dirty', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            const callback = vi.fn();
            hookManager.api.on(HookEvent.FORM_CLEAN, callback);

            form.dispatchEvent(new Event('submit', {bubbles: true}));

            expect(callback).toHaveBeenCalledWith(expect.objectContaining({
                formId: 'test-form',
                timestamp: expect.any(Number)
            }));
        });

        it('should ignore submit events on untracked forms', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({id: 'test-form', noTrack: true});

            expect(() => form.dispatchEvent(new Event('submit', {bubbles: true}))).not.toThrow();
        });
    });

    describe('destroy', () => {
        it('should remove event listeners', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);
            formStateManager.destroy();

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should clear tracked forms', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);
            (form.querySelector('input') as HTMLInputElement).value = 'changed';

            expect(formStateManager.hasDirtyForms()).toBe(true);

            formStateManager.destroy();

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });
    });

    describe('input events on untracked elements', () => {
        it('should ignore input events on elements outside forms', () => {
            formStateManager = createFormStateManager({hookManager});

            const standaloneInput = document.createElement('input');
            standaloneInput.name = 'standalone';
            standaloneInput.value = 'initial';
            testContainer.appendChild(standaloneInput);

            standaloneInput.value = 'changed';
            standaloneInput.dispatchEvent(new Event('input', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should ignore input events on untracked forms', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}],
                noTrack: true
            });

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });
    });

    describe('untrackAll', () => {
        it('should clear all tracked forms', () => {
            formStateManager = createFormStateManager({hookManager});

            const form1 = createForm({
                id: 'form-1',
                fields: [{name: 'field1', value: 'initial1'}]
            });
            const form2 = createForm({
                id: 'form-2',
                fields: [{name: 'field2', value: 'initial2'}]
            });

            formStateManager.trackForm(form1);
            formStateManager.trackForm(form2);

            (form1.querySelector('input') as HTMLInputElement).value = 'changed1';
            (form2.querySelector('input') as HTMLInputElement).value = 'changed2';

            expect(formStateManager.hasDirtyForms()).toBe(true);

            formStateManager.untrackAll();

            expect(formStateManager.hasDirtyForms()).toBe(false);
            expect(formStateManager.getDirtyFormIds()).toEqual([]);
        });

        it('should remove data-pk-tracked attribute from all forms', () => {
            formStateManager = createFormStateManager({hookManager});

            const form1 = createForm({id: 'form-1'});
            const form2 = createForm({id: 'form-2'});

            formStateManager.trackForm(form1);
            formStateManager.trackForm(form2);

            expect(form1.getAttribute('data-pk-tracked')).toBe('true');
            expect(form2.getAttribute('data-pk-tracked')).toBe('true');

            formStateManager.untrackAll();

            expect(form1.hasAttribute('data-pk-tracked')).toBe(false);
            expect(form2.hasAttribute('data-pk-tracked')).toBe(false);
        });

        it('should allow re-tracking forms after untrackAll', () => {
            formStateManager = createFormStateManager({hookManager});

            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });

            formStateManager.trackForm(form);
            (form.querySelector('input') as HTMLInputElement).value = 'changed';
            expect(formStateManager.hasDirtyForms()).toBe(true);

            formStateManager.untrackAll();
            expect(formStateManager.hasDirtyForms()).toBe(false);

            formStateManager.trackForm(form);
            expect(formStateManager.hasDirtyForms()).toBe(false);
            expect(form.getAttribute('data-pk-tracked')).toBe('true');
        });

        it('should be safe to call when no forms are tracked', () => {
            formStateManager = createFormStateManager({hookManager});

            expect(() => formStateManager.untrackAll()).not.toThrow();
            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should not remove event listeners', () => {
            formStateManager = createFormStateManager({hookManager});

            formStateManager.untrackAll();

            const form = createForm({
                id: 'test-form',
                fields: [{name: 'username', value: 'initial'}]
            });
            formStateManager.trackForm(form);

            const input = form.querySelector('input[name="username"]') as HTMLInputElement;
            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(true);
        });
    });

    describe('dirty state persistence bug', () => {
        describe('bug isolation (without fix)', () => {
            it('should persist dirty state when scanAndTrackForms runs while form is still in DOM', () => {
                formStateManager = createFormStateManager({hookManager});

                const form = createForm({
                    id: 'dirty-form',
                    fields: [{name: 'username', value: 'initial'}]
                });

                formStateManager.trackForm(form);
                (form.querySelector('input') as HTMLInputElement).value = 'changed';
                expect(formStateManager.hasDirtyForms()).toBe(true);

                expect(document.contains(form)).toBe(true);
                formStateManager.scanAndTrackForms();

                expect(formStateManager.hasDirtyForms()).toBe(true);
            });

            it('should repeat the dirty warning on every navigation without untrackAll', () => {
                const confirmFn = vi.fn().mockReturnValue(true);
                formStateManager = createFormStateManager({hookManager, confirmFn});

                const form = createForm({
                    id: 'test-form',
                    fields: [{name: 'field', value: 'initial'}]
                });

                formStateManager.trackForm(form);
                (form.querySelector('input') as HTMLInputElement).value = 'changed';

                expect(formStateManager.confirmNavigation()).toBe(true);
                expect(confirmFn).toHaveBeenCalledTimes(1);

                formStateManager.scanAndTrackForms();

                formStateManager.confirmNavigation();
                expect(confirmFn).toHaveBeenCalledTimes(2);
            });
        });

        describe('bug verification (with fix)', () => {
            it('should clear dirty state when untrackAll is called before scanAndTrackForms', () => {
                formStateManager = createFormStateManager({hookManager});

                const form = createForm({
                    id: 'dirty-form',
                    fields: [{name: 'username', value: 'initial'}]
                });

                formStateManager.trackForm(form);
                (form.querySelector('input') as HTMLInputElement).value = 'changed';
                expect(formStateManager.hasDirtyForms()).toBe(true);

                formStateManager.untrackAll();

                expect(document.contains(form)).toBe(true);
                formStateManager.scanAndTrackForms();

                expect(formStateManager.hasDirtyForms()).toBe(false);
            });

            it('should not repeat the dirty warning after untrackAll', () => {
                const confirmFn = vi.fn().mockReturnValue(true);
                formStateManager = createFormStateManager({hookManager, confirmFn});

                const form = createForm({
                    id: 'test-form',
                    fields: [{name: 'field', value: 'initial'}]
                });

                formStateManager.trackForm(form);
                (form.querySelector('input') as HTMLInputElement).value = 'changed';

                expect(formStateManager.confirmNavigation()).toBe(true);
                expect(confirmFn).toHaveBeenCalledTimes(1);

                formStateManager.untrackAll();
                formStateManager.scanAndTrackForms();

                expect(formStateManager.confirmNavigation()).toBe(true);
                expect(confirmFn).toHaveBeenCalledTimes(1);
            });
        });
    });

    describe('deferred snapshot for custom elements', () => {
        async function waitForDeferredSnapshot(): Promise<void> {
            await new Promise<void>(resolve => requestAnimationFrame(() => {
                requestAnimationFrame(() => resolve());
            }));
            await new Promise(resolve => setTimeout(resolve, 0));
        }

        it('should not consider form dirty while snapshot is pending', async () => {
            formStateManager = createFormStateManager({hookManager});

            const form = document.createElement('form');
            form.id = 'deferred-form';
            const input = document.createElement('input');
            input.name = 'title';
            input.value = 'initial';
            form.appendChild(input);
            form.appendChild(document.createElement('pp-defer-test-a'));
            testContainer.appendChild(form);

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(form.getAttribute('data-pk-tracked')).toBe('true');
            expect(formStateManager.hasDirtyForms()).toBe(false);

            customElements.define('pp-defer-test-a', class extends HTMLElement {});
            await customElements.whenDefined('pp-defer-test-a');
            await waitForDeferredSnapshot();

            input.value = 'changed';
            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should take immediate snapshot for forms with only native elements', () => {
            formStateManager = createFormStateManager({hookManager});

            const form = createForm({
                id: 'native-only-form',
                fields: [{name: 'title', value: 'initial'}]
            });

            formStateManager.trackForm(form);

            expect(formStateManager.hasDirtyForms()).toBe(false);

            (form.querySelector('input') as HTMLInputElement).value = 'changed';
            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should not mark form dirty when custom elements initialise', async () => {
            formStateManager = createFormStateManager({hookManager});

            const form = document.createElement('form');
            form.id = 'init-form';
            const input = document.createElement('input');
            input.name = 'title';
            input.value = 'Test Page';
            form.appendChild(input);
            form.appendChild(document.createElement('pp-defer-test-b'));
            testContainer.appendChild(form);

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(formStateManager.hasDirtyForms()).toBe(false);

            customElements.define('pp-defer-test-b', class extends HTMLElement {});
            await customElements.whenDefined('pp-defer-test-b');
            await waitForDeferredSnapshot();

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should handle form removal during deferred snapshot', async () => {
            formStateManager = createFormStateManager({hookManager});

            const form = document.createElement('form');
            form.id = 'removed-deferred-form';
            form.appendChild(document.createElement('pp-defer-test-c'));
            testContainer.appendChild(form);

            await new Promise(resolve => setTimeout(resolve, 0));

            form.remove();
            await new Promise(resolve => setTimeout(resolve, 0));

            customElements.define('pp-defer-test-c', class extends HTMLElement {});
            await customElements.whenDefined('pp-defer-test-c');
            await waitForDeferredSnapshot();

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should ignore input events while snapshot is pending', async () => {
            formStateManager = createFormStateManager({hookManager});

            const form = document.createElement('form');
            form.id = 'pending-input-form';
            const input = document.createElement('input');
            input.name = 'title';
            input.value = 'initial';
            form.appendChild(input);
            form.appendChild(document.createElement('pp-defer-test-d'));
            testContainer.appendChild(form);

            await new Promise(resolve => setTimeout(resolve, 0));

            const dirtyCallback = vi.fn();
            hookManager.api.on(HookEvent.FORM_DIRTY, dirtyCallback);

            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(dirtyCallback).not.toHaveBeenCalled();

            customElements.define('pp-defer-test-d', class extends HTMLElement {});
            await customElements.whenDefined('pp-defer-test-d');
            await waitForDeferredSnapshot();

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should defer snapshot for already-defined custom elements on second visit', async () => {
            formStateManager = createFormStateManager({hookManager});

            if (!customElements.get('pp-defer-test-predef')) {
                customElements.define('pp-defer-test-predef', class extends HTMLElement {});
            }

            const form = document.createElement('form');
            form.id = 'second-visit-form';
            const input = document.createElement('input');
            input.name = 'title';
            input.value = 'initial';
            form.appendChild(input);
            form.appendChild(document.createElement('pp-defer-test-predef'));
            testContainer.appendChild(form);

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(form.getAttribute('data-pk-tracked')).toBe('true');
            expect(formStateManager.hasDirtyForms()).toBe(false);

            await waitForDeferredSnapshot();

            input.value = 'changed';
            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should allow markFormClean to resolve pending snapshot', async () => {
            formStateManager = createFormStateManager({hookManager});

            const form = document.createElement('form');
            form.id = 'mark-clean-form';
            const input = document.createElement('input');
            input.name = 'title';
            input.value = 'initial';
            form.appendChild(input);
            form.appendChild(document.createElement('pp-defer-test-e'));
            testContainer.appendChild(form);

            await new Promise(resolve => setTimeout(resolve, 0));

            formStateManager.markFormClean(form);

            input.value = 'changed';
            expect(formStateManager.hasDirtyForms()).toBe(true);
        });
    });

    describe('automatic form observation', () => {
        it('should automatically track a form added to the DOM after creation', async () => {
            formStateManager = createFormStateManager({hookManager});

            expect(formStateManager.hasDirtyForms()).toBe(false);

            const form = document.createElement('form');
            form.id = 'late-form';
            const input = document.createElement('input');
            input.name = 'title';
            input.value = 'initial';
            form.appendChild(input);
            testContainer.appendChild(form);

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(form.getAttribute('data-pk-tracked')).toBe('true');

            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(true);
            expect(formStateManager.getDirtyFormIds()).toEqual(['late-form']);
        });

        it('should automatically untrack a form removed from the DOM', async () => {
            formStateManager = createFormStateManager({hookManager});

            const form = createForm({
                id: 'removable-form',
                fields: [{name: 'field', value: 'initial'}]
            });

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(form.getAttribute('data-pk-tracked')).toBe('true');

            (form.querySelector('input') as HTMLInputElement).value = 'changed';
            expect(formStateManager.hasDirtyForms()).toBe(true);

            form.remove();

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should track forms nested inside a container added to the DOM', async () => {
            formStateManager = createFormStateManager({hookManager});

            const container = document.createElement('div');
            const form = document.createElement('form');
            form.id = 'nested-form';
            const input = document.createElement('input');
            input.name = 'content';
            input.value = 'initial';
            form.appendChild(input);
            container.appendChild(form);
            testContainer.appendChild(container);

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(form.getAttribute('data-pk-tracked')).toBe('true');

            input.value = 'changed';
            input.dispatchEvent(new Event('input', {bubbles: true}));

            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should not track forms with pk-no-track attribute added to the DOM', async () => {
            formStateManager = createFormStateManager({hookManager});

            const form = document.createElement('form');
            form.id = 'no-track-form';
            form.setAttribute('pk-no-track', '');
            testContainer.appendChild(form);

            await new Promise(resolve => setTimeout(resolve, 0));

            expect(form.hasAttribute('data-pk-tracked')).toBe(false);
        });

        it('should not re-track forms cleared by untrackAll when observer fires', async () => {
            formStateManager = createFormStateManager({hookManager});

            const form = createForm({
                id: 'cleared-form',
                fields: [{name: 'field', value: 'initial'}]
            });

            await new Promise(resolve => setTimeout(resolve, 0));

            (form.querySelector('input') as HTMLInputElement).value = 'changed';
            expect(formStateManager.hasDirtyForms()).toBe(true);

            formStateManager.untrackAll();
            expect(formStateManager.hasDirtyForms()).toBe(false);

            formStateManager.scanAndTrackForms();
            expect(formStateManager.hasDirtyForms()).toBe(false);
        });
    });

    describe('pk-no-track on child elements', () => {
        it('should exclude fields inside a pk-no-track container from dirty tracking', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = document.createElement('form');
            form.id = 'captcha-form';

            const textInput = document.createElement('input');
            textInput.type = 'text';
            textInput.name = 'username';
            textInput.value = 'alice';
            form.appendChild(textInput);

            const noTrackDiv = document.createElement('div');
            noTrackDiv.setAttribute('pk-no-track', '');
            const hiddenInput = document.createElement('input');
            hiddenInput.type = 'hidden';
            hiddenInput.name = '_captcha_token';
            hiddenInput.value = '';
            noTrackDiv.appendChild(hiddenInput);
            form.appendChild(noTrackDiv);

            testContainer.appendChild(form);

            formStateManager.trackForm(form);
            expect(formStateManager.hasDirtyForms()).toBe(false);

            hiddenInput.value = 'generated-captcha-token';
            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should still track fields outside pk-no-track containers', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = document.createElement('form');
            form.id = 'mixed-form';

            const emailInput = document.createElement('input');
            emailInput.type = 'text';
            emailInput.name = 'email';
            emailInput.value = 'original@test.com';
            form.appendChild(emailInput);

            const noTrackDiv = document.createElement('div');
            noTrackDiv.setAttribute('pk-no-track', '');
            const tokenInput = document.createElement('input');
            tokenInput.type = 'hidden';
            tokenInput.name = '_csrf_token';
            tokenInput.value = 'token123';
            noTrackDiv.appendChild(tokenInput);
            form.appendChild(noTrackDiv);

            testContainer.appendChild(form);

            formStateManager.trackForm(form);
            expect(formStateManager.hasDirtyForms()).toBe(false);

            emailInput.value = 'changed@test.com';
            expect(formStateManager.hasDirtyForms()).toBe(true);
        });

        it('should exclude fields with pk-no-track directly on the input', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = document.createElement('form');
            form.id = 'direct-no-track-form';

            const trackedInput = document.createElement('input');
            trackedInput.type = 'text';
            trackedInput.name = 'name';
            trackedInput.value = 'original';
            form.appendChild(trackedInput);

            const untrackedInput = document.createElement('input');
            untrackedInput.type = 'hidden';
            untrackedInput.name = '_token';
            untrackedInput.value = '';
            untrackedInput.setAttribute('pk-no-track', '');
            form.appendChild(untrackedInput);

            testContainer.appendChild(form);

            formStateManager.trackForm(form);
            expect(formStateManager.hasDirtyForms()).toBe(false);

            untrackedInput.value = 'new-token-value';
            expect(formStateManager.hasDirtyForms()).toBe(false);
        });

        it('should not exclude visually hidden checkboxes without pk-no-track', () => {
            formStateManager = createFormStateManager({hookManager});
            const form = document.createElement('form');
            form.id = 'styled-checkbox-form';

            const checkbox = document.createElement('input');
            checkbox.type = 'checkbox';
            checkbox.name = 'agree';
            checkbox.style.display = 'none';
            form.appendChild(checkbox);

            testContainer.appendChild(form);

            formStateManager.trackForm(form);
            expect(formStateManager.hasDirtyForms()).toBe(false);

            checkbox.checked = true;
            expect(formStateManager.hasDirtyForms()).toBe(true);
        });
    });
});
