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

import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {
    ActionRepeatGuard,
    DEFAULT_GUARD_HOLD_MS,
    guardHoldMilliseconds,
    requestSubmitFrom
} from '@/core/actionRepeatGuard';

describe('actionRepeatGuard', () => {
    let guard: ActionRepeatGuard;
    let form: HTMLFormElement;
    let saveButton: HTMLElement;
    let sends: number;
    let settleSends: Array<() => void>;

    beforeEach(() => {
        guard = new ActionRepeatGuard();
        sends = 0;
        settleSends = [];

        form = document.body.appendChild(document.createElement('form'));
        form.noValidate = true;
        saveButton = document.body.appendChild(document.createElement('pp-button'));

        form.addEventListener('submit', event => {
            event.preventDefault();
            void guard.run(
                {actionName: 'suppliers.Create', element: form, event, holdMilliseconds: DEFAULT_GUARD_HOLD_MS},
                () => new Promise<void>(resolve => {
                    sends++;
                    settleSends.push(resolve);
                })
            );
        });
    });

    afterEach(() => {
        document.body.innerHTML = '';
        vi.restoreAllMocks();
    });

    describe('requestSubmitFrom()', () => {
        it('sends one action for two submits from a Save button outside the form', () => {
            requestSubmitFrom(form, saveButton);
            requestSubmitFrom(form, saveButton);

            expect(sends).toBe(1);
        });

        it('marks the Save button and the form busy until the send settles', async () => {
            requestSubmitFrom(form, saveButton);

            expect(saveButton.getAttribute('aria-busy')).toBe('true');
            expect(form.getAttribute('aria-busy')).toBe('true');

            settleSends[0]();
            await vi.waitFor(() => {
                expect(saveButton.hasAttribute('aria-busy')).toBe(false);
            });
            expect(form.hasAttribute('aria-busy')).toBe(false);
        });

        it('allows a repeat when the Save button has pk-allow-repeat', () => {
            saveButton.setAttribute('pk-allow-repeat', '');

            requestSubmitFrom(form, saveButton);
            requestSubmitFrom(form, saveButton);

            expect(sends).toBe(2);
        });

        it('does not record the trigger for a later submit of the form', async () => {
            requestSubmitFrom(form, saveButton);
            settleSends[0]();
            await vi.waitFor(() => {
                expect(form.hasAttribute('aria-busy')).toBe(false);
            });

            form.requestSubmit();

            expect(sends).toBe(2);
            expect(form.getAttribute('aria-busy')).toBe('true');
            expect(saveButton.hasAttribute('aria-busy')).toBe(false);
        });
    });

    describe('isInFlight()', () => {
        it('is true only while the send is in flight', async () => {
            let settle = (): void => {};
            const button = document.body.appendChild(document.createElement('button'));
            const send = guard.run(
                {actionName: 'quotes.Upsert', element: button, holdMilliseconds: DEFAULT_GUARD_HOLD_MS},
                () => new Promise<void>(resolve => {
                    settle = resolve;
                })
            );

            expect(guard.isInFlight(button, 'quotes.Upsert')).toBe(true);
            settle();
            await send;
            expect(guard.isInFlight(button, 'quotes.Upsert')).toBe(false);
        });

        it('is false after a send that throws', async () => {
            const button = document.body.appendChild(document.createElement('button'));

            await expect(guard.run(
                {actionName: 'quotes.Upsert', element: button, holdMilliseconds: DEFAULT_GUARD_HOLD_MS},
                () => {
                    throw new Error('handler failed');
                }
            )).rejects.toThrow('handler failed');

            expect(guard.isInFlight(button, 'quotes.Upsert')).toBe(false);
            expect(button.hasAttribute('aria-busy')).toBe(false);
        });
    });

    describe('guardHoldMilliseconds()', () => {
        it.each([
            {name: 'no timeout', timeout: undefined, want: DEFAULT_GUARD_HOLD_MS},
            {name: 'a shorter timeout', timeout: 5_000, want: DEFAULT_GUARD_HOLD_MS},
            {name: 'a longer timeout', timeout: 300_000, want: 300_000}
        ])('returns the hold time for $name', ({timeout, want}) => {
            expect(guardHoldMilliseconds(timeout)).toBe(want);
        });
    });
});
