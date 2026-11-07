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
 * The most folding ranges returned for one document. VS Code shows at most
 * 5000 by default, so more than this is wasted work.
 */
export const MAX_FOLDING_RANGES = 10_000;

/**
 * The deepest element nesting tracked while matching tags. Deeper elements
 * are not folded.
 */
export const MAX_TAG_DEPTH = 512;

/** The HTML comment opener. */
const COMMENT_OPEN = '<!--';

/** The HTML comment closer. */
const COMMENT_CLOSE = '-->';

/** The interpolation opener. */
const INTERPOLATION_OPEN = '{{';

/** The interpolation closer. */
const INTERPOLATION_CLOSE = '}}';

/**
 * Holds a markup region (template or timeline content) to fold within.
 */
interface MarkupRegion {
    /** The offset where the region starts. */
    start: number;
    /** The offset where the region ends. */
    end: number;
}

/**
 * Holds an element whose closing tag has not been seen yet.
 */
interface OpenElement {
    /** The lower-case tag name. */
    name: string;
    /** The offset of the opening tag's `<`. */
    start: number;
}

/**
 * Collects folding ranges for one document, up to `MAX_FOLDING_RANGES`.
 */
class FoldingRangeCollector {
    /** The ranges collected so far. */
    readonly ranges: vscode.FoldingRange[] = [];

    /**
     * Creates a collector for a document.
     *
     * @param document - The Piko document.
     * @param token - The cancellation token for the request.
     */
    constructor(
        private readonly document: vscode.TextDocument,
        private readonly token: vscode.CancellationToken
    ) {
    }

    /**
     * Reports whether no more ranges should be collected.
     *
     * @returns True when the range limit is reached or the request was cancelled.
     */
    isDone(): boolean {
        return this.ranges.length >= MAX_FOLDING_RANGES || this.token.isCancellationRequested === true;
    }

    /**
     * Adds a folding range when it spans more than one line.
     *
     * @param startOffset - The start offset.
     * @param endOffset - The end offset.
     * @param kind - The optional folding range kind.
     */
    add(startOffset: number, endOffset: number, kind?: vscode.FoldingRangeKind): void {
        if (this.isDone()) {
            return;
        }
        const startLine = this.document.positionAt(startOffset).line;
        const endLine = this.document.positionAt(endOffset).line;
        if (endLine > startLine) {
            this.ranges.push(kind === undefined
                ? new vscode.FoldingRange(startLine, endLine)
                : new vscode.FoldingRange(startLine, endLine, kind));
        }
    }
}

/**
 * Provides code folding ranges for Piko files.
 *
 * Folds top-level blocks, HTML comments, interpolations, and nested HTML tags
 * (including `<piko:*>` elements and nested `<template>` elements). Markup
 * folding is limited to template and timeline content so that `<` characters
 * in Go, TypeScript or CSS never produce bogus ranges.
 *
 * Every scan runs in linear time, so very large documents stay responsive.
 *
 * Registered only as a fallback while pikopls is not running; the server
 * provides folding ranges itself.
 */
export class PikoFoldingRangeProvider implements vscode.FoldingRangeProvider {

    /**
     * Gets all folding ranges for a Piko document.
     *
     * @param document - The Piko document.
     * @param _context - The folding context (unused).
     * @param token - The cancellation token.
     * @returns The list of folding ranges.
     */
    provideFoldingRanges(
        document: vscode.TextDocument,
        _context: vscode.FoldingContext,
        token: vscode.CancellationToken
    ): vscode.FoldingRange[] {
        const collector = new FoldingRangeCollector(document, token);
        const text = document.getText();
        const blocks = parseSfcBlocks(text);
        const markupRegions: MarkupRegion[] = blocks
            .filter(block => block.kind === 'template' || block.kind === 'timeline')
            .map(block => ({start: block.contentStart, end: block.contentEnd}));

        for (const block of blocks) {
            collector.add(block.start, block.end, vscode.FoldingRangeKind.Region);
        }
        this.addCommentRanges(text, blocks, collector);
        for (const region of markupRegions) {
            this.addInterpolationRanges(text, region, collector);
            this.addHtmlTagRanges(text, region, collector);
        }

        return collector.ranges;
    }

    /**
     * Adds folding ranges for multi-line HTML comments outside raw-text
     * (script and style) block content.
     *
     * @param text - The document text.
     * @param blocks - The parsed blocks, in source order.
     * @param collector - The range collector.
     */
    private addCommentRanges(text: string, blocks: SfcBlock[], collector: FoldingRangeCollector): void {
        const rawBlocks = blocks.filter(block => block.tag === 'script' || block.tag === 'style');
        let rawIndex = 0;
        let index = 0;

        while (!collector.isDone()) {
            const open = text.indexOf(COMMENT_OPEN, index);
            if (open === -1) {
                return;
            }
            while (rawIndex < rawBlocks.length && rawBlocks[rawIndex].contentEnd <= open) {
                rawIndex++;
            }
            const rawBlock = rawIndex < rawBlocks.length ? rawBlocks[rawIndex] : undefined;
            if (rawBlock !== undefined && open > rawBlock.contentStart) {
                index = rawBlock.contentEnd;
                continue;
            }
            const close = text.indexOf(COMMENT_CLOSE, open + COMMENT_OPEN.length);
            if (close === -1) {
                return;
            }
            index = close + COMMENT_CLOSE.length;
            collector.add(open, index, vscode.FoldingRangeKind.Comment);
        }
    }

    /**
     * Adds folding ranges for multi-line template interpolations.
     *
     * @param text - The document text.
     * @param region - The markup region to search.
     * @param collector - The range collector.
     */
    private addInterpolationRanges(text: string, region: MarkupRegion, collector: FoldingRangeCollector): void {
        let index = region.start;
        while (!collector.isDone()) {
            const open = text.indexOf(INTERPOLATION_OPEN, index);
            if (open === -1 || open >= region.end) {
                return;
            }
            const close = text.indexOf(INTERPOLATION_CLOSE, open + INTERPOLATION_OPEN.length);
            if (close === -1) {
                return;
            }
            index = close + INTERPOLATION_CLOSE.length;
            collector.add(open, index);
        }
    }

    /**
     * Adds folding ranges for nested HTML elements within a markup region.
     *
     * Tags are matched with a stack in a single pass. A closing tag closes the
     * nearest open element with the same name; elements left open in between
     * are treated as unclosed.
     *
     * @param text - The document text.
     * @param region - The markup region to search.
     * @param collector - The range collector.
     */
    private addHtmlTagRanges(text: string, region: MarkupRegion, collector: FoldingRangeCollector): void {
        const stack: OpenElement[] = [];

        scanMarkupTags(text, region.start, region.end, tag => {
            if (collector.isDone()) {
                return false;
            }
            if (tag.kind === 'open') {
                if (!tag.selfClosing && !VOID_ELEMENTS.has(tag.name) && stack.length < MAX_TAG_DEPTH) {
                    stack.push({name: tag.name, start: tag.start});
                }
                return true;
            }
            const openIndex = findLastTagIndex(stack, tag.name);
            if (openIndex !== -1) {
                collector.add(stack[openIndex].start, tag.end);
                stack.length = openIndex;
            }
            return true;
        });
    }
}
