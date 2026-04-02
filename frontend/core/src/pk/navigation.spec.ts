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
import {
    navigate,
    goBack,
    goForward,
    go,
    currentRoute,
    buildUrl,
    updateQuery,
    registerNavigationGuard,
    matchPath,
    extractParams
} from '@/pk/navigation';

vi.mock('@/core/PPFramework', () => ({
    PPFramework: { navigateTo: undefined },
}));

describe('navigation (PK Navigation Helpers)', () => {
    const originalLocation = window.location;

    beforeEach(() => {
        Object.defineProperty(window, 'location', {
            value: {
                href: 'https://example.com/products/electronics?page=1&sort=price#reviews',
                pathname: '/products/electronics',
                search: '?page=1&sort=price',
                hash: '#reviews',
                origin: 'https://example.com',
                replace: vi.fn(),
            },
            writable: true,
            configurable: true
        });

        vi.spyOn(window.history, 'pushState').mockImplementation(() => {});
        vi.spyOn(window.history, 'replaceState').mockImplementation(() => {});
        vi.spyOn(window.history, 'back').mockImplementation(() => {});
        vi.spyOn(window.history, 'forward').mockImplementation(() => {});
        vi.spyOn(window.history, 'go').mockImplementation(() => {});

        vi.spyOn(window, 'scrollTo').mockImplementation(() => {});

        vi.spyOn(window, 'dispatchEvent').mockImplementation(() => true);
    });

    afterEach(() => {
        vi.restoreAllMocks();
        Object.defineProperty(window, 'location', {
            value: originalLocation,
            writable: true,
            configurable: true
        });
    });

    describe('currentRoute', () => {
        it('should return current path', () => {
            const route = currentRoute();
            expect(route.path).toBe('/products/electronics');
        });

        it('should parse query parameters', () => {
            const route = currentRoute();
            expect(route.query).toEqual({
                page: '1',
                sort: 'price'
            });
        });

        it('should return hash without #', () => {
            const route = currentRoute();
            expect(route.hash).toBe('reviews');
        });

        it('should return full href', () => {
            const route = currentRoute();
            expect(route.href).toBe('https://example.com/products/electronics?page=1&sort=price#reviews');
        });

        it('should return origin', () => {
            const route = currentRoute();
            expect(route.origin).toBe('https://example.com');
        });

        describe('getParam', () => {
            it('should return param value', () => {
                const route = currentRoute();
                expect(route.getParam('page')).toBe('1');
            });

            it('should return null for missing param', () => {
                const route = currentRoute();
                expect(route.getParam('missing')).toBeNull();
            });
        });

        describe('hasParam', () => {
            it('should return true for existing param', () => {
                const route = currentRoute();
                expect(route.hasParam('page')).toBe(true);
            });

            it('should return false for missing param', () => {
                const route = currentRoute();
                expect(route.hasParam('missing')).toBe(false);
            });
        });

        describe('getParams', () => {
            it('should return all values for repeated param', () => {
                Object.defineProperty(window, 'location', {
                    value: {
                        ...window.location,
                        search: '?tag=a&tag=b&tag=c'
                    },
                    writable: true,
                    configurable: true
                });

                const route = currentRoute();
                expect(route.getParams('tag')).toEqual(['a', 'b', 'c']);
            });
        });
    });

    describe('buildUrl', () => {
        it('should build URL with query params', () => {
            const url = buildUrl('/products', { category: 'electronics', page: '2' });
            expect(url).toBe('/products?category=electronics&page=2');
        });

        it('should handle numeric params', () => {
            const url = buildUrl('/products', { page: 1, limit: 10 });
            expect(url).toBe('/products?page=1&limit=10');
        });

        it('should handle boolean params', () => {
            const url = buildUrl('/products', { featured: true, archived: false });
            expect(url).toBe('/products?featured=true&archived=false');
        });

        it('should skip null and undefined params', () => {
            const url = buildUrl('/products', {
                category: 'electronics',
                subcategory: null,
                brand: undefined
            });
            expect(url).toBe('/products?category=electronics');
        });

        it('should add hash fragment', () => {
            const url = buildUrl('/docs', { version: '2.0' }, 'installation');
            expect(url).toBe('/docs?version=2.0#installation');
        });

        it('should work with just path', () => {
            const url = buildUrl('/about');
            expect(url).toBe('/about');
        });

        it('should work with path and hash only', () => {
            const url = buildUrl('/docs', undefined, 'section-1');
            expect(url).toBe('/docs#section-1');
        });
    });

    describe('navigate', () => {
        it('should use history.pushState by default', async () => {
            await navigate('/new-page');

            expect(window.history.pushState).toHaveBeenCalledWith(null, '', '/new-page');
        });

        it('should use history.replaceState when replace option is true', async () => {
            await navigate('/new-page', { replace: true });

            expect(window.history.replaceState).toHaveBeenCalledWith(null, '', '/new-page');
        });

        it('should scroll to top by default', async () => {
            await navigate('/new-page');

            expect(window.scrollTo).toHaveBeenCalledWith(0, 0);
        });

        it('should not scroll when scroll option is false', async () => {
            await navigate('/new-page', { scroll: false });

            expect(window.scrollTo).not.toHaveBeenCalled();
        });

        it('should pass state to history', async () => {
            await navigate('/new-page', { state: { fromCart: true } });

            expect(window.history.pushState).toHaveBeenCalledWith(
                { fromCart: true },
                '',
                '/new-page'
            );
        });

        it('should dispatch popstate event', async () => {
            await navigate('/new-page');

            expect(window.dispatchEvent).toHaveBeenCalledWith(
                expect.any(PopStateEvent)
            );
        });

        it('should do full page load when fullReload is true', async () => {
            const mockHref = vi.fn();
            Object.defineProperty(window.location, 'href', {
                set: mockHref,
                configurable: true
            });

            await navigate('/external', { fullReload: true });

            expect(mockHref).toHaveBeenCalledWith('/external');
        });

        it('should use location.replace for fullReload with replace option', async () => {
            await navigate('/external', { fullReload: true, replace: true });

            expect(window.location.replace).toHaveBeenCalledWith('/external');
        });
    });

    describe('goBack', () => {
        it('should call history.back', () => {
            goBack();
            expect(window.history.back).toHaveBeenCalled();
        });
    });

    describe('goForward', () => {
        it('should call history.forward', () => {
            goForward();
            expect(window.history.forward).toHaveBeenCalled();
        });
    });

    describe('go', () => {
        it('should call history.go with delta', () => {
            go(-2);
            expect(window.history.go).toHaveBeenCalledWith(-2);
        });
    });

    describe('updateQuery', () => {
        it('should add new query parameters', async () => {
            await updateQuery({ filter: 'active' });

            expect(window.history.pushState).toHaveBeenCalled();
        });

        it('should not scroll by default', async () => {
            await updateQuery({ filter: 'active' });

            expect(window.scrollTo).not.toHaveBeenCalled();
        });

        it('should respect scroll option', async () => {
            await updateQuery({ filter: 'active' }, { scroll: true });

            expect(window.scrollTo).toHaveBeenCalledWith(0, 0);
        });
    });

    describe('registerNavigationGuard', () => {
        it('should return unregister function', () => {
            const guard = { beforeNavigate: vi.fn().mockReturnValue(true) };
            const unregister = registerNavigationGuard(guard);

            expect(typeof unregister).toBe('function');
        });

        it('should call beforeNavigate before navigation', async () => {
            const beforeNavigate = vi.fn().mockReturnValue(true);
            const unregister = registerNavigationGuard({ beforeNavigate });

            await navigate('/test');

            expect(beforeNavigate).toHaveBeenCalledWith('/test', expect.any(String));

            unregister();
        });

        it('should cancel navigation when beforeNavigate returns false', async () => {
            const beforeNavigate = vi.fn().mockReturnValue(false);
            const unregister = registerNavigationGuard({ beforeNavigate });

            await navigate('/test');

            expect(window.history.pushState).not.toHaveBeenCalled();

            unregister();
        });

        it('should call afterNavigate after navigation', async () => {
            const afterNavigate = vi.fn();
            const unregister = registerNavigationGuard({ afterNavigate });

            await navigate('/test');

            expect(afterNavigate).toHaveBeenCalledWith('/test', expect.any(String));

            unregister();
        });

        it('should support async beforeNavigate', async () => {
            const beforeNavigate = vi.fn().mockResolvedValue(true);
            const unregister = registerNavigationGuard({ beforeNavigate });

            await navigate('/test');

            expect(beforeNavigate).toHaveBeenCalled();
            expect(window.history.pushState).toHaveBeenCalled();

            unregister();
        });

        it('should stop calling guards after unregister', async () => {
            const beforeNavigate = vi.fn().mockReturnValue(true);
            const unregister = registerNavigationGuard({ beforeNavigate });

            unregister();

            await navigate('/test');

            expect(beforeNavigate).not.toHaveBeenCalled();
        });
    });

    describe('matchPath', () => {
        beforeEach(() => {
            Object.defineProperty(window, 'location', {
                value: {
                    ...window.location,
                    pathname: '/products/electronics/laptop-123'
                },
                writable: true,
                configurable: true
            });
        });

        it('should match exact path', () => {
            Object.defineProperty(window.location, 'pathname', {
                value: '/products',
                configurable: true
            });

            expect(matchPath('/products')).toBe(true);
            expect(matchPath('/other')).toBe(false);
        });

        it('should match path with wildcard', () => {
            expect(matchPath('/products/*')).toBe(true);
            expect(matchPath('/users/*')).toBe(false);
        });

        it('should match path with parameter placeholder', () => {
            expect(matchPath('/products/:category/:id')).toBe(true);
        });

        it('should not match different path structure', () => {
            expect(matchPath('/products/:id')).toBe(false);
        });

        it('should escape special regex characters', () => {
            Object.defineProperty(window.location, 'pathname', {
                value: '/api/v1.0/users',
                configurable: true
            });

            expect(matchPath('/api/v1.0/users')).toBe(true);
        });
    });

    describe('extractParams', () => {
        beforeEach(() => {
            Object.defineProperty(window, 'location', {
                value: {
                    ...window.location,
                    pathname: '/products/electronics/laptop-123'
                },
                writable: true,
                configurable: true
            });
        });

        it('should extract path parameters', () => {
            const params = extractParams('/products/:category/:id');

            expect(params).toEqual({
                category: 'electronics',
                id: 'laptop-123'
            });
        });

        it('should return null for non-matching pattern', () => {
            const params = extractParams('/users/:id');

            expect(params).toBeNull();
        });

        it('should handle single parameter', () => {
            Object.defineProperty(window.location, 'pathname', {
                value: '/user/42',
                configurable: true
            });

            const params = extractParams('/user/:id');

            expect(params).toEqual({ id: '42' });
        });

        it('should handle mixed static and dynamic segments', () => {
            Object.defineProperty(window.location, 'pathname', {
                value: '/api/users/123/posts/456',
                configurable: true
            });

            const params = extractParams('/api/users/:userId/posts/:postId');

            expect(params).toEqual({
                userId: '123',
                postId: '456'
            });
        });

        it('should return empty object for pattern with no params', () => {
            Object.defineProperty(window.location, 'pathname', {
                value: '/about',
                configurable: true
            });

            const params = extractParams('/about');

            expect(params).toEqual({});
        });
    });
});
