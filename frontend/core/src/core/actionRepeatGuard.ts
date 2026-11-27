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
 * Attribute that lets a trigger element or form send the same action again
 * while an earlier send is still in flight.
 */
export const ALLOW_REPEAT_ATTR = 'pk-allow-repeat';

/** ARIA attribute set on the trigger and form while an action is in flight. */
const ARIA_BUSY_ATTR = 'aria-busy';

/**
 * Longest time, in milliseconds, a guard is held when the action sets no
 * longer timeout. After it, a repeat is allowed even if the first send never
 * settled, so a guard can never stay stuck.
 */
export const DEFAULT_GUARD_HOLD_MS = 60_000;

/** Describes one guarded send of an action. */
export interface GuardedSend {
    /** The server action name. */
    actionName: string;
    /** The element the action is bound to, which is the trigger or the form for a submit. */
    element: HTMLElement;
    /** The browser event that started the send, if any. */
    event?: Event;
    /** How long, in milliseconds, the guard may be held before it is released. */
    holdMilliseconds: number;
}

/** Counts the in-flight sends that mark one element as busy. */
interface BusyMark {
    /** The number of in-flight sends that marked the element. */
    holds: number;
    /** Whether the guard added aria-busy, so it only removes its own attribute. */
    addedAttribute: boolean;
}

/**
 * Ignores a second send of an action from the same element while the first
 * is in flight, and marks the trigger and form with aria-busy meanwhile.
 *
 * All state is held in WeakMaps keyed by element, so removed elements are
 * freed with no clean-up.
 */
export class ActionRepeatGuard {
    /** Holds, for each element, the token of every action in flight from it. */
    private readonly inFlight = new WeakMap<HTMLElement, Map<string, object>>();

    /** Holds the busy marks the guard placed on elements. */
    private readonly busyMarks = new WeakMap<HTMLElement, BusyMark>();

    /**
     * Runs a send unless the same action is already in flight from the same
     * element. The guard is released when the send settles, whether it
     * succeeds, fails or throws, or when holdMilliseconds pass.
     *
     * @param send - Describes the action, its element and its event.
     * @param run - Sends the action.
     * @returns A promise that settles with the send, or at once when the send is ignored.
     */
    async run(send: GuardedSend, run: () => Promise<void>): Promise<void> {
        if (allowsRepeat(send)) {
            return run();
        }
        if (this.isInFlight(send.element, send.actionName)) {
            return;
        }

        const release = this.acquire(send);
        try {
            await run();
        } finally {
            release();
        }
    }

    /**
     * Reports whether an action is in flight from an element.
     *
     * @param element - The element the action is bound to.
     * @param actionName - The server action name.
     * @returns True while a guarded send of that action from that element is in flight.
     */
    isInFlight(element: HTMLElement, actionName: string): boolean {
        return this.inFlight.get(element)?.has(actionName) ?? false;
    }

    /**
     * Holds the guard for a send and marks its elements as busy.
     *
     * @param send - The send to guard.
     * @returns A function that releases the guard. Calling it again does nothing.
     */
    private acquire(send: GuardedSend): () => void {
        const token = {};
        let actions = this.inFlight.get(send.element);
        if (!actions) {
            actions = new Map<string, object>();
            this.inFlight.set(send.element, actions);
        }
        actions.set(send.actionName, token);

        const busyElements = busyElementsOf(send);
        busyElements.forEach(element => this.markBusy(element));

        let released = false;
        const release = (): void => {
            if (released) {
                return;
            }
            released = true;
            clearTimeout(timer);
            this.dropToken(send.element, send.actionName, token);
            busyElements.forEach(element => this.unmarkBusy(element));
        };
        const timer = setTimeout(release, send.holdMilliseconds);
        return release;
    }

    /**
     * Removes an action's token from an element, unless a later send now owns it.
     *
     * @param element - The element the action is bound to.
     * @param actionName - The server action name.
     * @param token - The token of the send being released.
     */
    private dropToken(element: HTMLElement, actionName: string, token: object): void {
        const actions = this.inFlight.get(element);
        if (actions?.get(actionName) !== token) {
            return;
        }
        actions.delete(actionName);
        if (actions.size === 0) {
            this.inFlight.delete(element);
        }
    }

