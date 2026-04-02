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
 * Remote Renderer Regression Test Harness
 *
 * This harness tests the RemoteRenderer which fetches HTML from a server
 * and patches it into the DOM.
 */

import { createRemoteRenderer, type RemoteRenderer } from '../../../src/core/RemoteRenderer';
import { createModuleLoader } from '../../../src/services/ModuleLoader';
import { createSpriteSheetManager } from '../../../src/services/SpriteSheetManager';
import { createLinkHeaderParser } from '../../../src/services/LinkHeaderParser';
import type { RemoteRendererTestSpec, DOMAssertion } from '../shared/types';
import {
    discoverTestCases,
    readTestFile,
    readOptionalTestFile,
    createTestContainer,
    verifyAssertion,
    compareWithGolden,
    type GoldenResult
} from '../shared/utils';
import * as path from 'path';

const TESTDATA_DIR = path.join(__dirname, 'testdata');

export interface RemoteRendererTestCase {
    name: string;
    path: string;
    spec: RemoteRendererTestSpec;
    initialHTML: string;
    responseHTML: string;
}

/**
 * Discovers all RemoteRenderer test cases.
 */
export function discoverRemoteRendererTestCases(): RemoteRendererTestCase[] {
    const baseCases = discoverTestCases<RemoteRendererTestSpec>(TESTDATA_DIR);

    return baseCases.map(tc => {
        const spec = tc.spec as RemoteRendererTestSpec;
        const responseFile = spec.mockResponse.bodyFile ?? 'response.html';

        return {
            ...tc,
            spec,
            initialHTML: readOptionalTestFile(tc.path, 'initial.html') ?? '<div id="app"></div>',
            responseHTML: readTestFile(tc.path, responseFile)
        };
    });
}

/**
 * Result of running a RemoteRenderer test.
 */
export interface RemoteRendererTestResult {
    passed: boolean;
    errors: string[];
    assertionResults: {
        assertion: DOMAssertion;
        passed: boolean;
        message: string;
        expected?: unknown;
        actual?: unknown;
    }[];
    sideEffectResults: {
        description: string;
        passed: boolean;
        message: string;
    }[];
    goldenResult?: GoldenResult;
    finalHTML?: string;
}

/**
 * Creates mock dependencies for RemoteRenderer.
 */
function createMockDependencies(
    tc: RemoteRendererTestCase,
    container: HTMLElement
) {
    const moduleLoader = createModuleLoader();
    const spriteSheetManager = createSpriteSheetManager();
    const linkHeaderParser = createLinkHeaderParser();

    const addedLinks: HTMLLinkElement[] = [];
    const addedStyles: HTMLStyleElement[] = [];
    const mergedSprites: string[] = [];

    const mockHTTP = {
        fetch: async (_input: RequestInfo | URL, _init?: RequestInit): Promise<Response> => {
            const headers = new Headers(tc.spec.mockResponse.headers);
            return new Response(tc.responseHTML, {
                status: tc.spec.mockResponse.status,
                headers
            });
        }
    };

    const onDOMUpdated = (_root: HTMLElement) => {
    };

    return {
        moduleLoader,
        spriteSheetManager,
        linkHeaderParser,
        onDOMUpdated,
        http: mockHTTP,
        tracking: {
            addedLinks,
            addedStyles,
            mergedSprites
        }
    };
}

/**
 * Runs a single RemoteRenderer test case.
 */
export async function runRemoteRendererTestCase(
    tc: RemoteRendererTestCase
): Promise<RemoteRendererTestResult> {
    const errors: string[] = [];
    const assertionResults: RemoteRendererTestResult['assertionResults'] = [];
    const sideEffectResults: RemoteRendererTestResult['sideEffectResults'] = [];

    const { container, cleanup } = createTestContainer();

    try {
        container.innerHTML = tc.initialHTML;

        let patchLocation: HTMLElement | undefined;
        if (tc.spec.renderOptions.querySelector) {
            patchLocation = container.querySelector(tc.spec.renderOptions.querySelector) as HTMLElement;
        } else {
            patchLocation = container.querySelector('#app') as HTMLElement ?? container;
        }

        if (!patchLocation) {
            throw new Error(`Patch location not found: ${tc.spec.renderOptions.querySelector ?? '#app'}`);
        }

        const deps = createMockDependencies(tc, container);
        const renderer = createRemoteRenderer({
            ...deps,
            onDOMUpdated: deps.onDOMUpdated
        });

        await renderer.render({
            src: tc.spec.renderOptions.src,
            args: tc.spec.renderOptions.args,
            formData: tc.spec.renderOptions.formData,
            patchMethod: tc.spec.renderOptions.patchMethod ?? 'morph',
            childrenOnly: tc.spec.renderOptions.childrenOnly ?? true,
            preservePartialScopes: tc.spec.renderOptions.preservePartialScopes,
            patchLocation
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

        if (tc.spec.sideEffects) {
            if (tc.spec.sideEffects.linksAdded) {
                for (const expectedLink of tc.spec.sideEffects.linksAdded) {
                    const link = document.head.querySelector(`link[href="${expectedLink.href}"]`);
                    if (!link) {
                        sideEffectResults.push({
                            description: `Link ${expectedLink.href} should be added`,
                            passed: false,
                            message: `Expected link with href="${expectedLink.href}" was not added to head`
                        });
                        errors.push(`Expected link "${expectedLink.href}" was not added`);
                    } else {
                        sideEffectResults.push({
                            description: `Link ${expectedLink.href} was added`,
                            passed: true,
                            message: 'OK'
                        });
                    }
                }
            }

            if (tc.spec.sideEffects.stylesAdded) {
                for (const expectedStyle of tc.spec.sideEffects.stylesAdded) {
                    const styles = document.head.querySelectorAll('style[pk-page]');
                    const found = Array.from(styles).some(
                        s => s.textContent?.includes(expectedStyle.contains)
                    );

                    if (!found) {
                        sideEffectResults.push({
                            description: `Style containing "${expectedStyle.contains}" should be added`,
                            passed: false,
                            message: `Expected style containing "${expectedStyle.contains}" was not added`
                        });
                        errors.push(`Expected style containing "${expectedStyle.contains}" was not added`);
                    } else {
                        sideEffectResults.push({
                            description: `Style containing "${expectedStyle.contains}" was added`,
                            passed: true,
                            message: 'OK'
                        });
                    }
                }
            }

            if (tc.spec.sideEffects.spritesMerged) {
                const spriteSheet = document.getElementById('sprite');
                for (const expectedSprite of tc.spec.sideEffects.spritesMerged) {
                    const symbol = spriteSheet?.querySelector(`symbol#${expectedSprite.symbolId}`);
                    if (!symbol) {
                        sideEffectResults.push({
                            description: `Sprite ${expectedSprite.symbolId} should be merged`,
                            passed: false,
                            message: `Expected sprite symbol "${expectedSprite.symbolId}" was not found in sprite sheet`
                        });
                        errors.push(`Expected sprite "${expectedSprite.symbolId}" was not merged`);
                    } else {
                        sideEffectResults.push({
                            description: `Sprite ${expectedSprite.symbolId} was merged`,
                            passed: true,
                            message: 'OK'
                        });
                    }
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
            sideEffectResults,
            goldenResult,
            finalHTML
        };

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
        document.head.querySelectorAll('link[href^="/"]').forEach(el => el.remove());
        document.head.querySelectorAll('style[pk-page]').forEach(el => el.remove());
        document.getElementById('sprite')?.remove();
    }

    return {
        passed: errors.length === 0,
        errors,
        assertionResults,
        sideEffectResults
    };
}
