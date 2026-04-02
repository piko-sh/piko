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

class PatchTestComponent extends PPElement {
  public currentVDOM: VirtualNode | null = null;

  constructor() {
    super();
    this.init({ state: makeReactive({}, this as any) });
  }

  override renderVDOM(): VirtualNode {
    return this.currentVDOM || dom.cmt('initial-fragment-host-state', 'init-frag-host-cmt');
  }

  public setAndRender(newVDOM: VirtualNode) {
    this.currentVDOM = newVDOM;
    this.render();
  }
}
customElements.define('fragment-patch-test-component', PatchTestComponent);


describe('PPElement Patching - Fragments', () => {
  let host: HTMLElement;
  let component: PatchTestComponent;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
    component = document.createElement('fragment-patch-test-component') as PatchTestComponent;
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

  describe('Creating New Fragments', () => {
    it('should create and append a new visible fragment with children', () => {
      const vnode = dom.frag('frag-vis-1', [
        dom.el('span', 'span-in-frag1', {}, [dom.txt('hello', 'txt1')]),
        dom.txt(' world', 'txt2-in-frag1')
      ]);
      component.setAndRender(vnode);

      expect(getRenderedContent(component.shadowRoot!)).toBe('<span>hello</span> world');
      expect(vnode.elm).toBeUndefined();
      expect((vnode.children![0] as VirtualNode).elm).toBeInstanceOf(HTMLSpanElement);
      expect((vnode.children![1] as VirtualNode).elm).toBeInstanceOf(Text);
    });

    it('should create and append a new hidden fragment as a keyed comment', () => {
      const vnode = dom.frag('frag-hid-1', [dom.txt('secret', 'txt-secret')], { _c: false });
      component.setAndRender(vnode);

      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--hidden fragment _k=frag-hid-1-->');
      expect(vnode.elm).toBeInstanceOf(Comment);
      expect((vnode.elm as Comment).textContent).toBe('hidden fragment _k=frag-hid-1');
      expect((vnode.children![0] as VirtualNode).elm).toBeUndefined();
    });

    it('should create an empty visible fragment (renders nothing)', () => {
      const vnode = dom.frag('frag-empty-vis', []);
      component.setAndRender(vnode);
      expect(getRenderedContent(component.shadowRoot!)).toBe('');
      expect(vnode.elm).toBeUndefined();
    });
  });

  describe('Removing Fragments', () => {
    it('should remove a visible fragment (its children DOM)', () => {
      const vnodeVisible = dom.frag('frag-to-remove-vis', [dom.el('p', 'p-in-frag-rem', {}, [dom.txt('content','txt-in-p-rem')])]);
      component.setAndRender(vnodeVisible);
      expect(component.shadowRoot!.querySelector('p')).not.toBeNull();

      const placeholder = dom.cmt('after fragment removed', 'cmt-after-frag-rem');
      component.setAndRender(placeholder);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--after fragment removed-->');
      expect(component.shadowRoot!.querySelector('p')).toBeNull();
    });

    it('should remove a hidden fragment (its comment placeholder)', () => {
      const vnodeHidden = dom.frag('frag-to-remove-hid', [], { _c: false });
      component.setAndRender(vnodeHidden);
      expect(getRenderedContent(component.shadowRoot!)).toContain('<!--hidden fragment _k=frag-to-remove-hid-->');

      const placeholder = dom.cmt('after hidden fragment removed', 'cmt-after-hid-frag-rem');
      component.setAndRender(placeholder);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<!--after hidden fragment removed-->');
      expect(getRenderedContent(component.shadowRoot!)).not.toContain('frag-to-remove-hid');
    });
  });

  describe('Replacing Fragments', () => {
    it('should replace a visible fragment with a non-fragment (element)', () => {
      const oldFrag = dom.frag('frag-to-elem', [dom.txt('frag content', 'txt-f2e')]);
      component.setAndRender(oldFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('frag content');

      const newElem = dom.el('div', 'elem-replaces-frag', {id: 'newDiv'});
      component.setAndRender(newElem);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<div id="newDiv"></div>');
      expect(newElem.elm).toBeInstanceOf(HTMLDivElement);
    });

    it('should replace an element with a visible fragment', () => {
      const oldElem = dom.el('div', 'elem-to-frag', {id: 'oldDiv'});
      component.setAndRender(oldElem);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<div id="oldDiv"></div>');

      const newFrag = dom.frag('frag-replaces-elem', [dom.txt('new frag content', 'txt-nf')]);
      component.setAndRender(newFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('new frag content');
      expect(newFrag.elm).toBeUndefined();
    });

    it('should replace a hidden fragment (comment) with a non-fragment (element)', () => {
      const oldFragHidden = dom.frag('frag-hid-to-elem', [], {_c: false});
      component.setAndRender(oldFragHidden);
      expect(getRenderedContent(component.shadowRoot!)).toContain('<!--hidden fragment _k=frag-hid-to-elem-->');

      const newElem = dom.el('span', 'span-replaces-hid-frag', {id: 'newSpan'});
      component.setAndRender(newElem);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<span id="newSpan"></span>');
    });

    it('should replace an element with a hidden fragment (comment)', () => {
      const oldElem = dom.el('i', 'elem-to-hid-frag');
      component.setAndRender(oldElem);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<i></i>');

      const newFragHidden = dom.frag('hid-frag-replaces-elem', [], {_c: false});
      component.setAndRender(newFragHidden);
      expect(getRenderedContent(component.shadowRoot!)).toContain('<!--hidden fragment _k=hid-frag-replaces-elem-->');
    });
  });

  describe('Conditional Fragments (Visibility Toggling)', () => {
    it('should transition a fragment from visible to hidden', () => {
      const vNodeKey = 'cond-frag-toggle';
      const visibleFrag = dom.frag(vNodeKey, [dom.el('p', 'p-in-toggle', {}, [dom.txt('Is Visible','txt-vis-t')])], { _c: true });
      component.setAndRender(visibleFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<p>Is Visible</p>');
      expect(visibleFrag.elm).toBeUndefined();

      const hiddenFrag = dom.frag(vNodeKey, [dom.el('p', 'p-in-toggle', {}, [dom.txt('Is Visible','txt-vis-t')])], { _c: false });
      component.setAndRender(hiddenFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden fragment _k=${vNodeKey}-->`);
      expect(hiddenFrag.elm).toBeInstanceOf(Comment);
    });

    it('should transition a fragment from hidden to visible', () => {
      const vNodeKey = 'cond-frag-toggle-2';
      const hiddenFrag = dom.frag(vNodeKey, [dom.el('p', 'p-in-toggle-2', {}, [dom.txt('Content Here','txt-vis-t2')])], { _c: false });
      component.setAndRender(hiddenFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden fragment _k=${vNodeKey}-->`);
      const placeholderElm = hiddenFrag.elm;

      const visibleFrag = dom.frag(vNodeKey, [dom.el('p', 'p-in-toggle-2', {}, [dom.txt('Content Here','txt-vis-t2')])], { _c: true });
      component.setAndRender(visibleFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<p>Content Here</p>');
      expect(visibleFrag.elm).toBeUndefined();
      expect(placeholderElm?.parentNode).toBeNull();
    });

    it('should toggle fragment visibility multiple times correctly', () => {
      const vNodeKey = 'cond-frag-multi-toggle';
      const children = [dom.el('b', 'b-multi', {}, [dom.txt('Bold Content', 'txt-b-multi')])];

      let currentVNode: VirtualNode = dom.frag(vNodeKey, children, { _c: true });
      component.setAndRender(currentVNode);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<b>Bold Content</b>');

      currentVNode = dom.frag(vNodeKey, children, { _c: false });
      component.setAndRender(currentVNode);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden fragment _k=${vNodeKey}-->`);
      const commentElm = currentVNode.elm;

      currentVNode = dom.frag(vNodeKey, children, { _c: true });
      component.setAndRender(currentVNode);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<b>Bold Content</b>');
      expect(commentElm?.parentNode).toBeNull();

      currentVNode = dom.frag(vNodeKey, children, { _c: false });
      component.setAndRender(currentVNode);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden fragment _k=${vNodeKey}-->`);
    });
  });

  describe('Content Change within a Visible Fragment', () => {
    it('should update children of a visible fragment (calls updateChildren)', () => {
      const key = 'frag-content-change';
      const oldChildren = [dom.txt('Old text', 'txt-old-frag-child')];
      const vnode1 = dom.frag(key, oldChildren, { _c: true });
      component.setAndRender(vnode1);
      expect(getRenderedContent(component.shadowRoot!)).toBe('Old text');
      const oldTextElm = (vnode1.children![0] as VirtualNode).elm;

      const newChildren = [
        dom.el('span', 'span-new-frag-child', {}, [dom.txt('New ', 'txt-new-1')]),
        dom.txt('content', 'txt-new-2')
      ];
      const vnode2 = dom.frag(key, newChildren, { _c: true });
      component.setAndRender(vnode2);

      expect(getRenderedContent(component.shadowRoot!)).toBe('<span>New </span>content');
      expect(oldTextElm?.parentNode).toBeNull();
      expect((vnode2.children![0] as VirtualNode).elm).toBeInstanceOf(HTMLSpanElement);
      expect((vnode2.children![1] as VirtualNode).elm).toBeInstanceOf(Text);
    });
  });

  describe('Fragment Empty-to-Content and Content-to-Empty Transitions', () => {
    it('should transition from an empty visible fragment to one with children', () => {
      const emptyFrag = dom.frag('frag-e2c', []);
      component.setAndRender(emptyFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('');

      const filledFrag = dom.frag('frag-e2c', [
        dom.el('p', 'p-e2c', {}, [dom.txt('Now has content', 'txt-e2c')]),
      ]);
      component.setAndRender(filledFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<p>Now has content</p>');
      expect((filledFrag.children![0] as VirtualNode).elm).toBeInstanceOf(HTMLParagraphElement);
    });

    it('should transition from a fragment with children to an empty one', () => {
      const filledFrag = dom.frag('frag-c2e', [
        dom.el('span', 'span-c2e', {}, [dom.txt('Will vanish', 'txt-c2e')]),
        dom.txt(' extra', 'txt-c2e-extra'),
      ]);
      component.setAndRender(filledFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<span>Will vanish</span> extra');

      const emptyFrag = dom.frag('frag-c2e', []);
      component.setAndRender(emptyFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('');
    });

    it('should cycle: empty -> content -> empty -> content', () => {
      const key = 'frag-cycle';

      let current: VirtualNode = dom.frag(key, []);
      component.setAndRender(current);
      expect(getRenderedContent(component.shadowRoot!)).toBe('');

      current = dom.frag(key, [dom.txt('Round 1', 'txt-r1')]);
      component.setAndRender(current);
      expect(getRenderedContent(component.shadowRoot!)).toBe('Round 1');

      current = dom.frag(key, []);
      component.setAndRender(current);
      expect(getRenderedContent(component.shadowRoot!)).toBe('');

      current = dom.frag(key, [dom.el('b', 'b-r2', {}, [dom.txt('Round 2', 'txt-r2')])]);
      component.setAndRender(current);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<b>Round 2</b>');
    });

    it('should transition from hidden to visible with new children', () => {
      const key = 'frag-h2vc';

      const hidden = dom.frag(key, [], { _c: false });
      component.setAndRender(hidden);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden fragment _k=${key}-->`);

      const visible = dom.frag(key, [
        dom.el('div', 'div-h2vc', { id: 'revealed' }, [dom.txt('Revealed!', 'txt-h2vc')]),
      ], { _c: true });
      component.setAndRender(visible);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<div id="revealed">Revealed!</div>');
      expect(visible.elm).toBeUndefined();
    });

    it('should transition from visible with children to hidden', () => {
      const key = 'frag-vc2h';

      const visible = dom.frag(key, [
        dom.el('div', 'div-vc2h', {}, [dom.txt('Visible content', 'txt-vc2h')]),
      ], { _c: true });
      component.setAndRender(visible);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<div>Visible content</div>');
      const childElm = (visible.children![0] as VirtualNode).elm;

      const hidden = dom.frag(key, [], { _c: false });
      component.setAndRender(hidden);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden fragment _k=${key}-->`);
      expect(hidden.elm).toBeInstanceOf(Comment);
      expect(childElm?.parentNode).toBeNull();
    });

    it('should handle hidden-to-hidden transition (same key, stays hidden)', () => {
      const key = 'frag-h2h';

      const hidden1 = dom.frag(key, [dom.txt('secret A', 'txt-ha')], { _c: false });
      component.setAndRender(hidden1);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden fragment _k=${key}-->`);
      const commentElm = hidden1.elm;

      const hidden2 = dom.frag(key, [dom.txt('secret B', 'txt-hb')], { _c: false });
      component.setAndRender(hidden2);
      expect(getRenderedContent(component.shadowRoot!)).toBe(`<!--hidden fragment _k=${key}-->`);
      expect(hidden2.elm).toBe(commentElm);
    });

    it('should handle replacing a fragment with a different-keyed fragment', () => {
      const frag1 = dom.frag('frag-key-a', [dom.txt('Fragment A', 'txt-ka')]);
      component.setAndRender(frag1);
      expect(getRenderedContent(component.shadowRoot!)).toBe('Fragment A');

      const frag2 = dom.frag('frag-key-b', [dom.el('em', 'em-kb', {}, [dom.txt('Fragment B', 'txt-kb')])]);
      component.setAndRender(frag2);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<em>Fragment B</em>');
    });

    it('should replace a visible fragment with multiple children with a single-child fragment', () => {
      const key = 'frag-multi-to-single';

      const multiFrag = dom.frag(key, [
        dom.el('span', 'sp-m1', {}, [dom.txt('A', 'txt-m1')]),
        dom.el('span', 'sp-m2', {}, [dom.txt('B', 'txt-m2')]),
        dom.el('span', 'sp-m3', {}, [dom.txt('C', 'txt-m3')]),
      ]);
      component.setAndRender(multiFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('<span>A</span><span>B</span><span>C</span>');

      const singleFrag = dom.frag(key, [dom.txt('Only one', 'txt-single')]);
      component.setAndRender(singleFrag);
      expect(getRenderedContent(component.shadowRoot!)).toBe('Only one');
    });
  });
});
