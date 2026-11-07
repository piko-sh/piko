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

/**
 * A zero-based line and character, like vscode.Position.
 */
export interface MockPosition {
    /** The zero-based line. */
    line: number;
    /** The zero-based character within the line. */
    character: number;
}

/**
 * The subset of vscode.TextDocument used by the code under test.
 */
export interface MockTextDocument {
    /** The document URI. */
    uri: { fsPath: string };
    /** The document language. */
    languageId: string;

    /** Returns the full document text. */
    getText(): string;

    /** Converts a position to an offset. */
    offsetAt(position: MockPosition): number;

    /** Converts an offset to a position. */
    positionAt(offset: number): MockPosition;
}

/**
 * Creates a mock TextDocument from a string.
 *
 * Implements the subset of vscode.TextDocument the providers use. Offsets
 * and positions are converted with a binary search over line starts, so
 * large documents stay fast.
 *
 * @param text - The document text.
 * @param languageId - The document language.
 * @returns The mock document.
 */
export function createMockDocument(text: string, languageId = 'piko'): MockTextDocument {
    const lineStarts = computeLineStarts(text);

    return {
        getText: () => text,

        offsetAt: (position: MockPosition): number => {
            if (position.line >= lineStarts.length) {
                return text.length;
            }
            const lineStart = lineStarts[position.line];
            const lineEnd = position.line + 1 < lineStarts.length ? lineStarts[position.line + 1] - 1 : text.length;
            return lineStart + Math.min(position.character, lineEnd - lineStart);
        },

        positionAt: (offset: number): MockPosition => {
            const clamped = Math.max(0, Math.min(offset, text.length));
            let low = 0;
            let high = lineStarts.length - 1;
            while (low < high) {
                const middle = Math.ceil((low + high) / 2);
                if (lineStarts[middle] <= clamped) {
                    low = middle;
                } else {
                    high = middle - 1;
                }
            }
            return {line: low, character: clamped - lineStarts[low]};
        },

        uri: {fsPath: '/test/file.pk'},
        languageId,
    };
}

/**
 * Creates a MockPosition (simulates vscode.Position).
 *
 * @param line - The zero-based line.
 * @param character - The zero-based character.
 * @returns The position.
 */
export function createPosition(line: number, character: number): MockPosition {
    return {line, character};
}

/**
 * Lists the offset where each line starts.
 *
 * @param text - The document text.
 * @returns The line start offsets, beginning with 0.
 */
function computeLineStarts(text: string): number[] {
    const starts = [0];
    for (let index = text.indexOf('\n'); index !== -1; index = text.indexOf('\n', index + 1)) {
        starts.push(index + 1);
    }
    return starts;
}
