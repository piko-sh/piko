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

import {describe, expect, it, vi} from 'vitest';
import {BlockType, detectBlockAtPosition, isInTemplateBlock, isInTypeScriptBlock} from './blockDetector';
import {createMockDocument, createPosition} from './test/mockTextDocument';

vi.mock('vscode', () => ({
    Position: class {
        constructor(public line: number, public character: number) {
        }
    },
    Uri: {
        file: (path: string) => ({fsPath: path}),
    },
}));

describe('blockDetector', () => {
    describe('detectBlockAtPosition', () => {
        it('should detect template block', () => {
            const doc = createMockDocument('<template><div>Hello</div></template>');
            const position = createPosition(0, 15);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Template);
        });

        it('should detect script block with application/x-go type', () => {
            const doc = createMockDocument('<script type="application/x-go">package main</script>');
            const position = createPosition(0, 35);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Script);
        });

        it('should detect TypeScript block with lang="ts"', () => {
            const doc = createMockDocument('<script lang="ts">const x: number = 1;</script>');
            const position = createPosition(0, 25);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.ScriptTS);
        });

        it('should detect TypeScript block with lang="typescript"', () => {
            const doc = createMockDocument('<script lang="typescript">const x: number = 1;</script>');
            const position = createPosition(0, 30);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.ScriptTS);
        });

        it('should detect TypeScript block with type="application/typescript"', () => {
            const doc = createMockDocument('<script type="application/typescript">const x: number = 1;</script>');
            const position = createPosition(0, 42);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.ScriptTS);
        });

        it('should detect TypeScript block with type="text/typescript"', () => {
            const doc = createMockDocument('<script type="text/typescript">const x: number = 1;</script>');
            const position = createPosition(0, 35);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.ScriptTS);
        });

        it('should detect style block', () => {
            const doc = createMockDocument('<style>.foo { color: red; }</style>');
            const position = createPosition(0, 10);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Style);
        });

        it('should detect i18n block', () => {
            const doc = createMockDocument('<i18n>{"key": "value"}</i18n>');
            const position = createPosition(0, 10);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.I18n);
        });

        it('should return Unknown for positions outside any block', () => {
            const doc = createMockDocument('Some text before\n<template>content</template>');
            const position = createPosition(0, 5);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Unknown);
        });

        it('should track correct line numbers', () => {
            const doc = createMockDocument(`<template>
  <div>content</div>
</template>`);
            const position = createPosition(1, 5);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Template);
            expect(result.startLine).toBe(0);
            expect(result.endLine).toBe(2);
        });

        it('should track content start offset correctly', () => {
            const doc = createMockDocument('<template>content</template>');
            const position = createPosition(0, 12);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.contentStartOffset).toBe(10);
        });

        it('should track content end offset correctly', () => {
            const doc = createMockDocument('<template>content</template>');
            const position = createPosition(0, 12);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.contentEndOffset).toBe(17);
        });

        it('should handle multiple blocks and detect correct one', () => {
            const doc = createMockDocument(`<script type="application/x-go">package main</script>
<template><div>Hello</div></template>
<style>.foo {}</style>`);

            const templatePos = createPosition(1, 15);
            const templateResult = detectBlockAtPosition(doc as never, templatePos as never);
            expect(templateResult.type).toBe(BlockType.Template);

            const scriptPos = createPosition(0, 35);
            const scriptResult = detectBlockAtPosition(doc as never, scriptPos as never);
            expect(scriptResult.type).toBe(BlockType.Script);

            const stylePos = createPosition(2, 10);
            const styleResult = detectBlockAtPosition(doc as never, stylePos as never);
            expect(styleResult.type).toBe(BlockType.Style);
        });

        it('should handle script with single quotes', () => {
            const doc = createMockDocument("<script type='application/x-go'>package main</script>");
            const position = createPosition(0, 35);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Script);
        });

        it('should handle script with extra attributes', () => {
            const doc = createMockDocument('<script type="application/x-go" lang="go">package main</script>');
            const position = createPosition(0, 45);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Script);
        });

        it('should handle position at the very start of block', () => {
            const doc = createMockDocument('<template>content</template>');
            const position = createPosition(0, 0);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Template);
        });

        it('should handle position at the very end of block', () => {
            const doc = createMockDocument('<template>content</template>');
            const position = createPosition(0, 27);

            const result = detectBlockAtPosition(doc as never, position as never);

            expect(result.type).toBe(BlockType.Template);
        });
    });

    describe('isInTemplateBlock', () => {
        it('should return true when position is in template block', () => {
            const doc = createMockDocument('<template>content</template>');
            const position = createPosition(0, 12);

            const result = isInTemplateBlock(doc as never, position as never);

            expect(result).toBe(true);
        });

        it('should return false when position is in script block', () => {
            const doc = createMockDocument('<script type="application/x-go">code</script>');
            const position = createPosition(0, 33);

            const result = isInTemplateBlock(doc as never, position as never);

            expect(result).toBe(false);
        });

        it('should return false when position is outside any block', () => {
            const doc = createMockDocument('text before <template>content</template>');
            const position = createPosition(0, 5);

            const result = isInTemplateBlock(doc as never, position as never);

            expect(result).toBe(false);
        });
    });

    describe('isInTypeScriptBlock', () => {
        it('should return true when position is in TypeScript block with lang="ts"', () => {
            const doc = createMockDocument('<script lang="ts">const x = 1;</script>');
            const position = createPosition(0, 22);

            const result = isInTypeScriptBlock(doc as never, position as never);

            expect(result).toBe(true);
        });

        it('should return true when position is in TypeScript block with lang="typescript"', () => {
            const doc = createMockDocument('<script lang="typescript">const x = 1;</script>');
            const position = createPosition(0, 30);

            const result = isInTypeScriptBlock(doc as never, position as never);

            expect(result).toBe(true);
        });

        it('should return false when position is in Go script block', () => {
            const doc = createMockDocument('<script type="application/x-go">package main</script>');
            const position = createPosition(0, 35);

            const result = isInTypeScriptBlock(doc as never, position as never);

            expect(result).toBe(false);
        });

        it('should return false when position is in template block', () => {
            const doc = createMockDocument('<template>content</template>');
            const position = createPosition(0, 12);

            const result = isInTypeScriptBlock(doc as never, position as never);

            expect(result).toBe(false);
        });

        it('should return false when position is outside any block', () => {
            const doc = createMockDocument('text before <script lang="ts">code</script>');
            const position = createPosition(0, 5);

            const result = isInTypeScriptBlock(doc as never, position as never);

            expect(result).toBe(false);
        });
    });

    describe('real-world Piko file', () => {
        const pikoFile = `<script type="application/x-go">
package components

import "fmt"

func Render() string {
    return fmt.Sprintf("Hello, %s!", "World")
}
</script>

<template>
  <div class="container">
    <h1>{{ .Title }}</h1>
  </div>
</template>

<style>
.container {
    max-width: 800px;
}
</style>`;

        it('should correctly detect all block types', () => {
            const doc = createMockDocument(pikoFile);

            const scriptPos = createPosition(5, 10);
            expect(detectBlockAtPosition(doc as never, scriptPos as never).type).toBe(BlockType.Script);

            const templatePos = createPosition(12, 10);
            expect(detectBlockAtPosition(doc as never, templatePos as never).type).toBe(BlockType.Template);

            const stylePos = createPosition(18, 5);
            expect(detectBlockAtPosition(doc as never, stylePos as never).type).toBe(BlockType.Style);

            const betweenPos = createPosition(9, 0);
            expect(detectBlockAtPosition(doc as never, betweenPos as never).type).toBe(BlockType.Unknown);
        });
    });
});