    /**
     * Adds one busy hold to an element, setting aria-busy on the first.
     *
     * @param element - The element to mark.
     */
    private markBusy(element: HTMLElement): void {
        const mark = this.busyMarks.get(element);
        if (mark) {
            mark.holds++;
            return;
        }
        const addedAttribute = !element.hasAttribute(ARIA_BUSY_ATTR);
        if (addedAttribute) {
            element.setAttribute(ARIA_BUSY_ATTR, 'true');
        }
        this.busyMarks.set(element, {holds: 1, addedAttribute});
    }

    /**
     * Removes one busy hold from an element, removing aria-busy after the last
     * when the guard added it.
     *
     * @param element - The element to unmark.
     */
    private unmarkBusy(element: HTMLElement): void {
        const mark = this.busyMarks.get(element);
        if (!mark) {
            return;
        }
        mark.holds--;
        if (mark.holds > 0) {
            return;
        }
        this.busyMarks.delete(element);
        if (mark.addedAttribute) {
            element.removeAttribute(ARIA_BUSY_ATTR);
        }
    }
}

/** Trigger elements of submit events started by a helper, keyed by event. */
const helperSubmitTriggers = new WeakMap<Event, HTMLElement>();

/**
 * Submits a form for a trigger element that is not its submit button, such
 * as a Save button outside the form, and records that element as the
 * trigger of the submit event.
 *
 * Uses requestSubmit, so constraint validation and submit handlers run as
 * for a native submit.
 *
 * @param form - The form to submit.
 * @param trigger - The element that asked for the submit.
 */
export function requestSubmitFrom(form: HTMLFormElement, trigger: HTMLElement): void {
    const ownerDocument = form.ownerDocument;
    const recordTrigger = (event: Event): void => {
        if (event.target === form) {
            helperSubmitTriggers.set(event, trigger);
        }
    };
    ownerDocument.addEventListener('submit', recordTrigger, {capture: true});
    try {
        form.requestSubmit();
    } finally {
        ownerDocument.removeEventListener('submit', recordTrigger, {capture: true});
    }
}

/**
 * Returns the guard hold time for an action.
 *
 * @param timeoutMilliseconds - The action's own request timeout, if any.
 * @returns The longer of the action timeout and DEFAULT_GUARD_HOLD_MS.
 */
export function guardHoldMilliseconds(timeoutMilliseconds?: number): number {
    return Math.max(DEFAULT_GUARD_HOLD_MS, timeoutMilliseconds ?? 0);
}

/**
 * Finds the element that started a submit, which is its submit button or the
 * element recorded by requestSubmitFrom.
 *
 * @param event - The event that started the send, if any.
 * @returns The trigger element, or null when the event is not a submit or names none.
 */
function submitTriggerOf(event?: Event): HTMLElement | null {
    if (event?.type !== 'submit') {
        return null;
    }
    const submitter = (event as SubmitEvent).submitter;
    if (submitter instanceof HTMLElement) {
        return submitter;
    }
    return helperSubmitTriggers.get(event) ?? null;
}

/**
 * Lists the elements marked busy while a send is in flight. These are the bound
 * element and, for a submit, the element that started it.
 *
 * @param send - The guarded send.
 * @returns The distinct elements to mark.
 */
function busyElementsOf(send: GuardedSend): HTMLElement[] {
    const trigger = submitTriggerOf(send.event);
    if (trigger && trigger !== send.element) {
        return [send.element, trigger];
    }
    return [send.element];
}

/**
 * Reports whether a send opted out of the guard, through the opt-out
 * attribute on its element, its submit trigger or its form.
 *
 * @param send - The guarded send.
 * @returns True when repeated sends are allowed.
 */
function allowsRepeat(send: GuardedSend): boolean {
    const candidates = [send.element, submitTriggerOf(send.event), send.element.closest('form')];
    return candidates.some(candidate => candidate?.hasAttribute(ALLOW_REPEAT_ATTR) === true);
}
