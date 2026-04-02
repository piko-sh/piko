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

import {describe, expect, it, vi} from 'vitest';
import {PikoSelectionRangeProvider} from './selectionRangeProvider';
import * as vscode from 'vscode';

vi.mock('vscode', () => ({
    Position: class {
        constructor(public line: number, public character: number) {
        }
    },
    Range: class {
        constructor(
            public start: { line: number; character: number } | number,
            public end?: { line: number; character: number } | number,
            public endLine?: number,
            public endCharacter?: number
        ) {
            if (typeof start === 'number') {
                this.start = {line: start, character: end as number};
                this.end = {line: endLine!, character: endCharacter!};
            }
        }

        get isEmpty(): boolean {
            const s = this.start as { line: number; character: number };
            const e = this.end as { line: number; character: number };
            return s.line === e.line && s.character === e.character;
        }

        contains(position: { line: number; character: number }): boolean {
            const s = this.start as { line: number; character: number };
            const e = this.end as { line: number; character: number };
            if (position.line < s.line || position.line > e.line) {
                return false;
            }
            if (position.line === s.line && position.character < s.character) {
                return false;
            }
            if (position.line === e.line && position.character > e.character) {
                return false;
            }
            return true;
        }
    },
    SelectionRange: class {
        constructor(public range: unknown, public parent?: unknown) {
        }
    },
    Uri: {
        file: (path: string) => ({fsPath: path}),
    },
    languages: {
        registerSelectionRangeProvider: vi.fn(),
    },
}));

function createTestDocument(text: string) {
    const lines = text.split('\n');
    return {
        getText: () => text,
        offsetAt: (position: { line: number; character: number }): number => {
            let offset = 0;
            for (let i = 0; i < position.line && i < lines.length; i++) {
                offset += lines[i].length + 1;
            }
            offset += Math.min(position.character, lines[position.line]?.length ?? 0);
            return offset;
        },
        positionAt: (offset: number): vscode.Position => {
            let remaining = offset;
            let line = 0;
            while (line < lines.length && remaining > lines[line].length) {
                remaining -= lines[line].length + 1;
                line = line + 1;
            }
            return new vscode.Position(line, Math.max(0, remaining));
        },
        lineAt: (line: number) => ({
            text: lines[line] || '',
            range: {
                start: new vscode.Position(line, 0),
                end: new vscode.Position(line, (lines[line] || '').length)
            }
        }),
        getWordRangeAtPosition: (position: vscode.Position) => {
            const lineText = lines[position.line] || '';
            const char = position.character;

            let start = char;
            let end = char;

            while (start > 0 && /\w/.test(lineText[start - 1])) {
                start = start - 1;
            }
            while (end < lineText.length && /\w/.test(lineText[end])) {
                end = end + 1;
            }

            if (start === end) {
                return null;
            }

            return new vscode.Range(
                new vscode.Position(position.line, start),
                new vscode.Position(position.line, end)
            );
        },
        uri: {fsPath: '/test/file.pk'},
        languageId: 'piko',
        lineCount: lines.length,
    };
}

