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

import { describe, it, expect, vi, beforeEach } from 'vitest';
import type { PikoComponent, TimelineAction } from './timeline';

interface TestComponent extends PikoComponent {
    _afterRender: (() => void) | null;
    _disconnected: (() => void) | null;
    triggerAfterRender(): void;
    triggerDisconnect(): void;
}

function createTimelineComponent(timeline: unknown[]): TestComponent {
    const element = document.createElement('div');

    let afterRenderCallback: (() => void) | null = null;
    let disconnectedCallback: (() => void) | null = null;

    const refs: Record<string, HTMLElement> = {};

    Object.defineProperty(element, 'refs', { value: refs, configurable: true, writable: true });
    Object.defineProperty(element, 'onAfterRender', {
        value: (callback: () => void) => { afterRenderCallback = callback; },
        configurable: true,
    });
    Object.defineProperty(element, 'onDisconnected', {
        value: (callback: () => void) => { disconnectedCallback = callback; },
        configurable: true,
    });

    Object.defineProperty(element.constructor, '$$timeline', {
        value: timeline,
        configurable: true,
        writable: true,
    });

    const component = element as unknown as TestComponent;

    Object.defineProperty(component, '_afterRender', {
        get: () => afterRenderCallback,
        configurable: true,
    });
    Object.defineProperty(component, '_disconnected', {
        get: () => disconnectedCallback,
        configurable: true,
    });
    Object.defineProperty(component, 'triggerAfterRender', {
        value: () => afterRenderCallback?.(),
        configurable: true,
    });
    Object.defineProperty(component, 'triggerDisconnect', {
        value: () => disconnectedCallback?.(),
        configurable: true,
    });

    return component;
}

describe('setupAnimation', () => {
    beforeEach(() => {
        vi.resetModules();
    });

    it('should warn and return when no $$timeline exists', async () => {
        const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
        const { setupAnimation } = await import('./timeline');

        const element = document.createElement('div');
        setupAnimation(element);

        expect(warnSpy).toHaveBeenCalledOnce();
        expect(warnSpy.mock.calls[0][0]).toContain('no $$timeline data found');
        warnSpy.mockRestore();
    });

    it('should evaluate timeline after first render when time changes', async () => {
        const { setupAnimation } = await import('./timeline');

        const el = document.createElement('div');
        const timeline: TimelineAction[] = [
            { time: 0, action: 'hide', ref: 'box' },
            { time: 1, action: 'show', ref: 'box' },
        ];
        const component = createTimelineComponent(timeline);
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { box: el };

        setupAnimation(component);
        component.triggerAfterRender();

        component.setAttribute('time', '2');

        await vi.waitFor(() => {
            expect(el.hasAttribute('p-timeline-hidden')).toBe(false);
        });
    });

    it('should evaluate the timeline when the time attribute changes', async () => {
        const { setupAnimation } = await import('./timeline');

        const el = document.createElement('div');
        const timeline: TimelineAction[] = [
            { time: 0, action: 'hide', ref: 'box' },
            { time: 2, action: 'show', ref: 'box' },
        ];
        const component = createTimelineComponent(timeline);
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { box: el };

        setupAnimation(component);
        component.triggerAfterRender();

        component.setAttribute('time', '3');

        await vi.waitFor(() => {
            expect(el.hasAttribute('p-timeline-hidden')).toBe(false);
        });
    });

    it('should disconnect the observer on component disconnect', async () => {
        const { setupAnimation } = await import('./timeline');

        const timeline: TimelineAction[] = [
            { time: 1, action: 'show', ref: 'box' },
        ];
        const component = createTimelineComponent(timeline);

        setupAnimation(component);
        component.triggerAfterRender();

        expect(() => {
            component.triggerDisconnect();
        }).not.toThrow();
    });

    it('should capture texts on first mutation after render', async () => {
        const { setupAnimation } = await import('./timeline');

        const el = document.createElement('p');
        el.textContent = 'Typed text';
        const timeline: TimelineAction[] = [
            { time: 0, action: 'type', ref: 'text' },
        ];
        const component = createTimelineComponent(timeline);
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { text: el };

        setupAnimation(component);
        component.triggerAfterRender();

        component.setAttribute('time', '0.15');

        await vi.waitFor(() => {
            expect(el.textContent).not.toBe('Typed text');
        });
    });

    it('should handle responsive timelines with a matching media query', async () => {
        const { setupAnimation } = await import('./timeline');

        const el = document.createElement('div');
        const matchMediaMock = vi.fn().mockReturnValue({
            matches: true,
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
        });
        vi.stubGlobal('matchMedia', matchMediaMock);

        const timeline = [
            {
                media: '(max-width: 768px)',
                actions: [{ time: 0, action: 'show', ref: 'box' }],
            },
            {
                media: null,
                actions: [{ time: 0, action: 'hide', ref: 'box' }],
            },
        ];
        const component = createTimelineComponent(timeline);
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { box: el };

        setupAnimation(component);
        component.triggerAfterRender();

        component.setAttribute('time', '1');

        await vi.waitFor(() => {
            expect(el.hasAttribute('p-timeline-hidden')).toBe(false);
        });

        vi.unstubAllGlobals();
    });

    it('should handle responsive timelines with media queries', async () => {
        const { setupAnimation } = await import('./timeline');

        const el = document.createElement('div');
        const matchMediaMock = vi.fn().mockReturnValue({
            matches: false,
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
        });
        vi.stubGlobal('matchMedia', matchMediaMock);

        const timeline = [
            {
                media: '(max-width: 768px)',
                actions: [{ time: 0, action: 'show', ref: 'box' }],
            },
            {
                media: null,
                actions: [{ time: 0, action: 'hide', ref: 'box' }],
            },
        ];
        const component = createTimelineComponent(timeline);
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { box: el };

        setupAnimation(component);
        component.triggerAfterRender();

        component.setAttribute('time', '1');

        await vi.waitFor(() => {
            expect(el.hasAttribute('p-timeline-hidden')).toBe(true);
        });

        vi.unstubAllGlobals();
    });

    it('should re-evaluate when a media query changes after init', async () => {
        const { setupAnimation } = await import('./timeline');

        const el = document.createElement('div');
        let changeHandler: (() => void) | undefined;

        const matchMediaMock = vi.fn().mockReturnValue({
            matches: true,
            addEventListener: (_event: string, handler: () => void) => { changeHandler = handler; },
            removeEventListener: vi.fn(),
        });
        vi.stubGlobal('matchMedia', matchMediaMock);

        const timeline = [
            {
                media: '(max-width: 768px)',
                actions: [
                    { time: 0, action: 'hide', ref: 'box' },
                    { time: 1, action: 'show', ref: 'box' },
                ],
            },
        ];
        const component = createTimelineComponent(timeline);
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { box: el };

        setupAnimation(component);
        component.triggerAfterRender();
        component.setAttribute('time', '2');

        if (changeHandler) {
            changeHandler();
        }

        expect(el.hasAttribute('p-timeline-hidden')).toBe(false);

        vi.unstubAllGlobals();
    });

    it('should sort actions by time', async () => {
        const { setupAnimation } = await import('./timeline');

        const el = document.createElement('div');
        const timeline: TimelineAction[] = [
            { time: 5, action: 'show', ref: 'box' },
            { time: 0, action: 'hide', ref: 'box' },
        ];
        const component = createTimelineComponent(timeline);
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { box: el };

        setupAnimation(component);
        component.triggerAfterRender();
        component.setAttribute('time', '3');

        await vi.waitFor(() => {
            expect(el.hasAttribute('p-timeline-hidden')).toBe(true);
        });
    });
});
