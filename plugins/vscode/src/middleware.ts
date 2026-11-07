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

import * as vscode from 'vscode';
import {Middleware} from 'vscode-languageclient';
import {BlockInfo, BlockType, detectBlockAtPosition, getDocumentBlocks, isInServerBlock} from './blockDetector';
import {findBlockAtOffset, SfcBlock} from './sfcBlocks';
import {createVirtualTsUri, getHeaderLineCount, VIRTUAL_TS_SCHEME} from './virtualTs';

/**
 * Maps a position in the .pk document into the virtual TypeScript document.
 *
 * @param contentStart - The position where the script content starts in the .pk document.
 * @param headerLineCount - The number of header lines in the virtual document.
 * @param position - The position in the .pk document.
 * @returns The position in the virtual document.
 */
export function toVirtualPosition(
    contentStart: vscode.Position,
    headerLineCount: number,
    position: vscode.Position
): vscode.Position {
    if (position.line === contentStart.line) {
        return new vscode.Position(headerLineCount, Math.max(0, position.character - contentStart.character));
    }
    return new vscode.Position(position.line - contentStart.line + headerLineCount, position.character);
}

/**
 * Maps a position in the virtual TypeScript document back into the .pk document.
 *
 * Positions inside the generated header are clamped to the start of the
 * script content, since the header has no source in the .pk file.
 *
 * @param contentStart - The position where the script content starts in the .pk document.
 * @param headerLineCount - The number of header lines in the virtual document.
 * @param position - The position in the virtual document.
 * @returns The position in the .pk document.
 */
export function toPikoPosition(
    contentStart: vscode.Position,
    headerLineCount: number,
    position: vscode.Position
): vscode.Position {
    const scriptLine = position.line - headerLineCount;
    if (scriptLine < 0) {
        return new vscode.Position(contentStart.line, contentStart.character);
    }
    if (scriptLine === 0) {
        return new vscode.Position(contentStart.line, position.character + contentStart.character);
    }
    return new vscode.Position(scriptLine + contentStart.line, position.character);
}

/**
 * Calculates the position within the virtual TypeScript document.
 *
 * @param document - The Piko document.
 * @param position - The position in the full document.
 * @param block - The script block info.
 * @param headerLineCount - The number of header lines in the virtual document.
 * @returns The position within the virtual TypeScript document.
 */
export function calculateTsPositionInBlock(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo,
    headerLineCount: number
): vscode.Position {
    return toVirtualPosition(document.positionAt(block.contentStartOffset), headerLineCount, position);
}

/**
 * Translates a range from the virtual TypeScript document back to the .pk document.
 *
 * @param document - The Piko document.
 * @param range - The range in the virtual document.
 * @param block - The script block info.
 * @param headerLineCount - The number of header lines in the virtual document.
 * @returns The translated range.
 */
export function translateTsRange(
    document: vscode.TextDocument,
    range: vscode.Range,
    block: BlockInfo,
    headerLineCount: number
): vscode.Range {
    const contentStart = document.positionAt(block.contentStartOffset);
    return new vscode.Range(
        toPikoPosition(contentStart, headerLineCount, range.start),
        toPikoPosition(contentStart, headerLineCount, range.end)
    );
}

/**
 * Translates a location from the virtual TypeScript document back to the .pk document.
 *
 * Locations not in a virtual TypeScript document are returned unchanged;
 * locations inside the generated header are clamped to the script start.
 *
 * @param document - The Piko document.
 * @param location - The location from the virtual document.
 * @param block - The script block info.
 * @param headerLineCount - The number of header lines in the virtual document.
 * @returns The translated location in the .pk document.
 */
export function translateTsLocation(
    document: vscode.TextDocument,
    location: vscode.Location,
    block: BlockInfo,
    headerLineCount: number
): vscode.Location {
    if (location.uri.scheme !== VIRTUAL_TS_SCHEME) {
        return location;
    }
    return new vscode.Location(document.uri, translateTsRange(document, location.range, block, headerLineCount));
}

/**
 * Reports whether a virtual range lies entirely inside the generated header.
 *
 * @param range - The range in the virtual document.
 * @param headerLineCount - The number of header lines in the virtual document.
 * @returns True when the range has no counterpart in the .pk file.
 */
function isInHeader(range: vscode.Range, headerLineCount: number): boolean {
    return range.end.line < headerLineCount;
}

/**
 * Translates the ranges of completion items from the virtual document back
 * to the .pk document.
 *
 * Item ranges are rewritten (including insert/replace pairs) and additional
 * edits that target the generated header are dropped.
 *
 * @param document - The Piko document.
 * @param list - The completion list from the virtual document.
 * @param block - The script block info.
 * @param headerLineCount - The number of header lines in the virtual document.
 * @returns The same list with translated ranges.
 */
