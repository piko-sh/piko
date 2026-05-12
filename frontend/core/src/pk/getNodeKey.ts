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
 * Returns a stable key for a DOM node using data-stable-id, p-key, or element id.
 * Keys are namespaced by `partial_name` when present, so an element keyed by p-key
 * does not collide with a different-partial element occupying the same tree
 * position across SPA navigations or partial reloads.
 *
 * Shared between RemoteRenderer's full-page patch path and the partial-reload
 * morph path so keyed-child reordering and focus restoration behave the same
 * whichever code triggers the morph.
 *
 * @param node - The node to extract a key from.
 * @returns The key string, or null if no key is available.
 */
export function getNodeKey(node: Node): string | null {
    if (node.nodeType !== Node.ELEMENT_NODE) {
        return null;
    }
    const el = node as HTMLElement;
    const baseKey = el.dataset.stableId ?? el.getAttribute('p-key') ?? (el.id || null);
    if (baseKey === null) {
        return null;
    }
    const partialName = el.getAttribute('partial_name');
    return partialName ? `${partialName}@${baseKey}` : baseKey;
}
