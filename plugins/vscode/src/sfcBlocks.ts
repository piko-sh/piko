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

/** The top-level tag names recognised as Piko blocks. */
export type SfcBlockTag = 'template' | 'script' | 'style' | 'i18n' | 'piko:timeline';

/**
 * The classified kind of a top-level block.
 *
 * `script` is a Go script, `script-ts` a TypeScript script and `script-js` any
 * other script (JavaScript or an unrecognised type).
 */
export type SfcBlockKind = 'template' | 'script' | 'script-ts' | 'script-js' | 'style' | 'i18n' | 'timeline';

/**
 * Describes one top-level block in a Piko file.
 *
 * All offsets are UTF-16 code unit offsets into the source text, which is the
 * unit VS Code uses for `TextDocument.positionAt`.
 */
export interface SfcBlock {
    /** The lower-case tag name. */
    tag: SfcBlockTag;
    /** The classified block kind. */
    kind: SfcBlockKind;
    /** The attributes on the opening tag, keyed by lower-case name. */
    attributes: Readonly<Record<string, string>>;
    /** The offset of the `<` that opens the block. */
    start: number;
    /** The offset just after the `>` of the opening tag. */
    contentStart: number;
    /** The offset of the `</` that closes the block (equals `end` when unclosed). */
    contentEnd: number;
    /** The offset just after the `>` of the closing tag. */
    end: number;
    /** The raw content between the opening and closing tags. */
    content: string;
    /** Whether the opening tag was self-closing (`<script src="x" />`). */
    selfClosing: boolean;
}

/** MIME type values that mark a script as Go. */
const GO_MIME_TYPES = new Set(['application/x-go', 'application/go']);

/** `lang` values that mark a script as Go. */
const GO_LANG_VALUES = new Set(['go', 'golang', 'application/x-go', 'application/go']);

/** `lang` values that mark a script as TypeScript. */
const TS_LANG_VALUES = new Set(['ts', 'typescript']);

/** The MIME type that marks a script as TypeScript (compared case-insensitively). */
const TS_MIME_TYPE = 'application/typescript';

/** Tag names whose content is raw text (not tokenised) inside nestable blocks. */
const NESTED_RAW_TEXT_TAGS = new Set(['script', 'style', 'textarea', 'title', 'xmp', 'iframe']);

/** HTML void elements, which never have content or a closing tag. */
export const VOID_ELEMENTS: ReadonlySet<string> = new Set([
    'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input',
    'link', 'meta', 'param', 'source', 'track', 'wbr'
]);

/** Block tags whose content is raw text at the top level. */
const RAW_BLOCK_TAGS = new Set<SfcBlockTag>(['script', 'style']);

/** All recognised top-level block tags. */
const BLOCK_TAGS = new Set<string>(['template', 'script', 'style', 'i18n', 'piko:timeline']);

/** Length of the `<!--` comment opener. */
const COMMENT_OPEN_LENGTH = 4;

/** Length of the `-->` comment closer. */
const COMMENT_CLOSE_LENGTH = 3;

/**
 * The most top-level blocks one parse returns. Real files have a handful; the
 * cap keeps a malformed or hostile file from producing a huge outline.
 */
export const MAX_SFC_BLOCKS = 10_000;

/** The most tags one markup scan visits before it stops. */
export const MAX_MARKUP_TAGS = 200_000;

/**
 * Describes one tag found while scanning markup.
 */
export interface MarkupTag {
    /** Whether the tag opens or closes an element. */
    kind: 'open' | 'close';
    /** The lower-case tag name. */
    name: string;
    /** The offset of the `<` that starts the tag. */
    start: number;
    /** The offset just after the `>` that ends the tag. */
    end: number;
    /** Whether an opening tag ended with `/>`. */
    selfClosing: boolean;
}

/**
 * Holds the result of scanning one opening tag.
 */
