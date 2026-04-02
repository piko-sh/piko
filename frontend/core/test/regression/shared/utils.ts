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
 * Shared utilities for regression test harnesses.
 */

import * as fs from 'fs';
import * as path from 'path';
import type { BaseTestSpec, TestCase, DOMAssertion } from './types';

/**
 * Check if we're in golden file update mode.
 * Set UPDATE_GOLDEN=true environment variable to update golden files.
 */
export function isUpdateGoldenMode(): boolean {
    return process.env.UPDATE_GOLDEN === 'true' || process.env.UPDATE_GOLDEN === '1';
}

/**
 * Golden file comparison result.
 */
export interface GoldenResult {
    passed: boolean;
    message: string;
    expected?: string;
    actual?: string;
    goldenPath?: string;
    updated?: boolean;
}

/**
 * Reads a golden file from a test case directory.
 * Returns undefined if the golden file doesn't exist.
 */
export function readGoldenFile(casePath: string, filename: string = 'golden.html'): string | undefined {
    const filePath = path.join(casePath, filename);
    if (!fs.existsSync(filePath)) {
        return undefined;
    }
    return fs.readFileSync(filePath, 'utf-8');
}

/**
 * Writes a golden file to a test case directory.
 */
export function writeGoldenFile(casePath: string, content: string, filename: string = 'golden.html'): void {
    const filePath = path.join(casePath, filename);
    fs.writeFileSync(filePath, content, 'utf-8');
}

/**
 * Formats HTML for golden file storage.
 * Pretty-prints the HTML with consistent indentation.
 */
export function formatHTMLForGolden(html: string): string {
    let formatted = html.trim();

    formatted = formatted.replace(/>\s+</g, '>\n<');

    const lines = formatted.split('\n');
    let indent = 0;
    const indentedLines: string[] = [];

    for (const line of lines) {
        const trimmed = line.trim();
        if (!trimmed) {
            continue;
        }

        if (trimmed.startsWith('</')) {
            indent = Math.max(0, indent - 1);
        }

        indentedLines.push('    '.repeat(indent) + trimmed);

        if (trimmed.startsWith('<') &&
            !trimmed.startsWith('</') &&
            !trimmed.endsWith('/>') &&
            !trimmed.startsWith('<!') &&
            !isVoidElement(trimmed)) {
            indent++;
        }

        if (trimmed.includes('</') && !trimmed.startsWith('</')) {
            indent = Math.max(0, indent - 1);
        }
    }

    return indentedLines.join('\n') + '\n';
}

/**
 * Checks if an HTML tag is a void element (self-closing).
 */
function isVoidElement(tag: string): boolean {
    const voidElements = ['area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input',
        'link', 'meta', 'param', 'source', 'track', 'wbr'];
    const match = tag.match(/^<(\w+)/);
    if (!match) {
        return false;
    }
    return voidElements.includes(match[1].toLowerCase());
}

/**
 * Compares actual HTML output against a golden file.
 * If UPDATE_GOLDEN is set, updates the golden file instead of comparing.
 */
export function compareWithGolden(
    casePath: string,
    actualHTML: string,
    goldenFilename: string = 'golden.html'
): GoldenResult {
    const goldenPath = path.join(casePath, goldenFilename);
    const formattedActual = formatHTMLForGolden(actualHTML);

    if (isUpdateGoldenMode()) {
        writeGoldenFile(casePath, formattedActual, goldenFilename);
        return {
            passed: true,
            message: `Golden file updated: ${goldenPath}`,
            actual: formattedActual,
            goldenPath,
            updated: true
        };
    }

    const expectedHTML = readGoldenFile(casePath, goldenFilename);

    if (expectedHTML === undefined) {
        return {
            passed: false,
            message: `Golden file not found: ${goldenPath}. Run with UPDATE_GOLDEN=true to create it.`,
            actual: formattedActual,
            goldenPath
        };
    }

    const formattedExpected = formatHTMLForGolden(expectedHTML);

    if (formattedActual === formattedExpected) {
        return {
            passed: true,
            message: 'Output matches golden file',
            expected: formattedExpected,
            actual: formattedActual,
            goldenPath
        };
    }

    return {
        passed: false,
        message: `Output does not match golden file: ${goldenPath}`,
        expected: formattedExpected,
        actual: formattedActual,
        goldenPath
    };
}

