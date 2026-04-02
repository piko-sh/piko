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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractBodyStyles(t *testing.T) {
	tests := []struct {
		name             string
		inputCSS         string
		expectedInline   string
		expectedCleanCSS string
	}{
		{
			name: "simple body rule",
			inputCSS: `body {
				background-color: #f4f7f6;
			}`,
			expectedInline:   "background-color: #f4f7f6;",
			expectedCleanCSS: "",
		},
		{
			name: "body with important (always stripped from inline)",
			inputCSS: `body {
				background-color: #f4f7f6 !important;
			}`,
			expectedInline:   "background-color: #f4f7f6;",
			expectedCleanCSS: "",
		},
		{
			name: "compound selector with body",
			inputCSS: `.body-container, body {
				background-color: #f4f7f6 !important;
			}`,
			expectedInline:   "background-color: #f4f7f6;",
			expectedCleanCSS: ".body-container {\n\tbackground-color: #f4f7f6 !important;\n}",
		},
		{
			name: "body with other rules",
			inputCSS: `.foo { color: red; }
body { background-color: #f4f7f6; }
.bar { color: blue; }`,
			expectedInline:   "background-color: #f4f7f6;",
			expectedCleanCSS: ".foo",
		},
		{
			name:             "empty CSS input",
			inputCSS:         "",
			expectedInline:   "",
			expectedCleanCSS: "",
		},
		{
			name: "body with pseudo-class should NOT be extracted",
			inputCSS: `body:hover {
				color: red;
			}`,
			expectedInline:   "",
			expectedCleanCSS: "body:hover",
		},
		{
			name: "body.class should NOT be extracted (compound selector)",
			inputCSS: `body.dark-mode {
				background-color: black;
			}`,
			expectedInline:   "",
			expectedCleanCSS: "body.dark-mode",
		},
		{
			name: "multiple body rules",
			inputCSS: `body {
				background-color: white;
			}
			body {
				color: black;
			}`,
			expectedInline:   "background-color: #ffffff; color: #000000;",
			expectedCleanCSS: "",
		},
		{
			name: "body rule among media queries",
			inputCSS: `@media (max-width: 600px) {
				.container { width: 100%; }
			}
			body { margin: 0; }
			@media (min-width: 601px) {
				.container { width: 960px; }
			}`,
			expectedInline:   "margin: 0;",
			expectedCleanCSS: "@media (max-width: 600px)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inline, cleaned := ExtractBodyStyles(tt.inputCSS)

			t.Logf("Input CSS:\n%s", tt.inputCSS)
			t.Logf("Extracted inline: %s", inline)
			t.Logf("Cleaned CSS:\n%s", cleaned)

			if inline != tt.expectedInline {
				t.Errorf("Expected inline styles %q, got %q", tt.expectedInline, inline)
			}

			if tt.expectedCleanCSS == "" && cleaned != "" {
				t.Errorf("Expected empty cleaned CSS, got %q", cleaned)
			}
			if tt.expectedCleanCSS != "" && cleaned == "" {
				t.Errorf("Expected non-empty cleaned CSS, got empty string")
			}

			if tt.expectedCleanCSS != "" {

				normalisedExpected := strings.Join(strings.Fields(tt.expectedCleanCSS), " ")
				normalisedCleaned := strings.Join(strings.Fields(cleaned), " ")
				assert.Contains(t, normalisedCleaned, normalisedExpected, "Cleaned CSS should contain expected content")
			}
		})
	}
}

