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

import { describe, expect, it } from 'vitest';
import type { PPElement as ElementType } from '@/element';
import type { dom as DomType } from '@/vdom';

describe.each(['es', 'min.es'])('shipped %s component runtime', (build) => {
  it('skips a keyed row until its emitted _memo dependency changes', async () => {
    const bundle = build === 'es'
      ? await import('../../../../internal/daemon/daemon_frontend/built/ppframework.components.es.js')
      : await import('../../../../internal/daemon/daemon_frontend/built/ppframework.components.min.es.js');
    const { PPElement, dom } = bundle as {
      PPElement: typeof ElementType;
      dom: typeof DomType;
    };
    class RowList extends PPElement {
      version = 1;
      label = 'first';
      override renderVDOM() {
        return dom.el('ul', 'list', {}, [
          dom.el('li', 'row', { _memo: this.version }, [dom.txt(this.label, 'label')]),
        ]);
      }
    }
    const tag = 'test-shipped-memo-' + build.replace('.', '-');
    customElements.define(tag, RowList);
    const list = document.createElement(tag) as RowList;
    list.init({state: {}});
    document.body.appendChild(list);
    list.render();
    expect(list.shadowRoot?.querySelector('li')?.textContent).toBe('first');
    list.label = 'buffered';
    list.render();
    expect(list.shadowRoot?.querySelector('li')?.textContent).toBe('first');
    list.version++;
    list.render();
    expect(list.shadowRoot?.querySelector('li')?.textContent).toBe('buffered');
  });

  it('keeps the emitted _model marker so a p-model checkbox stays controlled', async () => {
    const bundle = build === 'es'
      ? await import('../../../../internal/daemon/daemon_frontend/built/ppframework.components.es.js')
      : await import('../../../../internal/daemon/daemon_frontend/built/ppframework.components.min.es.js');
    const { PPElement, dom } = bundle as {
      PPElement: typeof ElementType;
      dom: typeof DomType;
    };
    class ModelBox extends PPElement {
      checked = false;
      label = 'first';
      override renderVDOM() {
        return dom.el('label', 'row', {}, [
          dom.el('input', 'flag', { type: 'checkbox', '?checked': this.checked, _model: true }),
          dom.txt(this.label, 'label'),
        ]);
      }
    }
    const tag = 'test-shipped-model-' + build.replace('.', '-');
    customElements.define(tag, ModelBox);
    const box = document.createElement(tag) as ModelBox;
    box.init({state: {}});
    document.body.appendChild(box);
    box.render();
    const input = box.shadowRoot?.querySelector('input') as HTMLInputElement;
    input.click();
    expect(input.checked).toBe(true);
    box.label = 'second';
    box.render();
    expect(input.checked).toBe(false);
    expect(input.hasAttribute('_model')).toBe(false);
  });
});