/**
 * Compares actual HTML against golden file with detailed diff output.
 */
export function assertGolden(
    casePath: string,
    actualHTML: string,
    goldenFilename: string = 'golden.html'
): void {
    const result = compareWithGolden(casePath, actualHTML, goldenFilename);

    if (!result.passed && !result.updated) {
        const diffMessage = generateDiff(result.expected ?? '', result.actual ?? '');
        throw new Error(`${result.message}\n\n${diffMessage}`);
    }
}

/**
 * Generates a simple line-by-line diff between two strings.
 */
function generateDiff(expected: string, actual: string): string {
    const expectedLines = expected.split('\n');
    const actualLines = actual.split('\n');
    const maxLines = Math.max(expectedLines.length, actualLines.length);

    const diffLines: string[] = ['--- Expected', '+++ Actual', ''];

    for (let i = 0; i < maxLines; i++) {
        const expectedLine = expectedLines[i] ?? '';
        const actualLine = actualLines[i] ?? '';

        if (expectedLine === actualLine) {
            diffLines.push(`  ${expectedLine}`);
        } else {
            if (expectedLines[i] !== undefined) {
                diffLines.push(`- ${expectedLine}`);
            }
            if (actualLines[i] !== undefined) {
                diffLines.push(`+ ${actualLine}`);
            }
        }
    }

    return diffLines.join('\n');
}

/**
 * Discovers all test cases in a testdata directory.
 * Test cases are directories that contain a testspec.json file.
 */
export function discoverTestCases<T extends BaseTestSpec>(
    testdataDir: string
): TestCase[] {
    const cases: TestCase[] = [];

    if (!fs.existsSync(testdataDir)) {
        return cases;
    }

    const entries = fs.readdirSync(testdataDir, { withFileTypes: true });

    for (const entry of entries) {
        if (!entry.isDirectory()) {
            continue;
        }

        const casePath = path.join(testdataDir, entry.name);
        const specPath = path.join(casePath, 'testspec.json');

        if (!fs.existsSync(specPath)) {
            continue;
        }

        try {
            const specContent = fs.readFileSync(specPath, 'utf-8');
            const spec = JSON.parse(specContent) as T;

            cases.push({
                name: entry.name,
                path: casePath,
                spec
            });
        } catch (e) {
            console.warn(`Failed to parse testspec.json for ${entry.name}:`, e);
        }
    }

    cases.sort((a, b) => a.name.localeCompare(b.name));

    return cases;
}

/**
 * Reads an HTML file from a test case directory.
 */
export function readTestFile(casePath: string, filename: string): string {
    const filePath = path.join(casePath, filename);
    if (!fs.existsSync(filePath)) {
        throw new Error(`Test file not found: ${filePath}`);
    }
    return fs.readFileSync(filePath, 'utf-8');
}

/**
 * Reads an optional HTML file from a test case directory.
 * Returns undefined if file doesn't exist.
 */
export function readOptionalTestFile(casePath: string, filename: string): string | undefined {
    const filePath = path.join(casePath, filename);
    if (!fs.existsSync(filePath)) {
        return undefined;
    }
    return fs.readFileSync(filePath, 'utf-8');
}

/**
 * Creates a DOM container for testing.
 * Returns the container and a cleanup function.
 */
export function createTestContainer(): { container: HTMLElement; cleanup: () => void } {
    const container = document.createElement('div');
    container.id = 'test-container';
    document.body.appendChild(container);

    return {
        container,
        cleanup: () => {
            if (container.parentNode) {
                container.parentNode.removeChild(container);
            }
        }
    };
}

/**
 * Sets up the DOM with initial HTML content.
 */
