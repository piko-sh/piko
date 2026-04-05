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
 * Behaviour setup function type.
 */
export type PPBehaviour<T = HTMLElement> = (component: T) => void;

/**
 * Applies behaviour mixins to custom element components.
 */
export interface BehaviourApplicator {
    /** Applies all enabled behaviours to the component. */
    applyBehaviours(): void;
}

/**
 * Options for creating a BehaviourApplicator.
 */
export interface BehaviourApplicatorOptions<T = HTMLElement> {
    /** The host custom element. */
    host: T;

    /** List of enabled behaviour names to apply. */
    enabledBehaviours: string[];

    /** Function to retrieve a behaviour by name from the registry. */
    getBehaviour: (name: string) => PPBehaviour<T> | undefined;
}

/**
 * Creates a BehaviourApplicator for applying behaviour mixins to components.
 *
 * @param options - Configuration options including the host and behaviours.
 * @returns A new BehaviourApplicator instance.
 */
export function createBehaviourApplicator<T extends HTMLElement>(
    options: BehaviourApplicatorOptions<T>
): BehaviourApplicator {
    const {host, enabledBehaviours, getBehaviour} = options;

    return {
        applyBehaviours(): void {
            for (const behaviourName of enabledBehaviours) {
                const setupFunction = getBehaviour(behaviourName);
                if (setupFunction) {
                    setupFunction(host);
                } else {
                    console.warn(
                        `PPElement: Unknown behaviour "${behaviourName}" enabled for ${host.tagName}.`
                    );
                }
            }
        },
    };
}
