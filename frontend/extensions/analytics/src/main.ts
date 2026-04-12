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

/// <reference path="../../../shared/types/index.d.ts" />

import { waitForPiko } from '../../../shared/utils';

/**
 * Sends a command to Google Analytics via the global gtag function.
 *
 * @param command - The gtag command name (e.g. "config", "event", "js").
 * @param args - The remaining arguments for the command.
 */
declare function gtag(command: string, ...args: unknown[]): void;

/** Single entry in the Google Analytics data layer (object or arguments array). */
type DataLayerItem = Record<string, unknown> | IArguments | unknown[];

declare global {
    interface Window {
        /** Google Analytics data layer array. */
        dataLayer: DataLayerItem[];
    }
}

/** Resolved analytics module configuration from the server. */
type AnalyticsConfig = import('@piko/shared-types').AnalyticsModuleConfig;

/** The Piko namespace type exposed on the window object. */
type PikoInstance = typeof window.piko;

/** Tracks which analytics modes are active and whether debug logging is on. */
interface AnalyticsMode {
    hasGTM: boolean;
    hasGA4: boolean;
    debugMode: boolean;
}

/**
 * Reads page-level analytics data from a `<script id="pk-page-data">` element.
 *
 * Pages can include a JSON script block with arbitrary metadata (e.g. property
 * details, product info) that gets merged into dataLayer pushes and GA4 events.
 * The element is re-read on each call so it picks up new data after SPA navigation.
 *
 * @returns The parsed page data object, or an empty object if none is present.
 */
