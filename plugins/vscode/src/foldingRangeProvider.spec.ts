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
import {MAX_FOLDING_RANGES, MAX_TAG_DEPTH, PikoFoldingRangeProvider} from './foldingRangeProvider';
import {createMockDocument} from './test/mockTextDocument';

vi.mock('vscode', () => ({
    Position: class {
        constructor(public line: number, public character: number) {
        }
    },
    FoldingRange: class {
        constructor(
            public start: number,
            public end: number,
            public kind?: number
        ) {
        }
    },
    FoldingRangeKind: {
        Comment: 1,
        Imports: 2,
        Region: 3
    },
    Uri: {
        file: (path: string) => ({fsPath: path}),
    },
}));

describe('PikoFoldingRangeProvider', () => {
    const provider = new PikoFoldingRangeProvider();

    describe('provideFoldingRanges', () => {
        it('should fold multi-line template blocks', () => {
            const doc = createMockDocument(`<template>
  <div>content</div>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges.length).toBeGreaterThanOrEqual(1);
            const templateRange = ranges.find(r => r.start === 0 && r.end === 2);
            expect(templateRange).toBeDefined();
            expect(templateRange?.kind).toBe(3);
        });

        it('should fold multi-line script blocks', () => {
            const doc = createMockDocument(`<script type="application/x-go">
package main

func Render() {}
</script>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const scriptRange = ranges.find(r => r.start === 0 && r.end === 4);
            expect(scriptRange).toBeDefined();
        });

        it('should fold multi-line style blocks', () => {
            const doc = createMockDocument(`<style>
.foo {
  color: red;
}
</style>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const styleRange = ranges.find(r => r.start === 0 && r.end === 4);
            expect(styleRange).toBeDefined();
        });

        it('should fold multi-line i18n blocks', () => {
            const doc = createMockDocument(`<i18n lang="json">
{
  "en": {}
}
</i18n>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const i18nRange = ranges.find(r => r.start === 0 && r.end === 4);
            expect(i18nRange).toBeDefined();
        });

        it('should not fold single-line blocks', () => {
            const doc = createMockDocument(`<template><div>content</div></template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const singleLineRanges = ranges.filter(r => r.start === 0 && r.end === 0);
            expect(singleLineRanges).toHaveLength(0);
        });

        it('should fold multi-line HTML comments', () => {
            const doc = createMockDocument(`<template>
  <!--
    This is a
    multi-line comment
  -->
  <div>content</div>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const commentRange = ranges.find(r => r.kind === 1 && r.start === 1 && r.end === 4);
            expect(commentRange).toBeDefined();
        });

        it('should not fold single-line HTML comments', () => {
            const doc = createMockDocument(`<template>
  <!-- single line comment -->
  <div>content</div>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const singleLineComment = ranges.filter(r => r.kind === 1 && r.start === r.end);
            expect(singleLineComment).toHaveLength(0);
        });

        it('should fold multi-line interpolations', () => {
            const doc = createMockDocument(`<template>
  <div>{{
    someValue
  }}</div>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const interpRange = ranges.find(r => r.start === 1 && r.end === 3);
            expect(interpRange).toBeDefined();
        });

        it('should fold nested HTML tags with depth tracking', () => {
            const doc = createMockDocument(`<template>
  <div>
    <span>content</span>
  </div>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const divRange = ranges.find(r => r.start === 1 && r.end === 3);
            expect(divRange).toBeDefined();
        });

        it('should handle deeply nested tags correctly', () => {
            const doc = createMockDocument(`<template>
  <div>
    <div>
      <div>
        content
      </div>
    </div>
  </div>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const divRanges = ranges.filter(r =>
                (r.start === 1 && r.end === 7) ||
                (r.start === 2 && r.end === 6) ||
                (r.start === 3 && r.end === 5)
            );
            expect(divRanges.length).toBe(3);
        });

        it('should skip self-closing tags', () => {
            const doc = createMockDocument(`<template>
  <br>
  <img src="test.png">
  <input type="text">
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const templateRange = ranges.find(r => r.start === 0 && r.end === 4);
            expect(templateRange).toBeDefined();
            const selfClosingRanges = ranges.filter(r => r.start >= 1 && r.start <= 3);
            expect(selfClosingRanges).toHaveLength(0);
        });

        it('should handle empty document', () => {
            const doc = createMockDocument('');

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges).toHaveLength(0);
        });

        it('should handle document with multiple blocks', () => {
            const doc = createMockDocument(`<template>
  <div>content</div>
</template>

<script type="application/x-go">
package main
</script>

<style>
.foo { color: red; }
</style>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges.filter(r => r.kind === 3).length).toBeGreaterThanOrEqual(3);
        });

        it('should handle same tag name at different nesting levels', () => {
            const doc = createMockDocument(`<template>
  <ul>
    <li>
      <ul>
        <li>nested</li>
      </ul>
    </li>
  </ul>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            const outerUl = ranges.find(r => r.start === 1 && r.end === 7);
            expect(outerUl).toBeDefined();

            const innerUl = ranges.find(r => r.start === 3 && r.end === 5);
            expect(innerUl).toBeDefined();
        });

        it('should fold <piko:*> elements', () => {
            const doc = createMockDocument(`<template>
  <piko:svg src="icon.svg">
    <title>Icon</title>
  </piko:svg>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges.find(r => r.start === 1 && r.end === 3)).toBeDefined();
        });

        it('should fold nested <template p-for> elements and attributed templates', () => {
            const doc = createMockDocument(`<template public>
  <dl>
    <template p-for="item in state.Items" p-key="item.ID">
      <dt p-text="item.Term"></dt>
    </template>
  </dl>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges.find(r => r.start === 0 && r.end === 6 && r.kind === 3)).toBeDefined();
            expect(ranges.find(r => r.start === 2 && r.end === 4)).toBeDefined();
        });

        it('should not fold markup-like text inside Go scripts', () => {
            const doc = createMockDocument(`<script type="application/x-go">
var s = "<div>
</div>"
</script>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges).toHaveLength(1);
            expect(ranges[0].kind).toBe(3);
        });
    });

    describe('limits', () => {
        it('folds an element whose children are left unclosed', () => {
            const doc = createMockDocument(`<template>
  <div>
    <p>one
    <p>two
  </div>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges.find(r => r.start === 1 && r.end === 4)).toBeDefined();
        });

        it('stays fast on pathological markup', () => {
            const body = '<div>\n{{\n<!--\n'.repeat(20_000);
            const doc = createMockDocument(`<template>\n${body}</template>`);
            const startedAt = performance.now();

            provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(performance.now() - startedAt).toBeLessThan(5000);
        });

        it('stops collecting once the request is cancelled', () => {
            const doc = createMockDocument(`<template>
  <div>
  </div>
</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {isCancellationRequested: true} as never);

            expect(ranges).toHaveLength(0);
        });

        it('returns at most the maximum number of ranges', () => {
            const body = '<b>\n</b>\n'.repeat(MAX_FOLDING_RANGES + 10);
            const doc = createMockDocument(`<template>\n${body}</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges).toHaveLength(MAX_FOLDING_RANGES);
        });

        it('does not fold elements nested deeper than the depth limit', () => {
            const depth = MAX_TAG_DEPTH + 1;
            const body = `${'<i>\n'.repeat(depth)}${'</i>\n'.repeat(depth)}`;
            const doc = createMockDocument(`<template>\n${body}</template>`);

            const ranges = provider.provideFoldingRanges(doc as never, {} as never, {} as never);

            expect(ranges.filter(r => r.kind === undefined)).toHaveLength(MAX_TAG_DEPTH);
        });
    });
});
