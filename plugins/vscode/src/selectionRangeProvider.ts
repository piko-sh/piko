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
 * Holds data about a block in a Piko file.
 */
interface BlockInfo {
    /** The type of block. */
    type: 'template' | 'script' | 'style' | 'i18n';
    /** The line where the opening tag starts. */
    startLine: number;
    /** The line where the closing tag ends. */
    endLine: number;
    /** The line where the block body content starts. */
    bodyStartLine: number;
    /** The line where the block body content ends. */
    bodyEndLine: number;
}

/**
 * Holds data about an HTML tag and its position.
 */
interface TagInfo {
    /** The tag name in lowercase. */
    name: string;
    /** The line where the opening tag starts. */
    openStartLine: number;
    /** The line where the opening tag ends. */
    openEndLine: number;
    /** The line where the closing tag starts. */
    closeStartLine: number;
    /** The line where the closing tag ends. */
    closeEndLine: number;
    /** The line where the tag content starts. */
    contentStartLine: number;
    /** The line where the tag content ends. */
    contentEndLine: number;
}

/**
 * Holds data about an opening tag candidate.
 */
interface TagCandidate {
    /** The tag name in lowercase. */
    name: string;
    /** The start offset of the opening tag. */
    start: number;
    /** The end offset of the opening tag. */
    end: number;
}

/**
 * Holds the regex patterns for a block type.
 */
interface BlockPattern {
    /** The block type. */
    type: 'template' | 'script' | 'style' | 'i18n';
    /** The regex pattern for the opening tag. */
    open: RegExp;
    /** The regex pattern for the closing tag. */
    close: RegExp;
}

/**
 * HTML tags that are self-closing and cannot contain content.
 */
const SELF_CLOSING_TAGS = new Set([
    'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input',
    'link', 'meta', 'param', 'source', 'track', 'wbr'
]);

/**
 * Regex patterns for finding top-level Piko blocks.
 */
const BLOCK_PATTERNS: BlockPattern[] = [
    {type: 'template', open: /<template[^>]*>/gi, close: /<\/template>/gi},
    {type: 'script', open: /<script[^>]*>/gi, close: /<\/script>/gi},
    {type: 'style', open: /<style[^>]*>/gi, close: /<\/style>/gi},
    {type: 'i18n', open: /<i18n[^>]*>/gi, close: /<\/i18n>/gi}
];

/**
 * Provides selection ranges for Piko template files.
 *
 * Builds a chain of ranges from innermost to outermost:
 * word, interpolation, attribute, tag content, full tag, block body, full block.
 */
export class PikoSelectionRangeProvider implements vscode.SelectionRangeProvider {

    /**
     * Gets selection ranges for the given positions.
     *
     * @param document - The Piko document.
     * @param positions - The positions to get ranges for.
     * @param _token - The cancellation token (unused).
     * @returns The selection ranges for each position.
     */
    provideSelectionRanges(
        document: vscode.TextDocument,
        positions: vscode.Position[],
        _token: vscode.CancellationToken
    ): vscode.ProviderResult<vscode.SelectionRange[]> {
        return positions.map(position => this.buildSelectionRangeChain(document, position));
    }

    /**
     * Builds the chain of selection ranges for a position.
     *
     * @param document - The Piko document.
     * @param position - The cursor position.
     * @returns The innermost selection range with parents linked.
     */
    private buildSelectionRangeChain(
        document: vscode.TextDocument,
        position: vscode.Position
    ): vscode.SelectionRange {
        const ranges: vscode.Range[] = [];
        const text = document.getText();
        const offset = document.offsetAt(position);

        const wordRange = document.getWordRangeAtPosition(position);
        if (wordRange && !wordRange.isEmpty) {
            ranges.push(wordRange);
        }

        const interpRange = this.findInterpolationRange(text, offset, document);
        if (interpRange) {
            ranges.push(interpRange);
        }

        const attrRange = this.findAttributeValueRange(offset, document);
        if (attrRange) {
            ranges.push(attrRange);
        }

        const tagRanges = this.findHtmlTagRanges(document, position);
        ranges.push(...tagRanges);

        const blockRanges = this.findBlockRanges(document, position);
        ranges.push(...blockRanges);

        return this.buildChain(this.deduplicateAndSort(ranges, document));
    }