export function translateTsCompletionList(
    document: vscode.TextDocument,
    list: vscode.CompletionList,
    block: BlockInfo,
    headerLineCount: number
): vscode.CompletionList {
    const translate = (range: vscode.Range): vscode.Range => translateTsRange(document, range, block, headerLineCount);

    for (const item of list.items) {
        const range = item.range;
        if (range !== undefined) {
            item.range = 'inserting' in range
                ? {inserting: translate(range.inserting), replacing: translate(range.replacing)}
                : translate(range);
        }
        if (item.additionalTextEdits) {
            item.additionalTextEdits = item.additionalTextEdits
                .filter(edit => !isInHeader(edit.range, headerLineCount))
                .map(edit => new vscode.TextEdit(translate(edit.range), edit.newText));
        }
    }
    return list;
}

/**
 * Prepares a TypeScript request: ensures the virtual document has been
 * served (so its header length is known) and maps the cursor position.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The TypeScript block info.
 * @returns The virtual URI, mapped position and header line count.
 */
async function prepareTsRequest(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo
): Promise<{ virtualUri: vscode.Uri; virtualPosition: vscode.Position; headerLineCount: number }> {
    const virtualUri = createVirtualTsUri(document.uri);
    await vscode.workspace.openTextDocument(virtualUri);
    const headerLineCount = getHeaderLineCount(virtualUri);
    return {
        virtualUri,
        virtualPosition: calculateTsPositionInBlock(document, position, block, headerLineCount),
        headerLineCount,
    };
}

/**
 * Handles Go-to-Definition requests for TypeScript blocks.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The TypeScript block info.
 * @returns The definition locations, or null if not found.
 */
async function handleTsDefinition(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo
): Promise<vscode.Location[] | null> {
    const {virtualUri, virtualPosition, headerLineCount} = await prepareTsRequest(document, position, block);

    const results = await vscode.commands.executeCommand<Array<vscode.Location | vscode.LocationLink> | undefined>(
        'vscode.executeDefinitionProvider',
        virtualUri,
        virtualPosition
    );

    if (!results || results.length === 0) {
        return null;
    }
    return results.map(result => {
        const location = 'targetUri' in result
            ? new vscode.Location(result.targetUri, result.targetSelectionRange ?? result.targetRange)
            : result;
        return translateTsLocation(document, location, block, headerLineCount);
    });
}

/**
 * Handles Hover requests for TypeScript blocks.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The TypeScript block info.
 * @returns The hover info, or null if not found.
 */
async function handleTsHover(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo
): Promise<vscode.Hover | null> {
    const {virtualUri, virtualPosition, headerLineCount} = await prepareTsRequest(document, position, block);

    const hovers = await vscode.commands.executeCommand<vscode.Hover[] | undefined>(
        'vscode.executeHoverProvider',
        virtualUri,
        virtualPosition
    );

    if (!hovers || hovers.length === 0) {
        return null;
    }

    const hover = hovers[0];
    if (hover.range) {
        hover.range = translateTsRange(document, hover.range, block, headerLineCount);
    }
    return hover;
}

/**
 * Handles Code Completion requests for TypeScript blocks.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The TypeScript block info.
 * @param triggerCharacter - The character that triggered completion.
 * @returns The completion list with ranges mapped back, or null if not available.
 */
async function handleTsCompletion(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo,
    triggerCharacter: string | undefined
): Promise<vscode.CompletionList | null> {
    const {virtualUri, virtualPosition, headerLineCount} = await prepareTsRequest(document, position, block);

    const list = await vscode.commands.executeCommand<vscode.CompletionList | undefined>(
        'vscode.executeCompletionItemProvider',
        virtualUri,
        virtualPosition,
        triggerCharacter
    );

    return list ? translateTsCompletionList(document, list, block, headerLineCount) : null;
}

/**
 * Handles Signature Help requests for TypeScript blocks.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The TypeScript block info.
 * @param triggerCharacter - The character that triggered signature help.
 * @returns The signature help, or null if not available.
 */
async function handleTsSignatureHelp(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo,
    triggerCharacter: string | undefined
): Promise<vscode.SignatureHelp | null> {
    const {virtualUri, virtualPosition} = await prepareTsRequest(document, position, block);

    const help = await vscode.commands.executeCommand<vscode.SignatureHelp | undefined>(
        'vscode.executeSignatureHelpProvider',
        virtualUri,
        virtualPosition,
        triggerCharacter
    );
    return help ?? null;
}

