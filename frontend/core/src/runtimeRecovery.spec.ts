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
import {
    claimRecoveryReload,
    installRuntimeRecovery,
    RECOVERY_RELOAD_INTERVAL_MS,
    RECOVERY_RELOAD_STORAGE_KEY,
    watchedFailureURL,
    type RecoveryReloadStore,
    type RuntimeRecoveryDependencies
} from '@/runtimeRecovery';
import {createRuntimeVersionGate, RELOAD_INTERVAL_MS, RELOAD_STORAGE_KEY} from '@/core/RuntimeVersion';

const PAGE_LOCATION = {href: 'https://example.com/quotes/1', origin: 'https://example.com'};
const CORE_URL = 'https://example.com/_piko/dist/0123456789abcdef/ppframework.core.es.js';

function createMemoryStore(): RecoveryReloadStore & {values: Map<string, string>} {
    const values = new Map<string, string>();
    return {
        values,
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => {
            values.set(key, value);
        }
    };
}

function createElement(tagName: string, attributes: Record<string, string>): Element {
    const element = document.createElement(tagName);
    for (const [name, value] of Object.entries(attributes)) {
        element.setAttribute(name, value);
    }
    return element;
}

interface RecoveryHarness {
    page: Document;
    reload: ReturnType<typeof vi.fn>;
    reportError: ReturnType<typeof vi.fn>;
    failToLoad: (element: Element) => void;
    violatePolicy: (violation?: Partial<PolicyViolation>) => void;
}

interface PolicyViolation {
    blockedURI: string;
    disposition: string;
    effectiveDirective: string;
}

const ENFORCED_SCRIPT_VIOLATION: PolicyViolation = {
    blockedURI: CORE_URL,
    disposition: 'enforce',
    effectiveDirective: 'script-src-elem'
};

function installOnFreshPage(overrides: Partial<RuntimeRecoveryDependencies> = {}): RecoveryHarness {
    const page = document.implementation.createHTMLDocument('page');
    const reload = vi.fn();
    const reportError = vi.fn();
    installRuntimeRecovery({
        target: page,
        location: {...PAGE_LOCATION, reload},
        reloadStore: createMemoryStore(),
        now: () => 1_000_000,
        isOffline: () => false,
        defer: (task) => task(),
        reportError,
        ...overrides
    });
    return {
        page,
        reload,
        reportError,
        failToLoad: (element) => {
            const adopted = page.importNode(element, true);
            page.head.appendChild(adopted);
            adopted.dispatchEvent(new Event('error'));
        },
        violatePolicy: (violation = {}) => {
            const event = new Event('securitypolicyviolation');
            for (const [name, value] of Object.entries({...ENFORCED_SCRIPT_VIOLATION, ...violation})) {
                Object.defineProperty(event, name, {value});
            }
            page.dispatchEvent(event);
        }
    };
}

describe('watchedFailureURL', () => {
    it.each([
        ['a versioned runtime script', createElement('script', {type: 'module', src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}), CORE_URL],
        ['an unversioned runtime script', createElement('script', {src: '/_piko/dist/ppframework.core.es.js'}), 'https://example.com/_piko/dist/ppframework.core.es.js'],
        ['a generated page module', createElement('script', {type: 'module', src: '/_piko/assets/pk-js/pages/main.js'}), 'https://example.com/_piko/assets/pk-js/pages/main.js'],
        ['a runtime module preload', createElement('link', {rel: 'modulepreload', href: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}), CORE_URL],
        ['a module preload with several relations', createElement('link', {rel: 'prefetch ModulePreload', href: '/_piko/dist/x.js'}), 'https://example.com/_piko/dist/x.js'],
        ['an application script', createElement('script', {src: '/static/app.js'}), null],
        ['a runtime script on another origin', createElement('script', {src: 'https://cdn.example.net/_piko/dist/x.js'}), null],
        ['a stylesheet below the runtime path', createElement('link', {rel: 'stylesheet', href: '/_piko/dist/theme.css'}), null],
        ['an image below the runtime path', createElement('img', {src: '/_piko/dist/logo.png'}), null],
        ['an inline script', createElement('script', {}), null],
        ['an asset outside the generated modules', createElement('script', {src: '/_piko/assets/images/a.js'}), null]
    ])('reads %s', (_name, element, expected) => {
        expect(watchedFailureURL(element, PAGE_LOCATION)).toBe(expected);
    });

    it.each([
        ['nothing', null],
        ['the window', window],
        ['a plain object', {tagName: 'SCRIPT'}]
    ])('ignores an error targeted at %s', (_name, target) => {
        expect(watchedFailureURL(target, PAGE_LOCATION)).toBeNull();
    });
});

