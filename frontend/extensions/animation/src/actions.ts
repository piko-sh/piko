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

import type {TimelineAction, PikoComponent} from './timeline';

/** Default typing speed for typewriter actions, in milliseconds per character. */
const DEFAULT_TYPE_SPEED = 50;

/** Default typing speed for typehtml actions, in milliseconds per character. */
const DEFAULT_TYPEHTML_SPEED = 25;

/** Conversion factor from seconds to milliseconds. */
const MILLISECONDS_PER_SECOND = 1000;

/** Maximum length of an HTML entity reference (e.g. &amp;lt;). */
const MAX_HTML_ENTITY_LENGTH = 10;

/** Fallback width for anchored elements when offsetWidth is zero. */
const DEFAULT_ANCHOR_WIDTH = 200;

/** Fallback height for anchored elements when offsetHeight is zero. */
const DEFAULT_ANCHOR_HEIGHT = 40;

/** Padding between anchored elements and their targets/container edges. */
const ANCHOR_PADDING = 6;

/**
 * Handler function for a custom (user-defined) timeline action.
 *
 * Receives the target element (or null when the ref is missing),
 * the full action descriptor, the current time in seconds, and
 * the PKC component instance.
 */
export type CustomActionHandler = (
    el: HTMLElement | null,
    action: TimelineAction,
    currentTime: number,
    component: PikoComponent,
) => void;

/** Registry of custom action handlers, keyed by action name. */
const customHandlers = new Map<string, CustomActionHandler>();

/**
 * Registers a handler for a custom timeline action type.
 *
 * Custom actions use the piko:NAME syntax in timeline blocks. The
 * compiler passes them through with all attributes in the params
 * field. Handlers are called during evaluateTimeline when the
 * action type matches.
 *
 * @param name - The action name (without the piko: prefix).
 * @param handler - The function to call when the action is evaluated.
 */
export function registerTimelineAction(name: string, handler: CustomActionHandler): void {
    customHandlers.set(name, handler);
}

/**
 * Captures the original text content of elements targeted by type actions.
 *
 * Stores the text in the map and clears the element so the typewriter effect
 * can progressively reveal it.
 *
 * @param component - The PKC component instance with refs.
 * @param actions - The sorted list of timeline actions.
 * @param textMap - Map to store captured text, keyed by ref name.
 */
export function captureTypewriterTexts(
    component: PikoComponent,
    actions: TimelineAction[],
    textMap: Map<string, string>,
): void {
    for (const action of actions) {
        if (action.action !== 'type') {continue;}

        const el = component.refs?.[action.ref];
        if (!el) {continue;}

        textMap.set(action.ref, el.textContent);
        el.textContent = '';
    }
}

/**
 * Captures the original innerHTML of elements targeted by typehtml actions.
 *
 * Stores the HTML in the map and clears the element so the typehtml effect
 * can progressively reveal it character by character while preserving
 * syntax highlighting spans.
 *
 * @param component - The PKC component instance with refs.
 * @param actions - The sorted list of timeline actions.
 * @param htmlMap - Map to store captured HTML, keyed by ref name.
 */
export function captureTypehtmlContents(
    component: PikoComponent,
    actions: TimelineAction[],
    htmlMap: Map<string, string>,
): void {
    for (const action of actions) {
        if (action.action !== 'typehtml') {continue;}

        const el = component.refs?.[action.ref];
        if (!el) {continue;}

        htmlMap.set(action.ref, el.innerHTML);
        el.innerHTML = '';
    }
}

/**
 * Evaluates all timeline actions at the given time.
 *
 * Every action is a pure function of currentTime, which means seeking
 * backwards works correctly since every action recomputes from scratch.
 *
 * @param component - The PKC component instance with refs.
 * @param actions - The sorted list of timeline actions.
 * @param currentTime - The current timeline position in seconds.
 * @param typewriterTexts - Map of captured typewriter texts, keyed by ref name.
 * @param typehtmlContents - Map of captured innerHTML, keyed by ref name.
 */
export function evaluateTimeline(
    component: PikoComponent,
    actions: TimelineAction[],
    currentTime: number,
    typewriterTexts: Map<string, string>,
    typehtmlContents: Map<string, string>,
): void {
    evaluateVisibility(component, actions, currentTime);
    evaluateClasses(component, actions, currentTime);
    dispatchActions(component, actions, currentTime, typewriterTexts, typehtmlContents);
}

/**
 * Applies show/hide visibility state for all visibility actions.
 *
 * Scans actions for show/hide pairs, determines the initial visibility
 * for each ref (opposite of the first action), then applies the latest
 * state that has been reached by currentTime.
 *
 * @param component - The PKC component instance with refs.
 * @param actions - The sorted list of timeline actions.
 * @param currentTime - The current timeline position in seconds.
 */
