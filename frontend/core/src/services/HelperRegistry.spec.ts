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

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { createHelperRegistry } from '@/services/HelperRegistry';
import type { HelperRegistry, PPHelper } from '@/services/HelperRegistry';

describe('HelperRegistry', () => {
  let registry: HelperRegistry;

  beforeEach(() => {
    registry = createHelperRegistry();
  });

  describe('register()', () => {
    it('should register a helper function', () => {
      const helper: PPHelper = vi.fn();
      registry.register('testHelper', helper);
      expect(registry.has('testHelper')).toBe(true);
    });

    it('should warn when overwriting an existing helper', () => {
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const helper1: PPHelper = vi.fn();
      const helper2: PPHelper = vi.fn();

      registry.register('testHelper', helper1);
      registry.register('testHelper', helper2);

      expect(consoleWarnSpy).toHaveBeenCalledWith(
        'HelperRegistry: Overwriting already registered helper "testHelper".'
      );

      consoleWarnSpy.mockRestore();
    });
  });

  describe('get()', () => {
    it('should return the registered helper', () => {
      const helper: PPHelper = vi.fn();
      registry.register('testHelper', helper);
      expect(registry.get('testHelper')).toBe(helper);
    });

    it('should return undefined for non-existent helper', () => {
      expect(registry.get('nonExistent')).toBeUndefined();
    });
  });

  describe('has()', () => {
    it('should return true for registered helper', () => {
      const helper: PPHelper = vi.fn();
      registry.register('testHelper', helper);
      expect(registry.has('testHelper')).toBe(true);
    });

    it('should return false for non-existent helper', () => {
      expect(registry.has('nonExistent')).toBe(false);
    });
  });

  describe('execute()', () => {
    it('should execute the registered helper with args', () => {
      const helper: PPHelper = vi.fn();
      registry.register('testHelper', helper);

      const element = document.createElement('div');
      const event = new Event('click');
      const args = ['arg1', 'arg2'];

      registry.execute('testHelper', element, event, args);

      expect(helper).toHaveBeenCalledWith(element, event, 'arg1', 'arg2');
    });

    it('should warn when executing non-existent helper', () => {
      const consoleWarnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const element = document.createElement('div');
      const event = new Event('click');

      registry.execute('nonExistent', element, event, []);

      expect(consoleWarnSpy).toHaveBeenCalledWith(
        'HelperRegistry: Unknown helper "nonExistent"'
      );

      consoleWarnSpy.mockRestore();
    });
  });
});
