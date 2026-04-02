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
 * Creates a mock VS Code TextDocument for testing purposes.
 * This allows testing blockDetector without requiring the actual VS Code API.
 */
export interface MockPosition {
    line: number;
    character: number;
}

export interface MockTextDocument {
    uri: { fsPath: string };
    languageId: string;

    getText(): string;

    offsetAt(position: MockPosition): number;

    positionAt(offset: number): MockPosition;
}

/**
 * Creates a mock TextDocument from a string.
 * Implements the subset of vscode.TextDocument needed for blockDetector.
 */
export function createMockDocument(text: string, languageId = 'piko'): MockTextDocument {
    const lines = text.split('\n');

    return {
        getText: () => text,

        offsetAt: (position: MockPosition): number => {
            let offset = 0;
            for (let i = 0; i < position.line && i < lines.length; i++) {
                offset += lines[i].length + 1; // +1 for newline
            }
            offset += Math.min(position.character, lines[position.line]?.length ?? 0);
            return offset;
        },

        positionAt: (offset: number): MockPosition => {
            let remaining = offset;
            let line = 0;

            while (line < lines.length && remaining > lines[line].length) {
                remaining -= lines[line].length + 1; // +1 for newline
                line++;
            }

            return {
                line,
                character: Math.max(0, remaining)
            };
        },

        uri: {fsPath: '/test/file.pk'},
        languageId,
    };
}

/**
 * Creates a MockPosition (simulates vscode.Position)
 */
export function createPosition(line: number, character: number): MockPosition {
    return {line, character};
}
