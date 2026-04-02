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
import {parseSfc} from './sfcParserClient';

describe('sfcParserClient', () => {
    describe('parseSfc', () => {
        it('should parse an empty string', () => {
            const result = parseSfc('');
            expect(result).toEqual([]);
        });

        it('should parse a template block', () => {
            const input = '<template><div>Hello</div></template>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('template');
            expect(result[0].content).toBe('<div>Hello</div>');
        });

        it('should parse a script block with Go type', () => {
            const input = '<script type="application/x-go">package main</script>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('script');
            expect(result[0].content).toBe('package main');
        });

        it('should parse a TypeScript script block with lang="ts"', () => {
            const input = '<script lang="ts">const x: number = 1;</script>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('script-ts');
            expect(result[0].content).toBe('const x: number = 1;');
        });

        it('should parse a TypeScript script block with lang="typescript"', () => {
            const input = '<script lang="typescript">const x: number = 1;</script>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('script-ts');
            expect(result[0].content).toBe('const x: number = 1;');
        });

        it('should parse a TypeScript script block with type="application/typescript"', () => {
            const input = '<script type="application/typescript">const x: number = 1;</script>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('script-ts');
            expect(result[0].content).toBe('const x: number = 1;');
        });

        it('should parse a TypeScript script block with type="text/typescript"', () => {
            const input = '<script type="text/typescript">const x: number = 1;</script>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('script-ts');
            expect(result[0].content).toBe('const x: number = 1;');
        });

        it('should parse a style block', () => {
            const input = '<style>.foo { color: red; }</style>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('style');
            expect(result[0].content).toBe('.foo { color: red; }');
        });

        it('should parse an i18n block', () => {
            const input = '<i18n>{"en": {"hello": "Hello"}}</i18n>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('i18n');
            expect(result[0].content).toBe('{"en": {"hello": "Hello"}}');
        });

        it('should parse multiple blocks', () => {
            const input = `<template><div>Hello</div></template>
<script type="application/x-go">package main</script>
<style>.foo { color: red; }</style>`;

            const result = parseSfc(input);

            expect(result).toHaveLength(3);
            expect(result.map(b => b.type)).toContain('template');
            expect(result.map(b => b.type)).toContain('script');
            expect(result.map(b => b.type)).toContain('style');
        });

        it('should track correct start and end offsets', () => {
            const input = '<template>content</template>';
            const result = parseSfc(input);

            expect(result[0].start).toBe(0);
            expect(result[0].end).toBe(input.length);
        });

        it('should handle multiline content', () => {
            const input = `<template>
  <div>
    <span>Hello</span>
  </div>
</template>`;
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].content).toContain('<div>');
            expect(result[0].content).toContain('<span>Hello</span>');
        });

        it('should handle script with single quotes in type attribute', () => {
            const input = "<script type='application/x-go'>package main</script>";
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('script');
            expect(result[0].content).toBe('package main');
        });

        it('should handle script with additional attributes', () => {
            const input = '<script type="application/x-go" lang="go">package main</script>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].type).toBe('script');
            expect(result[0].content).toBe('package main');
        });

        it('should handle real-world Piko file structure', () => {
            const input = `<script type="application/x-go">
package components

import "fmt"

func Render() string {
    return fmt.Sprintf("Hello, %s!", "World")
}
</script>

<template>
  <div class="container">
    <h1>{{ .Title }}</h1>
    <p>{{ .Content }}</p>
  </div>
</template>

<style>
.container {
    max-width: 800px;
    margin: 0 auto;
}
</style>`;

            const result = parseSfc(input);

            expect(result).toHaveLength(3);

            const script = result.find(b => b.type === 'script');
            const template = result.find(b => b.type === 'template');
            const style = result.find(b => b.type === 'style');

            expect(script).toBeDefined();
            expect(script!.content).toContain('package components');
            expect(script!.content).toContain('func Render()');

            expect(template).toBeDefined();
            expect(template!.content).toContain('<div class="container">');
            expect(template!.content).toContain('{{ .Title }}');

            expect(style).toBeDefined();
            expect(style!.content).toContain('.container');
            expect(style!.content).toContain('max-width: 800px');
        });

        it('should not match regular script tags without x-go type', () => {
            const input = '<script>console.log("hello")</script>';
            const result = parseSfc(input);

            expect(result).toHaveLength(0);
        });

        it('should handle empty blocks', () => {
            const input = '<template></template>';
            const result = parseSfc(input);

            expect(result).toHaveLength(1);
            expect(result[0].content).toBe('');
        });
    });
});
