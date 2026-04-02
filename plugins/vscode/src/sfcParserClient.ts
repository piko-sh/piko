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

/**
 * A lightweight parser for extracting blocks from Piko (.pk) files.
 *
 * Uses simple regex patterns for fast parsing. This is enough for the LSP
 * to find template, script, style, and i18n blocks.
 */

/**
 * Holds the data for a single block in a Piko file.
 */
export interface SfcBlock {
    /** The type of block. */
    type: 'template' | 'script' | 'script-ts' | 'style' | 'i18n';
    /** The text content inside the block tags. */
    content: string;
    /** The start offset of the block in the file. */
    start: number;
    /** The end offset of the block in the file. */
    end: number;
}

/**
 * Parses a Piko file and extracts all blocks.
 *
 * Finds template, script (Go and TypeScript), style, and i18n blocks using regex.
 * Go script blocks use type="application/x-go".
 * TypeScript script blocks use lang="ts" or type="application/typescript".
 *
 * @param text - The full text of the Piko file.
 * @returns The list of blocks found in the file.
 */
export function parseSfc(text: string): SfcBlock[] {
    const blocks: SfcBlock[] = [];

    const scriptGoRegex = /<script\s+type\s*=\s*["']application\/x-go["'][^>]*>([\s\S]*?)<\/script>/gi;
    const scriptTsLangRegex = /<script\s+lang\s*=\s*["'](ts|typescript)["'][^>]*>([\s\S]*?)<\/script>/gi;
    const scriptTsTypeRegex = /<script\s+type\s*=\s*["'](application\/typescript|text\/typescript)["'][^>]*>([\s\S]*?)<\/script>/gi;
    const templateRegex = /<template>([\s\S]*?)<\/template>/gi;
    const styleRegex = /<style[^>]*>([\s\S]*?)<\/style>/gi;
    const i18nRegex = /<i18n[^>]*>([\s\S]*?)<\/i18n>/gi;

    let match;

    while ((match = scriptGoRegex.exec(text)) !== null) {
        blocks.push({
            type: 'script',
            content: match[1],
            start: match.index,
            end: match.index + match[0].length,
        });
    }

    while ((match = scriptTsLangRegex.exec(text)) !== null) {
        blocks.push({
            type: 'script-ts',
            content: match[2],
            start: match.index,
            end: match.index + match[0].length,
        });
    }

    while ((match = scriptTsTypeRegex.exec(text)) !== null) {
        blocks.push({
            type: 'script-ts',
            content: match[2],
            start: match.index,
            end: match.index + match[0].length,
        });
    }

    while ((match = templateRegex.exec(text)) !== null) {
        blocks.push({
            type: 'template',
            content: match[1],
            start: match.index,
            end: match.index + match[0].length,
        });
    }

    while ((match = styleRegex.exec(text)) !== null) {
        blocks.push({
            type: 'style',
            content: match[1],
            start: match.index,
            end: match.index + match[0].length,
        });
    }

    while ((match = i18nRegex.exec(text)) !== null) {
        blocks.push({
            type: 'i18n',
            content: match[1],
            start: match.index,
            end: match.index + match[0].length,
        });
    }

    return blocks;
}
