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

import type {PPElement} from '@/element';

/** Function that applies a behaviour to a PPElement component. */
export type PPBehaviour = (component: PPElement) => void;

/** Registry mapping behaviour names to their setup functions. */
const behaviourRegistry: Partial<Record<string, PPBehaviour>> = {};

/**
 * Registers a behaviour for use with PPElement components.
 *
 * Behaviours are applied to components via the static enabledBehaviours property.
 *
 * @param name - The name to register the behaviour under.
 * @param setupFunction - The setup function to run when applying the behaviour.
 */
export function registerBehaviour(name: string, setupFunction: PPBehaviour): void {
    if (behaviourRegistry[name]) {
        console.warn(`PPBehaviour: Overwriting already registered behaviour "${name}".`);
    }
    behaviourRegistry[name] = setupFunction;
}

/**
 * Gets a registered behaviour by name.
 *
 * @param name - The name of the behaviour to retrieve.
 * @returns The behaviour function, or undefined if not found.
 */
export function getBehaviour(name: string): PPBehaviour | undefined {
    return behaviourRegistry[name];
}
