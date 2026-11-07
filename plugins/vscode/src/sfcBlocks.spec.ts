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

import {describe, expect, it} from 'vitest';
import {
    classifyBlock,
    findBlockAtOffset,
    findLastTagIndex,
    isGoScript,
    isTypeScriptScript,
    MarkupTag,
    MAX_MARKUP_TAGS,
    MAX_SFC_BLOCKS,
    parseSfcBlocks,
    scanMarkupTags,
    SfcBlockKind,
} from './sfcBlocks';

const PKC_NAMED_TEMPLATE = `<template name="example-greeting">
    <p class="greeting" p-text="state.message"></p>
</template>

<script lang="ts">
    const state = {
        message: "Hello from Piko!" as string,
    };
</script>

<style>
    .greeting {
        margin: 0;
    }
</style>
`;

const PK_PUBLIC_TEMPLATE = `
<template public>
  <div class="file-list-section">
    <h2>Uploaded files</h2>
  </div>
</template>

<script type="application/x-go">
package main
</script>
`;

const PK_NESTED_P_FOR = `<template>
  <div>
    <h1>Definitions</h1>
    <dl>
      <template p-for="(idx, item) in state.Definitions" p-key="item.ID">
        <dt p-text="item.Term"></dt>
        <dd p-text="item.Definition"></dd>
      </template>
    </dl>
  </div>
</template>

<script type="application/x-go">
  package main
</script>
`;

const PK_SCRIPT_IN_STRING = `<template>
  <div>
    <p>{{ state.MaliciousInput }}</p>
  </div>
</template>

<script type="application/x-go">
  package main
  import "piko.sh/piko"

  type Response struct { MaliciousInput string }

  func Render(r *piko.RequestData, props piko.NoProps) (Response, piko.Metadata, error) {
    return Response{MaliciousInput: "<script>alert('xss')</script>"}, piko.Metadata{}, nil
  }
</script>`;

interface BlockCase {
    name: string;
    input: string;
    kinds: SfcBlockKind[];
    contents?: Record<number, string>;
    contains?: Record<number, string[]>;
}

const CASES: BlockCase[] = [
    {
        name: 'pkc with <template name> and TypeScript script',
        input: PKC_NAMED_TEMPLATE,
        kinds: ['template', 'script-ts', 'style'],
        contents: {0: '\n    <p class="greeting" p-text="state.message"></p>\n'},
    },
    {
        name: 'pk with leading comment and <template public>',
        input: PK_PUBLIC_TEMPLATE,
        kinds: ['template', 'script'],
        contains: {0: ['file-list-section'], 1: ['package main']},
    },
    {
        name: 'nested p-for template does not end the outer template',
        input: PK_NESTED_P_FOR,
        kinds: ['template', 'script'],
        contains: {0: ['<template p-for=', '</template>\n    </dl>']},
    },
    {
        name: '</script> inside a Go string literal does not end the script',
        input: PK_SCRIPT_IN_STRING,
        kinds: ['template', 'script'],
        contains: {1: ["alert('xss')</script>\"}", 'piko.Metadata{}, nil\n  }\n']},
    },
    {
        name: 'second template is discarded',
        input: '<template>First</template>\n<template>Second</template>',
        kinds: ['template'],
        contents: {0: 'First'},
    },
    {
        name: 'tags inside comments are ignored',
        input: '<!-- <template>commented</template> -->',
        kinds: [],
    },
    {
        name: 'mixed-case tags are recognised',
        input: '<TEMPLATE>UPPER</TEMPLATE>',
        kinds: ['template'],
        contents: {0: 'UPPER'},
    },
    {
        name: 'self-closing script has empty content',
        input: '<template>Main</template>\n<script src="./my-script.js" />',
        kinds: ['template', 'script-js'],
        contents: {1: ''},
    },
    {
        name: 'template containing an inline script and style',
        input: '<template><div><script>let t = "</template>";</script><style>a{}</style></div></template>',
        kinds: ['template'],
        contents: {0: '<div><script>let t = "</template>";</script><style>a{}</style></div>'},
    },
    {
        name: 'interleaved blocks of every type',
        input: '<style>s1</style>\n<script>js1</script>\n<template>tmpl</template>\n<i18n>i</i18n>\n<piko:timeline>t</piko:timeline>',
        kinds: ['style', 'script-js', 'template', 'i18n', 'timeline'],
        contents: {0: 's1', 1: 'js1', 2: 'tmpl', 3: 'i', 4: 't'},
    },
    {
        name: 'quoted attribute values may contain >',
        input: '<template data-x="a > b"><p>x</p></template>',
        kinds: ['template'],
        contents: {0: '<p>x</p>'},
    },
    {
        name: 'unclosed script runs to the end of the text',
        input: '<script type="application/x-go">package main',
        kinds: ['script'],
        contents: {0: 'package main'},
    },
    {
        name: 'unterminated opening tag stops parsing',
        input: '<template><div>hello</div></template><script',
        kinds: ['template'],
    },
    {
        name: 'multiple i18n blocks',
        input: '<i18n lang="en">{ "a": "Hello" }</i18n>\n<i18n lang="es">{ "a": "Hola" }</i18n>',
        kinds: ['i18n', 'i18n'],
        contents: {0: '{ "a": "Hello" }', 1: '{ "a": "Hola" }'},
    },
    {
        name: 'stray < and end tags at the top level are skipped',
        input: 'a < b </div> <!DOCTYPE html>\n<template>ok</template>',
        kinds: ['template'],
        contents: {0: 'ok'},
    },
];

