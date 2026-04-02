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

vi.mock('@/vdom/renderer', () => ({
    replaceElementWithTracking: vi.fn((original: Element, replacement: Node) => {
        original.replaceWith(replacement);
    }),
}));

import {PikoSvgInline, registerPikoSvgInline} from './PikoSvgInline';
import {replaceElementWithTracking} from '@/vdom/renderer';

const VALID_SVG = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M1 1"/></svg>';
const VALID_SVG_WITH_CLASS = '<svg xmlns="http://www.w3.org/2000/svg" class="svg-original"><path d="M1 1"/></svg>';

function mockSvgResponse(svg: string = VALID_SVG): Response {
    return new Response(svg, {
        status: 200,
        statusText: 'OK',
        headers: {'Content-Type': 'image/svg+xml'},
    });
}

function mockErrorResponse(status: number, statusText: string = 'Not Found'): Response {
    return new Response('', {status, statusText});
}

function freshSvgFetchImpl(svg: string = VALID_SVG) {
    return () => Promise.resolve(mockSvgResponse(svg));
}

describe('PikoSvgInline', () => {
    let fetchSpy: ReturnType<typeof vi.spyOn>;
    const mockedReplace = vi.mocked(replaceElementWithTracking);

    beforeEach(() => {
        fetchSpy = vi.spyOn(globalThis, 'fetch');
        mockedReplace.mockClear();
    });

    afterEach(() => {
        document.body.innerHTML = '';
        vi.restoreAllMocks();
    });

    describe('registerPikoSvgInline()', () => {
        it('should register the piko-svg-inline custom element', () => {
            registerPikoSvgInline();
            expect(customElements.get('piko-svg-inline')).toBe(PikoSvgInline);
        });

        it('should be idempotent when called multiple times', () => {
            registerPikoSvgInline();
            registerPikoSvgInline();
            expect(customElements.get('piko-svg-inline')).toBe(PikoSvgInline);
        });
    });

    describe('connectedCallback()', () => {
        it('should fetch the SVG when the element has a src attribute', async () => {
            fetchSpy.mockResolvedValueOnce(mockSvgResponse());

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/test.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(fetchSpy).toHaveBeenCalledWith(
                    '/icons/test.svg',
                    expect.objectContaining({signal: expect.any(AbortSignal)})
                );
            });
        });

        it('should not fetch when the element has no src attribute', async () => {
            const el = document.createElement('piko-svg-inline');
            document.body.appendChild(el);

            await new Promise((r) => setTimeout(r, 10));

            expect(fetchSpy).not.toHaveBeenCalled();
        });

        it('should parse and inline the SVG after a successful fetch', async () => {
            fetchSpy.mockResolvedValueOnce(mockSvgResponse());

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/check.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalled();
            });

            const args = mockedReplace.mock.calls[0];
            expect(args[0]).toBe(el);
            expect((args[1] as Element).tagName.toLowerCase()).toBe('svg');
            expect(args[2]).toEqual({watchProps: ['src']});
        });
    });

    describe('attributeChangedCallback()', () => {
        it('should trigger a new fetch when src attribute changes', async () => {
            fetchSpy.mockImplementation(freshSvgFetchImpl());

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/a.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(fetchSpy).toHaveBeenCalledWith('/icons/a.svg', expect.anything());
            });

            fetchSpy.mockClear();
            mockedReplace.mockClear();

            el.setAttribute('src', '/icons/b.svg');

            await vi.waitFor(() => {
                expect(fetchSpy).toHaveBeenCalledWith('/icons/b.svg', expect.anything());
            });
        });

        it('should not re-fetch when setting the same src value via the property setter', async () => {
            fetchSpy.mockImplementation(freshSvgFetchImpl());

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/same.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(fetchSpy).toHaveBeenCalledTimes(1);
            });

            const instance = el as unknown as PikoSvgInline;
            instance.src = '/icons/same.svg';

            await new Promise((r) => setTimeout(r, 10));
            expect(fetchSpy).toHaveBeenCalledTimes(1);
        });
    });

    describe('disconnectedCallback()', () => {
        it('should abort the in-flight fetch when the element is removed', async () => {
            let abortSignal: AbortSignal | undefined;
            fetchSpy.mockImplementation((_url: string | URL | Request, init?: RequestInit) => {
                abortSignal = init?.signal as AbortSignal;
                return new Promise((_resolve, reject) => {
                    abortSignal!.addEventListener('abort', () => {
                        reject(new DOMException('The operation was aborted.', 'AbortError'));
                    });
                });
            });

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/slow.svg');
            document.body.appendChild(el);

            await new Promise((r) => setTimeout(r, 5));

            expect(abortSignal).toBeDefined();
            expect(abortSignal!.aborted).toBe(false);

            el.remove();

            expect(abortSignal!.aborted).toBe(true);
        });

        it('should not produce warnings when disconnected during a fetch', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            fetchSpy.mockImplementation((_url: string | URL | Request, init?: RequestInit) => {
                const signal = init?.signal as AbortSignal;
                return new Promise((_resolve, reject) => {
                    signal.addEventListener('abort', () => {
                        reject(new DOMException('The operation was aborted.', 'AbortError'));
                    });
                });
            });

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/abort-graceful.svg');
            document.body.appendChild(el);

            await new Promise((r) => setTimeout(r, 5));

            el.remove();

            await new Promise((r) => setTimeout(r, 20));

            expect(warnSpy).not.toHaveBeenCalled();

            warnSpy.mockRestore();
        });
    });

    describe('SVG inlining', () => {
        it('should extract the SVG element from the parsed response', async () => {
            fetchSpy.mockResolvedValueOnce(mockSvgResponse());

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/extract.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalled();
            });

            const replacement = mockedReplace.mock.calls[0][1] as Element;
            expect(replacement.tagName.toLowerCase()).toBe('svg');
        });

        it('should copy attributes from the host element to the SVG, excluding src', async () => {
            fetchSpy.mockResolvedValueOnce(mockSvgResponse());

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/attrs.svg');
            el.setAttribute('data-testid', 'my-icon');
            el.setAttribute('aria-label', 'Test icon');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalled();
            });

            const svg = mockedReplace.mock.calls[0][1] as Element;
            expect(svg.getAttribute('data-testid')).toBe('my-icon');
            expect(svg.getAttribute('aria-label')).toBe('Test icon');
            expect(svg.hasAttribute('src')).toBe(false);
        });

        it('should merge class attributes from host and SVG elements', async () => {
            fetchSpy.mockResolvedValueOnce(mockSvgResponse(VALID_SVG_WITH_CLASS));

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/class-merge.svg');
            el.setAttribute('class', 'host-class');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalled();
            });

            const svg = mockedReplace.mock.calls[0][1] as Element;
            const classes = svg.getAttribute('class');
            expect(classes).toContain('svg-original');
            expect(classes).toContain('host-class');
        });

        it('should set class from host when SVG has no existing class', async () => {
            fetchSpy.mockResolvedValueOnce(mockSvgResponse());

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/class-only-host.svg');
            el.setAttribute('class', 'only-host-class');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalled();
            });

            const svg = mockedReplace.mock.calls[0][1] as Element;
            expect(svg.getAttribute('class')).toBe('only-host-class');
        });

        it('should preserve SVG-specific attributes like viewBox', async () => {
            fetchSpy.mockResolvedValueOnce(mockSvgResponse());

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/viewbox.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalled();
            });

            const svg = mockedReplace.mock.calls[0][1] as Element;
            expect(svg.getAttribute('viewBox')).toBe('0 0 24 24');
        });

        it('should not overwrite attributes already present on the SVG', async () => {
            const svgWithWidth = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="100"><path d="M1 1"/></svg>';
            fetchSpy.mockResolvedValueOnce(mockSvgResponse(svgWithWidth));

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/no-overwrite.svg');
            el.setAttribute('width', '200');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalled();
            });

            const svg = mockedReplace.mock.calls[0][1] as Element;
            expect(svg.getAttribute('width')).toBe('100');
        });
    });

    describe('error handling', () => {
        it('should log a warning and not crash on network error', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            fetchSpy.mockRejectedValueOnce(new TypeError('Failed to fetch'));

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/network-error.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(warnSpy).toHaveBeenCalledWith(
                    expect.stringContaining('Failed to load SVG from /icons/network-error.svg'),
                    expect.any(TypeError)
                );
            });

            warnSpy.mockRestore();
        });

        it('should log a warning on HTTP error response', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            fetchSpy.mockResolvedValueOnce(mockErrorResponse(404));

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/not-found.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(warnSpy).toHaveBeenCalledWith(
                    expect.stringContaining('Failed to load SVG from /icons/not-found.svg'),
                    expect.any(Error)
                );
            });

            warnSpy.mockRestore();
        });

        it('should log a warning when the response contains no SVG element', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            fetchSpy.mockResolvedValueOnce(
                new Response('<html><body>not svg</body></html>', {status: 200})
            );

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/invalid.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(warnSpy).toHaveBeenCalledWith(
                    expect.stringContaining('Failed to load SVG from /icons/invalid.svg'),
                    expect.any(Error)
                );
            });

            warnSpy.mockRestore();
        });

        it('should set error comment in innerHTML on fetch failure', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
            fetchSpy.mockRejectedValueOnce(new TypeError('Failed to fetch'));

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/error-comment.svg');
            document.body.appendChild(el);

            await vi.waitFor(() => {
                expect(el.innerHTML).toContain('piko-svg-inline error');
            });

            warnSpy.mockRestore();
        });
    });

    describe('caching', () => {
        it('should only make one network request when the same URL is fetched twice sequentially', async () => {
            const url = '/icons/cache-test-sequential.svg';
            fetchSpy.mockImplementation(freshSvgFetchImpl());

            const el1 = document.createElement('piko-svg-inline');
            el1.setAttribute('src', url);
            document.body.appendChild(el1);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalledTimes(1);
            });

            mockedReplace.mockClear();

            const el2 = document.createElement('piko-svg-inline');
            el2.setAttribute('src', url);
            document.body.appendChild(el2);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalledTimes(1);
            });

            expect(fetchSpy).toHaveBeenCalledTimes(1);
        });

        it('should make separate fetches for different URLs', async () => {
            const urlA = '/icons/cache-test-diff-a.svg';
            const urlB = '/icons/cache-test-diff-b.svg';
            fetchSpy.mockImplementation(freshSvgFetchImpl());

            const el1 = document.createElement('piko-svg-inline');
            el1.setAttribute('src', urlA);
            document.body.appendChild(el1);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalledTimes(1);
            });

            mockedReplace.mockClear();

            const el2 = document.createElement('piko-svg-inline');
            el2.setAttribute('src', urlB);
            document.body.appendChild(el2);

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalledTimes(1);
            });

            expect(fetchSpy).toHaveBeenCalledTimes(2);
        });
    });

    describe('request deduplication', () => {
        it('should deduplicate concurrent fetches for the same URL', async () => {
            const url = '/icons/dedup-concurrent.svg';
            let resolvePromise!: (value: Response) => void;
            fetchSpy.mockImplementation(() => new Promise((resolve) => {
                resolvePromise = resolve;
            }));

            const el1 = document.createElement('piko-svg-inline');
            el1.setAttribute('src', url);
            document.body.appendChild(el1);

            const el2 = document.createElement('piko-svg-inline');
            el2.setAttribute('src', url);
            document.body.appendChild(el2);

            await new Promise((r) => setTimeout(r, 10));

            expect(fetchSpy).toHaveBeenCalledTimes(1);

            resolvePromise(mockSvgResponse());

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalledTimes(2);
            });
        });
    });

    describe('race conditions', () => {
        it('should only apply the result from the last src change when changed rapidly', async () => {
            const svgA = '<svg xmlns="http://www.w3.org/2000/svg" data-id="a"><path d="M1 1"/></svg>';
            const svgB = '<svg xmlns="http://www.w3.org/2000/svg" data-id="b"><path d="M2 2"/></svg>';

            let resolveA!: (value: Response) => void;
            let resolveB!: (value: Response) => void;

            fetchSpy
                .mockImplementationOnce(() => new Promise((resolve) => { resolveA = resolve; }))
                .mockImplementationOnce(() => new Promise((resolve) => { resolveB = resolve; }));

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/race-cond-a.svg');
            document.body.appendChild(el);

            await new Promise((r) => setTimeout(r, 5));

            el.setAttribute('src', '/icons/race-cond-b.svg');

            await new Promise((r) => setTimeout(r, 5));

            resolveB(mockSvgResponse(svgB));

            await vi.waitFor(() => {
                expect(mockedReplace).toHaveBeenCalledTimes(1);
            });

            const svg = mockedReplace.mock.calls[0][1] as Element;
            expect(svg.getAttribute('data-id')).toBe('b');

            resolveA(mockSvgResponse(svgA));
            await new Promise((r) => setTimeout(r, 20));

            expect(mockedReplace).toHaveBeenCalledTimes(1);
        });

        it('should handle disconnect during in-flight fetch gracefully', async () => {
            const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            fetchSpy.mockImplementation((_url: string | URL | Request, init?: RequestInit) => {
                const signal = init?.signal as AbortSignal;
                return new Promise((_resolve, reject) => {
                    signal.addEventListener('abort', () => {
                        reject(new DOMException('The operation was aborted.', 'AbortError'));
                    });
                });
            });

            const el = document.createElement('piko-svg-inline');
            el.setAttribute('src', '/icons/disconnect-graceful.svg');
            document.body.appendChild(el);

            await new Promise((r) => setTimeout(r, 5));

            el.remove();

            await new Promise((r) => setTimeout(r, 20));

            expect(warnSpy).not.toHaveBeenCalled();
            expect(mockedReplace).not.toHaveBeenCalled();

            warnSpy.mockRestore();
        });
    });

    describe('src property', () => {
        it('should return the current src value', () => {
            fetchSpy.mockImplementation(freshSvgFetchImpl());

            const el = document.createElement('piko-svg-inline') as PikoSvgInline;
            expect(el.src).toBe('');

            el.setAttribute('src', '/icons/prop-get.svg');
            document.body.appendChild(el);
            expect(el.src).toBe('/icons/prop-get.svg');
        });

        it('should trigger a fetch when setting the src property', async () => {
            fetchSpy.mockResolvedValueOnce(mockSvgResponse());

            const el = document.createElement('piko-svg-inline') as PikoSvgInline;
            document.body.appendChild(el);

            el.src = '/icons/property-set.svg';

            await vi.waitFor(() => {
                expect(fetchSpy).toHaveBeenCalledWith('/icons/property-set.svg', expect.anything());
            });
        });

        it('should also set the src attribute when the property is set', async () => {
            fetchSpy.mockImplementation(freshSvgFetchImpl());

            const el = document.createElement('piko-svg-inline') as PikoSvgInline;
            document.body.appendChild(el);

            el.src = '/icons/attr-sync.svg';

            expect(el.getAttribute('src')).toBe('/icons/attr-sync.svg');
        });
    });
});
