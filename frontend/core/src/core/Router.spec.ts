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

import {describe, it, expect, beforeEach, afterEach, vi} from 'vitest';
import {createRouter} from '@/core/Router';
import type {Router, PageLoadScrollOptions, RouterDependencies} from '@/core/Router';
import type {FetchClient, FetchResult, FetchClientOptions} from '@/core/FetchClient';
import type {LoaderUI} from '@/services/LoaderUI';
import type {ErrorDisplay} from '@/services/ErrorDisplay';
import type {EventBus} from '@/services/EventBus';
import type {FormStateManager} from '@/services/FormStateManager';
import type {A11yAnnouncer} from '@/services/A11yAnnouncer';
import type {HookManager} from '@/services/HookManager';
import type {WindowOperations, DOMOperations} from '@/core/BrowserAPIs';

function createMockWindowOps(overrides: Partial<WindowOperations> = {}): WindowOperations {
    return {
        getLocation: vi.fn(() => window.location),
        getLocationOrigin: vi.fn(() => 'http://localhost'),
        getLocationHref: vi.fn(() => 'http://localhost/current'),
        setLocationHref: vi.fn(),
        locationReload: vi.fn(),
        historyPushState: vi.fn(),
        historyReplaceState: vi.fn(),
        getHistoryState: vi.fn(() => null),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        getScrollY: vi.fn(() => 0),
        scrollTo: vi.fn(),
        setScrollRestoration: vi.fn(),
        getScrollRestoration: vi.fn(() => 'manual' as ScrollRestoration),
        ...overrides,
    };
}

function createMockDOMOps(overrides: Partial<DOMOperations> = {}): DOMOperations {
    return {
        createElement: vi.fn((tag: string) => document.createElement(tag)) as DOMOperations['createElement'],
        createTextNode: vi.fn((data: string) => document.createTextNode(data)),
        createComment: vi.fn((data: string) => document.createComment(data)),
        createDocumentFragment: vi.fn(() => document.createDocumentFragment()),
        querySelector: vi.fn(() => null) as DOMOperations['querySelector'],
        querySelectorAll: vi.fn(() => document.querySelectorAll('.noop')) as DOMOperations['querySelectorAll'],
        getElementById: vi.fn(() => null),
        getHead: vi.fn(() => document.head),
        getActiveElement: vi.fn(() => null),
        parseHTML: vi.fn((html: string) => new DOMParser().parseFromString(html, 'text/html')),
        ...overrides,
    };
}

function createMockFetchClient(overrides: Partial<FetchClient> = {}): FetchClient {
    return {
        get: vi.fn(async (): Promise<FetchResult> => [true, '<html><head><title>Test</title></head><body><div id="app">content</div></body></html>']),
        post: vi.fn(async () => new Response('ok')),
        abort: vi.fn(),
        getController: vi.fn(() => null),
        ...overrides,
    };
}

function createMockLoader(): LoaderUI {
    return {
        show: vi.fn(),
        hide: vi.fn(),
        setProgress: vi.fn(),
        destroy: vi.fn(),
    };
}

function createMockErrorDisplay(): ErrorDisplay {
    return {
        show: vi.fn(),
        clear: vi.fn(),
    };
}

function createMockEventBus(): EventBus {
    return {
        on: vi.fn(() => vi.fn()),
        off: vi.fn(),
        emit: vi.fn(),
        clear: vi.fn(),
    };
}

function createMockFormStateManager(overrides: Partial<FormStateManager> = {}): FormStateManager {
    return {
        trackForm: vi.fn(),
        untrackForm: vi.fn(),
        markFormClean: vi.fn(),
        hasDirtyForms: vi.fn(() => false),
        getDirtyFormIds: vi.fn(() => []),
        confirmNavigation: vi.fn(() => true),
        scanAndTrackForms: vi.fn(),
        untrackAll: vi.fn(),
        destroy: vi.fn(),
        ...overrides,
    };
}

