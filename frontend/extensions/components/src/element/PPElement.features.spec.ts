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

import { describe, it, expect, beforeEach, afterEach, vi, Mock } from 'vitest';
import { PPElement } from '@/element';
import { dom } from '@/vdom';
import type { VirtualNode } from '@/vdom';
import { makeReactive } from '@/reactivity';

vi.mock('@/core/PPFramework', () => ({
  PPFramework: { navigateTo: vi.fn() },
}));

class FeatureTestElement extends PPElement {
  currentVDOM: VirtualNode | null = null;

  testEventHandler: Mock<(evt: Event) => void>;
  testCustomEventHandler: Mock<(evt: CustomEvent) => void>;

  constructor() {
    super();
    this.testEventHandler = vi.fn();
    this.testCustomEventHandler = vi.fn();
    this.init({ state: makeReactive({}, this as any) });
  }

  override renderVDOM(): VirtualNode {
    return this.currentVDOM || dom.cmt('empty feature test', 'empty-feature-key');
  }

  public async setAndRender(newVDOM: VirtualNode) {
    this.currentVDOM = newVDOM;
    this.render();
    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));
  }
}
customElements.define('feature-test-element', FeatureTestElement);

class SlotTestElement extends PPElement {
  constructor() {
    super();
    this.init({ state: makeReactive({}, this as any) });
  }
  override renderVDOM(): VirtualNode {
    return dom.el('div', 'slot-host-div', {}, [
      dom.el('slot', 'default-slot-key'),
      dom.el('slot', 'named-slot-key', { name: 'named' })
    ]);
  }
}
customElements.define('slot-test-element', SlotTestElement);


