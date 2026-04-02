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

/** Manages SVG sprite sheet merging. */
export interface SpriteSheetManager {
    /**
     * Merges new symbols from a sprite sheet into the main sheet.
     * Replaces existing symbols with the same ID and appends new ones.
     * If no main sheet exists, promotes the new sheet to become the main sheet.
     * @param newSheet - The SVG sprite sheet element to merge, or null to skip.
     */
    merge(newSheet: SVGSVGElement | null): void;

    /** Ensures the main sprite sheet element exists in the DOM, creating one if absent. */
    ensureExists(): void;
}

/**
 * Creates a SpriteSheetManager for merging SVG sprite sheets.
 * @returns A new SpriteSheetManager instance.
 */
export function createSpriteSheetManager(): SpriteSheetManager {
    return {
        merge(newSheet: SVGSVGElement | null) {
            if (!newSheet) {
                return;
            }

            const mainSheet = document.getElementById('sprite') as SVGSVGElement | null;
            if (!mainSheet) {
                console.warn("SpriteSheetManager: Main sprite sheet with id='sprite' not found. Cannot merge new sprites.");
                newSheet.id = 'sprite';
                newSheet.style.display = 'none';
                document.body.appendChild(newSheet);
                return;
            }

            const newSymbols = newSheet.querySelectorAll('symbol');

            newSymbols.forEach(newSymbol => {
                const symbolId = newSymbol.id;
                if (!symbolId) {
                    console.warn('SpriteSheetManager: Found a symbol without an ID, skipping.', newSymbol);
                    return;
                }

                const existingSymbol = mainSheet.querySelector(`symbol[id="${symbolId}"]`);

                if (existingSymbol) {
                    existingSymbol.replaceWith(newSymbol.cloneNode(true));
                } else {
                    mainSheet.appendChild(newSymbol.cloneNode(true));
                }
            });
        },

        ensureExists() {
            if (!document.getElementById('sprite')) {
                const sheet = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
                sheet.id = 'sprite';
                sheet.style.display = 'none';
                document.body.appendChild(sheet);
            }
        }
    };
}