describe('claimRecoveryReload', () => {
    it('allows one forced reload per minute and records it in session storage', () => {
        const store = createMemoryStore();

        expect(claimRecoveryReload(store, 1_000_000)).toBe(true);
        expect(store.values.get(RECOVERY_RELOAD_STORAGE_KEY)).toBe('1000000');
        expect(claimRecoveryReload(store, 1_000_000 + RECOVERY_RELOAD_INTERVAL_MS - 1)).toBe(false);
        expect(claimRecoveryReload(store, 1_000_000 + RECOVERY_RELOAD_INTERVAL_MS)).toBe(true);
    });

    it('refuses the reload without session storage', () => {
        expect(claimRecoveryReload(null, 1_000_000)).toBe(false);
    });

    it('refuses the reload when session storage throws', () => {
        const throwingStore: RecoveryReloadStore = {
            getItem: () => {
                throw new Error('denied');
            },
            setItem: () => {
                throw new Error('denied');
            }
        };

        expect(claimRecoveryReload(throwingStore, 1_000_000)).toBe(false);
    });

    it('shares its key and interval with the runtime', () => {
        expect(RECOVERY_RELOAD_STORAGE_KEY).toBe(RELOAD_STORAGE_KEY);
        expect(RECOVERY_RELOAD_INTERVAL_MS).toBe(RELOAD_INTERVAL_MS);

        const store = createMemoryStore();
        const now = (): number => 2_000_000;
        expect(claimRecoveryReload(store, now())).toBe(true);
        expect(createRuntimeVersionGate({ownVersion: '0123456789abcdef', reloadStore: store, now}).claimReload()).toBe(false);
    });
});

