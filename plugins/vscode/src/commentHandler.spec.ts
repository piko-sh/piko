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

import {beforeEach, describe, expect, it, vi} from 'vitest';
import * as vscode from 'vscode';
import {BlockType} from './blockDetector';
import {
    EXPR_COMMENT_PREFIX,
    EXPR_COMMENT_SUFFIX,
    findAttributeNameBounds,
    findAttributeNameStart,
    findEqualSignBeforeQuote,
    findExistingComment,
    findQuoteAndEquals,
    hasValidClosingQuote,
    HTML_COMMENT_PREFIX,
    HTML_COMMENT_SUFFIX,
    isExpressionDirectiveAttr,
    isInsideExpressionDirective,
    isInsideInterpolation,
    isTagBoundary,
    isWhitespace,
    registerCommentCommands,
    resolveCommentStrategy,
} from './commentHandler';
import {createMockDocument} from './test/mockTextDocument';

const handlers = new Map<string, () => Promise<void>>();

vi.mock('vscode', () => ({
    Position: class {
        constructor(public line: number, public character: number) {
        }
    },
    Range: class {
        constructor(
            public start: { line: number; character: number },
            public end: { line: number; character: number }
        ) {
        }
    },
    Selection: class {
        isEmpty: boolean;

        constructor(
            public anchor: { line: number; character: number },
            public active: { line: number; character: number }
        ) {
            this.isEmpty = anchor.line === active.line && anchor.character === active.character;
        }
    },
    Uri: {
        file: (path: string) => ({fsPath: path}),
    },
    commands: {
        registerCommand: (name: string, handler: () => Promise<void>) => {
            handlers.set(name, handler);
            return {dispose: () => undefined};
        },
        executeCommand: vi.fn(),
    },
    window: {
        activeTextEditor: undefined,
    },
}));

