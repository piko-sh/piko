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
import {createSyncPartialManager, type SyncPartialManager} from '@/services/SyncPartialManager';

/** Public API registered with the core shim. */
export interface PartialsCapabilityAPI {
    /** Creates and returns a SyncPartialManager bound to a callback. */
    createSyncPartialManager(callbacks: { onRemoteRender: (options: unknown) => Promise<void> }): SyncPartialManager;
}

/** Factory signature expected by PPFramework. */
export type PartialsCapabilityFactory = () => PartialsCapabilityAPI;

/**
 * Creates the partials capability.
 *
 * @returns The partials capability API.
 */
function createPartialsCapability(): PartialsCapabilityAPI {
    return {
        createSyncPartialManager(callbacks) {
            return createSyncPartialManager(callbacks);
        },
    };
}

waitForPiko('partials')
    .then((pk) => {
        pk._registerCapability('partials', createPartialsCapability);
    })
    .catch((err: unknown) => {
        console.error('[piko/partials] Failed to initialise:', err);
    });
