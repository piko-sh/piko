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

/**
 * Registers all analytics hook listeners for page views, navigation, actions,
 * modals, and errors on the given Piko instance.
 * @param pk - The Piko namespace instance.
 * @param config - The resolved analytics module configuration.
 */
function registerAnalyticsHooks(pk: PikoInstance, config: AnalyticsConfig): void {
    if (!config.disablePageView) {
        pk.hooks.on('page:view', (payload) => {
            const p = payload as import('@piko/shared-types').PageViewPayload;
            gtag('event', 'page_view', {
                page_path: new URL(p.url).pathname,
                page_title: p.title,
                page_referrer: p.referrer,
            });
        });
    }

    pk.hooks.on('navigation:complete', (payload) => {
        const p = payload as import('@piko/shared-types').NavigationCompletePayload;
        gtag('event', 'navigation', {
            event_category: 'SPA',
            event_label: new URL(p.url).pathname,
            value: Math.round(p.duration),
            navigation_trigger: p.trigger,
        });
    });

    pk.hooks.on('navigation:error', (payload) => {
        const p = payload as import('@piko/shared-types').NavigationErrorPayload;
        gtag('event', 'exception', {
            description: `Navigation error: ${p.error}`,
            fatal: false,
            page_path: new URL(p.url).pathname,
        });
    });

    pk.hooks.on('action:complete', (payload) => {
        const p = payload as import('@piko/shared-types').ActionCompletePayload;
        gtag('event', 'server_action', {
            event_category: 'Actions',
            event_label: p.actionName,
            value: Math.round(p.duration),
            action_success: p.success,
        });
    });

    pk.hooks.on('modal:open', (payload) => {
        const p = payload as import('@piko/shared-types').ModalPayload;
        gtag('event', 'modal_view', {
            event_category: 'Modals',
            event_label: p.modalId,
        });
    });

    pk.hooks.on('error', (payload) => {
        const p = payload as import('@piko/shared-types').ErrorPayload;
        gtag('event', 'exception', {
            description: `${p.type}: ${p.message}`,
            fatal: false,
        });
    });
}

/**
 * Initialises Google Analytics tracking for the Piko framework.
 *
 * Sets up the global `dataLayer` array and `gtag` function, loads the
 * gtag.js script for the primary tracking ID, configures each tracking ID
 * with the server-provided options, and registers hook listeners for
 * page views, navigation, actions, modals, and errors.
 */
waitForPiko('analytics')
    .then((pk) => {
        const config = pk.getModuleConfig<AnalyticsConfig>('analytics');

        if (!config?.trackingIds.length) {
            console.warn('[piko/analytics] No tracking IDs configured. Analytics will be disabled.');
            console.warn('[piko/analytics] Configure via piko.WithFrontendModule(piko.ModuleAnalytics, piko.AnalyticsConfig{...})');
            return;
        }

        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- dataLayer may not exist at runtime despite the type declaration
        window.dataLayer = window.dataLayer ?? [];
        function gtagInit(...args: unknown[]): void {
            window.dataLayer.push(args);
        }
        (window as unknown as { gtag: typeof gtagInit }).gtag = gtagInit;

        const primaryId = config.trackingIds[0];
        const script = document.createElement('script');
        script.async = true;
        script.src = `https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(primaryId)}`;
        document.head.appendChild(script);

        gtagInit('js', new Date());

        for (const trackingId of config.trackingIds) {
            const configOptions: Record<string, unknown> = {};

            if (config.anonymizeIp) {
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

        registerAnalyticsHooks(pk, config);

        const trackingCount = config.trackingIds.length;
        const idList = config.trackingIds.join(', ');
        console.warn(`[piko/analytics] Extension loaded - ${trackingCount} tracking ID(s): ${idList}`);
        console.warn('[piko/analytics] Tracking: page views, navigation, actions, modals, errors');
    })
    .catch((err: Error) => console.error(err.message));

export {};