describe('commentHandler', () => {
    describe('findExistingComment', () => {
        it('should find HTML comment at start of text', () => {
            const result = findExistingComment('<!-- comment -->');
            expect(result).toEqual({
                start: 0,
                end: 16,
                prefix: HTML_COMMENT_PREFIX,
                suffix: HTML_COMMENT_SUFFIX,
            });
        });

        it('should find HTML comment with leading whitespace', () => {
            const result = findExistingComment('  <!-- comment -->  ');
            expect(result).toEqual({
                start: 2,
                end: 18,
                prefix: HTML_COMMENT_PREFIX,
                suffix: HTML_COMMENT_SUFFIX,
            });
        });

        it('should find expression comment', () => {
            const result = findExistingComment('/* comment */');
            expect(result).toEqual({
                start: 0,
                end: 13,
                prefix: EXPR_COMMENT_PREFIX,
                suffix: EXPR_COMMENT_SUFFIX,
            });
        });

        it('should find expression comment with whitespace', () => {
            const result = findExistingComment('  /* expression */  ');
            expect(result).toEqual({
                start: 2,
                end: 18,
                prefix: EXPR_COMMENT_PREFIX,
                suffix: EXPR_COMMENT_SUFFIX,
            });
        });

        it('should return undefined for non-comment text', () => {
            expect(findExistingComment('plain text')).toBeUndefined();
            expect(findExistingComment('<div>content</div>')).toBeUndefined();
        });

        it('should return undefined for incomplete HTML comment', () => {
            expect(findExistingComment('<!-- no ending')).toBeUndefined();
            expect(findExistingComment('no start -->')).toBeUndefined();
        });

        it('should return undefined for incomplete expression comment', () => {
            expect(findExistingComment('/* no ending')).toBeUndefined();
            expect(findExistingComment('no start */')).toBeUndefined();
        });

        it('should handle empty string', () => {
            expect(findExistingComment('')).toBeUndefined();
        });

        it('should handle whitespace-only string', () => {
            expect(findExistingComment('   ')).toBeUndefined();
        });
    });

    describe('isInsideInterpolation', () => {
        it('should return true when inside {{ }}', () => {
            const text = 'Hello {{ name }} world';
            expect(isInsideInterpolation(text, 10)).toBe(true);
            expect(isInsideInterpolation(text, 9)).toBe(true);
            expect(isInsideInterpolation(text, 13)).toBe(true);
        });

        it('should return false when outside interpolation', () => {
            const text = 'Hello {{ name }} world';
            expect(isInsideInterpolation(text, 3)).toBe(false);
            expect(isInsideInterpolation(text, 18)).toBe(false);
        });

        it('should return false when at interpolation boundary', () => {
            const text = '{{ value }}';
            expect(isInsideInterpolation(text, 0)).toBe(false);
            expect(isInsideInterpolation(text, 1)).toBe(false);
        });

        it('should handle nested braces correctly', () => {
            const text = '{{ foo.bar }}';
            expect(isInsideInterpolation(text, 5)).toBe(true);
        });

        it('should return false for text without interpolation', () => {
            expect(isInsideInterpolation('plain text', 5)).toBe(false);
        });

        it('should handle multiple interpolations', () => {
            const text = '{{ a }} and {{ b }}';
            expect(isInsideInterpolation(text, 3)).toBe(true);
            expect(isInsideInterpolation(text, 10)).toBe(false);
            expect(isInsideInterpolation(text, 15)).toBe(true);
        });

        it('should return false after closing braces', () => {
            const text = '{{ done }} after';
            expect(isInsideInterpolation(text, 12)).toBe(false);
        });
    });

    describe('isWhitespace', () => {
        it('should return true for space', () => {
            expect(isWhitespace(' ')).toBe(true);
        });

        it('should return true for tab', () => {
            expect(isWhitespace('\t')).toBe(true);
        });

        it('should return true for newline', () => {
            expect(isWhitespace('\n')).toBe(true);
        });

        it('should return true for carriage return', () => {
            expect(isWhitespace('\r')).toBe(true);
        });

        it('should return false for letters', () => {
            expect(isWhitespace('a')).toBe(false);
            expect(isWhitespace('Z')).toBe(false);
        });

        it('should return false for special characters', () => {
            expect(isWhitespace('<')).toBe(false);
            expect(isWhitespace('=')).toBe(false);
            expect(isWhitespace('"')).toBe(false);
        });
    });

    describe('isTagBoundary', () => {
        it('should return true for >', () => {
            expect(isTagBoundary('>')).toBe(true);
        });

        it('should return true for <', () => {
            expect(isTagBoundary('<')).toBe(true);
        });

        it('should return false for other characters', () => {
            expect(isTagBoundary('/')).toBe(false);
            expect(isTagBoundary('=')).toBe(false);
            expect(isTagBoundary('"')).toBe(false);
            expect(isTagBoundary(' ')).toBe(false);
            expect(isTagBoundary('a')).toBe(false);
        });
    });

    describe('findEqualSignBeforeQuote', () => {
        it('should find = immediately before quote', () => {
            const text = 'class="value"';
            expect(findEqualSignBeforeQuote(text, 6)).toBe(5);
        });

        it('should find = with whitespace before quote', () => {
            const text = 'class = "value"';
            expect(findEqualSignBeforeQuote(text, 8)).toBe(6);
        });

        it('should return -1 if no = found', () => {
            const text = '"value"';
            expect(findEqualSignBeforeQuote(text, 0)).toBe(-1);
        });

        it('should return -1 if non-whitespace before =', () => {
            const text = 'other class="value"';
            expect(findEqualSignBeforeQuote(text, 6)).toBe(-1);
        });

        it('should handle multiple = in text', () => {
            const text = 'a="1" b="2"';
            expect(findEqualSignBeforeQuote(text, 8)).toBe(7);
        });
    });

    describe('hasValidClosingQuote', () => {
        it('should return true when closing quote exists before tag boundary', () => {
            const text = 'class="value">';
            expect(hasValidClosingQuote(text, '"', 7)).toBe(true);
        });

        it('should return false when no closing quote', () => {
            const text = 'class="value';
            expect(hasValidClosingQuote(text, '"', 7)).toBe(false);
        });

        it('should return false when closing quote is after tag boundary', () => {
            const text = 'class="val>ue"';
            expect(hasValidClosingQuote(text, '"', 7)).toBe(false);
        });

        it('should handle < as tag boundary', () => {
            const text = 'class="val<ue"';
            expect(hasValidClosingQuote(text, '"', 7)).toBe(false);
        });

        it('should work with single quotes', () => {
            const text = "class='value'>";
            expect(hasValidClosingQuote(text, "'", 7)).toBe(true);
        });

        it('should handle text without tag boundaries', () => {
            const text = 'standalone="value" more';
            expect(hasValidClosingQuote(text, '"', 12)).toBe(true);
        });
    });

    describe('findQuoteAndEquals', () => {
        it('should find quote and equals for double-quoted attribute', () => {
            const text = '<div class="test-value">';
            const result = findQuoteAndEquals(text, 15);
            expect(result).not.toBeNull();
            expect(result!.quoteStart).toBe(11);
            expect(result!.equalSign).toBe(10);
        });

        it('should find quote and equals for single-quoted attribute', () => {
            const text = "<div class='test-value'>";
            const result = findQuoteAndEquals(text, 15);
            expect(result).not.toBeNull();
            expect(result!.quoteStart).toBe(11);
        });

        it('should return null outside attribute value', () => {
            const text = '<div>content</div>';
            expect(findQuoteAndEquals(text, 7)).toBeNull();
        });

        it('should return null when quote is before tag boundary', () => {
            const text = '<div>"orphan"</div>';
            expect(findQuoteAndEquals(text, 8)).toBeNull();
        });

        it('should handle attribute without proper equals', () => {
            const text = '<div "orphan-value">';
            expect(findQuoteAndEquals(text, 10)).toBeNull();
        });

        it('should work with spaced equals', () => {
            const text = '<div class = "value">';
            const result = findQuoteAndEquals(text, 16);
            expect(result).not.toBeNull();
        });
    });

    describe('findAttributeNameBounds', () => {
        it('should find simple attribute name', () => {
            const text = '<div class="value">';
            const result = findAttributeNameBounds(text, 10);
            expect(result).not.toBeNull();
            expect(text.slice(result!.start, result!.end)).toBe('class');
        });

        it('should find bind directive name', () => {
            const text = '<div :class="value">';
            const result = findAttributeNameBounds(text, 11);
            expect(result).not.toBeNull();
            expect(text.slice(result!.start, result!.end)).toBe(':class');
        });

        it('should find event directive name', () => {
            const text = '<div @click="handler">';
            const result = findAttributeNameBounds(text, 11);
            expect(result).not.toBeNull();
            expect(text.slice(result!.start, result!.end)).toBe('@click');
        });

        it('should find piko directive name', () => {
            const text = '<div p-if="condition">';
            const result = findAttributeNameBounds(text, 9);
            expect(result).not.toBeNull();
            expect(text.slice(result!.start, result!.end)).toBe('p-if');
        });

        it('should return null at tag boundary', () => {
            const text = '<="value">';
            expect(findAttributeNameBounds(text, 1)).toBeNull();
        });

        it('should handle whitespace around equals', () => {
            const text = '<div class = "value">';
            const result = findAttributeNameBounds(text, 11);
            expect(result).not.toBeNull();
            expect(text.slice(result!.start, result!.end)).toBe('class');
        });
    });

    describe('findAttributeNameStart', () => {
        it('should find start after whitespace', () => {
            const text = '<div class="value">';
            expect(findAttributeNameStart(text, 10)).toBe(5);
        });

        it('should find start with colon prefix', () => {
            const text = '<div :class="value">';
            expect(findAttributeNameStart(text, 11)).toBe(5);
        });

        it('should find start with @ prefix', () => {
            const text = '<div @click="handler">';
            expect(findAttributeNameStart(text, 11)).toBe(5);
        });

        it('should stop at tag boundary', () => {
            const text = '<attr="value">';
            expect(findAttributeNameStart(text, 5)).toBe(1);
        });

        it('should return -1 for empty search', () => {
            const text = '';
            expect(findAttributeNameStart(text, 0)).toBe(-1);
        });
    });

    describe('isExpressionDirectiveAttr', () => {
        describe('bind syntax', () => {
            it('should detect :class as expression directive', () => {
                expect(isExpressionDirectiveAttr(':class')).toBe(true);
            });

            it('should detect :href as expression directive', () => {
                expect(isExpressionDirectiveAttr(':href')).toBe(true);
            });

            it('should detect :style as expression directive', () => {
                expect(isExpressionDirectiveAttr(':style')).toBe(true);
            });

            it('should detect :id as expression directive', () => {
                expect(isExpressionDirectiveAttr(':id')).toBe(true);
            });
        });

        describe('event syntax', () => {
            it('should detect @click as expression directive', () => {
                expect(isExpressionDirectiveAttr('@click')).toBe(true);
            });

            it('should detect @submit as expression directive', () => {
                expect(isExpressionDirectiveAttr('@submit')).toBe(true);
            });

            it('should detect @input as expression directive', () => {
                expect(isExpressionDirectiveAttr('@input')).toBe(true);
            });
        });

        describe('piko directives', () => {
            it('should detect p-if', () => {
                expect(isExpressionDirectiveAttr('p-if')).toBe(true);
            });

            it('should detect p-else-if', () => {
                expect(isExpressionDirectiveAttr('p-else-if')).toBe(true);
            });

            it('should detect p-else', () => {
                expect(isExpressionDirectiveAttr('p-else')).toBe(true);
            });

            it('should detect p-for', () => {
                expect(isExpressionDirectiveAttr('p-for')).toBe(true);
            });

            it('should detect p-show', () => {
                expect(isExpressionDirectiveAttr('p-show')).toBe(true);
            });

            it('should detect p-text', () => {
                expect(isExpressionDirectiveAttr('p-text')).toBe(true);
            });

            it('should detect p-html', () => {
                expect(isExpressionDirectiveAttr('p-html')).toBe(true);
            });

            it('should detect p-model', () => {
                expect(isExpressionDirectiveAttr('p-model')).toBe(true);
            });

            it('should detect p-class', () => {
                expect(isExpressionDirectiveAttr('p-class')).toBe(true);
            });

            it('should detect p-style', () => {
                expect(isExpressionDirectiveAttr('p-style')).toBe(true);
            });

            it('should detect p-bind', () => {
                expect(isExpressionDirectiveAttr('p-bind')).toBe(true);
            });

            it('should detect p-on', () => {
                expect(isExpressionDirectiveAttr('p-on')).toBe(true);
            });

            it('should detect p-event', () => {
                expect(isExpressionDirectiveAttr('p-event')).toBe(true);
            });

            it('should detect directive with modifier (p-on-click)', () => {
                expect(isExpressionDirectiveAttr('p-on-click')).toBe(true);
            });
        });

        describe('regular attributes', () => {
            it('should reject class', () => {
                expect(isExpressionDirectiveAttr('class')).toBe(false);
            });

            it('should reject href', () => {
                expect(isExpressionDirectiveAttr('href')).toBe(false);
            });

            it('should reject id', () => {
                expect(isExpressionDirectiveAttr('id')).toBe(false);
            });

            it('should reject style', () => {
                expect(isExpressionDirectiveAttr('style')).toBe(false);
            });

            it('should reject data attributes', () => {
                expect(isExpressionDirectiveAttr('data-id')).toBe(false);
            });

            it('should reject aria attributes', () => {
                expect(isExpressionDirectiveAttr('aria-label')).toBe(false);
            });
        });
    });

    describe('isInsideExpressionDirective', () => {
        it('should return true inside bind directive value', () => {
            const text = '<div :class="activeClass">';
            expect(isInsideExpressionDirective(text, 18)).toBe(true);
        });

        it('should return true inside event directive value', () => {
            const text = '<div @click="handleClick">';
            expect(isInsideExpressionDirective(text, 18)).toBe(true);
        });

        it('should return true inside p-if value', () => {
            const text = '<div p-if="isVisible">';
            expect(isInsideExpressionDirective(text, 15)).toBe(true);
        });

        it('should return true inside p-for value', () => {
            const text = '<div p-for="item in items">';
            expect(isInsideExpressionDirective(text, 18)).toBe(true);
        });

        it('should return false inside regular attribute value', () => {
            const text = '<div class="my-class">';
            expect(isInsideExpressionDirective(text, 14)).toBe(false);
        });

        it('should return false outside attribute values', () => {
            const text = '<div>content</div>';
            expect(isInsideExpressionDirective(text, 7)).toBe(false);
        });

        it('should return false for incomplete attributes', () => {
            const text = '<div :class=';
            expect(isInsideExpressionDirective(text, 11)).toBe(false);
        });

        it('should handle multiple attributes', () => {
            const text = '<div class="static" :dynamic="value">';
            expect(isInsideExpressionDirective(text, 14)).toBe(false);
            expect(isInsideExpressionDirective(text, 32)).toBe(true);
        });
    });
});