interface OpenTag {
    /** The tag name as written. */
    name: string;
    /** The parsed attributes, keyed by lower-case name. */
    attributes: Record<string, string>;
    /** The offset just after the closing `>` (or the end of the text). */
    end: number;
    /** Whether the tag ended with `/>`. */
    selfClosing: boolean;
    /** Whether a closing `>` was found before the end of the text. */
    terminated: boolean;
}

/**
 * Reads an attribute value.
 *
 * @param attributes - The tag attributes keyed by lower-case name.
 * @param name - The lower-case attribute name.
 * @returns The value, or undefined when the attribute is absent.
 */
export function getAttribute(attributes: Readonly<Record<string, string>>, name: string): string | undefined {
    return Object.prototype.hasOwnProperty.call(attributes, name) ? attributes[name] : undefined;
}

/**
 * Reports whether a script's attributes mark it as Go.
 *
 * Mirrors `Script.IsGo` in internal/sfcparser/dto.go.
 *
 * @param attributes - The script tag attributes keyed by lower-case name.
 * @returns True when the script is Go.
 */
export function isGoScript(attributes: Readonly<Record<string, string>>): boolean {
    const type = getAttribute(attributes, 'type');
    if (type !== undefined && GO_MIME_TYPES.has(type)) {
        return true;
    }
    const lang = getAttribute(attributes, 'lang');
    return lang !== undefined && GO_LANG_VALUES.has(lang);
}

/**
 * Reports whether a script's attributes mark it as TypeScript.
 *
 * Mirrors `Script.IsTypeScript` in internal/sfcparser/dto.go.
 *
 * @param attributes - The script tag attributes keyed by lower-case name.
 * @returns True when the script is TypeScript.
 */
export function isTypeScriptScript(attributes: Readonly<Record<string, string>>): boolean {
    const lang = getAttribute(attributes, 'lang');
    if (lang !== undefined && TS_LANG_VALUES.has(lang)) {
        return true;
    }
    return getAttribute(attributes, 'type')?.toLowerCase() === TS_MIME_TYPE;
}

/**
 * Classifies a block from its tag name and attributes.
 *
 * @param tag - The lower-case block tag name.
 * @param attributes - The opening tag attributes.
 * @returns The block kind.
 */
export function classifyBlock(tag: SfcBlockTag, attributes: Readonly<Record<string, string>>): SfcBlockKind {
    switch (tag) {
        case 'script':
            if (isGoScript(attributes)) {
                return 'script';
            }
            return isTypeScriptScript(attributes) ? 'script-ts' : 'script-js';
        case 'piko:timeline':
            return 'timeline';
        default:
            return tag;
    }
}

/**
 * Parses the top-level blocks of a Piko file.
 *
 * @param text - The full source text.
 * @returns The blocks in source order.
 */
export function parseSfcBlocks(text: string): SfcBlock[] {
    const blocks: SfcBlock[] = [];
    let templateSeen = false;
    let index = 0;

    while (index < text.length && blocks.length < MAX_SFC_BLOCKS) {
        const lt = text.indexOf('<', index);
        if (lt === -1) {
            break;
        }

        const skipTo = skipNonStartTag(text, lt);
        if (skipTo !== -1) {
            index = skipTo;
            continue;
        }

        const tag = scanOpenTag(text, lt);
        if (tag === undefined) {
            index = lt + 1;
            continue;
        }
        if (!tag.terminated) {
            break;
        }

        const lowerName = tag.name.toLowerCase();
        if (!BLOCK_TAGS.has(lowerName)) {
            index = tag.end;
            continue;
        }

        const blockTag = lowerName as SfcBlockTag;
        if (blockTag === 'template' && templateSeen) {
            index = tag.selfClosing ? tag.end : findNestableEnd(text, blockTag, tag.end).end;
            continue;
        }
        if (blockTag === 'template') {
            templateSeen = true;
        }

        const block = buildBlock(text, blockTag, lt, tag);
        blocks.push(block);
        index = Math.max(block.end, lt + 1);
    }

    return blocks;
}

