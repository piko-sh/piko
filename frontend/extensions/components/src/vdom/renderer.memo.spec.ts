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

class MemoTestComponent extends PPElement {
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
customElements.define('memo-test-component', MemoTestComponent);

interface Row {
  id: string;
  label: string;
}

function rowVNode(row: Row, extraProps: Record<string, unknown> = {}): VirtualNode {
  return dom.el('div', row.id, { _memo: row, 'data-label': row.label, ...extraProps }, [
    dom.txt(row.label, row.id + '-t'),
  ]);
}

function listVNode(rows: VirtualNode[]): VirtualNode {
  return dom.el('div', 'list', {}, rows);
}

describe('renderer: keyed-row _memo fast path', () => {
  let host: HTMLElement;
  let component: MemoTestComponent;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
    component = document.createElement('memo-test-component') as MemoTestComponent;
    host.appendChild(component);
  });

  afterEach(() => {
    if (component.parentNode) component.parentNode.removeChild(component);
    if (host.parentNode) host.parentNode.removeChild(host);
    vi.restoreAllMocks();
  });

  const rowEls = (): HTMLElement[] =>
    Array.from(component.shadowRoot!.querySelectorAll('[data-label]'));

  it('skips the props walk when _memo deps are reference-equal', () => {
    const a: Row = { id: 'a', label: 'one' };
    component.setAndRender(listVNode([rowVNode(a)]));
    const el = rowEls()[0];
    el.setAttribute('data-label', 'hand-mutated');

    component.setAndRender(listVNode([rowVNode(a)]));
    expect(rowEls()[0].getAttribute('data-label')).toBe('hand-mutated');
  });

  it('patches normally when _memo deps differ', () => {
    const a1: Row = { id: 'a', label: 'one' };
    const a2: Row = { id: 'a', label: 'two' };
    component.setAndRender(listVNode([rowVNode(a1)]));
    component.setAndRender(listVNode([rowVNode(a2)]));
    expect(rowEls()[0].getAttribute('data-label')).toBe('two');
    expect(rowEls()[0].textContent).toBe('two');
  });

  it('unwraps reactive proxies: fresh proxy wrappers of the same raw object match', () => {
    const raw: Row = { id: 'a', label: 'one' };
    const state = makeReactive({ rows: [raw] }, component as any);
    const proxy1 = state.rows[0];
    const proxy2 = state.rows[0];
    expect(proxy1 === proxy2).toBe(false);

    component.setAndRender(listVNode([rowVNode(proxy1 as Row)]));
    rowEls()[0].setAttribute('data-label', 'hand-mutated');
    component.setAndRender(listVNode([rowVNode(proxy2 as Row)]));
    expect(rowEls()[0].getAttribute('data-label')).toBe('hand-mutated');
  });

  it('compares array deps element-wise', () => {
    const a: Row = { id: 'a', label: 'one' };
    const mk = (flag: boolean): VirtualNode =>
      listVNode([
        dom.el('div', 'a', { _memo: [a, flag], 'data-label': a.label, 'data-flag': String(flag) }, []),
      ]);
    component.setAndRender(mk(true));
    rowEls()[0].setAttribute('data-flag', 'hand');
    component.setAndRender(mk(true));
    expect(rowEls()[0].getAttribute('data-flag')).toBe('hand');
    component.setAndRender(mk(false));
    expect(rowEls()[0].getAttribute('data-flag')).toBe('false');
  });

  it('preserves element identity across keyed reorder with skips', () => {
    const a: Row = { id: 'a', label: 'one' };
    const b: Row = { id: 'b', label: 'two' };
    component.setAndRender(listVNode([rowVNode(a), rowVNode(b)]));
    const [elA, elB] = rowEls();
    component.setAndRender(listVNode([rowVNode(b), rowVNode(a)]));
    const after = rowEls();
    expect(after[0]).toBe(elB);
    expect(after[1]).toBe(elA);
  });

  it('keeps listeners live across skips and swaps them on the next real patch', () => {
    const a: Row = { id: 'a', label: 'one' };
    let calls1 = 0;
    let calls2 = 0;
    component.setAndRender(listVNode([rowVNode(a, { onClick: () => calls1++ })]));
    const el = rowEls()[0];

    component.setAndRender(listVNode([rowVNode(a, { onClick: () => calls1++ })]));
    el.dispatchEvent(new Event('click'));
    expect(calls1).toBe(1);

    const a2: Row = { id: 'a', label: 'one-b' };
    component.setAndRender(listVNode([rowVNode(a2, { onClick: () => calls2++ })]));
    el.dispatchEvent(new Event('click'));
    expect(calls1).toBe(1);
    expect(calls2).toBe(1);
  });

  it('never serialises _memo onto the DOM', () => {
    const a1: Row = { id: 'a', label: 'one' };
    const a2: Row = { id: 'a', label: 'two' };
    component.setAndRender(listVNode([rowVNode(a1)]));
    expect(rowEls()[0].hasAttribute('_memo')).toBe(false);
    component.setAndRender(listVNode([rowVNode(a2)]));
    expect(rowEls()[0].hasAttribute('_memo')).toBe(false);
  });

  it('grafts children on skip so a later change render does not duplicate DOM', () => {
    const a: Row = { id: 'a', label: 'one' };
    component.setAndRender(listVNode([rowVNode(a)]));
    component.setAndRender(listVNode([rowVNode(a)]));
    const a2: Row = { id: 'a', label: 'two' };
    component.setAndRender(listVNode([rowVNode(a2)]));
    const el = rowEls()[0];
    expect(el.textContent).toBe('two');
    expect(el.childNodes.length).toBe(1);
    expect(rowEls().length).toBe(1);
  });

  it('keeps p-ref valid inside skipped rows and cleans up on removal', () => {
    const a: Row = { id: 'a', label: 'one' };
    const withRef = (row: Row): VirtualNode =>
      dom.el('div', row.id, { _memo: row, _ref: 'rowa', 'data-label': row.label }, []);
    component.setAndRender(listVNode([withRef(a)]));
    const refs = component.refs;
    const el = rowEls()[0];
    expect(refs['rowa']).toBe(el);

    component.setAndRender(listVNode([withRef(a)]));
    expect(refs['rowa']).toBe(el);

    component.setAndRender(listVNode([]));
    expect(rowEls().length).toBe(0);
    expect(refs['rowa']).toBeUndefined();
  });
});