export function setupDOM(container: HTMLElement, html: string): void {
    container.innerHTML = html;
}

/**
 * Parses HTML string into a DocumentFragment or Element.
 */
export function parseHTML(html: string): DocumentFragment {
    const template = document.createElement('template');
    template.innerHTML = html.trim();
    return template.content;
}

/**
 * Verifies a single DOM assertion.
 */
export function verifyAssertion(
    container: HTMLElement,
    assertion: DOMAssertion
): { passed: boolean; message: string; expected?: unknown; actual?: unknown } {
    const elements = container.querySelectorAll(assertion.selector);

    if (assertion.count !== undefined) {
        if (elements.length !== assertion.count) {
            return {
                passed: false,
                message: `Expected ${assertion.count} elements matching "${assertion.selector}", found ${elements.length}`,
                expected: assertion.count,
                actual: elements.length
            };
        }
    }

    if (assertion.exists !== undefined) {
        const exists = elements.length > 0;
        if (exists !== assertion.exists) {
            return {
                passed: false,
                message: assertion.exists
                    ? `Expected element "${assertion.selector}" to exist, but it was not found`
                    : `Expected element "${assertion.selector}" NOT to exist, but it was found`,
                expected: assertion.exists,
                actual: exists
            };
        }
    }

    if (elements.length === 0 && (
        assertion.textContent !== undefined ||
        assertion.textContains !== undefined ||
        assertion.innerHTML !== undefined ||
        assertion.innerHTMLContains !== undefined ||
        assertion.attributes !== undefined ||
        assertion.styles !== undefined ||
        assertion.isVisible !== undefined ||
        assertion.hasFocus !== undefined
    )) {
        return {
            passed: false,
            message: `No elements found for selector "${assertion.selector}"`,
            expected: 'at least one element',
            actual: 0
        };
    }

    const element = elements[0] as HTMLElement;

    if (assertion.textContent !== undefined) {
        const actual = element.textContent ?? '';
        if (actual !== assertion.textContent) {
            return {
                passed: false,
                message: `Text content mismatch for "${assertion.selector}"`,
                expected: assertion.textContent,
                actual
            };
        }
    }

    if (assertion.textContains !== undefined) {
        const actual = element.textContent ?? '';
        if (!actual.includes(assertion.textContains)) {
            return {
                passed: false,
                message: `Text content for "${assertion.selector}" does not contain expected substring`,
                expected: `contains "${assertion.textContains}"`,
                actual
            };
        }
    }

    if (assertion.innerHTML !== undefined) {
        if (element.innerHTML !== assertion.innerHTML) {
            return {
                passed: false,
                message: `innerHTML mismatch for "${assertion.selector}"`,
                expected: assertion.innerHTML,
                actual: element.innerHTML
            };
        }
    }

    if (assertion.innerHTMLContains !== undefined) {
        if (!element.innerHTML.includes(assertion.innerHTMLContains)) {
            return {
                passed: false,
                message: `innerHTML for "${assertion.selector}" does not contain expected substring`,
                expected: `contains "${assertion.innerHTMLContains}"`,
                actual: element.innerHTML
            };
        }
    }

    if (assertion.attributes !== undefined) {
        for (const [attr, expectedValue] of Object.entries(assertion.attributes)) {
            const actualValue = element.getAttribute(attr);

            if (expectedValue === null) {
                if (actualValue !== null) {
                    return {
                        passed: false,
                        message: `Attribute "${attr}" should not exist on "${assertion.selector}"`,
                        expected: null,
                        actual: actualValue
                    };
                }
            } else {
                if (actualValue !== expectedValue) {
                    return {
                        passed: false,
                        message: `Attribute "${attr}" mismatch on "${assertion.selector}"`,
                        expected: expectedValue,
                        actual: actualValue
                    };
                }
            }
        }
    }

    if (assertion.styles !== undefined) {
        const computedStyle = window.getComputedStyle(element);
        for (const [prop, expectedValue] of Object.entries(assertion.styles)) {
            const actualValue = computedStyle.getPropertyValue(prop);
            if (actualValue !== expectedValue) {
                return {
                    passed: false,
                    message: `Style "${prop}" mismatch on "${assertion.selector}"`,
                    expected: expectedValue,
                    actual: actualValue
                };
            }
        }
    }

    if (assertion.isVisible !== undefined) {
        const computedStyle = window.getComputedStyle(element);
        const isHidden = computedStyle.display === 'none' ||
            computedStyle.visibility === 'hidden' ||
            computedStyle.opacity === '0';
        const isVisible = !isHidden;

        if (isVisible !== assertion.isVisible) {
            return {
                passed: false,
                message: assertion.isVisible
                    ? `Expected "${assertion.selector}" to be visible, but it is hidden`
                    : `Expected "${assertion.selector}" to be hidden, but it is visible`,
                expected: assertion.isVisible ? 'visible' : 'hidden',
                actual: isVisible ? 'visible' : 'hidden'
            };
        }
    }

    if (assertion.hasFocus !== undefined) {
        const hasFocus = document.activeElement === element;
        if (hasFocus !== assertion.hasFocus) {
            return {
                passed: false,
                message: assertion.hasFocus
                    ? `Expected "${assertion.selector}" to have focus`
                    : `Expected "${assertion.selector}" NOT to have focus`,
                expected: assertion.hasFocus,
                actual: hasFocus
            };
        }
    }

    return { passed: true, message: 'OK' };
}

