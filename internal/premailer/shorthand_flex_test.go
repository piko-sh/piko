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

func TestExpandFlexShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "keyword none",
			value: "none",
			expected: map[string]string{
				"flex-grow":   "0",
				"flex-shrink": "0",
				"flex-basis":  "auto",
			},
		},
		{
			name:  "keyword auto",
			value: "auto",
			expected: map[string]string{
				"flex-grow":   "1",
				"flex-shrink": "1",
				"flex-basis":  "auto",
			},
		},
		{
			name:  "single number (flex-grow)",
			value: "2",
			expected: map[string]string{
				"flex-grow":   "2",
				"flex-shrink": "1",
				"flex-basis":  "0",
			},
		},
		{
			name:  "single length (flex-basis)",
			value: "100px",
			expected: map[string]string{
				"flex-basis": "100px",
			},
		},
		{
			name:  "single percentage (flex-basis)",
			value: "50%",
			expected: map[string]string{
				"flex-basis": "50%",
			},
		},
		{
			name:  "single auto (flex-basis)",
			value: "auto",
			expected: map[string]string{
				"flex-grow":   "1",
				"flex-shrink": "1",
				"flex-basis":  "auto",
			},
		},
		{
			name:  "single content (flex-basis)",
			value: "content",
			expected: map[string]string{
				"flex-basis": "content",
			},
		},
		{
			name:  "two values (grow and shrink)",
			value: "2 3",
			expected: map[string]string{
				"flex-grow":   "2",
				"flex-shrink": "3",
				"flex-basis":  "0",
			},
		},
		{
			name:  "three values",
			value: "1 0 200px",
			expected: map[string]string{
				"flex-grow":   "1",
				"flex-shrink": "0",
				"flex-basis":  "200px",
			},
		},
		{
			name:     "empty value",
			value:    "",
			expected: nil,
		},
		{
			name:     "too many values",
			value:    "1 2 3 4",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandFlexShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestExpandFlexFlowShorthand(t *testing.T) {
	testCases := []struct {
		expected map[string]string
		name     string
		value    string
	}{
		{
			name:  "direction only",
			value: "row",
			expected: map[string]string{
				"flex-direction": "row",
			},
		},
		{
			name:  "wrap only",
			value: "wrap",
			expected: map[string]string{
				"flex-wrap": "wrap",
			},
		},
		{
			name:  "direction and wrap",
			value: "column wrap",
			expected: map[string]string{
				"flex-direction": "column",
				"flex-wrap":      "wrap",
			},
		},
		{
			name:  "wrap before direction",
			value: "wrap-reverse row-reverse",
			expected: map[string]string{
				"flex-wrap":      "wrap-reverse",
				"flex-direction": "row-reverse",
			},
		},
		{
			name:  "column-reverse and nowrap",
			value: "column-reverse nowrap",
			expected: map[string]string{
				"flex-direction": "column-reverse",
				"flex-wrap":      "nowrap",
			},
		},
		{
			name:     "empty value",
			value:    "",
			expected: nil,
		},
		{
			name:     "unrecognised value",
			value:    "invalid",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := expandFlexFlowShorthand(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestIsFlexBasisValue(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{value: "auto", expected: true},
		{value: "content", expected: true},
		{value: "100px", expected: true},
		{value: "50%", expected: true},
		{value: "10em", expected: true},
		{value: "2rem", expected: true},
		{value: "100vw", expected: true},
		{value: "50vh", expected: true},
		{value: "0", expected: false},
		{value: "1", expected: false},
		{value: "2.5", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			actual := isFlexBasisValue(tc.value)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestFlexShorthandViaExpandShorthand(t *testing.T) {
	t.Run("flex shorthand is registered", func(t *testing.T) {
		result := expandShorthand("flex", "1 0 auto")
		assert.NotNil(t, result)
		assert.Equal(t, "1", result["flex-grow"])
		assert.Equal(t, "0", result["flex-shrink"])
		assert.Equal(t, "auto", result["flex-basis"])
	})

	t.Run("flex-flow shorthand is registered", func(t *testing.T) {
		result := expandShorthand("flex-flow", "row wrap")
		assert.NotNil(t, result)
		assert.Equal(t, "row", result["flex-direction"])
		assert.Equal(t, "wrap", result["flex-wrap"])
	})
}
