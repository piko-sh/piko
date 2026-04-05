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
import {handleAction, setActionExecutorDependencies} from '@/core/ActionExecutor';
import type {ActionsCoreServices} from '@/coreServices';
import type {ActionDescriptor} from '@/pk/action';

/** Public API registered with the core shim. */
export interface ActionsCapabilityAPI {
    /** Executes an action descriptor through the full action lifecycle. */
    handleAction(descriptor: ActionDescriptor, element: HTMLElement, event?: Event): Promise<void>;
}

/** Factory signature expected by PPFramework when the capability registers. */
export type ActionsCapabilityFactory = (services: ActionsCoreServices) => ActionsCapabilityAPI;

/**
 * Creates the actions capability from core services.
 *
 * @param services - Core services provided by the shim.
 * @returns The actions capability API.
 */
function createActionsCapability(services: ActionsCoreServices): ActionsCapabilityAPI {
    setActionExecutorDependencies({
        hookManager: services.hookManager,
        formStateManager: services.formStateManager ?? undefined,
        helperRegistry: services.helperRegistry,
    });

    return {
        handleAction,
    };
}

waitForPiko('actions')
    .then((pk) => {
        pk._registerCapability('actions', createActionsCapability);
    })
    .catch((err: unknown) => {
        console.error('[piko/actions] Failed to initialise:', err);
    });
