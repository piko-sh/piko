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
import * as path from 'path';

// Imported as raw text strings via esbuild's fallback-types-text plugin.
// Copied as .txt to prevent TypeScript from treating them as declaration files.
import fallbackPikoIdeTypes from './fallbackTypes/piko-ide.txt';
import fallbackPikoActionsTypes from './fallbackTypes/piko-actions.txt';

/** File permission mode for created type definition files. */
const TYPE_FILE_PERMISSIONS = 0o640;

/** Fallback type definition files to write. */
const FALLBACK_FILES = [
    {name: 'piko-ide.d.ts', content: fallbackPikoIdeTypes},
    {name: 'piko-actions.d.ts', content: fallbackPikoActionsTypes},
] as const;

/**
 * Ensures fallback type definitions exist on disk.
 *
 * If dist/ts/piko-ide.d.ts or dist/ts/piko-actions.d.ts do not exist,
 * writes the embedded fallback content to those paths. This provides
 * autocomplete before `piko run` is executed for the first time.
 *
 * When `piko run` subsequently writes the real files, the IDE's file
 * watcher will detect the change and pick up the updated versions.
 *
 * @param projectRoot - The project root directory path.
 * @param log - Optional logging callback.
 */
export function ensureFallbackTypes(projectRoot: string, log?: (msg: string) => void): void {
    const distTsDir = path.join(projectRoot, 'dist', 'ts');

    for (const file of FALLBACK_FILES) {
        const filePath = path.join(distTsDir, file.name);
        if (!fs.existsSync(filePath)) {
            try {
                fs.mkdirSync(distTsDir, {recursive: true, mode: 0o750});
                fs.writeFileSync(filePath, file.content, {encoding: 'utf-8', mode: TYPE_FILE_PERMISSIONS});
                log?.(`[FallbackTypes] Wrote fallback ${file.name} to ${filePath}`);
            } catch (error) {
                log?.(`[FallbackTypes] Failed to write ${file.name}: ${error}`);
            }
        }
    }
}