describe('installRuntimeRecovery', () => {
    it('reloads the page once when the runtime fails to load', () => {
        const harness = installOnFreshPage();

        harness.failToLoad(createElement('link', {rel: 'modulepreload', href: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));
        harness.failToLoad(createElement('script', {type: 'module', src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));
        harness.failToLoad(createElement('script', {type: 'module', src: '/_piko/assets/pk-js/pages/main.js'}));

        expect(harness.reload).toHaveBeenCalledTimes(1);
        expect(harness.reportError).not.toHaveBeenCalled();
    });

    it('does not reload again on the page loaded by the reload, and names the failed URL', () => {
        const store = createMemoryStore();
        const first = installOnFreshPage({reloadStore: store});
        first.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        const second = installOnFreshPage({reloadStore: store, now: () => 1_000_000 + RECOVERY_RELOAD_INTERVAL_MS - 1});
        second.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        expect(first.reload).toHaveBeenCalledTimes(1);
        expect(second.reload).not.toHaveBeenCalled();
        expect(second.reportError).toHaveBeenCalledTimes(1);
        expect(second.reportError).toHaveBeenCalledWith(expect.stringContaining('[piko]'), CORE_URL);
    });

    it('does not reload when there is no session storage to keep the limit', () => {
        const harness = installOnFreshPage({reloadStore: null});

        harness.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        expect(harness.reload).not.toHaveBeenCalled();
        expect(harness.reportError).toHaveBeenCalledWith(expect.any(String), CORE_URL);
    });

    it('does not reload for a script the Content Security Policy blocked', () => {
        const harness = installOnFreshPage();

        harness.violatePolicy();
        harness.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        expect(harness.reload).not.toHaveBeenCalled();
        expect(harness.reportError).toHaveBeenCalledWith(expect.stringContaining('Content Security Policy'), CORE_URL);
    });

    it('does not reload when a policy violation follows the load error in the same task', () => {
        const deferred: Array<() => void> = [];
        const harness = installOnFreshPage({defer: (task) => deferred.push(task)});

        harness.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));
        harness.violatePolicy();
        for (const task of deferred) {
            task();
        }

        expect(harness.reload).not.toHaveBeenCalled();
    });

    it('does not reload after the policy blocked an import of a module that then fails', () => {
        const harness = installOnFreshPage();

        harness.violatePolicy({blockedURI: 'https://cdn.example.org/library.js'});
        harness.failToLoad(createElement('script', {type: 'module', src: '/_piko/assets/pk-js/pages/main.js'}));

        expect(harness.reload).not.toHaveBeenCalled();
        expect(harness.reportError).toHaveBeenCalledWith(expect.stringContaining('Content Security Policy'),
            'https://example.com/_piko/assets/pk-js/pages/main.js');
    });

    it('reloads when the policy only reported a script violation', () => {
        const harness = installOnFreshPage();

        harness.violatePolicy({disposition: 'report'});
        harness.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        expect(harness.reload).toHaveBeenCalledTimes(1);
    });

    it('reloads after the policy blocked something other than a script', () => {
        const harness = installOnFreshPage();

        harness.violatePolicy({blockedURI: CORE_URL, effectiveDirective: 'style-src-elem'});
        harness.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        expect(harness.reload).toHaveBeenCalledTimes(1);
    });

    it('does not reload while the browser is offline', () => {
        const store = createMemoryStore();
        const harness = installOnFreshPage({reloadStore: store, isOffline: () => true});

        harness.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        expect(harness.reload).not.toHaveBeenCalled();
        expect(store.values.size).toBe(0);
        expect(harness.reportError).toHaveBeenCalledWith(expect.stringContaining('offline'), CORE_URL);
    });

    it('says the page is already reloading when the runtime took the reload on this page', () => {
        const store = createMemoryStore();
        let now = 1_000_000;
        const harness = installOnFreshPage({reloadStore: store, now: () => now});
        now += 5;
        expect(createRuntimeVersionGate({ownVersion: '0123456789abcdef', reloadStore: store, now: () => now}).claimReload()).toBe(true);

        harness.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        expect(harness.reload).not.toHaveBeenCalled();
        expect(harness.reportError).toHaveBeenCalledWith(expect.stringContaining('already reloading'), CORE_URL);
    });

    it('reloads when the recorded reload is later than now', () => {
        const store = createMemoryStore();
        store.values.set(RECOVERY_RELOAD_STORAGE_KEY, String(1_000_000 + 10 * RECOVERY_RELOAD_INTERVAL_MS));
        const harness = installOnFreshPage({reloadStore: store});

        harness.failToLoad(createElement('script', {src: '/_piko/dist/0123456789abcdef/ppframework.core.es.js'}));

        expect(harness.reload).toHaveBeenCalledTimes(1);
    });

    it('ignores scripts that a reload cannot fix', () => {
        const harness = installOnFreshPage();

        harness.failToLoad(createElement('script', {src: '/static/app.js'}));
        harness.failToLoad(createElement('img', {src: '/_piko/dist/logo.png'}));

        expect(harness.reload).not.toHaveBeenCalled();
        expect(harness.reportError).not.toHaveBeenCalled();
    });

    it('listens for load errors and policy violations in the capture phase', () => {
        const addEventListener = vi.fn();

        installRuntimeRecovery({
            target: {addEventListener},
            location: {...PAGE_LOCATION, reload: vi.fn()},
            reloadStore: null,
            now: () => 0,
            isOffline: () => false,
            defer: (task) => task(),
            reportError: vi.fn()
        });

        expect(addEventListener).toHaveBeenCalledWith('error', expect.any(Function), true);
        expect(addEventListener).toHaveBeenCalledWith('securitypolicyviolation', expect.any(Function), true);
    });
});
