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
import {
    calculatePositionInBlock,
    calculateTsPositionInBlock,
    createVirtualGoUri,
    createVirtualTsUri,
    translateLocation,
    translateTsLocation
} from './middleware';
import {createMockDocument, createPosition} from './test/mockTextDocument';
import {BlockInfo, BlockType} from './blockDetector';

vi.mock('vscode', () => ({
    Position: class {
        constructor(public line: number, public character: number) {
        }
    },
    Range: class {
        constructor(
            public start: { line: number; character: number },
            public end: { line: number; character: number }
        ) {
        }
    },
    Location: class {
        constructor(
            public uri: { scheme: string; fsPath: string },
            public range: { start: { line: number; character: number }; end: { line: number; character: number } }
        ) {
        }
    },
    Uri: {
        file: (path: string) => ({scheme: 'file', fsPath: path}),
        parse: (uri: string) => {
            const scheme = uri.split('://')[0];
            const path = uri.replace(`${scheme}://`, '');
            return {scheme, path, fsPath: path};
        },
    },
    commands: {
        executeCommand: vi.fn(),
    },
}));

describe('middleware', () => {
    describe('createVirtualGoUri', () => {
        it('should create virtual Go URI from file URI', () => {
            const fileUri = {scheme: 'file', fsPath: '/path/to/file.pk'};
            const virtualUri = createVirtualGoUri(fileUri as never);

            expect(virtualUri.scheme).toBe('piko-virtual-go');
            expect(virtualUri.path).toContain('/path/to/file.pk.go');
        });

        it('should append .go extension', () => {
            const fileUri = {scheme: 'file', fsPath: '/project/components/header.pk'};
            const virtualUri = createVirtualGoUri(fileUri as never);

            expect(virtualUri.path).toContain('header.pk.go');
        });

        it('should handle Windows-style paths', () => {
            const fileUri = {scheme: 'file', fsPath: 'C:\\Users\\dev\\file.pk'};
            const virtualUri = createVirtualGoUri(fileUri as never);

            expect(virtualUri.scheme).toBe('piko-virtual-go');
            expect(virtualUri.path).toContain('.go');
        });

        it('should handle paths with spaces', () => {
            const fileUri = {scheme: 'file', fsPath: '/path/to/my file.pk'};
            const virtualUri = createVirtualGoUri(fileUri as never);

            expect(virtualUri.path).toContain('my file.pk.go');
        });
    });

    describe('calculatePositionInBlock', () => {
        it('should calculate position on first line of script block', () => {
            const doc = createMockDocument('<script type="application/x-go">package main</script>');
            const position = createPosition(0, 39);
            const block: BlockInfo = {
                type: BlockType.Script,
                contentStartOffset: 31,
                contentEndOffset: 43,
                startLine: 0,
                endLine: 0,
                contentStartLine: 0,
            };

            const result = calculatePositionInBlock(doc as never, position as never, block);

            expect(result.line).toBe(0);
            expect(result.character).toBe(8);
        });

        it('should calculate position on subsequent lines', () => {
            const doc = createMockDocument(`<script type="application/x-go">
package main

func Test() {}
</script>`);
            const position = createPosition(3, 5);
            const block: BlockInfo = {
                type: BlockType.Script,
                contentStartOffset: 31,
                contentEndOffset: 60,
                startLine: 0,
                endLine: 4,
                contentStartLine: 1,
            };

            const result = calculatePositionInBlock(doc as never, position as never, block);

            expect(result.line).toBe(3);
            expect(result.character).toBe(5);
        });

        it('should handle position at content start', () => {
            const doc = createMockDocument('<script type="application/x-go">x</script>');
            const position = createPosition(0, 31);
            const block: BlockInfo = {
                type: BlockType.Script,
                contentStartOffset: 31,
                contentEndOffset: 32,
                startLine: 0,
                endLine: 0,
                contentStartLine: 0,
            };

            const result = calculatePositionInBlock(doc as never, position as never, block);

            expect(result.line).toBe(0);
            expect(result.character).toBe(0);
        });

        it('should preserve character position on non-first lines', () => {
            const doc = createMockDocument(`<script type="application/x-go">
    func main() {
        fmt.Println("hello")
    }
</script>`);
            const position = createPosition(2, 12);
            const block: BlockInfo = {
                type: BlockType.Script,
                contentStartOffset: 31,
                contentEndOffset: 80,
                startLine: 0,
                endLine: 4,
                contentStartLine: 1,
            };

            const result = calculatePositionInBlock(doc as never, position as never, block);

            expect(result.character).toBe(12);
        });
    });

    describe('translateLocation', () => {
        it('should pass through non-virtual locations unchanged', () => {
            const doc = createMockDocument('<script type="application/x-go">package main</script>');
            const goStdlibLocation = {
                uri: {scheme: 'file', fsPath: '/usr/local/go/src/fmt/print.go'},
                range: {
                    start: {line: 10, character: 0},
                    end: {line: 10, character: 20},
                },
            };
            const block: BlockInfo = {
                type: BlockType.Script,
                contentStartOffset: 31,
                contentEndOffset: 43,
                startLine: 0,
                endLine: 0,
                contentStartLine: 0,
            };

            const result = translateLocation(doc as never, goStdlibLocation as never, block);

            expect(result.uri.scheme).toBe('file');
            expect(result.uri.fsPath).toBe('/usr/local/go/src/fmt/print.go');
        });

        it('should translate virtual location on line 0', () => {
            const doc = createMockDocument('<script type="application/x-go">package main</script>');
            const virtualLocation = {
                uri: {scheme: 'piko-virtual-go', fsPath: '/test.pk.go'},
                range: {
                    start: {line: 0, character: 0},
                    end: {line: 0, character: 7},
                },
            };
            const block: BlockInfo = {
                type: BlockType.Script,
                contentStartOffset: 31,
                contentEndOffset: 43,
                startLine: 0,
                endLine: 0,
                contentStartLine: 0,
            };

            const result = translateLocation(doc as never, virtualLocation as never, block);

            expect(result.range.start.line).toBe(0);
            expect(result.range.start.character).toBe(31);
            expect(result.range.end.character).toBe(38);
        });

        it('should translate virtual location on subsequent lines', () => {
            const doc = createMockDocument(`<script type="application/x-go">
package main

func Test() {}
</script>`);
            const virtualLocation = {
                uri: {scheme: 'piko-virtual-go', fsPath: '/test.pk.go'},
                range: {
                    start: {line: 2, character: 5},
                    end: {line: 2, character: 9},
                },
            };
            const block: BlockInfo = {
                type: BlockType.Script,
                contentStartOffset: 31,
                contentEndOffset: 60,
                startLine: 0,
                endLine: 4,
                contentStartLine: 1,
            };

            const result = translateLocation(doc as never, virtualLocation as never, block);

            expect(result.range.start.character).toBe(5);
            expect(result.range.end.character).toBe(9);
        });

        it('should use document URI for translated location', () => {
            const doc = createMockDocument('<script type="application/x-go">x</script>');
            (doc as unknown as Record<string, unknown>).uri = {scheme: 'file', fsPath: '/project/file.pk'};

            const virtualLocation = {
                uri: {scheme: 'piko-virtual-go', fsPath: '/project/file.pk.go'},
                range: {
                    start: {line: 0, character: 0},
                    end: {line: 0, character: 1},
                },
            };
            const block: BlockInfo = {
                type: BlockType.Script,
                contentStartOffset: 31,
                contentEndOffset: 32,
                startLine: 0,
                endLine: 0,
                contentStartLine: 0,
            };

            const result = translateLocation(doc as never, virtualLocation as never, block);

            expect(result.uri.fsPath).toBe('/project/file.pk');
        });
    });

    describe('createVirtualTsUri', () => {
        it('should create virtual TypeScript URI from file URI', () => {
            const fileUri = {scheme: 'file', fsPath: '/path/to/file.pk'};
            const virtualUri = createVirtualTsUri(fileUri as never);

            expect(virtualUri.scheme).toBe('piko-virtual-ts');
            expect(virtualUri.path).toContain('/path/to/file.pk.ts');
        });

        it('should append .ts extension', () => {
            const fileUri = {scheme: 'file', fsPath: '/project/components/header.pk'};
            const virtualUri = createVirtualTsUri(fileUri as never);

            expect(virtualUri.path).toContain('header.pk.ts');
        });
    });

    describe('calculateTsPositionInBlock', () => {
        it('should calculate position accounting for reference directive offset', () => {
            const doc = createMockDocument('<script lang="ts">const x = 1;</script>');
            const position = createPosition(0, 25);
            const block: BlockInfo = {
                type: BlockType.ScriptTS,
                contentStartOffset: 18,
                contentEndOffset: 30,
                startLine: 0,
                endLine: 0,
                contentStartLine: 0,
            };

            const result = calculateTsPositionInBlock(doc as never, position as never, block);

            expect(result.line).toBe(5);
            expect(result.character).toBe(7);
        });

        it('should handle multiline TypeScript content', () => {
            const doc = createMockDocument(`<script lang="ts">
const x: number = 1;
const y: string = "test";
</script>`);
            const position = createPosition(2, 10);
            const block: BlockInfo = {
                type: BlockType.ScriptTS,
                contentStartOffset: 18,
                contentEndOffset: 60,
                startLine: 0,
                endLine: 3,
                contentStartLine: 0,
            };

            const result = calculateTsPositionInBlock(doc as never, position as never, block);

            expect(result.line).toBe(7);
            expect(result.character).toBe(10);
        });
    });

    describe('translateTsLocation', () => {
        it('should pass through non-virtual TypeScript locations unchanged', () => {
            const doc = createMockDocument('<script lang="ts">const x = 1;</script>');
            const externalLocation = {
                uri: {scheme: 'file', fsPath: '/node_modules/typescript/lib.d.ts'},
                range: {
                    start: {line: 10, character: 0},
                    end: {line: 10, character: 20},
                },
            };
            const block: BlockInfo = {
                type: BlockType.ScriptTS,
                contentStartOffset: 18,
                contentEndOffset: 30,
                startLine: 0,
                endLine: 0,
                contentStartLine: 0,
            };

            const result = translateTsLocation(doc as never, externalLocation as never, block);

            expect(result.uri.scheme).toBe('file');
            expect(result.uri.fsPath).toBe('/node_modules/typescript/lib.d.ts');
        });

        it('should translate virtual TypeScript location accounting for reference offset', () => {
            const doc = createMockDocument('<script lang="ts">const x = 1;</script>');
            const virtualLocation = {
                uri: {scheme: 'piko-virtual-ts', fsPath: '/test.pk.ts'},
                range: {
                    start: {line: 5, character: 0},
                    end: {line: 5, character: 5},
                },
            };
            const block: BlockInfo = {
                type: BlockType.ScriptTS,
                contentStartOffset: 18,
                contentEndOffset: 30,
                startLine: 0,
                endLine: 0,
                contentStartLine: 0,
            };

            const result = translateTsLocation(doc as never, virtualLocation as never, block);

            expect(result.range.start.line).toBe(0);
            expect(result.range.start.character).toBe(18);
        });

        it('should translate multiline virtual TypeScript location', () => {
            const doc = createMockDocument(`<script lang="ts">
const x: number = 1;
const y: string = "test";
</script>`);
            const virtualLocation = {
                uri: {scheme: 'piko-virtual-ts', fsPath: '/test.pk.ts'},
                range: {
                    start: {line: 6, character: 5},
                    end: {line: 6, character: 10},
                },
            };
            const block: BlockInfo = {
                type: BlockType.ScriptTS,
                contentStartOffset: 18,
                contentEndOffset: 60,
                startLine: 0,
                endLine: 3,
                contentStartLine: 0,
            };

            const result = translateTsLocation(doc as never, virtualLocation as never, block);

            expect(result.range.start.line).toBe(1);
            expect(result.range.start.character).toBe(5);
        });
    });
});
