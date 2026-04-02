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
import { createLifecycleManager } from "./LifecycleManager";
import type { LifecycleManager } from "./LifecycleManager";

describe("LifecycleManager", () => {
  let lifecycleManager: LifecycleManager;

  beforeEach(() => {
    lifecycleManager = createLifecycleManager();
  });

  describe("initial state", () => {
    it("should not have connected initially", () => {
      expect(lifecycleManager.hasConnectedOnce()).toBe(false);
    });
  });

  describe("onConnected() registration", () => {
    it("should register and execute a single callback", () => {
      const callback = vi.fn();

      lifecycleManager.onConnected(callback);
      lifecycleManager.executeConnected();

      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should register and execute multiple callbacks in order", () => {
      const callOrder: string[] = [];

      lifecycleManager.onConnected(() => callOrder.push("first"));
      lifecycleManager.onConnected(() => callOrder.push("second"));
      lifecycleManager.onConnected(() => callOrder.push("third"));
      lifecycleManager.executeConnected();

      expect(callOrder).toEqual(["first", "second", "third"]);
    });
  });

  describe("executeConnected()", () => {
    it("should execute registered callbacks", () => {
      const callback1 = vi.fn();
      const callback2 = vi.fn();

      lifecycleManager.onConnected(callback1);
      lifecycleManager.onConnected(callback2);
      lifecycleManager.executeConnected();

      expect(callback1).toHaveBeenCalledTimes(1);
      expect(callback2).toHaveBeenCalledTimes(1);
    });

    it("should only execute once (first connection)", () => {
      const callback = vi.fn();

      lifecycleManager.onConnected(callback);
      lifecycleManager.executeConnected();
      lifecycleManager.executeConnected();
      lifecycleManager.executeConnected();

      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should set hasConnectedOnce to true", () => {
      lifecycleManager.executeConnected();

      expect(lifecycleManager.hasConnectedOnce()).toBe(true);
    });

    it("should not throw when no callbacks are registered", () => {
      expect(() => lifecycleManager.executeConnected()).not.toThrow();
    });
  });

  describe("onDisconnected() registration", () => {
    it("should register and execute a single callback", () => {
      const callback = vi.fn();

      lifecycleManager.onDisconnected(callback);
      lifecycleManager.executeDisconnected();

      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should execute multiple callbacks in registration order", () => {
      const callOrder: string[] = [];

      lifecycleManager.onDisconnected(() => callOrder.push("first"));
      lifecycleManager.onDisconnected(() => callOrder.push("second"));
      lifecycleManager.executeDisconnected();

      expect(callOrder).toEqual(["first", "second"]);
    });

    it("should execute every time (not once-only like connected)", () => {
      const callback = vi.fn();

      lifecycleManager.onDisconnected(callback);
      lifecycleManager.executeDisconnected();
      lifecycleManager.executeDisconnected();

      expect(callback).toHaveBeenCalledTimes(2);
    });
  });

  describe("onBeforeRender() registration", () => {
    it("should register and execute callbacks", () => {
      const callback = vi.fn();

      lifecycleManager.onBeforeRender(callback);
      lifecycleManager.executeBeforeRender();

      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should execute multiple callbacks in order", () => {
      const callOrder: string[] = [];

      lifecycleManager.onBeforeRender(() => callOrder.push("first"));
      lifecycleManager.onBeforeRender(() => callOrder.push("second"));
      lifecycleManager.executeBeforeRender();

      expect(callOrder).toEqual(["first", "second"]);
    });
  });

  describe("onAfterRender() registration", () => {
    it("should register and execute callbacks", () => {
      const callback = vi.fn();

      lifecycleManager.onAfterRender(callback);
      lifecycleManager.executeAfterRender();

      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should execute multiple callbacks in order", () => {
      const callOrder: string[] = [];

      lifecycleManager.onAfterRender(() => callOrder.push("first"));
      lifecycleManager.onAfterRender(() => callOrder.push("second"));
      lifecycleManager.executeAfterRender();

      expect(callOrder).toEqual(["first", "second"]);
    });
  });

  describe("onUpdated() registration", () => {
    it("should pass changed properties to callbacks", () => {
      const callback = vi.fn();

      lifecycleManager.onUpdated(callback);

      const changedProps = new Set(["name", "count"]);
      lifecycleManager.executeUpdated(changedProps);

      expect(callback).toHaveBeenCalledWith(changedProps);
    });

    it("should execute multiple callbacks with changed props", () => {
      const callback1 = vi.fn();
      const callback2 = vi.fn();

      lifecycleManager.onUpdated(callback1);
      lifecycleManager.onUpdated(callback2);

      const changedProps = new Set(["value"]);
      lifecycleManager.executeUpdated(changedProps);

      expect(callback1).toHaveBeenCalledWith(changedProps);
      expect(callback2).toHaveBeenCalledWith(changedProps);
    });
  });

  describe("resetConnectedState()", () => {
    it("should reset hasConnectedOnce to false", () => {
      lifecycleManager.executeConnected();
      expect(lifecycleManager.hasConnectedOnce()).toBe(true);

      lifecycleManager.resetConnectedState();
      expect(lifecycleManager.hasConnectedOnce()).toBe(false);
    });

    it("should allow executeConnected to run again", () => {
      const callback = vi.fn();

      lifecycleManager.onConnected(callback);
      lifecycleManager.executeConnected();
      lifecycleManager.resetConnectedState();
      lifecycleManager.executeConnected();

      expect(callback).toHaveBeenCalledTimes(2);
    });
  });

  describe("onCleanup() registration", () => {
    it("should register and execute a single cleanup callback", () => {
      const callback = vi.fn();

      lifecycleManager.onCleanup(callback);
      lifecycleManager.executeCleanups();

      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should execute multiple cleanups in registration order", () => {
      const callOrder: string[] = [];

      lifecycleManager.onCleanup(() => callOrder.push("first"));
      lifecycleManager.onCleanup(() => callOrder.push("second"));
      lifecycleManager.onCleanup(() => callOrder.push("third"));
      lifecycleManager.executeCleanups();

      expect(callOrder).toEqual(["first", "second", "third"]);
    });

    it("should clear the cleanup array after execution", () => {
      const callback = vi.fn();

      lifecycleManager.onCleanup(callback);
      lifecycleManager.executeCleanups();
      lifecycleManager.executeCleanups();

      expect(callback).toHaveBeenCalledTimes(1);
    });

    it("should allow cleanups registered inside onConnected to work", () => {
      const cleanupSpy = vi.fn();

      lifecycleManager.onConnected(() => {
        lifecycleManager.onCleanup(cleanupSpy);
      });

      lifecycleManager.executeConnected();
      lifecycleManager.executeCleanups();

      expect(cleanupSpy).toHaveBeenCalledTimes(1);
    });
  });

  describe("with no callbacks registered", () => {
    it("should handle all execute methods without throwing", () => {
      expect(() => {
        lifecycleManager.executeConnected();
        lifecycleManager.executeDisconnected();
        lifecycleManager.executeBeforeRender();
        lifecycleManager.executeAfterRender();
        lifecycleManager.executeUpdated(new Set());
        lifecycleManager.executeCleanups();
      }).not.toThrow();
    });
  });
});