/**
 * Verifies multiple DOM assertions.
 */
export function verifyAssertions(
    container: HTMLElement,
    assertions: DOMAssertion[]
): { allPassed: boolean; results: ReturnType<typeof verifyAssertion>[] } {
    const results = assertions.map(a => verifyAssertion(container, a));
    return {
        allPassed: results.every(r => r.passed),
        results
    };
}

/**
 * Simulates a DOM event on a target element.
 */
export function simulateEvent(
    container: HTMLElement,
    target: string,
    eventType: string,
    options?: { value?: string; detail?: unknown }
): void {
    const element = container.querySelector(target);
    if (!element) {
        throw new Error(`Event target not found: ${target}`);
    }

    if (options?.value !== undefined && element instanceof HTMLInputElement) {
        element.value = options.value;
    }

    let event: Event;
    if (eventType.startsWith('custom:')) {
        const customEventType = eventType.slice(7);
        event = new CustomEvent(customEventType, {
            bubbles: true,
            cancelable: true,
            detail: options?.detail
        });
    } else if (['click', 'mousedown', 'mouseup', 'mouseover', 'mouseout'].includes(eventType)) {
        event = new MouseEvent(eventType, { bubbles: true, cancelable: true });
    } else if (['input', 'change', 'focus', 'blur'].includes(eventType)) {
        event = new Event(eventType, { bubbles: true, cancelable: true });
    } else if (['keydown', 'keyup', 'keypress'].includes(eventType)) {
        event = new KeyboardEvent(eventType, { bubbles: true, cancelable: true });
    } else {
        event = new Event(eventType, { bubbles: true, cancelable: true });
    }

    element.dispatchEvent(event);
}

/**
 * Waits for a specified number of milliseconds.
 */
export function wait(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Waits for the next animation frame.
 */
export function nextFrame(): Promise<void> {
    return new Promise(resolve => requestAnimationFrame(() => resolve()));
}

/**
 * Waits for multiple animation frames (useful for batched updates).
 */
export async function nextFrames(count: number = 2): Promise<void> {
    for (let i = 0; i < count; i++) {
        await nextFrame();
    }
}

/**
 * Normalises HTML string for comparison (removes extra whitespace).
 */
export function normaliseHTML(html: string): string {
    return html
        .replace(/>\s+</g, '><')
        .replace(/\s+/g, ' ')
        .trim();
}

/**
 * Compares two HTML strings, ignoring whitespace differences.
 */
export function htmlEquals(a: string, b: string): boolean {
    return normaliseHTML(a) === normaliseHTML(b);
}
