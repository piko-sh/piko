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
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestValidateEmailCompatibility(t *testing.T) {
	testCases := []struct {
		name                  string
		css                   string
		expectedPropertyNames []string
		expectedWarningCount  int
	}{
		{
			name:                  "Flexbox properties should generate warnings",
			css:                   `.container { display: flex; flex-direction: row; justify-content: center; }`,
			expectedWarningCount:  3,
			expectedPropertyNames: []string{"flex", "flex-direction", "justify-content"},
		},
		{
			name:                  "Grid properties should generate warnings",
			css:                   `.layout { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }`,
			expectedWarningCount:  3,
			expectedPropertyNames: []string{"grid", "grid-template-columns", "gap"},
		},
		{
			name:                  "Animation properties should generate warnings",
			css:                   `.animated { animation: fadeIn 1s; transition: all 0.3s; }`,
			expectedWarningCount:  2,
			expectedPropertyNames: []string{"animation", "transition"},
		},
		{
			name:                  "Transform properties should generate warnings",
			css:                   `.rotated { transform: rotate(45deg); transform-origin: center; }`,
			expectedWarningCount:  2,
			expectedPropertyNames: []string{"transform", "transform-origin"},
		},
		{
			name:                  "Position property should generate warning",
			css:                   `.fixed { position: fixed; }`,
			expectedWarningCount:  1,
			expectedPropertyNames: []string{"position"},
		},
		{
			name:                  "Filter properties should generate warnings",
			css:                   `.blurred { filter: blur(5px); backdrop-filter: blur(10px); }`,
			expectedWarningCount:  2,
			expectedPropertyNames: []string{"filter", "backdrop-filter"},
		},
		{
			name:                  "Object-fit should generate warning",
			css:                   `img { object-fit: cover; object-position: center; }`,
			expectedWarningCount:  2,
			expectedPropertyNames: []string{"object-fit", "object-position"},
		},
		{
			name:                  "Multi-column properties should generate warnings",
			css:                   `.columns { columns: 2; column-gap: 20px; column-rule: 1px solid black; }`,
			expectedWarningCount:  2,
			expectedPropertyNames: []string{"columns", "column-gap"},
		},
		{
			name:                  "Clip-path should generate warning",
			css:                   `.clipped { clip-path: circle(50%); }`,
			expectedWarningCount:  1,
			expectedPropertyNames: []string{"clip-path"},
		},
		{
			name:                  "Safe properties should NOT generate warnings",
			css:                   `.safe { color: red; background-color: blue; padding: 10px; margin: 20px; width: 100%; border: 1px solid black; }`,
			expectedWarningCount:  0,
			expectedPropertyNames: []string{},
		},
		{
			name:                  "Properties we handle automatically should NOT generate warnings",
			css:                   `#main { color: blue; } .container { padding: 10px; }`,
			expectedWarningCount:  0,
			expectedPropertyNames: []string{},
		},
		{
			name:                  "Mixed safe and problematic properties",
			css:                   `.mixed { color: red; display: flex; padding: 10px; animation: fadeIn 1s; }`,
			expectedWarningCount:  2,
			expectedPropertyNames: []string{"flex", "animation"},
		},
		{
			name:                  "Multiple selectors with problematic properties",
			css:                   `.flex1 { display: flex; } .flex2 { flex-direction: column; } .grid { display: grid; }`,
			expectedWarningCount:  3,
			expectedPropertyNames: []string{"flex", "flex-direction", "grid"},
		},
		{
			name:                  "Shorthand properties that are expanded should not cause double warnings",
			css:                   `.box { margin: 10px; padding: 20px; border: 1px solid black; }`,
			expectedWarningCount:  0,
			expectedPropertyNames: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cssAST := parseTestCSS(t, tc.css)
			var parseDiagnostics []*ast_domain.Diagnostic
			ruleSet := ProcessCSS(cssAST, &Options{ExpandShorthands: true}, &parseDiagnostics, "test.css")

			diagnostics := validateEmailCompatibility(ruleSet, "test.css")

			assert.Len(t, diagnostics, tc.expectedWarningCount, "Expected %d warnings, got %d", tc.expectedWarningCount, len(diagnostics))

			for _, diagnostic := range diagnostics {
				assert.Equal(t, ast_domain.Warning, diagnostic.Severity, "All diagnostics should be warnings")
			}

			for _, expectedProp := range tc.expectedPropertyNames {
				found := false
				for _, diagnostic := range diagnostics {
					if strings.Contains(diagnostic.Message, expectedProp) || diagnostic.Expression == expectedProp {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected to find warning about property '%s'", expectedProp)
			}
		})
	}
}

func TestValidateEmailCompatibilityWithLeftoverRules(t *testing.T) {

	testCases := []struct {
		name                 string
		css                  string
		expectedWarningCount int
	}{
		{
			name:                 "Flexbox in hover pseudo-class should generate warning",
			css:                  `a:hover { display: flex; flex-direction: row; }`,
			expectedWarningCount: 2,
		},
		{
			name:                 "Grid in media query should generate warning",
			css:                  `@media (max-width: 600px) { .container { display: grid; grid-template-columns: 1fr; } }`,
			expectedWarningCount: 2,
		},
		{
			name:                 "Animation in pseudo-class should generate warning",
			css:                  `button:active { animation: pulse 0.3s; }`,
			expectedWarningCount: 1,
		},
		{
			name:                 "Safe properties in pseudo-class should NOT generate warnings",
			css:                  `a:hover { color: red; text-decoration: underline; }`,
			expectedWarningCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			cssAST := parseTestCSS(t, tc.css)
			var parseDiagnostics []*ast_domain.Diagnostic
			ruleSet := ProcessCSS(cssAST, &Options{ExpandShorthands: false}, &parseDiagnostics, "test.css")

			diagnostics := validateEmailCompatibility(ruleSet, "test.css")

			assert.Len(t, diagnostics, tc.expectedWarningCount, "Expected %d warnings for leftover rules", tc.expectedWarningCount)
		})
	}
}

