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
 * Creates a proxy-based refs object that lazily queries elements by p-ref attribute.
 *
 * When the scope element has a `partial` attribute, refs are searched across ALL
 * elements with that same partial ID. This handles the case where partial content
 * is distributed across multiple slots of a parent component - each slot container
 * becomes a sibling with the same partial ID, and refs need to be accessible
 * across all of them.
 *
 * @param scope - The element to scope queries to (defaults to document.body).
 * @returns A proxy that returns elements by their p-ref name.
 */
export function createRefs(scope: Element = document.body): Record<string, HTMLElement | null> {
    const partialId = scope.getAttribute('partial') ?? scope.closest('[partial]')?.getAttribute('partial');

    return new Proxy({} as Record<string, HTMLElement | null>, {
        get(_, name: string | symbol) {
            if (typeof name !== 'string' || name === 'then') {
                return undefined;
            }

            let el: HTMLElement | null;

            if (partialId) {
                el = document.querySelector(`[partial~="${partialId}"][p-ref="${name}"]`) as HTMLElement | null;
            }

            el ??= scope.querySelector(`[p-ref="${name}"]`) as HTMLElement | null;

            if (!el) {
                console.warn(`[pk] ref "${name}" not found in scope`);
            }
            return el;
        }
    });
}

/**
 * Global refs object scoped to document.body.
 *
 * Uses createRefs() for custom scoping.
 */
export const refs = createRefs();
