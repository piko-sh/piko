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

import {describe, it, expect, vi} from 'vitest';
import {ACTIVE_RUNTIME_PROPERTY, claimRuntime, RuntimeConflictError, type ActiveRuntime} from '@/core/runtimeGuard';

const FIRST: ActiveRuntime = {version: '0123456789abcdef', moduleURL: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'};
const SECOND: ActiveRuntime = {version: 'fedcba9876543210', moduleURL: '/_piko/dist/fedcba9876543210/ppframework.core.es.js'};

function createDeps(win: object = {}, allowReload = true) {
    return {
        win,
        reload: vi.fn(),
        claimReload: vi.fn(() => allowReload)
    };
}

describe('claimRuntime', () => {
    it('records the first runtime in the window without reloading', () => {
        const deps = createDeps();

        claimRuntime(deps, FIRST);

        expect((deps.win as Record<string, unknown>)[ACTIVE_RUNTIME_PROPERTY]).toEqual(FIRST);
        expect(deps.reload).not.toHaveBeenCalled();
        expect(deps.claimReload).not.toHaveBeenCalled();
    });

    it('keeps the record out of enumeration and read-only', () => {
        const deps = createDeps();
        claimRuntime(deps, FIRST);

        expect(Object.keys(deps.win)).not.toContain(ACTIVE_RUNTIME_PROPERTY);
        expect(() => {
            'use strict';
            (deps.win as Record<string, unknown>)[ACTIVE_RUNTIME_PROPERTY] = SECOND;
        }).toThrow(TypeError);
    });

    it('stops a second runtime build and reloads the page', () => {
        const deps = createDeps();
        claimRuntime(deps, FIRST);

        expect(() => claimRuntime(deps, SECOND)).toThrow(RuntimeConflictError);
        expect(deps.reload).toHaveBeenCalledTimes(1);
        expect((deps.win as Record<string, unknown>)[ACTIVE_RUNTIME_PROPERTY]).toEqual(FIRST);
    });

    it('stops a second copy of the same build reached through another URL', () => {
        const deps = createDeps();
        claimRuntime(deps, FIRST);

        expect(() => claimRuntime(deps, {...FIRST, moduleURL: '/_piko/dist/ppframework.core.es.js'})).toThrow(RuntimeConflictError);
    });

    it('stops a runtime loading into a page run by a runtime from before versioning', () => {
        const deps = createDeps({__pikoPageContext: {loadModule: () => undefined}});

        expect(() => claimRuntime(deps, SECOND)).toThrow(/an earlier runtime/);
        expect(deps.reload).toHaveBeenCalledTimes(1);
    });

    it.each([
        {name: 'an element whose id names the runtime record', win: () => ({[ACTIVE_RUNTIME_PROPERTY]: document.createElement('div')})},
        {name: 'a record without a module URL', win: () => ({[ACTIVE_RUNTIME_PROPERTY]: {version: FIRST.version}})},
        {name: 'an element whose id names the earlier page context', win: () => ({__pikoPageContext: document.createElement('div')})},
    ])('starts when the window holds $name', ({win}) => {
        const deps = createDeps(win());

        expect(() => claimRuntime(deps, FIRST)).not.toThrow();
        expect(Reflect.get(deps.win, ACTIVE_RUNTIME_PROPERTY)).toEqual(FIRST);
        expect(deps.reload).not.toHaveBeenCalled();
    });

    it('does not reload again inside the rate limit, but still refuses to start', () => {
        const deps = createDeps({}, false);
        claimRuntime(deps, FIRST);

        expect(() => claimRuntime(deps, SECOND)).toThrow(/Reload the page to update it/);
        expect(deps.reload).not.toHaveBeenCalled();
    });
});
