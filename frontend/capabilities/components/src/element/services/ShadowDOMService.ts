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

/** Reset CSS applied to all shadow DOM components. */
export const RESET_CSS = `*, *::before, *::after { margin:0; padding:0; box-sizing:border-box; } :host { display: block; }`;

/**
 * Manages shadow DOM creation and CSS injection for custom elements.
 */
export interface ShadowDOMService {
    /** Attaches shadow root if not already attached, injecting reset CSS and component CSS. Returns existing if present. */
    ensureShadowRoot(): ShadowRoot;

    /** Gets the shadow root, or null if not attached. */
    getShadowRoot(): ShadowRoot | null;

    /** Returns whether the shadow root is attached. */
    hasShadowRoot(): boolean;
}

/**
 * Options for creating a ShadowDOMService.
 */
export interface ShadowDOMServiceOptions {
    /** The host custom element. */
    host: HTMLElement;

    /** Component-specific CSS from static css getter. */
    componentCSS?: string;

    /** Shadow DOM mode: 'open' or 'closed'. */
    mode?: ShadowRootMode;

    /** Whether to delegate focus to the first focusable element in the shadow DOM. */
    delegatesFocus?: boolean;
}

/**
 * Creates a ShadowDOMService for managing shadow DOM creation and CSS injection.
 *
 * @param options - Configuration options including host and CSS.
 * @returns A new ShadowDOMService instance.
 */
export function createShadowDOMService(options: ShadowDOMServiceOptions): ShadowDOMService {
    const {host, componentCSS, mode = "open", delegatesFocus = false} = options;

    return {
        ensureShadowRoot(): ShadowRoot {
            if (host.shadowRoot) {
                return host.shadowRoot;
            }

            const shadow = host.attachShadow({mode, delegatesFocus, serializable: true});

            const resetStyleEl = document.createElement("style");
            resetStyleEl.textContent = RESET_CSS;
            shadow.appendChild(resetStyleEl);

            if (typeof componentCSS === "string" && componentCSS.trim()) {
                const userStyleElement = document.createElement("style");
                userStyleElement.textContent = componentCSS;
                shadow.appendChild(userStyleElement);
            }

            return shadow;
        },

        getShadowRoot(): ShadowRoot | null {
            return host.shadowRoot;
        },

        hasShadowRoot(): boolean {
            return host.shadowRoot !== null;
        },
    };
}