function getPageData(): Record<string, unknown> {
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
 * Loads the Google Tag Manager container script.
 *
 * Initialises the dataLayer with the GTM start event and dynamically
 * injects the gtm.js script into the document head.
 *
 * @param containerId - The GTM container ID (e.g. "GTM-XXXXXXX").
 */
function initGTM(containerId: string): void {
    window.dataLayer.push({
        'gtm.start': new Date().getTime(),
        event: 'gtm.js',
    });

    const script = document.createElement('script');
    script.async = true;
    script.src = `https://www.googletagmanager.com/gtm.js?id=${encodeURIComponent(containerId)}`;
    document.head.appendChild(script);
}

/**
 * Loads the Google Analytics gtag.js script and configures each tracking ID.
 *
 * @param config - The resolved analytics module configuration.
 */
function initGA4(config: AnalyticsConfig): void {
    const trackingIds = config.trackingIds ?? [];

    function gtagInit(...args: unknown[]): void {
        window.dataLayer.push(args);
    }
    (window as unknown as { gtag: typeof gtagInit }).gtag = gtagInit;

    const primaryId = trackingIds[0];
    const script = document.createElement('script');
    script.async = true;
    script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(primaryId)}`;
    document.head.appendChild(script);

    gtagInit('js', new Date());

    for (const trackingId of trackingIds) {
        const configOptions: Record<string, unknown> = {};

        if (config.anonymiseIp) {
            configOptions.anonymize_ip = true;
        }

        if (config.debugMode) {
            configOptions.debug_mode = true;
        }

        gtagInit('config', trackingId, configOptions);

        if (config.debugMode) {
            console.warn(`[piko/analytics] Configured tracking ID: ${trackingId}`);
        }
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
function dispatchEvent(
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

/**
 * Registers all analytics hook listeners for page views, navigation, actions,
 * modals, and errors on the given Piko instance.
 *
 * When GA4 is active, events are sent via the global gtag function.
 * When GTM is active, structured events are pushed to the dataLayer
 * for GTM triggers to consume.
 *
 * @param pk - The Piko namespace instance.
 * @param config - The resolved analytics module configuration.
 * @param mode - Which analytics modes are active.
 */
function registerAnalyticsHooks(pk: PikoInstance, config: AnalyticsConfig, mode: AnalyticsMode): void {
    if (!config.disablePageView) {
        pk.hooks.on('page:view', (payload) => {
            const p = payload as import('@piko/shared-types').PageViewPayload;
            const pagePath = new URL(p.url, window.location.origin).pathname;
            const pageData = getPageData();

            dispatchEvent(
                mode,
                'page_view',
                { page_path: pagePath, page_title: p.title, page_referrer: p.referrer, ...pageData },
                'piko_page_view',
                { page_path: pagePath, page_title: p.title, page_referrer: p.referrer, ...pageData },
            );
        });
    }

    pk.hooks.on('navigation:complete', (payload) => {
        const p = payload as import('@piko/shared-types').NavigationCompletePayload;
        const pagePath = new URL(p.url, window.location.origin).pathname;
        const duration = Math.round(p.duration);
        const pageData = getPageData();

        dispatchEvent(
            mode,
            'navigation',
            { event_category: 'SPA', event_label: pagePath, value: duration, navigation_trigger: p.trigger, ...pageData },
            'piko_navigation',
            { page_path: pagePath, navigation_trigger: p.trigger, navigation_duration: duration, ...pageData },
        );
    });

    pk.hooks.on('navigation:error', (payload) => {
        const p = payload as import('@piko/shared-types').NavigationErrorPayload;
        const pagePath = new URL(p.url, window.location.origin).pathname;
        const description = `Navigation error: ${p.error}`;

        dispatchEvent(
            mode,
            'exception',
            { description, fatal: false, page_path: pagePath },
            'piko_error',
            { error_description: description, error_page: pagePath },
        );
    });

    pk.hooks.on('action:complete', (payload) => {
        const p = payload as import('@piko/shared-types').ActionCompletePayload;
        const duration = Math.round(p.duration);

        dispatchEvent(
            mode,
            'server_action',
            { event_category: 'Actions', event_label: p.actionName, value: duration, action_success: p.success },
            'piko_action',
            { action_name: p.actionName, action_success: p.success, action_duration: duration },
        );
    });

    pk.hooks.on('modal:open', (payload) => {
        const p = payload as import('@piko/shared-types').ModalPayload;

        dispatchEvent(
            mode,
            'modal_view',
            { event_category: 'Modals', event_label: p.modalId },
            'piko_modal_view',
            { modal_id: p.modalId },
        );
    });

    pk.hooks.on('error', (payload) => {
        const p = payload as import('@piko/shared-types').ErrorPayload;
        const description = `${p.type}: ${p.message}`;

        dispatchEvent(
            mode,
            'exception',
            { description, fatal: false },
            'piko_error',
            { error_description: description },
        );
    });

    pk.hooks.on('analytics:track', (payload) => {
        handleAnalyticsTrack(payload, mode);
    });
}

/**
 * Handles custom analytics tracking events dispatched via the
 * `analytics:track` hook.
 *
 * @param payload - The raw hook payload.
 * @param mode - Which analytics modes are active.
 */
function handleAnalyticsTrack(payload: unknown, mode: AnalyticsMode): void {
    const p = payload as import('@piko/shared-types').AnalyticsTrackPayload;
    if (!p.eventName) {
        return;
    }

    dispatchEvent(
        mode,
        p.eventName,
        { ...p.params },
        `piko_${p.eventName}`,
        { ...p.params },
    );

    if (mode.debugMode) {
        console.warn(`[piko/analytics] Custom track: ${p.eventName}`, p.params);
    }
}

/**
 * Initialises analytics tracking for the Piko framework.
 *
 * Supports three modes:
 * - GA4 only: loads gtag.js and sends events via gtag()
 * - GTM only: loads gtm.js and pushes events to dataLayer
 * - GTM + GA4: loads both scripts, pushes events both ways
 */
waitForPiko('analytics')
    .then((pk) => {
        const config = pk.getModuleConfig<AnalyticsConfig>('analytics');
        if (!config) {
            console.warn('[piko/analytics] No tracking IDs or GTM container configured. Analytics will be disabled.');
            console.warn('[piko/analytics] Configure via piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{...})');
            return;
        }

        const hasGA4 = (config.trackingIds?.length ?? 0) > 0;
        const hasGTM = !!config.gtmContainerId;

        if (!hasGA4 && !hasGTM) {
            console.warn('[piko/analytics] No tracking IDs or GTM container configured. Analytics will be disabled.');
            console.warn('[piko/analytics] Configure via piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{...})');
            return;
        }

        window.dataLayer = (window as Omit<Window, 'dataLayer'> & { dataLayer?: DataLayerItem[] }).dataLayer ?? [];

        if (hasGTM && config.gtmContainerId) {
            initGTM(config.gtmContainerId);
        }

        if (hasGA4) {
            initGA4(config);
        }

        const mode: AnalyticsMode = { hasGTM, hasGA4, debugMode: !!config.debugMode };

        registerAnalyticsHooks(pk, config, mode);

        if (!config.disablePageView) {
            const pageData = getPageData();
            dispatchEvent(
                mode,
                'page_view',
                { page_path: window.location.pathname, page_title: document.title, page_referrer: document.referrer, ...pageData },
                'piko_page_view',
                { page_path: window.location.pathname, page_title: document.title, page_referrer: document.referrer, ...pageData },
            );
        }

        if (config.debugMode) {
            const parts: string[] = [];
            if (hasGTM) { parts.push(`GTM: ${config.gtmContainerId}`); }
            if (hasGA4) { parts.push(`GA4: ${config.trackingIds?.join(', ')}`); }
            console.warn(`[piko/analytics] Extension loaded - ${parts.join('; ')}`);
            console.warn('[piko/analytics] Tracking: page views, navigation, actions, modals, errors');
        }
    })
    .catch((err: Error) => console.error(err.message));

export {};
