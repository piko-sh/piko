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
 * Finds a stable key in data-stable-id, p-key or id, in that order. A nonempty
 * partial_name prefixes the key.
 *
 * @param node - The node to check for a key.
 * @returns The stable key or null for a node without one.
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

/**
 * Replaces the source p-key prefix with the target p-key. The function updates the source
 * tree in place when both roots have nonempty keys. Descendant keys with a different
 * prefix remain unchanged.
 *
 * @param sourceEl - The root to rekey.
 * @param targetEl - The element that supplies the replacement key.
 */
export function transformRelativeKeys(sourceEl: Element, targetEl: Element): void {
    const targetKey = targetEl.getAttribute('p-key');
    const sourceKey = sourceEl.getAttribute('p-key');

    if (!targetKey || !sourceKey) {
        return;
    }

    const elementsWithKeys = [sourceEl, ...Array.from(sourceEl.querySelectorAll('[p-key]'))];

    for (const el of elementsWithKeys) {
        const currentKey = el.getAttribute('p-key');
        if (!currentKey) {
            continue;
        }

        if (currentKey === sourceKey) {
            el.setAttribute('p-key', targetKey);
        } else if (currentKey.startsWith(`${sourceKey}:`)) {
            const suffix = currentKey.slice(sourceKey.length);
            el.setAttribute('p-key', targetKey + suffix);
        }
    }
}
