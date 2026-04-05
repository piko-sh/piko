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

import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { createLoaderUI } from '@/services/LoaderUI';
import type { LoaderUI } from '@/services/LoaderUI';

describe('LoaderUI', () => {
  let loader: LoaderUI;
  let container: HTMLDivElement;

  beforeEach(() => {
    container = document.createElement('div');
    document.body.appendChild(container);
    loader = createLoaderUI({ container });
  });

  afterEach(() => {
    loader.destroy();
    container.remove();
  });

  describe('createLoaderUI()', () => {
    it('should create a loader element in the container', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar');
      expect(loaderEl).not.toBeNull();
    });

    it('should use custom color if provided', () => {
      loader.destroy();
      loader = createLoaderUI({ container, colour: 'red' });
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;
      expect(loaderEl.style.background).toBe('red');
    });

    it('should use default color if not provided', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;
      expect(loaderEl.style.background).toBe('rgb(34, 153, 238)');
    });
  });

  describe('show()', () => {
    it('should show the loader bar', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;
      expect(loaderEl.style.display).toBe('none');

      loader.show();

      expect(loaderEl.style.display).toBe('block');
      expect(loaderEl.style.width).toBe('0%');
    });
  });

  describe('hide()', () => {
    it('should hide the loader bar after animation', async () => {
      loader.destroy();
      const customLoader = createLoaderUI({ container, fadeMs: 50 });
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      customLoader.show();
      customLoader.hide();

      expect(loaderEl.style.width).toBe('100%');

      await new Promise(r => setTimeout(r, 100));

      expect(loaderEl.style.display).toBe('none');
      expect(loaderEl.style.width).toBe('0%');

      loader = customLoader;
    });
  });

  describe('setProgress()', () => {
    it('should set the progress percentage', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.setProgress(50);

      expect(loaderEl.style.display).toBe('block');
      expect(loaderEl.style.width).toBe('50%');
    });

    it('should clamp progress to 0-100 range', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.setProgress(-10);
      expect(loaderEl.style.width).toBe('0%');

      loader.setProgress(150);
      expect(loaderEl.style.width).toBe('100%');
    });
  });

  describe('destroy()', () => {
    it('should remove the loader element from DOM', () => {
      expect(container.querySelector('#ppf-loader-bar')).not.toBeNull();

      loader.destroy();

      expect(container.querySelector('#ppf-loader-bar')).toBeNull();
    });
  });
});