const SOURCE = `<template>
  <p>Hello</p>
</template>
<script type="application/x-go">
package main
</script>
<style>
.a {}
</style>
<i18n lang="json">
{}
</i18n>
<script lang="ts">
const a = 1;
</script>`;

/**
 * Builds a fake editor positioned on a line, recording edits.
 *
 * @param line - The cursor line.
 * @param character - The cursor character.
 * @param languageId - The document language.
 * @returns The editor and the recorded replacements.
 */
function makeEditor(line: number, character: number, languageId = 'piko') {
    const doc = createMockDocument(SOURCE, languageId);
    const lines = SOURCE.split('\n');
    const replacements: string[] = [];
    const position = {line, character};
    const editor = {
        document: {
            ...doc,
            lineAt: (index: number) => ({text: lines[index], range: {start: {line: index, character: 0}, end: {line: index, character: lines[index].length}}}),
        },
        selection: {active: position, anchor: position, isEmpty: true},
        edit: async (callback: (builder: { replace: (range: unknown, text: string) => void }) => void) => {
            callback({replace: (_range, text) => replacements.push(text)});
            return true;
        },
    };
    return {editor, replacements};
}

/**
 * Runs a registered comment command against an editor.
 *
 * @param name - The command name.
 * @param editor - The active editor.
 */
