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
 * Holds the symbol settings for a block type.
 */
interface BlockConfig {
    /** The display name for the block symbol. */
    name: string;
    /** The symbol kind for the block. */
    kind: vscode.SymbolKind;
    /** The detail text shown alongside the symbol. */
    detail: string;
    /** The display name for the child content symbol. */
    childName: string;
    /** The symbol kind for the child content. */
    childKind: vscode.SymbolKind;
}

/**
 * Holds the data needed to create a block symbol.
 */
interface BlockMatch {
    /** The block type (template, script, style, or i18n). */
    type: string;
    /** The full matched text including tags. */
    matchText: string;
    /** The start index of the match in the document. */
    matchIndex: number;
    /** The total length of the matched text. */
    matchLength: number;
}

/**
 * Maps block types to their symbol settings.
 */
const BLOCK_CONFIG: Record<string, BlockConfig> = {
    template: {
        name: '<template>',
        kind: vscode.SymbolKind.Namespace,
        detail: 'Template block',
        childName: 'content',
        childKind: vscode.SymbolKind.Property
    },
    script: {
        name: '<script>',
        kind: vscode.SymbolKind.Module,
        detail: 'Script block',
        childName: 'Go code',
        childKind: vscode.SymbolKind.Function
    },
    style: {
        name: '<style>',
        kind: vscode.SymbolKind.Namespace,
        detail: 'Style block',
        childName: 'CSS rules',
        childKind: vscode.SymbolKind.Property
    },
    i18n: {
        name: '<i18n>',
        kind: vscode.SymbolKind.Namespace,
        detail: 'Internationalisation block',
        childName: 'translations',
        childKind: vscode.SymbolKind.Property
    }
};

/**
 * Regex patterns for finding each block type.
 */
const BLOCK_PATTERNS = [
    {regex: /<template>([\s\S]*?)<\/template>/gi, type: 'template'},
    {regex: /<script\s+[^>]*>([\s\S]*?)<\/script>/gi, type: 'script'},
    {regex: /<style>([\s\S]*?)<\/style>/gi, type: 'style'},
    {regex: /<i18n[^>]*>([\s\S]*?)<\/i18n>/gi, type: 'i18n'}
];

/**
 * Gets the language from a script tag.
 *
 * Checks for lang="..." or type="..." attributes.
 *
 * @param tagContent - The opening script tag text.
 * @returns The language name, or null if not found.
 */
