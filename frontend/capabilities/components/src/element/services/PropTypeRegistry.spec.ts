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

import { describe, it, expect } from "vitest";
import {
  createPropTypeRegistry,
  propertyToAttributeName,
} from "./PropTypeRegistry";
import type { PropTypeDefinition } from "../types";

describe("PropTypeRegistry", () => {
  describe("propertyToAttributeName()", () => {
    it("should convert camelCase to kebab-case", () => {
      expect(propertyToAttributeName("myProperty")).toBe("my-property");
      expect(propertyToAttributeName("someLongName")).toBe("some-long-name");
      expect(propertyToAttributeName("HTTPHeader")).toBe("-h-t-t-p-header");
    });

    it("should return unchanged for lowercase names", () => {
      expect(propertyToAttributeName("simple")).toBe("simple");
      expect(propertyToAttributeName("name")).toBe("name");
    });
  });

  describe("createPropTypeRegistry()", () => {
    const samplePropTypes: Record<string, PropTypeDefinition> = {
      name: { type: "string", default: "default-name" },
      count: { type: "number", default: 0 },
      isActive: { type: "boolean", default: false },
      items: { type: "array", default: () => [], reflectToAttribute: false },
      config: { type: "object", reflectToAttribute: true },
    };

    describe("get()", () => {
      it("should return the prop definition for a known property", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.get("name")).toEqual({ type: "string", default: "default-name" });
        expect(registry.get("count")).toEqual({ type: "number", default: 0 });
      });

      it("should return undefined for unknown properties", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.get("unknown")).toBeUndefined();
      });

      it("should handle undefined propTypes", () => {
        const registry = createPropTypeRegistry({ propTypes: undefined });

        expect(registry.get("anything")).toBeUndefined();
      });
    });

    describe("getPropertyNames()", () => {
      it("should return all property names", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        const names = registry.getPropertyNames();
        expect(names).toContain("name");
        expect(names).toContain("count");
        expect(names).toContain("isActive");
        expect(names).toContain("items");
        expect(names).toContain("config");
        expect(names).toHaveLength(5);
      });

      it("should return empty array for undefined propTypes", () => {
        const registry = createPropTypeRegistry({ propTypes: undefined });

        expect(registry.getPropertyNames()).toEqual([]);
      });
    });

    describe("deriveObservedAttributes()", () => {
      it("should include primitive types by default", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        const observed = registry.deriveObservedAttributes();
        expect(observed).toContain("name");
        expect(observed).toContain("count");
        expect(observed).toContain("is-active");
      });

      it("should exclude arrays/objects by default", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        const observed = registry.deriveObservedAttributes();
        expect(observed).not.toContain("items");
      });

      it("should respect explicit reflectToAttribute: false", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        const observed = registry.deriveObservedAttributes();
        expect(observed).not.toContain("items");
      });

      it("should respect explicit reflectToAttribute: true for complex types", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        const observed = registry.deriveObservedAttributes();
        expect(observed).toContain("config");
      });
    });

    describe("getDefaultValue()", () => {
      it("should return static default values", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.getDefaultValue("name")).toBe("default-name");
        expect(registry.getDefaultValue("count")).toBe(0);
        expect(registry.getDefaultValue("isActive")).toBe(false);
      });

      it("should evaluate factory functions for defaults", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        const items1 = registry.getDefaultValue("items");
        const items2 = registry.getDefaultValue("items");

        expect(items1).toEqual([]);
        expect(items2).toEqual([]);
        expect(items1).not.toBe(items2);
      });

      it("should return undefined for unknown properties", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.getDefaultValue("unknown")).toBeUndefined();
      });

      it("should return undefined for properties without defaults", () => {
        const registry = createPropTypeRegistry({
          propTypes: { noDefault: { type: "string" } },
        });

        expect(registry.getDefaultValue("noDefault")).toBeUndefined();
      });
    });

    describe("shouldReflect()", () => {
      it("should return true for primitive types by default", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.shouldReflect("name")).toBe(true);
        expect(registry.shouldReflect("count")).toBe(true);
        expect(registry.shouldReflect("isActive")).toBe(true);
      });

      it("should return false for arrays by default", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.shouldReflect("items")).toBe(false);
      });

      it("should respect explicit reflectToAttribute setting", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.shouldReflect("config")).toBe(true);
      });

      it("should return false for unknown properties", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.shouldReflect("unknown")).toBe(false);
      });
    });

    describe("attributeToPropertyName()", () => {
      it("should find property name from propTypes mapping", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.attributeToPropertyName("is-active")).toBe("isActive");
        expect(registry.attributeToPropertyName("name")).toBe("name");
      });

      it("should fallback to standard conversion for unknown attributes", () => {
        const registry = createPropTypeRegistry({ propTypes: samplePropTypes });

        expect(registry.attributeToPropertyName("some-other-attr")).toBe("someOtherAttr");
      });
    });
  });
});
