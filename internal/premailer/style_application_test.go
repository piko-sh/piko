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

package premailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldApplyProperty(t *testing.T) {
	testCases := []struct {
		existingProp  *property
		originalProps map[string]bool
		name          string
		propName      string
		reason        string
		newProp       property
		expected      bool
	}{
		{
			name:          "Apply new property when nothing exists",
			propName:      "color",
			newProp:       property{value: "red", important: false},
			existingProp:  nil,
			originalProps: nil,
			expected:      true,
			reason:        "Should apply when no existing property",
		},
		{
			name:          "Normal rule overrides normal rule (specificity wins)",
			propName:      "color",
			newProp:       property{value: "blue", important: false},
			existingProp:  &property{value: "red", important: false},
			originalProps: nil,
			expected:      true,
			reason:        "Higher specificity rule should override",
		},
		{
			name:          "Important rule overrides normal rule",
			propName:      "color",
			newProp:       property{value: "blue", important: true},
			existingProp:  &property{value: "red", important: false},
			originalProps: nil,
			expected:      true,
			reason:        "Important rule should override normal rule",
		},
		{
			name:          "Important rule overrides important rule (specificity wins)",
			propName:      "color",
			newProp:       property{value: "blue", important: true},
			existingProp:  &property{value: "red", important: true},
			originalProps: nil,
			expected:      true,
			reason:        "Higher specificity important rule should override",
		},
		{
			name:          "Normal rule DOES NOT override important rule",
			propName:      "color",
			newProp:       property{value: "blue", important: false},
			existingProp:  &property{value: "red", important: true},
			originalProps: nil,
			expected:      false,
			reason:        "Normal rule should NOT override important rule",
		},
		{
			name:          "Normal CSS rule DOES NOT override original inline style",
			propName:      "color",
			newProp:       property{value: "blue", important: false},
			existingProp:  &property{value: "red", important: false},
			originalProps: map[string]bool{"color": true},
			expected:      false,
			reason:        "CSS rule should NOT override original inline style",
		},
		{
			name:          "Important CSS rule DOES override original inline style",
			propName:      "color",
			newProp:       property{value: "blue", important: true},
			existingProp:  &property{value: "red", important: false},
			originalProps: map[string]bool{"color": true},
			expected:      true,
			reason:        "Important CSS rule should override original inline style",
		},
		{
			name:          "CSS rule can override non-original inline style",
			propName:      "color",
			newProp:       property{value: "blue", important: false},
			existingProp:  &property{value: "red", important: false},
			originalProps: map[string]bool{"font-size": true},
			expected:      true,
			reason:        "CSS rule can override inline style that wasn't original",
		},
		{
			name:          "Empty original props map allows override",
			propName:      "color",
			newProp:       property{value: "blue", important: false},
			existingProp:  &property{value: "red", important: false},
			originalProps: map[string]bool{},
			expected:      true,
			reason:        "Empty original props means no protection",
		},
		{
			name:          "Important rule overrides original inline important style",
			propName:      "color",
			newProp:       property{value: "blue", important: true},
			existingProp:  &property{value: "red", important: true},
			originalProps: map[string]bool{"color": true},
			expected:      true,
			reason:        "Important CSS rule with higher specificity wins",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := shouldApplyProperty(
				tc.propName,
				tc.newProp,
				tc.existingProp,
				tc.originalProps,
			)
			assert.Equal(t, tc.expected, actual, "Test case: %s\nReason: %s", tc.name, tc.reason)
		})
	}
}