/**
 * Finds the innermost block containing an offset.
 *
 * The range is inclusive of both the opening and closing tags.
 *
 * @param blocks - The parsed blocks.
 * @param offset - The offset to look up.
 * @returns The containing block, or undefined when outside every block.
 */
export function findBlockAtOffset(blocks: readonly SfcBlock[], offset: number): SfcBlock | undefined {
    return blocks.find(block => offset >= block.start && offset <= block.end);
}

/**
 * Visits the opening and closing tags in a range of markup, in source order.
 *
 * Comments, doctypes and the content of raw-text elements (such as a nested
 * `<script>` or `<style>`) are skipped. The scan runs in linear time, stops at
 * the first unterminated tag and visits at most `MAX_MARKUP_TAGS` tags.
 *
 * @param text - The full source text.
 * @param start - The offset to start scanning at.
 * @param end - The offset at which no new tag may start.
 * @param visit - Called for each tag; returning false stops the scan.
 */
export function scanMarkupTags(text: string, start: number, end: number, visit: (tag: MarkupTag) => boolean): void {
    let index = start;
    let visited = 0;

    while (index < end && visited < MAX_MARKUP_TAGS) {
        const lt = text.indexOf('<', index);
        if (lt === -1 || lt >= end) {
            return;
        }

        if (text.charAt(lt + 1) === '/') {
            const endTag = scanEndTag(text, lt);
            visited++;
            if (!visit({kind: 'close', name: endTag.name, start: lt, end: endTag.end, selfClosing: false})) {
                return;
            }
            index = endTag.end;
            continue;
        }

        const skipTo = skipNonStartTag(text, lt);
        if (skipTo !== -1) {
            index = skipTo;
            continue;
        }

        const openTag = scanOpenTag(text, lt);
        if (openTag === undefined) {
            index = lt + 1;
            continue;
        }
        if (!openTag.terminated) {
            return;
        }

        const name = openTag.name.toLowerCase();
        visited++;
        if (!visit({kind: 'open', name, start: lt, end: openTag.end, selfClosing: openTag.selfClosing})) {
            return;
        }
        index = openTag.end;
        if (!openTag.selfClosing && NESTED_RAW_TEXT_TAGS.has(name)) {
            index = skipRawText(text, name, index);
        }
    }
}

/**
 * Builds a block record from an opening tag.
 *
 * @param text - The full source text.
 * @param tag - The lower-case block tag.
 * @param start - The offset of the opening `<`.
 * @param openTag - The scanned opening tag.
 * @returns The block record.
 */
function buildBlock(text: string, tag: SfcBlockTag, start: number, openTag: OpenTag): SfcBlock {
    const kind = classifyBlock(tag, openTag.attributes);
    if (openTag.selfClosing) {
        return {
            tag, kind, attributes: openTag.attributes, start,
            contentStart: openTag.end, contentEnd: openTag.end, end: openTag.end,
            content: '', selfClosing: true,
        };
    }

    const bounds = RAW_BLOCK_TAGS.has(tag)
        ? findRawTextEnd(text, tag, openTag.end)
        : findNestableEnd(text, tag, openTag.end);

    return {
        tag, kind, attributes: openTag.attributes, start,
        contentStart: openTag.end,
        contentEnd: bounds.contentEnd,
        end: bounds.end,
        content: text.slice(openTag.end, bounds.contentEnd),
        selfClosing: false,
    };
}

/**
 * Skips markup at `lt` that cannot open a block: comments, end tags,
 * doctype/processing instructions and stray `<` characters.
 *
 * @param text - The full source text.
 * @param lt - The offset of a `<`.
 * @returns The offset to resume from, or -1 when `lt` starts an opening tag.
 */
