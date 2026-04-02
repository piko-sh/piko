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

package browser

import (
	"fmt"
	"testing"
	"time"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

// TestSpec is an alias for browser_provider_chromedp.TestSpec.
// It lets users create test specs in their own code.
type TestSpec = browser_provider_chromedp.TestSpec

// BrowserStep is an alias for browser_provider_chromedp.BrowserStep.
// It is provided for users who want to create specs in code.
type BrowserStep = browser_provider_chromedp.BrowserStep

// RunSpec runs the test steps defined in a testspec JSON file, enabling
// declarative testing where steps are set out in JSON.
//
// Takes t (testing.TB) which receives test failures and logging.
// Takes specPath (string) which is the path to the testspec JSON file.
// Takes opts (...SpecOption) which provides optional behaviour controls.
//
// Example:
//
//	func TestHomepage(t *testing.T) {
//	    browser.RunSpec(t, "testdata/homepage.json")
//	}
func RunSpec(t testing.TB, specPath string, opts ...SpecOption) {
	options := defaultSpecOptions()
	for _, opt := range opts {
		opt(&options)
	}

	h := getGlobalHarness()
	if h == nil {
		t.Fatal("browser: no harness available - create one in TestMain and call Setup()")
	}

	spec, err := browser_provider_chromedp.LoadTestSpec(specPath)
	if err != nil {
		t.Fatalf("browser: failed to load spec %q: %v", specPath, err)
	}

	page, err := h.newPage(t)
	if err != nil {
		t.Fatalf("browser: failed to create page: %v", err)
	}
	defer page.Close()

	if spec.RequestURL != "" {
		if err := browser_provider_chromedp.Navigate(page.actionCtx(), spec.RequestURL); err != nil {
			t.Fatalf("browser: failed to navigate to %q: %v", spec.RequestURL, err)
		}
	}

	executeBrowserSteps(t, page, spec.BrowserSteps)
}

// StepDetail returns a human-readable detail string for a browser step,
// summarising what the step does based on its action type and fields.
//
// Takes step (*BrowserStep) which is the step to describe.
//
// Returns string which is the formatted detail text.
func StepDetail(step *BrowserStep) string {
	switch step.Action {
	case "checkText":
		if step.Expected != nil {
			return fmt.Sprintf("%s = %q", step.Selector, step.Expected)
		}
		return step.Selector
	case "click", "waitForSelector", "focus", "blur":
		return step.Selector
	case "wait":
		return step.Value + "ms"
	case "navigate":
		return step.Value
	case "captureDOM":
		return fmt.Sprintf("%s → %s", step.Selector, step.GoldenFile)
	case "fill", "setValue":
		return fmt.Sprintf("%s = %q", step.Selector, step.Value)
	default:
		if step.Selector != "" {
			return step.Selector
		}
		return step.Value
	}
}

// getGlobalHarness returns the global harness instance.
//
// Returns *Harness which is the shared harness instance, or nil if not set.
//
// Safe for concurrent use by multiple goroutines.
func getGlobalHarness() *Harness {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalHarness
}

// executeBrowserSteps runs all browser steps in a spec.
//
// Takes t (testing.TB) which reports test failures and errors.
// Takes page (*Page) which provides the browser page context.
// Takes steps ([]browser_provider_chromedp.BrowserStep) which defines the
// actions to execute.
func executeBrowserSteps(t testing.TB, page *Page, steps []browser_provider_chromedp.BrowserStep) {
	for i := range steps {
		step := &steps[i]
		stepDesc := formatStepDescription(i+1, step)

		if isAssertionAction(step.Action) {
			if err := browser_provider_chromedp.ExecuteAssertion(page.actionCtx(), step); err != nil {
				t.Errorf("browser: %s failed: %v", stepDesc, err)
			}
		} else {
			if err := browser_provider_chromedp.ExecuteStep(page.actionCtx(), step); err != nil {
				t.Fatalf("browser: %s failed: %v", stepDesc, err)
			}
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// formatStepDescription creates a human-readable description of a browser step.
//
// Takes stepNum (int) which is the step number in the sequence.
// Takes step (*browser_provider_chromedp.BrowserStep) which is the step to
// describe.
//
// Returns string which is the formatted description including the action and
// optional selector.
func formatStepDescription(stepNum int, step *browser_provider_chromedp.BrowserStep) string {
	description := fmt.Sprintf("step %d: %s", stepNum, step.Action)
	if step.Selector != "" {
		description += fmt.Sprintf(" %q", step.Selector)
	}
	return description
}

// isAssertionAction reports whether the given action is an assertion or check.
//
// Takes action (string) which is the action name to test.
//
// Returns bool which is true if the action is an assertion type.
func isAssertionAction(action string) bool {
	switch action {
	case "checkText", "checkAttribute", "checkElementCount", "checkVisible", "checkValue", "captureDOM":
		return true
	default:
		return false
	}
}