describe('parseSfcBlocks', () => {
    it.each(CASES)('$name', ({input, kinds, contents, contains}) => {
        const blocks = parseSfcBlocks(input);

        expect(blocks.map(b => b.kind)).toEqual(kinds);
        for (const [index, content] of Object.entries(contents ?? {})) {
            expect(blocks[Number(index)].content).toBe(content);
        }
        for (const [index, needles] of Object.entries(contains ?? {})) {
            for (const needle of needles) {
                expect(blocks[Number(index)].content).toContain(needle);
            }
        }
    });

    it('records offsets that slice back to the block and its content', () => {
        const blocks = parseSfcBlocks(PK_NESTED_P_FOR);

        for (const block of blocks) {
            expect(PK_NESTED_P_FOR.slice(block.contentStart, block.contentEnd)).toBe(block.content);
            expect(PK_NESTED_P_FOR.slice(block.start, block.start + 1)).toBe('<');
            expect(PK_NESTED_P_FOR.slice(block.end - 1, block.end)).toBe('>');
        }
        expect(PK_NESTED_P_FOR.slice(blocks[0].contentEnd, blocks[0].end)).toBe('</template>');
    });

    it('parses attribute quoting variations', () => {
        const [block] = parseSfcBlocks(`<script name='single' type=unquoted enabled value="">x</script>`);

        expect(block.attributes).toEqual({name: 'single', type: 'unquoted', enabled: '', value: ''});
    });

    it('lower-cases attribute names', () => {
        const [block] = parseSfcBlocks('<script TYPE="application/x-go">x</script>');

        expect(block.kind).toBe('script');
    });

    it('marks the block as self-closing', () => {
        const [block] = parseSfcBlocks('<template />');

        expect(block.selfClosing).toBe(true);
        expect(block.content).toBe('');
    });

    it('handles empty input', () => {
        expect(parseSfcBlocks('')).toEqual([]);
        expect(parseSfcBlocks(' \n\t ')).toEqual([]);
    });
});

describe('script classification', () => {
    it.each([
        [{type: 'application/x-go'}, 'script'],
        [{type: 'application/go'}, 'script'],
        [{lang: 'go'}, 'script'],
        [{lang: 'golang'}, 'script'],
        [{lang: 'application/x-go'}, 'script'],
        [{lang: 'application/go'}, 'script'],
        [{lang: 'ts'}, 'script-ts'],
        [{lang: 'typescript'}, 'script-ts'],
        [{type: 'application/typescript'}, 'script-ts'],
        [{type: 'Application/TypeScript'}, 'script-ts'],
        [{type: 'text/x-go'}, 'script-js'],
        [{type: 'text/typescript'}, 'script-js'],
        [{type: 'application/javascript'}, 'script-js'],
        [{lang: 'js'}, 'script-js'],
        [{}, 'script-js'],
    ] as Array<[Record<string, string>, SfcBlockKind]>)('%o is %s', (attributes, expected) => {
        expect(classifyBlock('script', attributes)).toBe(expected);
    });

    it('classifies non-script tags by name', () => {
        expect(classifyBlock('template', {})).toBe('template');
        expect(classifyBlock('style', {})).toBe('style');
        expect(classifyBlock('i18n', {})).toBe('i18n');
        expect(classifyBlock('piko:timeline', {})).toBe('timeline');
    });

    it('exposes the Go and TypeScript predicates', () => {
        expect(isGoScript({type: 'application/x-go'})).toBe(true);
        expect(isGoScript({lang: 'ts'})).toBe(false);
        expect(isTypeScriptScript({lang: 'ts'})).toBe(true);
        expect(isTypeScriptScript({type: 'application/x-go'})).toBe(false);
    });
});

