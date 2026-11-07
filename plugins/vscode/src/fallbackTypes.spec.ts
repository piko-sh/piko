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
import {ensureFallbackTypes} from './fallbackTypes';

vi.mock('./fallbackTypes/piko-ide.txt', () => ({default: '// ide types'}));
vi.mock('./fallbackTypes/piko-actions.txt', () => ({default: '// action types'}));

describe('ensureFallbackTypes', () => {
    let root: string;

    beforeEach(() => {
        root = fs.mkdtempSync(path.join(os.tmpdir(), 'piko-types-'));
    });

    afterEach(() => {
        fs.rmSync(root, {recursive: true, force: true});
    });

    it('writes missing type files and logs them', () => {
        const log = vi.fn();

        ensureFallbackTypes(root, log);

        expect(fs.readFileSync(path.join(root, 'dist', 'ts', 'piko-ide.d.ts'), 'utf-8')).toBe('// ide types');
        expect(fs.readFileSync(path.join(root, 'dist', 'ts', 'piko-actions.d.ts'), 'utf-8')).toBe('// action types');
        expect(log).toHaveBeenCalledTimes(2);
    });

    it('never overwrites existing files', () => {
        fs.mkdirSync(path.join(root, 'dist', 'ts'), {recursive: true});
        fs.writeFileSync(path.join(root, 'dist', 'ts', 'piko-ide.d.ts'), 'real');

        ensureFallbackTypes(root);

        expect(fs.readFileSync(path.join(root, 'dist', 'ts', 'piko-ide.d.ts'), 'utf-8')).toBe('real');
    });

    it('logs write failures', () => {
        const blocker = path.join(root, 'dist');
        fs.writeFileSync(blocker, 'not a directory');
        const log = vi.fn();

        ensureFallbackTypes(root, log);

        expect(log).toHaveBeenCalledWith(expect.stringContaining('Failed to write'));
    });
});