function skipNonStartTag(text: string, lt: number): number {
    if (text.startsWith('<!--', lt)) {
        const close = text.indexOf('-->', lt + COMMENT_OPEN_LENGTH);
        return close === -1 ? text.length : close + COMMENT_CLOSE_LENGTH;
    }
    const next = text.charAt(lt + 1);
    if (next === '/' || next === '!' || next === '?') {
        const gt = text.indexOf('>', lt + 1);
        return gt === -1 ? text.length : gt + 1;
    }
    return -1;
}

/**
 * Scans an opening tag starting at `lt`, tolerating any attributes.
 *
 * Attribute values may be double-quoted, single-quoted or unquoted, and quoted
 * values may contain `>`.
 *
 * @param text - The full source text.
 * @param lt - The offset of the `<`.
 * @returns The scanned tag, or undefined when `lt` does not start a tag name.
 */
function scanOpenTag(text: string, lt: number): OpenTag | undefined {
    let index = lt + 1;
    if (!/[A-Za-z]/.test(text.charAt(index))) {
        return undefined;
    }
    while (index < text.length && !/[\s/>]/.test(text.charAt(index))) {
        index++;
    }
    const name = text.slice(lt + 1, index);
    const attributes: Record<string, string> = {};

    while (index < text.length) {
        const char = text.charAt(index);
        if (char === '>') {
            return {name, attributes, end: index + 1, selfClosing: false, terminated: true};
        }
        if (char === '/' && text.charAt(index + 1) === '>') {
            return {name, attributes, end: index + 2, selfClosing: true, terminated: true};
        }
        if (/[\s/]/.test(char)) {
            index++;
            continue;
        }
        index = scanAttribute(text, index, attributes);
    }

    return {name, attributes, end: text.length, selfClosing: false, terminated: false};
}

/**
 * Scans one attribute and records it.
 *
 * @param text - The full source text.
 * @param index - The offset of the attribute name.
 * @param attributes - The map to record the attribute in.
 * @returns The offset after the attribute.
 */
function scanAttribute(text: string, index: number, attributes: Record<string, string>): number {
    const nameStart = index;
    while (index < text.length && !/[\s=>]/.test(text.charAt(index)) && !text.startsWith('/>', index)) {
        index++;
    }
    const name = text.slice(nameStart, index).toLowerCase();

    let afterName = index;
    while (afterName < text.length && /\s/.test(text.charAt(afterName))) {
        afterName++;
    }
    if (text.charAt(afterName) !== '=') {
        attributes[name] = '';
        return index;
    }

    let valueStart = afterName + 1;
    while (valueStart < text.length && /\s/.test(text.charAt(valueStart))) {
        valueStart++;
    }

    const quote = text.charAt(valueStart);
    if (quote === '"' || quote === "'") {
        const close = text.indexOf(quote, valueStart + 1);
        const valueEnd = close === -1 ? text.length : close;
        attributes[name] = text.slice(valueStart + 1, valueEnd);
        return close === -1 ? text.length : close + 1;
    }

    let valueEnd = valueStart;
    while (valueEnd < text.length && !/[\s>]/.test(text.charAt(valueEnd))) {
        valueEnd++;
    }
    attributes[name] = text.slice(valueStart, valueEnd);
    return valueEnd;
}

/**
 * Finds the end of a raw-text block (script or style).
 *
 * Mirrors `findRawTextEndOffset` and `determineSearchRange` in
 * internal/sfcparser/sfc.go: the content ends at the last `</tag` before the
 * next `<tag`, falling back to the last `</tag` anywhere and then to the end
 * of the text.
 *
 * @param text - The full source text.
 * @param tag - The lower-case tag name.
 * @param contentStart - The offset just after the opening tag.
 * @returns The content end and block end offsets.
 */
