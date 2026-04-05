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
 * VDOM Regression Test Harness
 *
 * This harness tests the VDOM patching system by simulating component
 * state changes and verifying the resulting DOM updates.
 */

import { PPElement } from '../../../src/element';
import { dom, type VirtualNode } from '../../../src/vdom';
import { makeReactive } from '../../../src/reactivity';
import type { VDOMTestSpec, VDOMTestStep, DOMAssertion } from '../../../../../core/test/regression/shared/types';
import {
    discoverTestCases,
    createTestContainer,
    verifyAssertion,
    simulateEvent,
    nextFrames,
    compareWithGolden,
    type GoldenResult
} from '../../../../../core/test/regression/shared/utils';
import * as path from 'path';

const TESTDATA_DIR = path.join(__dirname, 'testdata');

export interface VDOMTestCase {
    name: string;
    path: string;
    spec: VDOMTestSpec;
}

/**
 * Discovers all VDOM test cases.
 */
export function discoverVDOMTestCases(): VDOMTestCase[] {
    const baseCases = discoverTestCases<VDOMTestSpec>(TESTDATA_DIR);

    return baseCases.map(tc => ({
        ...tc,
        spec: tc.spec as VDOMTestSpec
    }));
}

/**
 * Result of running a VDOM test.
 */
export interface VDOMTestResult {
    passed: boolean;
    errors: string[];
    stepResults: {
        step: VDOMTestStep;
        stepIndex: number;
        passed: boolean;
        assertionResults: {
            assertion: DOMAssertion;
            passed: boolean;
            message: string;
            expected?: unknown;
            actual?: unknown;
        }[];
        goldenResult?: GoldenResult;
        finalHTML?: string;
    }[];
}

// Counter for unique component names
let componentCounter = 0;

/**
 * Creates a test component dynamically based on the test spec.
 */
function createTestComponent(
    spec: VDOMTestSpec
): { tagName: string; ComponentClass: typeof PPElement } {
    const tagName = `vdom-test-component-${++componentCounter}`;

    class TestComponent extends PPElement {
        public currentVDOM: VirtualNode | null = null;

        constructor() {
            super();
            const initialState = { ...spec.initialState };
            this.init({ state: makeReactive(initialState, this as PPElement) });
        }

        override renderVDOM(): VirtualNode {
            if (this.currentVDOM) {
                return this.currentVDOM;
            }
            return evaluateVDOMExpression(spec.initialVDOM, this.state as Record<string, unknown>);
        }

        public setVDOM(vdomExpr: string): void {
            this.currentVDOM = evaluateVDOMExpression(vdomExpr, this.state as Record<string, unknown>);
            this.render();
        }

        public updateState(changes: Record<string, unknown>): void {
            for (const [key, value] of Object.entries(changes)) {
                setNestedValue(this.state as Record<string, unknown>, key, value);
            }
        }
    }

    if (!customElements.get(tagName)) {
        customElements.define(tagName, TestComponent);
    }

    return { tagName, ComponentClass: TestComponent as typeof PPElement };
}

/**
 * Sets a nested value in an object using dot notation path.
 */
function setNestedValue(target: Record<string, unknown>, path: string, value: unknown): void {
    const parts = path.split('.');
    let current = target;

    for (let i = 0; i < parts.length - 1; i++) {
        const part = parts[i];
        if (!(part in current)) {
            current[part] = {};
        }
        current = current[part] as Record<string, unknown>;
    }

    current[parts[parts.length - 1]] = value;
}

/**
 * Evaluates a VDOM expression string into a VirtualNode.
 *
 * The expression can use:
 * - dom.el(tag, key, props, children)
 * - dom.txt(text, key)
 * - dom.frag(key, children, props)
 * - dom.cmt(text, key)
 * - dom.ws(key)
 * - state.propertyName for state access
 */
