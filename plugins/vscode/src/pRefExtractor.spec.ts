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
import {extractPRefNames, generatePKDeclaration, generatePKCDeclaration} from './pRefExtractor';

describe('pRefExtractor', () => {
    describe('extractPRefNames', () => {
        it('should return empty array for empty template', () => {
            expect(extractPRefNames('')).toEqual([]);
        });

        it('should return empty array for template without p-ref', () => {
            const template = '<div class="container"><input type="text" /></div>';
            expect(extractPRefNames(template)).toEqual([]);
        });

        it('should extract a single p-ref name', () => {
            const template = '<input p-ref="emailInput" type="text" />';
            expect(extractPRefNames(template)).toEqual(['emailInput']);
        });

        it('should extract multiple p-ref names in order', () => {
            const template = `
                <input p-ref="emailInput" type="text" />
                <button p-ref="submitBtn">Submit</button>
                <canvas p-ref="chart"></canvas>
            `;
            expect(extractPRefNames(template)).toEqual(['emailInput', 'submitBtn', 'chart']);
        });

        it('should deduplicate p-ref names', () => {
            const template = `
                <input p-ref="myRef" />
                <div p-ref="myRef"></div>
            `;
            expect(extractPRefNames(template)).toEqual(['myRef']);
        });

        it('should handle single-quoted values', () => {
            const template = "<input p-ref='emailInput' />";
            expect(extractPRefNames(template)).toEqual(['emailInput']);
        });

        it('should handle spaces around equals sign', () => {
            const template = '<input p-ref = "emailInput" />';
            expect(extractPRefNames(template)).toEqual(['emailInput']);
        });

        it('should ignore p-ref with empty value', () => {
            const template = '<input p-ref="" />';
            expect(extractPRefNames(template)).toEqual([]);
        });

        it('should handle ref names starting with underscore', () => {
            const template = '<div p-ref="_internal"></div>';
            expect(extractPRefNames(template)).toEqual(['_internal']);
        });

        it('should handle ref names starting with dollar sign', () => {
            const template = '<div p-ref="$el"></div>';
            expect(extractPRefNames(template)).toEqual(['$el']);
        });
    });

    describe('generatePKDeclaration', () => {
        it('should declare pk namespace with empty refs for empty list', () => {
            const result = generatePKDeclaration([]);
            expect(result).toContain('declare namespace pk');
            expect(result).toContain('const refs: {}');
        });

        it('should include lifecycle methods for empty list', () => {
            const result = generatePKDeclaration([]);
            expect(result).toContain('onConnected(cb: () => void): void');
            expect(result).toContain('onDisconnected(cb: () => void): void');
            expect(result).toContain('onBeforeRender(cb: () => void): void');
            expect(result).toContain('onAfterRender(cb: () => void): void');
            expect(result).toContain('onUpdated(cb: (context?: unknown) => void): void');
            expect(result).toContain('onCleanup(fn: () => void): void');
        });

        it('should generate pk with single ref', () => {
            const result = generatePKDeclaration(['emailInput']);
            expect(result).toContain('declare namespace pk');
            expect(result).toContain('readonly emailInput: HTMLElement | null');
        });

        it('should generate pk with multiple refs', () => {
            const result = generatePKDeclaration(['emailInput', 'submitBtn']);
            expect(result).toContain('readonly emailInput: HTMLElement | null');
            expect(result).toContain('readonly submitBtn: HTMLElement | null');
        });

        it('should wrap refs in const refs', () => {
            const result = generatePKDeclaration(['myRef']);
            expect(result).toContain('const refs: { readonly myRef: HTMLElement | null }');
        });

        it('should always include lifecycle methods alongside refs', () => {
            const result = generatePKDeclaration(['myRef']);
            expect(result).toContain('onConnected(cb: () => void): void');
            expect(result).toContain('onCleanup(fn: () => void): void');
        });
    });

    describe('generatePKCDeclaration', () => {
        it('should declare pkc as interface extending HTMLElement with empty refs', () => {
            const result = generatePKCDeclaration([]);
            expect(result).toContain('interface _PikoComponent extends HTMLElement');
            expect(result).toContain('declare const pkc: _PikoComponent');
            expect(result).toContain('readonly refs: {}');
        });

        it('should include component instance methods', () => {
            const result = generatePKCDeclaration([]);
            expect(result).toContain('setState(partialState: Record<string, unknown>): void');
            expect(result).toContain('render(): void');
            expect(result).toContain('scheduleRender(): void');
            expect(result).toContain('state: Record<string, unknown> | undefined');
        });

        it('should include lifecycle registration methods', () => {
            const result = generatePKCDeclaration([]);
            expect(result).toContain('onConnected(cb: () => void): void');
            expect(result).toContain('onDisconnected(cb: () => void): void');
            expect(result).toContain('onBeforeRender(cb: () => void): void');
            expect(result).toContain('onAfterRender(cb: () => void): void');
            expect(result).toContain('onUpdated(cb: (changedProperties: Set<string>) => void): void');
        });

        it('should include onCleanup', () => {
            const result = generatePKCDeclaration([]);
            expect(result).toContain('onCleanup(cb: () => void): void');
        });

        it('should include typed refs for pkc', () => {
            const result = generatePKCDeclaration(['myInput', 'submitBtn']);
            expect(result).toContain('readonly myInput: HTMLElement | null');
            expect(result).toContain('readonly submitBtn: HTMLElement | null');
        });

        it('should extend HTMLElement for access to standard DOM methods', () => {
            const result = generatePKCDeclaration(['myRef']);
            expect(result).toMatch(/interface _PikoComponent extends HTMLElement/);
            expect(result).toMatch(/declare const pkc: _PikoComponent/);
        });

        it('should include slot management methods', () => {
            const result = generatePKCDeclaration([]);
            expect(result).toContain('attachSlotListener(slotName: string, callback: (elements: Element[]) => void): void');
            expect(result).toContain('getSlottedElements(slotName?: string): Element[]');
            expect(result).toContain('hasSlotContent(slotName?: string): boolean');
        });
    });
});
