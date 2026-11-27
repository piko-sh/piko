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

import {describe, it, expect, beforeEach, vi} from 'vitest';
import {registerDiagnosticEmitter, reportDiagnostic} from './DiagnosticReporter';
import type {DiagnosticPayload} from './HookManager';
import {_endPageVisit} from '@/pk/lifecycle';

const MAX_DIAGNOSTICS_PER_PAGE_VISIT = 100;

describe('reportDiagnostic', () => {
    let emitter: ReturnType<typeof vi.fn<(payload: DiagnosticPayload) => void>>;

    beforeEach(() => {
        emitter = vi.fn<(payload: DiagnosticPayload) => void>();
        registerDiagnosticEmitter(emitter);
        emitter.mockClear();
        _endPageVisit();
    });

    it('sends the kind, name, context, source and a timestamp', () => {
        const before = Date.now();

        reportDiagnostic({kind: 'ref-not-found', name: 'saveButton', source: 'pk.refs', context: 'partial-7'});

        expect(emitter).toHaveBeenCalledTimes(1);
        const payload = emitter.mock.calls[0][0];
        expect(payload).toMatchObject({
            kind: 'ref-not-found',
            name: 'saveButton',
            source: 'pk.refs',
            context: 'partial-7'
        });
        expect(payload.timestamp).toBeGreaterThanOrEqual(before);
    });

    it('uses the page path as the context when no partial is given', () => {
        reportDiagnostic({kind: 'partial-not-found', name: 'cart', source: 'piko.partials.reload'});

        expect(emitter.mock.calls[0][0].context).toBe(window.location.pathname);
    });

    it('sends each kind and name only once per page visit', () => {
        for (let attempt = 0; attempt < 10; attempt++) {
            reportDiagnostic({kind: 'partial-not-found', name: 'cart', source: 'piko.partials.reload'});
        }
        reportDiagnostic({kind: 'ref-not-found', name: 'cart', source: 'pk.refs'});

        expect(emitter).toHaveBeenCalledTimes(2);
    });

    it('sends the same kind and name again after the page visit ends', () => {
        reportDiagnostic({kind: 'modal-not-found', name: '#confirm', source: 'openModal'});
        _endPageVisit();
        reportDiagnostic({kind: 'modal-not-found', name: '#confirm', source: 'openModal'});

        expect(emitter).toHaveBeenCalledTimes(2);
    });

    it('stops sending once a page visit reaches the limit of distinct diagnostics', () => {
        for (let index = 0; index < MAX_DIAGNOSTICS_PER_PAGE_VISIT + 20; index++) {
            reportDiagnostic({kind: 'ref-not-found', name: `item-${index}`, source: 'pk.refs'});
        }

        expect(emitter).toHaveBeenCalledTimes(MAX_DIAGNOSTICS_PER_PAGE_VISIT);
    });
});
