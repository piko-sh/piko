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

import { describe, it, expect, beforeEach, afterEach, vi, Mock } from 'vitest';
import { PPFramework, RegisterHelper, getGlobalHelperRegistry } from './PPFramework';
import { addPageTokenScript } from '../../test/pageTokenScript';
import type { PPFrameworkOptions } from './PPFramework';
import * as actionModule from '@/pk/action';
import * as ActionExecutor from '@/core/ActionExecutor';
import { _createPagePKContext } from '@/pk/context';
import { _currentPageVisit, _endPageVisit } from '@/pk/lifecycle';
import { reloadPartial } from '@/pk/coordination';
import { getGlobalPageContext } from '@/services/PageContext';

const mockFetch = vi.fn();
global.fetch = mockFetch;

const mockObserve = vi.fn();
const mockUnobserve = vi.fn();
const mockDisconnect = vi.fn();

class MockIntersectionObserver {
  readonly root: Element | Document | null = null;
  readonly rootMargin: string = '';
  readonly thresholds: ReadonlyArray<number> = [];

  constructor(_callback: IntersectionObserverCallback, _options?: IntersectionObserverInit) {
  }

  observe = mockObserve;
  unobserve = mockUnobserve;
  disconnect = mockDisconnect;
  takeRecords = vi.fn(() => [] as IntersectionObserverEntry[]);
}

global.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver;