    /**
     * Finds the opening {{ before the offset.
     *
     * @param text - The document text.
     * @param offset - The cursor offset.
     * @returns The index of the first { or -1 if not found.
     */
    private findOpeningBraces(text: string, offset: number): number {
        for (let i = offset; i >= 1; i--) {
            if (text[i] === '{' && text[i - 1] === '{') {
                return i - 1;
            }
            if (text[i] === '}' && i > 0 && text[i - 1] === '}') {
                return -1;
            }
        }
        return -1;
    }

    /**
     * Finds the closing }} after the offset.
     *
     * @param text - The document text.
     * @param offset - The cursor offset.
     * @returns The index after the second } or -1 if not found.
     */
    private findClosingBraces(text: string, offset: number): number {
        for (let i = offset; i < text.length - 1; i++) {
            if (text[i] === '}' && text[i + 1] === '}') {
                return i + 2;
            }
            if (text[i] === '{' && i < text.length - 1 && text[i + 1] === '{') {
                return -1;
            }
        }
        return -1;
    }

    /**
     * Finds the interpolation range containing the offset.
     *
     * @param text - The document text.
     * @param offset - The cursor offset.
     * @param document - The Piko document.
     * @returns The range of the interpolation content, or null if not found.
     */
    private findInterpolationRange(
        text: string,
        offset: number,
        document: vscode.TextDocument
    ): vscode.Range | null {
        const openIndex = this.findOpeningBraces(text, offset);
        if (openIndex === -1) {
            return null;
        }

        const closeIndex = this.findClosingBraces(text, offset);
        if (closeIndex === -1) {
            return null;
        }

        const contentStart = openIndex + 2;
        const contentEnd = closeIndex - 2;

        if (contentEnd <= contentStart) {
            return null;
        }

        return new vscode.Range(
            document.positionAt(contentStart),
            document.positionAt(contentEnd)
        );
    }

    /**
     * Finds the attribute value range containing the offset.
     *
     * Looks for quoted attribute values on the current line.
     *
     * @param offset - The cursor offset.
     * @param document - The Piko document.
     * @returns The range of the attribute value, or null if not found.
     */
    private findAttributeValueRange(
        offset: number,
        document: vscode.TextDocument
    ): vscode.Range | null {
        const line = document.lineAt(document.positionAt(offset).line);
        const lineText = line.text;
        const lineOffset = offset - document.offsetAt(line.range.start);

        const openQuote = this.findOpeningQuote(lineText, lineOffset);
        if (!openQuote) {
            return null;
        }

        const closeQuoteIndex = this.findClosingQuote(lineText, lineOffset, openQuote.char);
        if (closeQuoteIndex === -1) {
            return null;
        }

        const lineStart = document.offsetAt(line.range.start);
        return new vscode.Range(
            document.positionAt(lineStart + openQuote.index + 1),
            document.positionAt(lineStart + closeQuoteIndex)
        );
    }

    /**
     * Finds the opening quote before a position in a line.
     *
     * @param lineText - The line text.
     * @param lineOffset - The offset within the line.
     * @returns The quote character and index, or null if not found.
     */
    private findOpeningQuote(lineText: string, lineOffset: number): { char: string; index: number } | null {
        for (let i = lineOffset - 1; i >= 0; i--) {
            const char = lineText[i];
            if (char === '"' || char === "'") {
                return {char, index: i};
            }
            if (char === '>' || char === '<') {
                return null;
            }
        }
        return null;
    }

