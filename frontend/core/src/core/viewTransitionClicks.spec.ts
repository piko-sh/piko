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

import {describe, it, expect, beforeEach, afterEach, vi, type Mock} from 'vitest';
import {replayClicksLostToTransition, type ActiveViewTransition} from './viewTransitionClicks';

interface FakeTransition extends ActiveViewTransition {
    end(): Promise<void>;
    fail(): Promise<void>;
    skipTransition: Mock<() => void>;
}

function createFakeTransition(): FakeTransition {
    let resolve: () => void = () => undefined;
    let reject: (reason: unknown) => void = () => undefined;
    const finished = new Promise<void>((resolveFinished, rejectFinished) => {
        resolve = resolveFinished;
        reject = rejectFinished;
    });
    return {
        finished,
        skipTransition: vi.fn<() => void>(),
        async end() {
            resolve();
            await flushPromises();
        },
        async fail() {
            reject(new Error('transition failed'));
            await flushPromises();
        }
    };
}

async function flushPromises(): Promise<void> {
    for (let round = 0; round < 5; round++) {
        await Promise.resolve();
    }
}

function clickRootElement(init: MouseEventInit = {}): MouseEvent {
    const click = new MouseEvent('click', {bubbles: true, cancelable: true, detail: 1, clientX: 20, clientY: 30, ...init});
    document.documentElement.dispatchEvent(click);
    return click;
}

function stubElementFromPoint(find: () => Element | null): void {
    Object.defineProperty(document, 'elementFromPoint', {value: vi.fn(find), configurable: true, writable: true});
}

describe('replayClicksLostToTransition', () => {
    let target: HTMLAnchorElement;
    let clicks: MouseEvent[];
    let listeners: AbortController;

    beforeEach(() => {
        document.body.innerHTML = '<a id="target" href="#next">Next</a>';
        target = document.getElementById('target') as HTMLAnchorElement;
        clicks = [];
        listeners = new AbortController();
        document.body.addEventListener('click', event => {
            clicks.push(event);
            event.preventDefault();
        }, {signal: listeners.signal});
        stubElementFromPoint(() => document.getElementById('target'));
    });

    afterEach(() => {
        listeners.abort();
        Reflect.deleteProperty(document, 'elementFromPoint');
        document.body.innerHTML = '';
    });

    it('ends the transition and passes the click on to the element under the pointer', async () => {
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        const lost = clickRootElement({ctrlKey: true, metaKey: true});
        expect(lost.defaultPrevented).toBe(true);
        expect(transition.skipTransition).toHaveBeenCalled();
        expect(clicks).toHaveLength(0);

        await transition.end();

        expect(clicks).toHaveLength(1);
        expect(clicks[0].target).toBe(target);
        expect(clicks[0].ctrlKey).toBe(true);
        expect(clicks[0].metaKey).toBe(true);
        expect(clicks[0].clientX).toBe(20);
        expect(document.elementFromPoint).toHaveBeenCalledWith(20, 30);
        expect(document.activeElement).toBe(target);
    });

    it('passes the click on to the element of the page shown after the update', async () => {
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);
        clickRootElement();

        document.body.innerHTML = '<a id="target" href="#other">New page link</a>';
        const newPageLink = document.getElementById('target');
        const newPageClicks: EventTarget[] = [];
        newPageLink?.addEventListener('click', event => {
            newPageClicks.push(event.currentTarget as EventTarget);
            event.preventDefault();
        });
        await transition.end();

        expect(newPageClicks).toEqual([newPageLink]);
    });

    it('passes on only one click per transition', async () => {
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        clickRootElement();
        clickRootElement();
        await transition.end();
        clickRootElement();

        expect(clicks).toHaveLength(1);
        expect(clicks[0].target).toBe(target);
    });

    it('passes on a click that arrives after the transition ended but was pressed during it', async () => {
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        document.documentElement.dispatchEvent(new MouseEvent('pointerdown', {bubbles: true}));
        expect(transition.skipTransition).toHaveBeenCalled();
        await transition.end();
        clickRootElement();

        expect(clicks).toHaveLength(1);
        expect(clicks[0].target).toBe(target);
    });

    it('removes its listeners when the transition ends', async () => {
        const removeEventListener = vi.spyOn(window, 'removeEventListener');
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        await transition.end();
        const later = clickRootElement();

        expect(later.defaultPrevented).toBe(false);
        expect(clicks).toHaveLength(0);
        expect(removeEventListener.mock.calls.map(call => call[0]).sort()).toEqual(['click', 'pointercancel', 'pointerdown']);
        removeEventListener.mockRestore();
    });

    it('removes its listeners when the transition fails', async () => {
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        clickRootElement();
        await transition.fail();
        const later = clickRootElement();

        expect(clicks).toHaveLength(1);
        expect(later.defaultPrevented).toBe(false);
    });

    it('leaves a click on a real element alone', async () => {
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        target.dispatchEvent(new MouseEvent('click', {bubbles: true, cancelable: true, detail: 1}));
        await transition.end();

        expect(clicks).toHaveLength(1);
        expect(transition.skipTransition).not.toHaveBeenCalled();
    });

    it('leaves a click from the keyboard alone', async () => {
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        const keyboardClick = clickRootElement({detail: 0});
        await transition.end();

        expect(keyboardClick.defaultPrevented).toBe(false);
        expect(clicks).toHaveLength(0);
    });

    it('does nothing when the point is outside the viewport', async () => {
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        clickRootElement({clientX: -5, clientY: 30});
        await transition.end();

        expect(clicks).toHaveLength(0);
        expect(document.elementFromPoint).not.toHaveBeenCalled();
    });

    it('does nothing when no element is under the pointer any more', async () => {
        stubElementFromPoint(() => null);
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        clickRootElement();
        await transition.end();

        expect(clicks).toHaveLength(0);
    });

    it('passes the click on to a button inside a component\'s shadow root', async () => {
        const host = document.createElement('div');
        host.id = 'shadow-host';
        const shadowRoot = host.attachShadow({mode: 'open'});
        shadowRoot.innerHTML = '<button id="inner-next" type="button">Next</button>';
        document.body.appendChild(host);
        const innerButton = shadowRoot.getElementById('inner-next') as HTMLButtonElement;
        const innerClicks: EventTarget[] = [];
        innerButton.addEventListener('click', event => innerClicks.push(event.currentTarget as EventTarget));
        const originsSeenOutside: EventTarget[] = [];
        document.body.addEventListener('click', event => originsSeenOutside.push(event.composedPath()[0]), {signal: listeners.signal});
        stubElementFromPoint(() => host);
        Object.defineProperty(shadowRoot, 'elementFromPoint', {value: vi.fn(() => innerButton), configurable: true});
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        clickRootElement();
        await transition.end();

        expect(innerClicks).toEqual([innerButton]);
        expect(clicks).toHaveLength(1);
        expect(originsSeenOutside).toEqual([innerButton]);
        expect(shadowRoot.activeElement).toBe(innerButton);
    });

    it('stops at the shadow host when its shadow root finds only the host', async () => {
        const host = document.createElement('div');
        const shadowRoot = host.attachShadow({mode: 'open'});
        document.body.appendChild(host);
        stubElementFromPoint(() => host);
        Object.defineProperty(shadowRoot, 'elementFromPoint', {value: vi.fn(() => host), configurable: true});
        const transition = createFakeTransition();
        replayClicksLostToTransition(transition);

        clickRootElement();
        await transition.end();

        expect(clicks).toHaveLength(1);
        expect(clicks[0].target).toBe(host);
    });
});
