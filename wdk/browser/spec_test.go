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
	"testing"

	"piko.sh/piko/wdk/browser/internal/browser_provider_chromedp"
)

func TestFormatStepDescription(t *testing.T) {
	testCases := []struct {
		name     string
		action   string
		selector string
		want     string
		stepNum  int
	}{
		{
			name:     "action without selector",
			stepNum:  1,
			action:   "click",
			selector: "",
			want:     "step 1: click",
		},
		{
			name:     "action with selector",
			stepNum:  3,
			action:   "fill",
			selector: "#username",
			want:     `step 3: fill "#username"`,
		},
		{
			name:     "large step number",
			stepNum:  100,
			action:   "navigate",
			selector: "",
			want:     "step 100: navigate",
		},
		{
			name:     "selector with special characters",
			stepNum:  2,
			action:   "checkText",
			selector: `div.main > span[data-id="42"]`,
			want:     `step 2: checkText "div.main > span[data-id=\"42\"]"`,
		},
		{
			name:     "empty action",
			stepNum:  1,
			action:   "",
			selector: "",
			want:     "step 1: ",
		},
		{
			name:     "step number zero",
			stepNum:  0,
			action:   "wait",
			selector: "",
			want:     "step 0: wait",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			step := &browser_provider_chromedp.BrowserStep{
				Action:   tc.action,
				Selector: tc.selector,
			}
			got := formatStepDescription(tc.stepNum, step)
			if got != tc.want {
				t.Errorf("formatStepDescription(%d, step) = %q, want %q", tc.stepNum, got, tc.want)
			}
		})
	}
}

func TestIsAssertionAction(t *testing.T) {
	assertionActions := []string{
		"checkText",
		"checkAttribute",
		"checkElementCount",
		"checkVisible",
		"checkValue",
		"captureDOM",
	}

	for _, action := range assertionActions {
		t.Run(action+" is assertion", func(t *testing.T) {
			if !isAssertionAction(action) {
				t.Errorf("isAssertionAction(%q) = false, want true", action)
			}
		})
	}

	nonAssertionActions := []string{
		"click",
		"fill",
		"press",
		"navigate",
		"wait",
		"type",
		"hover",
		"focus",
		"blur",
		"submit",
		"scroll",
		"checkClass",
		"checkHTML",
		"unknown",
		"",
	}

	for _, action := range nonAssertionActions {
		t.Run(action+" is not assertion", func(t *testing.T) {
			if isAssertionAction(action) {
				t.Errorf("isAssertionAction(%q) = true, want false", action)
			}
		})
	}
}
