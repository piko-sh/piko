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
import { discoverVDOMTestCases, runVDOMTestCase } from './harness';

describe('VDOM regression tests', () => {
    const testCases = discoverVDOMTestCases();

    if (testCases.length === 0) {
        it.skip('no test cases found', () => {});
        return;
    }

    for (const tc of testCases) {
        describe(tc.name, () => {
            it(tc.spec.description, async () => {
                const result = await runVDOMTestCase(tc);

                if (!result.passed) {
                    const errorMessages = result.errors.join('\n');
                    expect.fail(`Test failed:\n${errorMessages}`);
                }

                expect(result.passed).toBe(true);
            });
        });
    }
});
