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

import { describe, it, expect } from 'vitest';
import { discoverMorpherTestCases, runMorpherTestCase } from './harness';

describe('fragmentMorpher regression tests', () => {
    const testCases = discoverMorpherTestCases();

    if (testCases.length === 0) {
        it.skip('No test cases found in testdata directory', () => {});
        return;
    }

    for (const tc of testCases) {
        describe(tc.name, () => {
            it(tc.spec.description, () => {
                const result = runMorpherTestCase(tc);

                if (!result.passed) {
                    console.log('\n--- Test Case Failed ---');
                    console.log(`Name: ${tc.name}`);
                    console.log(`Description: ${tc.spec.description}`);
                    console.log('\nErrors:');
                    result.errors.forEach((err, i) => {
                        console.log(`  ${i + 1}. ${err}`);
                    });

                    if (result.assertionResults.length > 0) {
                        console.log('\nAssertion Results:');
                        result.assertionResults.forEach(ar => {
                            const status = ar.passed ? '✓' : '✗';
                            console.log(`  ${status} ${ar.assertion.selector}: ${ar.message}`);
                            if (!ar.passed) {
                                console.log(`      Expected: ${JSON.stringify(ar.expected)}`);
                                console.log(`      Actual: ${JSON.stringify(ar.actual)}`);
                            }
                        });
                    }

                    if (result.preservationResults.length > 0) {
                        console.log('\nPreservation Results:');
                        result.preservationResults.forEach(pr => {
                            const status = pr.passed ? '✓' : '✗';
                            console.log(`  ${status} ${pr.description}: ${pr.message}`);
                        });
                    }

                    console.log('--- End Test Case ---\n');
                }

                expect(result.errors).toEqual([]);
                expect(result.passed).toBe(true);
            });
        });
    }
});
