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
import {createRemoteRenderer, getNodeKey, type RemoteRenderer} from './RemoteRenderer';
import type {ModuleLoader} from '@/services/ModuleLoader';
import type {SpriteSheetManager} from '@/services/SpriteSheetManager';
import type {LinkHeaderParser} from '@/services/LinkHeaderParser';
import {createHookManager, HookEvent, type HookManager} from '@/services/HookManager';
import type {DOMOperations, WindowOperations, HTTPOperations} from '@/core/BrowserAPIs';
import {resetGlobalPageContext, getGlobalPageContext} from '@/services/PageContext';

describe('RemoteRenderer', () => {
    let remoteRenderer: RemoteRenderer;
    let moduleLoader: ModuleLoader;
    let spriteSheetManager: SpriteSheetManager;
    let linkHeaderParser: LinkHeaderParser;
    let hookManager: HookManager;
    let onDOMUpdated: (root: HTMLElement) => void;
    let mockDomOps: DOMOperations;
    let mockWindowOps: WindowOperations;
    let mockHttpOps: HTTPOperations;
    let mockFetch: ReturnType<typeof vi.fn>;

    function createMockResponse(options: {
        ok?: boolean;
        status?: number;
        statusText?: string;
        headers?: Record<string, string | null>;
        body?: string;
    }) {
        const {
            ok = true,
            status = 200,
            statusText = 'OK',
            headers = {'X-PP-Response-Support': 'fragment-patch'},
            body = '<div id="app"><div>Content</div></div>'
        } = options;

        return {
            ok,
            status,
            statusText,
            headers: {
                get: (name: string) => headers[name] ?? null
            },
            text: () => Promise.resolve(body)
        };
    }

    beforeEach(() => {
        hookManager = createHookManager();
        hookManager.setReady();

        moduleLoader = {
            loadFromDocument: vi.fn(),
            loadFromDocumentAsync: vi.fn().mockResolvedValue(undefined),
            hasLoaded: vi.fn().mockReturnValue(false),
            getLoadedModules: vi.fn().mockReturnValue(new Set<string>())
        } as unknown as ModuleLoader;

        spriteSheetManager = {
            merge: vi.fn()
        } as unknown as SpriteSheetManager;

        linkHeaderParser = {
            parseAndApply: vi.fn()
        } as unknown as LinkHeaderParser;

        onDOMUpdated = vi.fn<(root: HTMLElement) => void>();
        mockFetch = vi.fn();

        mockDomOps = {
            createElement: (tag: string) => document.createElement(tag),
            createTextNode: (data: string) => document.createTextNode(data),
            createComment: (data: string) => document.createComment(data),
            createDocumentFragment: () => document.createDocumentFragment(),
            getElementById: (id: string) => document.getElementById(id),
            querySelector: (sel: string) => document.querySelector(sel),
            querySelectorAll: (sel: string) => document.querySelectorAll(sel),
            parseHTML: (html: string) => new DOMParser().parseFromString(html, 'text/html'),
            getHead: () => document.head,
            getActiveElement: () => document.activeElement
        };

        mockWindowOps = {
            getLocation: () => window.location,
            getLocationOrigin: () => 'http://localhost:3000',
            getLocationHref: () => 'http://localhost:3000/',
            setLocationHref: vi.fn(),
            locationReload: vi.fn(),
            historyPushState: vi.fn(),
            historyReplaceState: vi.fn(),
            getHistoryState: () => window.history.state,
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
            getScrollY: () => window.scrollY,
            scrollTo: vi.fn(),
            setScrollRestoration: vi.fn(),
            getScrollRestoration: () => 'auto' as ScrollRestoration
        };

        mockHttpOps = {
            fetch: mockFetch as unknown as (input: URL | RequestInfo, init?: RequestInit) => Promise<Response>
        };

        remoteRenderer = createRemoteRenderer({
            moduleLoader,
            spriteSheetManager,
            linkHeaderParser,
            onDOMUpdated,
            domOps: mockDomOps,
            windowOps: mockWindowOps,
            http: mockHttpOps,
            hookManager
        });
    });

    afterEach(() => {
        vi.clearAllMocks();
        document.body.innerHTML = '';
        document.head.querySelectorAll('style[pk-page]').forEach(el => el.remove());
        resetGlobalPageContext();
    });

    describe('createRemoteRenderer', () => {
        it('should create a remote renderer instance', () => {
            expect(remoteRenderer).toBeDefined();
            expect(typeof remoteRenderer.render).toBe('function');
            expect(typeof remoteRenderer.patchPartial).toBe('function');
        });
    });

    describe('getNodeKey', () => {
        it('returns null for non-element nodes', () => {
            expect(getNodeKey(document.createTextNode('hi'))).toBe(null);
            expect(getNodeKey(document.createComment('c'))).toBe(null);
        });

        it('returns null for elements with no identifying attribute', () => {
            const el = document.createElement('div');
            expect(getNodeKey(el)).toBe(null);
        });

        it('returns the p-key when present', () => {
            const el = document.createElement('section');
            el.setAttribute('p-key', 'r.0:1:1');
            expect(getNodeKey(el)).toBe('r.0:1:1');
        });

        it('namespaces the key with partial_name when present', () => {
            const examples = document.createElement('section');
            examples.setAttribute('partial_name', 'examples/grid');
            examples.setAttribute('p-key', 'r.0:1:1');

            const integrations = document.createElement('section');
            integrations.setAttribute('partial_name', 'integrations/grid');
            integrations.setAttribute('p-key', 'r.0:1:1');

            expect(getNodeKey(examples)).toBe('examples/grid@r.0:1:1');
            expect(getNodeKey(integrations)).toBe('integrations/grid@r.0:1:1');
            expect(getNodeKey(examples)).not.toBe(getNodeKey(integrations));
        });

        it('preserves the original key when the same partial_name is present on both sides', () => {
            const a = document.createElement('section');
            a.setAttribute('partial_name', 'integrations/grid');
            a.setAttribute('p-key', 'r.0:1:1');

            const b = document.createElement('section');
            b.setAttribute('partial_name', 'integrations/grid');
            b.setAttribute('p-key', 'r.0:1:1');

            expect(getNodeKey(a)).toBe(getNodeKey(b));
        });

        it('falls back through data-stable-id, p-key, then id', () => {
            const stable = document.createElement('div');
            stable.dataset.stableId = 'stable';
            stable.setAttribute('p-key', 'pk');
            stable.id = 'id';
            expect(getNodeKey(stable)).toBe('stable');

            const pkOnly = document.createElement('div');
            pkOnly.setAttribute('p-key', 'pk');
            pkOnly.id = 'id';
            expect(getNodeKey(pkOnly)).toBe('pk');

            const idOnly = document.createElement('div');
            idOnly.id = 'id';
            expect(getNodeKey(idOnly)).toBe('id');
        });
    });

    describe('render', () => {
        describe('basic rendering', () => {
            it('should fetch and render remote content', async () => {
                const patchLocation = document.createElement('div');
                patchLocation.id = 'target';
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: '<div id="app"><p>Hello World</p></div>'
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    'http://localhost:3000/partial/test?_f=1',
                    expect.objectContaining({method: 'GET'})
                );
                expect(patchLocation.innerHTML).toContain('Hello World');
            });

            it('should build URL with args as query params', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/item',
                    args: {id: '123'},
                    patchLocation
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    expect.stringContaining('/partial/item'),
                    expect.any(Object)
                );
                expect(mockFetch).toHaveBeenCalledWith(
                    expect.stringContaining('id=123'),
                    expect.any(Object)
                );
            });

            it('should call onDOMUpdated after patching', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(onDOMUpdated).toHaveBeenCalledWith(patchLocation);
            });
        });

        describe('formData handling', () => {
            it('should use POST method when formData is provided', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/test',
                    formData: {name: 'test', value: 42},
                    patchLocation
                });

                expect(mockFetch).toHaveBeenCalledWith(
                    expect.stringContaining('/partial/test'),
                    expect.objectContaining({
                        method: 'POST'
                    })
                );
                const callArgs = mockFetch.mock.calls[0][1];
                expect(typeof callArgs.body).toBe('string');
                expect(callArgs.headers).toEqual({'Content-Type': 'application/x-www-form-urlencoded'});
                const params = new URLSearchParams(callArgs.body as string);
                expect(params.get('name')).toBe('test');
                expect(params.get('value')).toBe('42');
            });

            it('should handle FormData instance', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                const formData = new FormData();
                formData.append('field', 'value');

                await remoteRenderer.render({
                    src: '/partial/test',
                    formData,
                    patchLocation
                });

                const callBody = mockFetch.mock.calls[0][1].body as string;
                expect(typeof callBody).toBe('string');
                const params = new URLSearchParams(callBody);
                expect(params.get('field')).toBe('value');
            });

            it('should handle URLSearchParams', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                const params = new URLSearchParams();
                params.append('search', 'query');

                await remoteRenderer.render({
                    src: '/partial/test',
                    formData: params,
                    patchLocation
                });

                const callBody = mockFetch.mock.calls[0][1].body as string;
                expect(typeof callBody).toBe('string');
                const resultParams = new URLSearchParams(callBody);
                expect(resultParams.get('search')).toBe('query');
            });

            it('should handle Map input', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                const mapData = new Map<string, string | number>();
                mapData.set('mapKey', 'mapValue');

                await remoteRenderer.render({
                    src: '/partial/test',
                    formData: mapData,
                    patchLocation
                });

                const callBody = mockFetch.mock.calls[0][1].body as string;
                expect(typeof callBody).toBe('string');
                const params = new URLSearchParams(callBody);
                expect(params.get('mapKey')).toBe('mapValue');
            });

            it('should handle object with arrays', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/test',
                    formData: {
                        tags: ['a', 'b', 'c']
                    },
                    patchLocation
                });

                const callBody = mockFetch.mock.calls[0][1].body as string;
                expect(typeof callBody).toBe('string');
                const params = new URLSearchParams(callBody);
                expect(params.getAll('tags')).toEqual(['a', 'b', 'c']);
            });

            it('should skip null and undefined values in arrays', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/test',
                    formData: {
                        items: ['valid', null as unknown as string, undefined as unknown as string, 'also-valid']
                    },
                    patchLocation
                });

                const callBody = mockFetch.mock.calls[0][1].body as string;
                expect(typeof callBody).toBe('string');
                const params = new URLSearchParams(callBody);
                expect(params.getAll('items')).toEqual(['valid', 'also-valid']);
            });

            it('should skip null and undefined scalar values', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/test',
                    formData: {
                        valid: 'yes',
                        nullField: null as unknown as string,
                        undefinedField: undefined as unknown as string
                    },
                    patchLocation
                });

                const callBody = mockFetch.mock.calls[0][1].body as string;
                expect(typeof callBody).toBe('string');
                const params = new URLSearchParams(callBody);
                expect(params.get('valid')).toBe('yes');
                expect(params.has('nullField')).toBe(false);
                expect(params.has('undefinedField')).toBe(false);
            });
        });

        describe('error handling', () => {
            it('should handle network errors gracefully', async () => {
                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockRejectedValueOnce(new Error('Network failure'));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(consoleSpy).toHaveBeenCalledWith(
                    'RemoteRenderer: network error:',
                    expect.any(Error)
                );
                consoleSpy.mockRestore();
            });

            it('should handle non-ok responses', async () => {
                const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    ok: false,
                    status: 500,
                    statusText: 'Internal Server Error'
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(consoleSpy).toHaveBeenCalledWith(
                    'RemoteRenderer: fetch failed',
                    500,
                    'Internal Server Error'
                );
                consoleSpy.mockRestore();
            });

            it('should reload page when response type is not fragment-patch', async () => {
                const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    headers: {'X-PP-Response-Support': null}
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(mockWindowOps.locationReload).toHaveBeenCalled();
                expect(consoleSpy).toHaveBeenCalledWith(
                    expect.stringContaining("expected 'fragment-patch' response")
                );
                consoleSpy.mockRestore();
            });

            it('should warn when target has no patchLocation', async () => {
                const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/test',
                    targets: [{querySelector: '#missing'}]
                });

                expect(consoleSpy).toHaveBeenCalledWith(
                    'RemoteRenderer: target has no patchLocation'
                );
                consoleSpy.mockRestore();
            });
        });

        describe('Link header parsing', () => {
            it('should parse and apply Link header when present', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    headers: {
                        'X-PP-Response-Support': 'fragment-patch',
                        'Link': '</style.css>; rel=stylesheet'
                    }
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(linkHeaderParser.parseAndApply).toHaveBeenCalledWith('</style.css>; rel=stylesheet');
            });

            it('should not call parseAndApply when no Link header', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    headers: {
                        'X-PP-Response-Support': 'fragment-patch'
                    }
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(linkHeaderParser.parseAndApply).not.toHaveBeenCalled();
            });
        });

        describe('sprite sheet handling', () => {
            it('should merge sprite sheet when present', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: '<div id="app"><div>Content</div></div><svg id="sprite"></svg>'
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(spriteSheetManager.merge).toHaveBeenCalled();
            });
        });

        describe('style block processing', () => {
            it('should process style blocks with pk-page attribute', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: `
                        <div id="app" data-pageid="test-page"><div>Content</div></div>
                        <style pk-page>.test { color: red; }</style>
                    `
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                const styleEl = document.head.querySelector('style[data-pk-style-key="test-page"]');
                expect(styleEl).not.toBeNull();
                expect(styleEl?.textContent).toContain('.test { color: red; }');
            });

            it('should not duplicate style blocks with same key', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                const html = `
                    <div id="app" data-pageid="unique-page"><div>Content</div></div>
                    <style pk-page>.test { color: red; }</style>
                `;

                mockFetch.mockResolvedValueOnce(createMockResponse({body: html}));
                await remoteRenderer.render({src: '/partial/test', patchLocation});

                mockFetch.mockResolvedValueOnce(createMockResponse({body: html}));
                await remoteRenderer.render({src: '/partial/test', patchLocation});

                const styleEls = document.head.querySelectorAll('style[data-pk-style-key="unique-page"]');
                expect(styleEls.length).toBe(1);
            });

            it('should skip empty style blocks', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: `
                        <div id="app" data-pageid="empty-style-page"><div>Content</div></div>
                        <style pk-page>   </style>
                    `
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                const styleEl = document.head.querySelector('style[data-pk-style-key="empty-style-page"]');
                expect(styleEl).toBeNull();
            });
        });

        describe('module loading', () => {
            it('should call loadFromDocumentAsync with parsed document', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(moduleLoader.loadFromDocumentAsync).toHaveBeenCalled();
            });
        });

        describe('patch methods', () => {
            it('should use replace method by default', async () => {
                const patchLocation = document.createElement('div');
                patchLocation.innerHTML = '<span>Old content</span>';
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: '<div id="app"><p>New content</p></div>'
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(patchLocation.querySelector('span')).toBeNull();
                expect(patchLocation.querySelector('p')).not.toBeNull();
            });

            it('should use morph method when specified', async () => {
                const patchLocation = document.createElement('div');
                patchLocation.innerHTML = '<p>Original</p>';
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: '<div id="app"><p>Updated</p></div>'
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation,
                    patchMethod: 'morph'
                });

                expect(patchLocation.querySelector('p')?.textContent).toBe('Updated');
            });
        });

        describe('querySelector targeting', () => {
            it('should extract specific element using querySelector', async () => {
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: `
                        <div id="app">
                            <div class="header"><span>Header</span></div>
                            <div class="content"><span>Main content</span></div>
                            <div class="footer"><span>Footer</span></div>
                        </div>
                    `
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation,
                    querySelector: '.content'
                });

                expect(patchLocation.textContent).toContain('Main content');
                expect(patchLocation.textContent).not.toContain('Header');
            });

            it('should warn when querySelector finds no match', async () => {
                const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: '<div id="app"><div>Content</div></div>'
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation,
                    querySelector: '.nonexistent'
                });

                expect(consoleSpy).toHaveBeenCalledWith(
                    'RemoteRenderer: selector ".nonexistent" not found'
                );
                consoleSpy.mockRestore();
            });
        });

        describe('multiple targets', () => {
            it('should patch multiple targets', async () => {
                const target1 = document.createElement('div');
                target1.id = 'target1';
                const target2 = document.createElement('div');
                target2.id = 'target2';
                document.body.appendChild(target1);
                document.body.appendChild(target2);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: `
                        <div id="app">
                            <div class="section1"><span>Section 1</span></div>
                            <div class="section2"><span>Section 2</span></div>
                        </div>
                    `
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    targets: [
                        {patchLocation: target1, querySelector: '.section1'},
                        {patchLocation: target2, querySelector: '.section2'}
                    ]
                });

                expect(target1.textContent).toContain('Section 1');
                expect(target2.textContent).toContain('Section 2');
            });
        });

        describe('patchAttributes', () => {
            it('should sync specified attributes from source to target', async () => {
                const patchLocation = document.createElement('div');
                patchLocation.id = 'target';
                patchLocation.setAttribute('data-state', 'old');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: '<div id="app" data-state="new" data-other="value"><p>Content</p></div>'
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation,
                    patchAttributes: ['data-state']
                });

                expect(patchLocation.getAttribute('data-state')).toBe('new');
                expect(patchLocation.getAttribute('data-other')).toBeNull();
            });
        });

        describe('p-key transformation', () => {
            it('should transform relative p-key values to absolute keys', async () => {
                const patchLocation = document.createElement('div');
                patchLocation.setAttribute('p-key', 'r.0:1:3:0');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: `
                        <div id="app" p-key="r.0">
                            <div p-key="r.0:0">Child 1</div>
                            <div p-key="r.0:1">Child 2</div>
                        </div>
                    `
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation,
                    patchMethod: 'morph'
                });

                const children = patchLocation.querySelectorAll('[p-key]');
                expect(children.length).toBeGreaterThan(0);
            });
        });

        describe('hook events', () => {
            it('should emit PARTIAL_RENDER hook after patching', async () => {
                const callback = vi.fn();
                hookManager.api.on(HookEvent.PARTIAL_RENDER, callback);

                const patchLocation = document.createElement('div');
                patchLocation.id = 'test-target';
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({}));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation
                });

                expect(callback).toHaveBeenCalledWith(expect.objectContaining({
                    src: '/partial/test',
                    patchLocation: 'test-target',
                    timestamp: expect.any(Number)
                }));
            });

            it('should use querySelector as patchLocation identifier if available', async () => {
                const callback = vi.fn();
                hookManager.api.on(HookEvent.PARTIAL_RENDER, callback);

                const patchLocation = document.createElement('div');
                document.body.appendChild(patchLocation);

                mockFetch.mockResolvedValueOnce(createMockResponse({
                    body: '<div id="app"><div class="target">Content</div></div>'
                }));

                await remoteRenderer.render({
                    src: '/partial/test',
                    patchLocation,
                    querySelector: '.target'
                });

                expect(callback).toHaveBeenCalledWith(expect.objectContaining({
                    patchLocation: '.target'
                }));
            });
        });
    });

    describe('patchPartial', () => {
        it('should patch partial content into existing element', () => {
            const existing = document.createElement('div');
            existing.id = 'partial';
            existing.innerHTML = '<p>Old</p>';
            document.body.appendChild(existing);

            remoteRenderer.patchPartial(
                '<div id="partial"><p>New</p></div>',
                '#partial'
            );

            expect(existing.innerHTML).toBe('<p>New</p>');
            expect(onDOMUpdated).toHaveBeenCalledWith(existing);
        });

        it('should warn when source element not found in HTML', () => {
            const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const existing = document.createElement('div');
            existing.id = 'partial';
            document.body.appendChild(existing);

            remoteRenderer.patchPartial(
                '<div id="other">Content</div>',
                '#partial'
            );

            expect(consoleSpy).toHaveBeenCalledWith(
                expect.stringContaining('no element found for selector #partial')
            );
            consoleSpy.mockRestore();
        });

        it('should warn when existing element not found in DOM', () => {
            const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            remoteRenderer.patchPartial(
                '<div id="partial">Content</div>',
                '#partial'
            );

            expect(consoleSpy).toHaveBeenCalledWith(
                expect.stringContaining('no existing element found for selector #partial')
            );
            consoleSpy.mockRestore();
        });

        it('should emit PARTIAL_RENDER hook with inline source', () => {
            const callback = vi.fn();
            hookManager.api.on(HookEvent.PARTIAL_RENDER, callback);

            const existing = document.createElement('div');
            existing.id = 'partial';
            document.body.appendChild(existing);

            remoteRenderer.patchPartial(
                '<div id="partial"><p>Content</p></div>',
                '#partial'
            );

            expect(callback).toHaveBeenCalledWith(expect.objectContaining({
                src: 'inline',
                patchLocation: '#partial',
                timestamp: expect.any(Number)
            }));
        });
    });

    describe('without hookManager', () => {
        it('should work without hookManager', async () => {
            const rendererWithoutHooks = createRemoteRenderer({
                moduleLoader,
                spriteSheetManager,
                linkHeaderParser,
                onDOMUpdated,
                domOps: mockDomOps,
                windowOps: mockWindowOps,
                http: mockHttpOps
            });

            const patchLocation = document.createElement('div');
            document.body.appendChild(patchLocation);

            mockFetch.mockResolvedValueOnce(createMockResponse({}));

            await rendererWithoutHooks.render({
                src: '/partial/test',
                patchLocation
            });

            expect(patchLocation.innerHTML).not.toBe('');
        });
    });

    describe('preservePartialScopes', () => {
        it('should preserve parent scopes on existing elements when morphing', async () => {
            const patchLocation = document.createElement('div');
            patchLocation.setAttribute('partial', 'list_scope');
            patchLocation.innerHTML = `
                <div class="item" partial="list_scope modal_scope page_scope">Item 1</div>
            `;
            document.body.appendChild(patchLocation);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app" partial="list_scope">
                        <div class="item" partial="list_scope">Item 1 Updated</div>
                    </div>
                `
            }));

            await remoteRenderer.render({
                src: '/partial/list',
                patchLocation,
                patchMethod: 'morph',
                preservePartialScopes: true
            });

            const item = patchLocation.querySelector('.item');
            expect(item?.textContent).toBe('Item 1 Updated');
            expect(item?.getAttribute('partial')).toBe('list_scope modal_scope page_scope');
        });

        it('should inherit parent scopes from existing children when adding new elements', async () => {
            const patchLocation = document.createElement('div');
            patchLocation.setAttribute('partial', 'list_scope');
            patchLocation.setAttribute('p-key', 'r.0');
            patchLocation.innerHTML = `
                <div class="item" partial="list_scope modal_scope page_scope" p-key="r.0:0">Item 1</div>
            `;
            document.body.appendChild(patchLocation);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app" partial="list_scope" p-key="r.0">
                        <div class="item" partial="list_scope" p-key="r.0:0">Item 1</div>
                        <div class="item" partial="list_scope" p-key="r.0:1">Item 2 (new)</div>
                    </div>
                `
            }));

            await remoteRenderer.render({
                src: '/partial/list',
                patchLocation,
                patchMethod: 'morph',
                preservePartialScopes: true
            });

            const items = patchLocation.querySelectorAll('.item');
            expect(items.length).toBe(2);

            expect(items[0].getAttribute('partial')).toBe('list_scope modal_scope page_scope');
            expect(items[1].getAttribute('partial')).toBe('list_scope modal_scope page_scope');
        });

        it('should NOT modify server scopes on initial load (no existing children)', async () => {
            const patchLocation = document.createElement('div');
            patchLocation.setAttribute('partial', 'modal_scope');
            document.body.appendChild(patchLocation);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app" partial="modal_scope">
                        <div class="form" partial="form_scope modal_scope page_scope">
                            <div class="list" partial="list_scope modal_scope page_scope">Items</div>
                        </div>
                    </div>
                `
            }));

            await remoteRenderer.render({
                src: '/partial/modal',
                patchLocation,
                patchMethod: 'morph',
                preservePartialScopes: true
            });

            const form = patchLocation.querySelector('.form');
            const list = patchLocation.querySelector('.list');
            expect(form?.getAttribute('partial')).toBe('form_scope modal_scope page_scope');
            expect(list?.getAttribute('partial')).toBe('list_scope modal_scope page_scope');
        });

        it('should handle nested partials correctly during refresh', async () => {
            const patchLocation = document.createElement('div');
            patchLocation.setAttribute('partial', 'list_scope');
            patchLocation.setAttribute('slot', 'list');
            patchLocation.setAttribute('p-key', 'r.0');
            patchLocation.innerHTML = `
                <div class="contact" partial="list_scope modal_scope page_scope" p-key="r.0:0">
                    <input partial="list_scope modal_scope page_scope" name="contact[0]" value="John" />
                </div>
            `;
            document.body.appendChild(patchLocation);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app" partial="list_scope" slot="list" p-key="r.0">
                        <div class="contact" partial="list_scope" p-key="r.0:0">
                            <input partial="list_scope" name="contact[0]" value="John" />
                        </div>
                        <div class="contact" partial="list_scope" p-key="r.0:1">
                            <input partial="list_scope" name="contact[1]" value="" />
                        </div>
                    </div>
                `
            }));

            await remoteRenderer.render({
                src: '/partial/contacts',
                patchLocation,
                querySelector: '[slot="list"]',
                patchMethod: 'morph',
                preservePartialScopes: true
            });

            const contacts = patchLocation.querySelectorAll('.contact');
            const inputs = patchLocation.querySelectorAll('input');

            expect(contacts.length).toBe(2);

            contacts.forEach(contact => {
                expect(contact.getAttribute('partial')).toBe('list_scope modal_scope page_scope');
            });
            inputs.forEach(input => {
                expect(input.getAttribute('partial')).toBe('list_scope modal_scope page_scope');
            });
        });
    });

    describe('childrenOnly option', () => {
        it('should replace children while preserving parent element', async () => {
            const patchLocation = document.createElement('div');
            patchLocation.id = 'parent';
            patchLocation.setAttribute('data-custom', 'value');
            patchLocation.innerHTML = '<span>Old content</span>';
            document.body.appendChild(patchLocation);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: '<div id="app"><p>New child 1</p><p>New child 2</p></div>'
            }));

            await remoteRenderer.render({
                src: '/partial/test',
                patchLocation
            });

            const parent = document.getElementById('parent');
            expect(parent).not.toBeNull();
            expect(parent?.getAttribute('data-custom')).toBe('value');
            expect(parent?.querySelector('span')).toBeNull();
            expect(parent?.querySelectorAll('p').length).toBe(2);
        });
    });

    describe('cached script re-registration', () => {
        it('should not call loadModule for scripts that are not yet loaded', async () => {
            const patchLocation = document.createElement('div');
            document.body.appendChild(patchLocation);

            vi.mocked(moduleLoader.hasLoaded).mockReturnValue(false);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app"><div>Content</div></div>
                    <script type="module" src="/_piko/assets/pk-js/partials/step1.js"></script>
                `
            }));

            const pageContext = getGlobalPageContext();
            const loadModuleSpy = vi.spyOn(pageContext, 'loadModule');

            await remoteRenderer.render({
                src: '/partial/test',
                patchLocation
            });

            expect(loadModuleSpy).not.toHaveBeenCalled();
        });

        it('should call loadModule for scripts that are already cached', async () => {
            const patchLocation = document.createElement('div');
            document.body.appendChild(patchLocation);

            const scriptUrl = '/_piko/assets/pk-js/partials/step2.js';

            vi.mocked(moduleLoader.hasLoaded).mockImplementation((src: string) => src === scriptUrl);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app"><div>Content</div></div>
                    <script type="module" src="${scriptUrl}"></script>
                `
            }));

            const pageContext = getGlobalPageContext();
            const loadModuleSpy = vi.spyOn(pageContext, 'loadModule');

            await remoteRenderer.render({
                src: '/partial/test',
                patchLocation
            });

            expect(loadModuleSpy).toHaveBeenCalledWith(scriptUrl);
        });

        it('should handle multiple scripts with mixed cache states', async () => {
            const patchLocation = document.createElement('div');
            document.body.appendChild(patchLocation);

            const cachedScript = '/_piko/assets/pk-js/partials/cached.js';
            const newScript = '/_piko/assets/pk-js/partials/new.js';

            vi.mocked(moduleLoader.hasLoaded).mockImplementation((src: string) => src === cachedScript);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app"><div>Content</div></div>
                    <script type="module" src="${cachedScript}"></script>
                    <script type="module" src="${newScript}"></script>
                `
            }));

            const pageContext = getGlobalPageContext();
            const loadModuleSpy = vi.spyOn(pageContext, 'loadModule');

            await remoteRenderer.render({
                src: '/partial/test',
                patchLocation
            });

            expect(loadModuleSpy).toHaveBeenCalledTimes(1);
            expect(loadModuleSpy).toHaveBeenCalledWith(cachedScript);
            expect(loadModuleSpy).not.toHaveBeenCalledWith(newScript);
        });

        it('should skip scripts without src attribute', async () => {
            const patchLocation = document.createElement('div');
            document.body.appendChild(patchLocation);

            vi.mocked(moduleLoader.hasLoaded).mockReturnValue(true);

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app"><div>Content</div></div>
                    <script type="module">console.log('inline');</script>
                `
            }));

            const pageContext = getGlobalPageContext();
            const loadModuleSpy = vi.spyOn(pageContext, 'loadModule');

            await remoteRenderer.render({
                src: '/partial/test',
                patchLocation
            });

            expect(loadModuleSpy).not.toHaveBeenCalled();
        });

        it('should handle modal navigation scenario: step2 -> step3 -> step2', async () => {
            const patchLocation = document.createElement('div');
            document.body.appendChild(patchLocation);

            const step2Script = '/_piko/assets/pk-js/partials/step2.js';
            const step3Script = '/_piko/assets/pk-js/partials/step3.js';
            const loadedScripts = new Set<string>();

            vi.mocked(moduleLoader.hasLoaded).mockImplementation((src: string) => loadedScripts.has(src));

            vi.mocked(moduleLoader.loadFromDocumentAsync).mockImplementation(async (doc: Document) => {
                doc.querySelectorAll('script[type="module"]').forEach(el => {
                    const src = el.getAttribute('src');
                    if (src) loadedScripts.add(src);
                });
            });

            const pageContext = getGlobalPageContext();
            const loadModuleSpy = vi.spyOn(pageContext, 'loadModule');

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app"><div>Step 2</div></div>
                    <script type="module" src="${step2Script}"></script>
                `
            }));
            await remoteRenderer.render({src: '/partial/step2', patchLocation});

            expect(loadModuleSpy).not.toHaveBeenCalled();

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app"><div>Step 3</div></div>
                    <script type="module" src="${step3Script}"></script>
                `
            }));
            await remoteRenderer.render({src: '/partial/step3', patchLocation});

            expect(loadModuleSpy).not.toHaveBeenCalled();

            mockFetch.mockResolvedValueOnce(createMockResponse({
                body: `
                    <div id="app"><div>Step 2</div></div>
                    <script type="module" src="${step2Script}"></script>
                `
            }));
            await remoteRenderer.render({src: '/partial/step2', patchLocation});

            expect(loadModuleSpy).toHaveBeenCalledWith(step2Script);
        });
    });
});
