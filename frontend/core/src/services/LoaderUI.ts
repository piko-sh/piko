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

/** Provides a progress bar indicator for navigation and loading states. */
export interface LoaderUI {
    /** Shows the loader bar and resets progress to zero. */
    show(): void;

    /** Hides the loader bar with a fade animation after reaching full progress. */
    hide(): void;

    /**
     * Updates the progress bar percentage, clamped between 0 and 100.
     * @param percent - The progress percentage to display.
     */
    setProgress(percent: number): void;

    /** Removes the loader element from the DOM. */
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
        container = document.body
    } = options;

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
    container.appendChild(el);

    return {
        show() {
            el.style.display = 'block';
            el.style.width = '0%';
        },

        hide() {
            el.style.width = '100%';
            setTimeout(() => {
                el.style.width = '0%';
                el.style.display = 'none';
            }, fadeMs);
        },

        setProgress(percent: number) {
            let pct = percent;
            if (pct < PROGRESS_MIN) {
                pct = PROGRESS_MIN;
            }
            if (pct > PROGRESS_MAX) {
                pct = PROGRESS_MAX;
            }
            el.style.display = 'block';
            el.style.width = `${pct}%`;
        },

        destroy() {
            el.remove();
        }
    };
}
