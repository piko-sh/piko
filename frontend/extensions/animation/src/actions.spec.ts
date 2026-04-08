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

import { describe, it, expect, vi } from 'vitest';
import type { PikoComponent, TimelineAction } from './timeline';
import {
    registerTimelineAction,
    captureTypewriterTexts,
    captureTypehtmlContents,
    evaluateTimeline,
    evaluateAnchors,
} from './actions';

function createComponent(refMap: Record<string, HTMLElement> = {}): PikoComponent {
    const element = document.createElement('div');
    const component = element as unknown as PikoComponent;
    Object.defineProperty(component, 'refs', { value: refMap, configurable: true, writable: true });
    Object.defineProperty(component, 'onAfterRender', { value: vi.fn(), configurable: true });
    Object.defineProperty(component, 'onDisconnected', { value: vi.fn(), configurable: true });
    return component;
}

function action(overrides: Partial<TimelineAction> & { action: string; ref: string }): TimelineAction {
    return { time: 0, ...overrides };
}

describe('registerTimelineAction', () => {
    it('should dispatch to a registered custom handler', () => {
        const handler = vi.fn();
        registerTimelineAction('highlight', handler);

        const el = document.createElement('span');
        const component = createComponent({ target: el });
        const actions: TimelineAction[] = [
            action({ action: 'highlight', ref: 'target', time: 1, params: { colour: 'red' } }),
        ];

        evaluateTimeline(component, actions, 1.5, new Map(), new Map());

        expect(handler).toHaveBeenCalledOnce();
        expect(handler).toHaveBeenCalledWith(el, actions[0], 1.5, component);
    });

    it('should pass null when the ref is missing', () => {
        const handler = vi.fn();
        registerTimelineAction('sound', handler);

        const component = createComponent({});
        const actions: TimelineAction[] = [
            action({ action: 'sound', ref: '', time: 0, params: { src: 'click.wav' } }),
        ];

        evaluateTimeline(component, actions, 1, new Map(), new Map());

        expect(handler).toHaveBeenCalledWith(null, actions[0], 1, component);
    });

    it('should not call handler for unregistered action types', () => {
        const component = createComponent({});
        const actions: TimelineAction[] = [
            action({ action: 'nonexistent', ref: '', time: 0 }),
        ];

        expect(() => {
            evaluateTimeline(component, actions, 1, new Map(), new Map());
        }).not.toThrow();
    });
});

describe('captureTypewriterTexts', () => {
    it('should capture text content and clear the element', () => {
        const el = document.createElement('p');
        el.textContent = 'Hello world';

        const component = createComponent({ greeting: el });
        const actions: TimelineAction[] = [action({ action: 'type', ref: 'greeting', time: 1 })];
        const textMap = new Map<string, string>();

        captureTypewriterTexts(component, actions, textMap);

        expect(textMap.get('greeting')).toBe('Hello world');
        expect(el.textContent).toBe('');
    });

    it('should skip non-type actions', () => {
        const el = document.createElement('p');
        el.textContent = 'Unchanged';

        const component = createComponent({ box: el });
        const actions: TimelineAction[] = [action({ action: 'show', ref: 'box', time: 0 })];
        const textMap = new Map<string, string>();

        captureTypewriterTexts(component, actions, textMap);

        expect(textMap.size).toBe(0);
        expect(el.textContent).toBe('Unchanged');
    });

    it('should skip missing refs', () => {
        const component = createComponent({});
        const actions: TimelineAction[] = [action({ action: 'type', ref: 'missing', time: 0 })];
        const textMap = new Map<string, string>();

        captureTypewriterTexts(component, actions, textMap);

        expect(textMap.size).toBe(0);
    });
});

