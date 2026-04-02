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
 * Extracts p-ref directive names from template HTML content and generates
 * TypeScript declarations for the `pk` context object.
 *
 * Enables autocompletion for `pk.refs.refName` and `pk.onConnected()`
 * etc. in script blocks by providing per-file type information based on
 * the template's p-ref attributes.
 */

/**
 * Regex to extract p-ref attribute values from template HTML content.
 *
 * Matches `p-ref="name"` or `p-ref='name'` where name is a valid JS
 * identifier. The first capturing group extracts the ref name.
 */
const P_REF_REGEX = /p-ref\s*=\s*["']([a-zA-Z_$][a-zA-Z0-9_$]*)["']/g;

/**
 * Extracts unique p-ref names from template HTML content.
 *
 * @param templateContent - The raw HTML content of the template block.
 * @returns Ordered list of unique p-ref names found in the template.
 */
export function extractPRefNames(templateContent: string): string[] {
    const names: string[] = [];
    const seen = new Set<string>();

    P_REF_REGEX.lastIndex = 0;
    let match: RegExpExecArray | null;
    while ((match = P_REF_REGEX.exec(templateContent)) !== null) {
        const name = match[1];
        if (!seen.has(name)) {
            seen.add(name);
            names.push(name);
        }
    }

    return names;
}

/**
 * Namespace member declarations for the `pk` context object.
 *
 * These are constant across all PK files - only the `refs` property varies.
 * Using `declare namespace` rather than `declare const` gives IntelliJ
 * proper autocomplete on `pk.` in injected TypeScript fragments.
 */
const PK_NAMESPACE_MEMBERS = [
    '/** Register a callback invoked when the element connects to the DOM. */ function onConnected(cb: () => void): void',
    '/** Register a callback invoked when the element disconnects from the DOM. */ function onDisconnected(cb: () => void): void',
    '/** Register a callback invoked before the template is rendered. */ function onBeforeRender(cb: () => void): void',
    '/** Register a callback invoked after the template is rendered. */ function onAfterRender(cb: () => void): void',
    '/** Register a callback invoked when the partial is updated. */ function onUpdated(cb: (context?: unknown) => void): void',
    '/** Register a cleanup function to run when the element is removed. */ function onCleanup(fn: () => void): void',
].join('; ');

/**
 * Generates a TypeScript `pk` namespace declaration from extracted ref names.
 *
 * Produces a `declare namespace pk` with a `refs` const containing each ref
 * name typed as `HTMLElement | null`, plus lifecycle function declarations.
 *
 * @param refNames - The list of ref names to include.
 * @returns The TypeScript declaration string (always non-empty, single line).
 */
export function generatePKDeclaration(refNames: string[]): string {
    const refsType = buildRefsType(refNames);
    return `declare namespace pk { const refs: ${refsType}; ${PK_NAMESPACE_MEMBERS}; }`;
}

/**
 * Interface member declarations for the PKC component instance.
 *
 * These are the user-facing methods available on `pkc` (an alias for `this`
 * in PKC web components, which extends HTMLElement via PPElement).
 * Declared as interface members (not namespace members) so that `pkc` also
 * exposes all standard HTMLElement methods like addEventListener, querySelector, etc.
 */
const PKC_INTERFACE_MEMBERS = [
    '/** Reactive state object for the component. */ state: Record<string, unknown> | undefined',
    '/** Merge partial state and schedule a re-render. */ setState(partialState: Record<string, unknown>): void',
    '/** Immediately re-render the component. */ render(): void',
    '/** Schedule a re-render on the next microtask. */ scheduleRender(): void',
    '/** Register a callback invoked when the component connects to the DOM. */ onConnected(cb: () => void): void',
    '/** Register a callback invoked when the component disconnects from the DOM. */ onDisconnected(cb: () => void): void',
    '/** Register a callback invoked before the component renders. */ onBeforeRender(cb: () => void): void',
    '/** Register a callback invoked after the component renders. */ onAfterRender(cb: () => void): void',
    '/** Register a callback invoked when observed attributes change. */ onUpdated(cb: (changedProperties: Set<string>) => void): void',
    '/** Register a cleanup function to run when the component is removed. */ onCleanup(cb: () => void): void',
    '/** Attaches a listener for slot content changes. The callback is invoked immediately with initial content. */ attachSlotListener(slotName: string, callback: (elements: Element[]) => void): void',
    '/** Returns elements assigned to a named slot. */ getSlottedElements(slotName?: string): Element[]',
    '/** Checks whether a slot has any assigned content. */ hasSlotContent(slotName?: string): boolean',
].join('; ');

/**
 * Generates a TypeScript declaration for the `pkc` component instance.
 *
 * Produces an interface extending HTMLElement with per-file refs and component
 * instance methods, then declares `pkc` as an instance of that interface.
 * This enables autocompletion for both custom members (`pkc.refs.refName`,
 * `pkc.setState()`) and inherited HTMLElement methods (`pkc.addEventListener()`,
 * `pkc.removeEventListener()`, etc.) in `.pkc` script blocks.
 *
 * @param refNames - The list of ref names to include.
 * @returns The TypeScript declaration string (always non-empty, single line).
 */
export function generatePKCDeclaration(refNames: string[]): string {
    const refsType = buildRefsType(refNames);
    return `interface _PikoComponent extends HTMLElement { readonly refs: ${refsType}; ${PKC_INTERFACE_MEMBERS}; } declare const pkc: _PikoComponent;`;
}

/**
 * Builds the refs type object from extracted ref names.
 *
 * @param refNames - The list of ref names to include.
 * @returns The refs type string (e.g. `{ readonly foo: HTMLElement | null }` or `{}`).
 */
function buildRefsType(refNames: string[]): string {
    const refsFields = refNames.map(name => `readonly ${name}: HTMLElement | null`).join('; ');
    return refsFields ? `{ ${refsFields} }` : '{}';
}
