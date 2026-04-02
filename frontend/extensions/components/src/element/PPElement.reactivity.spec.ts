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

import { describe, it, expect, beforeEach, afterEach, vi, Mock, MockInstance } from 'vitest';
import { PPElement, PropTypeDefinition } from '@/element';
import { dom } from '@/vdom';
import type { VirtualNode } from '@/vdom';
import { makeReactive } from '@/reactivity';

vi.mock('@/core/PPFramework', () => ({
  PPFramework: {
    navigateTo: vi.fn(),
  },
}));

interface TestState {
  message: string;
  counter: number;
  nested?: { value: string };
}

type ScheduleRenderFn = () => void;

class ReactiveTestElement extends PPElement {
  scheduleRenderSpy!: MockInstance<ScheduleRenderFn>;
  onUpdatedSpy: Mock<(changedProps: Set<string>) => void>;

  constructor() {
    super();
    this.onUpdatedSpy = vi.fn();
    this.onUpdated((changed) => this.onUpdatedSpy(changed));
  }

  static override get propTypes(): Record<string, PropTypeDefinition> | undefined {
    return {
      message: { type: 'string' },
      counter: { type: 'number' },
      nested: { type: 'object' },
    };
  }

  public testSetup(initialState: TestState) {
    const reactiveState = makeReactive(initialState, this as any);
    super.init({
      state: reactiveState as unknown as Record<string, unknown>,
      $$initialState: { ...initialState } as Record<string, unknown>,
    });
  }

  override renderVDOM(): VirtualNode {
    return dom.el('div', 'reactive-test-root', {}, [
      dom.txt(this.state?.message || '', 'msg-node'),
      dom.txt(String(this.state?.counter || 0), 'count-node')
    ]);
  }

  override scheduleRender(): void {
    super.scheduleRender();
  }
}
customElements.define('reactive-test-element', ReactiveTestElement);

describe('PPElement Reactivity', () => {
  let element: ReactiveTestElement;
  let host: HTMLElement;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
    element = document.createElement('reactive-test-element') as ReactiveTestElement;

    element.scheduleRenderSpy = vi.spyOn(element, 'scheduleRender');
  });

  afterEach(() => {
    vi.restoreAllMocks();
    if (element.parentNode) {
      element.parentNode.removeChild(element);
    }
    if (host.parentNode) {
      host.parentNode.removeChild(host);
    }
  });

  it('setState() should update properties on this.state', () => {
    element.testSetup({ message: 'Initial', counter: 0 });
    expect(element.state!.message).toBe('Initial');

    element.setState({ message: 'Updated via setState' });
    expect(element.state!.message).toBe('Updated via setState');
    expect(element.state!.counter).toBe(0);

    element.setState({ counter: 5, message: 'Also updated' });
    expect(element.state!.counter).toBe(5);
    expect(element.state!.message).toBe('Also updated');
  });

  it('setState() should trigger scheduleRender', () => {
    element.testSetup({ message: 'Hello', counter: 1 });
    host.appendChild(element);
    element.scheduleRenderSpy.mockClear();

    element.setState({ message: 'Trigger Render' });
    expect(element.scheduleRenderSpy).toHaveBeenCalledTimes(1);
  });

  it('Directly setting a state property should trigger scheduleRender via proxy', () => {
    element.testSetup({ message: 'Direct Set Test', counter: 10 });
    host.appendChild(element);
    element.scheduleRenderSpy.mockClear();

    element.state!.counter = 11;
    expect(element.scheduleRenderSpy).toHaveBeenCalledTimes(1);
    expect(element.state!.counter).toBe(11);
  });

  it('Setting a state property to the same primitive value should not trigger scheduleRender', () => {
    element.testSetup({ message: 'No Change', counter: 20 });
    host.appendChild(element);
    element.scheduleRenderSpy.mockClear();

    element.state!.message = 'No Change';
    element.state!.counter = 20;
    expect(element.scheduleRenderSpy).not.toHaveBeenCalled();
  });

  it('Setting a state property to the same object reference should trigger scheduleRender (current makeReactive behaviour)', () => {
    const obj = { value: 'test' };
    element.testSetup({ message: 'Object Test', counter: 0, nested: obj });
    host.appendChild(element);
    element.scheduleRenderSpy.mockClear();

    element.state!.nested = obj;
    expect(element.scheduleRenderSpy).toHaveBeenCalledTimes(1);
  });

  it('changedPropsSet should be populated by makeReactive and cleared after onUpdated', async () => {
    element.testSetup({ message: 'Initial Message', counter: 0 });
    element.scheduleRenderSpy.mockRestore();
    element.scheduleRenderSpy = vi.spyOn(element, 'scheduleRender');

    host.appendChild(element);

    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve,0));

    element.onUpdatedSpy.mockClear();
    (element as any).changedPropsSet.clear();

    element.state!.message = 'Changed Message';
    element.state!.counter = 1;

    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(element.onUpdatedSpy).toHaveBeenCalledTimes(1);
    const changedPropsArg = element.onUpdatedSpy.mock.calls[0][0];
    expect(changedPropsArg).toBeInstanceOf(Set);
    expect(changedPropsArg.has('message')).toBe(true);
    expect(changedPropsArg.has('counter')).toBe(true);

    expect((element as any).changedPropsSet.size).toBe(0);

    element.onUpdatedSpy.mockClear();
    (element as any).renderScheduled = false;

    element.scheduleRender();

    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(element.onUpdatedSpy).not.toHaveBeenCalled();
  });
});
