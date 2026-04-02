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
import {BlockInfo, BlockType, detectBlockAtPosition, isInTemplateBlock} from './blockDetector';

/**
 * Number of lines added by type reference directives in virtual TS documents.
 * - 2 lib references (esnext, dom) for modern JS features
 * - 2 Piko type references (piko-ide.d.ts, piko-actions.d.ts) for autocomplete
 * - 1 refs declaration (per-file p-ref types or generic fallback)
 */
const TS_REFERENCE_LINE_OFFSET = 5;

/**
 * Creates a URI for a virtual Go document.
 *
 * Converts file:///path/to/doc.pk to piko-virtual-go:///path/to/doc.pk.go
 *
 * @param docUri - The original document URI.
 * @returns The virtual Go URI with .go extension.
 */
export function createVirtualGoUri(docUri: vscode.Uri): vscode.Uri {
    return vscode.Uri.parse(`piko-virtual-go:///${docUri.fsPath}.go`);
}

/**
 * Creates a URI for a virtual TypeScript document.
 *
 * Converts file:///path/to/doc.pk to piko-virtual-ts:///path/to/doc.pk.ts
 *
 * @param docUri - The original document URI.
 * @returns The virtual TypeScript URI with .ts extension.
 */
export function createVirtualTsUri(docUri: vscode.Uri): vscode.Uri {
    return vscode.Uri.parse(`piko-virtual-ts:///${docUri.fsPath}.ts`);
}

/**
 * Calculates the position within the virtual Go document.
 *
 * Maps a position in the full .pk file to the position within the script block.
 *
 * @param document - The Piko document.
 * @param position - The position in the full document.
 * @param block - The script block info.
 * @returns The position within the script block content.
 */
export function calculatePositionInBlock(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo
): vscode.Position {
    const contentStartPos = document.positionAt(block.contentStartOffset);
    const lineOffset = contentStartPos.line;
    const charOffset = contentStartPos.character;

    if (position.line === lineOffset) {
        return new vscode.Position(0, position.character - charOffset);
    }
    return new vscode.Position(position.line - lineOffset, position.character);

}

/**
 * Translates a location from the virtual Go document back to the .pk document.
 *
 * Locations not in the virtual file (like Go stdlib) are returned unchanged.
 *
 * @param document - The Piko document.
 * @param location - The location from the virtual document.
 * @param block - The script block info.
 * @returns The translated location in the .pk document.
 */
export function translateLocation(
    document: vscode.TextDocument,
    location: vscode.Location,
    block: BlockInfo
): vscode.Location {
    if (location.uri.scheme !== 'piko-virtual-go') {
        return location;
    }

    const contentStartPos = document.positionAt(block.contentStartOffset);
    const lineOffset = contentStartPos.line;
    const charOffset = contentStartPos.character;

    const translate = (pos: vscode.Position): vscode.Position => {
        if (pos.line === 0) {
            return new vscode.Position(lineOffset, pos.character + charOffset);
        }
        return new vscode.Position(pos.line + lineOffset, pos.character);

    };

    return new vscode.Location(
        document.uri,
        new vscode.Range(
            translate(location.range.start),
            translate(location.range.end)
        )
    );
}

/**
 * Handles Go-to-Definition requests for script blocks by proxying to gopls.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The script block info.
 * @returns The definition locations, or null if not found.
 */
async function handleScriptDefinition(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo
): Promise<vscode.Location[] | null> {
    const virtualUri = createVirtualGoUri(document.uri);
    const positionInScript = calculatePositionInBlock(document, position, block);

    const locations = await vscode.commands.executeCommand<vscode.Location[]>(
        'vscode.executeDefinitionProvider',
        virtualUri,
        positionInScript
    );

    if (locations.length === 0) {
        return null;
    }
    return locations.map(loc => translateLocation(document, loc, block));
}

/**
 * Handles Hover requests for script blocks by proxying to gopls.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The script block info.
 * @returns The hover info, or null if not found.
 */
async function handleScriptHover(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo
): Promise<vscode.Hover | null> {
    const virtualUri = createVirtualGoUri(document.uri);
    const positionInScript = calculatePositionInBlock(document, position, block);

    const hovers = await vscode.commands.executeCommand<vscode.Hover[]>(
        'vscode.executeHoverProvider',
        virtualUri,
        positionInScript
    );

    if (hovers.length === 0) {
        return null;
    }

    const translatedHover = hovers[0];
    const hoverRange = translatedHover.range;
    if (hoverRange) {
        const start = translateLocation(document, new vscode.Location(virtualUri, hoverRange), block).range.start;
        const end = translateLocation(document, new vscode.Location(virtualUri, hoverRange), block).range.end;
        translatedHover.range = new vscode.Range(start, end);
    }

    return translatedHover;
}

