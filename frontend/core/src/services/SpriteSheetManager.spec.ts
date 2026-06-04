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

import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { createSpriteSheetManager, type SpriteSheetManager } from './SpriteSheetManager';

describe('SpriteSheetManager', () => {
    let manager: SpriteSheetManager;
    let mainSprite: SVGSVGElement | null;

    beforeEach(() => {
        manager = createSpriteSheetManager();
        mainSprite = null;
    });

    afterEach(() => {
        const sprites = document.querySelectorAll('#sprite');
        sprites.forEach(sprite => sprite.remove());
    });

    const createSpriteSheet = (symbols: { id: string; content?: string }[]): SVGSVGElement => {
        const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
        symbols.forEach(({ id, content }) => {
            const symbol = document.createElementNS('http://www.w3.org/2000/svg', 'symbol');
            if (id) {
                symbol.id = id;
            }
            if (content) {
                symbol.innerHTML = content;
            }
            svg.appendChild(symbol);
        });
        return svg;
    };

    const setupMainSprite = (symbols: { id: string; content?: string }[] = []): SVGSVGElement => {
        mainSprite = createSpriteSheet(symbols);
        mainSprite.id = 'sprite';
        mainSprite.style.display = 'none';
        document.body.appendChild(mainSprite);
        return mainSprite;
    };

    describe('merge()', () => {
        it('should do nothing when passed null', () => {
            manager.merge(null);
            expect(document.getElementById('sprite')).toBeNull();
        });

        it('should create main sprite if it does not exist and merge new sheet', () => {
            const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const newSheet = createSpriteSheet([
                { id: 'icon-home', content: '<path d="M1 1"/>' }
            ]);

            manager.merge(newSheet);

            const mainSheet = document.getElementById('sprite') as unknown as SVGSVGElement;
            expect(mainSheet).not.toBeNull();
            expect(mainSheet.style.display).not.toBe('none');
            expect(mainSheet.style.position).toBe('absolute');
            expect(mainSheet.style.overflow).toBe('hidden');
            expect(mainSheet.querySelector('symbol#icon-home')).not.toBeNull();
            expect(consoleSpy).toHaveBeenCalledWith(
                "SpriteSheetManager: Main sprite sheet with id='sprite' not found. Cannot merge new sprites."
            );

            consoleSpy.mockRestore();
        });

        it('should merge new symbols into existing main sprite', () => {
            setupMainSprite([{ id: 'icon-existing' }]);

            const newSheet = createSpriteSheet([
                { id: 'icon-new', content: '<circle r="5"/>' }
            ]);

            manager.merge(newSheet);

            const mainSheet = document.getElementById('sprite') as unknown as SVGSVGElement;
            expect(mainSheet.querySelector('symbol#icon-existing')).not.toBeNull();
            expect(mainSheet.querySelector('symbol#icon-new')).not.toBeNull();
        });

        it('should replace existing symbols with same ID', () => {
            setupMainSprite([
                { id: 'icon-replace', content: '<path d="old"/>' }
            ]);

            const newSheet = createSpriteSheet([
                { id: 'icon-replace', content: '<path d="new"/>' }
            ]);

            manager.merge(newSheet);

            const mainSheet = document.getElementById('sprite') as unknown as SVGSVGElement;
            const symbol = mainSheet.querySelector('symbol#icon-replace');
            expect(symbol).not.toBeNull();
            expect(symbol!.innerHTML).toContain('d="new"');
        });

        it('should skip symbols without an ID and log warning', () => {
            setupMainSprite([]);
            const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const newSheet = createSpriteSheet([
                { id: '', content: '<path d="no-id"/>' },
                { id: 'icon-valid', content: '<path d="valid"/>' }
            ]);
            const noIdSymbol = newSheet.querySelector('symbol:first-child');
            if (noIdSymbol) {
                noIdSymbol.removeAttribute('id');
            }

            manager.merge(newSheet);

            const mainSheet = document.getElementById('sprite') as unknown as SVGSVGElement;
            expect(mainSheet.querySelectorAll('symbol')).toHaveLength(1);
            expect(mainSheet.querySelector('symbol#icon-valid')).not.toBeNull();
            expect(consoleSpy).toHaveBeenCalledWith(
                'SpriteSheetManager: Found a symbol without an ID, skipping.',
                expect.any(Element)
            );

            consoleSpy.mockRestore();
        });

        it('should merge multiple symbols at once', () => {
            setupMainSprite([]);

            const newSheet = createSpriteSheet([
                { id: 'icon-a' },
                { id: 'icon-b' },
                { id: 'icon-c' }
            ]);

            manager.merge(newSheet);

            const mainSheet = document.getElementById('sprite') as unknown as SVGSVGElement;
            expect(mainSheet.querySelectorAll('symbol')).toHaveLength(3);
        });

        it('should clone nodes when merging (not move)', () => {
            setupMainSprite([]);

            const newSheet = createSpriteSheet([
                { id: 'icon-clone', content: '<rect/>' }
            ]);
            const originalSymbol = newSheet.querySelector('symbol#icon-clone');

            manager.merge(newSheet);

            expect(newSheet.querySelector('symbol#icon-clone')).toBe(originalSymbol);
            const mainSheet = document.getElementById('sprite') as unknown as SVGSVGElement;
            expect(mainSheet.querySelector('symbol#icon-clone')).not.toBe(originalSymbol);
        });

        it('should not throw when a symbol id contains a double quote', () => {
            const main = setupMainSprite([]);

            const newSheet = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            const symbol = document.createElementNS('http://www.w3.org/2000/svg', 'symbol');
            symbol.id = 'sym"bol';
            newSheet.appendChild(symbol);

            expect(() => manager.merge(newSheet)).not.toThrow();
            expect(Array.from(main.children).some(child => child.id === 'sym"bol')).toBe(true);
        });
    });

    describe('defs merging', () => {
        const NS = 'http://www.w3.org/2000/svg';

        const createSheetWithDefs = (
            defs: { id: string; tag?: string }[],
            symbols: { id: string; content?: string }[] = []
        ): SVGSVGElement => {
            const svg = document.createElementNS(NS, 'svg');
            const defsEl = document.createElementNS(NS, 'defs');
            defs.forEach(({ id, tag }) => {
                const el = document.createElementNS(NS, tag ?? 'linearGradient');
                if (id) {
                    el.id = id;
                }
                defsEl.appendChild(el);
            });
            svg.appendChild(defsEl);
            symbols.forEach(({ id, content }) => {
                const symbol = document.createElementNS(NS, 'symbol');
                if (id) {
                    symbol.id = id;
                }
                if (content) {
                    symbol.innerHTML = content;
                }
                svg.appendChild(symbol);
            });
            return svg;
        };

        it('should create a defs block and merge definitions when the main sprite has none', () => {
            setupMainSprite([{ id: 'icon-existing' }]);

            const newSheet = createSheetWithDefs(
                [{ id: 'icon-grad' }],
                [{ id: 'icon-new', content: '<path fill="url(#icon-grad)"/>' }]
            );

            manager.merge(newSheet);

            const mainSheet = document.getElementById('sprite') as unknown as SVGSVGElement;
            const mainDefs = mainSheet.querySelector(':scope > defs');
            expect(mainDefs).not.toBeNull();
            expect(mainDefs!.querySelector('#icon-grad')).not.toBeNull();
            expect(mainSheet.querySelector('symbol#icon-new')).not.toBeNull();
        });

        it('should merge definitions into an existing defs block', () => {
            const main = setupMainSprite([]);
            const existingDefs = document.createElementNS(NS, 'defs');
            const existingGrad = document.createElementNS(NS, 'linearGradient');
            existingGrad.id = 'grad-existing';
            existingDefs.appendChild(existingGrad);
            main.insertBefore(existingDefs, main.firstChild);

            const newSheet = createSheetWithDefs([{ id: 'grad-new' }]);

            manager.merge(newSheet);

            const mainDefs = main.querySelector(':scope > defs');
            expect(main.querySelectorAll(':scope > defs')).toHaveLength(1);
            expect(mainDefs!.querySelector('#grad-existing')).not.toBeNull();
            expect(mainDefs!.querySelector('#grad-new')).not.toBeNull();
        });

        it('should replace an existing definition with the same id', () => {
            const main = setupMainSprite([]);
            const defs = document.createElementNS(NS, 'defs');
            const grad = document.createElementNS(NS, 'linearGradient');
            grad.id = 'grad';
            grad.setAttribute('data-version', 'old');
            defs.appendChild(grad);
            main.insertBefore(defs, main.firstChild);

            const newSheet = createSheetWithDefs([{ id: 'grad' }]);
            newSheet.querySelector('#grad')!.setAttribute('data-version', 'new');

            manager.merge(newSheet);

            const merged = main.querySelectorAll('#grad');
            expect(merged).toHaveLength(1);
            expect(merged[0].getAttribute('data-version')).toBe('new');
        });

        it('should carry definitions when promoting a new sheet', () => {
            const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

            const newSheet = createSheetWithDefs(
                [{ id: 'grad-promoted' }],
                [{ id: 'icon-promoted', content: '<path fill="url(#grad-promoted)"/>' }]
            );

            manager.merge(newSheet);

            const mainSheet = document.getElementById('sprite') as unknown as SVGSVGElement;
            expect(mainSheet.querySelector(':scope > defs #grad-promoted')).not.toBeNull();
            expect(mainSheet.querySelector('symbol#icon-promoted')).not.toBeNull();

            consoleSpy.mockRestore();
        });

        it('should not throw when a definition id contains a double quote', () => {
            const main = setupMainSprite([]);

            const newSheet = document.createElementNS(NS, 'svg');
            const defs = document.createElementNS(NS, 'defs');
            const grad = document.createElementNS(NS, 'linearGradient');
            grad.id = 'ab"cd';
            defs.appendChild(grad);
            newSheet.appendChild(defs);

            expect(() => manager.merge(newSheet)).not.toThrow();

            const mainDefs = main.querySelector(':scope > defs');
            expect(mainDefs).not.toBeNull();
            expect(Array.from(mainDefs!.children).some(child => child.id === 'ab"cd')).toBe(true);
        });

        it('should not accumulate id-less defs children across repeated merges', () => {
            setupMainSprite([]);

            const buildSheet = (): SVGSVGElement => {
                const svg = document.createElementNS(NS, 'svg');
                const defs = document.createElementNS(NS, 'defs');
                const style = document.createElementNS(NS, 'style');
                style.textContent = '.cls{fill:red}';
                defs.appendChild(style);
                const grad = document.createElementNS(NS, 'linearGradient');
                grad.id = 'grad-shared';
                defs.appendChild(grad);
                svg.appendChild(defs);
                return svg;
            };

            manager.merge(buildSheet());
            manager.merge(buildSheet());
            manager.merge(buildSheet());

            const mainDefs = document.getElementById('sprite')!.querySelector(':scope > defs');
            expect(mainDefs!.querySelectorAll('style')).toHaveLength(1);
            expect(mainDefs!.querySelectorAll('#grad-shared')).toHaveLength(1);
        });

        it('should leave the main sprite defs untouched when the new sheet has no defs', () => {
            const main = setupMainSprite([{ id: 'icon-x' }]);

            manager.merge(createSpriteSheet([{ id: 'icon-y' }]));

            expect(main.querySelector(':scope > defs')).toBeNull();
            expect(main.querySelector('symbol#icon-x')).not.toBeNull();
            expect(main.querySelector('symbol#icon-y')).not.toBeNull();
        });
    });

    describe('ensureExists()', () => {
        it('should create sprite sheet if it does not exist', () => {
            expect(document.getElementById('sprite')).toBeNull();

            manager.ensureExists();

            const sprite = document.getElementById('sprite');
            expect(sprite).not.toBeNull();
            expect(sprite!.tagName.toLowerCase()).toBe('svg');
            expect((sprite as unknown as SVGSVGElement).style.display).not.toBe('none');
            expect((sprite as unknown as SVGSVGElement).style.position).toBe('absolute');
            expect((sprite as unknown as SVGSVGElement).style.overflow).toBe('hidden');
        });

        it('should not create duplicate if sprite already exists', () => {
            setupMainSprite([]);

            manager.ensureExists();

            const sprites = document.querySelectorAll('#sprite');
            expect(sprites).toHaveLength(1);
        });

        it('should be idempotent when called multiple times', () => {
            manager.ensureExists();
            manager.ensureExists();
            manager.ensureExists();

            const sprites = document.querySelectorAll('#sprite');
            expect(sprites).toHaveLength(1);
        });
    });
});
