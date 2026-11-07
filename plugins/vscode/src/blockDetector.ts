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
import {findBlockAtOffset, parseSfcBlocks, SfcBlock} from './sfcBlocks';

/**
 * The type of block in a Piko (.pk or .pkc) file.
 */
export enum BlockType {
    /** The HTML template block. */
    Template = 'template',
    /** The Go script block (type="application/x-go", lang="go" and friends). */
    Script = 'script',
    /** The TypeScript script block (lang="ts" or type="application/typescript"). */
    ScriptTS = 'script-ts',
    /** Any other script block (JavaScript or an unrecognised type). */
    ScriptJS = 'script-js',
    /** The CSS style block. */
    Style = 'style',
    /** The internationalisation block. */
    I18n = 'i18n',
    /** The piko:timeline block. */
    Timeline = 'timeline',
    /** Fallback when no block is found at a position. */
    Unknown = 'unknown'
}

/**
 * Holds data about a block in a Piko file.
 */
export interface BlockInfo {
    /** The type of block. */
    type: BlockType;
    /** The line where the opening tag starts. */
    startLine: number;
    /** The line where the closing tag ends. */
    endLine: number;
    /** The line where the content starts (after the opening tag). */
    contentStartLine: number;
    /** The character offset where the content starts. */
    contentStartOffset: number;
    /** The character offset where the content ends. */
    contentEndOffset: number;
}

/** Default block info returned when no block is found at a position. */
const UNKNOWN_BLOCK: BlockInfo = {
    type: BlockType.Unknown,
    startLine: 0,
    endLine: 0,
    contentStartLine: 0,
    contentStartOffset: 0,
    contentEndOffset: 0
};

/**
 * Caches the parsed blocks of each document by version, so the many
 * per-request block lookups do not reparse an unchanged document.
 */
const blockCache = new WeakMap<object, { version: number; text: string; blocks: SfcBlock[] }>();

/**
 * Returns the parsed top-level blocks of a document.
 *
 * @param document - The Piko document.
 * @returns The parsed blocks in source order.
 */
export function getDocumentBlocks(document: vscode.TextDocument): SfcBlock[] {
    const text = document.getText();
    const version = typeof document.version === 'number' ? document.version : -1;
    const cached = blockCache.get(document);
    if (cached?.version === version && cached.text === text) {
        return cached.blocks;
    }
    const blocks = parseSfcBlocks(text);
    blockCache.set(document, {version, text, blocks});
    return blocks;
}

/**
 * Converts a parsed block into the position-based block info.
 *
 * @param document - The Piko document.
 * @param block - The parsed block.
 * @returns The block info.
 */
export function toBlockInfo(document: vscode.TextDocument, block: SfcBlock): BlockInfo {
    return {
        type: block.kind as BlockType,
        startLine: document.positionAt(block.start).line,
        endLine: document.positionAt(block.end).line,
        contentStartLine: document.positionAt(block.contentStart).line,
        contentStartOffset: block.contentStart,
        contentEndOffset: block.contentEnd
    };
}

/**
 * Finds the block at a position in a Piko document.
 *
 * Uses the shared SFC block parser, which mirrors the server's parser.
 *
 * @param document - The Piko document to search.
 * @param position - The cursor position to check.
 * @returns The block info, or Unknown if no block is found.
 */
export function detectBlockAtPosition(
    document: vscode.TextDocument,
    position: vscode.Position
): BlockInfo {
    const block = findBlockAtOffset(getDocumentBlocks(document), document.offsetAt(position));
    return block ? toBlockInfo(document, block) : UNKNOWN_BLOCK;
}

/**
 * Checks if a position is inside a template block.
 *
 * @param document - The Piko document.
 * @param position - The position to check.
 * @returns True if inside a template block, false if not.
 */
export function isInTemplateBlock(
    document: vscode.TextDocument,
    position: vscode.Position
): boolean {
    return detectBlockAtPosition(document, position).type === BlockType.Template;
}

/**
 * Checks if a position is inside a TypeScript script block.
 *
 * @param document - The Piko document.
 * @param position - The position to check.
 * @returns True if inside a TypeScript script block, false if not.
 */
export function isInTypeScriptBlock(
    document: vscode.TextDocument,
    position: vscode.Position
): boolean {
    return detectBlockAtPosition(document, position).type === BlockType.ScriptTS;
}

/**
 * Checks if a position is in a region served by pikopls: the template or the
 * Go script block.
 *
 * @param document - The Piko document.
 * @param position - The position to check.
 * @returns True when pikopls should handle requests at the position.
 */
export function isInServerBlock(
    document: vscode.TextDocument,
    position: vscode.Position
): boolean {
    const type = detectBlockAtPosition(document, position).type;
    return type === BlockType.Template || type === BlockType.Script;
}