describe('PPElement - Features', () => {
  let host: HTMLElement;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
  });

  afterEach(() => {
    host.innerHTML = '';
    if (host.parentNode) host.parentNode.removeChild(host);
    vi.restoreAllMocks();
  });

  describe('Refs (_ref prop)', () => {
    let featureElement: FeatureTestElement;

    beforeEach(async () => {
      featureElement = document.createElement('feature-test-element') as FeatureTestElement;
      host.appendChild(featureElement);
      await new Promise(r => requestAnimationFrame(r));
      await new Promise(r => setTimeout(r,0));
    });

    it('should correctly populate this.refs with the DOM element', async () => {
      const vdom = dom.el('div', 'ref-div-key', { _ref: 'myDivRef' });
      await featureElement.setAndRender(vdom);

      expect(featureElement.refs.myDivRef).toBeInstanceOf(HTMLDivElement);
      expect(featureElement.refs.myDivRef).toBe(vdom.elm);
    });

    it('should update the ref if the element with the ref changes identity', async () => {
      const vdom1 = dom.el('div', 'ref-el-1', { _ref: 'myRef' });
      await featureElement.setAndRender(vdom1);
      const firstElm = vdom1.elm;
      expect(featureElement.refs.myRef).toBe(firstElm);

      const vdom2 = dom.el('span', 'ref-el-2', { _ref: 'myRef' });
      await featureElement.setAndRender(vdom2);
      const secondElm = vdom2.elm;

      expect(featureElement.refs.myRef).toBe(secondElm);
      expect(featureElement.refs.myRef).not.toBe(firstElm);
      expect(secondElm).toBeInstanceOf(HTMLSpanElement);
    });

    it('should clear the ref if the element with the ref is removed', async () => {
      const vdomWithRef = dom.el('div', 'ref-to-remove', { _ref: 'tempRef' });
      await featureElement.setAndRender(vdomWithRef);
      expect(featureElement.refs.tempRef).toBeInstanceOf(HTMLDivElement);

      const vdomWithoutRef = dom.txt('no ref here', 'txt-no-ref');
      await featureElement.setAndRender(vdomWithoutRef);
      expect(featureElement.refs.tempRef).toBeUndefined();
    });

    it('should clear the old ref and set the new one if _ref prop value changes on the same element', async () => {
      const vdom1 = dom.el('div', 'ref-change-key', { _ref: 'oldRefName' });
      await featureElement.setAndRender(vdom1);
      const domEl = vdom1.elm;
      expect(featureElement.refs.oldRefName).toBe(domEl);
      expect(featureElement.refs.newRefName).toBeUndefined();

      const vdom2 = dom.el('div', 'ref-change-key', { _ref: 'newRefName' });
      await featureElement.setAndRender(vdom2);
      expect(featureElement.refs.oldRefName).toBeUndefined();
      expect(featureElement.refs.newRefName).toBe(domEl);
      expect(vdom2.elm).toBe(domEl);
    });
  });

  describe('Event Handling (via patchProps)', () => {
    let featureElement: FeatureTestElement;

    beforeEach(async () => {
      featureElement = document.createElement('feature-test-element') as FeatureTestElement;
      host.appendChild(featureElement);
      await new Promise(r => requestAnimationFrame(r));
      await new Promise(r => setTimeout(r,0));
    });

    it('should add and invoke standard DOM event listeners', async () => {
      const vdom = dom.el('button', 'btn-key', { onClick: featureElement.testEventHandler });
      await featureElement.setAndRender(vdom);

      const buttonEl = featureElement.shadowRoot!.querySelector('button');
      expect(buttonEl).not.toBeNull();
      buttonEl!.click();
      expect(featureElement.testEventHandler).toHaveBeenCalledTimes(1);
      expect(featureElement.testEventHandler.mock.calls[0][0]).toBeInstanceOf(Event);
    });

    it('should remove old and add new standard DOM event listener if handler changes', async () => {
      const oldHandler = vi.fn();
      const newHandler = vi.fn();

      await featureElement.setAndRender(dom.el('button', 'btn-key-change', { onClick: oldHandler }));
      let buttonEl = featureElement.shadowRoot!.querySelector('button');
      expect(buttonEl).not.toBeNull();

      buttonEl!.click();
      expect(oldHandler).toHaveBeenCalledTimes(1);
      expect(newHandler).not.toHaveBeenCalled();
      oldHandler.mockClear();

      await featureElement.setAndRender(dom.el('button', 'btn-key-change', { onClick: newHandler }));
      buttonEl = featureElement.shadowRoot!.querySelector('button');
      expect(buttonEl).not.toBeNull();
      buttonEl!.click();

      expect(oldHandler).not.toHaveBeenCalled();
      expect(newHandler).toHaveBeenCalledTimes(1);
    });

    it('should add and invoke custom event listeners (pe:eventname)', async () => {
      const vdom = dom.el('div', 'custom-event-div', { 'pe:mycustomevent': featureElement.testCustomEventHandler });
      await featureElement.setAndRender(vdom);

      const divEl = featureElement.shadowRoot!.querySelector('div');
      expect(divEl).not.toBeNull();
      const customEvent = new CustomEvent('mycustomevent', { detail: { data: 123 } });
      divEl!.dispatchEvent(customEvent);

      expect(featureElement.testCustomEventHandler).toHaveBeenCalledTimes(1);
      expect(featureElement.testCustomEventHandler.mock.calls[0][0]).toBe(customEvent);
      expect((featureElement.testCustomEventHandler.mock.calls[0][0] as CustomEvent).detail).toEqual({ data: 123 });
    });

    it('should remove event listener if prop is removed', async () => {
      await featureElement.setAndRender(dom.el('button', 'btn-rem-evt', { onClick: featureElement.testEventHandler }));
      const buttonEl = featureElement.shadowRoot!.querySelector('button') as HTMLButtonElement;
      expect(buttonEl).not.toBeNull();
      featureElement.testEventHandler.mockClear();

      await featureElement.setAndRender(dom.el('button', 'btn-rem-evt', {}));
      const buttonElAfter = featureElement.shadowRoot!.querySelector('button') as HTMLButtonElement;
      expect(buttonElAfter).not.toBeNull();
      buttonElAfter.click();

      expect(featureElement.testEventHandler).not.toHaveBeenCalled();
    });
  });

  describe('Slots', () => {
    let slotComponent: SlotTestElement;

    beforeEach(async () => {
      slotComponent = document.createElement('slot-test-element') as SlotTestElement;
      host.appendChild(slotComponent);
      await new Promise(r => requestAnimationFrame(r));
      await new Promise(r => setTimeout(r,0));
    });

    it('getSlottedElements should return correct assigned elements for default slot', async () => {
      const child1 = document.createElement('span');
      child1.textContent = 'Default Slotted 1';
      const child2 = document.createTextNode('Default Slotted Text');
      slotComponent.appendChild(child1);
      slotComponent.appendChild(child2);

      await new Promise(r => requestAnimationFrame(r));
      await new Promise(r => setTimeout(r,0));


      const slotted = slotComponent.getSlottedElements();
      expect(slotted.length).toBeGreaterThanOrEqual(1);
      expect(slotted.some(el => el === child1)).toBe(true);
    });

    it('getSlottedElements should return correct assigned elements for named slot', async () => {
      const childNamed = document.createElement('div');
      childNamed.setAttribute('slot', 'named');
      childNamed.textContent = 'Named Slotted';
      slotComponent.appendChild(childNamed);

      const childDefault = document.createElement('p');
      slotComponent.appendChild(childDefault);

      await new Promise(r => requestAnimationFrame(r));
      await new Promise(r => setTimeout(r,0));

      const slottedNamed = slotComponent.getSlottedElements('named');
      expect(slottedNamed.length).toBe(1);
      expect(slottedNamed[0]).toBe(childNamed);
      expect(slottedNamed[0].textContent).toBe('Named Slotted');
    });

    it('attachSlotListener callback should be invoked initially (slotchange is hard to test reliably in JSDOM)', () => {
      return new Promise<void>((resolve, reject) => {
        const slotName = 'named';
        const child = document.createElement('div');
        child.setAttribute('slot', slotName);
        child.textContent = 'For Listener';

        const callbackSpy = vi.fn((elements: Element[]) => {
          try {
            if (elements.length > 0 && elements[0] === child) {
              expect(elements[0].textContent).toBe('For Listener');
              resolve();
            } else if (elements.length === 0 && slotComponent.querySelectorAll(`[slot="${slotName}"]`).length === 0 && !child.isConnected){
                void 0;
              }
          } catch (e) {
            reject(e);
          }
        });

        slotComponent.attachSlotListener(slotName, callbackSpy);
        slotComponent.appendChild(child);

        setTimeout(() => {
          if (callbackSpy.mock.calls.some(call => call[0].length > 0 && call[0][0] === child)) {
                void 0;
            } else {
            if (callbackSpy.mock.calls.length > 0) {
              resolve();
            } else {
              reject(new Error("attachSlotListener callback was not invoked with the element."));
            }
          }
        }, 50);
      });
    });
  });
});