describe('captureTypehtmlContents', () => {
    it('should capture innerHTML and clear the element', () => {
        const el = document.createElement('pre');
        el.innerHTML = '<span class="kw">const</span> x = 1;';

        const component = createComponent({ code: el });
        const actions: TimelineAction[] = [action({ action: 'typehtml', ref: 'code', time: 0 })];
        const htmlMap = new Map<string, string>();

        captureTypehtmlContents(component, actions, htmlMap);

        expect(htmlMap.get('code')).toBe('<span class="kw">const</span> x = 1;');
        expect(el.innerHTML).toBe('');
    });

    it('should skip non-typehtml actions', () => {
        const el = document.createElement('pre');
        el.innerHTML = '<b>bold</b>';

        const component = createComponent({ text: el });
        const actions: TimelineAction[] = [action({ action: 'type', ref: 'text', time: 0 })];
        const htmlMap = new Map<string, string>();

        captureTypehtmlContents(component, actions, htmlMap);

        expect(htmlMap.size).toBe(0);
    });

    it('should skip missing refs', () => {
        const component = createComponent({});
        const actions: TimelineAction[] = [action({ action: 'typehtml', ref: 'gone', time: 0 })];
        const htmlMap = new Map<string, string>();

        captureTypehtmlContents(component, actions, htmlMap);

        expect(htmlMap.size).toBe(0);
    });
});