function evaluateVisibility(
    component: PikoComponent,
    actions: TimelineAction[],
    currentTime: number,
): void {
    const visibilityState = new Map<string, boolean>();
    const visibilityInitial = new Map<string, boolean>();

    for (const action of actions) {
        if (action.action !== 'show' && action.action !== 'hide') {continue;}
        if (!visibilityInitial.has(action.ref)) {
            visibilityInitial.set(action.ref, action.action === 'hide');
        }
        if (currentTime >= action.time) {
            visibilityState.set(action.ref, action.action === 'show');
        }
    }

    for (const [ref, initialVisible] of visibilityInitial) {
        const el = component.refs?.[ref];
        if (!el) {continue;}
        const visible = visibilityState.get(ref) ?? initialVisible;
        if (visible) {
            el.removeAttribute('p-timeline-hidden');
        } else {
            el.setAttribute('p-timeline-hidden', '');
        }
    }
}

/**
 * Applies addclass/removeclass state for all class actions.
 *
 * Scans actions for addclass/removeclass pairs, determines the initial
 * class presence for each ref+class combination, then applies the
 * latest state that has been reached by currentTime.
 *
 * @param component - The PKC component instance with refs.
 * @param actions - The sorted list of timeline actions.
 * @param currentTime - The current timeline position in seconds.
 */
function evaluateClasses(
    component: PikoComponent,
    actions: TimelineAction[],
    currentTime: number,
): void {
    const classState = new Map<string, boolean>();
    const classInitial = new Map<string, boolean>();

    for (const action of actions) {
        if (action.action !== 'addclass' && action.action !== 'removeclass') {continue;}
        const key = `${action.ref}\0${action.class}`;
        if (!classInitial.has(key)) {
            classInitial.set(key, action.action === 'removeclass');
        }
        if (currentTime >= action.time) {
            classState.set(key, action.action === 'addclass');
        }
    }

    for (const [key, initialPresent] of classInitial) {
        const sep = key.indexOf('\0');
        const ref = key.substring(0, sep);
        const className = key.substring(sep + 1);
        const el = component.refs?.[ref];
        if (!el) {continue;}
        const present = classState.get(key) ?? initialPresent;
        if (present) {
            el.classList.add(className);
        } else {
            el.classList.remove(className);
        }
    }
}

/**
 * Clears all tooltip titles so they can be re-evaluated at the current time.
 *
 * Removes the title attribute from every element targeted by a tooltip
 * action, ensuring stale tooltips are not left behind when seeking.
 *
 * @param component - The PKC component instance with refs.
 * @param actions - The sorted list of timeline actions.
 */
function clearTooltips(component: PikoComponent, actions: TimelineAction[]): void {
    for (const action of actions) {
        if (action.action !== 'tooltip') {continue;}
        const el = component.refs?.[action.ref];
        if (el) {el.removeAttribute('title');}
    }
}

/**
 * Dispatches typing, tooltip, and custom actions at the given time.
 *
 * Clears all tooltip titles first, then iterates the action list and
 * delegates to the appropriate handler based on action type. Built-in
 * actions (type, typehtml, tooltip) are handled directly; unknown
 * action types are dispatched to registered custom handlers.
 *
 * @param component - The PKC component instance with refs.
 * @param actions - The sorted list of timeline actions.
 * @param currentTime - The current timeline position in seconds.
 * @param typewriterTexts - Map of captured typewriter texts, keyed by ref name.
 * @param typehtmlContents - Map of captured innerHTML, keyed by ref name.
 */
function dispatchActions(
    component: PikoComponent,
    actions: TimelineAction[],
    currentTime: number,
    typewriterTexts: Map<string, string>,
    typehtmlContents: Map<string, string>,
): void {
    clearTooltips(component, actions);

    for (const action of actions) {
        const el = component.refs?.[action.ref];

        switch (action.action) {
            case 'type':
                if (el) {evaluateTypewriter(el, action, currentTime, typewriterTexts);}
                break;
            case 'typehtml':
                if (el) {evaluateTypehtmlWriter(el, action, currentTime, typehtmlContents);}
                break;
            case 'tooltip':
                if (el && currentTime >= action.time && action.value) {
                    el.setAttribute('title', action.value);
                }
                break;
            default: {
                const handler = customHandlers.get(action.action);
                if (handler) {handler(el ?? null, action, currentTime, component);}
                break;
            }
        }
    }
}

/**
 * Evaluates the typewriter effect for a single element at the given time.
 *
 * Computes the number of characters to display as a pure function of
 * currentTime, making the effect fully seekable. Characters appear at
 * a rate of one per `speed` milliseconds (default 50ms).
 *
 * @param element - The DOM element to update.
 * @param action - The timeline action describing the typewriter effect.
 * @param currentTime - The current timeline position in seconds.
 * @param typewriterTexts - Map of captured typewriter texts, keyed by ref name.
 */