/**
 * Handles Code Completion requests for script blocks by proxying to gopls.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The script block info.
 * @param triggerCharacter - The character that triggered completion.
 * @returns The completion list, or null if not available.
 */
async function handleScriptCompletion(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo,
    triggerCharacter: string | undefined
): Promise<vscode.CompletionList | null> {
    const virtualUri = createVirtualGoUri(document.uri);
    const positionInScript = calculatePositionInBlock(document, position, block);

    return vscode.commands.executeCommand<vscode.CompletionList>(
        'vscode.executeCompletionItemProvider',
        virtualUri,
        positionInScript,
        triggerCharacter
    );
}

/**
 * Handles Signature Help requests for script blocks by proxying to gopls.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The script block info.
 * @param triggerCharacter - The character that triggered signature help.
 * @returns The signature help, or null if not available.
 */
async function handleScriptSignatureHelp(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo,
    triggerCharacter: string | undefined
): Promise<vscode.SignatureHelp | null> {
    const virtualUri = createVirtualGoUri(document.uri);
    const positionInScript = calculatePositionInBlock(document, position, block);

    return vscode.commands.executeCommand<vscode.SignatureHelp>(
        'vscode.executeSignatureHelpProvider',
        virtualUri,
        positionInScript,
        triggerCharacter
    );
}

/**
 * Calculates the position within the virtual TypeScript document.
 *
 * Maps a position in the full .pk file to the position within the TypeScript
 * virtual document, accounting for the reference directives prepended.
 *
 * @param document - The Piko document.
 * @param position - The position in the full document.
 * @param block - The script block info.
 * @returns The position within the virtual TypeScript document.
 */
export function calculateTsPositionInBlock(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo
): vscode.Position {
    const contentStartPos = document.positionAt(block.contentStartOffset);
    const lineOffset = contentStartPos.line;
    const charOffset = contentStartPos.character;

    if (position.line === lineOffset) {
        return new vscode.Position(TS_REFERENCE_LINE_OFFSET, position.character - charOffset);
    }
    return new vscode.Position(position.line - lineOffset + TS_REFERENCE_LINE_OFFSET, position.character);
}

/**
 * Translates a location from the virtual TypeScript document back to the .pk document.
 *
 * Locations not in the virtual file are returned unchanged.
 *
 * @param document - The Piko document.
 * @param location - The location from the virtual document.
 * @param block - The script block info.
 * @returns The translated location in the .pk document.
 */
export function translateTsLocation(
    document: vscode.TextDocument,
    location: vscode.Location,
    block: BlockInfo
): vscode.Location {
    if (location.uri.scheme !== 'piko-virtual-ts') {
        return location;
    }

    const contentStartPos = document.positionAt(block.contentStartOffset);
    const lineOffset = contentStartPos.line;
    const charOffset = contentStartPos.character;

    const translate = (pos: vscode.Position): vscode.Position => {
        const adjustedLine = pos.line - TS_REFERENCE_LINE_OFFSET;
        if (adjustedLine === 0) {
            return new vscode.Position(lineOffset, pos.character + charOffset);
        }
        return new vscode.Position(adjustedLine + lineOffset, pos.character);
    };

    return new vscode.Location(
        document.uri,
        new vscode.Range(
            translate(location.range.start),
            translate(location.range.end)
        )
    );
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
    const virtualUri = createVirtualTsUri(document.uri);
    const positionInScript = calculateTsPositionInBlock(document, position, block);

    const locations = await vscode.commands.executeCommand<vscode.Location[]>(
        'vscode.executeDefinitionProvider',
        virtualUri,
        positionInScript
    );

    if (locations.length === 0) {
        return null;
    }
    return locations.map(loc => translateTsLocation(document, loc, block));
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
    const virtualUri = createVirtualTsUri(document.uri);
    const positionInScript = calculateTsPositionInBlock(document, position, block);

    const hovers = await vscode.commands.executeCommand<vscode.Hover[]>(
        'vscode.executeHoverProvider',
        virtualUri,
        positionInScript
    );

    if (hovers.length === 0) {
        return null;
    }

    const translatedHover = hovers[0];
    const hoverRange = translatedHover.range;
    if (hoverRange) {
        const start = translateTsLocation(document, new vscode.Location(virtualUri, hoverRange), block).range.start;
        const end = translateTsLocation(document, new vscode.Location(virtualUri, hoverRange), block).range.end;
        translatedHover.range = new vscode.Range(start, end);
    }

    return translatedHover;
}

