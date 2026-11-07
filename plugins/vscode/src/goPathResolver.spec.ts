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

import * as fs from 'fs';
import * as os from 'os';
import * as path from 'path';
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {buildEnvWithGo, isExecutableFile, listGlobalGoLocations, resolveGoBinPath} from './goPathResolver';

const settings: Record<string, Record<string, unknown>> = {piko: {}, go: {}};

vi.mock('vscode', () => ({
    workspace: {
        getConfiguration: (section: string) => ({
            get: (key: string, fallback?: unknown) => settings[section]?.[key] ?? fallback,
        }),
    },
}));

function makeGoBin(root: string, relative: string): string {
    const binDir = path.join(root, relative);
    fs.mkdirSync(binDir, {recursive: true});
    const goPath = path.join(binDir, process.platform === 'win32' ? 'go.exe' : 'go');
    fs.writeFileSync(goPath, '#!/bin/sh\n');
    fs.chmodSync(goPath, 0o755);
    return binDir;
}

describe.skipIf(process.platform === 'win32')('goPathResolver', () => {
    let root: string;
    const savedEnv = {...process.env};

    beforeEach(() => {
        root = fs.mkdtempSync(path.join(os.tmpdir(), 'piko-go-'));
        settings.piko = {};
        settings.go = {};
        process.env['PATH'] = path.join(root, 'empty');
        process.env['HOME'] = path.join(root, 'home');
        delete process.env['GOROOT'];
    });

    afterEach(() => {
        process.env = {...savedEnv};
        fs.rmSync(root, {recursive: true, force: true});
    });

    it('prefers a valid manual goBinPath', () => {
        const manual = makeGoBin(root, 'manual/bin');
        settings.piko = {goBinPath: manual};

        expect(resolveGoBinPath()).toBe(manual);
    });

    it('ignores an invalid manual path and logs it', () => {
        const lines: string[] = [];
        settings.piko = {goBinPath: path.join(root, 'missing'), searchGlobalGoLocations: false};

        expect(resolveGoBinPath({appendLine: (line: string) => lines.push(line)} as never)).toBeUndefined();
        expect(lines.some(line => line.includes('invalid'))).toBe(true);
    });

    it('detects go.alternateTools and go.goroot from the Go extension', () => {
        const alternate = makeGoBin(root, 'alt/bin');
        settings.go = {alternateTools: {go: path.join(alternate, 'go')}};
        expect(resolveGoBinPath()).toBe(alternate);

        const goroot = makeGoBin(root, 'goroot/bin');
        settings.go = {goroot: path.dirname(goroot)};
        expect(resolveGoBinPath()).toBe(goroot);
    });

    it('uses GOROOT from the environment', () => {
        const goroot = makeGoBin(root, 'envroot/bin');
        process.env['GOROOT'] = path.dirname(goroot);

        expect(resolveGoBinPath()).toBe(goroot);
    });

    it('looks up go on PATH before global locations', () => {
        const onPath = makeGoBin(root, 'path/bin');
        makeGoBin(root, 'home/sdk/go1.22.0/bin');
        process.env['PATH'] = `${path.join(root, 'nothing')}:${onPath}`;

        expect(resolveGoBinPath()).toBe(onPath);
    });

    const hasSystemGo = ['/usr/local/go/bin/go', '/opt/go/bin/go'].some(candidate => fs.existsSync(candidate));

    it.skipIf(hasSystemGo)('falls back to the newest ~/sdk install, sorted semantically', () => {
        makeGoBin(root, 'home/sdk/go1.9/bin');
        const newest = makeGoBin(root, 'home/sdk/go1.22.10/bin');
        makeGoBin(root, 'home/sdk/go1.22.9/bin');

        expect(resolveGoBinPath()).toBe(newest);
    });

    it('returns undefined when global search is disabled and nothing is found', () => {
        makeGoBin(root, 'home/sdk/go1.22.0/bin');
        settings.piko = {searchGlobalGoLocations: false, detectGoFromExtension: false};

        expect(resolveGoBinPath()).toBeUndefined();
    });

    it('checks executables', () => {
        const binDir = makeGoBin(root, 'x/bin');

        expect(isExecutableFile(path.join(binDir, 'go'))).toBe(true);
        expect(isExecutableFile(binDir)).toBe(false);
        expect(isExecutableFile(path.join(binDir, 'missing'))).toBe(false);
    });

    it('builds the server environment', () => {
        process.env['PATH'] = '/usr/bin';
        const env = buildEnvWithGo('/go/bin');

        expect(env['PATH']).toBe('/go/bin:/usr/bin');
        expect(env['PIKO_DISABLE_CONSOLE_LOG']).toBe('true');
    });

    it('accepts go.alternateTools pointing at a bin directory', () => {
        const alternate = makeGoBin(root, 'altdir/bin');
        settings.go = {alternateTools: {go: alternate}};

        expect(resolveGoBinPath()).toBe(alternate);
    });

    it('skips Go extension detection when it is disabled', () => {
        const alternate = makeGoBin(root, 'alt/bin');
        settings.go = {alternateTools: {go: path.join(alternate, 'go')}};
        settings.piko = {detectGoFromExtension: false, searchGlobalGoLocations: false};

        expect(resolveGoBinPath()).toBeUndefined();
    });

    it('lists the global locations for each platform, then ~/go/bin and ~/sdk newest first', () => {
        const home = path.join(root, 'home');
        makeGoBin(root, 'home/sdk/go1.21.0/bin');
        makeGoBin(root, 'home/sdk/go1.23rc1/bin');
        makeGoBin(root, 'home/sdk/go1.23.0/bin');
        fs.writeFileSync(path.join(home, 'sdk', 'notes.txt'), '');

        expect(listGlobalGoLocations('linux', home)).toEqual([
            '/usr/local/go/bin',
            '/opt/go/bin',
            path.join(home, 'go', 'bin'),
            path.join(home, 'sdk', 'go1.23.0', 'bin'),
            path.join(home, 'sdk', 'go1.23rc1', 'bin'),
            path.join(home, 'sdk', 'go1.21.0', 'bin'),
        ]);
        expect(listGlobalGoLocations('darwin', home)).toContain('/opt/homebrew/opt/go/bin');
        expect(listGlobalGoLocations('win32', home)).toContain('C:\\Go\\bin');
    });

    it('lists no SDK installs when ~/sdk is missing or not a directory', () => {
        const home = path.join(root, 'home');
        expect(listGlobalGoLocations('linux', home)).toHaveLength(3);

        fs.mkdirSync(home, {recursive: true});
        fs.writeFileSync(path.join(home, 'sdk'), '');
        expect(listGlobalGoLocations('linux', home)).toHaveLength(3);
    });
});
