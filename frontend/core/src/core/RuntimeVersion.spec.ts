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

import {describe, it, expect} from 'vitest';
import {
    createRuntimeVersionGate,
    isRuntimeVersion,
    runtimeVersionFromModuleURL,
    runtimeVersionGate,
    RELOAD_STORAGE_KEY,
    RUNTIME_VERSION_HEADER,
    type ReloadStore
} from '@/core/RuntimeVersion';

const OWN_VERSION = '0123456789abcdef';
const OTHER_VERSION = 'fedcba9876543210';

function headersFrom(version: string | null): Headers {
    const headers = new Headers();
    if (version !== null) {
        headers.set(RUNTIME_VERSION_HEADER, version);
    }
    return headers;
}

function createMemoryStore(): ReloadStore & {values: Map<string, string>} {
    const values = new Map<string, string>();
    return {
        values,
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => {
            values.set(key, value);
        }
    };
}

describe('isRuntimeVersion', () => {
    it.each([
        [OWN_VERSION, true],
        ['0123456789ABCDEF', false],
        ['0123456789abcde', false],
        ['0123456789abcdef0', false],
        ['ppframework.core', false],
        ['', false],
        [null, false],
        [undefined, false]
    ])('treats %s as a version: %s', (value, expected) => {
        expect(isRuntimeVersion(value)).toBe(expected);
    });
});

describe('runtimeVersionFromModuleURL', () => {
    it.each([
        ['https://example.com/_piko/dist/0123456789abcdef/ppframework.core.es.js', OWN_VERSION],
        ['https://example.com/cdn/prefix/0123456789abcdef/ppframework.core.min.es.js', OWN_VERSION],
        ['https://example.com/_piko/dist/ppframework.core.es.js', null],
        ['https://example.com/_piko/dist/0123456789abcdef/', null],
        ['blob:https://example.com/5b1c2f9e-0000-4000-8000-000000000000', null],
        ['file:///home/user/frontend/core/src/core/RuntimeVersion.ts', null],
        ['not a url', null]
    ])('reads %s as %s', (moduleURL, expected) => {
        expect(runtimeVersionFromModuleURL(moduleURL)).toBe(expected);
    });
});

describe('createRuntimeVersionGate', () => {
    it('reports no change while the server runs the same build', () => {
        const gate = createRuntimeVersionGate({ownVersion: OWN_VERSION});

        expect(gate.observe(headersFrom(OWN_VERSION))).toBe(false);
        expect(gate.isStale()).toBe(false);
    });

    it('reports a change, and stays stale, once another build answers', () => {
        const gate = createRuntimeVersionGate({ownVersion: OWN_VERSION});

        expect(gate.observe(headersFrom(OTHER_VERSION))).toBe(true);
        expect(gate.observe(headersFrom(OWN_VERSION))).toBe(false);
        expect(gate.isStale()).toBe(true);
    });

    it.each([
        ['the response names no version', OWN_VERSION, null],
        ['the response names a malformed version', OWN_VERSION, 'FEDCBA9876543210'],
        ['this tab does not know its own version', null, OTHER_VERSION],
        ['this tab was given a malformed version', 'v1', OTHER_VERSION]
    ])('never reports a change when %s', (_name, ownVersion, serverVersion) => {
        const gate = createRuntimeVersionGate({ownVersion});

        expect(gate.observe(headersFrom(serverVersion))).toBe(false);
        expect(gate.isStale()).toBe(false);
    });

    it.each([
        ['missing', undefined],
        ['null', null]
    ])('never reports a change when the response headers are %s', (_name, headers) => {
        const gate = createRuntimeVersionGate({ownVersion: OWN_VERSION});

        expect(gate.observe(headers)).toBe(false);
        expect(gate.isStale()).toBe(false);
    });

    it('exposes only a well-formed own version', () => {
        expect(createRuntimeVersionGate({ownVersion: OWN_VERSION}).ownVersion).toBe(OWN_VERSION);
        expect(createRuntimeVersionGate({ownVersion: 'v1'}).ownVersion).toBeNull();
    });
});

describe('claimReload', () => {
    it('allows one forced reload per minute and records it in session storage', () => {
        const store = createMemoryStore();
        let now = 1_000_000;
        const gate = createRuntimeVersionGate({ownVersion: OWN_VERSION, reloadStore: store, now: () => now});

        expect(gate.claimReload()).toBe(true);
        expect(store.values.size).toBe(1);

        now += 59_999;
        expect(gate.claimReload()).toBe(false);

        now += 1;
        expect(gate.claimReload()).toBe(true);
    });

    it('shares the limit with a page loaded after the reload, through session storage', () => {
        const store = createMemoryStore();
        const now = (): number => 2_000_000;

        expect(createRuntimeVersionGate({ownVersion: OWN_VERSION, reloadStore: store, now}).claimReload()).toBe(true);
        expect(createRuntimeVersionGate({ownVersion: OTHER_VERSION, reloadStore: store, now}).claimReload()).toBe(false);
    });

    it('allows the reload when the recorded reload is later than now', () => {
        const store = createMemoryStore();
        store.values.set(RELOAD_STORAGE_KEY, String(1_000_000 + 10 * 60_000));

        expect(createRuntimeVersionGate({ownVersion: OWN_VERSION, reloadStore: store, now: () => 1_000_000}).claimReload()).toBe(true);
    });

    it('ignores an unreadable stored value', () => {
        const store = createMemoryStore();
        store.values.set(RELOAD_STORAGE_KEY, 'garbage');

        expect(createRuntimeVersionGate({ownVersion: OWN_VERSION, reloadStore: store}).claimReload()).toBe(true);
    });

    it('refuses the reload when session storage throws', () => {
        const throwingStore: ReloadStore = {
            getItem: () => {
                throw new Error('denied');
            },
            setItem: () => {
                throw new Error('denied');
            }
        };

        expect(createRuntimeVersionGate({ownVersion: OWN_VERSION, reloadStore: throwingStore}).claimReload()).toBe(false);
    });

    it('refuses the reload when there is no session storage', () => {
        expect(createRuntimeVersionGate({ownVersion: OWN_VERSION}).claimReload()).toBe(false);
    });

    it('refuses the reload when the record cannot be written', () => {
        const store = createMemoryStore();
        const readOnlyStore: ReloadStore = {
            getItem: (key) => store.getItem(key),
            setItem: () => {
                throw new Error('quota exceeded');
            }
        };

        expect(createRuntimeVersionGate({ownVersion: OWN_VERSION, reloadStore: readOnlyStore}).claimReload()).toBe(false);
    });
});

describe('runtimeVersionGate', () => {
    it('does not know its version when the module was not loaded from a versioned URL', () => {
        expect(runtimeVersionGate.ownVersion).toBeNull();
        expect(runtimeVersionGate.observe(headersFrom(OTHER_VERSION))).toBe(false);
    });
});
