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
import { createModalManager, type ModalManager } from '@/core/ModalManager';

describe('ModalManager', () => {
    let modalManager: ModalManager;
    let triggerElement: HTMLButtonElement;

    beforeEach(() => {
        modalManager = createModalManager();
        triggerElement = document.createElement('button');
        document.body.appendChild(triggerElement);
    });

    afterEach(() => {
        triggerElement.remove();
        document.querySelectorAll('[data-test-modal]').forEach(el => el.remove());
        vi.clearAllMocks();
    });

    describe('openIfAvailable', () => {
        it('should dispatch fallback event when modal not found', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const eventSpy = vi.fn();
            triggerElement.addEventListener('modal-not-found', eventSpy);

            await modalManager.openIfAvailable({
                selector: '#nonexistent-modal',
                triggerElement
            });

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('Could not find modal "#nonexistent-modal"')
            );
            expect(eventSpy).toHaveBeenCalled();
            expect(eventSpy.mock.calls[0][0]).toBeInstanceOf(CustomEvent);

            warnSpy.mockRestore();
        });

        it('should dispatch custom fallback event name when specified', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const eventSpy = vi.fn();
            triggerElement.addEventListener('custom-fallback', eventSpy);

            await modalManager.openIfAvailable({
                selector: '#nonexistent-modal',
                triggerElement,
                fallbackEventName: 'custom-fallback'
            });

            expect(eventSpy).toHaveBeenCalled();

            warnSpy.mockRestore();
        });

        it('should call request() on modal with correct options', async () => {
            const mockRequest = vi.fn().mockResolvedValue(true);
            const modalEl = document.createElement('div');
            modalEl.id = 'test-modal';
            modalEl.setAttribute('data-test-modal', '');
            (modalEl as unknown as { request: typeof mockRequest }).request = mockRequest;
            document.body.appendChild(modalEl);

            const params = new Map([['itemId', '123']]);

            await modalManager.openIfAvailable({
                selector: '#test-modal',
                params,
                title: 'Confirm Action',
                message: 'Are you sure?',
                cancelLabel: 'Cancel',
                confirmLabel: 'Confirm',
                confirmAction: 'deleteItem',
                triggerElement
            });

            expect(mockRequest).toHaveBeenCalledWith({
                modal_title: 'Confirm Action',
                message: 'Are you sure?',
                cancel_label: 'Cancel',
                confirm_label: 'Confirm',
                confirm_action: 'deleteItem',
                params: { itemId: '123' }
            });
        });

        it('should dispatch modal-confirmed when request() returns true', async () => {
            const mockRequest = vi.fn().mockResolvedValue(true);
            const modalEl = document.createElement('div');
            modalEl.id = 'confirm-modal';
            modalEl.setAttribute('data-test-modal', '');
            (modalEl as unknown as { request: typeof mockRequest }).request = mockRequest;
            document.body.appendChild(modalEl);

            const eventSpy = vi.fn();
            triggerElement.addEventListener('modal-confirmed', eventSpy);

            await modalManager.openIfAvailable({
                selector: '#confirm-modal',
                triggerElement
            });

            expect(eventSpy).toHaveBeenCalled();
            const event = eventSpy.mock.calls[0][0] as CustomEvent;
            expect(event.type).toBe('modal-confirmed');
            expect(event.bubbles).toBe(true);
            expect(event.composed).toBe(true);
        });

        it('should dispatch modal-cancelled when request() returns false', async () => {
            const mockRequest = vi.fn().mockResolvedValue(false);
            const modalEl = document.createElement('div');
            modalEl.id = 'cancel-modal';
            modalEl.setAttribute('data-test-modal', '');
            (modalEl as unknown as { request: typeof mockRequest }).request = mockRequest;
            document.body.appendChild(modalEl);

            const eventSpy = vi.fn();
            triggerElement.addEventListener('modal-cancelled', eventSpy);

            await modalManager.openIfAvailable({
                selector: '#cancel-modal',
                triggerElement
            });

            expect(eventSpy).toHaveBeenCalled();
            const event = eventSpy.mock.calls[0][0] as CustomEvent;
            expect(event.type).toBe('modal-cancelled');
            expect(event.bubbles).toBe(true);
            expect(event.composed).toBe(true);
        });

        it('should fallback to setting open attribute when modal has no request()', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            const modalEl = document.createElement('div');
            modalEl.id = 'simple-modal';
            modalEl.setAttribute('data-test-modal', '');
            document.body.appendChild(modalEl);

            await modalManager.openIfAvailable({
                selector: '#simple-modal',
                triggerElement
            });

            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining('does not have a request() function')
            );
            expect(modalEl.getAttribute('open')).toBe('true');

            warnSpy.mockRestore();
        });

        it('should use default values for optional parameters', async () => {
            const mockRequest = vi.fn().mockResolvedValue(true);
            const modalEl = document.createElement('div');
            modalEl.id = 'default-modal';
            modalEl.setAttribute('data-test-modal', '');
            (modalEl as unknown as { request: typeof mockRequest }).request = mockRequest;
            document.body.appendChild(modalEl);

            await modalManager.openIfAvailable({
                selector: '#default-modal',
                triggerElement
            });

            expect(mockRequest).toHaveBeenCalledWith({
                modal_title: '',
                message: '',
                cancel_label: '',
                confirm_label: '',
                confirm_action: '',
                params: {}
            });
        });

        it('should convert Map params to object', async () => {
            const mockRequest = vi.fn().mockResolvedValue(true);
            const modalEl = document.createElement('div');
            modalEl.id = 'params-modal';
            modalEl.setAttribute('data-test-modal', '');
            (modalEl as unknown as { request: typeof mockRequest }).request = mockRequest;
            document.body.appendChild(modalEl);

            const params = new Map<string, string>([
                ['key1', 'value1'],
                ['key2', 'value2']
            ]);

            await modalManager.openIfAvailable({
                selector: '#params-modal',
                params,
                triggerElement
            });

            expect(mockRequest).toHaveBeenCalledWith(
                expect.objectContaining({
                    params: { key1: 'value1', key2: 'value2' }
                })
            );
        });
    });
});