func TestValidatePropertyValue(t *testing.T) {
	testCases := []struct {
		name          string
		propName      string
		propValue     string
		expectMessage string
		expectWarning bool
	}{
		{name: "calc() generates warning", propName: "width", propValue: "calc(100% - 20px)", expectWarning: true, expectMessage: "calc()"},
		{name: "display flex generates warning", propName: "display", propValue: "flex", expectWarning: true, expectMessage: "display: flex"},
		{name: "display inline-flex generates warning", propName: "display", propValue: "inline-flex", expectWarning: true, expectMessage: "display: inline-flex"},
		{name: "display grid generates warning", propName: "display", propValue: "grid", expectWarning: true, expectMessage: "display: grid"},
		{name: "display inline-grid generates warning", propName: "display", propValue: "inline-grid", expectWarning: true, expectMessage: "display: inline-grid"},
		{name: "display block is safe", propName: "display", propValue: "block", expectWarning: false},
		{name: "display none is safe", propName: "display", propValue: "none", expectWarning: false},
		{name: "position fixed generates warning", propName: "position", propValue: "fixed", expectWarning: true, expectMessage: "position: fixed"},
		{name: "position sticky generates warning", propName: "position", propValue: "sticky", expectWarning: true, expectMessage: "position: sticky"},
		{name: "position relative is safe", propName: "position", propValue: "relative", expectWarning: false},
		{name: "position absolute is safe (no specific warning)", propName: "position", propValue: "absolute", expectWarning: false},
		{name: "background gradient generates warning", propName: "background", propValue: "linear-gradient(red, blue)", expectWarning: true, expectMessage: "gradient"},
		{name: "background-image gradient generates warning", propName: "background-image", propValue: "radial-gradient(circle, red, blue)", expectWarning: true, expectMessage: "gradient"},
		{name: "background solid colour is safe", propName: "background", propValue: "#ff0000", expectWarning: false},
		{name: "other property is safe", propName: "color", propValue: "red", expectWarning: false},
		{name: "calc with whitespace", propName: "margin", propValue: "  CALC(100% - 10px)  ", expectWarning: true, expectMessage: "calc()"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := validatePropertyValue(tc.propName, tc.propValue, "test.css")
			if tc.expectWarning {
				require.NotEmpty(t, diagnostics, "expected a warning diagnostic")
				assert.Contains(t, diagnostics[0].Message, tc.expectMessage)
			} else {
				assert.Empty(t, diagnostics)
			}
		})
	}
}

func TestValidateDisplayValue(t *testing.T) {
	testCases := []struct {
		name          string
		value         string
		expectWarning bool
	}{
		{name: "flex", value: "flex", expectWarning: true},
		{name: "inline-flex", value: "inline-flex", expectWarning: true},
		{name: "grid", value: "grid", expectWarning: true},
		{name: "inline-grid", value: "inline-grid", expectWarning: true},
		{name: "block is safe", value: "block", expectWarning: false},
		{name: "inline is safe", value: "inline", expectWarning: false},
		{name: "none is safe", value: "none", expectWarning: false},
		{name: "table is safe", value: "table", expectWarning: false},
		{name: "inline-block is safe", value: "inline-block", expectWarning: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := validateDisplayValue(tc.value, "test.css")
			if tc.expectWarning {
				assert.Len(t, diagnostics, 1)
			} else {
				assert.Empty(t, diagnostics)
			}
		})
	}
}