/**
 * Handles Code Completion requests for TypeScript blocks.
 *
 * @param document - The Piko document.
 * @param position - The cursor position.
 * @param block - The TypeScript block info.
 * @param triggerCharacter - The character that triggered completion.
 * @returns The completion list, or null if not available.
 */
async function handleTsCompletion(
    document: vscode.TextDocument,
    position: vscode.Position,
    block: BlockInfo,
    triggerCharacter: string | undefined
): Promise<vscode.CompletionList | null> {
    const virtualUri = createVirtualTsUri(document.uri);
    const positionInScript = calculateTsPositionInBlock(document, position, block);

    return vscode.commands.executeCommand<vscode.CompletionList>(
        'vscode.executeCompletionItemProvider',
        virtualUri,
        positionInScript,
        triggerCharacter
    );
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
    const virtualUri = createVirtualTsUri(document.uri);
    const positionInScript = calculateTsPositionInBlock(document, position, block);

    return vscode.commands.executeCommand<vscode.SignatureHelp>(
        'vscode.executeSignatureHelpProvider',
        virtualUri,
        positionInScript,
        triggerCharacter
    );
}

/**
 * Creates the LSP middleware that proxies requests to the right handler.
 *
 * Template blocks are handled by the Piko LSP.
 * Script blocks are proxied to gopls via virtual Go documents.
 *
 * @returns The middleware for the language client.
 */
export function createPikoLspMiddleware(): Middleware {
    return {
        provideDefinition: async (document, position, token, next) => {
            if (document.languageId !== 'piko') {
                return null;
            }

            const block = detectBlockAtPosition(document, position);
            if (block.type === BlockType.Template) {
                return next(document, position, token);
            }
            if (block.type === BlockType.Script) {
                return handleScriptDefinition(document, position, block);
            }
            if (block.type === BlockType.ScriptTS) {
                return handleTsDefinition(document, position, block);
            }
            return null;
        },

        provideHover: async (document, position, token, next) => {
            if (document.languageId !== 'piko') {
                return null;
            }

            const block = detectBlockAtPosition(document, position);
            if (block.type === BlockType.Template) {
                return next(document, position, token);
            }
            if (block.type === BlockType.Script) {
                return handleScriptHover(document, position, block);
            }
            if (block.type === BlockType.ScriptTS) {
                return handleTsHover(document, position, block);
            }
            return null;
        },

        provideCompletionItem: async (document, position, context, token, next) => {
            if (document.languageId !== 'piko') {
                return null;
            }

            const block = detectBlockAtPosition(document, position);
            if (block.type === BlockType.Template) {
                return next(document, position, context, token);
            }
            if (block.type === BlockType.Script) {
                return handleScriptCompletion(document, position, block, context.triggerCharacter);
            }
            if (block.type === BlockType.ScriptTS) {
                return handleTsCompletion(document, position, block, context.triggerCharacter);
            }
            return null;
        },

        provideSignatureHelp: async (document, position, context, token, next) => {
            if (document.languageId !== 'piko') {
                return null;
            }

            const block = detectBlockAtPosition(document, position);
            if (block.type === BlockType.Template) {
                return next(document, position, context, token);
            }
            if (block.type === BlockType.Script) {
                return handleScriptSignatureHelp(document, position, block, context.triggerCharacter);
            }
            if (block.type === BlockType.ScriptTS) {
                return handleTsSignatureHelp(document, position, block, context.triggerCharacter);
            }
            return null;
        },

        provideCodeLenses: (document, token, next) => document.languageId === 'piko' ? next(document, token) : null,
        provideDocumentLinks: (document, token, next) => document.languageId === 'piko' ? next(document, token) : null,
        provideDocumentFormattingEdits: (document, options, token, next) => document.languageId === 'piko' ? next(document, options, token) : null,

        provideReferences: (document, position, context, token, next) => isInTemplateBlock(document, position) ? next(document, position, context, token) : null,
        provideDocumentHighlights: (document, position, token, next) => isInTemplateBlock(document, position) ? next(document, position, token) : null,
        prepareRename: (document, position, token, next) => isInTemplateBlock(document, position) ? next(document, position, token) : null,
        provideRenameEdits: (document, position, newName, token, next) => isInTemplateBlock(document, position) ? next(document, position, newName, token) : null,
        provideCodeActions: (document, range, context, token, next) => isInTemplateBlock(document, range.start) ? next(document, range, context, token) : null,
    };
}
