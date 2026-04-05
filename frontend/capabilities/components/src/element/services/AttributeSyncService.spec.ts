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
import { createAttributeSyncService } from "./AttributeSyncService";
import { createPropTypeRegistry } from "./PropTypeRegistry";
import type { AttributeSyncService } from "./AttributeSyncService";
import type { PropTypeRegistry } from "./PropTypeRegistry";
import type { PropTypeDefinition } from "../types";

describe("AttributeSyncService", () => {
  let container: HTMLElement;
  let host: HTMLElement;
  let propTypeRegistry: PropTypeRegistry;
  let state: Record<string, unknown>;
  let defaults: Record<string, unknown>;
  let attributeSyncService: AttributeSyncService;

  const samplePropTypes: Record<string, PropTypeDefinition> = {
    name: { type: "string", default: "default-name" },
    count: { type: "number", default: 0 },
    isActive: { type: "boolean", default: false },
    items: { type: "array", default: () => [], reflectToAttribute: false },
    config: { type: "object", reflectToAttribute: true },
    nullable: { type: "string", nullable: true },
  };

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);

    host = document.createElement("div");
    container.appendChild(host);

    propTypeRegistry = createPropTypeRegistry({ propTypes: samplePropTypes });
    state = { name: "test", count: 5, isActive: true };
    defaults = { name: "default-name", count: 0, isActive: false };

    attributeSyncService = createAttributeSyncService({
      host,
      propTypeRegistry,
      getState: () => state,
      getDefaults: () => defaults,
    });
  });

  afterEach(() => {
    document.body.removeChild(container);
  });

  describe("initial state", () => {
    it("should not be applying to state initially", () => {
      expect(attributeSyncService.isApplyingToState()).toBe(false);
    });

    it("should not be reflecting to attribute initially", () => {
      expect(attributeSyncService.isReflectingToAttribute()).toBe(false);
    });

    it("should be initialising initially", () => {
      expect(attributeSyncService.isInitialising()).toBe(true);
    });
  });

  describe("setInitialising()", () => {
    it("should update initialising state", () => {
      attributeSyncService.setInitialising(false);
      expect(attributeSyncService.isInitialising()).toBe(false);

      attributeSyncService.setInitialising(true);
      expect(attributeSyncService.isInitialising()).toBe(true);
    });
  });

  describe("translateAttributeValue()", () => {
    beforeEach(() => {
      attributeSyncService.setInitialising(false);
    });

    describe("boolean type", () => {
      it("should return false for null attribute", () => {
        expect(attributeSyncService.translateAttributeValue("boolean", null, "isActive")).toBe(
          false
        );
      });

      it('should return false for "false" string', () => {
        expect(
          attributeSyncService.translateAttributeValue("boolean", "false", "isActive")
        ).toBe(false);
      });

      it("should return true for any other value", () => {
        expect(attributeSyncService.translateAttributeValue("boolean", "", "isActive")).toBe(true);
        expect(
          attributeSyncService.translateAttributeValue("boolean", "true", "isActive")
        ).toBe(true);
        expect(
          attributeSyncService.translateAttributeValue("boolean", "yes", "isActive")
        ).toBe(true);
      });
    });

    describe("number type", () => {
      it("should parse valid numbers", () => {
        expect(attributeSyncService.translateAttributeValue("number", "42", "count")).toBe(42);
        expect(attributeSyncService.translateAttributeValue("number", "3.14", "count")).toBe(3.14);
        expect(attributeSyncService.translateAttributeValue("number", "-10", "count")).toBe(-10);
      });

      it("should return default for invalid numbers", () => {
        expect(attributeSyncService.translateAttributeValue("number", "not-a-number", "count")).toBe(
          0
        );
      });

      it("should return 0 when no default exists", () => {
        expect(
          attributeSyncService.translateAttributeValue("number", "invalid", "unknown")
        ).toBe(0);
      });
    });

    describe("string type", () => {
      it("should return the attribute value directly", () => {
        expect(attributeSyncService.translateAttributeValue("string", "hello", "name")).toBe(
          "hello"
        );
      });

      it("should return default for null value", () => {
        expect(attributeSyncService.translateAttributeValue("string", null, "name")).toBe(
          "default-name"
        );
      });
    });

    describe("json/array/object types", () => {
      it("should parse valid JSON for array", () => {
        const result = attributeSyncService.translateAttributeValue(
          "array",
          '[1, 2, 3]',
          "items"
        );
        expect(result).toEqual([1, 2, 3]);
      });

      it("should parse valid JSON for object", () => {
        const result = attributeSyncService.translateAttributeValue(
          "object",
          '{"key": "value"}',
          "config"
        );
        expect(result).toEqual({ key: "value" });
      });

      it("should return empty array for invalid JSON with array type", () => {
        const result = attributeSyncService.translateAttributeValue(
          "array",
          "invalid-json",
          "unknown"
        );
        expect(result).toEqual([]);
      });

      it("should return null for invalid JSON with object type", () => {
        const result = attributeSyncService.translateAttributeValue(
          "object",
          "invalid-json",
          "unknown"
        );
        expect(result).toBeNull();
      });
    });

    describe("nullable properties", () => {
      it("should return null for null attribute when nullable", () => {
        expect(
          attributeSyncService.translateAttributeValue("string", null, "nullable", true)
        ).toBeNull();
      });

      it("should return default for null attribute when not nullable", () => {
        expect(attributeSyncService.translateAttributeValue("string", null, "name", false)).toBe(
          "default-name"
        );
      });
    });
  });

  describe("applyAttributeToState()", () => {
    beforeEach(() => {
      attributeSyncService.setInitialising(false);
    });

    it("should update state with translated value", () => {
      attributeSyncService.applyAttributeToState("name", "new-value");

      expect(state.name).toBe("new-value");
    });

    it("should not update state if value is unchanged", () => {
      const originalState = { ...state };
      attributeSyncService.applyAttributeToState("name", "test");

      expect(state).toEqual(originalState);
    });

    it("should handle unknown properties gracefully", () => {
      expect(() => {
        attributeSyncService.applyAttributeToState("unknown", "value");
      }).not.toThrow();
    });
  });

  describe("reflectStateToAttribute()", () => {
    beforeEach(() => {
      attributeSyncService.setInitialising(false);
    });

    it("should set attribute for string values", () => {
      attributeSyncService.reflectStateToAttribute("name", "test-value");

      expect(host.getAttribute("name")).toBe("test-value");
    });

    it("should set attribute for number values", () => {
      attributeSyncService.reflectStateToAttribute("count", 42);

      expect(host.getAttribute("count")).toBe("42");
    });

    it("should add boolean attribute when true and remove when false", () => {
      attributeSyncService.reflectStateToAttribute("isActive", true);
      expect(host.hasAttribute("is-active")).toBe(true);

      attributeSyncService.reflectStateToAttribute("isActive", false);
      expect(host.hasAttribute("is-active")).toBe(false);
    });

    it("should remove attribute for null/undefined values", () => {
      host.setAttribute("name", "test");
      attributeSyncService.reflectStateToAttribute("name", null);

      expect(host.hasAttribute("name")).toBe(false);
    });

    it("should stringify JSON for object types", () => {
      attributeSyncService.reflectStateToAttribute("config", { key: "value" });

      expect(host.getAttribute("config")).toBe('{"key":"value"}');
    });

    it("should not reflect non-reflectable properties", () => {
      attributeSyncService.reflectStateToAttribute("items", [1, 2, 3]);

      expect(host.hasAttribute("items")).toBe(false);
    });

    it("should not update if attribute value is unchanged", () => {
      host.setAttribute("name", "same-value");
      const spy = vi.spyOn(host, "setAttribute");

      attributeSyncService.reflectStateToAttribute("name", "same-value");

      expect(spy).not.toHaveBeenCalled();
    });
  });

  describe("syncAllAttributesToState()", () => {
    beforeEach(() => {
      attributeSyncService.setInitialising(false);
    });

    it("should sync all known attributes to state", () => {
      host.setAttribute("name", "from-attr");
      host.setAttribute("count", "99");
      host.setAttribute("is-active", "true");

      attributeSyncService.syncAllAttributesToState();

      expect(state.name).toBe("from-attr");
      expect(state.count).toBe(99);
      expect(state.isActive).toBe(true);
    });

    it("should ignore unknown attributes", () => {
      host.setAttribute("unknown-attr", "value");

      expect(() => {
        attributeSyncService.syncAllAttributesToState();
      }).not.toThrow();
    });
  });

  describe("reflectAllStateToAttributes()", () => {
    beforeEach(() => {
      attributeSyncService.setInitialising(false);
    });

    it("should reflect all state properties to attributes", () => {
      attributeSyncService.reflectAllStateToAttributes();

      expect(host.getAttribute("name")).toBe("test");
      expect(host.getAttribute("count")).toBe("5");
      expect(host.hasAttribute("is-active")).toBe(true);
    });
  });

  describe("handleAttributeChanged()", () => {
    beforeEach(() => {
      attributeSyncService.setInitialising(false);
    });

    it("should apply attribute change to state", () => {
      attributeSyncService.handleAttributeChanged("name", "old", "new-value");

      expect(state.name).toBe("new-value");
    });

    it("should ignore changes during initialisation", () => {
      attributeSyncService.setInitialising(true);
      attributeSyncService.handleAttributeChanged("name", "old", "new-value");

      expect(state.name).toBe("test");
    });

    it("should ignore changes during reflection", () => {
      attributeSyncService.reflectStateToAttribute("name", "reflected");
      expect(state.name).toBe("test");
    });
  });
});