function extractScriptLanguage(tagContent: string): string | null {
    const langMatch = tagContent.match(/lang\s*=\s*["'](\w+)["']/);
    if (langMatch) {
        return langMatch[1];
    }

    const typeMatch = tagContent.match(/type\s*=\s*["']([^"']+)["']/);
    if (!typeMatch) {
        return null;
    }

    const type = typeMatch[1];
    if (type === 'application/x-go') {
        return 'go';
    }
    if (type.includes('javascript')) {
        return 'js';
    }
    if (type.includes('typescript')) {
        return 'typescript';
    }

    return null;
}

/**
 * Gets the display name for a script block based on its language.
 *
 * @param tagContent - The full opening script tag.
 * @returns The name and child name to show in the outline.
 */
function getScriptDisplayInfo(tagContent: string): { name: string; childName: string } {
    const lang = extractScriptLanguage(tagContent);

    if (!lang) {
        return {name: '<script>', childName: 'code'};
    }

    switch (lang) {
        case 'go':
            return {name: '<script type="application/x-go">', childName: 'Go code'};
        case 'js':
        case 'javascript':
            return {name: '<script lang="js">', childName: 'JavaScript code'};
        case 'typescript':
        case 'ts':
            return {name: '<script lang="typescript">', childName: 'TypeScript code'};
        default:
            return {name: `<script lang="${lang}">`, childName: `${lang} code`};
    }
}

/**
 * Gets the display name and child name for a block.
 *
 * @param blockType - The type of block (template, script, style, i18n).
 * @param matchText - The full matched text for script blocks.
 * @returns The name and child name.
 */
function getBlockDisplayInfo(blockType: string, matchText: string): { name: string; childName: string } {
    const config = BLOCK_CONFIG[blockType];

    if (blockType === 'script') {
        return getScriptDisplayInfo(matchText);
    }

    return {name: config.name, childName: config.childName};
}

/**
 * Creates a child symbol for the block content.
 *
 * @param text - The full document text.
 * @param document - The Piko document.
 * @param match - The block match data.
 * @param childName - The name for the child symbol.
 * @param childKind - The symbol kind for the child.
 * @returns The child symbol, or null if no content.
 */
function createChildSymbol(
    text: string,
    document: vscode.TextDocument,
    match: BlockMatch,
    childName: string,
    childKind: vscode.SymbolKind
): vscode.DocumentSymbol | null {
    const contentStart = text.indexOf('>', match.matchIndex) + 1;
    const contentEnd = match.matchIndex + match.matchText.lastIndexOf('</');

    if (contentEnd <= contentStart) {
        return null;
    }

    const childRange = new vscode.Range(
        document.positionAt(contentStart),
        document.positionAt(contentEnd)
    );

    return new vscode.DocumentSymbol(childName, '', childKind, childRange, childRange);
}

/**
 * Creates a document symbol for a block match.
 *
 * @param text - The full document text.
 * @param document - The Piko document.
 * @param match - The block match data.
 * @returns The document symbol for this block.
 */
function createBlockSymbol(
    text: string,
    document: vscode.TextDocument,
    match: BlockMatch
): vscode.DocumentSymbol {
    const config = BLOCK_CONFIG[match.type];
    const displayInfo = getBlockDisplayInfo(match.type, match.matchText);

    const range = new vscode.Range(
        document.positionAt(match.matchIndex),
        document.positionAt(match.matchIndex + match.matchLength)
    );

    const symbol = new vscode.DocumentSymbol(
        displayInfo.name,
        config.detail,
        config.kind,
        range,
        range
    );

    const child = createChildSymbol(text, document, match, displayInfo.childName, config.childKind);
    if (child) {
        symbol.children = [child];
    }

    return symbol;
}

/**
 * Finds all block matches in the document text.
 *
 * @param text - The document text to search.
 * @returns The list of block matches.
 */
function findBlockMatches(text: string): BlockMatch[] {
    const matches: BlockMatch[] = [];

    for (const {regex, type} of BLOCK_PATTERNS) {
        regex.lastIndex = 0;
        let match;
        while ((match = regex.exec(text)) !== null) {
            matches.push({
                type,
                matchText: match[0],
                matchIndex: match.index,
                matchLength: match[0].length
            });
        }
    }

    return matches;
}

/**
 * Creates document symbols for a Piko file.
 *
 * These symbols show in the breadcrumbs, outline panel, and go-to-symbol menu.
 *
 * @param document - The Piko document.
 * @returns The list of symbols found in the document.
 */
export function provideDocumentSymbols(document: vscode.TextDocument): vscode.DocumentSymbol[] {
    const text = document.getText();
    const matches = findBlockMatches(text);

    return matches.map(match => createBlockSymbol(text, document, match));
}

/**
 * VSCode provider for document symbols in Piko files.
 *
 * Used for breadcrumbs, outline view, and go-to-symbol.
 */
export class PikoDocumentSymbolProvider implements vscode.DocumentSymbolProvider {
    /**
     * Gets document symbols for a Piko file.
     *
     * @param document - The Piko document.
     * @param _token - The cancellation token (unused).
     * @returns The list of document symbols.
     */
    provideDocumentSymbols(
        document: vscode.TextDocument,
        _token: vscode.CancellationToken
    ): vscode.ProviderResult<vscode.DocumentSymbol[]> {
        return provideDocumentSymbols(document);
    }
}