/**
 * Runs a middleware handler with error isolation.
 *
 * An unexpected throw in a provide* handler would otherwise fail the request
 * with no logging, so failures are logged and resolved to null (which lets the
 * request fall through harmlessly).
 *
 * @param outputChannel - Optional output channel for logging handler failures.
 * @param label - The handler name used in failure logs.
 * @param handler - The handler body to run.
 * @returns The handler result, or null on failure.
 */
async function runHandler<T>(
    outputChannel: vscode.OutputChannel | undefined,
    label: string,
    handler: () => T | Thenable<T> | null | undefined
): Promise<T | null | undefined> {
    try {
        return await handler();
    } catch (error) {
        outputChannel?.appendLine(`[Middleware] ${label} failed: ${error}`);
        return null;
    }
}

/**
 * Creates the LSP middleware that routes requests to the right handler.
 *
 * Template and Go script blocks are served by pikopls (which delegates Go to
 * gopls through its bridge). TypeScript script blocks are served by VS Code's
 * TypeScript support through virtual documents. Requests in other blocks
 * (CSS, i18n, plain JavaScript) are not sent to pikopls.
 *
 * Block boundaries come from the shared parser in sfcBlocks.ts, which mirrors
 * the server's parser.
 *
 * @param outputChannel - Optional output channel for logging handler failures.
 * @returns The middleware for the language client.
 */
export function createPikoLspMiddleware(outputChannel?: vscode.OutputChannel): Middleware {
    return {
        provideDefinition: (document, position, token, next) => runHandler(outputChannel, 'provideDefinition', () => {
            const block = detectBlockAtPosition(document, position);
            if (block.type === BlockType.Template || block.type === BlockType.Script) {
                return next(document, position, token);
            }
            if (block.type === BlockType.ScriptTS) {
                return handleTsDefinition(document, position, block);
            }
            return null;
        }),

        provideHover: (document, position, token, next) => runHandler(outputChannel, 'provideHover', () => {
            const block = detectBlockAtPosition(document, position);
            if (block.type === BlockType.Template || block.type === BlockType.Script) {
                return next(document, position, token);
            }
            if (block.type === BlockType.ScriptTS) {
                return handleTsHover(document, position, block);
            }
            return null;
        }),

        provideCompletionItem: (document, position, context, token, next) => runHandler(outputChannel, 'provideCompletionItem', () => {
            const block = detectBlockAtPosition(document, position);
            if (block.type === BlockType.Template || block.type === BlockType.Script) {
                return next(document, position, context, token);
            }
            if (block.type === BlockType.ScriptTS) {
                return handleTsCompletion(document, position, block, context.triggerCharacter);
            }
            return null;
        }),

        provideSignatureHelp: (document, position, context, token, next) => runHandler(outputChannel, 'provideSignatureHelp', () => {
            const block = detectBlockAtPosition(document, position);
            if (block.type === BlockType.Template || block.type === BlockType.Script) {
                return next(document, position, context, token);
            }
            if (block.type === BlockType.ScriptTS) {
                return handleTsSignatureHelp(document, position, block, context.triggerCharacter);
            }
            return null;
        }),

        provideReferences: (document, position, context, token, next) =>
            isInServerBlock(document, position) ? next(document, position, context, token) : null,
        provideDocumentHighlights: (document, position, token, next) =>
            isInServerBlock(document, position) ? next(document, position, token) : null,
        provideTypeDefinition: (document, position, token, next) =>
            isInServerBlock(document, position) ? next(document, position, token) : null,
        provideDeclaration: (document, position, token, next) =>
            isInServerBlock(document, position) ? next(document, position, token) : null,
        provideImplementation: (document, position, token, next) =>
            isInServerBlock(document, position) ? next(document, position, token) : null,
        prepareRename: (document, position, token, next) =>
            isInServerBlock(document, position) ? next(document, position, token) : null,
        provideRenameEdits: (document, position, newName, token, next) =>
            isInServerBlock(document, position) ? next(document, position, newName, token) : null,
        provideCodeActions: (document, range, context, token, next) =>
            isInServerBlock(document, range.start) ? next(document, range, context, token) : null,
        provideDocumentLinks: async (document, token, next) => {
            const links = await next(document, token);
            if (!links) {
                return links;
            }
            const blocks = getDocumentBlocks(document);
            return links.filter(link => isServerBlock(findBlockAtOffset(blocks, document.offsetAt(link.range.start))));
        },
    };
}

/**
 * Reports whether a block is served by pikopls: the template or the Go script.
 *
 * @param block - The block, or undefined when outside every block.
 * @returns True when pikopls should handle requests in the block.
 */
function isServerBlock(block: SfcBlock | undefined): boolean {
    return block?.kind === 'template' || block?.kind === 'script';
}
