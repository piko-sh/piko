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
 * Re-exports pure functions from the analytics module for unit testing.
 *
 * The main entry point (`main.ts`) triggers `waitForPiko()` on import,
 * which requires a running Piko instance. This file re-implements the
 * pure, side-effect-free functions so tests can exercise them in
 * isolation without bootstrapping the full framework.
 */

/// <reference path="../../../shared/types/index.d.ts" />

type DataLayerItem = Record<string, unknown> | IArguments | unknown[];

declare function gtag(command: string, ...args: unknown[]): void;

declare global {
    interface Window {
        dataLayer: DataLayerItem[];
    }
}

/** Tracks which analytics modes are active and whether debug logging is on. */
export interface AnalyticsMode {
    hasGTM: boolean;
    hasGA4: boolean;
    debugMode: boolean;
}

/**
 * Reads page-level analytics data from a `<script id="pk-page-data">` element.
 *
 * @returns The parsed page data object, or an empty object if none is present.
 */
export function getPageData(): Record<string, unknown> {
    const el = document.getElementById('pk-page-data');
    if (!el?.textContent) {
        return {};
    }
    try {
        return JSON.parse(el.textContent) as Record<string, unknown>;
    } catch {
        console.warn('[piko/analytics] Failed to parse pk-page-data JSON');
        return {};
    }
}

/**
 * Dispatches an analytics event to GA4 and/or GTM depending on which modes
 * are active.
 *
 * @param mode - Which analytics modes are active.
 * @param ga4EventName - The GA4 event name passed to gtag().
 * @param ga4Params - Parameters for the GA4 event.
 * @param gtmEventName - The dataLayer event name for GTM.
 * @param gtmParams - Parameters pushed to the dataLayer for GTM.
 */
export function dispatchEvent(
    mode: AnalyticsMode,
    ga4EventName: string,
    ga4Params: Record<string, unknown>,
    gtmEventName: string,
    gtmParams: Record<string, unknown>,
): void {
    if (mode.hasGA4) {
        gtag('event', ga4EventName, ga4Params);
    }

    if (mode.hasGTM) {
        if (mode.debugMode) {
            console.warn(`[piko/analytics] GTM push: ${gtmEventName}`, gtmParams);
        }
        window.dataLayer.push({ event: gtmEventName, ...gtmParams });
    }
}
