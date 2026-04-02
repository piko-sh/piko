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

package browser_provider_chromedp

import (
	"testing"
)

func TestParseShadowDOMSelector(t *testing.T) {
	testCases := []struct {
		name           string
		selector       string
		expectedHost   string
		expectedShadow string
	}{
		{
			name:           "standard selector",
			selector:       "my-component >>> .inner",
			expectedHost:   "my-component",
			expectedShadow: ".inner",
		},
		{
			name:           "complex host selector",
			selector:       "#app > my-component >>> .shadow-child",
			expectedHost:   "#app > my-component",
			expectedShadow: ".shadow-child",
		},
		{
			name:           "complex shadow selector",
			selector:       "my-element >>> div.content > span.text",
			expectedHost:   "my-element",
			expectedShadow: "div.content > span.text",
		},
		{
			name:           "multiple separators uses SplitN 2",
			selector:       "host >>> inner >>> deep",
			expectedHost:   "host",
			expectedShadow: "inner >>> deep",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := parseShadowDOMSelector(tc.selector)
			if result.Host != tc.expectedHost {
				t.Errorf("Host = %q, expected %q", result.Host, tc.expectedHost)
			}
			if result.Shadow != tc.expectedShadow {
				t.Errorf("Shadow = %q, expected %q", result.Shadow, tc.expectedShadow)
			}
		})
	}
}