    /**
     * Finds the closing quote after a position in a line.
     *
     * @param lineText - The line text.
     * @param lineOffset - The offset within the line.
     * @param quoteChar - The quote character to find.
     * @returns The index of the closing quote, or -1 if not found.
     */
    private findClosingQuote(lineText: string, lineOffset: number, quoteChar: string): number {
        for (let i = lineOffset; i < lineText.length; i++) {
            if (lineText[i] === quoteChar) {
                return i;
            }
            if (lineText[i] === '>' || lineText[i] === '<') {
                return -1;
            }
        }
        return -1;
    }

    /**
     * Finds HTML tag ranges containing the position.
     *
     * @param document - The Piko document.
     * @param position - The cursor position.
     * @returns The content range and full tag range if found.
     */
    private findHtmlTagRanges(
        document: vscode.TextDocument,
        position: vscode.Position
    ): vscode.Range[] {
        const ranges: vscode.Range[] = [];
        const text = document.getText();
        const offset = document.offsetAt(position);

        const tagInfo = this.findEnclosingTag(text, offset, document);
        if (!tagInfo) {
            return ranges;
        }

        if (tagInfo.contentStartLine <= tagInfo.contentEndLine) {
            const contentRange = new vscode.Range(
                new vscode.Position(tagInfo.contentStartLine, 0),
                document.lineAt(tagInfo.contentEndLine).range.end
            );
            if (contentRange.contains(position)) {
                ranges.push(contentRange);
            }
        }

        const fullRange = new vscode.Range(
            new vscode.Position(tagInfo.openStartLine, 0),
            document.lineAt(tagInfo.closeEndLine).range.end
        );
        ranges.push(fullRange);

        return ranges;
    }

    /**
     * Finds the enclosing HTML tag for a position.
     *
     * @param text - The document text.
     * @param offset - The cursor offset.
     * @param document - The Piko document.
     * @returns The tag info, or null if no enclosing tag is found.
     */
    private findEnclosingTag(
        text: string,
        offset: number,
        document: vscode.TextDocument
    ): TagInfo | null {
        const candidates = this.collectTagCandidates(text, offset);

        for (let i = candidates.length - 1; i >= 0; i--) {
            const tagInfo = this.tryMatchCandidate(text, offset, candidates[i], document);
            if (tagInfo) {
                return tagInfo;
            }
        }

        return null;
    }

    /**
     * Collects opening tag candidates before the offset.
     *
     * @param text - The document text.
     * @param offset - The cursor offset.
     * @returns The list of tag candidates.
     */
    private collectTagCandidates(text: string, offset: number): TagCandidate[] {
        const openTagPattern = /<([a-zA-Z][a-zA-Z0-9-]*)[^>]*>/g;
        const candidates: TagCandidate[] = [];
        let match: RegExpExecArray | null;

        while ((match = openTagPattern.exec(text)) !== null) {
            if (match.index > offset) {
                break;
            }

            const tagName = match[1].toLowerCase();
            if (SELF_CLOSING_TAGS.has(tagName)) {
                continue;
            }
            if (match[0].endsWith('/>')) {
                continue;
            }

            candidates.push({
                name: tagName,
                start: match.index,
                end: match.index + match[0].length
            });
        }

        return candidates;
    }

    /**
     * Tries to match a candidate tag with its closing tag.
     *
     * @param text - The document text.
     * @param offset - The cursor offset.
     * @param candidate - The tag candidate.
     * @param document - The Piko document.
     * @returns The tag info if the candidate encloses the offset, or null.
     */
    private tryMatchCandidate(
        text: string,
        offset: number,
        candidate: TagCandidate,
        document: vscode.TextDocument
    ): TagInfo | null {
        const closeTag = this.findMatchingCloseTag(text, candidate.end, candidate.name);
        if (!closeTag || closeTag.end <= offset) {
            return null;
        }

        const openStartPos = document.positionAt(candidate.start);
        const openEndPos = document.positionAt(candidate.end);
        const closeStartPos = document.positionAt(closeTag.start);
        const closeEndPos = document.positionAt(closeTag.end);

        return {
            name: candidate.name,
            openStartLine: openStartPos.line,
            openEndLine: openEndPos.line,
            closeStartLine: closeStartPos.line,
            closeEndLine: closeEndPos.line,
            contentStartLine: openEndPos.line,
            contentEndLine: closeStartPos.line
        };
    }

