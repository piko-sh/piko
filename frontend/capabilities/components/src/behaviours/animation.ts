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

import {registerBehaviour} from '@/behaviours/registry';

/**
 * Thin delegation behaviour for timeline animation.
 *
 * All heavy logic lives in the animation extension
 * (frontend/extensions/animation). This behaviour simply reads the setup
 * function from the well-known global that the extension registers on load
 * and calls it with the component.
 */
const animationBehaviour = (component: HTMLElement): void => {
    const setup = (window as unknown as Record<string, unknown>).__piko_animation as
        ((c: HTMLElement) => void) | undefined;

    if (!setup) {
        console.warn(
            `Animation behaviour enabled on ${component.tagName}, ` +
            `but the animation extension is not loaded. ` +
            `Ensure the compiler is adding the extension import.`
        );
        return;
    }

    setup(component);
};

registerBehaviour('animation', animationBehaviour);