function evaluateTypewriter(
    element: HTMLElement,
    action: TimelineAction,
    currentTime: number,
    typewriterTexts: Map<string, string>,
): void {
    const fullText = typewriterTexts.get(action.ref) ?? '';
    const speed = action.speed ?? DEFAULT_TYPE_SPEED;
    const elapsed = (currentTime - action.time) * MILLISECONDS_PER_SECOND;

    if (elapsed <= 0) {
        element.textContent = '';
        return;
    }

    const charsToShow = Math.min(
        Math.floor(elapsed / speed),
        fullText.length,
    );
    element.textContent = fullText.substring(0, charsToShow);
}

/**
 * Evaluates the typehtml effect for a single element at the given time.
 *
 * Reveals innerHTML character by character while respecting HTML tag
 * boundaries, so partial tags are never rendered. Only visible characters
 * (not tag markup) are counted against the typing speed. This makes it
 * suitable for syntax-highlighted code where text is wrapped in spans.
 *
 * @param element - The DOM element to update.
 * @param action - The timeline action describing the typehtml effect.
 * @param currentTime - The current timeline position in seconds.
 * @param htmlMap - Map of captured innerHTML, keyed by ref name.
 */
function evaluateTypehtmlWriter(
    element: HTMLElement,
    action: TimelineAction,
    currentTime: number,
    htmlMap: Map<string, string>,
): void {
    const fullHtml = htmlMap.get(action.ref) ?? '';
    const speed = action.speed ?? DEFAULT_TYPEHTML_SPEED;
    const elapsed = (currentTime - action.time) * MILLISECONDS_PER_SECOND;

    if (elapsed <= 0) {
        element.innerHTML = '';
        return;
    }

    const charsToShow = Math.floor(elapsed / speed);
    element.innerHTML = sliceHtml(fullHtml, charsToShow);
}

/**
 * Tracks an HTML tag in the open-tags stack.
 *
 * Opening tags are pushed, closing tags pop the most recent entry,
 * and self-closing tags are ignored.
 */
function trackHtmlTag(tag: string, openTags: string[]): void {
    if (tag.startsWith('</')) {
        openTags.pop();
    } else if (!tag.endsWith('/>')) {
        const match = tag.match(/^<(\w+)/);
        if (match) {
            openTags.push(match[1]);
        }
    }
}

/**
 * Slices an HTML string to show only the first `visibleCount` visible
 * characters, keeping all HTML tags intact.
 *
 * Walks the HTML string counting only non-tag text characters. HTML
 * entities (e.g. &amp;lt;) are counted as a single visible character. When
 * the count reaches the limit, any tags at the cursor boundary are
 * included in full, and all open tags are properly closed. This
 * guarantees the returned string is always valid HTML.
 *
 * @param html - The full HTML string to slice.
 * @param visibleCount - The number of visible characters to include.
 * @returns The sliced HTML with all tags properly closed.
 */
function sliceHtml(html: string, visibleCount: number): string {
    let visible = 0;
    let i = 0;
    const openTags: string[] = [];

    while (i < html.length && visible < visibleCount) {
        if (html[i] === '<') {
            const tagEnd = html.indexOf('>', i);
            if (tagEnd === -1) {break;}
            const tag = html.substring(i, tagEnd + 1);
            trackHtmlTag(tag, openTags);
            i = tagEnd + 1;
        } else if (html[i] === '&') {
            const semiPos = html.indexOf(';', i);
            if (semiPos !== -1 && semiPos - i <= MAX_HTML_ENTITY_LENGTH) {
                i = semiPos + 1;
            } else {
                i++;
            }
            visible++;
        } else {
            visible++;
            i++;
        }
    }

    while (i < html.length && html[i] === '<') {
        const tagEnd = html.indexOf('>', i);
        if (tagEnd === -1) {break;}
        const tag = html.substring(i, tagEnd + 1);
        trackHtmlTag(tag, openTags);
        i = tagEnd + 1;
    }

    let result = html.substring(0, i);
    for (let t = openTags.length - 1; t >= 0; t--) {
        result += `</${openTags[t]}>`;
    }
    return result;
}

/**
 * Finds the first non-STYLE child element to use as the positioning container.
 *
 * Iterates the shadow root's direct children and returns the first
 * HTMLElement whose tag name is not STYLE. Returns null when no
 * suitable container is found.
 *
 * @param shadowRoot - The shadow root to search within.
 * @returns The container element, or null if none is found.
 */
function findAnchorContainer(shadowRoot: ShadowRoot): HTMLElement | null {
    for (const child of Array.from(shadowRoot.children)) {
        if (child instanceof HTMLElement && child.tagName !== 'STYLE') {
            return child;
        }
    }
    return null;
}

