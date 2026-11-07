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
import * as vscode from 'vscode';
import {
    calculateTsPositionInBlock,
    createPikoLspMiddleware,
    toPikoPosition,
    toVirtualPosition,
    translateTsCompletionList,
    translateTsLocation
} from './middleware';
import {createMockDocument, createPosition} from './test/mockTextDocument';
import {createVirtualTsUri, forgetHeaderLineCount, recordHeaderLineCount} from './virtualTs';
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
    TextEdit: class {
        constructor(
            public range: { start: { line: number; character: number }; end: { line: number; character: number } },
            public newText: string
        ) {
        }
    },
    workspace: {
        openTextDocument: vi.fn(),
    },
    commands: {
        executeCommand: vi.fn(),
    },
}));

/**
 * Creates a minimal URI mock that supports `with`.
 *
 * @param scheme - The URI scheme.
 * @param path - The URI path.
 * @returns The mock URI.
 */
function mockUri(scheme: string, path: string): { scheme: string; path: string; fsPath: string; with: (change: { scheme?: string; path?: string }) => unknown; toString: () => string } {
    return {
        scheme,
        path,
        fsPath: path,
        with: change => mockUri(change.scheme ?? scheme, change.path ?? path),
        toString: () => `${scheme}://${path}`,
    };
}

