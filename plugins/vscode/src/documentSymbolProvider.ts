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
import {getAttribute, parseSfcBlocks, SfcBlock, SfcBlockTag} from './sfcBlocks';

/**
 * Holds the symbol settings for a block tag.
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
 * Builds the symbol settings for each block tag.
 *
 * Built lazily so the vscode enums are read at call time.
 *
 * @param tag - The block tag.
 * @returns The symbol settings for the tag.
 */
function blockConfig(tag: SfcBlockTag): BlockConfig {
    switch (tag) {
        case 'template':
            return {name: '<template>', kind: vscode.SymbolKind.Namespace, detail: 'Template block', childName: 'content', childKind: vscode.SymbolKind.Property};
        case 'script':
            return {name: '<script>', kind: vscode.SymbolKind.Module, detail: 'Script block', childName: 'code', childKind: vscode.SymbolKind.Function};
        case 'style':
            return {name: '<style>', kind: vscode.SymbolKind.Namespace, detail: 'Style block', childName: 'CSS rules', childKind: vscode.SymbolKind.Property};
        case 'i18n':
            return {name: '<i18n>', kind: vscode.SymbolKind.Namespace, detail: 'Internationalisation block', childName: 'translations', childKind: vscode.SymbolKind.Property};
        case 'piko:timeline':
            return {name: '<piko:timeline>', kind: vscode.SymbolKind.Namespace, detail: 'Timeline block', childName: 'timeline', childKind: vscode.SymbolKind.Property};
    }
}

/**
 * Gets the language name from a script block's attributes.
 *
 * @param block - The script block.
 * @returns The language name, or null if not determinable.
 */
function extractScriptLanguage(block: SfcBlock): string | null {
    if (block.kind === 'script') {
        return 'go';
    }
    if (block.kind === 'script-ts') {
        return 'typescript';
    }

    const lang = getAttribute(block.attributes, 'lang');
    if (lang) {
        return lang;
    }

    const type = getAttribute(block.attributes, 'type');
    if (type?.includes('javascript')) {
        return 'js';
    }
    if (type?.includes('typescript')) {
        return 'typescript';
    }
    return null;
}

/**
 * Gets the display name for a script block based on its language.
 *
 * @param block - The script block.
 * @returns The name and child name to show in the outline.
 */
function getScriptDisplayInfo(block: SfcBlock): { name: string; childName: string } {
    const lang = extractScriptLanguage(block);

    switch (lang) {
        case null:
            return {name: '<script>', childName: 'code'};
        case 'go':
            return {name: '<script type="application/x-go">', childName: 'Go code'};
        case 'js':
        case 'javascript':
            return {name: '<script lang="js">', childName: 'JavaScript code'};
        case 'typescript':
            return {name: '<script lang="typescript">', childName: 'TypeScript code'};
        default:
            return {name: `<script lang="${lang}">`, childName: `${lang} code`};
    }
}

/**
 * Creates a document symbol for a block.
 *
 * @param document - The Piko document.
 * @param block - The parsed block.
 * @returns The document symbol for this block.
 */
function createBlockSymbol(document: vscode.TextDocument, block: SfcBlock): vscode.DocumentSymbol {
    const config = blockConfig(block.tag);
    const displayInfo = block.tag === 'script'
        ? getScriptDisplayInfo(block)
        : {name: config.name, childName: config.childName};

    const range = new vscode.Range(document.positionAt(block.start), document.positionAt(block.end));
    const symbol = new vscode.DocumentSymbol(displayInfo.name, config.detail, config.kind, range, range);

    if (block.contentEnd > block.contentStart) {
        const childRange = new vscode.Range(
            document.positionAt(block.contentStart),
            document.positionAt(block.contentEnd)
        );
        symbol.children = [
            new vscode.DocumentSymbol(displayInfo.childName, '', config.childKind, childRange, childRange)
        ];
    }

    return symbol;
}

/**
 * Creates document symbols for a Piko file.
 *
 * These symbols show in the breadcrumbs, outline panel, and go-to-symbol menu.
 *
 * @param document - The Piko document.
 * @returns The list of symbols found in the document, in source order.
 */
export function provideDocumentSymbols(document: vscode.TextDocument): vscode.DocumentSymbol[] {
    return parseSfcBlocks(document.getText()).map(block => createBlockSymbol(document, block));
}

/**
 * VS Code provider for document symbols in Piko files.
 *
 * Registered only as a fallback while pikopls is not running; the server
 * provides document symbols itself.
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
