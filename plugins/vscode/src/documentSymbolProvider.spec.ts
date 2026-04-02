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
import {provideDocumentSymbols} from './documentSymbolProvider';
import {createMockDocument} from './test/mockTextDocument';

vi.mock('vscode', () => ({
    Position: class {
        constructor(public line: number, public character: number) {
        }
    },
    Range: class {
        constructor(public start: { line: number; character: number }, public end: {
            line: number;
            character: number
        }) {
        }
    },
    DocumentSymbol: class {
        children: unknown[] = [];

        constructor(
            public name: string,
            public detail: string,
            public kind: number,
            public range: unknown,
            public selectionRange: unknown
        ) {
        }
    },
    SymbolKind: {
        Namespace: 3,
        Module: 2,
        Function: 12,
        Property: 7
    },
    Uri: {
        file: (path: string) => ({fsPath: path}),
    },
}));

describe('PikoDocumentSymbolProvider', () => {
    describe('provideDocumentSymbols', () => {
        it('should return symbols for all four block types', () => {
            const doc = createMockDocument(`<template>
  <div>content</div>
</template>

<script type="application/x-go">
package main
</script>

<style>
.foo { color: red; }
</style>

<i18n lang="json">
{"en": {}}
</i18n>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(4);
            expect(symbols[0].name).toBe('<template>');
            expect(symbols[1].name).toBe('<script type="application/x-go">');
            expect(symbols[2].name).toBe('<style>');
            expect(symbols[3].name).toBe('<i18n>');
        });

        it('should detect script language from type="application/x-go"', () => {
            const doc = createMockDocument(`<script type="application/x-go">
package main
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script type="application/x-go">');
            expect(symbols[0].children).toHaveLength(1);
            expect(symbols[0].children[0].name).toBe('Go code');
        });

        it('should detect script language from lang="js"', () => {
            const doc = createMockDocument(`<script lang="js">
console.log('hello');
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script lang="js">');
            expect(symbols[0].children).toHaveLength(1);
            expect(symbols[0].children[0].name).toBe('JavaScript code');
        });

        it('should detect script language from lang="typescript"', () => {
            const doc = createMockDocument(`<script lang="typescript">
const x: number = 1;
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script lang="typescript">');
            expect(symbols[0].children).toHaveLength(1);
            expect(symbols[0].children[0].name).toBe('TypeScript code');
        });

        it('should create hierarchical symbols with children', () => {
            const doc = createMockDocument(`<template>
  <div>Hello</div>
</template>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].children).toHaveLength(1);
            expect(symbols[0].children[0].name).toBe('content');
        });

        it('should provide correct ranges for navigation', () => {
            const doc = createMockDocument(`<template>content</template>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            const symbol = symbols[0];
            expect(symbol.range.start.line).toBe(0);
            expect(symbol.range.start.character).toBe(0);
            expect(symbol.range.end.line).toBe(0);
            expect(symbol.range.end.character).toBe(28);
        });

        it('should handle empty document', () => {
            const doc = createMockDocument('');

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(0);
        });

        it('should handle document with only whitespace', () => {
            const doc = createMockDocument('   \n\n   ');

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(0);
        });

        it('should handle multiple script blocks', () => {
            const doc = createMockDocument(`<script type="application/x-go">
package main
</script>
<script lang="js">
console.log('hello');
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(2);
            expect(symbols[0].name).toBe('<script type="application/x-go">');
            expect(symbols[1].name).toBe('<script lang="js">');
        });

        it('should handle i18n with lang attribute', () => {
            const doc = createMockDocument(`<i18n lang="json">
{"en": {}}
</i18n>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<i18n>');
            expect(symbols[0].detail).toBe('Internationalisation block');
        });

        it('should provide correct symbol kinds', () => {
            const doc = createMockDocument(`<template>content</template>
<script type="application/x-go">code</script>
<style>rules</style>
<i18n>data</i18n>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols[0].kind).toBe(3);
            expect(symbols[1].kind).toBe(2);
            expect(symbols[2].kind).toBe(3);
            expect(symbols[3].kind).toBe(3);
        });

        it('should handle script without lang or type attribute', () => {
            const doc = createMockDocument(`<script >
console.log('hello');
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script>');
            expect(symbols[0].children).toHaveLength(1);
            expect(symbols[0].children[0].name).toBe('code');
        });

        it('should handle script with unknown language', () => {
            const doc = createMockDocument(`<script lang="python">
print('hello')
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script lang="python">');
            expect(symbols[0].children).toHaveLength(1);
            expect(symbols[0].children[0].name).toBe('python code');
        });

        it('should handle script with lang="ts" shorthand', () => {
            const doc = createMockDocument(`<script lang="ts">
const x: number = 1;
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script lang="typescript">');
            expect(symbols[0].children[0].name).toBe('TypeScript code');
        });

        it('should handle script with lang="javascript"', () => {
            const doc = createMockDocument(`<script lang="javascript">
const x = 1;
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script lang="js">');
            expect(symbols[0].children[0].name).toBe('JavaScript code');
        });

        it('should handle empty template block', () => {
            const doc = createMockDocument(`<template></template>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<template>');
            expect(symbols[0].children).toHaveLength(0);
        });

        it('should handle script with type containing javascript', () => {
            const doc = createMockDocument(`<script type="text/javascript">
const x = 1;
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script lang="js">');
        });

        it('should handle script with type containing typescript', () => {
            const doc = createMockDocument(`<script type="text/typescript">
const x: number = 1;
</script>`);

            const symbols = provideDocumentSymbols(doc as never);

            expect(symbols).toHaveLength(1);
            expect(symbols[0].name).toBe('<script lang="typescript">');
        });
    });
});
