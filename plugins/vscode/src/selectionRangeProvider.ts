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
import {findLastTagIndex, parseSfcBlocks, scanMarkupTags, SfcBlock, VOID_ELEMENTS} from './sfcBlocks';

/**
 * The furthest the interpolation search looks either side of the cursor, in
 * UTF-16 code units.
 */
export const MAX_INTERPOLATION_SEARCH_LENGTH = 10_000;

/**
 * The deepest element nesting tracked while matching tags. Deeper elements
 * are not offered as selection ranges.
 */
export const MAX_TAG_DEPTH = 512;

/** The length of the `{{` and `}}` interpolation delimiters. */
const INTERPOLATION_DELIMITER_LENGTH = 2;

/**
 * Holds an element whose closing tag has not been seen yet.
 */
interface OpenElement {
    /** The lower-case tag name. */
    name: string;
    /** The offset of the opening tag's `<`. */
    start: number;
    /** The offset just after the opening tag's `>`. */
    end: number;
}

/**
 * Holds the offsets of an element that encloses the cursor.
 */
interface EnclosingElement {
    /** The offset of the opening tag's `<`. */
    openStart: number;
    /** The offset just after the opening tag's `>`. */
    openEnd: number;
    /** The offset of the closing tag's `<`. */
    closeStart: number;
    /** The offset just after the closing tag's `>`. */
    closeEnd: number;
}

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
        const text = document.getText();
        const blocks = parseSfcBlocks(text);
        return positions.map(position => this.buildSelectionRangeChain(document, text, blocks, position));
    }

    /**
     * Builds the chain of selection ranges for a position.
     *
     * @param document - The Piko document.
     * @param text - The document text.
     * @param blocks - The parsed top-level blocks.
     * @param position - The cursor position.
     * @returns The innermost selection range with parents linked.
     */
    private buildSelectionRangeChain(
        document: vscode.TextDocument,
        text: string,
        blocks: readonly SfcBlock[],
        position: vscode.Position
    ): vscode.SelectionRange {
        const ranges: vscode.Range[] = [];
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

        ranges.push(...this.findHtmlTagRanges(document, text, blocks, position));
        ranges.push(...this.findBlockRanges(document, blocks, position));

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
        const limit = Math.max(1, offset - MAX_INTERPOLATION_SEARCH_LENGTH);
        for (let i = Math.min(offset, text.length - 1); i >= limit; i--) {
            if (text[i] === '{' && text[i - 1] === '{') {
                return i - 1;
            }
            if (text[i] === '}' && text[i - 1] === '}') {
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
        const limit = Math.min(text.length - 1, offset + MAX_INTERPOLATION_SEARCH_LENGTH);
        for (let i = offset; i < limit; i++) {
            if (text[i] === '}' && text[i + 1] === '}') {
                return i + INTERPOLATION_DELIMITER_LENGTH;
            }
            if (text[i] === '{' && text[i + 1] === '{') {
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

        const contentStart = openIndex + INTERPOLATION_DELIMITER_LENGTH;
        const contentEnd = closeIndex - INTERPOLATION_DELIMITER_LENGTH;

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
     * Finds the ranges of the innermost HTML element that encloses the
     * position, when the position is inside template or timeline markup.
     *
     * @param document - The Piko document.
     * @param text - The document text.
     * @param blocks - The parsed top-level blocks.
     * @param position - The cursor position.
     * @returns The content range and full element range, if found.
     */
    private findHtmlTagRanges(
        document: vscode.TextDocument,
        text: string,
        blocks: readonly SfcBlock[],
        position: vscode.Position
    ): vscode.Range[] {
        const offset = document.offsetAt(position);
        const region = blocks.find(block =>
            (block.kind === 'template' || block.kind === 'timeline') &&
            offset >= block.contentStart && offset <= block.contentEnd);
        if (region === undefined) {
            return [];
        }

        const element = findEnclosingElement(text, region.contentStart, region.contentEnd, offset);
        if (element === undefined) {
            return [];
        }

        const ranges: vscode.Range[] = [];
        const contentStartLine = document.positionAt(element.openEnd).line;
        const contentEndLine = document.positionAt(element.closeStart).line;
        const contentRange = new vscode.Range(
            new vscode.Position(contentStartLine, 0),
            document.lineAt(contentEndLine).range.end
        );
        if (contentRange.contains(position)) {
            ranges.push(contentRange);
        }

        ranges.push(new vscode.Range(
            new vscode.Position(document.positionAt(element.openStart).line, 0),
            document.lineAt(document.positionAt(element.closeEnd).line).range.end
        ));
        return ranges;
    }

    /**
     * Finds the ranges of the top-level block containing the position.
     *
     * @param document - The Piko document.
     * @param blocks - The parsed top-level blocks.
     * @param position - The cursor position.
     * @returns The body range and full block range if found.
     */
    private findBlockRanges(
        document: vscode.TextDocument,
        blocks: readonly SfcBlock[],
        position: vscode.Position
    ): vscode.Range[] {
        for (const block of blocks) {
            const startLine = document.positionAt(block.start).line;
            const endLine = document.positionAt(block.end).line;
            if (position.line < startLine || position.line > endLine) {
                continue;
            }

            const ranges: vscode.Range[] = [];
            const openEndLine = document.positionAt(block.contentStart).line;
            const closeStartLine = document.positionAt(block.contentEnd).line;
            const bodyStartLine = openEndLine === startLine ? openEndLine + 1 : openEndLine;
            const bodyEndLine = closeStartLine === endLine ? closeStartLine - 1 : closeStartLine;
            if (bodyStartLine <= bodyEndLine) {
                ranges.push(new vscode.Range(
                    new vscode.Position(bodyStartLine, 0),
                    document.lineAt(bodyEndLine).range.end
                ));
            }

            ranges.push(new vscode.Range(
                new vscode.Position(startLine, 0),
                document.lineAt(endLine).range.end
            ));
            return ranges;
        }

        return [];
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

/**
 * Finds the innermost element that encloses an offset, in a single pass.
 *
 * @param text - The document text.
 * @param start - The offset where the markup region starts.
 * @param end - The offset where the markup region ends.
 * @param offset - The cursor offset.
 * @returns The enclosing element, or undefined when none encloses the offset.
 */
function findEnclosingElement(text: string, start: number, end: number, offset: number): EnclosingElement | undefined {
    const enclosing: OpenElement[] = [];
    const openedAfter: OpenElement[] = [];
    let result: EnclosingElement | undefined;

    scanMarkupTags(text, start, end, tag => {
        const isBeforeCursor = tag.kind === 'open' ? tag.start <= offset : tag.end <= offset;
        const stack = isBeforeCursor ? enclosing : openedAfter;
        if (tag.kind === 'open') {
            if (!tag.selfClosing && !VOID_ELEMENTS.has(tag.name) && stack.length < MAX_TAG_DEPTH) {
                stack.push({name: tag.name, start: tag.start, end: tag.end});
            }
            return true;
        }

        const stackIndex = findLastTagIndex(stack, tag.name);
        if (stackIndex !== -1) {
            stack.length = stackIndex;
            return true;
        }
        if (isBeforeCursor) {
            return true;
        }

        const enclosingIndex = findLastTagIndex(enclosing, tag.name);
        if (enclosingIndex === -1) {
            return true;
        }
        const element = enclosing[enclosingIndex];
        result = {openStart: element.start, openEnd: element.end, closeStart: tag.start, closeEnd: tag.end};
        return false;
    });

    return result;
}
