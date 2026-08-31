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
import { addFragmentQuery, buildRemoteUrl, isSameDomain, isSameOriginUrl } from '@/core/URLUtils';

describe('URLUtils', () => {
  let originalLocation: Location;

  const setPageUrl = (href: string): void => {
    const parsed = new URL(href);
    vi.stubGlobal('location', {
      href: parsed.href,
      origin: parsed.origin,
      hostname: parsed.hostname,
      pathname: parsed.pathname,
      search: parsed.search,
    });
  };

  const setBaseURI = (value: string): void => {
    Object.defineProperty(document, 'baseURI', { value, configurable: true });
  };

  beforeEach(() => {
    originalLocation = { ...window.location };
    vi.stubGlobal('location', {
      href: 'http://localhost:3000/',
      origin: 'http://localhost:3000',
      hostname: 'localhost',
      pathname: '/',
      search: '',
    });
  });

  afterEach(() => {
    vi.stubGlobal('location', originalLocation);
    Reflect.deleteProperty(document, 'baseURI');
  });

  describe('addFragmentQuery()', () => {
    it('should add _f=1 parameter to URL without query', () => {
      const result = addFragmentQuery('/page');
      expect(result).toBe('http://localhost:3000/page?_f=1');
    });

    it('should add _f=1 parameter to URL with existing query', () => {
      const result = addFragmentQuery('/page?query=abc');
      expect(result).toBe('http://localhost:3000/page?query=abc&_f=1');
    });

    it('should handle absolute URLs', () => {
      const result = addFragmentQuery('http://example.com/page?query=abc');
      expect(result).toBe('http://example.com/page?query=abc&_f=1');
    });

    it('should handle URLs with hash', () => {
      const result = addFragmentQuery('/page#section');
      expect(result).toBe('http://localhost:3000/page?_f=1#section');
    });

    it('should resolve a query-only href against the current page, not the site root', () => {
      setPageUrl('http://localhost:3000/g/core/a/site/monitor/dashboard');

      const result = addFragmentQuery('?range=7d');

      expect(result).toBe('http://localhost:3000/g/core/a/site/monitor/dashboard?range=7d&_f=1');
    });

    it('should resolve a path-relative href against the current directory', () => {
      setPageUrl('http://localhost:3000/docs/guide/intro');

      const result = addFragmentQuery('setup');

      expect(result).toBe('http://localhost:3000/docs/guide/setup?_f=1');
    });

    it('should leave absolute paths unchanged by the base', () => {
      setPageUrl('http://localhost:3000/g/core/a/site/monitor/dashboard');

      const result = addFragmentQuery('/other/page');

      expect(result).toBe('http://localhost:3000/other/page?_f=1');
    });

    it('should ignore a declared base href', () => {
      setPageUrl('http://localhost:3000/');
      setBaseURI('https://evil.example/');

      const result = addFragmentQuery('fragment');

      expect(result).toBe('http://localhost:3000/fragment?_f=1');
    });

    it('should keep a protocol-relative href cross-origin so callers can refuse it', () => {
      const result = addFragmentQuery('//evil.example/x');

      expect(result).toBe('http://evil.example/x?_f=1');
      expect(isSameOriginUrl(result)).toBe(false);
    });
  });

  describe('buildRemoteUrl()', () => {
    it('should build URL with args', () => {
      const result = buildRemoteUrl('/api/data', { id: 123, type: 'test' });
      expect(result).toBe('http://localhost:3000/api/data?_f=1&id=123&type=test');
    });

    it('should handle absolute URLs', () => {
      const result = buildRemoteUrl('http://otherserver.com/api/data', { id: 456 });
      expect(result).toBe('http://otherserver.com/api/data?_f=1&id=456');
    });

    it('should handle empty args', () => {
      const result = buildRemoteUrl('/api/data', {});
      expect(result).toBe('http://localhost:3000/api/data?_f=1');
    });

    it('should resolve a relative source against the current page', () => {
      setPageUrl('http://localhost:3000/g/core/a/site/monitor/dashboard');

      const result = buildRemoteUrl('fragment', {});

      expect(result).toBe('http://localhost:3000/g/core/a/site/monitor/fragment?_f=1');
    });

    it('should ignore a declared base href', () => {
      setPageUrl('http://localhost:3000/app/page');
      setBaseURI('https://evil.example/');

      const result = buildRemoteUrl('fragment', { id: 1 });

      expect(result).toBe('http://localhost:3000/app/fragment?_f=1&id=1');
    });
  });

  describe('isSameOriginUrl()', () => {
    it('should accept a URL on the page origin', () => {
      expect(isSameOriginUrl('http://localhost:3000/fragment')).toBe(true);
    });

    it('should reject another host, scheme or port', () => {
      expect(isSameOriginUrl('http://evil.example/fragment')).toBe(false);
      expect(isSameOriginUrl('https://localhost:3000/fragment')).toBe(false);
      expect(isSameOriginUrl('http://localhost:3001/fragment')).toBe(false);
    });

    it('should reject a value it cannot parse', () => {
      expect(isSameOriginUrl('not a url')).toBe(false);
    });
  });

  describe('isSameDomain()', () => {
    it('should return true for same domain anchor', () => {
      const anchor = document.createElement('a');
      anchor.href = 'http://localhost:3000/path';
      expect(isSameDomain(anchor)).toBe(true);
    });

    it('should return false for different domain anchor', () => {
      const anchor = document.createElement('a');
      anchor.href = 'http://example.com/path';
      expect(isSameDomain(anchor)).toBe(false);
    });

    it('should handle Location-like objects', () => {
      const locLike = { hostname: 'localhost' } as Location;
      expect(isSameDomain(locLike)).toBe(true);
    });

    it('should return false for different hostname', () => {
      const locLike = { hostname: 'other.com' } as Location;
      expect(isSameDomain(locLike)).toBe(false);
    });
  });
});