function evaluateVDOMExpression(
    expr: string,
    state: Record<string, unknown>
): VirtualNode {
    const evaluator = new Function('dom', 'state', `return ${expr};`);
    return evaluator(dom, state);
}

/**
 * Gets the rendered content from a shadow root, excluding style tags.
 */
function getRenderedContent(shadowRoot: ShadowRoot): string {
    let html = shadowRoot.innerHTML;
    html = html.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, '');
    return html.trim();
}

/**
 * Runs a single VDOM test case.
 */
export async function runVDOMTestCase(tc: VDOMTestCase): Promise<VDOMTestResult> {
    const errors: string[] = [];
    const stepResults: VDOMTestResult['stepResults'] = [];

    const { container, cleanup } = createTestContainer();

    try {
        const { tagName, ComponentClass } = createTestComponent(tc.spec);

        const component = document.createElement(tagName) as InstanceType<typeof ComponentClass>;
        container.appendChild(component);

        await nextFrames(2);

        for (let stepIndex = 0; stepIndex < tc.spec.steps.length; stepIndex++) {
            const step = tc.spec.steps[stepIndex];
            const stepAssertionResults: VDOMTestResult['stepResults'][0]['assertionResults'] = [];

            if (step.stateChanges) {
                for (const change of step.stateChanges) {
                    setNestedValue(
                        component.state as Record<string, unknown>,
                        change.path,
                        change.value
                    );
                }
            }

            if (step.newVDOM) {
                (component as { setVDOM: (expr: string) => void }).setVDOM(step.newVDOM);
            }

            await nextFrames(2);

            if (step.events) {
                for (const event of step.events) {
                    if (event.delay) {
                        await new Promise(resolve => setTimeout(resolve, event.delay));
                    }

                    const shadowRoot = component.shadowRoot;
                    if (shadowRoot) {
                        const target = shadowRoot.querySelector(event.target);
                        if (target) {
                            simulateEvent(
                                shadowRoot as unknown as HTMLElement,
                                event.target,
                                event.type,
                                { value: event.value, detail: event.detail }
                            );
                        }
                    }
                }
                await nextFrames(2);
            }

            const shadowRoot = component.shadowRoot;
            if (!shadowRoot) {
                errors.push(`Step "${step.description}": Component has no shadow root`);
                stepResults.push({
                    step,
                    stepIndex,
                    passed: false,
                    assertionResults: stepAssertionResults
                });
                continue;
            }

            const stepHTML = getRenderedContent(shadowRoot);

            for (const assertion of step.assertions) {
                const tempContainer = document.createElement('div');
                tempContainer.innerHTML = stepHTML;

                const result = verifyAssertion(tempContainer, assertion);
                stepAssertionResults.push({
                    assertion,
                    passed: result.passed,
                    message: result.message,
                    expected: result.expected,
                    actual: result.actual
                });

                if (!result.passed) {
                    errors.push(`Step "${step.description}": ${result.message}`);
                }
            }

            const goldenFilename = `golden-step-${stepIndex}.html`;
            const goldenResult = compareWithGolden(tc.path, stepHTML, goldenFilename);
            if (!goldenResult.passed && !goldenResult.updated) {
                errors.push(`Step ${stepIndex} (${step.description}): ${goldenResult.message}`);
            }

            stepResults.push({
                step,
                stepIndex,
                passed: stepAssertionResults.every(r => r.passed) && (goldenResult.passed || goldenResult.updated === true),
                assertionResults: stepAssertionResults,
                goldenResult,
                finalHTML: stepHTML
            });
        }

    } catch (e) {
        const error = e as Error;
        if (tc.spec.shouldError) {
            if (tc.spec.expectedErrorContains) {
                if (!error.message.includes(tc.spec.expectedErrorContains)) {
                    errors.push(
                        `Expected error to contain "${tc.spec.expectedErrorContains}" but got: ${error.message}`
                    );
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
        stepResults
    };
}
