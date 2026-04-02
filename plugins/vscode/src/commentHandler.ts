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
import {BlockType, detectBlockAtPosition} from './blockDetector';

/** The HTML comment opening delimiter. */
export const HTML_COMMENT_PREFIX = '<!--';
/** The HTML comment closing delimiter. */
export const HTML_COMMENT_SUFFIX = '-->';

/** The expression comment opening delimiter. */
export const EXPR_COMMENT_PREFIX = '/*';
/** The expression comment closing delimiter. */
export const EXPR_COMMENT_SUFFIX = '*/';

/**
 * Directives that contain Go/JS expressions and use expression comments.
 */
export const EXPRESSION_DIRECTIVES = new Set([
    'p-if', 'p-else-if', 'p-else', 'p-for', 'p-show',
    'p-text', 'p-html', 'p-model', 'p-class', 'p-style',
    'p-bind', 'p-on', 'p-event'
]);

/**
 * Registers the context-aware comment commands for Piko files.
 *
 * Overrides VSCode's default toggle comment to use the right syntax:
 * HTML comments for regular content, expression comments for interpolations.
 *
 * @param context - The extension context for managing subscriptions.
 */
export function registerCommentCommands(context: vscode.ExtensionContext): void {
    const toggleBlockCommentCommand = vscode.commands.registerCommand(
        'piko.toggleBlockComment',
        async () => {
            await toggleComment(true);
        }
    );

    const toggleLineCommentCommand = vscode.commands.registerCommand(
        'piko.toggleLineComment',
        async () => {
            await toggleComment(false);
        }
    );

    context.subscriptions.push(toggleBlockCommentCommand, toggleLineCommentCommand);
}

/**
 * Toggles a comment on the current selection or line.
 *
 * @param isBlockComment - Whether to use block comment style.
 */
async function toggleComment(isBlockComment: boolean): Promise<void> {
    const editor = vscode.window.activeTextEditor;
    if (!editor) {
        return;
    }

    const document = editor.document;
    if (document.languageId !== 'piko') {
        if (isBlockComment) {
            await vscode.commands.executeCommand('editor.action.blockComment');
        } else {
            await vscode.commands.executeCommand('editor.action.commentLine');
        }
        return;
    }

    const selection = editor.selection;
    const isExpression = isInExpressionContext(document, selection);

    const prefix = isExpression ? EXPR_COMMENT_PREFIX : HTML_COMMENT_PREFIX;
    const suffix = isExpression ? EXPR_COMMENT_SUFFIX : HTML_COMMENT_SUFFIX;

    await editor.edit(editBuilder => {
        if (selection.isEmpty) {
            const line = document.lineAt(selection.active.line);
            const lineText = line.text;

            const existingComment = findExistingComment(lineText);
            if (existingComment) {
                const uncommented = lineText.slice(0, existingComment.start) +
                    lineText.slice(existingComment.start + existingComment.prefix.length,
                        existingComment.end - existingComment.suffix.length) +
                    lineText.slice(existingComment.end);
                const range = new vscode.Range(line.range.start, line.range.end);
                editBuilder.replace(range, uncommented);
            } else {
                const trimmedStart = lineText.search(/\S/);
                if (trimmedStart >= 0) {
                    const trimmedEnd = lineText.trimEnd().length;
                    const newText = `${lineText.slice(0, trimmedStart)}${prefix} ${lineText.slice(trimmedStart, trimmedEnd)} ${suffix}${lineText.slice(trimmedEnd)}`;
                    const range = new vscode.Range(line.range.start, line.range.end);
                    editBuilder.replace(range, newText);
                }
            }
        } else {
            const selectedText = document.getText(selection);

            const existingComment = findExistingComment(selectedText);
            if (existingComment?.start === 0 &&
                existingComment.end === selectedText.length) {
                const uncommented = selectedText.slice(
                    existingComment.prefix.length,
                    selectedText.length - existingComment.suffix.length
                ).trim();
                editBuilder.replace(selection, uncommented);
            } else {
                editBuilder.replace(selection, `${prefix} ${selectedText} ${suffix}`);
            }
        }
    });
}

/**
 * Holds data about an existing comment found in text.
 */