/**
 * Computes clamped top/left position for an anchored element relative
 * to a container.
 *
 * Positions the element on the preferred vertical side of the target.
 * If the element would overflow the container in that direction, it
 * flips to the opposite side. Horizontal position is aligned to the
 * left or right edge of the target. Final values are clamped to keep
 * the element within the container bounds.
 *
 * @param targetRect - Bounding rectangle of the target element.
 * @param containerRect - Bounding rectangle of the positioning container.
 * @param elementWidth - Width of the anchored element in pixels.
 * @param elementHeight - Height of the anchored element in pixels.
 * @param wantBottom - Whether to prefer positioning below the target.
 * @param wantRight - Whether to align to the right edge of the target.
 * @returns The computed top and left offsets relative to the container.
 */
function computeAnchorPosition(
    targetRect: DOMRect,
    containerRect: DOMRect,
    elementWidth: number,
    elementHeight: number,
    wantBottom: boolean,
    wantRight: boolean,
): { top: number; left: number } {
    let top: number;
    if (wantBottom) {
        top = targetRect.bottom - containerRect.top + ANCHOR_PADDING;
        if (top + elementHeight > containerRect.height - ANCHOR_PADDING) {
            top = targetRect.top - containerRect.top - elementHeight - ANCHOR_PADDING;
        }
    } else {
        top = targetRect.top - containerRect.top - elementHeight - ANCHOR_PADDING;
        if (top < ANCHOR_PADDING) {
            top = targetRect.bottom - containerRect.top + ANCHOR_PADDING;
        }
    }

    let left: number;
    if (wantRight) {
        left = targetRect.right - containerRect.left - elementWidth;
    } else {
        left = targetRect.left - containerRect.left;
    }

    if (left + elementWidth > containerRect.width - ANCHOR_PADDING) {
        left = containerRect.width - elementWidth - ANCHOR_PADDING;
    }
    if (left < ANCHOR_PADDING) {left = ANCHOR_PADDING;}
    if (top < ANCHOR_PADDING) {top = ANCHOR_PADDING;}
    if (top + elementHeight > containerRect.height - ANCHOR_PADDING) {
        top = containerRect.height - elementHeight - ANCHOR_PADDING;
    }

    return { top, left };
}

/**
 * Positions elements with p-timeline-anchor attributes near their
 * target elements.
 *
 * The attribute value is "refName" or "refName position" where
 * position is one of: bottom-left (default), bottom-right,
 * top-left, top-right.
 *
 * Scans the component's shadow DOM for elements that have a
 * p-timeline-anchor attribute. For each visible anchored element,
 * reads the target ref and preferred position, computes the
 * target's bounding rect relative to a container element, and sets
 * absolute positioning accordingly. If the element would overflow
 * the container in the preferred direction, it flips to the
 * opposite vertical side.
 *
 * Elements that are hidden (have p-timeline-hidden attribute) are
 * skipped. When the target ref is missing or hidden, any previously
 * applied inline position styles are cleared so CSS defaults apply.
 *
 * @param component - The PKC component instance with refs and shadowRoot.
 */
export function evaluateAnchors(component: PikoComponent): void {
    const sr = component.shadowRoot;
    if (!sr) {return;}

    const anchored = sr.querySelectorAll('[p-timeline-anchor]');
    if (anchored.length === 0) {return;}

    const container = findAnchorContainer(sr);
    if (!container) {return;}

    const containerRect = container.getBoundingClientRect();

    for (const el of Array.from(anchored)) {
        const htmlEl = el as HTMLElement;

        if (htmlEl.hasAttribute('p-timeline-hidden')) {continue;}

        const raw = htmlEl.getAttribute('p-timeline-anchor');
        if (!raw) {continue;}

        const parts = raw.split(' ');
        const targetRef = parts[0];
        const position = parts[1] || 'bottom-left';
        const wantBottom = position.startsWith('bottom');
        const wantRight = position.endsWith('right');

        const target = component.refs?.[targetRef];
        if (!target || target.hasAttribute('p-timeline-hidden')) {
            htmlEl.style.top = '';
            htmlEl.style.left = '';
            htmlEl.style.right = '';
            htmlEl.style.bottom = '';
            continue;
        }

        const targetRect = target.getBoundingClientRect();
        const elementWidth = htmlEl.offsetWidth || DEFAULT_ANCHOR_WIDTH;
        const elementHeight = htmlEl.offsetHeight || DEFAULT_ANCHOR_HEIGHT;

        const pos = computeAnchorPosition(
            targetRect, containerRect, elementWidth, elementHeight, wantBottom, wantRight,
        );

        htmlEl.style.top = `${pos.top}px`;
        htmlEl.style.left = `${pos.left}px`;
        htmlEl.style.right = 'auto';
        htmlEl.style.bottom = 'auto';
    }
}
