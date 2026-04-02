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

/**
 * The type of block in a Piko (.pk) file.
 */
export enum BlockType {
    /** The HTML template block. */
    Template = 'template',
    /** The Go script block (type="application/x-go"). */
    Script = 'script',
    /** The TypeScript script block (lang="ts"). */
    ScriptTS = 'script-ts',
    /** The CSS style block. */
    Style = 'style',
    /** The internationalisation block. */
    I18n = 'i18n',
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

/** Regex patterns for matching each block type in a Piko file. */
const BLOCK_PATTERNS: Array<{ regex: RegExp; type: BlockType }> = [
    {
        regex: /<script\s+lang\s*=\s*["'](ts|typescript)["'][^>]*>([\s\S]*?)<\/script>/gi,
        type: BlockType.ScriptTS
    },
    {
        regex: /<script\s+type\s*=\s*["'](application\/typescript|text\/typescript)["'][^>]*>([\s\S]*?)<\/script>/gi,
        type: BlockType.ScriptTS
    },
    {
        regex: /<script\s+type\s*=\s*["']application\/x-go["'][^>]*>([\s\S]*?)<\/script>/gi,
        type: BlockType.Script
    },
    {
        regex: /<template>([\s\S]*?)<\/template>/gi,
        type: BlockType.Template
    },
    {
        regex: /<style>([\s\S]*?)<\/style>/gi,
        type: BlockType.Style
    },
    {
        regex: /<i18n>([\s\S]*?)<\/i18n>/gi,
        type: BlockType.I18n
    }
];

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
 * Finds the block at a position in a Piko document.
 *
 * Uses regex to find template, script, style, and i18n blocks.
 *
 * @param document - The Piko document to search.
 * @param position - The cursor position to check.
 * @returns The block info, or Unknown if no block is found.
 */
export function detectBlockAtPosition(
    document: vscode.TextDocument,
    position: vscode.Position
): BlockInfo {
    const text = document.getText();
    const offset = document.offsetAt(position);

    for (const {regex, type} of BLOCK_PATTERNS) {
        regex.lastIndex = 0;

        let match: RegExpExecArray | null;
        while ((match = regex.exec(text)) !== null) {
            const blockStart = match.index;
            const blockEnd = blockStart + match[0].length;

            if (offset < blockStart || offset > blockEnd) {
                continue;
            }

            const openTagEnd = text.indexOf('>', blockStart);
            const contentStart = openTagEnd + 1;

            return {
                type,
                startLine: document.positionAt(blockStart).line,
                endLine: document.positionAt(blockEnd).line,
                contentStartLine: document.positionAt(contentStart).line,
                contentStartOffset: contentStart,
                contentEndOffset: text.lastIndexOf('</', blockEnd)
            };
        }
    }

    return UNKNOWN_BLOCK;
}

/**
 * Checks if a position is inside a template block.
 *
 * Only template blocks are handled by the Piko LSP.
 *
 * @param document - The Piko document.
 * @param position - The position to check.
 * @returns True if inside a template block, false if not.
 */
export function isInTemplateBlock(
    document: vscode.TextDocument,
    position: vscode.Position
): boolean {
    const blockInfo = detectBlockAtPosition(document, position);
    return blockInfo.type === BlockType.Template;
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
    const blockInfo = detectBlockAtPosition(document, position);
    return blockInfo.type === BlockType.ScriptTS;
}
