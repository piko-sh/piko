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
 * Shared types for regression test specifications.
 *
 * This follows the testspec pattern from the Go backend, where each test case
 * is defined by a JSON specification file that describes inputs, expected
 * outputs, and assertions.
 */

/**
 * Base test specification that all regression tests extend.
 */
export interface BaseTestSpec {
    /** Human-readable description of what this test case validates */
    description: string;

    /** If true, the test is expected to produce an error */
    shouldError?: boolean;

    /** Expected error message substring (only checked if shouldError is true) */
    expectedErrorContains?: string;
}

/**
 * An event to simulate during test execution.
 */
export interface TestEvent {
    /** Type of event: 'click', 'input', 'change', 'custom', etc. */
    type: string;

    /** CSS selector for the target element */
    target: string;

    /** For input events, the value to set */
    value?: string;

    /** For custom events, the detail payload */
    detail?: unknown;

    /** Delay in ms before executing this event (for sequencing) */
    delay?: number;
}

/**
 * DOM assertion that can be checked after operations complete.
 */
export interface DOMAssertion {
    /** CSS selector to find the element(s) */
    selector: string;

    /** Human-readable description of what's being asserted */
    description?: string;

    /** Expected number of matching elements */
    count?: number;

    /** Expected text content (exact match) */
    textContent?: string;

    /** Expected text content (contains) */
    textContains?: string;

    /** Expected inner HTML (exact match) */
    innerHTML?: string;

    /** Expected inner HTML (contains) */
    innerHTMLContains?: string;

    /** Expected attribute values */
    attributes?: Record<string, string | null>;

    /** Expected computed styles */
    styles?: Record<string, string>;

    /** Assert element exists */
    exists?: boolean;

    /** Assert element is visible (not display:none, visibility:hidden) */
    isVisible?: boolean;

    /** Assert element has focus */
    hasFocus?: boolean;
}

/**
 * Test case metadata discovered at runtime.
 */
export interface TestCase {
    /** Name of the test case (directory name) */
    name: string;

    /** Absolute path to the test case directory */
    path: string;

    /** Parsed test specification */
    spec: BaseTestSpec;
}

/**
 * A state change to apply during VDOM testing.
 */
export interface VDOMStateChange {
    /** Path in state object (dot notation: 'items.0.name') */
    path: string;

    /** New value to set */
    value: unknown;
}

/**
 * Test specification for VDOM patching tests.
 */
export interface VDOMTestSpec extends BaseTestSpec {
    /** Initial state for the component */
    initialState: Record<string, unknown>;

    /** VDOM factory function call to produce initial tree (as code string) */
    initialVDOM: string;

    /** Sequence of state changes and their expected outcomes */
    steps: VDOMTestStep[];
}

/**
 * A single step in a VDOM test sequence.
 */
export interface VDOMTestStep {
    /** Description of this step */
    description: string;

    /** State changes to apply */
    stateChanges?: VDOMStateChange[];

    /** Events to simulate */
    events?: TestEvent[];

    /** New VDOM to render (as code string) */
    newVDOM?: string;

    /** Assertions to verify after this step */
    assertions: DOMAssertion[];
}

/**
 * Test specification for fragmentMorpher tests.
 */
export interface FragmentMorpherTestSpec extends BaseTestSpec {
    /** Options to pass to fragmentMorpher */
    options?: {
        childrenOnly?: boolean;
        preservePartialScopes?: boolean;
    };

    /** Assertions about the final DOM state */
    assertions: DOMAssertion[];

    /** Assertions about which nodes should be preserved (same reference) */
    preservedNodes?: {
        /** Selector for the node in the source */
        sourceSelector: string;
        /** Description of what should be preserved */
        description: string;
    }[];

    /** Assertions about which nodes should be removed */
    removedNodes?: {
        /** Selector that should NOT match after morph */
        selector: string;
        description: string;
    }[];
}

/**
 * Test specification for RemoteRenderer tests.
 */
export interface RemoteRendererTestSpec extends BaseTestSpec {
    /** Initial page HTML (the document before remote render) */
    initialPageHTML?: string;

    /** Mock HTTP response configuration */
    mockResponse: {
        /** HTTP status code */
        status: number;

        /** Response headers */
        headers: Record<string, string>;

        /** Response body (HTML content) - loaded from response.html file */
        bodyFile?: string;
    };

    /** Remote render options */
    renderOptions: {
        src: string;
        args?: Record<string, string | number>;
        formData?: Record<string, string | string[]>;
        patchMethod?: 'replace' | 'morph';
        childrenOnly?: boolean;
        querySelector?: string;
        preservePartialScopes?: boolean;
    };

    /** Assertions about the final DOM state */
    assertions: DOMAssertion[];

    /** Assertions about side effects */
    sideEffects?: {
        /** Links that should be added to <head> */
        linksAdded?: { href: string; rel?: string }[];

        /** Styles that should be added to <head> */
        stylesAdded?: { contains: string }[];

        /** Sprites that should be merged */
        spritesMerged?: { symbolId: string }[];
    };
}

/**
 * Result of running a regression test case.
 */
export interface TestResult {
    name: string;
    passed: boolean;
    error?: Error;
    assertions: {
        description: string;
        passed: boolean;
        expected?: unknown;
        actual?: unknown;
    }[];
}