describe('evaluateTimeline', () => {
    describe('visibility actions', () => {
        it('should show an element when currentTime reaches the show action', () => {
            const el = document.createElement('div');
            el.setAttribute('p-timeline-hidden', '');

            const component = createComponent({ box: el });
            const actions: TimelineAction[] = [
                action({ action: 'hide', ref: 'box', time: 0 }),
                action({ action: 'show', ref: 'box', time: 2 }),
            ];

            evaluateTimeline(component, actions, 2, new Map(), new Map());

            expect(el.hasAttribute('p-timeline-hidden')).toBe(false);
        });

        it('should hide an element before the show time', () => {
            const el = document.createElement('div');

            const component = createComponent({ box: el });
            const actions: TimelineAction[] = [
                action({ action: 'hide', ref: 'box', time: 0 }),
                action({ action: 'show', ref: 'box', time: 5 }),
            ];

            evaluateTimeline(component, actions, 1, new Map(), new Map());

            expect(el.hasAttribute('p-timeline-hidden')).toBe(true);
        });

        it('should set initial hidden state when first action is show', () => {
            const el = document.createElement('div');

            const component = createComponent({ box: el });
            const actions: TimelineAction[] = [
                action({ action: 'show', ref: 'box', time: 3 }),
            ];

            evaluateTimeline(component, actions, 0, new Map(), new Map());

            expect(el.hasAttribute('p-timeline-hidden')).toBe(true);
        });

        it('should set initial visible state when first action is hide', () => {
            const el = document.createElement('div');

            const component = createComponent({ box: el });
            const actions: TimelineAction[] = [
                action({ action: 'hide', ref: 'box', time: 3 }),
            ];

            evaluateTimeline(component, actions, 0, new Map(), new Map());

            expect(el.hasAttribute('p-timeline-hidden')).toBe(false);
        });

        it('should skip refs that do not exist on the component', () => {
            const component = createComponent({});
            const actions: TimelineAction[] = [
                action({ action: 'show', ref: 'missing', time: 0 }),
            ];

            expect(() => {
                evaluateTimeline(component, actions, 1, new Map(), new Map());
            }).not.toThrow();
        });
    });

    describe('class actions', () => {
        it('should add a class when currentTime reaches addclass', () => {
            const el = document.createElement('div');

            const component = createComponent({ box: el });
            const actions: TimelineAction[] = [
                action({ action: 'addclass', ref: 'box', time: 1, class: 'highlight' }),
            ];

            evaluateTimeline(component, actions, 1.5, new Map(), new Map());

            expect(el.classList.contains('highlight')).toBe(true);
        });

        it('should remove a class when currentTime reaches removeclass', () => {
            const el = document.createElement('div');
            el.classList.add('active');

            const component = createComponent({ box: el });
            const actions: TimelineAction[] = [
                action({ action: 'addclass', ref: 'box', time: 0, class: 'active' }),
                action({ action: 'removeclass', ref: 'box', time: 2, class: 'active' }),
            ];

            evaluateTimeline(component, actions, 3, new Map(), new Map());

            expect(el.classList.contains('active')).toBe(false);
        });

        it('should not add class before its time', () => {
            const el = document.createElement('div');

            const component = createComponent({ box: el });
            const actions: TimelineAction[] = [
                action({ action: 'removeclass', ref: 'box', time: 0, class: 'fade' }),
                action({ action: 'addclass', ref: 'box', time: 5, class: 'fade' }),
            ];

            evaluateTimeline(component, actions, 2, new Map(), new Map());

            expect(el.classList.contains('fade')).toBe(false);
        });
    });

    describe('tooltip actions', () => {
        it('should set title attribute at the correct time', () => {
            const el = document.createElement('div');

            const component = createComponent({ icon: el });
            const actions: TimelineAction[] = [
                action({ action: 'tooltip', ref: 'icon', time: 1, value: 'Click me' }),
            ];

            evaluateTimeline(component, actions, 2, new Map(), new Map());

            expect(el.getAttribute('title')).toBe('Click me');
        });

        it('should not set title before the action time', () => {
            const el = document.createElement('div');

            const component = createComponent({ icon: el });
            const actions: TimelineAction[] = [
                action({ action: 'tooltip', ref: 'icon', time: 5, value: 'Later' }),
            ];

            evaluateTimeline(component, actions, 2, new Map(), new Map());

            expect(el.hasAttribute('title')).toBe(false);
        });

        it('should clear previous tooltip before re-evaluating', () => {
            const el = document.createElement('div');
            el.setAttribute('title', 'Old tooltip');

            const component = createComponent({ icon: el });
            const actions: TimelineAction[] = [
                action({ action: 'tooltip', ref: 'icon', time: 5, value: 'New tooltip' }),
            ];

            evaluateTimeline(component, actions, 2, new Map(), new Map());

            expect(el.hasAttribute('title')).toBe(false);
        });
    });

    describe('typewriter actions', () => {
        it('should show no text before the start time', () => {
            const el = document.createElement('p');

            const component = createComponent({ text: el });
            const actions: TimelineAction[] = [
                action({ action: 'type', ref: 'text', time: 2 }),
            ];
            const textMap = new Map([['text', 'Hello']]);

            evaluateTimeline(component, actions, 1, textMap, new Map());

            expect(el.textContent).toBe('');
        });

        it('should show partial text during animation', () => {
            const el = document.createElement('p');

            const component = createComponent({ text: el });
            const actions: TimelineAction[] = [
                action({ action: 'type', ref: 'text', time: 0 }),
            ];
            const textMap = new Map([['text', 'Hello world']]);

            evaluateTimeline(component, actions, 0.15, textMap, new Map());

            expect(el.textContent).toBe('Hel');
        });

        it('should show full text after animation completes', () => {
            const el = document.createElement('p');

            const component = createComponent({ text: el });
            const actions: TimelineAction[] = [
                action({ action: 'type', ref: 'text', time: 0 }),
            ];
            const textMap = new Map([['text', 'Hi']]);

            evaluateTimeline(component, actions, 10, textMap, new Map());

            expect(el.textContent).toBe('Hi');
        });

        it('should respect a custom speed', () => {
            const el = document.createElement('p');

            const component = createComponent({ text: el });
            const actions: TimelineAction[] = [
                action({ action: 'type', ref: 'text', time: 0, speed: 100 }),
            ];
            const textMap = new Map([['text', 'ABCDE']]);

            evaluateTimeline(component, actions, 0.25, textMap, new Map());

            expect(el.textContent).toBe('AB');
        });
    });

    describe('typehtml actions', () => {
        it('should show no HTML before the start time', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 3 }),
            ];
            const htmlMap = new Map([['code', '<b>Hi</b>']]);

            evaluateTimeline(component, actions, 1, new Map(), htmlMap);

            expect(el.innerHTML).toBe('');
        });

        it('should slice HTML respecting tag boundaries', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 0 }),
            ];
            const htmlMap = new Map([['code', '<span>AB</span>CD']]);

            evaluateTimeline(component, actions, 0.075, new Map(), htmlMap);

            const text = el.textContent;
            expect(text).toBe('ABC');
        });

        it('should properly close open tags when slicing', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 0 }),
            ];
            const htmlMap = new Map([['code', '<b>Hello</b>']]);

            evaluateTimeline(component, actions, 0.05, new Map(), htmlMap);

            expect(el.innerHTML).toContain('</b>');
        });

        it('should show full HTML after animation completes', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 0 }),
            ];
            const htmlMap = new Map([['code', '<em>Done</em>']]);

            evaluateTimeline(component, actions, 10, new Map(), htmlMap);

            expect(el.innerHTML).toBe('<em>Done</em>');
        });

        it('should count HTML entities as a single visible character', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 0 }),
            ];
            const htmlMap = new Map([['code', 'A&amp;B']]);

            evaluateTimeline(component, actions, 0.05, new Map(), htmlMap);

            expect(el.innerHTML).toBe('A&amp;');
        });

        it('should handle bare ampersands without semicolons', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 0 }),
            ];
            const htmlMap = new Map([['code', 'A & B & C']]);

            evaluateTimeline(component, actions, 0.075, new Map(), htmlMap);

            expect(el.textContent).toBe('A &');
        });

        it('should include trailing tags at the slice boundary', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 0 }),
            ];
            const htmlMap = new Map([['code', 'AB</span>CD']]);

            evaluateTimeline(component, actions, 0.05, new Map(), htmlMap);

            expect(el.innerHTML).toBe('AB');
        });

        it('should handle self-closing tags', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 0 }),
            ];
            const htmlMap = new Map([['code', 'A<br/>B']]);

            evaluateTimeline(component, actions, 0.05, new Map(), htmlMap);

            expect(el.textContent).toBe('AB');
        });

        it('should respect a custom typehtml speed', () => {
            const el = document.createElement('pre');

            const component = createComponent({ code: el });
            const actions: TimelineAction[] = [
                action({ action: 'typehtml', ref: 'code', time: 0, speed: 100 }),
            ];
            const htmlMap = new Map([['code', 'ABCDEF']]);

            evaluateTimeline(component, actions, 0.25, new Map(), htmlMap);

            expect(el.textContent).toBe('AB');
        });
    });

    describe('seeking backwards', () => {
        it('should recompute visibility when seeking to an earlier time', () => {
            const el = document.createElement('div');

            const component = createComponent({ box: el });
            const actions: TimelineAction[] = [
                action({ action: 'hide', ref: 'box', time: 0 }),
                action({ action: 'show', ref: 'box', time: 3 }),
            ];

            evaluateTimeline(component, actions, 5, new Map(), new Map());
            expect(el.hasAttribute('p-timeline-hidden')).toBe(false);

            evaluateTimeline(component, actions, 1, new Map(), new Map());
            expect(el.hasAttribute('p-timeline-hidden')).toBe(true);
        });

        it('should recompute typewriter text when seeking backwards', () => {
            const el = document.createElement('p');

            const component = createComponent({ text: el });
            const actions: TimelineAction[] = [
                action({ action: 'type', ref: 'text', time: 0 }),
            ];
            const textMap = new Map([['text', 'ABCDE']]);

            evaluateTimeline(component, actions, 10, textMap, new Map());
            expect(el.textContent).toBe('ABCDE');

            evaluateTimeline(component, actions, 0.1, textMap, new Map());
            expect(el.textContent).toBe('AB');
        });
    });
});

