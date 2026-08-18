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
import { PPElement } from '@/element';
import { dom } from '@/vdom';
import type { VirtualNode } from '@/vdom';
import { makeReactive } from '@/reactivity';

vi.mock('@/core/PPFramework', () => ({
  PPFramework: { navigateTo: vi.fn() },
}));

class ListenerTestComponent extends PPElement {
  public currentVDOM: VirtualNode | null = null;

  constructor() {
    super();
    this.init({ state: makeReactive({}, this as any) });
  }

  override renderVDOM(): VirtualNode {
    return this.currentVDOM || dom.cmt('initial empty state', 'init-cmt');
  }

  public setAndRender(newVDOM: VirtualNode): void {
    this.currentVDOM = newVDOM;
    this.render();
  }
}
customElements.define('listener-test-component', ListenerTestComponent);

describe('renderer: managed event listeners', () => {
  let host: HTMLElement;
  let component: ListenerTestComponent;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
    component = document.createElement('listener-test-component') as ListenerTestComponent;
    host.appendChild(component);
  });

  afterEach(() => {
    if (component.parentNode) component.parentNode.removeChild(component);
    if (host.parentNode) host.parentNode.removeChild(host);
    vi.restoreAllMocks();
  });

  const target = (): HTMLElement => component.shadowRoot!.querySelector('[data-t]') as HTMLElement;

  const renderRow = (props: Record<string, unknown>): void => {
    component.setAndRender(dom.el('div', 'r', { 'data-t': '1', ...props }, []));
  };

  const captureOutOfBandErrors = (body: () => void): unknown[] => {
    const thrown: unknown[] = [];
    const original = globalThis.queueMicrotask;
    globalThis.queueMicrotask = (fn: () => void) => {
      try { fn(); } catch (error) { thrown.push(error); }
    };
    try {
      body();
    } finally {
      globalThis.queueMicrotask = original;
    }
    return thrown;
  };

  describe('namespace isolation', () => {
    it('fires both p-on and p-event handlers bound to the same event name', () => {
      let onCalls = 0;
      let peCalls = 0;
      renderRow({
        onUpdate: () => { onCalls++; },
        'pe:update': () => { peCalls++; },
      });

      target().dispatchEvent(new Event('update'));

      expect({ onCalls, peCalls }).toEqual({ onCalls: 1, peCalls: 1 });
    });

    it('removes only the namespace whose prop was dropped', () => {
      let onCalls = 0;
      let peCalls = 0;
      renderRow({
        onUpdate: () => { onCalls++; },
        'pe:update': () => { peCalls++; },
      });
      renderRow({ 'pe:update': () => { peCalls++; } });

      target().dispatchEvent(new Event('update'));

      expect({ onCalls, peCalls }).toEqual({ onCalls: 0, peCalls: 1 });
    });

    it('keeps capture and bubble bindings of one event independent', () => {
      const order: string[] = [];
      renderRow({
        onClick: () => { order.push('bubble'); },
        onClick$capture: () => { order.push('capture'); },
      });

      target().dispatchEvent(new Event('click'));

      expect(order.sort()).toEqual(['bubble', 'capture']);
    });
  });

  describe('handler-array parity with per-handler DOM listeners', () => {
    it('invokes every handler in array order', () => {
      const order: number[] = [];
      renderRow({ onClick: [() => order.push(1), () => order.push(2), () => order.push(3)] });

      target().dispatchEvent(new Event('click'));

      expect(order).toEqual([1, 2, 3]);
    });

    it('does not let a throwing handler stop its siblings', () => {
      let second = 0;
      let third = 0;
      captureOutOfBandErrors(() => {
        renderRow({
          onClick: [
            () => { throw new Error('boom'); },
            () => { second++; },
            () => { third++; },
          ],
        });
        target().dispatchEvent(new Event('click'));
      });

      expect({ second, third }).toEqual({ second: 1, third: 1 });
    });

    it('re-throws a handler error out of band so it still surfaces', () => {
      const thrown = captureOutOfBandErrors(() => {
        renderRow({ onClick: () => { throw new Error('boom'); } });
        target().dispatchEvent(new Event('click'));
      });

      expect(thrown).toHaveLength(1);
      expect((thrown[0] as Error).message).toBe('boom');
    });

    it('keeps every handler running when several throw', () => {
      let survivor = 0;
      const thrown = captureOutOfBandErrors(() => {
        renderRow({
          onClick: [
            () => { throw new Error('first'); },
            () => { throw new Error('second'); },
            () => { survivor++; },
          ],
        });
        target().dispatchEvent(new Event('click'));
      });

      expect(survivor).toBe(1);
      expect(thrown.map((error) => (error as Error).message)).toEqual(['first', 'second']);
    });

    it('binds `this` to the element the listener is attached to', () => {
      let received: unknown = null;
      renderRow({ onClick: function (this: unknown) { received = this; } });
      const element = target();

      element.dispatchEvent(new Event('click'));

      expect(received).toBe(element);
    });

    it('passes the dispatched event through to the handler', () => {
      let received: Event | null = null;
      renderRow({ onClick: (event: Event) => { received = event; } });
      const event = new Event('click');

      target().dispatchEvent(event);

      expect(received).toBe(event);
    });

    it('ignores non-function entries rather than throwing', () => {
      let calls = 0;
      renderRow({ onClick: [null, undefined, () => { calls++; }] as unknown as unknown[] });

      target().dispatchEvent(new Event('click'));

      expect(calls).toBe(1);
    });
  });

  describe('listener churn', () => {
    it('registers one DOM listener and never re-registers it across re-renders', () => {
      renderRow({ onClick: () => undefined });
      const element = target();
      const addSpy = vi.spyOn(element, 'addEventListener');
      const removeSpy = vi.spyOn(element, 'removeEventListener');

      for (let i = 0; i < 5; i++) {
        renderRow({ onClick: () => undefined });
      }

      expect(addSpy).not.toHaveBeenCalled();
      expect(removeSpy).not.toHaveBeenCalled();
    });

    it('dispatches to the newest handler after a re-render', () => {
      let first = 0;
      let second = 0;
      renderRow({ onClick: () => { first++; } });
      const element = target();

      renderRow({ onClick: () => { second++; } });
      element.dispatchEvent(new Event('click'));

      expect({ first, second }).toEqual({ first: 0, second: 1 });
    });

    it('detaches the DOM listener when the prop disappears', () => {
      let calls = 0;
      renderRow({ onClick: () => { calls++; } });
      const element = target();
      const removeSpy = vi.spyOn(element, 'removeEventListener');

      renderRow({});
      element.dispatchEvent(new Event('click'));

      expect(calls).toBe(0);
      expect(removeSpy).toHaveBeenCalledTimes(1);
    });

    it('detaches when the handler becomes undefined', () => {
      let calls = 0;
      renderRow({ onClick: () => { calls++; } });
      const element = target();

      renderRow({ onClick: undefined });
      element.dispatchEvent(new Event('click'));

      expect(calls).toBe(0);
    });
  });

  describe('listener options', () => {
    it('defaults focus and blur to capture so they fire from descendants', () => {
      let focusCalls = 0;
      component.setAndRender(
        dom.el('div', 'r', { 'data-t': '1', onFocus: () => { focusCalls++; } }, [
          dom.el('input', 'i', { 'data-inner': '1' }, []),
        ])
      );
      const inner = component.shadowRoot!.querySelector('[data-inner]') as HTMLElement;

      inner.dispatchEvent(new Event('focus'));

      expect(focusCalls).toBe(1);
    });

    it('honours an explicit capture modifier when removing the listener', () => {
      let calls = 0;
      renderRow({ onClick$capture: () => { calls++; } });
      const element = target();

      renderRow({});
      element.dispatchEvent(new Event('click'));

      expect(calls).toBe(0);
    });

    it('passes passive through to addEventListener', () => {
      const element = document.createElement('div');
      host.appendChild(element);
      const addSpy = vi.spyOn(HTMLElement.prototype, 'addEventListener');

      renderRow({ 'onScroll$passive': () => undefined });

      const scrollCall = addSpy.mock.calls.find((call) => call[0] === 'scroll');
      expect(scrollCall).toBeDefined();
      expect(scrollCall?.[2]).toMatchObject({ passive: true });
    });
  });
});