func TestParseStyleAttribute(t *testing.T) {
	testCases := []struct {
		expected map[string]property
		name     string
		style    string
	}{
		{
			name:     "empty string",
			style:    "",
			expected: map[string]property{},
		},
		{
			name:  "single property",
			style: "color: red",
			expected: map[string]property{
				"color": {value: "#ff0000", important: false},
			},
		},
		{
			name:  "multiple properties",
			style: "color: red; font-size: 12px",
			expected: map[string]property{
				"color":     {value: "#ff0000", important: false},
				"font-size": {value: "12px", important: false},
			},
		},
		{
			name:  "with important",
			style: "color: red !important; margin: 0",
			expected: map[string]property{
				"color":  {value: "#ff0000", important: true},
				"margin": {value: "0", important: false},
			},
		},
		{
			name:  "extra whitespace",
			style: "  color:  blue ; ",
			expected: map[string]property{
				"color": {value: "#0000ff", important: false},
			},
		},
		{
			name:  "malformed property (missing value)",
			style: "color:; font-size: 12px",
			expected: map[string]property{
				"font-size": {value: "12px", important: false},
			},
		},
		{
			name:  "malformed property (missing name)",
			style: ": red; font-size: 12px",
			expected: map[string]property{
				"font-size": {value: "12px", important: false},
			},
		},
		{
			name:  "trailing semicolon",
			style: "color: red;",
			expected: map[string]property{
				"color": {value: "#ff0000", important: false},
			},
		},
		{
			name:  "multiple semicolons",
			style: "color: red;; font-size: 12px",
			expected: map[string]property{
				"color":     {value: "#ff0000", important: false},
				"font-size": {value: "12px", important: false},
			},
		},
		{
			name:  "color values are converted",
			style: "color: blue; background: rgb(255, 0, 0); border-color: hsl(120, 100%, 50%)",
			expected: map[string]property{
				"color":        {value: "#0000ff", important: false},
				"background":   {value: "#ff0000", important: false},
				"border-color": {value: "#00ff00", important: false},
			},
		},
		{
			name:  "important with different cases",
			style: "color: red !IMPORTANT; font-size: 12px !ImPoRtAnT",
			expected: map[string]property{
				"color":     {value: "#ff0000", important: true},
				"font-size": {value: "12px", important: true},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := parseStyleAttribute(tc.style)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestBuildInlineStylesFromMap(t *testing.T) {
	testCases := []struct {
		name     string
		styleMap map[string]property
		expected string
	}{
		{
			name:     "empty map returns empty",
			styleMap: map[string]property{},
			expected: "",
		},
		{
			name: "single property with trailing semicolon",
			styleMap: map[string]property{
				"color": {value: "red"},
			},
			expected: "color: red;",
		},
		{
			name: "multiple properties sorted with trailing semicolon",
			styleMap: map[string]property{
				"font-size": {value: "12px"},
				"color":     {value: "blue"},
			},
			expected: "color: blue; font-size: 12px;",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, buildInlineStylesFromMap(tc.styleMap))
		})
	}
}

func TestReconstructStyleAttribute(t *testing.T) {
	testCases := []struct {
		name     string
		styleMap map[string]property
		expected string
	}{
		{
			name:     "empty map",
			styleMap: map[string]property{},
			expected: "",
		},
		{
			name: "single property",
			styleMap: map[string]property{
				"color": {value: "red", important: false},
			},
			expected: "color: red",
		},
		{
			name: "multiple properties (tests sorting)",
			styleMap: map[string]property{
				"font-size": {value: "12px", important: false},
				"color":     {value: "blue", important: false},
			},
			expected: "color: blue; font-size: 12px",
		},
		{
			name: "with important flag (always stripped from inline styles)",
			styleMap: map[string]property{
				"margin": {value: "0", important: false},
				"color":  {value: "red", important: true},
			},
			expected: "color: red; margin: 0",
		},
		{
			name: "multiple important flags (all stripped)",
			styleMap: map[string]property{
				"margin": {value: "0", important: true},
				"color":  {value: "red", important: true},
			},
			expected: "color: red; margin: 0",
		},
		{
			name: "alphabetical sorting",
			styleMap: map[string]property{
				"z-index":    {value: "10", important: false},
				"color":      {value: "red", important: false},
				"margin":     {value: "0", important: false},
				"background": {value: "white", important: false},
			},
			expected: "background: white; color: red; margin: 0; z-index: 10",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := reconstructStyleAttribute(tc.styleMap)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