describe('middleware', () => {

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

            const result = calculateTsPositionInBlock(doc as never, position as never, block, 5);

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

            const result = calculateTsPositionInBlock(doc as never, position as never, block, 5);

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

            const result = translateTsLocation(doc as never, externalLocation as never, block, 5);

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

            const result = translateTsLocation(doc as never, virtualLocation as never, block, 5);

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

            const result = translateTsLocation(doc as never, virtualLocation as never, block, 5);

            expect(result.range.start.line).toBe(1);
            expect(result.range.start.character).toBe(5);
        });
    });

    describe('header-offset mapping', () => {
        const contentStart = {line: 4, character: 18};

        it.each([0, 3, 5, 9])('round-trips positions with a %i-line header', headerLineCount => {
            const firstLine = {line: 4, character: 25};
            const laterLine = {line: 7, character: 3};

            const virtualFirst = toVirtualPosition(contentStart as never, headerLineCount, firstLine as never);
            const virtualLater = toVirtualPosition(contentStart as never, headerLineCount, laterLine as never);

            expect(virtualFirst).toEqual({line: headerLineCount, character: 7});
            expect(virtualLater).toEqual({line: headerLineCount + 3, character: 3});
            expect(toPikoPosition(contentStart as never, headerLineCount, virtualFirst)).toEqual(firstLine);
            expect(toPikoPosition(contentStart as never, headerLineCount, virtualLater)).toEqual(laterLine);
        });

        it('clamps header positions to the start of the script content', () => {
            expect(toPikoPosition(contentStart as never, 6, {line: 2, character: 40} as never))
                .toEqual({line: 4, character: 18});
        });

        it('never produces a negative virtual character', () => {
            expect(toVirtualPosition(contentStart as never, 2, {line: 4, character: 3} as never))
                .toEqual({line: 2, character: 0});
        });

        it('clamps definitions that point into the header', () => {
            const doc = createMockDocument('<script lang="ts">\nconst x = 1;\n</script>');
            const block: BlockInfo = {
                type: BlockType.ScriptTS,
                contentStartOffset: 18,
                contentEndOffset: 32,
                startLine: 0,
                endLine: 2,
                contentStartLine: 0,
            };
            const headerLocation = {
                uri: {scheme: 'piko-virtual-ts', fsPath: '/test.pk.ts'},
                range: {start: {line: 1, character: 4}, end: {line: 1, character: 9}},
            };

            const result = translateTsLocation(doc as never, headerLocation as never, block, 3);

            expect(result.range.start).toEqual({line: 0, character: 18});
            expect(result.range.end).toEqual({line: 0, character: 18});
        });
    });

    describe('translateTsCompletionList', () => {
        const doc = createMockDocument('<script lang="ts">\nconst value = pk.\n</script>');
        const block: BlockInfo = {
            type: BlockType.ScriptTS,
            contentStartOffset: 18,
            contentEndOffset: 36,
            startLine: 0,
            endLine: 2,
            contentStartLine: 0,
        };
        const range = (line: number, start: number, end: number) => ({
            start: {line, character: start},
            end: {line, character: end},
        });

        it('translates plain and insert/replace ranges and drops header edits', () => {
            const list = {
                items: [
                    {label: 'a', range: range(5, 14, 16)},
                    {label: 'b', range: {inserting: range(5, 14, 15), replacing: range(5, 14, 16)}},
                    {
                        label: 'c',
                        additionalTextEdits: [
                            {range: range(1, 0, 0), newText: 'header'},
                            {range: range(4, 0, 0), newText: 'import x;\n'},
                        ],
                    },
                    {label: 'd'},
                ],
            };

            const result = translateTsCompletionList(doc as never, list as never, block, 4);

            expect(result.items[0].range).toEqual({start: {line: 1, character: 14}, end: {line: 1, character: 16}});
            expect(result.items[1].range).toEqual({
                inserting: {start: {line: 1, character: 14}, end: {line: 1, character: 15}},
                replacing: {start: {line: 1, character: 14}, end: {line: 1, character: 16}},
            });
            expect(result.items[2].additionalTextEdits).toHaveLength(1);
            expect(result.items[2].additionalTextEdits?.[0].range.start).toEqual({line: 0, character: 18});
            expect(result.items[3].range).toBeUndefined();
        });
    });

    describe('createPikoLspMiddleware routing', () => {
        const source = '<template>\n  <p>{{ state.A }}</p>\n</template>\n<script lang="ts">\nconst a = pk.\n</script>\n<style>\n.a {}\n</style>\n<script type="application/x-go">\npackage main\n</script>';
        const executeCommand = vscode.commands.executeCommand as unknown as ReturnType<typeof vi.fn>;
        const openTextDocument = vscode.workspace.openTextDocument as unknown as ReturnType<typeof vi.fn>;

        const makeDoc = () => {
            const doc = createMockDocument(source);
            return {...doc, uri: mockUri('file', '/site/pages/index.pk')};
        };
        const token = {} as never;

        it('sends template and Go positions to the server', async () => {
            const middleware = createPikoLspMiddleware();
            const next = vi.fn().mockResolvedValue('server');
            const doc = makeDoc();

            await expect(middleware.provideHover?.(doc as never, createPosition(1, 8) as never, token, next as never)).resolves.toBe('server');
            await expect(middleware.provideDefinition?.(doc as never, createPosition(10, 2) as never, token, next as never)).resolves.toBe('server');
            await expect(middleware.provideCompletionItem?.(doc as never, createPosition(1, 8) as never, {} as never, token, next as never)).resolves.toBe('server');
            await expect(middleware.provideSignatureHelp?.(doc as never, createPosition(10, 2) as never, {} as never, token, next as never)).resolves.toBe('server');
            expect(next).toHaveBeenCalledTimes(4);
        });

        it('keeps CSS positions away from the server', async () => {
            const middleware = createPikoLspMiddleware();
            const next = vi.fn();
            const doc = makeDoc();
            const cssPosition = createPosition(7, 1) as never;

            await expect(middleware.provideHover?.(doc as never, cssPosition, token, next as never)).resolves.toBeNull();
            await expect(middleware.provideDefinition?.(doc as never, cssPosition, token, next as never)).resolves.toBeNull();
            await expect(middleware.provideCompletionItem?.(doc as never, cssPosition, {} as never, token, next as never)).resolves.toBeNull();
            await expect(middleware.provideSignatureHelp?.(doc as never, cssPosition, {} as never, token, next as never)).resolves.toBeNull();
            expect(middleware.provideReferences?.(doc as never, cssPosition, {} as never, token, next as never)).toBeNull();
            expect(middleware.provideDocumentHighlights?.(doc as never, cssPosition, token, next as never)).toBeNull();
            expect(middleware.provideTypeDefinition?.(doc as never, cssPosition, token, next as never)).toBeNull();
            expect(middleware.provideDeclaration?.(doc as never, cssPosition, token, next as never)).toBeNull();
            expect(middleware.provideImplementation?.(doc as never, cssPosition, token, next as never)).toBeNull();
            expect(middleware.prepareRename?.(doc as never, cssPosition, token, next as never)).toBeNull();
            expect(middleware.provideRenameEdits?.(doc as never, cssPosition, 'x', token, next as never)).toBeNull();
            expect(middleware.provideCodeActions?.(doc as never, {start: cssPosition} as never, {} as never, token, next as never)).toBeNull();
            expect(next).not.toHaveBeenCalled();
        });

        it('wires references, highlights and links through the block detector', async () => {
            const middleware = createPikoLspMiddleware();
            const next = vi.fn().mockReturnValue('server');
            const doc = makeDoc();
            const templatePosition = createPosition(1, 8) as never;

            expect(middleware.provideReferences?.(doc as never, templatePosition, {} as never, token, next as never)).toBe('server');
            expect(middleware.provideDocumentHighlights?.(doc as never, templatePosition, token, next as never)).toBe('server');

            const links = [
                {range: {start: createPosition(1, 4)}},
                {range: {start: createPosition(7, 0)}},
            ];
            const linkNext = vi.fn().mockResolvedValue(links);
            const result = await middleware.provideDocumentLinks?.(doc as never, token, linkNext as never);
            expect(result).toEqual([links[0]]);
        });

        it('routes TypeScript requests through the virtual document with the recorded header', async () => {
            const middleware = createPikoLspMiddleware();
            const next = vi.fn();
            const doc = makeDoc();
            openTextDocument.mockResolvedValue({});
            recordHeaderLineCount(createVirtualTsUri(doc.uri as never), 3);
            executeCommand.mockResolvedValueOnce([{contents: [], range: {start: {line: 4, character: 6}, end: {line: 4, character: 7}}}]);

            const hover = await middleware.provideHover?.(doc as never, createPosition(4, 6) as never, token, next as never) as { range: unknown };

            expect(next).not.toHaveBeenCalled();
            const [command, uri, position] = executeCommand.mock.calls[0];
            expect(command).toBe('vscode.executeHoverProvider');
            expect(uri.scheme).toBe('piko-virtual-ts');
            expect(position).toEqual({line: 4, character: 6});
            forgetHeaderLineCount(createVirtualTsUri(doc.uri as never));
            expect(hover.range).toEqual({start: {line: 4, character: 6}, end: {line: 4, character: 7}});
        });

        it('translates TypeScript definitions, including location links', async () => {
            const middleware = createPikoLspMiddleware();
            const doc = makeDoc();
            openTextDocument.mockResolvedValue({});
            executeCommand.mockResolvedValueOnce([
                {targetUri: {scheme: 'piko-virtual-ts'}, targetRange: {start: {line: 1, character: 6}, end: {line: 1, character: 7}}},
                {uri: {scheme: 'file', fsPath: '/lib.d.ts'}, range: {start: {line: 3, character: 0}, end: {line: 3, character: 1}}},
            ]);

            const result = await middleware.provideDefinition?.(doc as never, createPosition(4, 6) as never, token, vi.fn() as never) as Array<{ uri: unknown; range: { start: unknown } }>;

            expect(result[0].range.start).toEqual({line: 4, character: 6});
            expect(result[1].uri).toEqual({scheme: 'file', fsPath: '/lib.d.ts'});
        });

        it('returns null when TypeScript has no answer', async () => {
            const middleware = createPikoLspMiddleware();
            const doc = makeDoc();
            openTextDocument.mockResolvedValue({});
            executeCommand.mockResolvedValue(undefined);
            const tsPosition = createPosition(4, 6) as never;

            await expect(middleware.provideDefinition?.(doc as never, tsPosition, token, vi.fn() as never)).resolves.toBeNull();
            await expect(middleware.provideHover?.(doc as never, tsPosition, token, vi.fn() as never)).resolves.toBeNull();
            await expect(middleware.provideCompletionItem?.(doc as never, tsPosition, {} as never, token, vi.fn() as never)).resolves.toBeNull();
            await expect(middleware.provideSignatureHelp?.(doc as never, tsPosition, {} as never, token, vi.fn() as never)).resolves.toBeNull();
        });

        it('maps TypeScript completions back into the .pk file', async () => {
            const middleware = createPikoLspMiddleware();
            const doc = makeDoc();
            openTextDocument.mockResolvedValue({});
            executeCommand.mockResolvedValueOnce({items: [{label: 'refs', range: {start: {line: 1, character: 13}, end: {line: 1, character: 13}}}]});

            const list = await middleware.provideCompletionItem?.(doc as never, createPosition(4, 13) as never, {triggerCharacter: '.'} as never, token, vi.fn() as never) as { items: Array<{ range: unknown }> };

            expect(executeCommand.mock.calls[executeCommand.mock.calls.length - 1][3]).toBe('.');
            expect(list.items[0].range).toEqual({start: {line: 4, character: 13}, end: {line: 4, character: 13}});
        });

        it('logs and isolates handler failures', async () => {
            const appendLine = vi.fn();
            const middleware = createPikoLspMiddleware({appendLine} as never);
            const failing = vi.fn().mockImplementation(() => {
                throw new Error('boom');
            });

            await expect(middleware.provideHover?.(makeDoc() as never, createPosition(1, 8) as never, token, failing as never)).resolves.toBeNull();
            expect(appendLine).toHaveBeenCalledWith(expect.stringContaining('provideHover failed'));
        });
    });
});
