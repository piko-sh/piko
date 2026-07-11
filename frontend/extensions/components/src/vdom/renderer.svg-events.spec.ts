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

const SVG_NS = 'http://www.w3.org/2000/svg';

class SvgEventTestComponent extends PPElement {
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
customElements.define('svg-event-test-component', SvgEventTestComponent);

interface Item {
  id: string;
}

function buildSvgForVNode(items: Item[], onPick: (item: Item) => void): VirtualNode {
  return dom.el('svg', 'svg-root', { viewBox: '0 0 100 100' }, [
    dom.frag('gfor', items.map((item, index) =>
      dom.el('g', `g-${item.id}`, {}, [
        dom.el('rect', `rect-${item.id}`, {
          onClick: (): void => { onPick(item); },
          x: String(index * 10), y: '0', width: '10', height: '10',
        }),
      ]),
    )),
  ]);
}

describe('renderer: p-on handlers on p-for svg children', () => {
  let host: HTMLElement;
  let component: SvgEventTestComponent;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
    component = document.createElement('svg-event-test-component') as SvgEventTestComponent;
    host.appendChild(component);
  });

  afterEach(() => {
    if (component.parentNode) component.parentNode.removeChild(component);
    if (host.parentNode) host.parentNode.removeChild(host);
    vi.restoreAllMocks();
  });

  it('attaches a p-on click handler to a p-for svg child created on the initial render', () => {
    const picked: string[] = [];
    const onPick = (item: Item): void => { picked.push(item.id); };

    component.setAndRender(buildSvgForVNode([{ id: 'x' }, { id: 'y' }], onPick));

    const rects = component.shadowRoot!.querySelectorAll('rect');
    expect(rects.length).toBe(2);
    expect(rects[0].namespaceURI).toBe(SVG_NS);

    rects[1].dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(picked).toEqual(['y']);
  });

  it('attaches a p-on click handler to a p-for svg child inserted on a later render', () => {
    const picked: string[] = [];
    const onPick = (item: Item): void => { picked.push(item.id); };

    component.setAndRender(buildSvgForVNode([], onPick));
    expect(component.shadowRoot!.querySelector('rect')).toBeNull();

    component.setAndRender(buildSvgForVNode([{ id: 'a' }, { id: 'b' }], onPick));

    const rect = component.shadowRoot!.querySelector('rect');
    expect(rect).not.toBeNull();
    expect(rect!.namespaceURI).toBe(SVG_NS);

    rect!.dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(picked).toEqual(['a']);
  });

  it('attaches a working handler to an svg child appended to an existing p-for list', () => {
    const picked: string[] = [];
    const onPick = (item: Item): void => { picked.push(item.id); };

    component.setAndRender(buildSvgForVNode([{ id: 'a' }], onPick));
    component.setAndRender(buildSvgForVNode([{ id: 'a' }, { id: 'b' }], onPick));

    const rects = component.shadowRoot!.querySelectorAll('rect');
    expect(rects.length).toBe(2);
    expect(rects[1].namespaceURI).toBe(SVG_NS);

    rects[1].dispatchEvent(new MouseEvent('click', { bubbles: true }));

    expect(picked).toEqual(['b']);
  });
});
