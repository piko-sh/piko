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

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { createRenderScheduler } from "./RenderScheduler";
import { createLifecycleManager } from "./LifecycleManager";
import { createStateManager } from "./StateManager";
import { createAttributeSyncService } from "./AttributeSyncService";
import { createPropTypeRegistry } from "./PropTypeRegistry";
import type { RenderScheduler } from "./RenderScheduler";
import type { LifecycleManager } from "./LifecycleManager";
import type { StateManager } from "./StateManager";
import type { AttributeSyncService } from "./AttributeSyncService";
import { dom } from "../../vdom";
import type { VirtualNode } from "../../vdom";

describe("RenderScheduler", () => {
  let container: HTMLElement;
  let host: HTMLElement;
  let shadowRoot: ShadowRoot;
  let lifecycleManager: LifecycleManager;
  let stateManager: StateManager;
  let attributeSyncService: AttributeSyncService;
  let renderScheduler: RenderScheduler;
  let refs: Record<string, Node>;
  let renderVDOM: () => VirtualNode;
  let isConnected: boolean;
  let isInitialising: boolean;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);

    host = document.createElement("div");
    container.appendChild(host);
    shadowRoot = host.attachShadow({ mode: "open" });

    refs = {};
    isConnected = true;
    isInitialising = false;

    lifecycleManager = createLifecycleManager();

    stateManager = createStateManager({ tagName: "TEST-ELEMENT" });
    stateManager.setContext({ state: { name: "test" } });

    const propTypeRegistry = createPropTypeRegistry({
      propTypes: { name: { type: "string" } },
    });

    attributeSyncService = createAttributeSyncService({
      host,
      propTypeRegistry,
      getState: () => stateManager.getState(),
      getDefaults: () => ({}),
    });
    attributeSyncService.setInitialising(false);

    renderVDOM = vi.fn(() => dom.el("div", "root", {}));

    renderScheduler = createRenderScheduler({
      getShadowRoot: () => shadowRoot,
      isConnected: () => isConnected,
      isInitialising: () => isInitialising,
      renderVDOM,
      refs,
      lifecycleManager,
      stateManager,
      attributeSyncService,
    });
  });

  afterEach(() => {
    document.body.removeChild(container);
    vi.restoreAllMocks();
  });

  describe("initial state", () => {
    it("should not have rendered initially", () => {
      expect(renderScheduler.hasRendered()).toBe(false);
    });

    it("should not be scheduled initially", () => {
      expect(renderScheduler.isScheduled()).toBe(false);
    });

    it("should have no pending after init initially", () => {
      expect(renderScheduler.hasPendingAfterInit()).toBe(false);
    });

    it("should have null oldVDOM initially", () => {
      expect(renderScheduler.getOldVDOM()).toBeNull();
    });
  });

  describe("render()", () => {
    it("should call renderVDOM", () => {
      renderScheduler.render();

      expect(renderVDOM).toHaveBeenCalledTimes(1);
    });

    it("should execute beforeRender lifecycle", () => {
      const beforeRender = vi.fn();
      lifecycleManager.onBeforeRender(beforeRender);

      renderScheduler.render();

      expect(beforeRender).toHaveBeenCalled();
    });

    it("should mark as rendered", () => {
      renderScheduler.render();

      expect(renderScheduler.hasRendered()).toBe(true);
    });

    it("should update oldVDOM", () => {
      renderScheduler.render();

      expect(renderScheduler.getOldVDOM()).not.toBeNull();
    });

    it("should not render when not connected", () => {
      isConnected = false;

      renderScheduler.render();

      expect(renderVDOM).not.toHaveBeenCalled();
    });

    it("should not render when shadow root is null", () => {
      renderScheduler = createRenderScheduler({
        getShadowRoot: () => null,
        isConnected: () => true,
        isInitialising: () => false,
        renderVDOM,
        refs,
        lifecycleManager,
        stateManager,
        attributeSyncService,
      });

      renderScheduler.render();

      expect(renderVDOM).not.toHaveBeenCalled();
    });

    it("should handle changed properties", async () => {
      stateManager.recordChange("name");

      renderScheduler.render();

      expect(stateManager.getChangedProps().size).toBe(0);
    });

    it("should execute onUpdated with changed props", () => {
      const onUpdated = vi.fn();
      lifecycleManager.onUpdated(onUpdated);

      stateManager.recordChange("name");
      renderScheduler.render();

      expect(onUpdated).toHaveBeenCalledWith(expect.any(Set));
      expect(onUpdated.mock.calls[0][0].has("name")).toBe(true);
    });

    it("should execute afterRender on next frame", async () => {
      const afterRender = vi.fn();
      lifecycleManager.onAfterRender(afterRender);

      renderScheduler.render();

      expect(afterRender).not.toHaveBeenCalled();

      await new Promise((resolve) => requestAnimationFrame(resolve));

      expect(afterRender).toHaveBeenCalled();
    });
  });

  describe("scheduleRender()", () => {
    it("should mark as scheduled", () => {
      renderScheduler.scheduleRender();

      expect(renderScheduler.isScheduled()).toBe(true);
    });

    it("should set pending when initialising", () => {
      isInitialising = true;

      renderScheduler.scheduleRender();

      expect(renderScheduler.hasPendingAfterInit()).toBe(true);
      expect(renderScheduler.isScheduled()).toBe(false);
    });

    it("should not schedule when not connected", () => {
      isConnected = false;

      renderScheduler.scheduleRender();

      expect(renderScheduler.isScheduled()).toBe(false);
    });

    it("should batch multiple schedule calls", () => {
      renderScheduler.scheduleRender();
      renderScheduler.scheduleRender();
      renderScheduler.scheduleRender();

      expect(renderScheduler.isScheduled()).toBe(true);
    });

    it("should render on next animation frame", async () => {
      renderScheduler.scheduleRender();

      expect(renderVDOM).not.toHaveBeenCalled();

      await new Promise((resolve) => requestAnimationFrame(resolve));

      expect(renderVDOM).toHaveBeenCalledTimes(1);
    });

    it("should reset scheduled flag after render", async () => {
      renderScheduler.scheduleRender();

      await new Promise((resolve) => requestAnimationFrame(resolve));

      expect(renderScheduler.isScheduled()).toBe(false);
    });

    it("should execute onConnected after scheduled render", async () => {
      const onConnected = vi.fn();
      lifecycleManager.onConnected(onConnected);

      renderScheduler.scheduleRender();

      await new Promise((resolve) => requestAnimationFrame(resolve));

      expect(onConnected).toHaveBeenCalled();
    });
  });

  describe("setPendingAfterInit()", () => {
    it("should update pending flag", () => {
      renderScheduler.setPendingAfterInit(true);
      expect(renderScheduler.hasPendingAfterInit()).toBe(true);

      renderScheduler.setPendingAfterInit(false);
      expect(renderScheduler.hasPendingAfterInit()).toBe(false);
    });
  });

  describe("onRenderComplete callback", () => {
    it("should call onRenderComplete after render", () => {
      const onRenderComplete = vi.fn();

      renderScheduler = createRenderScheduler({
        getShadowRoot: () => shadowRoot,
        isConnected: () => isConnected,
        isInitialising: () => isInitialising,
        renderVDOM,
        refs,
        lifecycleManager,
        stateManager,
        attributeSyncService,
        onRenderComplete,
      });

      renderScheduler.render();

      expect(onRenderComplete).toHaveBeenCalledTimes(1);
    });
  });
});