function createMockA11yAnnouncer(): A11yAnnouncer {
    return {
        announce: vi.fn(),
        announceNavigation: vi.fn(),
        announceLoading: vi.fn(),
        announceError: vi.fn(),
        destroy: vi.fn(),
    };
}

function createMockHookManager(): HookManager {
    return {
        api: {} as HookManager['api'],
        emit: vi.fn(),
        processQueue: vi.fn(),
        setReady: vi.fn(),
    };
}

const VALID_HTML = '<html><head><title>New Page</title></head><body><div id="app">content</div></body></html>';

const NO_APP_HTML = '<html><head><title>Bad</title></head><body><div>no app</div></body></html>';

describe('Router', () => {
    let windowOps: WindowOperations;
    let domOps: DOMOperations;
    let fetchClient: FetchClient;
    let loader: LoaderUI;
    let errorDisplay: ErrorDisplay;
    let eventBus: EventBus;
    let onPageLoad: RouterDependencies['onPageLoad'];
    let router: Router;

    beforeEach(() => {
        windowOps = createMockWindowOps();
        domOps = createMockDOMOps();
        fetchClient = createMockFetchClient();
        loader = createMockLoader();
        errorDisplay = createMockErrorDisplay();
        eventBus = createMockEventBus();
        onPageLoad = vi.fn() as unknown as RouterDependencies['onPageLoad'];

        router = createRouter({
            fetchClient,
            loader,
            errorDisplay,
            eventBus,
            onPageLoad,
            windowOps,
            domOps,
        });
    });

    afterEach(() => {
        router.destroy();
        vi.restoreAllMocks();
    });

    describe('createRouter()', () => {
        it('should set scroll restoration to manual on creation', () => {
            expect(windowOps.setScrollRestoration).toHaveBeenCalledWith('manual');
        });

        it('should register a popstate listener on creation', () => {
            expect(windowOps.addEventListener).toHaveBeenCalledWith('popstate', expect.any(Function));
        });

        it('should initially report isNavigating() as false', () => {
            expect(router.isNavigating()).toBe(false);
        });
    });

    describe('navigateTo() - success path', () => {
        it('should show and hide the loader around navigation', async () => {
            await router.navigateTo('http://localhost/page');

            expect(loader.show).toHaveBeenCalled();
            expect(loader.hide).toHaveBeenCalled();
        });

        it('should call onPageLoad with the parsed document', async () => {
            await router.navigateTo('http://localhost/page');

            expect(onPageLoad).toHaveBeenCalledWith(
                expect.any(Object),
                'http://localhost/page',
                expect.any(Object)
            );
        });

        it('should emit navigation:start and navigation:complete events', async () => {
            await router.navigateTo('http://localhost/page');

            expect(eventBus.emit).toHaveBeenCalledWith('navigation:start', {url: 'http://localhost/page'});
            expect(eventBus.emit).toHaveBeenCalledWith('navigation:complete', {url: 'http://localhost/page'});
        });

        it('should prevent default on the event when provided', async () => {
            const evt = new Event('click');
            const preventSpy = vi.spyOn(evt, 'preventDefault');

            await router.navigateTo('http://localhost/page', evt);

            expect(preventSpy).toHaveBeenCalled();
        });

        it('should set isNavigating() to false after navigation completes', async () => {
            await router.navigateTo('http://localhost/page');

            expect(router.isNavigating()).toBe(false);
        });

        it('should call fetchClient.get with the URL plus _f=1 query', async () => {
            await router.navigateTo('http://localhost/page');

            const getCall = vi.mocked(fetchClient.get).mock.calls[0];
            expect(getCall[0]).toContain('_f=1');
        });
    });

    describe('history state management', () => {
        it('should save current scrollY before pushState', async () => {
            vi.mocked(windowOps.getScrollY).mockReturnValue(42);

            await router.navigateTo('http://localhost/new-page');

            expect(windowOps.historyReplaceState).toHaveBeenCalledWith(
                {scrollY: 42},
                '',
                'http://localhost/current'
            );
        });

        it('should push new state with scrollY 0 by default', async () => {
            await router.navigateTo('http://localhost/new-page');

            expect(windowOps.historyPushState).toHaveBeenCalledWith(
                {scrollY: 0},
                '',
                'http://localhost/new-page'
            );
        });

        it('should use replaceState instead of pushState when replaceHistory is true', async () => {
            await router.navigateTo('http://localhost/new-page', undefined, {replaceHistory: true});

            expect(windowOps.historyReplaceState).toHaveBeenCalledTimes(2);
            expect(windowOps.historyPushState).not.toHaveBeenCalled();
        });

        it('should NOT update history state during popstate navigation', async () => {
            await router.navigateTo('http://localhost/page', undefined, {isPopState: true});

            expect(windowOps.historyPushState).not.toHaveBeenCalled();
            const replaceStateCalls = vi.mocked(windowOps.historyReplaceState).mock.calls;
            for (const call of replaceStateCalls) {
                expect(call[2]).not.toBe('http://localhost/current');
            }
        });
    });

    describe('scroll restoration via onPageLoad options', () => {
        it('should pass restoreScrollY for popstate navigation when state has scrollY', async () => {
            await router.navigateTo('http://localhost/page', undefined, {
                isPopState: true,
                restoreScrollY: 150,
            });

            const scrollOptions = vi.mocked(onPageLoad).mock.calls[0][2] as PageLoadScrollOptions;
            expect(scrollOptions.restoreScrollY).toBe(150);
        });

        it('should pass hash for non-popstate navigation with a hash URL', async () => {
            await router.navigateTo('http://localhost/page#section', undefined);

            const scrollOptions = vi.mocked(onPageLoad).mock.calls[0][2] as PageLoadScrollOptions;
            expect(scrollOptions.hash).toBe('#section');
            expect(scrollOptions.restoreScrollY).toBeUndefined();
        });

        it('should pass hash when popstate has no restoreScrollY', async () => {
            await router.navigateTo('http://localhost/page#anchor', undefined, {
                isPopState: true,
                restoreScrollY: undefined,
            });

            const scrollOptions = vi.mocked(onPageLoad).mock.calls[0][2] as PageLoadScrollOptions;
            expect(scrollOptions.hash).toBe('#anchor');
        });

        it('should not pass hash when popstate has restoreScrollY', async () => {
            await router.navigateTo('http://localhost/page#anchor', undefined, {
                isPopState: true,
                restoreScrollY: 200,
            });

            const scrollOptions = vi.mocked(onPageLoad).mock.calls[0][2] as PageLoadScrollOptions;
            expect(scrollOptions.hash).toBeUndefined();
        });
    });

    describe('error fallback', () => {
        it('should show error and redirect when fetch returns failure', async () => {
            vi.mocked(fetchClient.get).mockResolvedValue([false, null]);

            await router.navigateTo('http://localhost/bad-page');

            expect(errorDisplay.show).toHaveBeenCalledWith(
                expect.stringContaining('failed')
            );
            expect(windowOps.setLocationHref).toHaveBeenCalledWith('http://localhost/bad-page');
        });

        it('should show error and redirect when response has no #app', async () => {
            vi.mocked(fetchClient.get).mockResolvedValue([true, NO_APP_HTML]);

            await router.navigateTo('http://localhost/no-app');

            expect(errorDisplay.show).toHaveBeenCalledWith(
                expect.stringContaining('No #app')
            );
            expect(windowOps.setLocationHref).toHaveBeenCalledWith('http://localhost/no-app');
        });

        it('should redirect via setLocationHref when fetch throws a non-abort error', async () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

            vi.mocked(fetchClient.get).mockRejectedValue(new Error('Network failure'));

            await router.navigateTo('http://localhost/fail');

            expect(consoleSpy).toHaveBeenCalledWith('navigateTo error:', expect.any(Error));
            expect(errorDisplay.show).toHaveBeenCalled();
            expect(windowOps.setLocationHref).toHaveBeenCalledWith('http://localhost/fail');

            consoleSpy.mockRestore();
        });
    });

    describe('AbortError handling', () => {
        it('should silently ignore AbortError without calling errorDisplay or redirect', async () => {
            const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const abortError = new DOMException('The operation was aborted.', 'AbortError');
            vi.mocked(fetchClient.get).mockRejectedValue(abortError);

            await router.navigateTo('http://localhost/aborted');

            expect(consoleSpy).toHaveBeenCalledWith('Fetch aborted:', 'http://localhost/aborted');
            expect(errorDisplay.show).not.toHaveBeenCalled();
            expect(windowOps.setLocationHref).not.toHaveBeenCalled();

            consoleSpy.mockRestore();
        });
    });

    describe('dirty form cancellation', () => {
        it('should cancel navigation when hasDirtyForms is true and user declines', async () => {
            const formStateManager = createMockFormStateManager({
                hasDirtyForms: vi.fn(() => true),
                confirmNavigation: vi.fn(() => false),
            });

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                formStateManager,
            });

            await router.navigateTo('http://localhost/leave');

            expect(fetchClient.get).not.toHaveBeenCalled();
            expect(loader.show).not.toHaveBeenCalled();
        });

        it('should proceed when hasDirtyForms is true but user confirms', async () => {
            const formStateManager = createMockFormStateManager({
                hasDirtyForms: vi.fn(() => true),
                confirmNavigation: vi.fn(() => true),
            });

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                formStateManager,
            });

            await router.navigateTo('http://localhost/leave');

            expect(fetchClient.get).toHaveBeenCalled();
        });

        it('should never cancel popstate navigation even with dirty forms', async () => {
            const formStateManager = createMockFormStateManager({
                hasDirtyForms: vi.fn(() => true),
                confirmNavigation: vi.fn(() => false),
            });

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                formStateManager,
            });

            await router.navigateTo('http://localhost/back', undefined, {isPopState: true});

            expect(fetchClient.get).toHaveBeenCalled();
        });

        it('should proceed when no formStateManager is provided', async () => {
            await router.navigateTo('http://localhost/page');

            expect(fetchClient.get).toHaveBeenCalled();
        });

        it('should call untrackAll when user confirms navigation with dirty forms', async () => {
            const formStateManager = createMockFormStateManager({
                hasDirtyForms: vi.fn(() => true),
                confirmNavigation: vi.fn(() => true),
            });

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                formStateManager,
            });

            await router.navigateTo('http://localhost/leave');

            expect(formStateManager.confirmNavigation).toHaveBeenCalled();
            expect(formStateManager.untrackAll).toHaveBeenCalled();
        });

        it('should NOT call untrackAll when user cancels navigation', async () => {
            const formStateManager = createMockFormStateManager({
                hasDirtyForms: vi.fn(() => true),
                confirmNavigation: vi.fn(() => false),
            });

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                formStateManager,
            });

            await router.navigateTo('http://localhost/leave');

            expect(formStateManager.confirmNavigation).toHaveBeenCalled();
            expect(formStateManager.untrackAll).not.toHaveBeenCalled();
        });

        it('should call untrackAll before scanAndTrackForms on successful navigation', async () => {
            const callOrder: string[] = [];
            const formStateManager = createMockFormStateManager({
                hasDirtyForms: vi.fn(() => true),
                confirmNavigation: vi.fn(() => true),
                untrackAll: vi.fn(() => callOrder.push('untrackAll')),
                scanAndTrackForms: vi.fn(() => callOrder.push('scanAndTrackForms')),
            });

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                formStateManager,
            });

            await router.navigateTo('http://localhost/leave');

            expect(callOrder).toEqual(['untrackAll', 'scanAndTrackForms']);
        });

        it('should call untrackAll for popstate navigation', async () => {
            const formStateManager = createMockFormStateManager({
                hasDirtyForms: vi.fn(() => true),
                confirmNavigation: vi.fn(() => false),
            });

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                formStateManager,
            });

            await router.navigateTo('http://localhost/back', undefined, {isPopState: true});

            expect(formStateManager.hasDirtyForms).not.toHaveBeenCalled();
            expect(formStateManager.untrackAll).toHaveBeenCalled();
        });
    });

    describe('concurrent navigations', () => {
        it('should abort the first navigation when a second starts', async () => {
            let resolveFirst: (val: FetchResult) => void;
            const firstPromise = new Promise<FetchResult>((resolve) => {
                resolveFirst = resolve;
            });

            vi.mocked(fetchClient.get)
                .mockReturnValueOnce(firstPromise)
                .mockResolvedValueOnce([true, VALID_HTML]);

            const first = router.navigateTo('http://localhost/first');
            const second = router.navigateTo('http://localhost/second');

            expect(fetchClient.abort).toHaveBeenCalledTimes(1);

            resolveFirst!([true, VALID_HTML]);
            await first;
            await second;
        });
    });

    describe('destroy()', () => {
        it('should remove the popstate listener', () => {
            const addCall = vi.mocked(windowOps.addEventListener).mock.calls.find(
                (c) => c[0] === 'popstate'
            );
            const handler = addCall![1];

            router.destroy();

            expect(windowOps.removeEventListener).toHaveBeenCalledWith('popstate', handler);
        });
    });

    describe('popstate handler', () => {
        it('should navigate with replaceHistory and isPopState on popstate', async () => {
            vi.mocked(windowOps.getHistoryState).mockReturnValue({scrollY: 300});
            vi.mocked(windowOps.getLocationHref).mockReturnValue('http://localhost/popped');

            const addCall = vi.mocked(windowOps.addEventListener).mock.calls.find(
                (c) => c[0] === 'popstate'
            );
            const handler = addCall![1] as () => void;

            handler();

            await vi.waitFor(() => {
                expect(fetchClient.get).toHaveBeenCalled();
            });
        });

        it('should reload when popstate fires on a different domain', () => {
            vi.mocked(windowOps.getLocation).mockReturnValue({
                hostname: 'other-domain.com',
            } as Location);

            const addCall = vi.mocked(windowOps.addEventListener).mock.calls.find(
                (c) => c[0] === 'popstate'
            );
            const handler = addCall![1] as () => void;

            handler();

            expect(windowOps.locationReload).toHaveBeenCalled();
            expect(fetchClient.get).not.toHaveBeenCalled();
        });

        it('should pass restoreScrollY from history state to navigation', async () => {
            vi.mocked(windowOps.getHistoryState).mockReturnValue({scrollY: 500});
            vi.mocked(windowOps.getLocationHref).mockReturnValue('http://localhost/restored');

            const addCall = vi.mocked(windowOps.addEventListener).mock.calls.find(
                (c) => c[0] === 'popstate'
            );
            const handler = addCall![1] as () => void;

            handler();

            await vi.waitFor(() => {
                expect(onPageLoad).toHaveBeenCalled();
            });

            const scrollOptions = vi.mocked(onPageLoad).mock.calls[0][2] as PageLoadScrollOptions;
            expect(scrollOptions.restoreScrollY).toBe(500);
        });

        it('should handle null history state gracefully', async () => {
            vi.mocked(windowOps.getHistoryState).mockReturnValue(null);
            vi.mocked(windowOps.getLocationHref).mockReturnValue('http://localhost/no-state');

            const addCall = vi.mocked(windowOps.addEventListener).mock.calls.find(
                (c) => c[0] === 'popstate'
            );
            const handler = addCall![1] as () => void;

            handler();

            await vi.waitFor(() => {
                expect(onPageLoad).toHaveBeenCalled();
            });

            const scrollOptions = vi.mocked(onPageLoad).mock.calls[0][2] as PageLoadScrollOptions;
            expect(scrollOptions.restoreScrollY).toBeUndefined();
        });
    });

    describe('setConfig()', () => {
        it('should invoke global beforeNavigate callback', async () => {
            const beforeNavigate = vi.fn();
            router.setConfig({beforeNavigate});

            await router.navigateTo('http://localhost/page');

            expect(beforeNavigate).toHaveBeenCalledWith('http://localhost/page');
        });

        it('should invoke global afterNavigate callback', async () => {
            const afterNavigate = vi.fn();
            router.setConfig({afterNavigate});

            await router.navigateTo('http://localhost/page');

            expect(afterNavigate).toHaveBeenCalledWith('http://localhost/page');
        });

        it('should use per-navigation callbacks over global ones', async () => {
            const globalBefore = vi.fn();
            const localBefore = vi.fn();
            router.setConfig({beforeNavigate: globalBefore});

            await router.navigateTo('http://localhost/page', undefined, {
                beforeNavigate: localBefore,
            });

            expect(localBefore).toHaveBeenCalledWith('http://localhost/page');
            expect(globalBefore).not.toHaveBeenCalled();
        });

        it('should catch and log errors from navigation callbacks', async () => {
            const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            router.setConfig({
                beforeNavigate: () => { throw new Error('callback boom'); },
            });

            await router.navigateTo('http://localhost/page');

            expect(consoleSpy).toHaveBeenCalledWith(
                '[Router] Error in navigation callback:',
                expect.objectContaining({error: expect.any(Error)})
            );

            consoleSpy.mockRestore();
        });
    });

    describe('optional dependencies', () => {
        it('should emit hook events when hookManager is provided', async () => {
            const hookManager = createMockHookManager();

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                hookManager,
            });

            await router.navigateTo('http://localhost/page');

            expect(hookManager.emit).toHaveBeenCalledWith('navigation:start', expect.any(Object));
            expect(hookManager.emit).toHaveBeenCalledWith('navigation:complete', expect.any(Object));
            expect(hookManager.emit).toHaveBeenCalledWith('page:view', expect.any(Object));
        });

        it('should announce loading and navigation when a11yAnnouncer is provided', async () => {
            const a11yAnnouncer = createMockA11yAnnouncer();

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                a11yAnnouncer,
            });

            await router.navigateTo('http://localhost/page');

            expect(a11yAnnouncer.announceLoading).toHaveBeenCalled();
            expect(a11yAnnouncer.announceNavigation).toHaveBeenCalled();
        });

        it('should announce error when navigation fails and a11yAnnouncer is provided', async () => {
            const a11yAnnouncer = createMockA11yAnnouncer();
            vi.mocked(fetchClient.get).mockResolvedValue([false, null]);

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                a11yAnnouncer,
            });

            await router.navigateTo('http://localhost/fail');

            expect(a11yAnnouncer.announceError).toHaveBeenCalledWith('Navigation failed');
        });

        it('should emit NAVIGATION_ERROR hook when fetch fails', async () => {
            const hookManager = createMockHookManager();
            vi.mocked(fetchClient.get).mockResolvedValue([false, null]);

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                hookManager,
            });

            await router.navigateTo('http://localhost/fail');

            expect(hookManager.emit).toHaveBeenCalledWith('navigation:error', expect.objectContaining({
                url: 'http://localhost/fail',
                error: 'Fetch failed',
            }));
        });

        it('should call formStateManager.scanAndTrackForms after successful navigation', async () => {
            const formStateManager = createMockFormStateManager();

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                formStateManager,
            });

            await router.navigateTo('http://localhost/page');

            expect(formStateManager.scanAndTrackForms).toHaveBeenCalled();
        });
    });

    describe('focusMainContent', () => {
        it('should focus the main content element after navigation', async () => {
            const mainEl = document.createElement('main');
            const focusSpy = vi.fn();
            mainEl.focus = focusSpy;
            mainEl.setAttribute = vi.fn();
            mainEl.hasAttribute = vi.fn(() => false) as unknown as typeof mainEl.hasAttribute;
            mainEl.getAttribute = vi.fn(() => null);
            mainEl.removeAttribute = vi.fn();

            vi.mocked(domOps.querySelector).mockReturnValue(mainEl);

            await router.navigateTo('http://localhost/page');

            expect(mainEl.setAttribute).toHaveBeenCalledWith('tabindex', '-1');
            expect(focusSpy).toHaveBeenCalledWith({preventScroll: true});
            expect(mainEl.removeAttribute).toHaveBeenCalledWith('tabindex');
        });

        it('should restore original tabindex if element had one', async () => {
            const mainEl = document.createElement('main');
            const focusSpy = vi.fn();
            mainEl.focus = focusSpy;
            mainEl.setAttribute = vi.fn();
            mainEl.hasAttribute = vi.fn(() => true) as unknown as typeof mainEl.hasAttribute;
            mainEl.getAttribute = vi.fn(() => '0');
            mainEl.removeAttribute = vi.fn();

            vi.mocked(domOps.querySelector).mockReturnValue(mainEl);

            await router.navigateTo('http://localhost/page');

            expect(mainEl.setAttribute).toHaveBeenCalledWith('tabindex', '0');
        });
    });

    describe('loader progress', () => {
        it('should pass onProgress to fetchClient that updates the loader', async () => {
            vi.mocked(fetchClient.get).mockImplementation(async (_url: string, options?: FetchClientOptions) => {
                options?.onProgress?.(50, 100);
                return [true, VALID_HTML];
            });

            await router.navigateTo('http://localhost/page');

            expect(loader.setProgress).toHaveBeenCalledWith(50);
        });
    });

    describe('handleNavigationError', () => {
        it('should use error.message for navigation error emit', async () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const hookManager = createMockHookManager();

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                hookManager,
            });

            vi.mocked(fetchClient.get).mockRejectedValue(new Error('Custom error message'));

            await router.navigateTo('http://localhost/error-page');

            expect(hookManager.emit).toHaveBeenCalledWith('navigation:error', expect.objectContaining({
                error: 'Custom error message',
            }));

            consoleSpy.mockRestore();
        });

        it('should use "Unknown error" for non-Error thrown values', async () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const hookManager = createMockHookManager();

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                hookManager,
            });

            vi.mocked(fetchClient.get).mockRejectedValue('string error');

            await router.navigateTo('http://localhost/error-page');

            expect(hookManager.emit).toHaveBeenCalledWith('navigation:error', expect.objectContaining({
                error: 'Unknown error',
            }));

            consoleSpy.mockRestore();
        });
    });

    describe('finally block behaviour', () => {
        it('should hide loader and call afterNavigate even when fetch fails', async () => {
            const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
            const afterNavigate = vi.fn();
            vi.mocked(fetchClient.get).mockRejectedValue(new Error('boom'));

            await router.navigateTo('http://localhost/fail', undefined, {afterNavigate});

            expect(loader.hide).toHaveBeenCalled();
            expect(afterNavigate).toHaveBeenCalledWith('http://localhost/fail');
            expect(router.isNavigating()).toBe(false);

            consoleSpy.mockRestore();
        });

        it('should hide loader and set navigating false even on AbortError', async () => {
            vi.spyOn(console, 'warn').mockImplementation(() => {});
            vi.mocked(fetchClient.get).mockRejectedValue(
                new DOMException('Aborted', 'AbortError')
            );

            await router.navigateTo('http://localhost/aborted');

            expect(loader.hide).toHaveBeenCalled();
            expect(router.isNavigating()).toBe(false);
        });
    });

    describe('no #app in response', () => {
        it('should use custom display message for missing #app', async () => {
            vi.mocked(fetchClient.get).mockResolvedValue([true, NO_APP_HTML]);
            const hookManager = createMockHookManager();

            router.destroy();
            router = createRouter({
                fetchClient,
                loader,
                errorDisplay,
                eventBus,
                onPageLoad,
                windowOps,
                domOps,
                hookManager,
            });

            await router.navigateTo('http://localhost/missing-app');

            expect(errorDisplay.show).toHaveBeenCalledWith('No #app in fragment. Loading full page...');
            expect(hookManager.emit).toHaveBeenCalledWith('navigation:error', expect.objectContaining({
                error: 'No #app in response',
            }));
        });
    });
});