    /**
     * Finds the matching close tag for an opening tag.
     *
     * @param text - The document text.
     * @param startOffset - Where to start searching.
     * @param tagName - The tag name to find.
     * @returns The close tag position, or null if not found.
     */
    private findMatchingCloseTag(
        text: string,
        startOffset: number,
        tagName: string
    ): { start: number; end: number } | null {
        const openPattern = new RegExp(`<${tagName}(?:\\s|>)`, 'gi');
        const closePattern = new RegExp(`</${tagName}>`, 'gi');

        let depth = 1;
        let searchOffset = startOffset;

        while (depth > 0 && searchOffset < text.length) {
            openPattern.lastIndex = searchOffset;
            closePattern.lastIndex = searchOffset;

            const nextOpen = openPattern.exec(text);
            const nextClose = closePattern.exec(text);

            const step = this.processTagMatch(text, nextOpen, nextClose, depth);
            if (!step) {
                return null;
            }
            if ('result' in step) {
                return step.result;
            }

            depth = step.depth;
            searchOffset = step.searchOffset;
        }

        return null;
    }

    /**
     * Processes a single step in the close tag search.
     *
     * @param text - The document text.
     * @param nextOpen - The next opening tag match.
     * @param nextClose - The next closing tag match.
     * @param depth - The current nesting depth.
     * @returns The close tag position if found, or updated search state.
     */
    private processTagMatch(
        text: string,
        nextOpen: RegExpExecArray | null,
        nextClose: RegExpExecArray | null,
        depth: number
    ): { result: { start: number; end: number } } | { depth: number; searchOffset: number } | null {
        if (!nextClose) {
            return null;
        }

        if (nextOpen && nextOpen.index < nextClose.index) {
            const tagEnd = text.indexOf('>', nextOpen.index);
            const isSelfClosed = tagEnd !== -1 && text[tagEnd - 1] === '/';
            return {
                depth: isSelfClosed ? depth : depth + 1,
                searchOffset: nextOpen.index + 1
            };
        }

        const newDepth = depth - 1;
        if (newDepth === 0) {
            return {result: {start: nextClose.index, end: nextClose.index + nextClose[0].length}};
        }

        return {depth: newDepth, searchOffset: nextClose.index + 1};
    }

    /**
     * Finds block ranges containing the position.
     *
     * @param document - The Piko document.
     * @param position - The cursor position.
     * @returns The body range and full block range if found.
     */
    private findBlockRanges(
        document: vscode.TextDocument,
        position: vscode.Position
    ): vscode.Range[] {
        const ranges: vscode.Range[] = [];
        const blocks = this.parseBlocks(document);

        for (const block of blocks) {
            if (position.line < block.startLine || position.line > block.endLine) {
                continue;
            }

            if (block.bodyStartLine <= block.bodyEndLine) {
                const bodyRange = new vscode.Range(
                    new vscode.Position(block.bodyStartLine, 0),
                    document.lineAt(block.bodyEndLine).range.end
                );
                ranges.push(bodyRange);
            }

            const fullRange = new vscode.Range(
                new vscode.Position(block.startLine, 0),
                document.lineAt(block.endLine).range.end
            );
            ranges.push(fullRange);

            break;
        }

        return ranges;
    }

    /**
     * Parses the document to find all top-level blocks.
     *
     * @param document - The Piko document.
     * @returns The list of blocks found.
     */
    private parseBlocks(document: vscode.TextDocument): BlockInfo[] {
        const text = document.getText();
        const blocks: BlockInfo[] = [];

        for (const pattern of BLOCK_PATTERNS) {
            const patternBlocks = this.parseBlocksOfType(text, document, pattern);
            blocks.push(...patternBlocks);
        }

        return blocks.sort((a, b) => a.startLine - b.startLine);
    }

