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

import {waitForPiko} from '../../../shared/utils';
import {createRouter, type Router} from '@/core/Router';
import {createFetchClient} from '@/core/FetchClient';
import {createLoaderUI, type LoaderUI} from '@/services/LoaderUI';
import {createRemoteRenderer, type RemoteRenderer} from '@/core/RemoteRenderer';
import {createA11yAnnouncer} from '@/services/A11yAnnouncer';
import {_setNavigateAdapter} from '@/pk/navigation';
import type {NavigationCoreServices} from '@/coreServices';

/** Public API returned by the navigation capability factory. */
export interface NavigationCapabilityAPI {
    /** Performs SPA navigation to a URL. */
    navigateTo: Router['navigateTo'];
    /** Fetches and patches remote content into the DOM. */
    remoteRender: RemoteRenderer['render'];
    /** Patches an HTML string directly into the DOM. */
    patchPartial: RemoteRenderer['patchPartial'];
    /** Whether a navigation is currently in progress. */
    isNavigating: Router['isNavigating'];
    /** Shows or hides the navigation loader. */
    toggleLoader(visible: boolean): void;
    /** Updates the loader progress bar. */
    updateProgressBar(percent: number): void;
    /** Replaces the loader with a new colour. */
    createLoaderIndicator(colour: string): void;
    /** Destroys the capability (cleanup). */
    destroy: Router['destroy'];
}

/** Factory signature expected by PPFramework when the capability registers. */
export type NavigationCapabilityFactory = (services: NavigationCoreServices) => NavigationCapabilityAPI;

/**
 * Creates the navigation capability from core services.
 *
 * @param services - Core services provided by the shim.
 * @returns The navigation capability API.
 */
function createNavigationCapability(services: NavigationCoreServices): NavigationCapabilityAPI {
    const fetchClient = createFetchClient();
    let loader: LoaderUI = createLoaderUI({colour: '#29e'});
    const a11yAnnouncer = createA11yAnnouncer();

    const remoteRenderer = createRemoteRenderer({
        moduleLoader: services.moduleLoader,
        spriteSheetManager: services.spriteSheetManager,
        linkHeaderParser: services.linkHeaderParser,
        onDOMUpdated: services.onDOMUpdated,
        hookManager: services.hookManager,
        getPageContext: services.getPageContext,
        executeBeforeRender: services.executeBeforeRender,
        executeAfterRender: services.executeAfterRender,
        executeUpdated: services.executeUpdated,
        executeConnectedForPartials: services.executeConnectedForPartials,
    });

    const router = createRouter({
        fetchClient,
        loader,
        errorDisplay: services.errorDisplay,
        onPageLoad: services.onPageLoad,
        hookManager: services.hookManager,
        formStateManager: services.formStateManager ?? undefined,
        a11yAnnouncer,
    });

    _setNavigateAdapter(router.navigateTo.bind(router));

    return {
        navigateTo: router.navigateTo.bind(router),
        remoteRender: remoteRenderer.render.bind(remoteRenderer),
        patchPartial: remoteRenderer.patchPartial.bind(remoteRenderer),
        isNavigating: router.isNavigating.bind(router),
        toggleLoader(visible: boolean): void {
            if (visible) { loader.show(); } else { loader.hide(); }
        },
        updateProgressBar(percent: number): void {
            loader.setProgress(percent);
        },
        createLoaderIndicator(colour: string): void {
            loader.destroy();
            loader = createLoaderUI({colour});
        },
        destroy: router.destroy.bind(router),
    };
}

waitForPiko('navigation')
    .then((pk) => {
        pk._registerCapability('navigation', createNavigationCapability);
    })
    .catch((err: unknown) => {
        console.error('[piko/navigation] Failed to initialise:', err);
    });
