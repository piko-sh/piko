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
import { createLoaderUI } from '@/services/LoaderUI';
import type { LoaderUI } from '@/services/LoaderUI';

const TICK_MS = 250;
const widthOf = (el: HTMLElement) => parseFloat(el.style.width);

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
    vi.useRealTimers();
  });

  describe('createLoaderUI()', () => {
    it('should create a loader element in the container', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar');
      expect(loaderEl).not.toBeNull();
    });

    it('should use custom colour if provided', () => {
      loader.destroy();
      loader = createLoaderUI({ container, colour: 'red' });
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;
      expect(loaderEl.style.background).toBe('red');
    });

    it('should use default colour if not provided', () => {
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

    it('should be monotonic - a lower value is ignored after a higher one', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.setProgress(75);
      expect(loaderEl.style.width).toBe('75%');

      loader.setProgress(40);
      expect(loaderEl.style.width).toBe('75%');
    });

    it('should reset to zero on a fresh show()', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.setProgress(80);
      expect(loaderEl.style.width).toBe('80%');

      loader.show();
      expect(loaderEl.style.width).toBe('0%');
    });

    it('should stop the trickle so explicit progress takes over', () => {
      vi.useFakeTimers();
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      vi.advanceTimersByTime(TICK_MS * 2);
      const trickled = widthOf(loaderEl);
      expect(trickled).toBeGreaterThan(0);

      loader.setProgress(80);
      expect(loaderEl.style.width).toBe('80%');
      
      vi.advanceTimersByTime(TICK_MS * 5);
      expect(loaderEl.style.width).toBe('80%');
    });
  });

  describe('destroy()', () => {
    it('should remove the loader element from DOM', () => {
      expect(container.querySelector('#ppf-loader-bar')).not.toBeNull();

      loader.destroy();

      expect(container.querySelector('#ppf-loader-bar')).toBeNull();
    });

    it('should clear the trickle timer so no ticks fire after destroy', () => {
      vi.useFakeTimers();
      loader.show();
      loader.destroy();
      vi.advanceTimersByTime(TICK_MS * 10);

      loader = createLoaderUI({ container });
      expect(container.querySelector('#ppf-loader-bar')).not.toBeNull();
    });
  });

  describe('trickle (TTFB phase)', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    it('should advance the bar asymptotically toward 50% on each tick', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      expect(widthOf(loaderEl)).toBe(0);

      vi.advanceTimersByTime(TICK_MS);
      const after1 = widthOf(loaderEl);
      expect(after1).toBeGreaterThan(0);
      expect(after1).toBeLessThan(50);

      vi.advanceTimersByTime(TICK_MS);
      const after2 = widthOf(loaderEl);
      expect(after2).toBeGreaterThan(after1);
      expect(after2).toBeLessThan(50);
    });

    it('should never reach 50% no matter how many ticks', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      vi.advanceTimersByTime(TICK_MS * 200);

      expect(widthOf(loaderEl)).toBeLessThan(50);
    });
  });

  describe('headersReceived()', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    it('should snap the bar to at least 50% on first call', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      vi.advanceTimersByTime(TICK_MS);
      expect(widthOf(loaderEl)).toBeLessThan(50);

      loader.headersReceived();

      expect(widthOf(loaderEl)).toBeGreaterThanOrEqual(50);
    });

    it('should be a no-op if called before show()', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.headersReceived();

      expect(loaderEl.style.display).toBe('none');
      expect(widthOf(loaderEl) || 0).toBe(0);
    });

    it('should be idempotent - second call does not re-snap', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      loader.headersReceived();
      vi.advanceTimersByTime(TICK_MS * 3);
      const beforeSecondCall = widthOf(loaderEl);

      loader.headersReceived();

      expect(widthOf(loaderEl)).toBe(beforeSecondCall);
    });

    it('should switch trickle ceiling from 50% to 90%', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      loader.headersReceived();
      vi.advanceTimersByTime(TICK_MS * 200);

      const finalWidth = widthOf(loaderEl);
      expect(finalWidth).toBeGreaterThan(50);
      expect(finalWidth).toBeLessThan(90);
    });
  });

  describe('setProgress() implicit transition', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    it('should cleanly take over from a running trickle', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      vi.advanceTimersByTime(TICK_MS * 2);

      loader.setProgress(75);

      expect(widthOf(loaderEl)).toBe(75);

      vi.advanceTimersByTime(TICK_MS * 50);
      expect(widthOf(loaderEl)).toBe(75);
    });
  });

  describe('hide() interaction with trickle', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    it('should snap to 100% from any trickle position and then fade', () => {
      loader.destroy();
      const customLoader = createLoaderUI({ container, fadeMs: 50 });
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      customLoader.show();
      vi.advanceTimersByTime(TICK_MS * 4);
      customLoader.hide();

      expect(loaderEl.style.width).toBe('100%');

      vi.advanceTimersByTime(60);

      expect(loaderEl.style.display).toBe('none');
      expect(loaderEl.style.width).toBe('0%');

      loader = customLoader;
    });

    it('should stop the trickle timer so no ticks fire after hide', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      vi.advanceTimersByTime(TICK_MS);
      loader.hide();

      const widthAfterHide = loaderEl.style.width;
      vi.advanceTimersByTime(TICK_MS * 5);

      expect(loaderEl.style.width === widthAfterHide || loaderEl.style.width === '0%').toBe(true);
    });
  });

  describe('trickle: false opt-out', () => {
    it('should restore stateless behaviour - show() draws zero, no timer fires', () => {
      vi.useFakeTimers();
      loader.destroy();
      const stateless = createLoaderUI({ container, trickle: false });
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      stateless.show();
      expect(loaderEl.style.width).toBe('0%');

      vi.advanceTimersByTime(TICK_MS * 10);

      expect(loaderEl.style.width).toBe('0%');

      loader = stateless;
    });

    it('should still honour explicit setProgress when trickle is off', () => {
      loader.destroy();
      const stateless = createLoaderUI({ container, trickle: false });
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      stateless.show();
      stateless.setProgress(60);

      expect(loaderEl.style.width).toBe('60%');

      loader = stateless;
    });
  });

  describe('show() called twice resets cleanly', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    it('should reset displayPercent to 0 on a second show()', () => {
      const loaderEl = container.querySelector('#ppf-loader-bar') as HTMLElement;

      loader.show();
      loader.headersReceived();
      vi.advanceTimersByTime(TICK_MS * 5);
      expect(widthOf(loaderEl)).toBeGreaterThan(50);

      loader.show();

      expect(widthOf(loaderEl)).toBe(0);
    });
  });
});
