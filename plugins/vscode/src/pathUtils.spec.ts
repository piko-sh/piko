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
    compareGoVersionsDescending,
    expandUserPath,
    findExecutableOnPath,
    findPathKey,
    getPathValue,
    isBareCommand,
    isPathWithin,
    parseGoVersion,
    resolveCommandPath,
    withPrependedPath,
} from './pathUtils';

describe('expandUserPath', () => {
    const context = {homeDir: '/home/me', workspaceFolder: '/work/site'};

    it.each([
        ['~', '/home/me'],
        ['~/go/bin/gopls', '/home/me/go/bin/gopls'],
        ['${userHome}/bin/pikopls', '/home/me/bin/pikopls'],
        ['${workspaceFolder}/bin/pikopls', '/work/site/bin/pikopls'],
        ['  /opt/pikopls  ', '/opt/pikopls'],
        ['pikopls', 'pikopls'],
    ])('expands %s', (raw, expected) => {
        expect(expandUserPath(raw, context)).toBe(expected);
    });

    it('leaves ${workspaceFolder} alone without a workspace', () => {
        expect(expandUserPath('${workspaceFolder}/x', {homeDir: '/home/me'})).toBe('${workspaceFolder}/x');
    });
});

describe('isBareCommand', () => {
    it.each([
        ['pikopls', true],
        ['pikopls.exe', true],
        ['./pikopls', false],
        ['/usr/bin/pikopls', false],
        ['C:\\tools\\pikopls.exe', false],
        ['', false],
    ])('%s -> %s', (value, expected) => {
        expect(isBareCommand(value)).toBe(expected);
    });
});

describe('PATH helpers', () => {
    it('uses PATH on POSIX regardless of other keys', () => {
        expect(findPathKey({Path: 'x'}, 'linux')).toBe('PATH');
        expect(getPathValue({PATH: '/bin'}, 'linux')).toBe('/bin');
    });

    it('finds the Windows Path key case-insensitively', () => {
        expect(findPathKey({Path: 'C:\\bin'}, 'win32')).toBe('Path');
        expect(getPathValue({Path: 'C:\\bin'}, 'win32')).toBe('C:\\bin');
        expect(findPathKey({}, 'win32')).toBe('PATH');
        expect(getPathValue({}, 'win32')).toBe('');
    });

    it('prepends on POSIX', () => {
        expect(withPrependedPath({PATH: '/bin', HOME: '/h'}, '/go/bin', 'linux'))
            .toEqual({PATH: '/go/bin:/bin', HOME: '/h'});
    });

    it('collapses Windows PATH variants into one key', () => {
        const env = withPrependedPath({Path: 'C:\\a', PATH: 'C:\\b', OTHER: '1'}, 'C:\\Go\\bin', 'win32');

        expect(Object.keys(env).filter(key => key.toUpperCase() === 'PATH')).toHaveLength(1);
        expect(Object.values(env)).toContain('C:\\Go\\bin;C:\\a');
        expect(env['OTHER']).toBe('1');
    });

    it('handles an empty PATH and no directory', () => {
        expect(withPrependedPath({}, '/go/bin', 'linux')).toEqual({PATH: '/go/bin'});
        expect(withPrependedPath({PATH: '/bin'}, undefined, 'linux')).toEqual({PATH: '/bin'});
    });
});

describe('findExecutableOnPath', () => {
    it('returns the first executable match on POSIX', () => {
        const executables = new Set(['/b/go', '/c/go']);

        const result = findExecutableOnPath('go', {PATH: '/a:/b:/c'}, 'linux', candidate => executables.has(candidate));

        expect(result).toBe('/b/go');
    });

    it('tries PATHEXT extensions on Windows', () => {
        const executables = new Set(['C:\\Go\\bin\\go.exe']);

        const result = findExecutableOnPath('go', {Path: 'C:\\tools;C:\\Go\\bin', PATHEXT: '.COM;.EXE'}, 'win32', candidate => executables.has(candidate));

        expect(result).toBe('C:\\Go\\bin\\go.exe');
    });

    it('uses default extensions on Windows when PATHEXT is missing', () => {
        const executables = new Set(['C:\\Go\\bin\\go.exe']);

        expect(findExecutableOnPath('go', {PATH: 'C:\\Go\\bin'}, 'win32', candidate => executables.has(candidate))).toBe('C:\\Go\\bin\\go.exe');
    });

    it('does not add extensions to names that already have one', () => {
        const seen: string[] = [];
        findExecutableOnPath('go.exe', {PATH: 'C:\\bin'}, 'win32', candidate => {
            seen.push(candidate);
            return false;
        });

        expect(seen).toEqual(['C:\\bin\\go.exe']);
    });

    it('returns undefined when nothing matches', () => {
        expect(findExecutableOnPath('go', {PATH: '/a'}, 'linux', () => false)).toBeUndefined();
    });
});

describe('resolveCommandPath', () => {
    const context = {homeDir: '/home/me', workspaceFolder: '/work/site'};
    const env = {PATH: '/usr/local/bin:/usr/bin'};

    it('looks bare commands up on PATH', () => {
        const result = resolveCommandPath('pikopls', context, env, 'linux', candidate => candidate === '/usr/bin/pikopls');

        expect(result).toEqual({expanded: 'pikopls', resolved: '/usr/bin/pikopls'});
    });

    it('resolves relative paths against the workspace folder', () => {
        const result = resolveCommandPath('bin/pikopls', context, env, 'linux', candidate => candidate === '/work/site/bin/pikopls');

        expect(result.resolved).toBe('/work/site/bin/pikopls');
    });

    it('expands ~ and reports missing executables', () => {
        const result = resolveCommandPath('~/bin/pikopls', context, env, 'linux', () => false);

        expect(result).toEqual({expanded: '/home/me/bin/pikopls', resolved: undefined});
    });

    it('keeps absolute paths', () => {
        const result = resolveCommandPath('/opt/pikopls', context, env, 'linux', () => true);

        expect(result.resolved).toBe('/opt/pikopls');
    });
});

describe('Go SDK version sorting', () => {
    it('parses versions', () => {
        expect(parseGoVersion('go1.22.10')).toEqual([1, 22, 10, 2, 0]);
        expect(parseGoVersion('go1.23rc1')).toEqual([1, 23, 0, 1, 1]);
        expect(parseGoVersion('go1.24beta2')).toEqual([1, 24, 0, 0, 2]);
        expect(parseGoVersion('gotip')).toBeUndefined();
    });

    it('sorts newest first, semantically rather than lexically', () => {
        const names = ['go1.9', 'go1.22.9', 'gotip', 'go1.22.10', 'go1.23rc1', 'go1.23.0', 'go1.21', 'goofy'];

        expect([...names].sort(compareGoVersionsDescending))
            .toEqual(['go1.23.0', 'go1.23rc1', 'go1.22.10', 'go1.22.9', 'go1.21', 'go1.9', 'gotip', 'goofy']);
    });

    it('treats equal versions as equal', () => {
        expect(compareGoVersionsDescending('go1.22', 'go1.22.0')).toBe(0);
    });
});

describe('isPathWithin', () => {
    it.each([
        ['/work/site/dist/ts', '/work/site', true],
        ['/work/site', '/work/site', true],
        ['/work/site-other', '/work/site', false],
        ['/work', '/work/site', false],
        ['/elsewhere', '/work/site', false],
    ])('%s within %s -> %s', (candidate, directory, expected) => {
        expect(isPathWithin(candidate, directory)).toBe(expected);
    });
});
