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

import * as vscode from 'vscode';

/**
 * Helpers for the virtual TypeScript documents that back `<script lang="ts">`
 * blocks.
 *
 * A virtual document is the script content with a header of reference
 * directives and a per-file `pk`/`pkc` declaration prepended. The header length
 * varies (missing type files, multi-line declarations), so it is computed when
 * the content is built and recorded per virtual URI for position mapping.
 */

/** The URI scheme for virtual TypeScript documents. */
export const VIRTUAL_TS_SCHEME = 'piko-virtual-ts';

/** The suffix appended to the source path to form the virtual path. */
const VIRTUAL_TS_SUFFIX = '.ts';

/**
 * Holds the inputs for building a virtual TypeScript document header.
 */
export interface TsHeaderInput {
    /** The absolute path of dist/ts/piko-ide.d.ts, when it exists. */
    pikoTypesPath?: string;
    /** The absolute path of dist/ts/piko-actions.d.ts, when it exists. */
    actionsTypesPath?: string;
    /** The generated `pk`/`pkc` context declaration (may span lines). */
    contextDeclaration: string;
}

/**
 * Holds a built virtual TypeScript document.
 */
export interface TsContent {
    /** The full document text. */
    content: string;
    /** The number of lines before the first line of script content. */
    headerLineCount: number;
}

/** Header line counts per virtual URI string, recorded as content is served. */
const headerLineCounts = new Map<string, number>();

/**
 * Builds the virtual TypeScript document for a script block.
 *
 * @param scriptContent - The raw script block content.
 * @param header - The header inputs, or undefined when no project root was found.
 * @returns The content and its header line count.
 */
export function buildTsContent(scriptContent: string, header: TsHeaderInput | undefined): TsContent {
    if (header === undefined) {
        return {content: scriptContent, headerLineCount: 0};
    }

    const lines: string[] = [
        '/// <reference lib="esnext" />',
        '/// <reference lib="dom" />',
    ];
    if (header.pikoTypesPath) {
        lines.push(`/// <reference path="${header.pikoTypesPath}" />`);
    }
    if (header.actionsTypesPath) {
        lines.push(`/// <reference path="${header.actionsTypesPath}" />`);
    }
    lines.push(header.contextDeclaration);

    const prefix = `${lines.join('\n')}\n`;
    return {content: `${prefix}${scriptContent}`, headerLineCount: countLineBreaks(prefix)};
}

/**
 * Counts the line breaks in a string.
 *
 * @param text - The text to count.
 * @returns The number of `\n` characters.
 */
function countLineBreaks(text: string): number {
    let count = 0;
    for (const char of text) {
        if (char === '\n') {
            count++;
        }
    }
    return count;
}

/**
 * Creates the virtual TypeScript URI for a Piko document.
 *
 * Built with `Uri.with` so authorities, Windows drive letters and special
 * characters survive the round trip.
 *
 * @param documentUri - The Piko document URI.
 * @returns The virtual TypeScript URI.
 */
export function createVirtualTsUri(documentUri: vscode.Uri): vscode.Uri {
    return documentUri.with({scheme: VIRTUAL_TS_SCHEME, path: `${documentUri.path}${VIRTUAL_TS_SUFFIX}`});
}

/**
 * Converts a virtual TypeScript URI back to the source document path.
 *
 * @param virtualUri - The virtual TypeScript URI.
 * @returns The source document URI path (without the `.ts` suffix).
 */
export function sourcePathFromVirtualTsUri(virtualUri: vscode.Uri): string {
    return virtualUri.path.endsWith(VIRTUAL_TS_SUFFIX)
        ? virtualUri.path.slice(0, -VIRTUAL_TS_SUFFIX.length)
        : virtualUri.path;
}

/**
 * Records the header line count served for a virtual document.
 *
 * @param virtualUri - The virtual TypeScript URI.
 * @param headerLineCount - The header line count.
 */
export function recordHeaderLineCount(virtualUri: vscode.Uri, headerLineCount: number): void {
    headerLineCounts.set(virtualUri.toString(), headerLineCount);
}

/**
 * Returns the header line count last served for a virtual document.
 *
 * @param virtualUri - The virtual TypeScript URI.
 * @returns The header line count, or 0 when the document has not been served.
 */
export function getHeaderLineCount(virtualUri: vscode.Uri): number {
    return headerLineCounts.get(virtualUri.toString()) ?? 0;
}

/**
 * Forgets the header line count for a virtual document.
 *
 * @param virtualUri - The virtual TypeScript URI.
 */
export function forgetHeaderLineCount(virtualUri: vscode.Uri): void {
    headerLineCounts.delete(virtualUri.toString());
}
