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
import { PPElement, PropTypeDefinition } from '@/element';
import { dom } from '@/vdom';
import type { VirtualNode } from '@/vdom';
import { makeReactive } from '@/reactivity';

vi.mock('@/core/PPFramework', () => ({
  PPFramework: {
    navigateTo: vi.fn(),
  },
}));

class TestLifecycleElement extends PPElement {
  onConnectedSpy: Mock<() => void>;
  onDisconnectedSpy: Mock<() => void>;
  onBeforeRenderSpy: Mock<() => void>;
  onUpdatedSpy: Mock<(changedProps: Set<string>) => void>;

  constructor() {
    super();
    this.onConnectedSpy = vi.fn();
    this.onDisconnectedSpy = vi.fn();
    this.onBeforeRenderSpy = vi.fn();
    this.onUpdatedSpy = vi.fn();

    this.onConnected(() => this.onConnectedSpy());
    this.onDisconnected(() => this.onDisconnectedSpy());
    this.onUpdated((changed) => this.onUpdatedSpy(changed));
  }

  static override get propTypes(): Record<string, PropTypeDefinition> | undefined {
    return {
      message: { type: 'string', default: 'Default Message' },
      count: { type: 'number', default: 0 }
    };
  }

  static override get css() {
    return ':host { border: 1px solid red; }';
  }

  override renderVDOM(): VirtualNode {
    this.onBeforeRenderSpy();
    return dom.el('div', 'root-div', {}, [
      dom.txt(this.state!.message, 'text-message'),
      dom.txt(String(this.state!.count), 'text-count'),
    ]);
  }

  public triggerStateChange(newMessage: string, newCount: number) {
    this.state!.message = newMessage;
    this.state!.count = newCount;
  }
}
customElements.define('test-lifecycle-element', TestLifecycleElement);