export interface ExistingComment {
    /** Start offset of the comment. */
    start: number;
    /** End offset of the comment (exclusive). */
    end: number;
    /** The comment prefix used. */
    prefix: string;
    /** The comment suffix used. */
    suffix: string;
}

/**
 * Finds an existing comment in the given text.
 *
 * @param text - The text to search for comments.
 * @returns The comment info if found, or undefined.
 */
export function findExistingComment(text: string): ExistingComment | undefined {
    const trimmed = text.trim();

    if (trimmed.startsWith(HTML_COMMENT_PREFIX) && trimmed.endsWith(HTML_COMMENT_SUFFIX)) {
        const start = text.indexOf(HTML_COMMENT_PREFIX);
        const end = text.lastIndexOf(HTML_COMMENT_SUFFIX) + HTML_COMMENT_SUFFIX.length;
        return {
            start,
            end,
            prefix: HTML_COMMENT_PREFIX,
            suffix: HTML_COMMENT_SUFFIX
        };
    }

    if (trimmed.startsWith(EXPR_COMMENT_PREFIX) && trimmed.endsWith(EXPR_COMMENT_SUFFIX)) {
        const start = text.indexOf(EXPR_COMMENT_PREFIX);
        const end = text.lastIndexOf(EXPR_COMMENT_SUFFIX) + EXPR_COMMENT_SUFFIX.length;
        return {
            start,
            end,
            prefix: EXPR_COMMENT_PREFIX,
            suffix: EXPR_COMMENT_SUFFIX
        };
    }

    return undefined;
}

/**
 * Checks if the current selection is in an expression context.
 *
 * Expression contexts include interpolations and expression directive values.
 *
 * @param document - The document to check.
 * @param selection - The current selection.
 * @returns True if in expression context, false if not.
 */
function isInExpressionContext(
    document: vscode.TextDocument,
    selection: vscode.Selection
): boolean {
    const position = selection.active;
    const blockInfo = detectBlockAtPosition(document, position);

    if (blockInfo.type !== BlockType.Template) {
        return false;
    }

    const text = document.getText();
    const offset = document.offsetAt(position);

    if (isInsideInterpolation(text, offset)) {
        return true;
    }

    return isInsideExpressionDirective(text, offset);
}

/**
 * Checks if the offset is inside an interpolation {{ }}.
 *
 * @param text - The full document text.
 * @param offset - The character offset to check.
 * @returns True if inside interpolation, false if not.
 */
export function isInsideInterpolation(text: string, offset: number): boolean {
    let depth = 0;

    for (let i = offset - 1; i >= 0; i--) {
        if (i > 0 && text[i - 1] === '{' && text[i] === '{') {
            if (depth === 0) {
                return true;
            }
            depth--;
            i--;
        } else if (i > 0 && text[i - 1] === '}' && text[i] === '}') {
            depth++;
            i--;
        }
    }

    return false;
}

/**
 * Holds the result of finding an attribute value quote.
 */
export interface QuoteSearchResult {
    /** The index of the opening quote character. */
    quoteStart: number;
    /** The index of the equals sign before the quote. */
    equalSign: number;
}

/**
 * Checks if a character is whitespace.
 *
 * @param char - The character to check.
 * @returns True if whitespace, false if not.
 */
export function isWhitespace(char: string): boolean {
    return char === ' ' || char === '\t' || char === '\n' || char === '\r';
}

/**
 * Checks if a character is a tag boundary (< or >).
 *
 * @param char - The character to check.
 * @returns True if tag boundary, false if not.
 */
export function isTagBoundary(char: string): boolean {
    return char === '>' || char === '<';
}

/**
 * Finds the equals sign before a quote position.
 *
 * @param text - The text to search.
 * @param quoteIndex - The index of the quote.
 * @returns The index of the equals sign, or -1 if not found.
 */
export function findEqualSignBeforeQuote(text: string, quoteIndex: number): number {
    for (let j = quoteIndex - 1; j >= 0; j--) {
        const c = text[j];
        if (c === '=') {
            return j;
        }
        if (!isWhitespace(c)) {
            break;
        }
    }
    return -1;
}

/**
 * Checks if a closing quote exists before any tag boundary.
 *
 * @param text - The text to search.
 * @param quoteChar - The quote character to find.
 * @param offset - The offset to start searching from.
 * @returns True if valid closing quote exists, false if not.
 */