describe('evaluateAnchors', () => {
    function createAnchorComponent(): { component: PikoComponent; container: HTMLElement; shadowRoot: ShadowRoot } {
        const host = document.createElement('div');
        const sr = host.attachShadow({ mode: 'open' });
        const container = document.createElement('div');
        sr.appendChild(container);

        const component = host as unknown as PikoComponent;
        Object.defineProperty(component, 'refs', { value: {}, configurable: true, writable: true });
        Object.defineProperty(component, 'onAfterRender', { value: vi.fn(), configurable: true });
        Object.defineProperty(component, 'onDisconnected', { value: vi.fn(), configurable: true });

        return { component, container, shadowRoot: sr };
    }

    it('should return early when there is no shadowRoot', () => {
        const component = createComponent({});

        expect(() => {
            evaluateAnchors(component);
        }).not.toThrow();
    });

    it('should return early when all children are STYLE elements', () => {
        const host = document.createElement('div');
        const sr = host.attachShadow({ mode: 'open' });
        const style = document.createElement('style');
        sr.appendChild(style);

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target');
        sr.appendChild(anchored);

        const component = host as unknown as PikoComponent;
        Object.defineProperty(component, 'refs', { value: {}, configurable: true });
        Object.defineProperty(component, 'onAfterRender', { value: vi.fn(), configurable: true });
        Object.defineProperty(component, 'onDisconnected', { value: vi.fn(), configurable: true });

        expect(() => {
            evaluateAnchors(component);
        }).not.toThrow();

        expect(anchored.style.top).toBe('');
    });

    it('should return early when no anchored elements exist', () => {
        const { component } = createAnchorComponent();

        expect(() => {
            evaluateAnchors(component);
        }).not.toThrow();
    });

    it('should skip hidden anchored elements', () => {
        const { component, container, shadowRoot } = createAnchorComponent();

        const target = document.createElement('div');
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { target };

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target');
        anchored.setAttribute('p-timeline-hidden', '');
        container.appendChild(anchored);
        shadowRoot.appendChild(container);

        evaluateAnchors(component);

        expect(anchored.style.top).toBe('');
    });

    it('should clear styles when the target ref is missing', () => {
        const { component, container } = createAnchorComponent();

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'missing');
        anchored.style.top = '10px';
        anchored.style.left = '20px';
        container.appendChild(anchored);

        evaluateAnchors(component);

        expect(anchored.style.top).toBe('');
        expect(anchored.style.left).toBe('');
    });

    it('should clear styles when the target is hidden', () => {
        const { component, container } = createAnchorComponent();

        const target = document.createElement('div');
        target.setAttribute('p-timeline-hidden', '');
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { target };

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target');
        anchored.style.top = '10px';
        container.appendChild(anchored);

        evaluateAnchors(component);

        expect(anchored.style.top).toBe('');
    });

    it('should position bottom-right relative to target', () => {
        const { component, container } = createAnchorComponent();

        const target = document.createElement('div');
        target.getBoundingClientRect = () => ({
            x: 100, y: 50, width: 200, height: 30, top: 50, right: 300, bottom: 80, left: 100, toJSON: vi.fn(),
        });
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { target };

        container.getBoundingClientRect = () => ({
            x: 0, y: 0, width: 800, height: 600, top: 0, right: 800, bottom: 600, left: 0, toJSON: vi.fn(),
        });

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target bottom-right');
        Object.defineProperty(anchored, 'offsetWidth', { value: 120, configurable: true });
        Object.defineProperty(anchored, 'offsetHeight', { value: 30, configurable: true });
        container.appendChild(anchored);

        evaluateAnchors(component);

        expect(parseFloat(anchored.style.left)).toBe(180);
    });

    it('should flip to top when bottom would overflow', () => {
        const { component, container } = createAnchorComponent();

        const target = document.createElement('div');
        target.getBoundingClientRect = () => ({
            x: 50, y: 560, width: 100, height: 30, top: 560, right: 150, bottom: 590, left: 50, toJSON: vi.fn(),
        });
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { target };

        container.getBoundingClientRect = () => ({
            x: 0, y: 0, width: 800, height: 600, top: 0, right: 800, bottom: 600, left: 0, toJSON: vi.fn(),
        });

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target bottom-left');
        Object.defineProperty(anchored, 'offsetWidth', { value: 120, configurable: true });
        Object.defineProperty(anchored, 'offsetHeight', { value: 30, configurable: true });
        container.appendChild(anchored);

        evaluateAnchors(component);

        expect(parseFloat(anchored.style.top)).toBeLessThan(560);
    });

    it('should flip to bottom when top would overflow', () => {
        const { component, container } = createAnchorComponent();

        const target = document.createElement('div');
        target.getBoundingClientRect = () => ({
            x: 50, y: 10, width: 100, height: 20, top: 10, right: 150, bottom: 30, left: 50, toJSON: vi.fn(),
        });
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { target };

        container.getBoundingClientRect = () => ({
            x: 0, y: 0, width: 800, height: 600, top: 0, right: 800, bottom: 600, left: 0, toJSON: vi.fn(),
        });

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target top-left');
        Object.defineProperty(anchored, 'offsetWidth', { value: 120, configurable: true });
        Object.defineProperty(anchored, 'offsetHeight', { value: 30, configurable: true });
        container.appendChild(anchored);

        evaluateAnchors(component);

        expect(parseFloat(anchored.style.top)).toBeGreaterThan(30);
    });

    it('should clamp left when element would overflow right edge', () => {
        const { component, container } = createAnchorComponent();

        const target = document.createElement('div');
        target.getBoundingClientRect = () => ({
            x: 750, y: 50, width: 40, height: 30, top: 50, right: 790, bottom: 80, left: 750, toJSON: vi.fn(),
        });
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { target };

        container.getBoundingClientRect = () => ({
            x: 0, y: 0, width: 800, height: 600, top: 0, right: 800, bottom: 600, left: 0, toJSON: vi.fn(),
        });

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target bottom-left');
        Object.defineProperty(anchored, 'offsetWidth', { value: 200, configurable: true });
        Object.defineProperty(anchored, 'offsetHeight', { value: 30, configurable: true });
        container.appendChild(anchored);

        evaluateAnchors(component);

        expect(parseFloat(anchored.style.left)).toBeLessThanOrEqual(594);
    });

    it('should clamp top when element overflows in both directions', () => {
        const { component, container } = createAnchorComponent();

        const target = document.createElement('div');
        target.getBoundingClientRect = () => ({
            x: 50, y: 280, width: 100, height: 30, top: 280, right: 150, bottom: 310, left: 50, toJSON: vi.fn(),
        });
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { target };

        container.getBoundingClientRect = () => ({
            x: 0, y: 0, width: 800, height: 320, top: 0, right: 800, bottom: 320, left: 0, toJSON: vi.fn(),
        });

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target bottom-left');
        Object.defineProperty(anchored, 'offsetWidth', { value: 120, configurable: true });
        Object.defineProperty(anchored, 'offsetHeight', { value: 300, configurable: true });
        container.appendChild(anchored);

        evaluateAnchors(component);

        expect(parseFloat(anchored.style.top)).toBeLessThanOrEqual(314);
    });

    it('should set position styles for visible anchored elements', () => {
        const { component, container } = createAnchorComponent();

        const target = document.createElement('div');
        target.getBoundingClientRect = () => ({
            x: 50, y: 50, width: 100, height: 30, top: 50, right: 150, bottom: 80, left: 50, toJSON: vi.fn(),
        });
        (component as unknown as { refs: Record<string, HTMLElement> }).refs = { target };

        container.getBoundingClientRect = () => ({
            x: 0, y: 0, width: 800, height: 600, top: 0, right: 800, bottom: 600, left: 0, toJSON: vi.fn(),
        });

        const anchored = document.createElement('div');
        anchored.setAttribute('p-timeline-anchor', 'target bottom-left');
        Object.defineProperty(anchored, 'offsetWidth', { value: 120, configurable: true });
        Object.defineProperty(anchored, 'offsetHeight', { value: 30, configurable: true });
        container.appendChild(anchored);

        evaluateAnchors(component);

        expect(anchored.style.top).not.toBe('');
        expect(anchored.style.left).not.toBe('');
        expect(anchored.style.right).toBe('auto');
        expect(anchored.style.bottom).toBe('auto');
    });
});