function findRawTextEnd(text: string, tag: string, contentStart: number): { contentEnd: number; end: number } {
    const endTag = `</${tag}`;
    const nextStart = text.indexOf(`<${tag}`, contentStart);

    let contentEnd = -1;
    if (nextStart === -1) {
        contentEnd = text.lastIndexOf(endTag);
    } else if (nextStart - endTag.length >= contentStart) {
        contentEnd = text.lastIndexOf(endTag, nextStart - endTag.length);
    }
    if (contentEnd < contentStart) {
        contentEnd = text.lastIndexOf(endTag);
    }
    if (contentEnd < contentStart) {
        return {contentEnd: text.length, end: text.length};
    }

    const gt = text.indexOf('>', contentEnd);
    return {contentEnd, end: gt === -1 ? text.length : gt + 1};
}

/**
 * Finds the end of a nestable block (template, i18n or piko:timeline).
 *
 * Tracks the depth of same-named tags, skipping comments and the content of
 * raw-text elements, as the server's lexer does.
 *
 * @param text - The full source text.
 * @param tag - The lower-case tag name.
 * @param contentStart - The offset just after the opening tag.
 * @returns The content end and block end offsets.
 */
function findNestableEnd(text: string, tag: string, contentStart: number): { contentEnd: number; end: number } {
    let depth = 1;
    let index = contentStart;

    while (index < text.length) {
        const lt = text.indexOf('<', index);
        if (lt === -1) {
            break;
        }

        if (text.startsWith('<!--', lt)) {
            const close = text.indexOf('-->', lt + COMMENT_OPEN_LENGTH);
            index = close === -1 ? text.length : close + COMMENT_CLOSE_LENGTH;
            continue;
        }

        if (text.charAt(lt + 1) === '/') {
            const endTag = scanEndTag(text, lt);
            if (endTag.name === tag) {
                depth--;
            }
            if (depth === 0) {
                return {contentEnd: lt, end: endTag.end};
            }
            index = endTag.end;
            continue;
        }

        const openTag = scanOpenTag(text, lt);
        if (openTag === undefined) {
            index = lt + 1;
            continue;
        }

        const name = openTag.name.toLowerCase();
        if (name === tag) {
            depth++;
        }
        index = openTag.end;
        if (!openTag.selfClosing && NESTED_RAW_TEXT_TAGS.has(name)) {
            index = skipRawText(text, name, index);
        }
    }

    return {contentEnd: text.length, end: text.length};
}

/**
 * Skips the content of a raw-text element nested inside a block.
 *
 * @param text - The full source text.
 * @param name - The lower-case raw-text tag name.
 * @param index - The offset just after the element's opening tag.
 * @returns The offset of the element's closing tag (or the end of the text).
 */
function skipRawText(text: string, name: string, index: number): number {
    const pattern = new RegExp(`</${name}`, 'gi');
    pattern.lastIndex = index;
    const match = pattern.exec(text);
    return match === null ? text.length : match.index;
}

/**
 * Scans an end tag starting at `lt`.
 *
 * @param text - The full source text.
 * @param lt - The offset of the `<` of `</name>`.
 * @returns The lower-case tag name and the offset after the closing `>`.
 */
function scanEndTag(text: string, lt: number): { name: string; end: number } {
    const nameEnd = readTagNameEnd(text, lt + 2);
    const gt = text.indexOf('>', nameEnd);
    return {name: text.slice(lt + 2, nameEnd).toLowerCase(), end: gt === -1 ? text.length : gt + 1};
}

/**
 * Reads past a tag name.
 *
 * @param text - The full source text.
 * @param index - The offset of the first name character.
 * @returns The offset just after the name.
 */
function readTagNameEnd(text: string, index: number): number {
    while (index < text.length && !/[\s/>]/.test(text.charAt(index))) {
        index++;
    }
    return index;
}

/**
 * Finds the nearest entry in a stack of open elements with a tag name.
 *
 * @param stack - The open elements, outermost first.
 * @param name - The lower-case tag name.
 * @returns The index in the stack, or -1 when no entry has the name.
 */
export function findLastTagIndex(stack: readonly { name: string }[], name: string): number {
    for (let index = stack.length - 1; index >= 0; index--) {
        if (stack[index].name === name) {
            return index;
        }
    }
    return -1;
}