describe('findBlockAtOffset', () => {
    const blocks = parseSfcBlocks(PK_PUBLIC_TEMPLATE);

    it('returns undefined in the leading comment', () => {
        expect(findBlockAtOffset(blocks, 5)).toBeUndefined();
    });

    it('finds the template, including its tags', () => {
        const templateStart = PK_PUBLIC_TEMPLATE.indexOf('<template public>');
        expect(findBlockAtOffset(blocks, templateStart)?.kind).toBe('template');
        expect(findBlockAtOffset(blocks, PK_PUBLIC_TEMPLATE.indexOf('</template>') + 3)?.kind).toBe('template');
    });

    it('finds the Go script', () => {
        expect(findBlockAtOffset(blocks, PK_PUBLIC_TEMPLATE.indexOf('package main'))?.kind).toBe('script');
    });
});

describe('parseSfcBlocks limits', () => {
    it('stops after the maximum number of blocks', () => {
        const text = '<style></style>'.repeat(MAX_SFC_BLOCKS + 5);

        expect(parseSfcBlocks(text)).toHaveLength(MAX_SFC_BLOCKS);
    });

    it('parses many unclosed script openers in linear time', () => {
        const text = '<script>'.repeat(50_000);
        const startedAt = performance.now();

        const blocks = parseSfcBlocks(text);

        expect(blocks).toHaveLength(1);
        expect(blocks[0].end).toBe(text.length);
        expect(performance.now() - startedAt).toBeLessThan(2000);
    });

    it('ends a raw block at the last closing tag before the next opener', () => {
        const text = '<script>a</script>b</script><script>c</script>';

        const blocks = parseSfcBlocks(text);

        expect(blocks.map(block => block.content)).toEqual(['a</script>b', 'c']);
    });
});

describe('scanMarkupTags', () => {
    function collect(text: string): MarkupTag[] {
        const tags: MarkupTag[] = [];
        scanMarkupTags(text, 0, text.length, tag => {
            tags.push(tag);
            return true;
        });
        return tags;
    }

    it('reports opening, closing and self-closing tags with offsets', () => {
        const tags = collect('<div class="a > b"><br/></DIV>');

        expect(tags).toEqual([
            {kind: 'open', name: 'div', start: 0, end: 19, selfClosing: false},
            {kind: 'open', name: 'br', start: 19, end: 24, selfClosing: true},
            {kind: 'close', name: 'div', start: 24, end: 30, selfClosing: false},
        ]);
    });

    it('skips comments, doctypes and raw-text content', () => {
        const tags = collect('<!-- <a> --><!doctype html><script>if (a <b) {}</script><p>');

        expect(tags.map(tag => `${tag.kind}:${tag.name}`)).toEqual(['open:script', 'close:script', 'open:p']);
    });

    it('stops at an unterminated tag', () => {
        expect(collect('<p><a href="x')).toHaveLength(1);
    });

    it('stops when the visitor returns false', () => {
        const tags: MarkupTag[] = [];
        scanMarkupTags('<a><b><c>', 0, 9, tag => {
            tags.push(tag);
            return tags.length < 2;
        });

        expect(tags).toHaveLength(2);
    });

    it('does not start tags at or after the end offset', () => {
        const text = '<a><b>';
        const tags: MarkupTag[] = [];
        scanMarkupTags(text, 0, 3, tag => {
            tags.push(tag);
            return true;
        });

        expect(tags.map(tag => tag.name)).toEqual(['a']);
    });

    it('visits at most the maximum number of tags', () => {
        const text = '<i>'.repeat(MAX_MARKUP_TAGS + 10);
        let count = 0;
        scanMarkupTags(text, 0, text.length, () => {
            count++;
            return true;
        });

        expect(count).toBe(MAX_MARKUP_TAGS);
    });
});

describe('findLastTagIndex', () => {
    it('finds the nearest entry with the name', () => {
        const stack = [{name: 'div'}, {name: 'p'}, {name: 'div'}];

        expect(findLastTagIndex(stack, 'div')).toBe(2);
        expect(findLastTagIndex(stack, 'p')).toBe(1);
        expect(findLastTagIndex(stack, 'span')).toBe(-1);
    });
});
