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

const SVG_NAMESPACE = 'http://www.w3.org/2000/svg';

/**
 * Hides the sprite host without using display:none, which makes some browsers drop paint
 * servers such as gradients. A zero-size absolutely positioned box keeps the definitions
 * live while taking no layout space.
 * @param sheet - The sprite sheet element to hide.
 */
function applyHiddenSpriteStyle(sheet: SVGSVGElement): void {
    sheet.style.position = 'absolute';
    sheet.style.width = '0';
    sheet.style.height = '0';
    sheet.style.overflow = 'hidden';
}

/**
 * Finds a direct child of parent whose id matches. Matching is done on the live id property
 * rather than a CSS selector, so ids containing characters that are invalid in a selector
 * (such as a double quote) cannot throw a SyntaxError.
 * @param parent - The element whose direct children are scanned.
 * @param id - The id to match.
 * @returns The matching child, or undefined when none matches.
 */
function findChildById(parent: Element, id: string): Element | undefined {
    return Array.from(parent.children).find(child => child.id === id);
}

/**
 * Merges the element children of one container into another. Children with an id replace any
 * existing child sharing that id, otherwise they are appended. Children without an id are
 * appended only when no structurally equal child already exists, so repeated merges across
 * navigations cannot accumulate duplicate id-less nodes (such as a hoisted <style>).
 * @param target - The container receiving the merged children.
 * @param source - The container whose children are merged in.
 */
function mergeById(target: Element, source: Element): void {
    Array.from(source.children).forEach(child => {
        const id = child.id;
        if (id) {
            const existing = findChildById(target, id);
            if (existing) {
                existing.replaceWith(child.cloneNode(true));
            } else {
                target.appendChild(child.cloneNode(true));
            }
            return;
        }

        const alreadyPresent = Array.from(target.children).some(
            existing => !existing.id && existing.isEqualNode(child)
        );
        if (!alreadyPresent) {
            target.appendChild(child.cloneNode(true));
        }
    });
}

/**
 * Merges the incoming sheet's root <defs> into the main sheet, creating the main <defs>
 * when absent. Definitions (gradients, patterns, filters, clip paths, masks, markers) are
 * hoisted out of symbols server-side, so they must be carried over alongside the symbols
 * or paint-server references such as fill="url(#id)" resolve to nothing.
 * @param mainSheet - The live sprite sheet in the document.
 * @param newSheet - The incoming sprite sheet to merge from.
 */
function mergeDefinitions(mainSheet: SVGSVGElement, newSheet: SVGSVGElement): void {
    const newDefs = newSheet.querySelector(':scope > defs');
    if (!newDefs) {
        return;
    }

    let mainDefs = mainSheet.querySelector(':scope > defs');
    if (!mainDefs) {
        mainDefs = document.createElementNS(SVG_NAMESPACE, 'defs');
        mainSheet.insertBefore(mainDefs, mainSheet.firstChild);
    }

    mergeById(mainDefs, newDefs);
}

/** Manages SVG sprite sheet merging. */
export interface SpriteSheetManager {
    /**
     * Merges new symbols and hoisted definitions from a sprite sheet into the main sheet.
     * Replaces existing symbols and definitions with the same id and appends new ones.
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
                applyHiddenSpriteStyle(newSheet);
                document.body.appendChild(newSheet);
                return;
            }

            mergeDefinitions(mainSheet, newSheet);

            const newSymbols = newSheet.querySelectorAll('symbol');

            newSymbols.forEach(newSymbol => {
                const symbolId = newSymbol.id;
                if (!symbolId) {
                    console.warn('SpriteSheetManager: Found a symbol without an ID, skipping.', newSymbol);
                    return;
                }

                const existingSymbol = findChildById(mainSheet, symbolId);

                if (existingSymbol) {
                    existingSymbol.replaceWith(newSymbol.cloneNode(true));
                } else {
                    mainSheet.appendChild(newSymbol.cloneNode(true));
                }
            });
        },

        ensureExists() {
            if (!document.getElementById('sprite')) {
                const sheet = document.createElementNS(SVG_NAMESPACE, 'svg');
                sheet.id = 'sprite';
                applyHiddenSpriteStyle(sheet);
                document.body.appendChild(sheet);
            }
        }
    };
}
