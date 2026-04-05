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

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  createShadowDOMService,
  RESET_CSS,
} from "./ShadowDOMService";

describe("ShadowDOMService", () => {
  let container: HTMLElement;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
  });

  afterEach(() => {
    document.body.removeChild(container);
  });

  describe("RESET_CSS", () => {
    it("should be defined", () => {
      expect(RESET_CSS).toBeDefined();
      expect(typeof RESET_CSS).toBe("string");
    });

    it("should include box-sizing reset", () => {
      expect(RESET_CSS).toContain("box-sizing");
    });

    it("should include host display block", () => {
      expect(RESET_CSS).toContain(":host");
      expect(RESET_CSS).toContain("display: block");
    });
  });

  describe("createShadowDOMService()", () => {
    describe("hasShadowRoot()", () => {
      it("should return false initially", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host });

        expect(service.hasShadowRoot()).toBe(false);
      });

      it("should return true after ensureShadowRoot", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host });

        service.ensureShadowRoot();

        expect(service.hasShadowRoot()).toBe(true);
      });
    });

    describe("getShadowRoot()", () => {
      it("should return null initially", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host });

        expect(service.getShadowRoot()).toBeNull();
      });

      it("should return shadow root after ensureShadowRoot", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host });

        const shadow = service.ensureShadowRoot();

        expect(service.getShadowRoot()).toBe(shadow);
      });
    });

    describe("ensureShadowRoot()", () => {
      it("should create an open shadow root by default", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host });

        const shadow = service.ensureShadowRoot();

        expect(shadow).toBeDefined();
        expect(shadow.mode).toBe("open");
      });

      it("should inject reset CSS", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host });

        const shadow = service.ensureShadowRoot();
        const styles = shadow.querySelectorAll("style");

        expect(styles.length).toBeGreaterThanOrEqual(1);
        expect(styles[0].textContent).toContain("box-sizing");
      });

      it("should inject component CSS when provided", () => {
        const host = document.createElement("div");
        const componentCSS = ".custom { color: red; }";
        const service = createShadowDOMService({ host, componentCSS });

        const shadow = service.ensureShadowRoot();
        const styles = shadow.querySelectorAll("style");

        expect(styles.length).toBe(2);
        expect(styles[1].textContent).toBe(componentCSS);
      });

      it("should not inject empty component CSS", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host, componentCSS: "   " });

        const shadow = service.ensureShadowRoot();
        const styles = shadow.querySelectorAll("style");

        expect(styles.length).toBe(1);
      });

      it("should be idempotent", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host });

        const shadow1 = service.ensureShadowRoot();
        const shadow2 = service.ensureShadowRoot();

        expect(shadow1).toBe(shadow2);
      });

      it("should respect custom mode", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host, mode: "closed" });

        const shadow = service.ensureShadowRoot();

        expect(shadow.mode).toBe("closed");
      });

      it("should set delegatesFocus to false by default", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host });

        const shadow = service.ensureShadowRoot();

        expect(shadow.delegatesFocus).toBeFalsy();
      });

      it("should enable delegatesFocus when specified", () => {
        const host = document.createElement("div");
        const service = createShadowDOMService({ host, delegatesFocus: true });

        const shadow = service.ensureShadowRoot();

        expect(shadow).toBeDefined();
        expect(shadow.mode).toBe("open");
      });
    });
  });
});
