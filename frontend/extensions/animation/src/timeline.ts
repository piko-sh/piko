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

import {evaluateTimeline, captureTypewriterTexts, captureTypehtmlContents, evaluateAnchors} from './actions';

/** Describes a single action within a timeline animation sequence. */
interface TimelineAction {
    /** Time in seconds at which the action fires. */
    time: number;
    /** The kind of action to perform on the target element. */
    action: string;
    /** The component ref name identifying the target element. */
    ref: string;
    /** Typing speed in milliseconds per character for type/typehtml actions. */
    speed?: number;
    /** CSS class name for addclass/removeclass actions. */
    class?: string;
    /** Arbitrary string value used by tooltip actions. */
    value?: string;
    /** Additional parameters for custom (user-defined) actions. */
    params?: Record<string, string>;
}

/** Minimal interface for a PKC component used by the animation extension. */
interface PikoComponent extends HTMLElement {
    /** Map of p-ref names to their corresponding DOM elements. */
    refs?: Record<string, HTMLElement>;
    /** Registers a callback invoked after the component's first render. */
    onAfterRender(callback: () => void): void;
    /** Registers a callback invoked when the component is removed from the DOM. */
    onDisconnected(callback: () => void): void;
}

/** A responsive timeline entry that associates a media query with a set of actions. */
interface TimelineEntry {
    /** CSS media query string, or null for the default fallback entry. */
    media: string | null;
    /** The timeline actions to use when the media query matches. */
    actions: TimelineAction[];
}

/**
 * Resolves the $$timeline static property into an array of TimelineAction.
 *
 * Handles two formats:
 * - Flat array of actions (single timeline, no media attribute).
 * - Array of {media, actions} objects (responsive timelines).
 *
 * For responsive timelines, the first entry whose media query matches the
 * current viewport is selected. Entries with media: null act as the default
 * fallback. The narrowest (most specific) matching entry wins because
 * entries are checked in source order, so authors should place the most
 * specific media query first and the default last.
 *
 * @param raw - The raw timeline data from the component's static property.
 * @returns The resolved array of actions, or null if the input is empty or invalid.
 */
function resolveTimeline(raw: TimelineAction[] | TimelineEntry[]): TimelineAction[] | null {
    if (!Array.isArray(raw) || raw.length === 0) {
        return null;
    }

    if ('time' in raw[0]) {
        return raw as TimelineAction[];
    }

    const entries = raw as TimelineEntry[];
    let fallback: TimelineAction[] | null = null;

    for (const entry of entries) {
        if (entry.media == null || entry.media === '') {
            fallback = entry.actions;
            continue;
        }
        if (window.matchMedia(entry.media).matches) {
            return entry.actions;
        }
    }

    return fallback;
}

/**
 * Sets up declarative timeline animation for a PKC component.
 *
 * Reads the static `$$timeline` property from the component's constructor,
 * sorts actions by time, and wires up lifecycle hooks to evaluate the
 * timeline whenever the `time` prop changes. All evaluation is a pure
 * function of currentTime, so seeking backwards works correctly.
 *
 * For responsive timelines (multiple piko:timeline blocks with media
 * attributes), the matching timeline is selected on initialisation and
 * swapped live when the viewport changes.
 *
 * @param component - The PKC component element to set up animation on.
 */
export function setupAnimation(component: HTMLElement): void {
    const constructor = component.constructor as { $$timeline?: TimelineAction[] | TimelineEntry[] };
    const rawTimeline = constructor.$$timeline;
    if (!rawTimeline || !Array.isArray(rawTimeline)) {
        console.warn(`Animation: no $$timeline data found on ${component.tagName}.`);
        return;
    }

    const validatedTimeline: TimelineAction[] | TimelineEntry[] = rawTimeline;

    let sortedActions: TimelineAction[] = [];
    let typewriterTexts = new Map<string, string>();
    let typehtmlContents = new Map<string, string>();
    let initialised = false;
    let captured = false;

    const ppEl = component as PikoComponent;

    function activateTimeline(): void {
        const actions = resolveTimeline(validatedTimeline);
        if (!actions) {return;}
        sortedActions = [...actions].sort((a, b) => a.time - b.time);
        typewriterTexts = new Map<string, string>();
        typehtmlContents = new Map<string, string>();
        captured = false;
    }

    activateTimeline();

    const isResponsive = rawTimeline.length > 0 && 'media' in rawTimeline[0];
    const mediaListeners: Array<{ mql: MediaQueryList; handler: () => void }> = [];

    if (isResponsive) {
        const entries = rawTimeline as TimelineEntry[];
        for (const entry of entries) {
            if (entry.media == null || entry.media === '') {continue;}
            const mql = window.matchMedia(entry.media);
            const handler = () => {
                activateTimeline();
                if (initialised) {
                    captureTypewriterTexts(ppEl, sortedActions, typewriterTexts);
                    captureTypehtmlContents(ppEl, sortedActions, typehtmlContents);
                    captured = true;
                    const currentTime = parseFloat(ppEl.getAttribute('time') ?? '0');
                    evaluateTimeline(ppEl, sortedActions, currentTime, typewriterTexts, typehtmlContents);
                    evaluateAnchors(ppEl);
                }
            };
            mql.addEventListener('change', handler);
            mediaListeners.push({mql, handler});
        }
    }

    const observer = new MutationObserver(() => {
        if (!initialised) {return;}
        if (!captured) {
            captured = true;
            captureTypewriterTexts(ppEl, sortedActions, typewriterTexts);
            captureTypehtmlContents(ppEl, sortedActions, typehtmlContents);
        }
        const currentTime = parseFloat(ppEl.getAttribute('time') ?? '0');
        evaluateTimeline(ppEl, sortedActions, currentTime, typewriterTexts, typehtmlContents);
        evaluateAnchors(ppEl);
    });

    ppEl.onAfterRender(() => {
        if (initialised) {return;}
        initialised = true;
        observer.observe(ppEl, { attributes: true, attributeFilter: ['time'] });
    });

    ppEl.onDisconnected(() => {
        observer.disconnect();
        for (const {mql, handler} of mediaListeners) {
            mql.removeEventListener('change', handler);
        }
    });
}

export type {TimelineAction, PikoComponent};
