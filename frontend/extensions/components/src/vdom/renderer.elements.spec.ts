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
import { replaceElementWithTracking } from '@/vdom/renderer';

vi.mock('@/core/PPFramework', () => ({
  PPFramework: { navigateTo: vi.fn() },
}));

class PatchTestComponent extends PPElement {
  public currentVDOM: VirtualNode | null = null;

  constructor() {
    super();
    this.init({ state: makeReactive({}, this as any) });
  }

  override renderVDOM(): VirtualNode {
    return this.currentVDOM || dom.cmt('initial empty state', 'init-cmt');
  }

  public setAndRender(newVDOM: VirtualNode) {
    this.currentVDOM = newVDOM;
    this.render();
  }
}
customElements.define('patch-test-component', PatchTestComponent);


describe('PPElement Patching - Elements, Text, Comments', () => {
  let host: HTMLElement;
  let component: PatchTestComponent;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);

    component = document.createElement('patch-test-component') as PatchTestComponent;
    host.appendChild(component);
  });

  afterEach(() => {
    if (component.parentNode) component.parentNode.removeChild(component);
    if (host.parentNode) host.parentNode.removeChild(host);
    vi.restoreAllMocks();
  });

  const getRenderedContent = (shadowRoot: ShadowRoot): string => {
    let html = shadowRoot.innerHTML;
    const styleEndTags = ['</style>', '</style>'];
    for (const tag of styleEndTags) {
      const styleEndIndex = html.indexOf(tag);
      if (styleEndIndex !== -1) {
        html = html.substring(styleEndIndex + tag.length);
      } else {
        const firstStyleTagEnd = html.indexOf('</style>');
        if(firstStyleTagEnd !== -1) html = html.substring(firstStyleTagEnd + '</style>'.length);
        break;
      }
    }
    return html.trim();
  };


  describe('Creating New Nodes', () => {
    it('should create and append a new element VNode', () => {
      const vnode = dom.el('div', 'el1', { id: 'myDiv' }, [dom.txt('hello', 'txt1')]);
      component.setAndRender(vnode);

      expect(getRenderedContent(component.shadowRoot!)).toBe('<div id="myDiv">hello</div>');
      expect(vnode.elm).toBeInstanceOf(HTMLDivElement);
      expect((vnode.elm as HTMLElement).id).toBe('myDiv');
      expect((vnode.children![0] as VirtualNode).elm).toBeInstanceOf(Text);
    });

    it('should create and append a new text VNode', () => {
      const vnode = dom.txt('hello text', 'txt2');
      component.setAndRender(vnode);
      expect(getRenderedContent(component.shadowRoot!)).toBe('hello text');
      expect(vnode.elm).toBeInstanceOf(Text);
      expect(vnode.elm!.textContent).toBe('hello text');
    });

    it('should create and append a new comment VNode', () => {
      const vnode = dom.cmt('my comment', 'cmt1');
      component.setAndRender(vnode);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--my comment-->');
      expect(vnode.elm).toBeInstanceOf(Comment);
      expect(vnode.elm!.textContent).toBe('my comment');
    });
  });

  describe('Removing Nodes', () => {
    it('should remove an existing element VNode', () => {
      const vnodeToRemove = dom.el('div', 'el-to-remove');
      component.setAndRender(vnodeToRemove);
      expect(component.shadowRoot!.querySelector('div')).not.toBeNull();

      const placeholder = dom.cmt('placeholder after remove', 'cmt-placeholder');
      component.setAndRender(placeholder);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--placeholder after remove-->');
    });

    it('should remove an existing text VNode', () => {
      const vnodeToRemove = dom.txt('text to remove', 'txt-to-remove');
      component.setAndRender(vnodeToRemove);
      expect(getRenderedContent(component.shadowRoot!)).toBe('text to remove');

      const placeholder = dom.cmt('placeholder after remove text', 'cmt-placeholder-text');
      component.setAndRender(placeholder);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--placeholder after remove text-->');
    });
  });

  describe('Replacing Nodes', () => {
    it('should replace an element with another element (different key)', () => {
      const oldV = dom.el('div', 'old-div');
      component.setAndRender(oldV);

      const newV = dom.el('p', 'new-p', { id: 'para' });
      component.setAndRender(newV);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<p id="para"></p>');
      expect(newV.elm).toBeInstanceOf(HTMLParagraphElement);
    });

    it('should replace an element with another element (different tag, same key)', () => {
      const oldV = dom.el('div', 'shared-key');
      component.setAndRender(oldV);

      const newV = dom.el('p', 'shared-key', { id: 'para-shared' });
      component.setAndRender(newV);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<p id="para-shared"></p>');
      expect(newV.elm).toBeInstanceOf(HTMLParagraphElement);
    });

    it('should replace an element with a text node', () => {
      const oldV = dom.el('div', 'div-to-text');
      component.setAndRender(oldV);

      const newV = dom.txt('now text', 'text-now');
      component.setAndRender(newV);
      expect(getRenderedContent(component.shadowRoot!)).toBe('now text');
      expect(newV.elm).toBeInstanceOf(Text);
    });

    it('should replace a text node with an element', () => {
      const oldV = dom.txt('text-to-div', 'text-to-div-key');
      component.setAndRender(oldV);

      const newV = dom.el('span', 'span-now', {}, [dom.txt('inside span', 'txt-inside-span')]);
      component.setAndRender(newV);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<span>inside span</span>');
      expect(newV.elm).toBeInstanceOf(HTMLSpanElement);
    });
  });

  describe('Updating Node Content/Props', () => {
    it('should update props of an existing element', () => {
      const vnode1 = dom.el('div', 'el-props', { id: 'one', class: 'a' });
      component.setAndRender(vnode1);
      const domElFromVNode1 = vnode1.elm as HTMLElement;
      expect(domElFromVNode1.id).toBe('one');
      expect(domElFromVNode1.className).toBe('a');

      const vnode2 = dom.el('div', 'el-props', { id: 'two', 'data-val': 'test' });
      component.setAndRender(vnode2);
      expect(vnode2.elm).toBe(domElFromVNode1);
      const domElFromVNode2 = vnode2.elm as HTMLElement;
      expect(domElFromVNode2.id).toBe('two');
      expect(domElFromVNode2.className).toBe('');
      expect(domElFromVNode2.getAttribute('data-val')).toBe('test');
    });

    it('should update text content of a text VNode', () => {
      const vnode1 = dom.txt('initial text', 'txt-update');
      component.setAndRender(vnode1);
      expect(getRenderedContent(component.shadowRoot!)).toBe('initial text');
      const domNodeFromVNode1 = vnode1.elm;

      const vnode2 = dom.txt('updated text', 'txt-update');
      component.setAndRender(vnode2);
      expect(getRenderedContent(component.shadowRoot!)).toBe('updated text');
      expect(vnode2.elm).toBe(domNodeFromVNode1);
    });
  });

  describe('Conditional Elements (Visibility Toggling)', () => {
    it('should replace visible element with a comment placeholder when hidden', () => {
      const visibleVNode = dom.el('div', 'cond-el', { id: 'myCondDiv' });
      component.setAndRender(visibleVNode);
      const originalDomEl = visibleVNode.elm;

      const hiddenVNode = dom.el('div', 'cond-el', { _c: false, id: 'myCondDiv' });
      component.setAndRender(hiddenVNode);

      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--hidden node _k=cond-el-->');
      expect(hiddenVNode.elm).toBeInstanceOf(Comment);
      expect((hiddenVNode.elm as Comment).textContent).toBe('hidden node _k=cond-el');
      expect(originalDomEl?.parentNode).toBeNull();
    });

    it('should replace a comment placeholder with a visible element when shown', () => {
      const hiddenVNode = dom.el('div', 'cond-el', { _c: false, id: 'myCondDivHiddenFirst' });
      component.setAndRender(hiddenVNode);
      const placeholderElm = hiddenVNode.elm;

      const visibleVNode = dom.el('div', 'cond-el', { _c: true, id: 'myCondDivHiddenFirst' });
      component.setAndRender(visibleVNode);

      expect(getRenderedContent(component.shadowRoot!)).toBe('<div id="myCondDivHiddenFirst"></div>');
      expect(visibleVNode.elm).toBeInstanceOf(HTMLDivElement);
      expect((visibleVNode.elm as HTMLElement).id).toBe('myCondDivHiddenFirst');
      expect(placeholderElm?.parentNode).toBeNull();
    });

    it('should toggle element visibility multiple times', () => {
      const vnodeKey = 'toggle-el';

      let currentVNode = dom.el('p', vnodeKey, { id: 'toggler' }, [dom.txt('Visible', 'txt-toggle')]);
      component.setAndRender(currentVNode);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<p id="toggler">Visible</p>');
      let previousElm = currentVNode.elm;

      currentVNode = dom.el('p', vnodeKey, { _c: false, id: 'toggler' });
      component.setAndRender(currentVNode);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden node _k=${vnodeKey}-->`);
      expect(previousElm?.parentNode).toBeNull();
      previousElm = currentVNode.elm;


      currentVNode = dom.el('p', vnodeKey, { _c: true, id: 'toggler' }, [dom.txt('Visible Again', 'txt-toggle-again')]);
      component.setAndRender(currentVNode);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<p id="toggler">Visible Again</p>');
      expect(previousElm?.parentNode).toBeNull();
      previousElm = currentVNode.elm;

      currentVNode = dom.el('p', vnodeKey, { _c: false, id: 'toggler' });
      component.setAndRender(currentVNode);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden node _k=${vnodeKey}-->`);
      expect(previousElm?.parentNode).toBeNull();
    });
  });

  describe('Show/Hide (_s prop) Toggling', () => {
    it('should set display:none when _s is false', () => {
      const vnode = dom.el('div', 'show-hide-1', { _s: false, id: 'sh1' });
      component.setAndRender(vnode);

      const domEl = vnode.elm as HTMLElement;
      expect(domEl).toBeInstanceOf(HTMLDivElement);
      expect(domEl.style.display).toBe('none');
    });

    it('should not set display:none when _s is true', () => {
      const vnode = dom.el('div', 'show-hide-2', { _s: true, id: 'sh2' });
      component.setAndRender(vnode);

      const domEl = vnode.elm as HTMLElement;
      expect(domEl.style.display).not.toBe('none');
    });

    it('should toggle from visible to hidden via _s prop', () => {
      const vnode1 = dom.el('div', 'show-hide-toggle', { _s: true, id: 'sht' });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;
      expect(domEl.style.display).not.toBe('none');

      const vnode2 = dom.el('div', 'show-hide-toggle', { _s: false, id: 'sht' });
      component.setAndRender(vnode2);
      expect(vnode2.elm).toBe(domEl);
      expect(domEl.style.display).toBe('none');
    });

    it('should toggle from hidden to visible via _s prop', () => {
      const vnode1 = dom.el('div', 'show-hide-toggle-2', { _s: false, id: 'sht2' });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;
      expect(domEl.style.display).toBe('none');

      const vnode2 = dom.el('div', 'show-hide-toggle-2', { _s: true, id: 'sht2' });
      component.setAndRender(vnode2);
      expect(vnode2.elm).toBe(domEl);
      expect(domEl.style.display).toBe('');
    });

    it('should toggle _s multiple times without duplicating style rules', () => {
      const key = 'show-hide-multi';

      let vnode = dom.el('div', key, { _s: true, id: 'shm' });
      component.setAndRender(vnode);
      const domEl = vnode.elm as HTMLElement;
      expect(domEl.style.display).not.toBe('none');

      vnode = dom.el('div', key, { _s: false, id: 'shm' });
      component.setAndRender(vnode);
      expect(domEl.style.display).toBe('none');

      vnode = dom.el('div', key, { _s: true, id: 'shm' });
      component.setAndRender(vnode);
      expect(domEl.style.display).toBe('');

      vnode = dom.el('div', key, { _s: false, id: 'shm' });
      component.setAndRender(vnode);
      expect(domEl.style.display).toBe('none');
    });

    it('should default to visible when _s prop is omitted', () => {
      const vnode1 = dom.el('div', 'show-hide-default', { _s: false, id: 'shd' });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;
      expect(domEl.style.display).toBe('none');

      const vnode2 = dom.el('div', 'show-hide-default', { id: 'shd' });
      component.setAndRender(vnode2);
      expect(domEl.style.display).toBe('');
    });
  });

  describe('Event Handler Survival After Morph', () => {
    it('should preserve event handlers when patching same element', () => {
      const handler = vi.fn();
      const vnode1 = dom.el('button', 'evt-btn', { onClick: handler, id: 'evtBtn' }, [dom.txt('Click', 'txt-evt')]);
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;

      domEl.click();
      expect(handler).toHaveBeenCalledTimes(1);

      const vnode2 = dom.el('button', 'evt-btn', { onClick: handler, id: 'evtBtn' }, [dom.txt('Click Updated', 'txt-evt')]);
      component.setAndRender(vnode2);

      expect(vnode2.elm).toBe(domEl);
      domEl.click();
      expect(handler).toHaveBeenCalledTimes(2);
    });

    it('should swap event handlers when the function changes', () => {
      const handler1 = vi.fn();
      const handler2 = vi.fn();

      const vnode1 = dom.el('button', 'evt-swap', { onClick: handler1 });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;

      domEl.click();
      expect(handler1).toHaveBeenCalledTimes(1);
      expect(handler2).not.toHaveBeenCalled();

      const vnode2 = dom.el('button', 'evt-swap', { onClick: handler2 });
      component.setAndRender(vnode2);

      domEl.click();
      expect(handler1).toHaveBeenCalledTimes(1);
      expect(handler2).toHaveBeenCalledTimes(1);
    });

    it('should remove event handlers when they are absent in new props', () => {
      const handler = vi.fn();
      const vnode1 = dom.el('button', 'evt-remove', { onClick: handler });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;

      domEl.click();
      expect(handler).toHaveBeenCalledTimes(1);

      const vnode2 = dom.el('button', 'evt-remove', {});
      component.setAndRender(vnode2);

      domEl.click();
      expect(handler).toHaveBeenCalledTimes(1);
    });

    it('should support array event handlers', () => {
      const handler1 = vi.fn();
      const handler2 = vi.fn();

      const vnode1 = dom.el('button', 'evt-arr', { onClick: [handler1, handler2] });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;

      domEl.click();
      expect(handler1).toHaveBeenCalledTimes(1);
      expect(handler2).toHaveBeenCalledTimes(1);
    });

    it('should handle pe: prefixed event listeners', () => {
      const handler = vi.fn();
      const vnode1 = dom.el('div', 'pe-evt', { 'pe:click': handler });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;

      domEl.click();
      expect(handler).toHaveBeenCalledTimes(1);

      const vnode2 = dom.el('div', 'pe-evt', {});
      component.setAndRender(vnode2);

      domEl.click();
      expect(handler).toHaveBeenCalledTimes(1);
    });
  });

  describe('replaceElementWithTracking', () => {
    it('should adopt a tracked replacement element during patch', () => {
      const vnode1 = dom.el('div', 'track-el', { id: 'origTrack' }, [dom.txt('original', 'txt-orig')]);
      component.setAndRender(vnode1);
      const origElm = vnode1.elm as HTMLElement;
      expect(origElm.id).toBe('origTrack');

      const replacementSpan = document.createElement('span');
      replacementSpan.textContent = 'replaced';
      replaceElementWithTracking(origElm, replacementSpan);

      const vnode2 = dom.el('div', 'track-el', { id: 'origTrack' });
      component.setAndRender(vnode2);

      expect(vnode2.elm).toBe(replacementSpan);
    });

    it('should invalidate tracked replacement when watched props change', () => {
      const vnode1 = dom.el('div', 'track-watch', { id: 'watchTrack', src: '/old.svg' });
      component.setAndRender(vnode1);
      const origElm = vnode1.elm as HTMLElement;

      const replacementSvg = document.createElement('svg');
      replaceElementWithTracking(origElm, replacementSvg, { watchProps: ['src'] });

      const vnode2 = dom.el('div', 'track-watch', { id: 'watchTrack', src: '/new.svg' });
      component.setAndRender(vnode2);

      expect(vnode2.elm).not.toBe(replacementSvg);
      expect(vnode2.elm).toBeInstanceOf(HTMLDivElement);
    });

    it('should keep tracked replacement when watched props have not changed', () => {
      const vnode1 = dom.el('div', 'track-keep', { id: 'keepTrack', src: '/same.svg' });
      component.setAndRender(vnode1);
      const origElm = vnode1.elm as HTMLElement;

      const replacementSvg = document.createElement('svg');
      replaceElementWithTracking(origElm, replacementSvg, { watchProps: ['src'] });

      const vnode2 = dom.el('div', 'track-keep', { id: 'keepTrack', src: '/same.svg' });
      component.setAndRender(vnode2);

      expect(vnode2.elm).toBe(replacementSvg);
    });
  });

  describe('Boolean Attribute Props', () => {
    it('should set a boolean attribute when value is truthy', () => {
      const vnode = dom.el('button', 'bool-1', { '?disabled': true });
      component.setAndRender(vnode);
      const domEl = vnode.elm as HTMLElement;
      expect(domEl.hasAttribute('disabled')).toBe(true);
    });

    it('should remove a boolean attribute when value is falsy', () => {
      const vnode1 = dom.el('button', 'bool-2', { '?disabled': true });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;
      expect(domEl.hasAttribute('disabled')).toBe(true);

      const vnode2 = dom.el('button', 'bool-2', { '?disabled': false });
      component.setAndRender(vnode2);
      expect(domEl.hasAttribute('disabled')).toBe(false);
    });

    it('should remove a boolean attribute when it is absent in new props', () => {
      const vnode1 = dom.el('button', 'bool-3', { '?disabled': true });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;
      expect(domEl.hasAttribute('disabled')).toBe(true);

      const vnode2 = dom.el('button', 'bool-3', {});
      component.setAndRender(vnode2);
      expect(domEl.hasAttribute('disabled')).toBe(false);
    });
  });

  describe('innerHTML (html prop) Patching', () => {
    it('should render innerHTML from the html property', () => {
      const vnode = dom.html('<em>bold</em>', 'html-1');
      component.setAndRender(vnode);
      const domEl = vnode.elm as HTMLElement;
      expect(domEl.innerHTML).toBe('<em>bold</em>');
    });

    it('should update innerHTML when html property changes', () => {
      const vnode1 = dom.html('<em>old</em>', 'html-update');
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;
      expect(domEl.innerHTML).toBe('<em>old</em>');

      const vnode2 = dom.html('<strong>new</strong>', 'html-update');
      component.setAndRender(vnode2);
      expect(vnode2.elm).toBe(domEl);
      expect(domEl.innerHTML).toBe('<strong>new</strong>');
    });
  });

  describe('Attribute Value Edge Cases', () => {
    it('should remove an attribute when prop value is null', () => {
      const vnode1 = dom.el('div', 'attr-null', { 'data-info': 'present' });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;
      expect(domEl.getAttribute('data-info')).toBe('present');

      const vnode2 = dom.el('div', 'attr-null', { 'data-info': null });
      component.setAndRender(vnode2);
      expect(domEl.hasAttribute('data-info')).toBe(false);
    });

    it('should remove an attribute when prop value is false', () => {
      const vnode1 = dom.el('div', 'attr-false', { 'data-active': 'yes' });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLElement;
      expect(domEl.getAttribute('data-active')).toBe('yes');

      const vnode2 = dom.el('div', 'attr-false', { 'data-active': false });
      component.setAndRender(vnode2);
      expect(domEl.hasAttribute('data-active')).toBe(false);
    });

    it('should serialise object prop values as JSON', () => {
      const objVal = { foo: 'bar', num: 42 };
      const vnode = dom.el('div', 'attr-obj', { 'data-config': objVal });
      component.setAndRender(vnode);
      const domEl = vnode.elm as HTMLElement;
      expect(domEl.getAttribute('data-config')).toBe(JSON.stringify(objVal));
    });

    it('should set the value property on input elements', () => {
      const vnode1 = dom.el('input', 'input-val', { value: 'hello' });
      component.setAndRender(vnode1);
      const domEl = vnode1.elm as HTMLInputElement;
      expect(domEl.value).toBe('hello');

      const vnode2 = dom.el('input', 'input-val', { value: 'world' });
      component.setAndRender(vnode2);
      expect(domEl.value).toBe('world');
    });
  });

  describe('Element Ref (_ref prop)', () => {
    it('should register an element ref during render', () => {
      const vnode = dom.el('div', 'ref-el', { _ref: 'myDiv', id: 'refDiv' });
      component.setAndRender(vnode);

      expect(component.refs['myDiv']).toBe(vnode.elm);
    });

    it('should update ref when ref name changes', () => {
      const vnode1 = dom.el('div', 'ref-change', { _ref: 'oldRef' });
      component.setAndRender(vnode1);
      expect(component.refs['oldRef']).toBe(vnode1.elm);

      const vnode2 = dom.el('div', 'ref-change', { _ref: 'newRef' });
      component.setAndRender(vnode2);
      expect(component.refs['oldRef']).toBeUndefined();
      expect(component.refs['newRef']).toBe(vnode2.elm);
    });

    it('should remove ref when _ref prop is absent in new props', () => {
      const vnode1 = dom.el('div', 'ref-remove', { _ref: 'toRemove' });
      component.setAndRender(vnode1);
      expect(component.refs['toRemove']).toBe(vnode1.elm);

      const vnode2 = dom.el('div', 'ref-remove', {});
      component.setAndRender(vnode2);
      expect(component.refs['toRemove']).toBeUndefined();
    });
  });

  describe('SVG Namespace Handling', () => {
    const SVG_NS = 'http://www.w3.org/2000/svg';

    it('should create an svg element with the SVG namespace', () => {
      const vnode = dom.el('svg', 'svg-1', { viewBox: '0 0 24 24' });
      component.setAndRender(vnode);

      const domEl = vnode.elm as Element;
      expect(domEl.namespaceURI).toBe(SVG_NS);
      expect(domEl.tagName.toLowerCase()).toBe('svg');
    });

    it('should create SVG child elements with the SVG namespace', () => {
      const vnode = dom.el('svg', 'svg-2', { viewBox: '0 0 48 48' }, [
        dom.el('circle', 'c1', { cx: '24', cy: '24', r: '20', fill: 'none', 'stroke-width': '4' }),
        dom.el('rect', 'r1', { x: '0', y: '0', width: '48', height: '48' }),
      ]);
      component.setAndRender(vnode);

      const svgEl = vnode.elm as Element;
      expect(svgEl.namespaceURI).toBe(SVG_NS);

      const circle = svgEl.querySelector('circle');
      expect(circle).not.toBeNull();
      expect(circle!.namespaceURI).toBe(SVG_NS);

      const rect = svgEl.querySelector('rect');
      expect(rect).not.toBeNull();
      expect(rect!.namespaceURI).toBe(SVG_NS);
    });

    it('should propagate SVG namespace to deeply nested children', () => {
      const vnode = dom.el('svg', 'svg-3', { viewBox: '0 0 24 24' }, [
        dom.el('g', 'g1', { transform: 'translate(0,0)' }, [
          dom.el('path', 'p1', { d: 'M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z', fill: 'currentColor' }),
        ]),
      ]);
      component.setAndRender(vnode);

      const svgEl = vnode.elm as Element;
      const g = svgEl.querySelector('g');
      expect(g).not.toBeNull();
      expect(g!.namespaceURI).toBe(SVG_NS);

      const path = svgEl.querySelector('path');
      expect(path).not.toBeNull();
      expect(path!.namespaceURI).toBe(SVG_NS);
    });

    it('should NOT create regular HTML elements with SVG namespace', () => {
      const vnode = dom.el('div', 'html-1', {}, [
        dom.el('span', 'span-1', {}, [dom.txt('hello', 'txt-1')]),
      ]);
      component.setAndRender(vnode);

      const divEl = vnode.elm as Element;
      expect(divEl.namespaceURI).toBe('http://www.w3.org/1999/xhtml');
      const span = divEl.querySelector('span');
      expect(span!.namespaceURI).toBe('http://www.w3.org/1999/xhtml');
    });

    it('should handle SVG inside a regular HTML container', () => {
      const vnode = dom.el('div', 'wrapper', {}, [
        dom.el('svg', 'inner-svg', { viewBox: '0 0 24 24' }, [
          dom.el('circle', 'inner-c', { cx: '12', cy: '12', r: '10' }),
        ]),
      ]);
      component.setAndRender(vnode);

      const divEl = vnode.elm as Element;
      expect(divEl.namespaceURI).toBe('http://www.w3.org/1999/xhtml');

      const svg = divEl.querySelector('svg');
      expect(svg).not.toBeNull();
      expect(svg!.namespaceURI).toBe(SVG_NS);

      const circle = divEl.querySelector('circle');
      expect(circle).not.toBeNull();
      expect(circle!.namespaceURI).toBe(SVG_NS);
    });

    it('should create SVG with correct namespace when nested inside a button via fragment', () => {
      const vnode = dom.el('button', 'btn', { class: 'toggle' },
        dom.el('svg', 'btn-svg', { viewBox: '0 0 24 24', width: '22', height: '22', fill: 'none', stroke: 'currentColor' },
          dom.frag('btn-svg_f', [
            dom.el('path', 'p1', { d: 'M7 8l-4 4 4 4' }),
            dom.el('path', 'p2', { d: 'M17 8l4 4-4 4' }),
            dom.el('line', 'l1', { x1: '14', y1: '4', x2: '10', y2: '20' }),
          ])
        )
      );
      component.setAndRender(vnode);

      const btn = vnode.elm as Element;
      expect(btn.namespaceURI).toBe('http://www.w3.org/1999/xhtml');

      const svg = btn.querySelector('svg');
      expect(svg).not.toBeNull();
      expect(svg!.namespaceURI).toBe(SVG_NS);

      const path = btn.querySelector('path');
      expect(path).not.toBeNull();
      expect(path!.namespaceURI).toBe(SVG_NS);

      const line = btn.querySelector('line');
      expect(line).not.toBeNull();
      expect(line!.namespaceURI).toBe(SVG_NS);
    });

    it('should create SVG with correct namespace when deeply nested in HTML elements', () => {
      const vnode = dom.el('div', 'outer', {}, [
        dom.el('nav', 'nav', {}, [
          dom.el('button', 'deep-btn', {}, [
            dom.el('svg', 'deep-svg', { viewBox: '0 0 24 24' }, [
              dom.el('circle', 'deep-c', { cx: '12', cy: '12', r: '10' }),
            ]),
          ]),
        ]),
      ]);
      component.setAndRender(vnode);

      const svg = (vnode.elm as Element).querySelector('svg');
      expect(svg).not.toBeNull();
      expect(svg!.namespaceURI).toBe(SVG_NS);

      const circle = (vnode.elm as Element).querySelector('circle');
      expect(circle).not.toBeNull();
      expect(circle!.namespaceURI).toBe(SVG_NS);
    });

    it('should render SVG with stroke-dasharray and stroke-dashoffset attributes', () => {
      const circumference = String(2 * Math.PI * 20);
      const vnode = dom.el('svg', 'svg-progress', { viewBox: '0 0 48 48' }, [
        dom.el('circle', 'bg', { cx: '24', cy: '24', r: '20', fill: 'none', 'stroke-width': '4', stroke: '#e6e0e9' }),
        dom.el('circle', 'indicator', {
          cx: '24', cy: '24', r: '20', fill: 'none',
          'stroke-width': '4', stroke: '#6750a4',
          'stroke-linecap': 'round',
          'stroke-dasharray': circumference,
          'stroke-dashoffset': String(parseFloat(circumference) * 0.5),
        }),
      ]);
      component.setAndRender(vnode);

      const indicator = (vnode.elm as Element).querySelector('circle:last-child') as Element;
      expect(indicator.namespaceURI).toBe(SVG_NS);
      expect(indicator.getAttribute('stroke-dasharray')).toBe(circumference);
    });
  });

  describe('Comment VNode Updates', () => {
    it('should update comment text content on patch', () => {
      const vnode1 = dom.cmt('original comment', 'cmt-update');
      component.setAndRender(vnode1);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--original comment-->');

      const vnode2 = dom.cmt('updated comment', 'cmt-update');
      component.setAndRender(vnode2);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--updated comment-->');
      expect(vnode2.elm).toBeInstanceOf(Comment);
      expect(vnode2.elm!.textContent).toBe('updated comment');
    });
  });
});
