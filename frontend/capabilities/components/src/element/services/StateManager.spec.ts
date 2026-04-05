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

import { describe, it, expect, vi, beforeEach } from "vitest";
import { createStateManager } from "./StateManager";
import type { StateManager } from "./StateManager";

describe("StateManager", () => {
  let stateManager: StateManager;

  beforeEach(() => {
    stateManager = createStateManager({ tagName: "TEST-ELEMENT" });
  });

  describe("initial state", () => {
    it("should have no state initially", () => {
      expect(stateManager.getState()).toBeUndefined();
      expect(stateManager.hasState()).toBe(false);
    });

    it("should have no context initially", () => {
      expect(stateManager.getContext()).toBeUndefined();
    });

    it("should have empty changed props initially", () => {
      expect(stateManager.getChangedProps().size).toBe(0);
    });
  });

  describe("setContext()", () => {
    it("should set the context", () => {
      const context = { state: { name: "test" } };
      stateManager.setContext(context);

      expect(stateManager.getContext()).toBe(context);
    });

    it("should make state available via getState()", () => {
      const state = { name: "test", count: 5 };
      stateManager.setContext({ state });

      expect(stateManager.getState()).toBe(state);
    });

    it("should indicate state is available via hasState()", () => {
      stateManager.setContext({ state: { name: "test" } });

      expect(stateManager.hasState()).toBe(true);
    });
  });

  describe("setState()", () => {
    it("should merge partial state with existing state", () => {
      stateManager.setContext({ state: { name: "original", count: 1 } });

      stateManager.setState({ count: 2 });

      expect(stateManager.getState()).toEqual({ name: "original", count: 2 });
    });

    it("should warn when called before state is initialised", () => {
      const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

      stateManager.setState({ name: "test" });

      expect(warnSpy).toHaveBeenCalledWith(
        expect.stringContaining("setState called before state was initialised")
      );

      warnSpy.mockRestore();
    });

    it("should not throw when called before state is initialised", () => {
      const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

      expect(() => stateManager.setState({ name: "test" })).not.toThrow();

      warnSpy.mockRestore();
    });
  });

  describe("changed props tracking", () => {
    it("should record property changes", () => {
      stateManager.recordChange("name");
      stateManager.recordChange("count");

      const changed = stateManager.getChangedProps();
      expect(changed.has("name")).toBe(true);
      expect(changed.has("count")).toBe(true);
    });

    it("should not duplicate recorded changes", () => {
      stateManager.recordChange("name");
      stateManager.recordChange("name");
      stateManager.recordChange("name");

      const changed = stateManager.getChangedProps();
      expect(changed.size).toBe(1);
    });

    it("should clear changed props and return a copy", () => {
      stateManager.recordChange("name");
      stateManager.recordChange("count");

      const cleared = stateManager.clearChangedProps();

      expect(cleared.has("name")).toBe(true);
      expect(cleared.has("count")).toBe(true);
      expect(stateManager.getChangedProps().size).toBe(0);
    });

    it("should return independent copies on each clear", () => {
      stateManager.recordChange("name");
      const first = stateManager.clearChangedProps();

      stateManager.recordChange("count");
      const second = stateManager.clearChangedProps();

      expect(first.has("name")).toBe(true);
      expect(first.has("count")).toBe(false);
      expect(second.has("name")).toBe(false);
      expect(second.has("count")).toBe(true);
    });
  });
});
