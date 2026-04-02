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
import { addFragmentQuery, buildRemoteUrl, isSameDomain } from '@/core/URLUtils';

describe('URLUtils', () => {
  let originalLocation: Location;

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
