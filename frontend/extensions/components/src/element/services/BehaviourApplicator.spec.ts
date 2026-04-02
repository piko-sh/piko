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
import { createBehaviourApplicator } from "./BehaviourApplicator";
import type { PPBehaviour } from "./BehaviourApplicator";

describe("BehaviourApplicator", () => {
  let mockHost: HTMLElement;
  let behaviours: Record<string, PPBehaviour>;
  let getBehaviour: (name: string) => PPBehaviour | undefined;

  beforeEach(() => {
    mockHost = { tagName: "TEST-ELEMENT" } as HTMLElement;
    behaviours = {};
    getBehaviour = (name: string) => behaviours[name];
  });

  describe("applyBehaviours()", () => {
    it("should apply all enabled behaviours", () => {
      const behaviour1 = vi.fn();
      const behaviour2 = vi.fn();
      behaviours["clickable"] = behaviour1;
      behaviours["draggable"] = behaviour2;

      const applicator = createBehaviourApplicator({
        host: mockHost,
        enabledBehaviours: ["clickable", "draggable"],
        getBehaviour,
      });

      applicator.applyBehaviours();

      expect(behaviour1).toHaveBeenCalledWith(mockHost);
      expect(behaviour2).toHaveBeenCalledWith(mockHost);
    });

    it("should apply behaviours in order", () => {
      const callOrder: string[] = [];
      behaviours["first"] = () => callOrder.push("first");
      behaviours["second"] = () => callOrder.push("second");
      behaviours["third"] = () => callOrder.push("third");

      const applicator = createBehaviourApplicator({
        host: mockHost,
        enabledBehaviours: ["first", "second", "third"],
        getBehaviour,
      });

      applicator.applyBehaviours();

      expect(callOrder).toEqual(["first", "second", "third"]);
    });

    it("should warn for unknown behaviours", () => {
      const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

      const applicator = createBehaviourApplicator({
        host: mockHost,
        enabledBehaviours: ["unknown-behaviour"],
        getBehaviour,
      });

      applicator.applyBehaviours();

      expect(warnSpy).toHaveBeenCalledWith(
        expect.stringContaining('Unknown behaviour "unknown-behaviour"')
      );

      warnSpy.mockRestore();
    });

    it("should continue applying behaviours after unknown one", () => {
      const validBehaviour = vi.fn();
      behaviours["valid"] = validBehaviour;

      const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});

      const applicator = createBehaviourApplicator({
        host: mockHost,
        enabledBehaviours: ["unknown", "valid"],
        getBehaviour,
      });

      applicator.applyBehaviours();

      expect(validBehaviour).toHaveBeenCalled();

      warnSpy.mockRestore();
    });

    it("should do nothing when no behaviours are enabled", () => {
      const applicator = createBehaviourApplicator({
        host: mockHost,
        enabledBehaviours: [],
        getBehaviour,
      });

      expect(() => applicator.applyBehaviours()).not.toThrow();
    });
  });
});
