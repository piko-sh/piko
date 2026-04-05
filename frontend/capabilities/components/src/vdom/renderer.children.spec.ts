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

class ChildrenPatchTestComponent extends PPElement {
  public currentChildren: VirtualNode[] = [];

  constructor() {
    super();
  }

  public testComponentInit() {
    super.init({ state: makeReactive({}, this as any) });
  }

  override renderVDOM(): VirtualNode {
    return dom.el('div', 'children-host-key', { id: 'children-host' }, this.currentChildren);
  }

  public async setChildrenAndRender(newChildren: VirtualNode[]) {
    this.currentChildren = newChildren;
    this.render();
    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));
  }

  public getHostInnerHTML(): string {
    const hostDiv = this.shadowRoot!.querySelector('#children-host');
    return hostDiv ? hostDiv.innerHTML : '';
  }
}
customElements.define('children-patch-test-component', ChildrenPatchTestComponent);

describe('PPElement Patching - updateChildren Algorithm', () => {
  let hostElement: HTMLElement;
  let component: ChildrenPatchTestComponent;
  let childrenHost: HTMLElement;

  beforeEach(async () => {
    hostElement = document.createElement('div');
    document.body.appendChild(hostElement);

    component = document.createElement('children-patch-test-component') as ChildrenPatchTestComponent;
    component.testComponentInit();

    hostElement.appendChild(component);

    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));

    childrenHost = component.shadowRoot!.querySelector('#children-host') as HTMLElement;
    expect(childrenHost, "The #children-host div should be present after initial render").not.toBeNull();
  });

  afterEach(() => {
    if (component.parentNode) component.parentNode.removeChild(component);
    if (hostElement.parentNode) hostElement.parentNode.removeChild(hostElement);
    vi.restoreAllMocks();
  });

  const item = (id: string | number, content?: string) => dom.el('div', String(id), {key: String(id)}, [dom.txt(content ?? String(id), `txt-${id}`)]);
  const hiddenFrag = (id: string | number, children: VirtualNode[]) => dom.frag(String(id), children, { _c: false });
  const visibleFrag = (id: string | number, children: VirtualNode[]) => dom.frag(String(id), children, { _c: true });

  it('should handle no-op (same list of children)', async () => {
    const children = [item(1), item(2)];
    await component.setChildrenAndRender(children);

    const initialHtml = childrenHost.innerHTML;
    const firstChildElm = (children[0] as VirtualNode).elm;

    await component.setChildrenAndRender([...children]);
    expect(childrenHost.innerHTML).toBe(initialHtml);
    const rootVDOM = (component as any).oldVDOM as VirtualNode;
    expect((rootVDOM.children![0] as VirtualNode).elm).toBe(firstChildElm);
  });

  describe('Append Operations', () => {
    it('should append a new child to an empty list', async () => {
      await component.setChildrenAndRender([]);
      expect(childrenHost.innerHTML).toBe('');

      await component.setChildrenAndRender([item(1)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div>');
    });

    it('should append a new child to the end of a non-empty list', async () => {
      await component.setChildrenAndRender([item(1), item(2)]);
      await component.setChildrenAndRender([item(1), item(2), item(3)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div><div key="2">2</div><div key="3">3</div>');
    });

    it('should append multiple children to the end', async () => {
      await component.setChildrenAndRender([item(1)]);
      await component.setChildrenAndRender([item(1), item(2), item(3)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div><div key="2">2</div><div key="3">3</div>');
    });
  });

  describe('Prepend Operations', () => {
    it('should prepend a new child to an empty list', async () => {
      await component.setChildrenAndRender([]);
      await component.setChildrenAndRender([item(1)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div>');
    });

    it('should prepend a new child to the beginning of a non-empty list', async () => {
      await component.setChildrenAndRender([item(2), item(3)]);
      await component.setChildrenAndRender([item(1), item(2), item(3)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div><div key="2">2</div><div key="3">3</div>');
    });

    it('should prepend multiple children to the beginning', async () => {
      await component.setChildrenAndRender([item(3)]);
      await component.setChildrenAndRender([item(1), item(2), item(3)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div><div key="2">2</div><div key="3">3</div>');
    });
  });

  describe('Insert Operations', () => {
    it('should insert a child in the middle of a list', async () => {
      await component.setChildrenAndRender([item(1), item(3)]);
      await component.setChildrenAndRender([item(1), item(2), item(3)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div><div key="2">2</div><div key="3">3</div>');
    });

    it('should insert multiple children in the middle', async () => {
      await component.setChildrenAndRender([item(1), item(4)]);
      await component.setChildrenAndRender([item(1), item(2), item(3), item(4)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div><div key="2">2</div><div key="3">3</div><div key="4">4</div>');
    });
  });

  describe('Remove Operations', () => {
    it('should remove a child from the beginning', async () => {
      await component.setChildrenAndRender([item(1), item(2), item(3)]);
      await component.setChildrenAndRender([item(2), item(3)]);
      expect(childrenHost.innerHTML).toBe('<div key="2">2</div><div key="3">3</div>');
    });

    it('should remove a child from the end', async () => {
      await component.setChildrenAndRender([item(1), item(2), item(3)]);
      await component.setChildrenAndRender([item(1), item(2)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div><div key="2">2</div>');
    });

    it('should remove a child from the middle', async () => {
      await component.setChildrenAndRender([item(1), item(2), item(3)]);
      await component.setChildrenAndRender([item(1), item(3)]);
      expect(childrenHost.innerHTML).toBe('<div key="1">1</div><div key="3">3</div>');
    });

    it('should remove all children', async () => {
      await component.setChildrenAndRender([item(1), item(2)]);
      await component.setChildrenAndRender([]);
      expect(childrenHost.innerHTML).toBe('');
    });
  });

  describe('Reorder Operations', () => {
    it('should handle simple swap of two elements', async () => {
      const children1 = [item(1), item(2)];
      await component.setChildrenAndRender(children1);
      const elm1 = (children1[0] as VirtualNode).elm;
      const elm2 = (children1[1] as VirtualNode).elm;

      const children2 = [item(2), item(1)];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="2">2</div><div key="1">1</div>');

      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBe(elm2);
      expect((renderedRoot.children![1] as VirtualNode).elm).toBe(elm1);
    });

    it('should move an item from start to end', async () => {
      const children1 = [item(1), item(2), item(3)];
      await component.setChildrenAndRender(children1);
      const elm1 = (children1[0] as VirtualNode).elm;

      const children2 = [item(2), item(3), item(1)];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="2">2</div><div key="3">3</div><div key="1">1</div>');
      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![2] as VirtualNode).elm).toBe(elm1);
    });

    it('should move an item from end to start', async () => {
      const children1 = [item(1), item(2), item(3)];
      await component.setChildrenAndRender(children1);
      const elm3 = (children1[2] as VirtualNode).elm;

      const children2 = [item(3), item(1), item(2)];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="3">3</div><div key="1">1</div><div key="2">2</div>');
      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBe(elm3);
    });

    it('should handle complex shuffles (reverse list)', async () => {
      const children1 = [item(1), item(2), item(3), item(4)];
      await component.setChildrenAndRender(children1);
      const elms = children1.map(c => (c as VirtualNode).elm);

      const children2 = [item(4), item(3), item(2), item(1)];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="4">4</div><div key="3">3</div><div key="2">2</div><div key="1">1</div>');
      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBe(elms[3]);
      expect((renderedRoot.children![1] as VirtualNode).elm).toBe(elms[2]);
      expect((renderedRoot.children![2] as VirtualNode).elm).toBe(elms[1]);
      expect((renderedRoot.children![3] as VirtualNode).elm).toBe(elms[0]);
    });
  });

  describe('Mixed Operations', () => {
    it('should handle add, remove, and reorder in one update', async () => {
      const children1 = [item(1), item(2), item(3), item(4), item(5)];
      await component.setChildrenAndRender(children1);
      const elm2_orig = (children1[1] as VirtualNode).elm;
      const elm3_orig = (children1[2] as VirtualNode).elm;
      const elm4_orig = (children1[3] as VirtualNode).elm;

      const children2 = [item(0), item(4), item(3), item(6), item(2)];
      await component.setChildrenAndRender(children2);

      expect(childrenHost.innerHTML).toBe('<div key="0">0</div><div key="4">4</div><div key="3">3</div><div key="6">6</div><div key="2">2</div>');
      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBeInstanceOf(HTMLDivElement);
      expect((renderedRoot.children![1] as VirtualNode).elm).toBe(elm4_orig);
      expect((renderedRoot.children![2] as VirtualNode).elm).toBe(elm3_orig);
      expect((renderedRoot.children![3] as VirtualNode).elm).toBeInstanceOf(HTMLDivElement);
      expect((renderedRoot.children![4] as VirtualNode).elm).toBe(elm2_orig);
    });
  });

  describe('Children with Conditional Fragments', () => {
    it('should correctly patch list when a child fragment toggles visibility', async () => {
      const children1 = [
        item('A'),
        visibleFrag('frag1', [item('F1C1', 'Fragment Content 1')]),
        item('B')
      ];
      await component.setChildrenAndRender(children1);
      expect(childrenHost.innerHTML).toBe('<div key="A">A</div><div key="F1C1">Fragment Content 1</div><div key="B">B</div>');

      const children2 = [
        item('A'),
        hiddenFrag('frag1', [item('F1C1', 'Fragment Content 1')]),
        item('B')
      ];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="A">A</div><!--hidden fragment _k=frag1--><div key="B">B</div>');

      const children3 = [
        item('A'),
        visibleFrag('frag1', [item('F1C1', 'Fragment Content 1 New')]),
        item('B')
      ];
      await component.setChildrenAndRender(children3);
      expect(childrenHost.innerHTML).toBe('<div key="A">A</div><div key="F1C1">Fragment Content 1 New</div><div key="B">B</div>');
    });

    it('should correctly reorder elements around a toggling fragment', async () => {
      const children1 = [
        item('A'),
        visibleFrag('KEY_FRAG', [item('F_Content')]),
        item('B')
      ];
      await component.setChildrenAndRender(children1);
      const elmA_orig = (children1[0] as VirtualNode).elm;
      const elmB_orig = (children1[2] as VirtualNode).elm;
      expect(childrenHost.innerHTML).toBe('<div key="A">A</div><div key="F_Content">F_Content</div><div key="B">B</div>');

      const children2 = [
        item('B'),
        hiddenFrag('KEY_FRAG', [item('F_Content')]),
        item('A')
      ];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="B">B</div><!--hidden fragment _k=KEY_FRAG--><div key="A">A</div>');

      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBe(elmB_orig);
      expect((renderedRoot.children![2] as VirtualNode).elm).toBe(elmA_orig);
      expect((renderedRoot.children![1] as VirtualNode).elm).toBeInstanceOf(Comment);
    });

    it('should add new items around a toggling fragment', async () => {
      const children1 = [item('A'), hiddenFrag('KEY_FRAG', [item('F_Content')]), item('B')];
      await component.setChildrenAndRender(children1);
      expect(childrenHost.innerHTML).toBe('<div key="A">A</div><!--hidden fragment _k=KEY_FRAG--><div key="B">B</div>');

      const children2 = [
        item('X'),
        item('A'),
        visibleFrag('KEY_FRAG', [item('F_Content Updated')]),
        item('B'),
        item('Y')
      ];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="X">X</div><div key="A">A</div><div key="F_Content Updated">F_Content Updated</div><div key="B">B</div><div key="Y">Y</div>');
    });
  });

  describe('Key-Based Reconciliation Edge Cases', () => {
    it('should handle keyed lookup when no pointer match is found', async () => {
      const children1 = [item('A'), item('B'), item('C'), item('D')];
      await component.setChildrenAndRender(children1);
      const elmB = (children1[1] as VirtualNode).elm;
      const elmC = (children1[2] as VirtualNode).elm;

      const children2 = [item('C'), item('X'), item('B'), item('D')];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="C">C</div><div key="X">X</div><div key="B">B</div><div key="D">D</div>');

      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBe(elmC);
      expect((renderedRoot.children![2] as VirtualNode).elm).toBe(elmB);
    });

    it('should create a new element when a new key appears that was not in old children', async () => {
      const children1 = [item('A'), item('B')];
      await component.setChildrenAndRender(children1);

      const children2 = [item('X'), item('A'), item('B')];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="X">X</div><div key="A">A</div><div key="B">B</div>');

      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBeInstanceOf(HTMLDivElement);
    });

    it('should replace an element when key matches but tag differs', async () => {
      const children1 = [
        dom.el('div', 'shared-key', { key: 'shared-key' }, [dom.txt('old', 'txt-old-shared')]),
        item('B'),
      ];
      await component.setChildrenAndRender(children1);
      const oldElm = (children1[0] as VirtualNode).elm;

      const children2 = [
        dom.el('span', 'shared-key', { key: 'shared-key' }, [dom.txt('new', 'txt-new-shared')]),
        item('B'),
      ];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toContain('<span');
      expect(childrenHost.innerHTML).toContain('new');

      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).not.toBe(oldElm);
    });

    it('should handle replacing entire list with completely new keys', async () => {
      const children1 = [item('A'), item('B'), item('C')];
      await component.setChildrenAndRender(children1);

      const children2 = [item('X'), item('Y'), item('Z')];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="X">X</div><div key="Y">Y</div><div key="Z">Z</div>');
    });

    it('should handle interleaved new and old keys', async () => {
      const children1 = [item('A'), item('B'), item('C')];
      await component.setChildrenAndRender(children1);
      const elmA = (children1[0] as VirtualNode).elm;
      const elmC = (children1[2] as VirtualNode).elm;

      const children2 = [item('A'), item('X'), item('C'), item('Y')];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe('<div key="A">A</div><div key="X">X</div><div key="C">C</div><div key="Y">Y</div>');

      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBe(elmA);
      expect((renderedRoot.children![2] as VirtualNode).elm).toBe(elmC);
    });

    it('should handle moving a middle element to the start via keyed lookup', async () => {
      const children1 = [item('A'), item('B'), item('C'), item('D'), item('E')];
      await component.setChildrenAndRender(children1);
      const elmC = (children1[2] as VirtualNode).elm;

      const children2 = [item('C'), item('A'), item('B'), item('D'), item('E')];
      await component.setChildrenAndRender(children2);
      expect(childrenHost.innerHTML).toBe(
        '<div key="C">C</div><div key="A">A</div><div key="B">B</div><div key="D">D</div><div key="E">E</div>'
      );

      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      expect((renderedRoot.children![0] as VirtualNode).elm).toBe(elmC);
    });
  });

  describe('Event Handlers Survive Child Reconciliation', () => {
    it('should preserve click handler on reordered children', async () => {
      const handler = vi.fn();
      const children1 = [
        dom.el('button', 'btn-A', { key: 'btn-A', onClick: handler }, [dom.txt('A', 'txt-btnA')]),
        item('B'),
        item('C'),
      ];
      await component.setChildrenAndRender(children1);
      const btnElm = (children1[0] as VirtualNode).elm as HTMLElement;
      btnElm.click();
      expect(handler).toHaveBeenCalledTimes(1);

      const children2 = [
        item('B'),
        dom.el('button', 'btn-A', { key: 'btn-A', onClick: handler }, [dom.txt('A', 'txt-btnA')]),
        item('C'),
      ];
      await component.setChildrenAndRender(children2);

      const renderedRoot = (component as any).oldVDOM as VirtualNode;
      const movedBtn = (renderedRoot.children![1] as VirtualNode).elm as HTMLElement;
      expect(movedBtn).toBe(btnElm);
      movedBtn.click();
      expect(handler).toHaveBeenCalledTimes(2);
    });

    it('should swap event handler when a child is updated during reconciliation', async () => {
      const handler1 = vi.fn();
      const handler2 = vi.fn();

      const children1 = [
        dom.el('button', 'btn-swap', { key: 'btn-swap', onClick: handler1 }, [dom.txt('Click', 'txt-btnS')]),
        item('X'),
      ];
      await component.setChildrenAndRender(children1);
      const btnElm = (children1[0] as VirtualNode).elm as HTMLElement;
      btnElm.click();
      expect(handler1).toHaveBeenCalledTimes(1);

      const children2 = [
        dom.el('button', 'btn-swap', { key: 'btn-swap', onClick: handler2 }, [dom.txt('Click', 'txt-btnS')]),
        item('X'),
      ];
      await component.setChildrenAndRender(children2);
      btnElm.click();
      expect(handler1).toHaveBeenCalledTimes(1);
      expect(handler2).toHaveBeenCalledTimes(1);
    });
  });

  describe('Children with _s (Show/Hide) Toggle', () => {
    it('should hide a child element via _s without removing it from DOM', async () => {
      const children = [
        dom.el('div', 'sh-child', { key: 'sh-child', _s: true }, [dom.txt('visible', 'txt-shc')]),
        item('B'),
      ];
      await component.setChildrenAndRender(children);
      const shElm = (children[0] as VirtualNode).elm as HTMLElement;
      expect(shElm.style.display).not.toBe('none');

      const children2 = [
        dom.el('div', 'sh-child', { key: 'sh-child', _s: false }, [dom.txt('visible', 'txt-shc')]),
        item('B'),
      ];
      await component.setChildrenAndRender(children2);
      expect(shElm.style.display).toBe('none');
      expect(childrenHost.contains(shElm)).toBe(true);
    });

    it('should show a hidden child element via _s', async () => {
      const children = [
        dom.el('div', 'sh-child-2', { key: 'sh-child-2', _s: false }, [dom.txt('hidden', 'txt-shc2')]),
        item('B'),
      ];
      await component.setChildrenAndRender(children);
      const shElm = (children[0] as VirtualNode).elm as HTMLElement;
      expect(shElm.style.display).toBe('none');

      const children2 = [
        dom.el('div', 'sh-child-2', { key: 'sh-child-2', _s: true }, [dom.txt('hidden', 'txt-shc2')]),
        item('B'),
      ];
      await component.setChildrenAndRender(children2);
      expect(shElm.style.display).toBe('');
    });
  });

  describe('Children Update with innerHTML', () => {
    it('should handle child with html property update', async () => {
      const children1 = [
        dom.html('<em>initial</em>', 'html-child'),
        item('B'),
      ];
      await component.setChildrenAndRender(children1);
      const htmlElm = (children1[0] as VirtualNode).elm as HTMLElement;
      expect(htmlElm.innerHTML).toBe('<em>initial</em>');

      const children2 = [
        dom.html('<strong>updated</strong>', 'html-child'),
        item('B'),
      ];
      await component.setChildrenAndRender(children2);
      expect(htmlElm.innerHTML).toBe('<strong>updated</strong>');
    });

    it('should transition from children to innerHTML within same element', async () => {
      const children1 = [
        dom.el('div', 'ch-to-html', { key: 'ch-to-html' }, [dom.txt('text child', 'txt-ch')]),
      ];
      await component.setChildrenAndRender(children1);
      const divElm = (children1[0] as VirtualNode).elm as HTMLElement;
      expect(divElm.textContent).toBe('text child');

      const htmlVNode: VirtualNode = {
        _type: 'element',
        tag: 'div',
        key: 'ch-to-html',
        props: { key: 'ch-to-html' },
        html: '<b>html content</b>',
        children: null,
      };
      await component.setChildrenAndRender([htmlVNode]);
      expect(divElm.innerHTML).toBe('<b>html content</b>');
    });
  });
});