    /**
     * Parses all blocks of a single type from the document.
     *
     * @param text - The document text.
     * @param document - The Piko document.
     * @param pattern - The block pattern to match.
     * @returns The list of blocks found.
     */
    private parseBlocksOfType(
        text: string,
        document: vscode.TextDocument,
        pattern: BlockPattern
    ): BlockInfo[] {
        const blocks: BlockInfo[] = [];
        pattern.open.lastIndex = 0;

        let openMatch: RegExpExecArray | null;
        while ((openMatch = pattern.open.exec(text)) !== null) {
            const block = this.parseBlockMatch(text, document, pattern, openMatch);
            if (block) {
                blocks.push(block);
            }
        }

        return blocks;
    }

    /**
     * Parses a single block from an opening tag match.
     *
     * @param text - The document text.
     * @param document - The Piko document.
     * @param pattern - The block pattern.
     * @param openMatch - The opening tag match.
     * @returns The block info, or null if no closing tag found.
     */
    private parseBlockMatch(
        text: string,
        document: vscode.TextDocument,
        pattern: BlockPattern,
        openMatch: RegExpExecArray
    ): BlockInfo | null {
        pattern.close.lastIndex = openMatch.index + openMatch[0].length;
        const closeMatch = pattern.close.exec(text);
        if (!closeMatch) {
            return null;
        }

        const startPos = document.positionAt(openMatch.index);
        const openEndPos = document.positionAt(openMatch.index + openMatch[0].length);
        const closeStartPos = document.positionAt(closeMatch.index);
        const endPos = document.positionAt(closeMatch.index + closeMatch[0].length);

        return {
            type: pattern.type,
            startLine: startPos.line,
            endLine: endPos.line,
            bodyStartLine: openEndPos.line === startPos.line ? openEndPos.line + 1 : openEndPos.line,
            bodyEndLine: closeStartPos.line === endPos.line ? closeStartPos.line - 1 : closeStartPos.line
        };
    }

    /**
     * Removes duplicate ranges and sorts from smallest to largest.
     *
     * @param ranges - The ranges to process.
     * @param document - The Piko document.
     * @returns The unique ranges sorted by size.
     */
    private deduplicateAndSort(ranges: vscode.Range[], document: vscode.TextDocument): vscode.Range[] {
        const unique = new Map<string, vscode.Range>();

        for (const range of ranges) {
            const key = `${range.start.line}:${range.start.character}-${range.end.line}:${range.end.character}`;
            if (!unique.has(key)) {
                unique.set(key, range);
            }
        }

        return Array.from(unique.values()).sort((a, b) => {
            const aSize = document.offsetAt(a.end) - document.offsetAt(a.start);
            const bSize = document.offsetAt(b.end) - document.offsetAt(b.start);
            return aSize - bSize;
        });
    }

    /**
     * Builds a SelectionRange chain from an array of ranges.
     *
     * The first range becomes the innermost, each next range becomes the parent.
     *
     * @param ranges - The ranges sorted from smallest to largest.
     * @returns The innermost selection range with parents linked.
     */
    private buildChain(ranges: vscode.Range[]): vscode.SelectionRange {
        if (ranges.length === 0) {
            return new vscode.SelectionRange(new vscode.Range(0, 0, 0, 0));
        }

        let current = new vscode.SelectionRange(ranges[ranges.length - 1]);

        for (let i = ranges.length - 2; i >= 0; i--) {
            current = new vscode.SelectionRange(ranges[i], current);
        }

        return current;
    }
}

/**
 * Registers the selection range provider for Piko files.
 *
 * @param _context - The extension context (unused).
 * @returns The disposable registration.
 */
export function registerSelectionRangeProvider(
    _context: vscode.ExtensionContext
): vscode.Disposable {
    return vscode.languages.registerSelectionRangeProvider(
        {language: 'piko'},
        new PikoSelectionRangeProvider()
    );
}
