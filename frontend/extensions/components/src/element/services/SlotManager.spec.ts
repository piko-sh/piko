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

import {describe, it, expect, vi, beforeEach, afterEach} from "vitest";
import {createSlotManager} from "./SlotManager";
import type {SlotManager} from "./SlotManager";

describe("SlotManager", () => {
    let container: HTMLElement;

    beforeEach(() => {
        container = document.createElement("div");
        document.body.appendChild(container);
    });

    afterEach(() => {
        document.body.removeChild(container);
    });

    describe("when shadow root is not available", () => {
        let slotManager: SlotManager;

        beforeEach(() => {
            slotManager = createSlotManager({
                getShadowRoot: () => null,
                tagName: "TEST-ELEMENT",
            });
        });

        describe("getSlottedElements()", () => {
            it("should return empty array for default slot", () => {
                expect(slotManager.getSlottedElements()).toEqual([]);
            });

            it("should return empty array for named slot", () => {
                expect(slotManager.getSlottedElements("header")).toEqual([]);
            });
        });

        describe("hasSlotContent()", () => {
            it("should return false for default slot", () => {
                expect(slotManager.hasSlotContent()).toBe(false);
            });

            it("should return false for named slot", () => {
                expect(slotManager.hasSlotContent("footer")).toBe(false);
            });
        });

        describe("attachSlotListener()", () => {
            it("should not call callback immediately", () => {
                const callback = vi.fn();
                slotManager.attachSlotListener("", callback);
                expect(callback).not.toHaveBeenCalled();
            });

            it("should not warn when shadow root is unavailable", () => {
                const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
                slotManager.attachSlotListener("", vi.fn());

                expect(warnSpy).not.toHaveBeenCalled();
                warnSpy.mockRestore();
            });
        });

        describe("flushPendingListeners()", () => {
            it("should be a no-op when queue is empty", () => {
                expect(() => slotManager.flushPendingListeners()).not.toThrow();
            });

            it("should be a no-op when shadow root is still null", () => {
                const callback = vi.fn();
                slotManager.attachSlotListener("", callback);

                slotManager.flushPendingListeners();
                expect(callback).not.toHaveBeenCalled();
            });
        });
    });

    describe("when shadow root is available", () => {
        let host: HTMLElement;
        let shadowRoot: ShadowRoot;
        let slotManager: SlotManager;

        beforeEach(() => {
            host = document.createElement("div");
            container.appendChild(host);
            shadowRoot = host.attachShadow({mode: "open"});

            slotManager = createSlotManager({
                getShadowRoot: () => shadowRoot,
                tagName: "TEST-ELEMENT",
            });
        });

        describe("getSlottedElements()", () => {
            it("should return empty array when no slot element exists", () => {
                expect(slotManager.getSlottedElements()).toEqual([]);
            });

            it("should return empty array when slot has no content", () => {
                shadowRoot.appendChild(document.createElement("slot"));
                expect(slotManager.getSlottedElements()).toEqual([]);
            });

            it("should return slotted elements for default slot", () => {
                shadowRoot.appendChild(document.createElement("slot"));
                const child = document.createElement("span");
                host.appendChild(child);

                expect(slotManager.getSlottedElements()).toContain(child);
            });

            it("should return slotted elements for named slot", () => {
                const slot = document.createElement("slot");
                slot.name = "header";
                shadowRoot.appendChild(slot);

                const h1 = document.createElement("h1");
                h1.slot = "header";
                host.appendChild(h1);

                expect(slotManager.getSlottedElements("header")).toContain(h1);
            });

            it("should not return elements from a different slot", () => {
                const defaultSlot = document.createElement("slot");
                const namedSlot = document.createElement("slot");
                namedSlot.name = "footer";
                shadowRoot.appendChild(defaultSlot);
                shadowRoot.appendChild(namedSlot);

                const footerEl = document.createElement("footer");
                footerEl.slot = "footer";
                host.appendChild(footerEl);

                expect(slotManager.getSlottedElements()).not.toContain(footerEl);
                expect(slotManager.getSlottedElements("footer")).toContain(footerEl);
            });

            it("should return multiple slotted elements", () => {
                shadowRoot.appendChild(document.createElement("slot"));
                const a = document.createElement("span");
                const b = document.createElement("div");
                host.appendChild(a);
                host.appendChild(b);

                const result = slotManager.getSlottedElements();
                expect(result).toContain(a);
                expect(result).toContain(b);
                expect(result).toHaveLength(2);
            });
        });

        describe("hasSlotContent()", () => {
            it("should return false when slot has no content", () => {
                shadowRoot.appendChild(document.createElement("slot"));
                expect(slotManager.hasSlotContent()).toBe(false);
            });

            it("should return true when default slot has content", () => {
                shadowRoot.appendChild(document.createElement("slot"));
                host.appendChild(document.createElement("span"));

                expect(slotManager.hasSlotContent()).toBe(true);
            });

            it("should return false for named slot without content", () => {
                const slot = document.createElement("slot");
                slot.name = "actions";
                shadowRoot.appendChild(slot);

                expect(slotManager.hasSlotContent("actions")).toBe(false);
            });

            it("should return true for named slot with content", () => {
                const slot = document.createElement("slot");
                slot.name = "actions";
                shadowRoot.appendChild(slot);

                const btn = document.createElement("button");
                btn.slot = "actions";
                host.appendChild(btn);

                expect(slotManager.hasSlotContent("actions")).toBe(true);
            });

            it("should distinguish between different named slots", () => {
                const headerSlot = document.createElement("slot");
                headerSlot.name = "header";
                const footerSlot = document.createElement("slot");
                footerSlot.name = "footer";
                shadowRoot.appendChild(headerSlot);
                shadowRoot.appendChild(footerSlot);

                const h1 = document.createElement("h1");
                h1.slot = "header";
                host.appendChild(h1);

                expect(slotManager.hasSlotContent("header")).toBe(true);
                expect(slotManager.hasSlotContent("footer")).toBe(false);
            });
        });

        describe("attachSlotListener()", () => {
            it("should warn when slot element is not found", () => {
                const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
                slotManager.attachSlotListener("nonexistent", vi.fn());

                expect(warnSpy).toHaveBeenCalledWith(
                    expect.stringContaining("not found")
                );
                warnSpy.mockRestore();
            });

            it("should include the tag name in the not-found warning", () => {
                const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
                slotManager.attachSlotListener("missing", vi.fn());

                expect(warnSpy).toHaveBeenCalledWith(
                    expect.stringContaining("TEST-ELEMENT")
                );
                warnSpy.mockRestore();
            });

            it("should call callback immediately with current content", () => {
                shadowRoot.appendChild(document.createElement("slot"));
                const child = document.createElement("span");
                host.appendChild(child);

                const callback = vi.fn();
                slotManager.attachSlotListener("", callback);

                expect(callback).toHaveBeenCalledTimes(1);
                expect(callback).toHaveBeenCalledWith(expect.arrayContaining([child]));
            });

            it("should call callback with empty array when slot has no content", () => {
                shadowRoot.appendChild(document.createElement("slot"));

                const callback = vi.fn();
                slotManager.attachSlotListener("", callback);

                expect(callback).toHaveBeenCalledTimes(1);
                expect(callback).toHaveBeenCalledWith([]);
            });

            it("should work with named slots", () => {
                const slot = document.createElement("slot");
                slot.name = "icon";
                shadowRoot.appendChild(slot);

                const svg = document.createElement("svg");
                svg.slot = "icon";
                host.appendChild(svg);

                const callback = vi.fn();
                slotManager.attachSlotListener("icon", callback);

                expect(callback).toHaveBeenCalledTimes(1);
                expect(callback).toHaveBeenCalledWith(expect.arrayContaining([svg]));
            });

            it("should support multiple listeners on different slots", () => {
                const defaultSlot = document.createElement("slot");
                const iconSlot = document.createElement("slot");
                iconSlot.name = "icon";
                shadowRoot.appendChild(defaultSlot);
                shadowRoot.appendChild(iconSlot);

                const defaultCb = vi.fn();
                const iconCb = vi.fn();
                slotManager.attachSlotListener("", defaultCb);
                slotManager.attachSlotListener("icon", iconCb);

                expect(defaultCb).toHaveBeenCalledTimes(1);
                expect(iconCb).toHaveBeenCalledTimes(1);
            });

            it("should support multiple listeners on the same slot", () => {
                shadowRoot.appendChild(document.createElement("slot"));

                const cb1 = vi.fn();
                const cb2 = vi.fn();
                slotManager.attachSlotListener("", cb1);
                slotManager.attachSlotListener("", cb2);

                expect(cb1).toHaveBeenCalledTimes(1);
                expect(cb2).toHaveBeenCalledTimes(1);
            });
        });

        describe("flushPendingListeners()", () => {
            it("should be a no-op when nothing was queued", () => {
                expect(() => slotManager.flushPendingListeners()).not.toThrow();
            });
        });
    });

    describe("deferred attach lifecycle", () => {
        let host: HTMLElement;
        let shadowRoot: ShadowRoot | null;
        let slotManager: SlotManager;

        beforeEach(() => {
            host = document.createElement("div");
            container.appendChild(host);
            shadowRoot = null;

            slotManager = createSlotManager({
                getShadowRoot: () => shadowRoot,
                tagName: "DEFERRED-ELEMENT",
            });
        });

        it("should queue a single listener and replay on flush", () => {
            const callback = vi.fn();
            slotManager.attachSlotListener("", callback);
            expect(callback).not.toHaveBeenCalled();

            shadowRoot = host.attachShadow({mode: "open"});
            shadowRoot.appendChild(document.createElement("slot"));

            slotManager.flushPendingListeners();
            expect(callback).toHaveBeenCalledTimes(1);
        });

        it("should queue multiple listeners for different slots", () => {
            const defaultCb = vi.fn();
            const iconCb = vi.fn();
            slotManager.attachSlotListener("", defaultCb);
            slotManager.attachSlotListener("icon", iconCb);

            shadowRoot = host.attachShadow({mode: "open"});
            shadowRoot.appendChild(document.createElement("slot"));
            const iconSlot = document.createElement("slot");
            iconSlot.name = "icon";
            shadowRoot.appendChild(iconSlot);

            slotManager.flushPendingListeners();

            expect(defaultCb).toHaveBeenCalledTimes(1);
            expect(iconCb).toHaveBeenCalledTimes(1);
        });

        it("should queue multiple listeners for the same slot", () => {
            const cb1 = vi.fn();
            const cb2 = vi.fn();
            slotManager.attachSlotListener("icon", cb1);
            slotManager.attachSlotListener("icon", cb2);

            shadowRoot = host.attachShadow({mode: "open"});
            const iconSlot = document.createElement("slot");
            iconSlot.name = "icon";
            shadowRoot.appendChild(iconSlot);

            slotManager.flushPendingListeners();

            expect(cb1).toHaveBeenCalledTimes(1);
            expect(cb2).toHaveBeenCalledTimes(1);
        });

        it("should drain the queue so a second flush is a no-op", () => {
            const callback = vi.fn();
            slotManager.attachSlotListener("", callback);

            shadowRoot = host.attachShadow({mode: "open"});
            shadowRoot.appendChild(document.createElement("slot"));

            slotManager.flushPendingListeners();
            expect(callback).toHaveBeenCalledTimes(1);

            slotManager.flushPendingListeners();
            expect(callback).toHaveBeenCalledTimes(1);
        });

        it("should receive initial slot content on flush", () => {
            const callback = vi.fn();
            slotManager.attachSlotListener("", callback);

            shadowRoot = host.attachShadow({mode: "open"});
            shadowRoot.appendChild(document.createElement("slot"));
            const child = document.createElement("span");
            host.appendChild(child);

            slotManager.flushPendingListeners();

            expect(callback).toHaveBeenCalledWith(expect.arrayContaining([child]));
        });

        it("should receive empty array when slot has no content on flush", () => {
            const callback = vi.fn();
            slotManager.attachSlotListener("", callback);

            shadowRoot = host.attachShadow({mode: "open"});
            shadowRoot.appendChild(document.createElement("slot"));

            slotManager.flushPendingListeners();

            expect(callback).toHaveBeenCalledWith([]);
        });

        it("should warn for queued listener when slot element is not found on flush", () => {
            const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
            const callback = vi.fn();
            slotManager.attachSlotListener("missing", callback);

            shadowRoot = host.attachShadow({mode: "open"});

            slotManager.flushPendingListeners();

            expect(callback).not.toHaveBeenCalled();
            expect(warnSpy).toHaveBeenCalledWith(
                expect.stringContaining("not found")
            );
            warnSpy.mockRestore();
        });

        it("should allow direct attach after shadow root becomes available", () => {
            shadowRoot = host.attachShadow({mode: "open"});
            shadowRoot.appendChild(document.createElement("slot"));

            const callback = vi.fn();
            slotManager.attachSlotListener("", callback);

            expect(callback).toHaveBeenCalledTimes(1);
        });

        it("should handle a mix of queued and direct attach", () => {
            const earlyCallback = vi.fn();
            slotManager.attachSlotListener("", earlyCallback);
            expect(earlyCallback).not.toHaveBeenCalled();

            shadowRoot = host.attachShadow({mode: "open"});
            shadowRoot.appendChild(document.createElement("slot"));

            const lateCallback = vi.fn();
            slotManager.attachSlotListener("", lateCallback);
            expect(lateCallback).toHaveBeenCalledTimes(1);

            slotManager.flushPendingListeners();
            expect(earlyCallback).toHaveBeenCalledTimes(1);
        });
    });
});
