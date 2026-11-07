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
import {
    buildTsContent,
    createVirtualTsUri,
    forgetHeaderLineCount,
    getHeaderLineCount,
    recordHeaderLineCount,
    sourcePathFromVirtualTsUri,
} from './virtualTs';
import {generatePKCDeclaration, generatePKDeclaration} from './pRefExtractor';

vi.mock('vscode', () => ({}));

interface MockUri {
    scheme: string;
    authority: string;
    path: string;
    with: (change: { scheme?: string; path?: string }) => MockUri;
    toString: () => string;
}

function mockUri(scheme: string, path: string, authority = ''): MockUri {
    return {
        scheme,
        authority,
        path,
        with: change => mockUri(change.scheme ?? scheme, change.path ?? path, authority),
        toString: () => `${scheme}://${authority}${path}`,
    };
}

describe('buildTsContent', () => {
    it('returns the script unchanged without a project root', () => {
        expect(buildTsContent('const a = 1;', undefined)).toEqual({content: 'const a = 1;', headerLineCount: 0});
    });

    it('counts the header lines that precede the script', () => {
        const result = buildTsContent('\nconst a = 1;\n', {
            pikoTypesPath: '/p/dist/ts/piko-ide.d.ts',
            actionsTypesPath: '/p/dist/ts/piko-actions.d.ts',
            contextDeclaration: generatePKDeclaration(['button']),
        });

        const lines = result.content.split('\n');
        expect(result.headerLineCount).toBe(5);
        expect(lines[2]).toContain('piko-ide.d.ts');
        expect(lines[result.headerLineCount]).toBe('');
        expect(lines[result.headerLineCount + 1]).toBe('const a = 1;');
    });

    it('shrinks the header when type files are missing', () => {
        const result = buildTsContent('x', {contextDeclaration: generatePKCDeclaration([])});

        expect(result.headerLineCount).toBe(3);
        expect(result.content.split('\n')[3]).toBe('x');
    });

    it('accounts for multi-line declarations', () => {
        const result = buildTsContent('x', {contextDeclaration: 'declare const a: 1;\ndeclare const b: 2;'});

        expect(result.headerLineCount).toBe(4);
        expect(result.content.split('\n')[4]).toBe('x');
    });
});

describe('virtual TypeScript URIs', () => {
    it('builds the virtual URI with Uri.with, keeping the authority', () => {
        const uri = createVirtualTsUri(mockUri('file', '/c:/Users/me/My Site/page.pk', 'server') as never) as unknown as MockUri;

        expect(uri.scheme).toBe('piko-virtual-ts');
        expect(uri.path).toBe('/c:/Users/me/My Site/page.pk.ts');
        expect(uri.authority).toBe('server');
    });

    it('appends .ts to the source path', () => {
        const uri = createVirtualTsUri(mockUri('file', '/project/components/header.pk') as never) as unknown as MockUri;

        expect(uri.scheme).toBe('piko-virtual-ts');
        expect(uri.path).toBe('/project/components/header.pk.ts');
    });

    it('maps the virtual URI back to the source path', () => {
        expect(sourcePathFromVirtualTsUri(mockUri('piko-virtual-ts', '/a/b.pkc.ts') as never)).toBe('/a/b.pkc');
        expect(sourcePathFromVirtualTsUri(mockUri('piko-virtual-ts', '/a/b.pk') as never)).toBe('/a/b.pk');
    });

    it('records, reads and forgets header line counts', () => {
        const uri = mockUri('piko-virtual-ts', '/a/b.pk.ts');

        expect(getHeaderLineCount(uri as never)).toBe(0);
        recordHeaderLineCount(uri as never, 7);
        expect(getHeaderLineCount(uri as never)).toBe(7);
        forgetHeaderLineCount(uri as never);
        expect(getHeaderLineCount(uri as never)).toBe(0);
    });
});
