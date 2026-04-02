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

import type {TimelineAction} from './timeline';

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
    component: any,
    actions: TimelineAction[],
    textMap: Map<string, string>,
): void {
    for (const action of actions) {
        if (action.action !== 'type') continue;

        const el = component.refs?.[action.ref] as HTMLElement | undefined;
        if (!el) continue;

        textMap.set(action.ref, el.textContent ?? '');
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
    component: any,
    actions: TimelineAction[],
    htmlMap: Map<string, string>,
): void {
    for (const action of actions) {
        if (action.action !== 'typehtml') continue;

        const el = component.refs?.[action.ref] as HTMLElement | undefined;
        if (!el) continue;

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
    component: any,
    actions: TimelineAction[],
    currentTime: number,
    typewriterTexts: Map<string, string>,
    typehtmlContents: Map<string, string>,
): void {
    const visibilityState = new Map<string, boolean>();
    const visibilityInitial = new Map<string, boolean>();

    for (const action of actions) {
        if (action.action !== 'show' && action.action !== 'hide') continue;
        if (!visibilityInitial.has(action.ref)) {
            visibilityInitial.set(action.ref, action.action === 'hide');
        }
        if (currentTime >= action.time) {
            visibilityState.set(action.ref, action.action === 'show');
        }
    }

    for (const [ref, initialVisible] of visibilityInitial) {
        const el = component.refs?.[ref] as HTMLElement | undefined;
        if (!el) continue;
        const visible = visibilityState.has(ref)
            ? visibilityState.get(ref)!
            : initialVisible;
        if (visible) {
            el.removeAttribute('p-timeline-hidden');
        } else {
            el.setAttribute('p-timeline-hidden', '');
        }
    }

    const classState = new Map<string, boolean>();
    const classInitial = new Map<string, boolean>();

    for (const action of actions) {
        if (action.action !== 'addclass' && action.action !== 'removeclass') continue;
        const key = action.ref + '\0' + action.class;
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
        const el = component.refs?.[ref] as HTMLElement | undefined;
        if (!el) continue;
        const present = classState.has(key)
            ? classState.get(key)!
            : initialPresent;
        if (present) {
            el.classList.add(className);
        } else {
            el.classList.remove(className);
        }
    }

    for (const action of actions) {
        if (action.action !== 'tooltip') continue;
        const el = component.refs?.[action.ref] as HTMLElement | undefined;
        if (el) el.removeAttribute('title');
    }

    for (const action of actions) {
        const el = component.refs?.[action.ref] as HTMLElement | undefined;
        if (!el) continue;

        switch (action.action) {
            case 'type':
                evaluateTypewriter(el, action, currentTime, typewriterTexts);
                break;
            case 'typehtml':
                evaluateTypehtmlWriter(el, action, currentTime, typehtmlContents);
                break;
            case 'tooltip':
                if (currentTime >= action.time && action.value) {
                    el.setAttribute('title', action.value);
                }
                break;
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
    const speed = action.speed ?? 50;
    const elapsed = (currentTime - action.time) * 1000;

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
    const speed = action.speed ?? 25;
    const elapsed = (currentTime - action.time) * 1000;

    if (elapsed <= 0) {
        element.innerHTML = '';
        return;
    }

    const charsToShow = Math.floor(elapsed / speed);
    element.innerHTML = sliceHtml(fullHtml, charsToShow);
}

/**
 * Slices an HTML string to show only the first `visibleCount` visible
 * characters, keeping all HTML tags intact.
 *
 * Walks the HTML string counting only non-tag text characters. HTML
 * entities (e.g. &lt;) are counted as a single visible character. When
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
            if (tagEnd === -1) break;
            const tag = html.substring(i, tagEnd + 1);

            if (tag.startsWith('</')) {
                openTags.pop();
            } else if (!tag.endsWith('/>')) {
                const match = tag.match(/^<(\w+)/);
                if (match) openTags.push(match[1]);
            }
            i = tagEnd + 1;
        } else if (html[i] === '&') {
            const semiPos = html.indexOf(';', i);
            if (semiPos !== -1 && semiPos - i <= 10) {
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
        if (tagEnd === -1) break;
        const tag = html.substring(i, tagEnd + 1);
        if (tag.startsWith('</')) {
            openTags.pop();
        } else if (!tag.endsWith('/>')) {
            const match = tag.match(/^<(\w+)/);
            if (match) openTags.push(match[1]);
        }
        i = tagEnd + 1;
    }

    let result = html.substring(0, i);
    for (let t = openTags.length - 1; t >= 0; t--) {
        result += '</' + openTags[t] + '>';
    }
    return result;
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
export function evaluateAnchors(component: any): void {
    const sr = component.shadowRoot as ShadowRoot | undefined;
    if (!sr) return;

    const anchored = sr.querySelectorAll('[p-timeline-anchor]');
    if (anchored.length === 0) return;

    let container: HTMLElement | null = null;
    for (const child of sr.children) {
        if (child instanceof HTMLElement && child.tagName !== 'STYLE') {
            container = child;
            break;
        }
    }
    if (!container) return;

    const cr = container.getBoundingClientRect();
    const pad = 6;

    for (const el of anchored) {
        const htmlEl = el as HTMLElement;

        if (htmlEl.hasAttribute('p-timeline-hidden')) continue;

        const raw = htmlEl.getAttribute('p-timeline-anchor');
        if (!raw) continue;

        const parts = raw.split(' ');
        const targetRef = parts[0];
        const position = parts[1] || 'bottom-left';
        const wantBottom = position.startsWith('bottom');
        const wantRight = position.endsWith('right');

        const target = component.refs?.[targetRef] as HTMLElement | undefined;
        if (!target || target.hasAttribute('p-timeline-hidden')) {
            htmlEl.style.top = '';
            htmlEl.style.left = '';
            htmlEl.style.right = '';
            htmlEl.style.bottom = '';
            continue;
        }

        const tr = target.getBoundingClientRect();
        const tw = htmlEl.offsetWidth || 200;
        const th = htmlEl.offsetHeight || 40;

        let top: number;
        if (wantBottom) {
            top = tr.bottom - cr.top + pad;
            if (top + th > cr.height - pad) {
                top = tr.top - cr.top - th - pad;
            }
        } else {
            top = tr.top - cr.top - th - pad;
            if (top < pad) {
                top = tr.bottom - cr.top + pad;
            }
        }

        let left: number;
        if (wantRight) {
            left = tr.right - cr.left - tw;
        } else {
            left = tr.left - cr.left;
        }

        if (left + tw > cr.width - pad) left = cr.width - tw - pad;
        if (left < pad) left = pad;
        if (top < pad) top = pad;
        if (top + th > cr.height - pad) top = cr.height - th - pad;

        htmlEl.style.top = top + 'px';
        htmlEl.style.left = left + 'px';
        htmlEl.style.right = 'auto';
        htmlEl.style.bottom = 'auto';
    }
}
