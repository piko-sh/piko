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
 * HTML tags that are self-closing and cannot be folded.
 */
const SELF_CLOSING_TAGS = new Set([
    'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input',
    'link', 'meta', 'param', 'source', 'track', 'wbr'
]);

/**
 * Top-level Piko blocks that are handled separately.
 */
const TOP_LEVEL_BLOCKS = new Set(['template', 'script', 'style', 'i18n']);

/**
 * Regex patterns for finding top-level blocks.
 */
const BLOCK_PATTERNS = [
    /<template>([\s\S]*?)<\/template>/gi,
    /<script\s+[^>]*>([\s\S]*?)<\/script>/gi,
    /<style>([\s\S]*?)<\/style>/gi,
    /<i18n[^>]*>([\s\S]*?)<\/i18n>/gi
];

/**
 * Provides code folding ranges for Piko files.
 *
 * Folds top-level blocks, HTML comments, interpolations, and nested HTML tags.
 */
export class PikoFoldingRangeProvider implements vscode.FoldingRangeProvider {

    /**
     * Gets all folding ranges for a Piko document.
     *
     * @param document - The Piko document.
     * @param _context - The folding context (unused).
     * @param _token - The cancellation token (unused).
     * @returns The list of folding ranges.
     */
    provideFoldingRanges(
        document: vscode.TextDocument,
        _context: vscode.FoldingContext,
        _token: vscode.CancellationToken
    ): vscode.FoldingRange[] {
        const ranges: vscode.FoldingRange[] = [];
        const text = document.getText();

        this.addBlockRanges(document, text, ranges);
        this.addCommentRanges(document, text, ranges);
        this.addInterpolationRanges(document, text, ranges);
        this.addHtmlTagRanges(document, text, ranges);

        return ranges;
    }

    /**
     * Adds folding ranges for top-level Piko blocks.
     *
     * @param document - The Piko document.
     * @param text - The document text.
     * @param ranges - The list to add ranges to.
     */
    private addBlockRanges(
        document: vscode.TextDocument,
        text: string,
        ranges: vscode.FoldingRange[]
    ): void {
        for (const regex of BLOCK_PATTERNS) {
            regex.lastIndex = 0;
            let match;
            while ((match = regex.exec(text)) !== null) {
                const startLine = document.positionAt(match.index).line;
                const endLine = document.positionAt(match.index + match[0].length).line;
                if (endLine > startLine) {
                    ranges.push(new vscode.FoldingRange(
                        startLine,
                        endLine,
                        vscode.FoldingRangeKind.Region
                    ));
                }
            }
        }
    }

    /**
     * Adds folding ranges for multi-line HTML comments.
     *
     * @param document - The Piko document.
     * @param text - The document text.
     * @param ranges - The list to add ranges to.
     */
    private addCommentRanges(
        document: vscode.TextDocument,
        text: string,
        ranges: vscode.FoldingRange[]
    ): void {
        const commentRegex = /<!--[\s\S]*?-->/g;
        let match;
        while ((match = commentRegex.exec(text)) !== null) {
            const startLine = document.positionAt(match.index).line;
            const endLine = document.positionAt(match.index + match[0].length).line;
            if (endLine > startLine) {
                ranges.push(new vscode.FoldingRange(
                    startLine,
                    endLine,
                    vscode.FoldingRangeKind.Comment
                ));
            }
        }
    }

    /**
     * Adds folding ranges for multi-line template interpolations.
     *
     * @param document - The Piko document.
     * @param text - The document text.
     * @param ranges - The list to add ranges to.
     */
    private addInterpolationRanges(
        document: vscode.TextDocument,
        text: string,
        ranges: vscode.FoldingRange[]
    ): void {
        const interpRegex = /\{\{[\s\S]*?\}\}/g;
        let match;
        while ((match = interpRegex.exec(text)) !== null) {
            const startLine = document.positionAt(match.index).line;
            const endLine = document.positionAt(match.index + match[0].length).line;
            if (endLine > startLine) {
                ranges.push(new vscode.FoldingRange(startLine, endLine));
            }
        }
    }

    /**
     * Adds folding ranges for nested HTML tags.
     *
     * @param document - The Piko document.
     * @param text - The document text.
     * @param ranges - The list to add ranges to.
     */
    private addHtmlTagRanges(
        document: vscode.TextDocument,
        text: string,
        ranges: vscode.FoldingRange[]
    ): void {
        const tagOpenRegex = /<([a-zA-Z][a-zA-Z0-9-]*)\b[^>]*(?<!\/)>/g;

        let match;
        while ((match = tagOpenRegex.exec(text)) !== null) {
            const tagName = match[1].toLowerCase();

            if (SELF_CLOSING_TAGS.has(tagName) || TOP_LEVEL_BLOCKS.has(tagName)) {
                continue;
            }

            if (tagName.startsWith('piko:')) {
                continue;
            }

            const closeTagEnd = this.findMatchingCloseTag(text, match.index + match[0].length, tagName);
            if (closeTagEnd !== -1) {
                const startLine = document.positionAt(match.index).line;
                const endLine = document.positionAt(closeTagEnd).line;
                if (endLine > startLine) {
                    ranges.push(new vscode.FoldingRange(startLine, endLine));
                }
            }
        }
    }

    /**
     * Finds the matching close tag for an open tag.
     *
     * Uses depth tracking to handle nested tags with the same name.
     *
     * @param text - The document text.
     * @param startOffset - Where to start searching (after the open tag).
     * @param tagName - The tag name to find.
     * @returns The offset after the close tag, or -1 if not found.
     */
    private findMatchingCloseTag(text: string, startOffset: number, tagName: string): number {
        let depth = 1;
        const openPattern = new RegExp(`<${tagName}\\b[^>]*(?<!/)>`, 'gi');
        const closePattern = new RegExp(`</${tagName}\\s*>`, 'gi');

        openPattern.lastIndex = startOffset;
        closePattern.lastIndex = startOffset;

        while (depth > 0) {
            const nextOpen = openPattern.exec(text);
            const nextClose = closePattern.exec(text);

            if (!nextClose) {
                return -1;
            }

            if (nextOpen && nextOpen.index < nextClose.index) {
                depth++;
                openPattern.lastIndex = nextOpen.index + nextOpen[0].length;
                closePattern.lastIndex = nextOpen.index + nextOpen[0].length;
            } else {
                depth--;
                if (depth === 0) {
                    return nextClose.index + nextClose[0].length;
                }
                openPattern.lastIndex = nextClose.index + nextClose[0].length;
                closePattern.lastIndex = nextClose.index + nextClose[0].length;
            }
        }

        return -1;
    }
}