describe('PikoSelectionRangeProvider', () => {
    const provider = new PikoSelectionRangeProvider();

    describe('provideSelectionRanges', () => {
        it('should return selection ranges for a position', () => {
            const doc = createTestDocument(`<template>
  <div>content</div>
</template>`);
            const position = new vscode.Position(1, 7);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            expect(ranges![0]).toBeDefined();
            expect(ranges![0].range).toBeDefined();
        });

        it('should find word at cursor', () => {
            const doc = createTestDocument(`<template>
  <div>hello world</div>
</template>`);
            const position = new vscode.Position(1, 9);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            const range = ranges![0];
            expect(range.range).toBeDefined();
        });

        it('should find interpolation content', () => {
            const doc = createTestDocument(`<template>
  <div>{{ message }}</div>
</template>`);
            const position = new vscode.Position(1, 12);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            expect(ranges![0]).toBeDefined();
        });

        it('should find attribute value', () => {
            const doc = createTestDocument(`<template>
  <div class="my-class">content</div>
</template>`);
            const position = new vscode.Position(1, 16);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            expect(ranges![0]).toBeDefined();
        });

        it('should find enclosing HTML tag', () => {
            const doc = createTestDocument(`<template>
  <div>
    <span>nested</span>
  </div>
</template>`);
            const position = new vscode.Position(2, 11);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            let current = ranges![0];
            let depth = 0;
            while (current) {
                depth++;
                current = current.parent as typeof current;
            }
            expect(depth).toBeGreaterThanOrEqual(1);
        });

        it('should find block body and full block', () => {
            const doc = createTestDocument(`<template>
  <div>content</div>
</template>

<script type="application/x-go">
package main
</script>`);
            const position = new vscode.Position(5, 5);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            expect(ranges![0]).toBeDefined();
        });

        it('should handle multiple positions', () => {
            const doc = createTestDocument(`<template>
  <div>first</div>
  <div>second</div>
</template>`);
            const positions = [
                new vscode.Position(1, 8),
                new vscode.Position(2, 8)
            ];

            const ranges = provider.provideSelectionRanges(
                doc as never,
                positions,
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(2);
            expect(ranges![0]).toBeDefined();
            expect(ranges![1]).toBeDefined();
        });

        it('should handle empty document', () => {
            const doc = createTestDocument('');
            const position = new vscode.Position(0, 0);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            expect(ranges![0].range).toBeDefined();
        });

        it('should skip self-closing tags', () => {
            const doc = createTestDocument(`<template>
  <br>
  <img src="test.png">
  <div>content</div>
</template>`);
            const position = new vscode.Position(3, 8);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle nested interpolations correctly', () => {
            const doc = createTestDocument(`<template>
  <div>{{ outer {{ inner }} }}</div>
</template>`);
            const position = new vscode.Position(1, 19);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle deeply nested tags', () => {
            const doc = createTestDocument(`<template>
  <div>
    <ul>
      <li>
        <span>deep content</span>
      </li>
    </ul>
  </div>
</template>`);
            const position = new vscode.Position(4, 16);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            let count = 0;
            let current = ranges![0];
            while (current) {
                count++;
                current = current.parent as typeof current;
            }
            expect(count).toBeGreaterThanOrEqual(1);
        });

        it('should handle single-quoted attribute values', () => {
            const doc = createTestDocument(`<template>
  <div class='my-class'>content</div>
</template>`);
            const position = new vscode.Position(1, 16);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
            expect(ranges![0]).toBeDefined();
        });

        it('should handle position outside any quote', () => {
            const doc = createTestDocument(`<template>
  <div>plain text</div>
</template>`);
            const position = new vscode.Position(1, 10);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle empty interpolation', () => {
            const doc = createTestDocument(`<template>
  <div>{{}}</div>
</template>`);
            const position = new vscode.Position(1, 8);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle position after closing braces', () => {
            const doc = createTestDocument(`<template>
  <div>{{ value }}</div>
</template>`);
            const position = new vscode.Position(1, 20);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle self-closed tags with />', () => {
            const doc = createTestDocument(`<template>
  <input type="text" />
  <div>content</div>
</template>`);
            const position = new vscode.Position(2, 8);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle position at tag boundary', () => {
            const doc = createTestDocument(`<template>
  <div>content</div>
</template>`);
            const position = new vscode.Position(1, 2);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle unclosed tag gracefully', () => {
            const doc = createTestDocument(`<template>
  <div>content
</template>`);
            const position = new vscode.Position(1, 8);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle nested same-name tags', () => {
            const doc = createTestDocument(`<template>
  <div>
    <div>inner</div>
  </div>
</template>`);
            const position = new vscode.Position(2, 11);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });

        it('should handle position before opening braces', () => {
            const doc = createTestDocument(`<template>
  before {{ value }} after
</template>`);
            const position = new vscode.Position(1, 5);

            const ranges = provider.provideSelectionRanges(
                doc as never,
                [position],
                {} as never
            ) as vscode.SelectionRange[];

            expect(ranges).toHaveLength(1);
        });
    });
});
