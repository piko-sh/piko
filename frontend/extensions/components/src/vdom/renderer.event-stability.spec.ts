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

class EventTestComponent extends PPElement {
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
customElements.define('event-test-component', EventTestComponent);

describe('renderer: event-handler array stability', () => {
  let host: HTMLElement;
  let component: EventTestComponent;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
    component = document.createElement('event-test-component') as EventTestComponent;
    host.appendChild(component);
  });

  afterEach(() => {
    if (component.parentNode) component.parentNode.removeChild(component);
    if (host.parentNode) host.parentNode.removeChild(host);
    vi.restoreAllMocks();
  });

  it('should keep a sibling input handler attached when an earlier handler re-renders synchronously', () => {
    let modelCalls = 0;
    let userCalls = 0;

    const modelHandler = (): void => {
      modelCalls++;
      component.setAndRender(makeVNode());
    };
    const userHandler = (): void => {
      userCalls++;
    };
    const makeVNode = (): VirtualNode =>
      dom.el('input', 'inp', { onInput: [modelHandler, userHandler] });

    component.setAndRender(makeVNode());
    const input = component.shadowRoot!.querySelector('input')!;

    input.dispatchEvent(new Event('input', { bubbles: true }));

    expect(modelCalls).toBe(1);
    expect(userCalls).toBe(1);
  });

  it('should not detach or re-attach identical handler arrays across renders', () => {
    const h1 = (): void => {};
    const h2 = (): void => {};
    const makeVNode = (): VirtualNode =>
      dom.el('input', 'inp2', { onInput: [h1, h2] });

    component.setAndRender(makeVNode());
    const input = component.shadowRoot!.querySelector('input')!;
    const addSpy = vi.spyOn(input, 'addEventListener');
    const removeSpy = vi.spyOn(input, 'removeEventListener');

    component.setAndRender(makeVNode());

    expect(removeSpy).not.toHaveBeenCalled();
    expect(addSpy).not.toHaveBeenCalled();
  });
});
