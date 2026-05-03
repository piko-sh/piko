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

/** Default fade animation duration in milliseconds. */
const LOADER_FADE_MS = 300;

/** Minimum progress percentage. */
const PROGRESS_MIN = 0;

/** Maximum progress percentage. */
const PROGRESS_MAX = 100;

/** How often the trickle timer advances the bar, in milliseconds. */
const TRICKLE_TICK_MS = 250;

/** Fraction of the remaining gap to the trickle target closed each tick. */
const TRICKLE_STEP = 0.10;

/** Asymptotic ceiling for the TTFB phase (before headers received). */
const TRICKLE_TTFB_TARGET = 50;

/** Asymptotic ceiling for the download phase (after headers received). */
const TRICKLE_DOWNLOAD_TARGET = 90;

/** How close to the asymptote the trickle is allowed to creep. */
const TRICKLE_EPSILON = 0.5;

/**
 * Internal lifecycle phases of the loader. Used to drive the trickle ceiling
 * and ensure transitions only fire from the right state.
 */
type LoaderPhase = 'idle' | 'trickle1' | 'trickle2' | 'finishing';

/**
 * Mutable state for a single loader instance, separated from the closure to
 * keep the factory function under the lint budget.
 */
interface LoaderState {
    phase: LoaderPhase;
    displayPercent: number;
    trickleTimer: ReturnType<typeof setInterval> | null;
    fadeTimer: ReturnType<typeof setTimeout> | null;
}

const trickleTargetFor = (phase: LoaderPhase): number =>
    phase === 'trickle1' ? TRICKLE_TTFB_TARGET : TRICKLE_DOWNLOAD_TARGET;

function paintBar(el: HTMLElement, percent: number): void {
    el.style.display = 'block';
    el.style.width = `${percent}%`;
}

function clearTrickleTimer(state: LoaderState): void {
    if (state.trickleTimer !== null) {
        clearInterval(state.trickleTimer);
        state.trickleTimer = null;
    }
}

function clearFadeTimer(state: LoaderState): void {
    if (state.fadeTimer !== null) {
        clearTimeout(state.fadeTimer);
        state.fadeTimer = null;
    }
}

function advanceTrickle(state: LoaderState, el: HTMLElement): void {
    const target = trickleTargetFor(state.phase);
    if (state.displayPercent >= target - TRICKLE_EPSILON) {
        return;
    }
    state.displayPercent += (target - state.displayPercent) * TRICKLE_STEP;
    if (state.displayPercent > target - TRICKLE_EPSILON) {
        state.displayPercent = target - TRICKLE_EPSILON;
    }
    paintBar(el, state.displayPercent);
}

function startTrickle(state: LoaderState, el: HTMLElement, enabled: boolean, tickMs: number): void {
    clearTrickleTimer(state);
    if (!enabled) {
        return;
    }
    state.trickleTimer = setInterval(() => advanceTrickle(state, el), tickMs);
}

function buildLoaderEl(colour: string): HTMLElement {
    const el = document.createElement('div');
    el.id = 'ppf-loader-bar';
    el.style.cssText = [
        'position:fixed',
        'top:0',
        'left:0',
        'width:0%',
        'height:2px',
        `background:${colour}`,
        'transition:width .2s',
        'z-index:9999',
        'pointer-events:none',
        'display:none',
    ].join(';');
    return el;
}

function clampPercent(percent: number): number {
    if (percent < PROGRESS_MIN) {
        return PROGRESS_MIN;
    }
    if (percent > PROGRESS_MAX) {
        return PROGRESS_MAX;
    }
    return percent;
}

/** Provides a progress bar indicator for navigation and loading states. */
export interface LoaderUI {
    /** Shows the loader bar and starts the TTFB-phase trickle. */
    show(): void;

    /** Hides the loader bar with a fade animation after snapping to 100%. */
    hide(): void;

    /**
     * Sets the progress bar to a specific percentage. Monotonic within a
     * show/hide cycle.
     * @param percent - The progress percentage to display, clamped to 0–100.
     */
    setProgress(percent: number): void;

    /**
     * Signals that response headers have been received (TTFB completed) and
     * the body is about to stream. Transitions the bar from the TTFB trickle
     * (asymptote 50%) to the download trickle (asymptote 90%) and snaps to
     * at least 50% so the user sees an immediate jump. Idempotent.
     */
    headersReceived(): void;

    /** Removes the loader element from the DOM and clears any timers. */
    destroy(): void;
}

/** Configuration options for the LoaderUI. */
export interface LoaderUIOptions {
    /** Colour of the progress bar. */
    colour?: string;

    /** Fade duration in milliseconds. */
    fadeMs?: number;

    /** Container element to append the loader to. */
    container?: HTMLElement;

    /**
     * Whether to run the NProgress-style trickle animation between
     * show/hide. When false, restores stateless behaviour: show() draws a
     * 0% bar, setProgress() draws an exact width. Default true.
     */
    trickle?: boolean;

    /** Override the trickle interval (milliseconds). Mostly for testing. */
    trickleTickMs?: number;
}

/**
 * Creates a LoaderUI for displaying a progress bar during navigation.
 * @param options - The optional configuration for the loader.
 * @returns A new LoaderUI instance.
 */
export function createLoaderUI(options: LoaderUIOptions = {}): LoaderUI {
    const {
        colour = '#29e',
        fadeMs = LOADER_FADE_MS,
        container = document.body,
        trickle = true,
        trickleTickMs = TRICKLE_TICK_MS,
    } = options;

    const el = buildLoaderEl(colour);
    container.appendChild(el);

    const state: LoaderState = {
        phase: 'idle',
        displayPercent: 0,
        trickleTimer: null,
        fadeTimer: null,
    };

    return {
        show() {
            clearFadeTimer(state);
            state.phase = 'trickle1';
            state.displayPercent = 0;
            paintBar(el, state.displayPercent);
            startTrickle(state, el, trickle, trickleTickMs);
        },

        hide() {
            clearTrickleTimer(state);
            state.phase = 'finishing';
            state.displayPercent = PROGRESS_MAX;
            paintBar(el, state.displayPercent);
            clearFadeTimer(state);
            state.fadeTimer = setTimeout(() => {
                el.style.width = '0%';
                el.style.display = 'none';
                state.displayPercent = 0;
                state.phase = 'idle';
                state.fadeTimer = null;
            }, fadeMs);
        },

        setProgress(percent: number) {
            const pct = clampPercent(percent);
            clearTrickleTimer(state);
            if (pct > state.displayPercent) {
                state.displayPercent = pct;
            }
            paintBar(el, state.displayPercent);
        },

        headersReceived() {
            if (state.phase !== 'trickle1') {
                return;
            }
            state.phase = 'trickle2';
            if (state.displayPercent < TRICKLE_TTFB_TARGET) {
                state.displayPercent = TRICKLE_TTFB_TARGET;
            }
            paintBar(el, state.displayPercent);
            startTrickle(state, el, trickle, trickleTickMs);
        },

        destroy() {
            clearTrickleTimer(state);
            clearFadeTimer(state);
            el.remove();
        }
    };
}