describe('PPFramework', () => {
  let originalLocation: Location;
  let originalHistory: History;
  let originalDocumentQuerySelector: typeof document.querySelector;
  let originalDocumentTitle: string;
  let appRoot: HTMLDivElement;

  beforeEach(() => {
    originalLocation = { ...window.location };
    originalHistory = { ...window.history };
    originalDocumentQuerySelector = document.querySelector;
    originalDocumentTitle = document.title;

    PPFramework.loadedModuleScripts.clear();
    PPFramework.globalConfig = {};

    vi.stubGlobal('location', {
      href: 'http://localhost:3000/',
      origin: 'http://localhost:3000',
      pathname: '/',
      search: '',
      hash: '',
      assign: vi.fn((url: string | URL) => {
        const newUrl = new URL(url.toString(), 'http://localhost:3000');
        (global.location as Location).href = newUrl.href;
        (global.location as Location).pathname = newUrl.pathname;
        (global.location as Location).search = newUrl.search;
      }),
      replace: vi.fn((url: string | URL) => {
        const newUrl = new URL(url.toString(), 'http://localhost:3000');
        (global.location as Location).href = newUrl.href;
        (global.location as Location).pathname = newUrl.pathname;
        (global.location as Location).search = newUrl.search;
      }),
      reload: vi.fn(),
      ancestorOrigins: {} as DOMStringList,
      protocol: 'http:',
      host: 'localhost:3000',
      hostname: 'localhost',
      port: '3000',
    });

    vi.stubGlobal('history', {
      pushState: vi.fn((_data: unknown, _unused: string, url?: string | URL | null) => {
        if (url) {
          const newUrl = new URL(url.toString(), window.location.origin);
          (window.location as Location).href = newUrl.href;
          (window.location as Location).pathname = newUrl.pathname;
          (window.location as Location).search = newUrl.search;
        }
      }),
      replaceState: vi.fn((_data: unknown, _unused: string, url?: string | URL | null) => {
        if (url) {
          const newUrl = new URL(url.toString(), window.location.origin);
          (window.location as Location).href = newUrl.href;
          (window.location as Location).pathname = newUrl.pathname;
          (window.location as Location).search = newUrl.search;
        }
      }),
      back: vi.fn(),
      forward: vi.fn(),
      go: vi.fn(),
      length: 0,
      scrollRestoration: 'auto',
      state: null,
    });

    document.querySelector = vi.fn((selector: string) => {
      if (selector === '#app') return appRoot;
      if (selector === 'title') {
        const titleEl = document.createElement('title');
        titleEl.textContent = document.title;
        return titleEl;
      }
      return originalDocumentQuerySelector.call(document, selector);
    }) as typeof document.querySelector;

    document.body.innerHTML = '';
    appRoot = document.createElement('div');
    appRoot.id = 'app';
    document.body.appendChild(appRoot);

    const loaderBar = document.getElementById('ppf-loader-bar') as HTMLDivElement | null;
    if (loaderBar) loaderBar.remove();
    const errorDisplay = document.getElementById('ppf-error-message') as HTMLDivElement | null;
    if (errorDisplay) errorDisplay.remove();

    mockFetch.mockReset();
    mockObserve.mockClear();
    mockUnobserve.mockClear();
    mockDisconnect.mockClear();
  });

  afterEach(() => {
    vi.stubGlobal('location', originalLocation);
    vi.stubGlobal('history', originalHistory);
    document.querySelector = originalDocumentQuerySelector;
    document.title = originalDocumentTitle;
    document.body.innerHTML = '';
    vi.restoreAllMocks();
  });

  const mockSuccessfulFetch = (htmlContent: string, headers?: Record<string, string>) => {
    const contentBytes = new TextEncoder().encode(htmlContent);

    const mockDefaultReader: ReadableStreamDefaultReader<Uint8Array> = {
      closed: Promise.resolve(undefined),
      cancel: vi.fn((reason?: unknown) => Promise.resolve(reason as undefined)),
      read: vi.fn()
        .mockResolvedValueOnce({ done: false, value: contentBytes })
        .mockResolvedValueOnce({ done: true, value: undefined }),
      releaseLock: vi.fn(),
    };

    const mockBYOBReader: ReadableStreamBYOBReader = {
      closed: Promise.resolve(undefined),
      cancel: vi.fn((reason?: unknown) => Promise.resolve(reason as undefined)),
      read: vi.fn() as unknown as <T extends ArrayBufferView>(view: T) => Promise<ReadableStreamReadResult<T>>,
      releaseLock: vi.fn(),
    };

    const createMockStreamInstance = <S = Uint8Array>(): {
      locked: boolean;
      cancel: Mock<() => Promise<void>>;
      getReader: { (options: { mode: "byob" }): ReadableStreamBYOBReader; (): ReadableStreamDefaultReader<S> };
      pipeTo: Mock<() => Promise<void>>;
      pipeThrough: <T_PIPE_THROUGH>(transform: ReadableWritablePair<T_PIPE_THROUGH, S>, options?: StreamPipeOptions) => ReadableStream<T_PIPE_THROUGH>;
      tee: Mock<() => [ReadableStream<S>, ReadableStream<S>]>;
      [Symbol.asyncIterator]: Mock<() => AsyncGenerator<S, void, unknown>>
    } => ({
      locked: false,
      cancel: vi.fn(() => Promise.resolve()),
      getReader: ((options?: { mode?: "byob" | undefined }) => {
        if (options?.mode === "byob") {
          return mockBYOBReader as unknown as ReadableStreamBYOBReader;
        }
        return mockDefaultReader as unknown as ReadableStreamDefaultReader<S>;
      }) as {
        (options: { mode: "byob"; }): ReadableStreamBYOBReader;
        (): ReadableStreamDefaultReader<S>;
      },
      pipeTo: vi.fn(() => Promise.resolve()),
      pipeThrough: vi.fn() as unknown as <T_PIPE_THROUGH>(
        transform: ReadableWritablePair<T_PIPE_THROUGH, S>,
        options?: StreamPipeOptions
      ) => ReadableStream<T_PIPE_THROUGH>,
      tee: vi.fn((): [ReadableStream<S>, ReadableStream<S>] => {
        return [createMockStreamInstance<S>(), createMockStreamInstance<S>()];
      }),
      [Symbol.asyncIterator]: vi.fn(async function*() {
        if (mockDefaultReader && typeof (mockDefaultReader as ReadableStreamDefaultReader<Uint8Array>).read === 'function') {
          const freshMockDefaultReader: ReadableStreamDefaultReader<Uint8Array> = {
            ...mockDefaultReader,
            read: vi.fn()
              .mockResolvedValueOnce({ done: false, value: contentBytes })
              .mockResolvedValueOnce({ done: true, value: undefined }),
          };
          let chunkResult = await (freshMockDefaultReader as ReadableStreamDefaultReader<S>).read();
          while (chunkResult && !chunkResult.done) {
            yield chunkResult.value;
            chunkResult = await (freshMockDefaultReader as ReadableStreamDefaultReader<S>).read();
          }
        }
      }),
    });

    const mainMockStreamUint8 = createMockStreamInstance<Uint8Array>();

    mockFetch.mockResolvedValueOnce({
      ok: true,
      text: () => Promise.resolve(htmlContent),
      headers: new Headers(headers ?? { 'Content-Length': String(contentBytes.length) }),
      body: mainMockStreamUint8,
    });
  };

  const mockFailedFetch = (status = 404) => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status,
      statusText: `HTTP Error ${status}`,
      text: () => Promise.resolve('Error'),
      headers: new Headers(),
      body: null,
    });
  };

  describe('init()', () => {
    it('does not end the current page visit when the framework starts', () => {
      const visitBefore = _currentPageVisit();

      PPFramework.init();

      expect(_currentPageVisit()).toBe(visitBefore);
    });

    it('loads the page scripts of the document through the page context', () => {
      const meta = document.createElement('meta');
      meta.setAttribute('name', 'pk-script');
      meta.setAttribute('content', '/_piko/assets/pk-js/pages/main.js');
      document.head.appendChild(meta);
      const loadModule = vi.spyOn(getGlobalPageContext(), 'loadModule').mockResolvedValue();

      PPFramework.init();
      meta.remove();

      expect(loadModule).toHaveBeenCalledWith('/_piko/assets/pk-js/pages/main.js', undefined);
      loadModule.mockRestore();
    });

    it('should set globalConfig', () => {
      const options: PPFrameworkOptions = { loaderColour: 'red' };
      PPFramework.init(options);
      expect(PPFramework.globalConfig).toEqual(options);
    });

    it('should create loader indicator', () => {
      PPFramework.init({ loaderColour: 'blue' });
      const loader = document.getElementById('ppf-loader-bar');
      expect(loader).not.toBeNull();
      expect(loader!.style.background).toBe('blue');
    });

    it('should add initial module scripts to loadedModuleScripts', () => {
      const script1 = document.createElement('script');
      script1.type = 'module';
      script1.src = '/assets/main.js';
      document.body.appendChild(script1);
      const script2 = document.createElement('script');
      script2.type = 'module';
      script2.src = '/assets/vendor.js';
      document.body.appendChild(script2);

      PPFramework.init();
      expect(PPFramework.loadedModuleScripts.has('/assets/main.js')).toBe(true);
      expect(PPFramework.loadedModuleScripts.has('/assets/vendor.js')).toBe(true);
    });

    it('should add popstate event listener', () => {
      const addEventListenerSpy = vi.spyOn(window, 'addEventListener');
      PPFramework.init();
      expect(addEventListenerSpy).toHaveBeenCalledWith('popstate', expect.any(Function));
    });
  });

  describe('navigateTo()', () => {
    beforeEach(() => {
      PPFramework.init();
      appRoot = document.getElementById('app') as HTMLDivElement;
      if (!appRoot) {
        appRoot = document.createElement('div');
        appRoot.id = 'app';
        document.body.appendChild(appRoot);
      }
      appRoot.innerHTML = '<p>Old Content</p>';
      document.title = 'Old Title';
      (history.pushState as Mock).mockClear();
      (history.replaceState as Mock).mockClear();
      (window.location.assign as Mock).mockClear();
    });

    it('should fetch HTML fragment and update #app and title', async () => {
      const newPageHtml = '<html><head><title>New Title</title></head><body><div id="app"><p>New Content</p></div></body></html>';
      mockSuccessfulFetch(newPageHtml);

      await PPFramework.navigateTo('/new-page');

      expect(mockFetch).toHaveBeenCalledWith('http://localhost:3000/new-page?_f=1', expect.anything());
      expect(appRoot.innerHTML).toBe('<p>New Content</p>');
      expect(document.title).toBe('New Title');
    });

    it('disconnects the page script of the page being left once, while its elements are still in the DOM', async () => {
      appRoot.innerHTML = '<div partial_name="layout"><button id="leaving-button">Leaving</button></div>';
      const seenOnDisconnect: Array<string | null> = [];
      _createPagePKContext(document.body).onDisconnected(() => {
        seenOnDisconnect.push(document.getElementById('leaving-button')?.textContent ?? null);
      });
      const visitBefore = _currentPageVisit();
      mockSuccessfulFetch('<html><body><div id="app"><div partial_name="layout"><p>Arrived</p></div></div></body></html>');
      await PPFramework.navigateTo('/next-page');
      mockSuccessfulFetch('<html><body><div id="app"><div partial_name="layout"><p>Arrived again</p></div></div></body></html>');

      await PPFramework.navigateTo('/another-page');

      expect(seenOnDisconnect).toEqual(['Leaving']);
      expect(_currentPageVisit()).not.toBe(visitBefore);
      expect(appRoot.textContent).toContain('Arrived again');
    });

    it('gives the page style and widget scripts it inserts the page CSP token', async () => {
      const tokenScript = addPageTokenScript('page-token');
      mockSuccessfulFetch('<html><head><style pk-page>h1 { color: green; }</style><meta name="pk-widget-script" content="/_piko/captcha/token-widget.js"></head><body><div id="app"><h1>Next</h1></div></body></html>');

      await PPFramework.navigateTo('/token-page');

      expect(document.head.querySelector<HTMLStyleElement>('style[pk-page]')?.nonce).toBe('page-token');
      expect(document.head.querySelector<HTMLScriptElement>('script[src="/_piko/captcha/token-widget.js"]')?.nonce).toBe('page-token');
      document.head.querySelectorAll('style[pk-page], script[src="/_piko/captcha/token-widget.js"]').forEach(element => element.remove());
      tokenScript.remove();
    });

    it('gives content swapped in without morphing the page token in place of the fragment token', async () => {
      const tokenScript = addPageTokenScript('page-token');
      const fragmentToken = 'token-of-the-fragment-response';
      mockSuccessfulFetch(`<html><body><div id="app"><style nonce="${fragmentToken}">h1{}</style><h1>Next</h1></div></body></html>`);

      await PPFramework.navigateTo('/unmorphed-page', undefined, { morph: 'none' });

      const style = appRoot.querySelector<HTMLStyleElement>('style');
      expect(appRoot.querySelector('h1')?.textContent).toBe('Next');
      expect(style?.nonce).toBe('page-token');
      expect(style?.getAttribute('nonce')).not.toBe(fragmentToken);
      tokenScript.remove();
    });

    it('loads the page scripts of the page navigated to in the new page visit', async () => {
      const visitBefore = _currentPageVisit();
      const visitsAtLoad: object[] = [];
      const loadModule = vi.spyOn(getGlobalPageContext(), 'loadModule').mockImplementation(() => {
        visitsAtLoad.push(_currentPageVisit());
        return Promise.resolve();
      });
      mockSuccessfulFetch('<html><head><meta name="pk-script" content="/_piko/assets/pk-js/pages/next.js"></head><body><div id="app"><p>Next</p></div></body></html>');

      await PPFramework.navigateTo('/next-page');

      expect(loadModule).toHaveBeenCalledWith('/_piko/assets/pk-js/pages/next.js', undefined);
      expect(visitsAtLoad).toHaveLength(1);
      expect(visitsAtLoad[0]).not.toBe(visitBefore);
      expect(visitsAtLoad[0]).toBe(_currentPageVisit());
      loadModule.mockRestore();
    });

    it('copies the page ID of the page navigated to onto the body', async () => {
      document.body.setAttribute('data-pageid', 'pages/old');
      mockSuccessfulFetch('<html><body><div id="app" data-pageid="pages/new"><p>New</p></div></body></html>');

      await PPFramework.navigateTo('/new-page');

      expect(document.body.getAttribute('data-pageid')).toBe('pages/new');
    });

    it('removes the page ID from the body when the page navigated to has none', async () => {
      document.body.setAttribute('data-pageid', 'pages/old');
      mockSuccessfulFetch('<html><body><div id="app"><p>New</p></div></body></html>');

      await PPFramework.navigateTo('/new-page');

      expect(document.body.hasAttribute('data-pageid')).toBe(false);
    });

    it('should update history (pushState by default) with the original targetUrl', async () => {
      mockSuccessfulFetch('<html><body><div id="app"></div></body></html>');

      await PPFramework.navigateTo('/history-test');
      expect(history.replaceState).toHaveBeenCalledWith({scrollY: 0}, "", 'http://localhost:3000/');
      expect(history.pushState).toHaveBeenCalledWith({scrollY: 0}, "", '/history-test');
    });

    it('should use replaceState if options.replaceHistory is true, with original targetUrl', async () => {
      mockSuccessfulFetch('<html><body><div id="app"></div></body></html>');

      await PPFramework.navigateTo('/replace-history', undefined, { replaceHistory: true });
      expect(history.replaceState).toHaveBeenNthCalledWith(1, {scrollY: 0}, "", 'http://localhost:3000/');
      expect(history.replaceState).toHaveBeenNthCalledWith(2, {scrollY: 0}, "", '/replace-history');
    });

    it('should not update history if options.isPopState is true', async () => {
      mockSuccessfulFetch('<html><body><div id="app"></div></body></html>');

      await PPFramework.navigateTo('/pop-nav', undefined, { isPopState: true });
      expect(history.pushState).not.toHaveBeenCalled();
      expect(history.replaceState).not.toHaveBeenCalled();
    });

    it('should call beforeNavigate and afterNavigate callbacks with the original targetUrl', async () => {
      mockSuccessfulFetch('<html><body><div id="app"></div></body></html>');
      const beforeNavCb = vi.fn();
      const afterNavCb = vi.fn();

      await PPFramework.navigateTo('/callbacks', undefined, { beforeNavigate: beforeNavCb, afterNavigate: afterNavCb });

      expect(beforeNavCb).toHaveBeenCalledWith('/callbacks');
      expect(afterNavCb).toHaveBeenCalledWith('/callbacks');
    });

    it('should handle fetch failure by redirecting, using original targetUrl in message and redirect', async () => {
      mockFailedFetch();

      await PPFramework.navigateTo('/fetch-fail');

      const errorEl = document.getElementById('ppf-error-message');
      expect(errorEl).not.toBeNull();
      expect(errorEl?.textContent).toBe('Navigation to /fetch-fail failed. Loading full page...');
      expect(window.location.href).toBe('/fetch-fail');
    });

    it('should handle missing #app in fragment by redirecting, using original targetUrl', async () => {
      mockSuccessfulFetch('<html><body><div>No App Root Here</div></body></html>');

      await PPFramework.navigateTo('/no-app-root');

      const errorEl = document.getElementById('ppf-error-message');
      expect(errorEl).not.toBeNull();
      expect(errorEl?.textContent).toBe('No #app in fragment. Loading full page...');
      expect(window.location.href).toBe('/no-app-root');
    });

    it('should abort previous navigation if a new one starts', async () => {
      PPFramework.init();

      const slowFetch = new Promise<Response>((resolve) => {
        setTimeout(() => {
          resolve({
            ok: true,
            text: () => Promise.resolve('<html><body><div id="app">Page 1</div></body></html>'),
            headers: new Headers(),
            body: null,
          } as Response);
        }, 1000);
      });

      mockFetch.mockReturnValueOnce(slowFetch);

      void PPFramework.navigateTo('/page1');

      const controller = PPFramework.currentAbortController;

      mockSuccessfulFetch('<html><body><div id="app">Page 2</div></body></html>');
      await PPFramework.navigateTo('/page2');

      expect(controller?.signal.aborted).toBe(true);
    });
  });

  describe('navigation View Transitions', () => {
    let startViewTransition: Mock<(update: () => void) => {updateCallbackDone: Promise<void>; ready: Promise<void>; finished: Promise<void>; skipTransition: () => void}>;
    let viewTransitionsMeta: HTMLMetaElement | null;

    beforeEach(() => {
      viewTransitionsMeta = null;
      startViewTransition = vi.fn((update: () => void) => {
        update();
        return {updateCallbackDone: Promise.resolve(), ready: Promise.resolve(), finished: Promise.resolve(), skipTransition: vi.fn()};
      });
      Object.defineProperty(document, 'startViewTransition', {value: startViewTransition, configurable: true, writable: true});
      vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
    });

    afterEach(() => {
      viewTransitionsMeta?.remove();
      Reflect.deleteProperty(document, 'startViewTransition');
      Reflect.deleteProperty(document, 'elementFromPoint');
    });

    const turnOffViewTransitions = (): void => {
      viewTransitionsMeta = document.createElement('meta');
      viewTransitionsMeta.setAttribute('name', 'pk-view-transitions');
      viewTransitionsMeta.setAttribute('content', 'off');
      document.head.appendChild(viewTransitionsMeta);
    };

    it('runs the page update in a View Transition by default', async () => {
      PPFramework.init();
      mockSuccessfulFetch('<html><body><div id="app"><p>Animated</p></div></body></html>');

      await PPFramework.navigateTo('/animated-page');

      expect(startViewTransition).toHaveBeenCalledTimes(1);
      expect(appRoot.innerHTML).toBe('<p>Animated</p>');
    });

    it('updates the page without a View Transition when the server turned them off', async () => {
      turnOffViewTransitions();
      PPFramework.init();
      mockSuccessfulFetch('<html><body><div id="app"><p>Plain</p></div></body></html>');

      await PPFramework.navigateTo('/plain-page');

      expect(startViewTransition).not.toHaveBeenCalled();
      expect(appRoot.innerHTML).toBe('<p>Plain</p>');
    });

    it('reads the setting once when the framework starts', async () => {
      PPFramework.init();
      turnOffViewTransitions();
      mockSuccessfulFetch('<html><body><div id="app"><p>Late</p></div></body></html>');

      await PPFramework.navigateTo('/late-page');

      expect(startViewTransition).toHaveBeenCalledTimes(1);
      expect(appRoot.innerHTML).toBe('<p>Late</p>');
    });

    it('passes a click lost to the transition on to the new page', async () => {
      let endTransition: () => void = () => undefined;
      const skipTransition = vi.fn();
      startViewTransition.mockImplementationOnce((update: () => void) => {
        update();
        return {updateCallbackDone: Promise.resolve(), ready: Promise.resolve(), finished: new Promise<void>(resolve => { endTransition = resolve; }), skipTransition};
      });
      PPFramework.init();
      mockSuccessfulFetch('<html><body><div id="app"><button id="next-button">Next</button></div></body></html>');
      await PPFramework.navigateTo('/button-page');
      const button = appRoot.querySelector('#next-button') as HTMLButtonElement;
      const buttonClicks = vi.fn();
      button.addEventListener('click', buttonClicks);
      Object.defineProperty(document, 'elementFromPoint', {value: () => button, configurable: true, writable: true});

      document.documentElement.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true, detail: 1, clientX: 5, clientY: 5}));
      endTransition();
      await vi.waitFor(() => expect(buttonClicks).toHaveBeenCalledTimes(1));

      expect(skipTransition).toHaveBeenCalled();
    });

    it('leaves no unhandled rejection when the transition is skipped', async () => {
      const unhandledReasons: unknown[] = [];
      const recordUnhandled = (reason: unknown): void => {
        unhandledReasons.push(reason);
      };
      process.on('unhandledRejection', recordUnhandled);
      startViewTransition.mockImplementationOnce((update: () => void) => {
        update();
        const skipped = new DOMException('Transition was skipped', 'AbortError');
        return {updateCallbackDone: Promise.resolve(), ready: Promise.reject(skipped), finished: Promise.reject(skipped), skipTransition: vi.fn()};
      });
      PPFramework.init();
      mockSuccessfulFetch('<html><body><div id="app"><p>Skipped</p></div></body></html>');

      await PPFramework.navigateTo('/skipped-page');
      await new Promise(resolve => setTimeout(resolve, 20));
      process.off('unhandledRejection', recordUnhandled);

      expect(unhandledReasons).toEqual([]);
      expect(appRoot.innerHTML).toBe('<p>Skipped</p>');
    });
  });

  describe('remoteRender()', () => {
    let patchLocationDiv: HTMLDivElement;

    beforeEach(() => {
      PPFramework.init();
      patchLocationDiv = document.createElement('div');
      document.body.appendChild(patchLocationDiv);
    });

    afterEach(() => {
      if (patchLocationDiv?.parentNode) patchLocationDiv.parentNode.removeChild(patchLocationDiv);
    });

    it('should fetch HTML and patch specified location', async () => {
      const remoteHtml = '<html><head><meta name="X-PP-Response-Support" content="fragment-patch"></head><body><div id="app"><em>Remote Content</em><button>Click</button></div></body></html>';
      mockFetch.mockResolvedValueOnce({
        ok: true,
        text: () => Promise.resolve(remoteHtml),
        headers: new Headers({ 'X-PP-Response-Support': 'fragment-patch' }),
        body: null,
      });

      await PPFramework.remoteRender({ src: '/remote/path', patchLocation: patchLocationDiv });

      expect(mockFetch).toHaveBeenCalledWith('http://localhost:3000/remote/path?_f=1', expect.anything());
      expect(patchLocationDiv.innerHTML).toBe('<em>Remote Content</em><button>Click</button>');
    });

    it('should use querySelector if provided', async () => {
      const remoteHtml = '<html><body><div id="app"><header>Ignore</header><main id="target-section"><p>Target Content</p></main></div></body></html>';
      mockFetch.mockResolvedValueOnce({
        ok: true,
        text: () => Promise.resolve(remoteHtml),
        headers: new Headers({ 'X-PP-Response-Support': 'fragment-patch' }),
        body: null,
      });

      await PPFramework.remoteRender({
        src: '/remote/selector',
        querySelector: '#target-section',
        patchLocation: patchLocationDiv
      });
      expect(patchLocationDiv.innerHTML).toBe('<p>Target Content</p>');
    });

    it('should handle fetch failure for remoteRender gracefully', async () => {
      mockFailedFetch(500);
      const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      await PPFramework.remoteRender({ src: '/remote/fail', patchLocation: patchLocationDiv });

      expect(patchLocationDiv.innerHTML).toBe('');
      expect(consoleErrorSpy).toHaveBeenCalledWith("RemoteRenderer: fetch failed", 500, "HTTP Error 500");
      consoleErrorSpy.mockRestore();
    });
  });

  describe('Utility Functions', () => {
    it('buildRemoteUrl should construct URL correctly relative to current origin', () => {
      const url = PPFramework.buildRemoteUrl('/api/data', { id: 123, type: 'test' });
      expect(url).toBe('http://localhost:3000/api/data?_f=1&id=123&type=test');

      const url2 = PPFramework.buildRemoteUrl('http://otherserver.com/api/data', { id: 456 });
      expect(url2).toBe('http://otherserver.com/api/data?_f=1&id=456');
    });

    it('addFragmentQuery should add _f=1 parameter', () => {
      expect(PPFramework.addFragmentQuery('/page')).toBe('http://localhost:3000/page?_f=1');
      expect(PPFramework.addFragmentQuery('/page?query=abc')).toBe('http://localhost:3000/page?query=abc&_f=1');
      expect(PPFramework.addFragmentQuery('http://example.com/page?query=abc')).toBe('http://example.com/page?query=abc&_f=1');
    });

    it('isSameDomain should correctly identify same and different domains', () => {
      const sameDomainAnchor = document.createElement('a');
      sameDomainAnchor.href = 'http://localhost:3000/path';
      expect(PPFramework.isSameDomain(sameDomainAnchor)).toBe(true);

      const differentDomainAnchor = document.createElement('a');
      differentDomainAnchor.href = 'http://example.com/path';
      expect(PPFramework.isSameDomain(differentDomainAnchor)).toBe(false);

      const locLike: Partial<Location> = { hostname: 'localhost', protocol: 'http:', port: '3000' };
      expect(PPFramework.isSameDomain(locLike as Location)).toBe(true);
    });
  });

  describe('UI Feedback (Loader, Error)', () => {
    let loaderBarElement: HTMLDivElement | null;

    beforeEach(() => {
      PPFramework.init({ loaderColour: 'green' });
      loaderBarElement = document.getElementById('ppf-loader-bar') as HTMLDivElement;
    });

    it('toggleLoader should show and hide loader bar', async () => {
      expect(loaderBarElement!.style.display).toBe('none');
      PPFramework.toggleLoader(true);
      expect(loaderBarElement!.style.display).toBe('block');
      expect(loaderBarElement!.style.width).toBe('0%');
      PPFramework.toggleLoader(false);
      expect(loaderBarElement!.style.width).toBe('100%');
      await new Promise(r => setTimeout(r, 350));
      expect(loaderBarElement!.style.display).toBe('none');
      expect(loaderBarElement!.style.width).toBe('0%');
    });

    it('updateProgressBar should update loader bar width and stay monotonic', () => {
      PPFramework.updateProgressBar(50);
      expect(loaderBarElement!.style.display).toBe('block');
      expect(loaderBarElement!.style.width).toBe('50%');
      PPFramework.updateProgressBar(110);
      expect(loaderBarElement!.style.width).toBe('100%');
      PPFramework.updateProgressBar(-10);
      expect(loaderBarElement!.style.width).toBe('100%');
    });

    it('displayError should show and then hide an error message', async () => {
      vi.useFakeTimers();
      PPFramework.displayError('Test Error Message');
      let errorEl = document.getElementById('ppf-error-message');
      expect(errorEl).not.toBeNull();
      expect(errorEl!.textContent).toBe('Test Error Message');

      await vi.advanceTimersByTimeAsync(5100);
      errorEl = document.getElementById('ppf-error-message');
      expect(errorEl).toBeNull();
      vi.useRealTimers();
    });
  });

  describe('Script and Module Loading', () => {
    beforeEach(() => {
      PPFramework.init();
      PPFramework.loadedModuleScripts.clear();
      document.querySelectorAll('script[type="module"][src^="/assets/"]').forEach(s => s.remove());
    });

    it('loadModuleScripts should add new module scripts to body and loaded set', () => {
      const mockDoc = new DOMParser().parseFromString(
        '<script type="module" src="/test-script1.js"></script><script src="/non-module.js"></script>',
        'text/html'
      );
      PPFramework.loadModuleScripts(mockDoc);
      expect(PPFramework.loadedModuleScripts.has('/test-script1.js')).toBe(true);
      const SCRIPT1 = document.querySelector('script[src="/test-script1.js"]');
      expect(SCRIPT1).not.toBeNull();
      if (SCRIPT1?.parentNode) SCRIPT1.parentNode.removeChild(SCRIPT1);
      expect(document.querySelector('script[src="/non-module.js"]')).toBeNull();
    });

    it('loadModuleScripts should not re-add already loaded scripts', () => {
      PPFramework.loadedModuleScripts.add('/test-script2.js');
      const mockDoc = new DOMParser().parseFromString('<script type="module" src="/test-script2.js"></script>', 'text/html');
      const appendSpy = vi.spyOn(document.body, 'appendChild');

      PPFramework.loadModuleScripts(mockDoc);

      expect(appendSpy).not.toHaveBeenCalledWith(expect.objectContaining({ src: expect.stringContaining('/test-script2.js') }));
    });
  });

  describe('assetSrc / resolveAssetSrc', () => {
    it('should prepend /_piko/assets/ to a relative path', () => {
      expect(PPFramework.assetSrc('images/logo.png')).toBe('/_piko/assets/images/logo.png');
    });

    it('should return an absolute https URL unchanged', () => {
      expect(PPFramework.assetSrc('https://cdn.example.com/logo.png')).toBe('https://cdn.example.com/logo.png');
    });

    it('should return an absolute http URL unchanged', () => {
      expect(PPFramework.assetSrc('http://cdn.example.com/logo.png')).toBe('http://cdn.example.com/logo.png');
    });

    it('should return a data URI unchanged', () => {
      const dataUri = 'data:image/png;base64,iVBORw0KGgo=';
      expect(PPFramework.assetSrc(dataUri)).toBe(dataUri);
    });

    it('should return a root-relative path unchanged', () => {
      expect(PPFramework.assetSrc('/static/logo.png')).toBe('/static/logo.png');
    });

    it('should resolve @/ alias with module name', () => {
      expect(PPFramework.assetSrc('@/icons/star.svg', 'dashboard')).toBe('/_piko/assets/dashboard/icons/star.svg');
    });

    it('should treat @/ without a module name as a literal path', () => {
      expect(PPFramework.assetSrc('@/icons/star.svg')).toBe('/_piko/assets/@/icons/star.svg');
    });

    it('should return an empty string for empty input', () => {
      expect(PPFramework.assetSrc('')).toBe('');
    });
  });

  describe('getModuleConfig', () => {
    it('should parse JSON from a #pk-module-config script element and cache the result', () => {
      const configEl = document.createElement('script');
      configEl.id = 'pk-module-config';
      configEl.type = 'application/json';
      configEl.textContent = JSON.stringify({
        myModule: { apiUrl: '/api/v1' },
        cached: { value: 42 },
      });
      document.body.appendChild(configEl);

      const config = PPFramework.getModuleConfig<{ apiUrl: string }>('myModule');
      expect(config).toEqual({ apiUrl: '/api/v1' });

      configEl.remove();
      const second = PPFramework.getModuleConfig<{ value: number }>('cached');
      expect(second).toEqual({ value: 42 });
    });

    it('should return null for an unknown module name (from cache)', () => {
      expect(PPFramework.getModuleConfig('nonExistent')).toBeNull();
    });
  });

  describe('diagnostic hook', () => {
    beforeEach(() => {
      PPFramework.init();
    });

    it('fires for a reload of a partial that is not in the page', async () => {
      vi.spyOn(console, 'warn').mockImplementation(() => {});
      _endPageVisit();
      const listener = vi.fn();
      const unsubscribe = PPFramework.hooks.on(PPFramework.hooks.events.DIAGNOSTIC, listener);

      await reloadPartial('no-such-partial');
      unsubscribe();

      expect(listener).toHaveBeenCalledWith(expect.objectContaining({
        kind: 'partial-not-found',
        name: 'no-such-partial',
        source: 'piko.partials.reload'
      }));
    });

    it('fires for an action that is not registered', () => {
      vi.spyOn(console, 'error').mockImplementation(() => {});
      _endPageVisit();
      const listener = vi.fn();
      const unsubscribe = PPFramework.hooks.on(PPFramework.hooks.events.DIAGNOSTIC, listener);

      PPFramework.dispatchAction('missing.action', document.createElement('button'));
      unsubscribe();

      expect(listener).toHaveBeenCalledWith(expect.objectContaining({
        kind: 'action-not-found',
        name: 'missing.action'
      }));
    });
  });

  describe('dispatchAction', () => {
    it('should report an error when action function is not registered', () => {
      const warnSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      const button = document.createElement('button');
      PPFramework.dispatchAction('nonexistent.action', button);

      expect(warnSpy).toHaveBeenCalledWith(
        expect.stringContaining('Action function "nonexistent.action" not found')
      );
      warnSpy.mockRestore();
    });

    it('should call handleAction when action function is registered', () => {
      const mockActionFn = vi.fn().mockReturnValue({ action: 'test.action', args: [] });
      vi.spyOn(actionModule, 'getActionFunction').mockReturnValue(mockActionFn);
      const handleActionSpy = vi.spyOn(ActionExecutor, 'handleAction').mockResolvedValue(undefined);

      const button = document.createElement('button');
      PPFramework.dispatchAction('test.action', button);

      expect(mockActionFn).toHaveBeenCalled();
      expect(handleActionSpy).toHaveBeenCalled();

      handleActionSpy.mockRestore();
      vi.restoreAllMocks();
    });

    it('should call preventDefault for form elements', () => {
      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const form = document.createElement('form');
      const event = new Event('submit', { cancelable: true });
      const preventSpy = vi.spyOn(event, 'preventDefault');

      PPFramework.dispatchAction('some.action', form, event);

      expect(preventSpy).toHaveBeenCalled();
      warnSpy.mockRestore();
    });

    it('should call preventDefault for submit button elements', () => {
      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const button = document.createElement('button');
      button.type = 'submit';
      const event = new Event('click', { cancelable: true });
      const preventSpy = vi.spyOn(event, 'preventDefault');

      PPFramework.dispatchAction('some.action', button, event);

      expect(preventSpy).toHaveBeenCalled();
      warnSpy.mockRestore();
    });
  });

  describe('executeHelper', () => {
    it('should parse and execute a simple helper string', () => {
      const helperFn = vi.fn();
      const registry = getGlobalHelperRegistry();
      registry.register('testHelper', helperFn);

      const element = document.createElement('button');
      const event = new Event('click', { cancelable: true });

      PPFramework.executeHelper(event, 'testHelper', element);

      expect(helperFn).toHaveBeenCalledWith(element, event);
    });

    it('should parse and pass arguments from helper string', () => {
      const helperFn = vi.fn();
      const registry = getGlobalHelperRegistry();
      registry.register('showToast', helperFn);

      const element = document.createElement('button');
      const event = new Event('click', { cancelable: true });

      PPFramework.executeHelper(event, 'showToast(Hello, World)', element);

      expect(helperFn).toHaveBeenCalledWith(element, event, 'Hello', 'World');
    });

    it('should call preventDefault on the event', () => {
      const element = document.createElement('button');
      const event = new Event('click', { cancelable: true });
      const preventSpy = vi.spyOn(event, 'preventDefault');

      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      PPFramework.executeHelper(event, 'unknownHelper', element);

      expect(preventSpy).toHaveBeenCalled();
      warnSpy.mockRestore();
    });

    it('should warn when the action string cannot be parsed', () => {
      const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
      const element = document.createElement('button');
      const event = new Event('click', { cancelable: true });

      PPFramework.executeHelper(event, '', element);

      expect(warnSpy).toHaveBeenCalledWith(
        expect.stringContaining('Could not parse helper action string')
      );
      warnSpy.mockRestore();
    });
  });

  describe('RegisterHelper (exported function)', () => {
    it('should register a helper in the global registry', () => {
      const helperFn = vi.fn();
      RegisterHelper('myGlobalHelper', helperFn);

      const registry = getGlobalHelperRegistry();
      expect(registry.has('myGlobalHelper')).toBe(true);
      expect(registry.get('myGlobalHelper')).toBe(helperFn);
    });
  });

  describe('registerHelper (instance method)', () => {
    it('should register a helper via the framework instance', () => {
      const helperFn = vi.fn();
      PPFramework.registerHelper('instanceHelper', helperFn);

      const registry = getGlobalHelperRegistry();
      expect(registry.has('instanceHelper')).toBe(true);
    });
  });

  describe('calls made before init', () => {
    const MAX_CALLS_BEFORE_INIT = 50;
    const FRAGMENT_PATCH_HTML = '<html><head><meta name="X-PP-Response-Support" content="fragment-patch"></head><body><div id="app"><em>Queued Content</em></div></body></html>';

    async function importFreshFramework(): Promise<typeof PPFramework> {
      vi.resetModules();
      const freshModule = await import('./PPFramework');
      return freshModule.PPFramework;
    }

    function mockFragmentResponse(): void {
      mockFetch.mockResolvedValue({
        ok: true,
        text: () => Promise.resolve(FRAGMENT_PATCH_HTML),
        headers: new Headers({ 'X-PP-Response-Support': 'fragment-patch' }),
        body: null,
      });
    }

    it('loads the page in full when navigateTo is called before init', async () => {
      const freshFramework = await importFreshFramework();

      await freshFramework.navigateTo('/before-init');

      expect(window.location.assign).toHaveBeenCalledWith('/before-init');
      expect(window.location.replace).not.toHaveBeenCalled();
    });

    it('replaces the history entry when navigateTo asks for it before init', async () => {
      const freshFramework = await importFreshFramework();

      await freshFramework.navigateTo('/before-init-replace', undefined, { replaceHistory: true });

      expect(window.location.replace).toHaveBeenCalledWith('/before-init-replace');
      expect(window.location.assign).not.toHaveBeenCalled();
    });

    it('runs a remote render queued before init once init completes', async () => {
      const freshFramework = await importFreshFramework();
      mockFragmentResponse();
      const patchLocation = document.createElement('div');
      document.body.appendChild(patchLocation);

      let settled = false;
      const pending = freshFramework.remoteRender({ src: '/queued', patchLocation }).then(() => {
        settled = true;
      });
      await Promise.resolve();

      expect(mockFetch).not.toHaveBeenCalled();
      expect(settled).toBe(false);

      freshFramework.init();
      await pending;

      expect(mockFetch).toHaveBeenCalledWith('http://localhost:3000/queued?_f=1', expect.anything());
      expect(patchLocation.innerHTML).toBe('<em>Queued Content</em>');
    });

    it('rejects remote renders past the queue limit and names the limit', async () => {
      const freshFramework = await importFreshFramework();
      mockFragmentResponse();
      const patchLocation = document.createElement('div');
      document.body.appendChild(patchLocation);

      const queued = Array.from({ length: MAX_CALLS_BEFORE_INIT }, () =>
        freshFramework.remoteRender({ src: '/queued', patchLocation })
      );
      const overflow = freshFramework.remoteRender({ src: '/overflow', patchLocation });

      await expect(overflow).rejects.toThrow(String(MAX_CALLS_BEFORE_INIT));

      freshFramework.init();
      await Promise.all(queued);
      expect(mockFetch).toHaveBeenCalledTimes(MAX_CALLS_BEFORE_INIT);
    });

    it('delivers a diagnostic raised before init to a listener queued before init', async () => {
      vi.spyOn(console, 'warn').mockImplementation(() => {});
      const listener = vi.fn();
      const hooksWindow = window as Window & { __PP_HOOKS_QUEUE__?: unknown[] };
      hooksWindow.__PP_HOOKS_QUEUE__ = [{ event: 'diagnostic', callback: listener }];
      const freshFramework = await importFreshFramework();
      const freshCoordination = await import('@/pk/coordination');

      await freshCoordination.reloadPartial('early-partial');
      expect(listener).not.toHaveBeenCalled();

      freshFramework.init();
      await freshCoordination.reloadPartial('early-partial');

      expect(listener).toHaveBeenCalledTimes(1);
      expect(listener).toHaveBeenCalledWith(expect.objectContaining({
        kind: 'partial-not-found',
        name: 'early-partial'
      }));
      delete hooksWindow.__PP_HOOKS_QUEUE__;
    });

    it('runs a partial patch queued before init once init completes', async () => {
      const freshFramework = await importFreshFramework();
      const target = document.createElement('div');
      target.id = 'queued-patch';
      target.textContent = 'old';
      document.body.appendChild(target);

      freshFramework.patchPartial('<div id="queued-patch">new</div>', '#queued-patch');
      expect(document.getElementById('queued-patch')?.textContent).toBe('old');

      freshFramework.init();

      expect(document.getElementById('queued-patch')?.textContent).toBe('new');
    });
  });

  describe('globalConfig setter', () => {
    it('should update the global configuration', () => {
      const beforeNav = vi.fn();
      const afterNav = vi.fn();

      PPFramework.globalConfig = {
        loaderColour: '#ff0',
        beforeNavigate: beforeNav,
        afterNavigate: afterNav,
      };

      expect(PPFramework.globalConfig.loaderColour).toBe('#ff0');
      expect(PPFramework.globalConfig.beforeNavigate).toBe(beforeNav);
      expect(PPFramework.globalConfig.afterNavigate).toBe(afterNav);
    });
  });

  describe('Getters and setters (backwards compatibility)', () => {
    beforeEach(() => {
      PPFramework.init();
    });

    it('navigating getter should return false when not navigating', () => {
      expect(PPFramework.navigating).toBe(false);
    });

    it('navigating setter should be a no-op (does not throw)', () => {
      expect(() => { PPFramework.navigating = true; }).not.toThrow();
    });

    it('loaderElement getter should return the loader bar element', () => {
      const loader = PPFramework.loaderElement;
      expect(loader).not.toBeNull();
      expect(loader?.id).toBe('ppf-loader-bar');
    });

    it('loaderElement setter should be a no-op (does not throw)', () => {
      expect(() => { PPFramework.loaderElement = null; }).not.toThrow();
    });

    it('currentAbortController setter should be a no-op (does not throw)', () => {
      expect(() => { PPFramework.currentAbortController = null; }).not.toThrow();
    });

    it('isOnline getter should return a boolean', () => {
      expect(typeof PPFramework.isOnline).toBe('boolean');
    });
  });

  describe('createLoaderIndicator', () => {
    beforeEach(() => {
      PPFramework.init();
    });

    it('should recreate the loader with a new colour', () => {
      PPFramework.createLoaderIndicator('purple');
      const loader = document.getElementById('ppf-loader-bar');
      expect(loader).not.toBeNull();
      expect(loader!.style.background).toBe('purple');
    });
  });

  describe('toggleLoader without init', () => {
    it('should not throw when called before loader is created', () => {
      expect(() => PPFramework.toggleLoader(true)).not.toThrow();
      expect(() => PPFramework.toggleLoader(false)).not.toThrow();
    });
  });
});