async function run(name: string, editor: unknown): Promise<void> {
    (vscode.window as { activeTextEditor: unknown }).activeTextEditor = editor;
    const handler = handlers.get(name);
    if (!handler) {
        throw new Error(`command ${name} not registered`);
    }
    await handler();
}

describe('resolveCommentStrategy', () => {
    it.each([
        [BlockType.Script, 'delegate'],
        [BlockType.ScriptTS, 'delegate'],
        [BlockType.ScriptJS, 'delegate'],
        [BlockType.Style, 'delegate'],
        [BlockType.I18n, 'delegate'],
        [BlockType.Template, 'markup'],
        [BlockType.Timeline, 'markup'],
        [BlockType.Unknown, 'markup'],
    ])('%s uses %s', (blockType, expected) => {
        expect(resolveCommentStrategy(blockType)).toBe(expected);
    });
});

describe('comment commands', () => {
    const executeCommand = vscode.commands.executeCommand as unknown as ReturnType<typeof vi.fn>;

    beforeEach(() => {
        executeCommand.mockReset();
        handlers.clear();
        registerCommentCommands({subscriptions: []} as never);
    });

    it.each([
        ['Go', 4],
        ['CSS', 7],
        ['i18n', 10],
        ['TypeScript', 13],
    ])('delegates line comments in %s blocks to VS Code', async (_name, line) => {
        const {editor, replacements} = makeEditor(line, 1);

        await run('piko.toggleLineComment', editor);

        expect(executeCommand).toHaveBeenCalledWith('editor.action.commentLine');
        expect(replacements).toEqual([]);
    });

    it('delegates block comments in script blocks', async () => {
        const {editor} = makeEditor(4, 1);

        await run('piko.toggleBlockComment', editor);

        expect(executeCommand).toHaveBeenCalledWith('editor.action.blockComment');
    });

    it('uses HTML comments in the template', async () => {
        const {editor, replacements} = makeEditor(1, 4);

        await run('piko.toggleLineComment', editor);

        expect(executeCommand).not.toHaveBeenCalled();
        expect(replacements).toEqual(['  <!-- <p>Hello</p> -->']);
    });

    it('delegates for other languages', async () => {
        const {editor} = makeEditor(1, 4, 'html');

        await run('piko.toggleBlockComment', editor);

        expect(executeCommand).toHaveBeenCalledWith('editor.action.blockComment');
    });

    it('does nothing without an editor', async () => {
        await run('piko.toggleLineComment', undefined);

        expect(executeCommand).not.toHaveBeenCalled();
    });
});
