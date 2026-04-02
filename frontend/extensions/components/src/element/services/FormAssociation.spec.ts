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

import { describe, it, expect, vi } from "vitest";
import { createFormAssociation } from "./FormAssociation";

describe("FormAssociation", () => {
  describe("when formAssociated is false", () => {
    it("should not attach internals", () => {
      const mockHost = {
        attachInternals: vi.fn(),
      } as unknown as HTMLElement;

      const formAssociation = createFormAssociation({
        host: mockHost,
        formAssociated: false,
      });

      expect(mockHost.attachInternals).not.toHaveBeenCalled();
      expect(formAssociation.getInternals()).toBeUndefined();
    });

    it("should report not form associated", () => {
      const mockHost = {
        attachInternals: vi.fn(),
      } as unknown as HTMLElement;

      const formAssociation = createFormAssociation({
        host: mockHost,
        formAssociated: false,
      });

      expect(formAssociation.isFormAssociated()).toBe(false);
    });
  });

  describe("when formAssociated is true", () => {
    it("should attach internals", () => {
      const mockInternals = {} as ElementInternals;
      const mockHost = {
        attachInternals: vi.fn(() => mockInternals),
      } as unknown as HTMLElement;

      const formAssociation = createFormAssociation({
        host: mockHost,
        formAssociated: true,
      });

      expect(mockHost.attachInternals).toHaveBeenCalledTimes(1);
      expect(formAssociation.getInternals()).toBe(mockInternals);
    });

    it("should report form associated", () => {
      const mockHost = {
        attachInternals: vi.fn(() => ({}) as ElementInternals),
      } as unknown as HTMLElement;

      const formAssociation = createFormAssociation({
        host: mockHost,
        formAssociated: true,
      });

      expect(formAssociation.isFormAssociated()).toBe(true);
    });
  });
});
