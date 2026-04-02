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

/** SSE endpoint served by the dev event broadcaster. */
const SSE_URL = '/_piko/dev/events';

/** Maximum reconnection attempts before giving up. */
const MAX_RECONNECTS = 30;

/** Base delay between reconnection attempts, in milliseconds. */
const BASE_DELAY_MS = 500;

/** Parsed payload from a rebuild-complete SSE event. */
interface RebuildEvent {
    /** Route paths affected by the rebuild, or `["*"]` for all routes. */
    affectedRoutes: string[];
}

/** Parsed payload from a template-changed SSE event. */
interface TemplateChangedEvent {
    /** Raw source paths that changed (e.g., "emails/welcome.pk"). */
    affectedPaths: string[];
}

/** Preview URL prefix for dev-mode template previews. */
const PREVIEW_PREFIX = '/_piko/dev/preview/';

waitForPiko('dev')
    .then((pk) => {
        let eventSource: EventSource | null = null;
        let reconnectCount = 0;
        let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
        let hasDirtyForm = false;

        pk.hooks.on('form:dirty', () => {
            hasDirtyForm = true;
        });
        pk.hooks.on('form:clean', () => {
            hasDirtyForm = false;
        });

        function connect(): void {
            eventSource = new EventSource(SSE_URL);

            eventSource.onopen = () => {
                reconnectCount = 0;
                console.debug('[piko/dev] Connected to dev event stream');
            };

            eventSource.addEventListener('rebuild-complete', (e: MessageEvent) => {
                const data = JSON.parse(e.data) as RebuildEvent;
                handleRebuild(pk, data.affectedRoutes);
            });

            eventSource.addEventListener('template-changed', (e: MessageEvent) => {
                const data = JSON.parse(e.data) as TemplateChangedEvent;
                handleTemplateChanged(data.affectedPaths);
            });

            eventSource.onerror = () => {
                eventSource?.close();
                eventSource = null;
                if (reconnectCount < MAX_RECONNECTS) {
                    reconnectCount++;
                    const delay = BASE_DELAY_MS * Math.min(reconnectCount, 5);
                    reconnectTimer = setTimeout(connect, delay);
                }
            };
        }

        function handleRebuild(
            piko: typeof window.piko,
            affectedRoutes: string[],
        ): void {
            const currentPath = window.location.pathname;

            if (affectedRoutes.length > 0 && !affectedRoutes.includes('*')) {
                const affected = affectedRoutes.some(
                    (route) =>
                        currentPath === route ||
                        currentPath.startsWith(route + '/'),
                );
                if (!affected) {
                    console.debug(
                        '[piko/dev] Rebuild did not affect current page',
                    );
                    return;
                }
            }

            if (hasDirtyForm) {
                console.debug('[piko/dev] Skipping refresh: dirty form');
                return;
            }

            console.debug('[piko/dev] Soft-refreshing page');
            piko.nav.navigate(window.location.href, { replace: true, scroll: false });
        }

        /**
         * Handles template-changed events for preview hot-reload.
         * If the browser is currently viewing a preview URL and the
         * changed template matches, the page is reloaded.
         */
        function handleTemplateChanged(affectedPaths: string[]): void {
            const currentPath = window.location.pathname;
            if (!currentPath.startsWith(PREVIEW_PREFIX)) {
                return;
            }

            const rest = currentPath.slice(PREVIEW_PREFIX.length);
            const slashIndex = rest.indexOf('/');
            if (slashIndex == -1) {
                return;
            }
            const componentType = rest.slice(0, slashIndex);
            let templatePath = rest.slice(slashIndex + 1);

            if (templatePath.endsWith('/render')) {
                templatePath = templatePath.slice(0, -'/render'.length);
            }

            const typeToDir: Record<string, string> = {
                email: 'emails/',
                pdf: 'pdfs/',
                page: 'pages/',
                partial: 'partials/',
            };
            const dir = typeToDir[componentType];
            if (!dir) {
                return;
            }
            const sourcePath = dir + templatePath + '.pk';

            if (affectedPaths.includes(sourcePath) || affectedPaths.includes('*')) {
                console.debug('[piko/dev] Preview template changed, reloading');

                const embed = document.querySelector('embed[type="application/pdf"]') as HTMLEmbedElement | null;
                if (embed) {
                    const url = new URL(embed.src);
                    url.searchParams.set('_t', String(Date.now()));
                    embed.src = url.toString();
                    return;
                }

                window.location.reload();
            }
        }

        connect();

        window.addEventListener('beforeunload', () => {
            if (reconnectTimer !== null) {
                clearTimeout(reconnectTimer);
            }
            eventSource?.close();
        });

        console.debug('[piko/dev] Extension loaded - auto-refresh enabled');
    })
    .catch((err: Error) => console.error('[piko/dev]', err.message));

export {};