func TestValidatePositionValue(t *testing.T) {
	testCases := []struct {
		name          string
		value         string
		expectWarning bool
	}{
		{name: "fixed", value: "fixed", expectWarning: true},
		{name: "sticky", value: "sticky", expectWarning: true},
		{name: "relative is safe", value: "relative", expectWarning: false},
		{name: "absolute is safe", value: "absolute", expectWarning: false},
		{name: "static is safe", value: "static", expectWarning: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := validatePositionValue(tc.value, "test.css")
			if tc.expectWarning {
				assert.Len(t, diagnostics, 1)
			} else {
				assert.Empty(t, diagnostics)
			}
		})
	}
}

func TestValidateBackgroundValue(t *testing.T) {
	testCases := []struct {
		name          string
		value         string
		propName      string
		expectWarning bool
	}{
		{name: "linear gradient", value: "linear-gradient(red, blue)", propName: "background", expectWarning: true},
		{name: "radial gradient", value: "radial-gradient(circle, red, blue)", propName: "background-image", expectWarning: true},
		{name: "solid colour", value: "#ff0000", propName: "background", expectWarning: false},
		{name: "url value", value: "url(image.png)", propName: "background", expectWarning: false},
		{name: "none", value: "none", propName: "background", expectWarning: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := validateBackgroundValue(tc.value, tc.propName, "test.css")
			if tc.expectWarning {
				assert.Len(t, diagnostics, 1)
			} else {
				assert.Empty(t, diagnostics)
			}
		})
	}
}

func TestValidateLeftoverTokenValue(t *testing.T) {
	testCases := []struct {
		name          string
		propName      string
		value         string
		expectWarning bool
	}{
		{name: "display flex in leftover", propName: "display", value: "flex", expectWarning: true},
		{name: "display inline-flex in leftover", propName: "display", value: "inline-flex", expectWarning: true},
		{name: "display grid in leftover", propName: "display", value: "grid", expectWarning: true},
		{name: "display inline-grid in leftover", propName: "display", value: "inline-grid", expectWarning: true},
		{name: "display block in leftover (safe)", propName: "display", value: "block", expectWarning: false},
		{name: "position fixed in leftover", propName: "position", value: "fixed", expectWarning: true},
		{name: "position sticky in leftover", propName: "position", value: "sticky", expectWarning: true},
		{name: "position relative in leftover (safe)", propName: "position", value: "relative", expectWarning: false},
		{name: "background gradient in leftover", propName: "background", value: "linear-gradient", expectWarning: true},
		{name: "background-image gradient in leftover", propName: "background-image", value: "radial-gradient", expectWarning: true},
		{name: "background solid in leftover (safe)", propName: "background", value: "red", expectWarning: false},
		{name: "unrelated property (safe)", propName: "color", value: "blue", expectWarning: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostic := validateLeftoverTokenValue(tc.propName, tc.value, "test.css")
			if tc.expectWarning {
				assert.NotNil(t, diagnostic)
			} else {
				assert.Nil(t, diagnostic)
			}
		})
	}
}

func TestTransformGeneratesDiagnostics(t *testing.T) {

	testCases := []struct {
		name                 string
		htmlInput            string
		expectedWarningCount int
	}{
		{
			name: "Flexbox in inline styles generates diagnostics",
			htmlInput: `<html><head><style>
				.container { display: flex; justify-content: center; }
			</style></head><body><div class="container">Content</div></body></html>`,
			expectedWarningCount: 2,
		},
		{
			name: "Grid in inline styles generates diagnostics",
			htmlInput: `<html><head><style>
				.layout { display: grid; grid-template-columns: 1fr 1fr; }
			</style></head><body><div class="layout">Content</div></body></html>`,
			expectedWarningCount: 2,
		},
		{
			name: "Safe CSS generates no diagnostics",
			htmlInput: `<html><head><style>
				.safe { color: blue; padding: 10px; background-color: white; }
			</style></head><body><div class="safe">Content</div></body></html>`,
			expectedWarningCount: 0,
		},
		{
			name: "Multiple problematic properties generate multiple diagnostics",
			htmlInput: `<html><head><style>
				.flex { display: flex; }
				.grid { display: grid; }
				.animated { animation: fadeIn 1s; }
			</style></head><body><div class="flex">F</div><div class="grid">G</div><div class="animated">A</div></body></html>`,
			expectedWarningCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			tree, err := ast_domain.Parse(context.Background(), tc.htmlInput, "test.html", nil)
			require.NoError(t, err, "Parsing HTML input should not fail")

			premailer := New(tree)

			transformedTree, err := premailer.Transform()
			require.NoError(t, err, "Transform should not fail")

			warnings := 0
			for _, diagnostic := range transformedTree.Diagnostics {
				if diagnostic.Severity == ast_domain.Warning {
					warnings++
				}
			}
			assert.Equal(t, tc.expectedWarningCount, warnings, "Expected %d warnings in diagnostics", tc.expectedWarningCount)
		})
	}
}
