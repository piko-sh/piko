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

/**
 * The parts of a running View Transition that the click replay uses.
 */
export interface ActiveViewTransition {
    /** Settles once the transition has ended, whether it ran, was skipped or failed. */
    readonly finished: Promise<unknown>;
    /** Ends the animation at once. The DOM update still runs. */
    skipTransition(): void;
}

/**
 * The parts of a click that are passed on to the element under the pointer.
 */
interface CapturedClick {
    clientX: number;
    clientY: number;
    screenX: number;
    screenY: number;
    button: number;
    ctrlKey: boolean;
    shiftKey: boolean;
    altKey: boolean;
    metaKey: boolean;
}

/** Selects elements that take focus when a pointer activates them. */
const FOCUSABLE_SELECTOR = 'a[href], button, input, select, textarea, summary, [tabindex]';

/** The most shadow roots a click is passed down through, so a faulty page cannot loop. */
const MAX_SHADOW_DEPTH = 32;

/**
 * Passes on a click that a running View Transition took away from the page.
 *
 * While a transition runs, the browser sends pointer events to the root element, so a
 * click on a link or button never reaches it. The replay listens in the capture phase,
 * ends the transition at the first pointer press, and once the transition has ended,
 * sends the click to the element under the pointer on the page now shown.
 */
class TransitionClickReplay {
    /** The click waiting for the transition to end, if any. */
    private pending: CapturedClick | null = null;
    /** Whether a click was passed on, so a second one never is. */
    private delivered = false;
    /** Whether the transition has ended. */
    private settled = false;
    /** Whether a press on the root element has not yet become a click. */
    private pressing = false;
    /** Whether the listeners were removed. */
    private removed = false;

    /**
     * Creates a replay for one transition.
     *
     * @param transition - The running transition.
     * @param doc - The document the transition runs in.
     * @param view - The window of that document.
     */
    constructor(
        private readonly transition: ActiveViewTransition,
        private readonly doc: Document,
        private readonly view: Window
    ) {}

    /** Adds the listeners and waits for the transition to end. */
    start(): void {
        this.view.addEventListener('pointerdown', this.onPointerDown, true);
        this.view.addEventListener('pointercancel', this.onPointerCancel, true);
        this.view.addEventListener('click', this.onClick, true);
        void this.waitForEnd();
    }

    /** Passes on a held click once the transition has ended, even when it failed. */
    private async waitForEnd(): Promise<void> {
        await this.transition.finished.catch(() => undefined);
        this.settled = true;
        this.deliver();
    }

    /**
     * Ends the transition when a press lands on the root element.
     *
     * @param event - The pointer press.
     */
    private readonly onPointerDown = (event: PointerEvent): void => {
        if (!this.isLostToTransition(event)) {
            this.stopIfSettled();
            return;
        }
        this.pressing = true;
        this.transition.skipTransition();
    };

    /** Forgets a press the browser cancelled, such as a touch that became a scroll. */
    private readonly onPointerCancel = (): void => {
        this.pressing = false;
        this.stopIfSettled();
    };

    /**
     * Holds a click that landed on the root element until the transition has ended.
     *
     * @param event - The click.
     */
    private readonly onClick = (event: MouseEvent): void => {
        if (!this.isLostToTransition(event) || event.detail === 0) {
            this.stopIfSettled();
            return;
        }
        event.preventDefault();
        event.stopImmediatePropagation();
        this.pressing = false;
        if (this.pending === null && !this.delivered) {
            this.pending = captureClick(event);
        }
        this.transition.skipTransition();
        if (this.settled) {
            this.deliver();
        }
    };

    /**
     * Reports whether an event was sent to the root element, where a running transition
     * sends every pointer event.
     *
     * @param event - The pointer event.
     * @returns True when the event targets the root element.
     */
    private isLostToTransition(event: Event): boolean {
        return event.target === this.doc.documentElement;
    }

    /** Sends the held click to the element now under the pointer, at most once. */
    private deliver(): void {
        const click = this.pending;
        this.pending = null;
        if (click !== null && !this.delivered) {
            this.delivered = true;
            dispatchClickAt(this.doc, this.view, click);
        }
        this.stopIfSettled();
    }