describe('PPElement Lifecycle & Initialisation', () => {
  let element: TestLifecycleElement;
  let host: HTMLElement;

  beforeEach(() => {
    host = document.createElement('div');
    document.body.appendChild(host);
    element = document.createElement('test-lifecycle-element') as TestLifecycleElement;
    element.init({
      state: makeReactive({
        message: (element.constructor as typeof TestLifecycleElement).propTypes!['message'].default as string,
        count: (element.constructor as typeof TestLifecycleElement).propTypes!['count'].default as number
      }, element as any),
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
    if (element.parentNode) {
      element.parentNode.removeChild(element);
    }
    if (host.parentNode) {
      host.parentNode.removeChild(host);
    }
  });

  it('should be an instance of PPElement and HTMLElement', () => {
    expect(element).toBeInstanceOf(TestLifecycleElement);
    expect(element).toBeInstanceOf(PPElement);
    expect(element).toBeInstanceOf(HTMLElement);
  });

  it('should have Shadow DOM created and open after init', () => {
    expect(element.shadowRoot).not.toBeNull();
    expect(element.shadowRoot!.mode).toBe('open');
  });

  it('should inject reset CSS and component static CSS into Shadow DOM after init', () => {
    const styles = element.shadowRoot!.querySelectorAll('style');
    expect(styles.length).toBe(2);
    expect(styles[0].textContent).toContain('box-sizing:border-box');
    expect(styles[1].textContent).toContain(':host { border: 1px solid red; }');
  });

  it('should initialise $$ctx and state with defaults during init', () => {
    expect(element.$$ctx).toBeDefined();
    expect(element.state).toBeDefined();
    expect(element.state!.message).toBe('Default Message');
    expect(element.state!.count).toBe(0);
  });

  it('should call render and onConnected when connected to DOM (after init)', async () => {
    host.appendChild(element);

    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(element.onBeforeRenderSpy).toHaveBeenCalled();
    expect(element.shadowRoot!.innerHTML).toContain('Default Message');
    expect(element.onConnectedSpy).toHaveBeenCalled();
  });

  it('should call onDisconnected when removed from DOM', () => {
    host.appendChild(element);
    host.removeChild(element);
    expect(element.onDisconnectedSpy).toHaveBeenCalled();
  });

  it('should call onBeforeRender before renderVDOM during a render cycle', async () => {
    host.appendChild(element);
    await new Promise(resolve => requestAnimationFrame(resolve));

    element.onBeforeRenderSpy.mockClear();

    element.state!.message = 'Updated';
    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(element.onBeforeRenderSpy).toHaveBeenCalledTimes(1);
  });

  it('should call onUpdated with changed properties after state change and render', async () => {
    host.appendChild(element);
    await new Promise(resolve => requestAnimationFrame(resolve));

    element.onUpdatedSpy.mockClear();

    element.triggerStateChange('New Value', 1);

    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(element.onUpdatedSpy).toHaveBeenCalledTimes(1);
    const changedPropsArg = element.onUpdatedSpy.mock.calls[0][0];
    expect(changedPropsArg).toBeInstanceOf(Set);
    expect(changedPropsArg.has('message')).toBe(true);
    expect(changedPropsArg.has('count')).toBe(true);
    expect(element.shadowRoot!.innerHTML).toContain('New Value');
    expect(element.shadowRoot!.innerHTML).toContain('1');
  });

  it('should render when connected to DOM after init', async () => {
    const freshElement = document.createElement('test-lifecycle-element') as TestLifecycleElement;

    freshElement.init({
      state: makeReactive({ message: 'Test Message', count: 42 }, freshElement as any)
    });

    host.appendChild(freshElement);

    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(freshElement.shadowRoot!.innerHTML).toContain('Test Message');
    expect(freshElement.shadowRoot!.innerHTML).toContain('42');
    expect(freshElement.onBeforeRenderSpy).toHaveBeenCalled();
  });

  it('should call onConnected every time it is connected after initial setup', async () => {
    host.appendChild(element);
    await new Promise(resolve => requestAnimationFrame(resolve));
    expect(element.onConnectedSpy).toHaveBeenCalledTimes(1);

    host.removeChild(element);
    element.onConnectedSpy.mockClear();

    host.appendChild(element);
    await new Promise(resolve => requestAnimationFrame(resolve));

    expect(element.onConnectedSpy).toHaveBeenCalledTimes(1);
  });

  it('should call onCleanup during disconnectedCallback', () => {
    const cleanupSpy = vi.fn();
    element.onCleanup(cleanupSpy);

    host.appendChild(element);
    host.removeChild(element);

    expect(cleanupSpy).toHaveBeenCalledTimes(1);
  });

  it('should call onCleanup AFTER onDisconnected', () => {
    const callOrder: string[] = [];
    element.onDisconnected(() => callOrder.push('disconnected'));
    element.onCleanup(() => callOrder.push('cleanup'));

    host.appendChild(element);
    host.removeChild(element);

    expect(callOrder).toEqual(['disconnected', 'cleanup']);
  });

  it('should clear onCleanup array after disconnection', async () => {
    const cleanupSpy = vi.fn();
    element.onCleanup(cleanupSpy);

    host.appendChild(element);
    host.removeChild(element);
    expect(cleanupSpy).toHaveBeenCalledTimes(1);

    cleanupSpy.mockClear();

    host.appendChild(element);
    await new Promise(resolve => requestAnimationFrame(resolve));
    host.removeChild(element);
    expect(cleanupSpy).toHaveBeenCalledTimes(0);
  });

  it('should support multiple callbacks for the same lifecycle event', async () => {
    const secondConnectedSpy = vi.fn();
    const freshElement = document.createElement('test-lifecycle-element') as TestLifecycleElement;
    freshElement.onConnected(() => secondConnectedSpy());

    freshElement.init({
      state: makeReactive({ message: 'Multi', count: 0 }, freshElement as any)
    });

    host.appendChild(freshElement);
    await new Promise(resolve => requestAnimationFrame(resolve));
    await new Promise(resolve => setTimeout(resolve, 0));

    expect(freshElement.onConnectedSpy).toHaveBeenCalledTimes(1);
    expect(secondConnectedSpy).toHaveBeenCalledTimes(1);
  });
});
