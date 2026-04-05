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
 * Fragment Morpher Regression Test Harness
 *
 * This harness tests the fragmentMorpher function which morphs one DOM tree
 * into another, preserving nodes where possible.
 */

import fragmentMorpher from '../../../src/core/fragmentMorpher';
import type { FragmentMorpherTestSpec, DOMAssertion } from '../shared/types';
import {
    discoverTestCases,
    readTestFile,
    createTestContainer,
    verifyAssertion,
    parseHTML,
    compareWithGolden,
    isUpdateGoldenMode,
    type GoldenResult
} from '../shared/utils';
import * as path from 'path';

const TESTDATA_DIR = path.join(__dirname, 'testdata');

export interface MorpherTestCase {
    name: string;
    path: string;
    spec: FragmentMorpherTestSpec;
    sourceHTML: string;
    targetHTML: string;
}

/**
 * Discovers all fragmentMorpher test cases.
 */
export function discoverMorpherTestCases(): MorpherTestCase[] {
    const baseCases = discoverTestCases<FragmentMorpherTestSpec>(TESTDATA_DIR);

    return baseCases.map(tc => ({
        ...tc,
        spec: tc.spec as FragmentMorpherTestSpec,
        sourceHTML: readTestFile(tc.path, 'source.html'),
        targetHTML: readTestFile(tc.path, 'target.html')
    }));
}

/**
 * Result of running a morpher test.
 */
export interface MorpherTestResult {
    passed: boolean;
    errors: string[];
    assertionResults: {
        assertion: DOMAssertion;
        passed: boolean;
        message: string;
        expected?: unknown;
        actual?: unknown;
    }[];
    preservationResults: {
        description: string;
        passed: boolean;
        message: string;
    }[];
    goldenResult?: GoldenResult;
    finalHTML?: string;
}

/**
 * Runs a single fragmentMorpher test case.
 */
export function runMorpherTestCase(tc: MorpherTestCase): MorpherTestResult {
    const errors: string[] = [];
    const assertionResults: MorpherTestResult['assertionResults'] = [];
    const preservationResults: MorpherTestResult['preservationResults'] = [];

    const { container, cleanup } = createTestContainer();

    try {
        container.innerHTML = tc.sourceHTML;

        const preservedNodeRefs: Map<string, Node> = new Map();
        if (tc.spec.preservedNodes) {
            for (const pn of tc.spec.preservedNodes) {
                const node = container.querySelector(pn.sourceSelector);
                if (node) {
                    preservedNodeRefs.set(pn.sourceSelector, node);
                }
            }
        }

        const targetFragment = parseHTML(tc.targetHTML);
        const targetElement = targetFragment.firstElementChild as HTMLElement;

        if (!targetElement) {
            throw new Error('Target HTML must contain at least one element');
        }

        const sourceElement = container.firstElementChild as HTMLElement;
        if (!sourceElement) {
            throw new Error('Source HTML must contain at least one element');
        }

        fragmentMorpher(sourceElement, targetElement, {
            childrenOnly: tc.spec.options?.childrenOnly ?? false,
            preservePartialScopes: tc.spec.options?.preservePartialScopes
        });

        for (const assertion of tc.spec.assertions) {
            const result = verifyAssertion(container, assertion);
            assertionResults.push({
                assertion,
                passed: result.passed,
                message: result.message,
                expected: result.expected,
                actual: result.actual
            });

            if (!result.passed) {
                errors.push(result.message);
            }
        }

        if (tc.spec.preservedNodes) {
            for (const pn of tc.spec.preservedNodes) {
                const originalNode = preservedNodeRefs.get(pn.sourceSelector);
                const currentNode = container.querySelector(pn.sourceSelector);

                if (!originalNode) {
                    preservationResults.push({
                        description: pn.description,
                        passed: false,
                        message: `Original node for "${pn.sourceSelector}" was not found before morph`
                    });
                } else if (!currentNode) {
                    preservationResults.push({
                        description: pn.description,
                        passed: false,
                        message: `Node "${pn.sourceSelector}" was not preserved (not found after morph)`
                    });
                } else if (originalNode !== currentNode) {
                    preservationResults.push({
                        description: pn.description,
                        passed: false,
                        message: `Node "${pn.sourceSelector}" was replaced (different reference after morph)`
                    });
                } else {
                    preservationResults.push({
                        description: pn.description,
                        passed: true,
                        message: 'Node preserved correctly'
                    });
                }

                if (!preservationResults[preservationResults.length - 1].passed) {
                    errors.push(preservationResults[preservationResults.length - 1].message);
                }
            }
        }

        if (tc.spec.removedNodes) {
            for (const rn of tc.spec.removedNodes) {
                const node = container.querySelector(rn.selector);
                if (node) {
                    errors.push(`Node "${rn.selector}" should have been removed: ${rn.description}`);
                }
            }
        }

        const finalHTML = container.innerHTML;

        const goldenResult = compareWithGolden(tc.path, finalHTML, 'golden.html');
        if (!goldenResult.passed && !goldenResult.updated) {
            errors.push(goldenResult.message);
        }

        return {
            passed: errors.length === 0,
            errors,
            assertionResults,
            preservationResults,
            goldenResult,
            finalHTML
        };

    } catch (e) {
        const error = e as Error;
        if (tc.spec.shouldError) {
            if (tc.spec.expectedErrorContains) {
                if (!error.message.includes(tc.spec.expectedErrorContains)) {
                    errors.push(`Expected error to contain "${tc.spec.expectedErrorContains}" but got: ${error.message}`);
                }
            }
        } else {
            errors.push(`Unexpected error: ${error.message}`);
        }
    } finally {
        cleanup();
    }

    return {
        passed: errors.length === 0,
        errors,
        assertionResults,
        preservationResults
    };
}