    /** Removes the listeners once the transition has ended and no press is waiting. */
    private stopIfSettled(): void {
        if (!this.settled || this.pressing || this.removed) {
            return;
        }
        this.removed = true;
        this.view.removeEventListener('pointerdown', this.onPointerDown, true);
        this.view.removeEventListener('pointercancel', this.onPointerCancel, true);
        this.view.removeEventListener('click', this.onClick, true);
    }
}

/**
 * Passes on clicks that a running View Transition takes away from the page.
 *
 * @param transition - The transition just started.
 * @param doc - The document the transition runs in.
 */
export function replayClicksLostToTransition(transition: ActiveViewTransition, doc: Document = document): void {
    const view = doc.defaultView;
    if (!view) {
        return;
    }
    new TransitionClickReplay(transition, doc, view).start();
}

/**
 * Copies the position, button and modifier keys of a click.
 *
 * @param event - The click.
 * @returns The parts of the click to pass on.
 */
function captureClick(event: MouseEvent): CapturedClick {
    return {
        clientX: event.clientX,
        clientY: event.clientY,
        screenX: event.screenX,
        screenY: event.screenY,
        button: event.button,
        ctrlKey: event.ctrlKey,
        shiftKey: event.shiftKey,
        altKey: event.altKey,
        metaKey: event.metaKey
    };
}

/**
 * Sends a click to the element at a point, focusing it first as a real click would.
 *
 * The click goes to the deepest element, inside any open shadow roots, and is composed
 * so it bubbles out of them as a real click does. Nothing happens when the point is
 * outside the viewport or only the root element is there. The click keeps its modifier
 * keys, so a ctrl or meta click on a link keeps its meaning for the page's handlers.
 *
 * @param doc - The document to click in.
 * @param view - The window whose viewport holds the point.
 * @param click - The click to pass on.
 */
function dispatchClickAt(doc: Document, view: Window, click: CapturedClick): void {
    if (!isInsideViewport(view, click)) {
        return;
    }
    const outerTarget = doc.elementFromPoint(click.clientX, click.clientY);
    if (outerTarget === null || outerTarget === doc.documentElement) {
        return;
    }
    const target = descendIntoShadowRoots(outerTarget, click);
    const focusable = target.closest<HTMLElement>(FOCUSABLE_SELECTOR);
    focusable?.focus({preventScroll: true});
    target.dispatchEvent(new MouseEvent('click', {
        ...click,
        bubbles: true,
        cancelable: true,
        composed: true,
        detail: 1
    }));
}

/**
 * Finds the deepest element at a click's point, looking inside open shadow roots.
 *
 * The document only finds the shadow host, so a click on a button inside a component
 * would reach the component instead of the button.
 *
 * @param outerTarget - The element the document found at the point.
 * @param click - The click.
 * @returns The deepest element at the point.
 */
function descendIntoShadowRoots(outerTarget: Element, click: CapturedClick): Element {
    let target = outerTarget;
    for (let depth = 0; depth < MAX_SHADOW_DEPTH; depth++) {
        const inner = elementInsideShadowRoot(target, click);
        if (inner === null) {
            return target;
        }
        target = inner;
    }
    return target;
}

/**
 * Finds the element at a click's point inside a host's open shadow root.
 *
 * @param host - The element that may host a shadow root.
 * @param click - The click.
 * @returns The inner element, or null when there is no shadow root or it finds only the host.
 */
function elementInsideShadowRoot(host: Element, click: CapturedClick): Element | null {
    const shadowRoot = host.shadowRoot;
    if (shadowRoot === null || typeof shadowRoot.elementFromPoint !== 'function') {
        return null;
    }
    const inner = shadowRoot.elementFromPoint(click.clientX, click.clientY);
    if (inner === null || inner === host) {
        return null;
    }
    return inner;
}

/**
 * Reports whether a click's point lies inside the viewport.
 *
 * @param view - The window whose viewport is checked.
 * @param click - The click.
 * @returns True when the point is inside the viewport.
 */
function isInsideViewport(view: Window, click: CapturedClick): boolean {
    return click.clientX >= 0 && click.clientY >= 0 &&
        click.clientX < view.innerWidth && click.clientY < view.innerHeight;
}