export function hasValidClosingQuote(text: string, quoteChar: string, offset: number): boolean {
    const closingQuote = text.indexOf(quoteChar, offset);
    if (closingQuote === -1) {
        return false;
    }

    const nextGt = text.indexOf('>', offset);
    const nextLt = text.indexOf('<', offset);
    const nextTagBoundary = Math.min(
        nextGt === -1 ? Infinity : nextGt,
        nextLt === -1 ? Infinity : nextLt
    );

    return nextTagBoundary === Infinity || closingQuote < nextTagBoundary;
}

/**
 * Searches backwards for the attribute value start quote and equals sign.
 *
 * @param text - The text to search.
 * @param offset - The offset to start from.
 * @returns The quote and equals positions, or null if not found.
 */
export function findQuoteAndEquals(text: string, offset: number): QuoteSearchResult | null {
    for (let i = offset - 1; i >= 0; i--) {
        const char = text[i];

        if (isTagBoundary(char)) {
            break;
        }

        if (char !== '"' && char !== "'") {
            continue;
        }

        if (!hasValidClosingQuote(text, char, offset)) {
            continue;
        }

        const equalSign = findEqualSignBeforeQuote(text, i);
        if (equalSign !== -1) {
            return {quoteStart: i, equalSign};
        }
    }
    return null;
}

/**
 * Finds the attribute name boundaries before the equals sign.
 *
 * @param text - The text to search.
 * @param equalSign - The index of the equals sign.
 * @returns The start and end of the attribute name, or null if not found.
 */
export function findAttributeNameBounds(
    text: string,
    equalSign: number
): { start: number; end: number } | null {
    let attrNameEnd = -1;
    let attrNameStart = -1;

    for (let i = equalSign - 1; i >= 0; i--) {
        const char = text[i];

        if (isWhitespace(char)) {
            if (attrNameEnd !== -1) {
                attrNameStart = i + 1;
                break;
            }
            continue;
        }

        if (isTagBoundary(char)) {
            break;
        }

        if (attrNameEnd === -1) {
            attrNameEnd = i + 1;
        }

        if (char === ':' || char === '@') {
            attrNameStart = i;
            break;
        }
    }

    if (attrNameStart === -1 && attrNameEnd !== -1) {
        attrNameStart = findAttributeNameStart(text, attrNameEnd);
    }

    if (attrNameStart === -1 || attrNameEnd === -1) {
        return null;
    }

    return {start: attrNameStart, end: attrNameEnd};
}

/**
 * Finds the start of an attribute name by scanning backwards.
 *
 * @param text - The text to search.
 * @param attrNameEnd - The end index of the attribute name.
 * @returns The start index, or -1 if not found.
 */
export function findAttributeNameStart(text: string, attrNameEnd: number): number {
    for (let i = attrNameEnd - 1; i >= 0; i--) {
        const char = text[i];

        if (isWhitespace(char) || isTagBoundary(char)) {
            return i + 1;
        }

        if (char === ':' || char === '@') {
            return i;
        }
    }
    return -1;
}

/**
 * Checks if an attribute name is an expression directive.
 *
 * @param attrName - The attribute name to check.
 * @returns True if expression directive, false if not.
 */
export function isExpressionDirectiveAttr(attrName: string): boolean {
    if (attrName.startsWith(':') || attrName.startsWith('@')) {
        return true;
    }

    for (const directive of EXPRESSION_DIRECTIVES) {
        if (attrName === directive || attrName.startsWith(`${directive}-`)) {
            return true;
        }
    }

    return false;
}

/**
 * Checks if the offset is inside an expression directive attribute value.
 *
 * Expression directives include bind (:attr), event (@event), and
 * directive syntax (p-if, p-for, etc.).
 *
 * @param text - The full document text.
 * @param offset - The character offset to check.
 * @returns True if inside expression directive, false if not.
 */
export function isInsideExpressionDirective(text: string, offset: number): boolean {
    const quoteResult = findQuoteAndEquals(text, offset);
    if (!quoteResult) {
        return false;
    }

    const nameBounds = findAttributeNameBounds(text, quoteResult.equalSign);
    if (!nameBounds) {
        return false;
    }

    const attrName = text.slice(nameBounds.start, nameBounds.end);
    return isExpressionDirectiveAttr(attrName);
}
