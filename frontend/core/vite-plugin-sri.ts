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

import { createHash } from 'crypto';
import { promises as fs } from 'fs';
import * as path from 'path';
import type { Plugin, ResolvedConfig } from 'vite';

/**
 * Vite plugin that generates .sri sidecar files containing SHA-384 Subresource
 * Integrity hashes for each uncompressed build output. Compressed variants
 * (.br, .gz) are skipped because the SRI hash must match the decompressed
 * content that the browser will verify.
 */
export default function sriPlugin(): Plugin {
    let resolvedConfig: ResolvedConfig;

    return {
        name: 'vite-plugin-sri',
        enforce: 'post',
        apply: 'build',

        configResolved(config) {
            resolvedConfig = config;
        },

        async writeBundle() {
            const outDir = path.resolve(process.cwd(), resolvedConfig.build.outDir);
            await generateSRIForDir(outDir);
        },
    };
}

async function generateSRIForDir(dirPath: string): Promise<void> {
    try {
        const entries = await fs.readdir(dirPath, { withFileTypes: true });

        for (const entry of entries) {
            const fullPath = path.join(dirPath, entry.name);

            if (entry.isDirectory()) {
                await generateSRIForDir(fullPath);
                continue;
            }

            // Skip sidecar files, compressed variants, and sourcemaps.
            if (entry.name.endsWith('.sri') || entry.name.endsWith('.etag') ||
                entry.name.endsWith('.br') || entry.name.endsWith('.gz') ||
                entry.name.endsWith('.map')) {
                continue;
            }

            try {
                const fileData = await fs.readFile(fullPath);
                const hash = createHash('sha384').update(fileData).digest('base64');

                const sriValue = `sha384-${hash}`;

                const sriPath = `${fullPath}.sri`;
                await fs.writeFile(sriPath, sriValue, 'utf-8');
            } catch (readErr) {
                console.error(`[SRI Plugin] Failed to read or write SRI for ${fullPath}:`, readErr);
            }
        }
    } catch (dirErr) {
        console.error(`[SRI Plugin] Failed to read directory ${dirPath}:`, dirErr);
    }
}
