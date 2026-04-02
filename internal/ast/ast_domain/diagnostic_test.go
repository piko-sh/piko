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

package ast_domain

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"piko.sh/piko/internal/colour"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeverity_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		severity Severity
	}{
		{name: "Debug", expected: "debug", severity: Debug},
		{name: "Info", expected: "info", severity: Info},
		{name: "Warning", expected: "warning", severity: Warning},
		{name: "Error", expected: "error", severity: Error},
		{name: "Unknown", expected: "unknown", severity: Severity(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.severity.String())
		})
	}
}

func TestSeverity_CodeString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expected string
		severity Severity
	}{
		{name: "Debug", expected: "Debug", severity: Debug},
		{name: "Info", expected: "Info", severity: Info},
		{name: "Warning", expected: "Warning", severity: Warning},
		{name: "Error", expected: "Error", severity: Error},
		{name: "Unknown", expected: "unknown", severity: Severity(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.severity.CodeString())
		})
	}
}

func TestDiagnostic_Error(t *testing.T) {
	t.Parallel()

	diagnostic := NewDiagnostic(
		Error,
		"a test message",
		"expression",
		Location{Line: 10, Column: 5},
		"test",
	)
	expected := "error in test at line 10, col 5: a test message"
	assert.Equal(t, expected, diagnostic.Error())
}

func TestHasErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		diagnostics []*Diagnostic
		expected    bool
	}{
		{name: "nil slice", diagnostics: nil, expected: false},
		{name: "empty slice", diagnostics: []*Diagnostic{}, expected: false},
		{name: "only warnings", diagnostics: []*Diagnostic{{Severity: Warning}}, expected: false},
		{name: "only errors", diagnostics: []*Diagnostic{{Severity: Error}}, expected: true},
		{name: "mixed with error first", diagnostics: []*Diagnostic{{Severity: Error}, {Severity: Warning}}, expected: true},
		{name: "mixed with warning first", diagnostics: []*Diagnostic{{Severity: Warning}, {Severity: Error}}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, HasErrors(tt.diagnostics))
		})
	}
}

func TestFormatDiagnostics(t *testing.T) {
	previous := colour.Enabled()
	colour.SetEnabled(false)
	t.Cleanup(func() { colour.SetEnabled(previous) })

	testCases := []struct {
		name                  string
		sourcePath            string
		source                string
		expectMessageContains string
		expectHighlight       string
		syntheticExpression   string
		expectLine            int
		expectColumn          int
		isSyntheticWarning    bool
	}{
		{
			name:                  "Single line attribute with syntax error",
			sourcePath:            "src/components/simple.pkc",
			source:                `<div p-if="user.isActive &&"></div>`,
			expectMessageContains: "Expected expression on the right side of the operator",
			expectHighlight:       "^",
			expectLine:            1,
			expectColumn:          26,
		},
		{
			name:       "Multi-line attribute with error on second line",
			sourcePath: "src/components/card.pkc",
			source: `
<div
  p-if="user.name +
    user.age > 100 &&"
></div>`,
			expectMessageContains: "Expected expression on the right side of the operator",
			expectHighlight:       "^",
			expectLine:            4,
			expectColumn:          20,
		},
		{
			name:                  "Synthetic warning for formatting test",
			sourcePath:            "src/components/linter.pkc",
			source:                `<div p-show="!!isVisible"></div>`,
			isSyntheticWarning:    true,
			syntheticExpression:   "!!",
			expectMessageContains: "Redundant double negation",
			expectHighlight:       "^^",
			expectLine:            1,
			expectColumn:          15,
		},
		{
			name:                  "Unrecognised character in expression",
			sourcePath:            "src/components/other.pkc",
			source:                `<div p-text="user.name # invalid"></div>`,
			expectMessageContains: "unrecognised character '#'",
			expectHighlight:       "^",
			expectLine:            1,
			expectColumn:          24,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var diagnostics []*Diagnostic
			if tc.isSyntheticWarning {
				diagnostics = []*Diagnostic{
					NewDiagnostic(Warning, tc.expectMessageContains, tc.syntheticExpression, Location{Line: tc.expectLine, Column: tc.expectColumn}, "test"),
				}
			} else {
				tree, err := ParseAndTransform(context.Background(), tc.source, "test")
				require.NoError(t, err, "Parser should not return a fatal error, only diagnostics")
				require.True(t, HasErrors(tree.Diagnostics), "Parser should have produced an error diagnostic for this source")
				diagnostics = tree.Diagnostics
			}

			output := FormatDiagnostics(tc.sourcePath, tc.source, diagnostics)

			t.Logf("\n--- Test Case: %s ---\n%s\n", tc.name, output)

			assert.Contains(t, output, tc.expectMessageContains, "Output should contain the correct error message")
			assert.Contains(t, output, fmt.Sprintf("--> %s:%d:%d", tc.sourcePath, tc.expectLine, tc.expectColumn), "Output should contain the correct file path and location")

			lines := strings.Split(output, "\n")
			foundHighlight := false
			for _, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "|") && strings.Contains(line, "^") {
					padding := strings.Repeat(" ", tc.expectColumn-1)
					expectedLine := fmt.Sprintf("  | %s%s", padding, tc.expectHighlight)
					if strings.Contains(line, expectedLine) {
						foundHighlight = true
						break
					}
				}
			}
			assert.True(t, foundHighlight, "Could not find a correctly formatted highlight line.\nExpected something like: '  | %s%s'", strings.Repeat(" ", tc.expectColumn-1), tc.expectHighlight)
		})
	}

	t.Run("no diagnostics", func(t *testing.T) {
		output := FormatDiagnostics("path/to/file.pkc", "some source code", nil)
		assert.Empty(t, output)
	})

	t.Run("invalid location", func(t *testing.T) {
		diagnostics := []*Diagnostic{
			NewDiagnostic(Error, "Something went wrong", ``, Location{Line: 99, Column: 1}, "test"),
		}
		output := FormatDiagnostics("path/to/file.pkc", "line 1\nline 2", diagnostics)
		expected := "● error: Something went wrong (at invalid location L99:1)"
		assert.Contains(t, output, expected)
	})
}

func TestFormatDiagnostics_WithHTMLEntities(t *testing.T) {
	previous := colour.Enabled()
	colour.SetEnabled(false)
	t.Cleanup(func() { colour.SetEnabled(previous) })

	testCases := []struct {
		name                string
		sourcePath          string
		sourceWithEntities  string
		diagnostic          *Diagnostic
		expectedDisplayLine string
		expectedHighlight   string
	}{
		{
			name:               "error after a single quote entity",
			sourcePath:         "src/components/encoded.pkc",
			sourceWithEntities: `<div title="'hello' & world"></div>`,
			diagnostic: NewDiagnostic(
				Error,
				"Invalid character",
				"&",
				Location{Line: 1, Column: 26},
				"test",
			),
			expectedDisplayLine: "1 | <div title=\"'hello' & world\"></div>",
			expectedHighlight:   "  |                          ^",
		},
		{
			name:               "error on a greater-than entity",
			sourcePath:         "src/components/encoded.pkc",
			sourceWithEntities: `<div p-if="a > b"></div>`,
			diagnostic: NewDiagnostic(
				Error,
				"Unsupported operator",
				">",
				Location{Line: 1, Column: 14},
				"test",
			),
			expectedDisplayLine: "1 | <div p-if=\"a > b\"></div>",
			expectedHighlight:   "  |              ^",
		},
		{
			name:               "error with multiple entities before it",
			sourcePath:         "src/components/encoded.pkc",
			sourceWithEntities: `<!-- '&" < > --> <p p-text="err"></p>`,
			diagnostic: NewDiagnostic(
				Error,
				"Unknown variable",
				"err",
				Location{Line: 1, Column: 38},
				"test",
			),
			expectedDisplayLine: "1 | <!-- '&\" < > --> <p p-text=\"err\"></p>",
			expectedHighlight:   "  |                                      ^^^",
		},
		{
			name:               "highlight length is also adjusted",
			sourcePath:         "src/components/encoded.pkc",
			sourceWithEntities: `<div title="error is 'bold'"></div>`,
			diagnostic: NewDiagnostic(
				Error,
				"Invalid emphasis",
				"'bold'",
				Location{Line: 1, Column: 22},
				"test",
			),
			expectedDisplayLine: "1 | <div title=\"error is 'bold'\"></div>",
			expectedHighlight:   "  |                      ^^^^^^",
		},
		{
			name:               "error with Arabic (RTL) text before it",
			sourcePath:         "src/components/rtl.pkc",
			sourceWithEntities: `<div p-text="مرحبا 'world' +"></div>`,
			diagnostic: NewDiagnostic(
				Error,
				"Expected expression on the right side of the operator",
				"+",
				Location{Line: 1, Column: 31},
				"test",
			),

			expectedDisplayLine: "1 | <div p-text=\"مرحبا 'world' +\"></div>",
			expectedHighlight:   "  |                          ^",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := []*Diagnostic{tc.diagnostic}

			output := FormatDiagnostics(tc.sourcePath, tc.sourceWithEntities, diagnostics)

			t.Logf("\n--- Test Case: %s ---\n%s\n", tc.name, output)

			assert.Contains(t, output, tc.expectedDisplayLine, "The displayed source line should be unescaped for readability")

			assert.Contains(t, output, tc.expectedHighlight, "The error highlight caret is not correctly positioned")

			assert.Contains(t, output,
				fmt.Sprintf("--> %s:%d:%d", tc.sourcePath, tc.diagnostic.Location.Line, tc.diagnostic.Location.Column),
				"The diagnostic header should always report the true, original location",
			)
		})
	}
}

func TestFormatDiagnostics_WithTabs(t *testing.T) {
	previous := colour.Enabled()
	colour.SetEnabled(false)
	t.Cleanup(func() { colour.SetEnabled(previous) })

	testCases := []struct {
		name                string
		sourcePath          string
		sourceWithTabs      string
		diagnostic          *Diagnostic
		expectedDisplayLine string
		expectedHighlight   string
	}{
		{
			name:           "single tab before directive",
			sourcePath:     "src/components/tabbed.pkc",
			sourceWithTabs: "\tp-for=\"item in items\"",
			diagnostic: NewDiagnostic(
				Warning,
				"Missing p-key",
				"p-for",
				Location{Line: 1, Column: 2},
				"test",
			),

			expectedDisplayLine: "1 |         p-for=\"item in items\"",
			expectedHighlight:   "  |         ^^^^^",
		},
		{
			name:           "multiple tabs before directive",
			sourcePath:     "src/components/tabbed.pkc",
			sourceWithTabs: "\t\t\t\tp-for=\"section in items\"",
			diagnostic: NewDiagnostic(
				Warning,
				"Missing p-key",
				"p-for",
				Location{Line: 1, Column: 5},
				"test",
			),

			expectedDisplayLine: "1 |                                 p-for=\"section in items\"",
			expectedHighlight:   "  |                                 ^^^^^",
		},
		{
			name:           "tab in middle of line",
			sourcePath:     "src/components/tabbed.pkc",
			sourceWithTabs: "<div\tp-class=\"active\">",
			diagnostic: NewDiagnostic(
				Warning,
				"Static class conflict",
				"p-class",
				Location{Line: 1, Column: 6},
				"test",
			),

			expectedDisplayLine: "1 | <div    p-class=\"active\">",
			expectedHighlight:   "  |         ^^^^^^^",
		},
		{
			name:           "mixed tabs and spaces",
			sourcePath:     "src/components/tabbed.pkc",
			sourceWithTabs: "\t  \tp-for=\"x in y\"",
			diagnostic: NewDiagnostic(
				Warning,
				"Missing p-key",
				"p-for",
				Location{Line: 1, Column: 5},
				"test",
			),

			expectedDisplayLine: "1 |                 p-for=\"x in y\"",
			expectedHighlight:   "  |                 ^^^^^",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics := []*Diagnostic{tc.diagnostic}

			output := FormatDiagnostics(tc.sourcePath, tc.sourceWithTabs, diagnostics)

			t.Logf("\n--- Test Case: %s ---\n%s\n", tc.name, output)

			assert.Contains(t, output, tc.expectedDisplayLine,
				"The displayed source line should have tabs expanded to spaces")

			assert.Contains(t, output, tc.expectedHighlight,
				"The error highlight caret should be positioned accounting for tab expansion")
		})
	}
}

func TestDiagnostic_SourceLength(t *testing.T) {
	t.Parallel()

	t.Run("NewDiagnosticForExpression sets SourceLength from expression", func(t *testing.T) {
		t.Parallel()

		parser := NewExpressionParser(context.Background(), "user.name.toUpperCase()", "test.pkc")
		expression, diagnostics := parser.ParseExpression(context.Background())
		require.Empty(t, diagnostics)
		require.NotNil(t, expression)

		diagnostic := NewDiagnosticForExpression(
			Error,
			"Type error in expression",
			expression,
			Location{Line: 10, Column: 5, Offset: 100},
			"test.pkc",
		)

		assert.Equal(t, expression.GetSourceLength(), diagnostic.SourceLength, "SourceLength should be set from expression")
		assert.Equal(t, 23, diagnostic.SourceLength, "Expected SourceLength for 'user.name.toUpperCase()'")
	})

	t.Run("GetEffectiveSourceLength returns SourceLength when set", func(t *testing.T) {
		t.Parallel()

		diagnostic := &Diagnostic{
			Expression:   "user.name",
			SourceLength: 15,
		}
		assert.Equal(t, 15, diagnostic.GetEffectiveSourceLength())
	})

	t.Run("GetEffectiveSourceLength falls back to len(Expression)", func(t *testing.T) {
		t.Parallel()

		diagnostic := &Diagnostic{
			Expression:   "user.name",
			SourceLength: 0,
		}
		assert.Equal(t, 9, diagnostic.GetEffectiveSourceLength(), "Should fallback to len(Expression)")
	})

	t.Run("NewDiagnostic has zero SourceLength for backward compatibility", func(t *testing.T) {
		t.Parallel()

		diagnostic := NewDiagnostic(
			Error,
			"Some error",
			"expression",
			Location{},
			"test.pkc",
		)
		assert.Equal(t, 0, diagnostic.SourceLength, "SourceLength should default to 0")
		assert.Equal(t, 10, diagnostic.GetEffectiveSourceLength(), "Should fallback to len(Expression)")
	})
}

func TestLocation_IsSynthetic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		location Location
		expected bool
	}{
		{
			name:     "line zero is synthetic",
			location: Location{Line: 0, Column: 1, Offset: 0},
			expected: true,
		},
		{
			name:     "line one is not synthetic",
			location: Location{Line: 1, Column: 1, Offset: 0},
			expected: false,
		},
		{
			name:     "line greater than zero is not synthetic",
			location: Location{Line: 10, Column: 5, Offset: 100},
			expected: false,
		},
		{
			name:     "zero location is synthetic",
			location: Location{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.location.IsSynthetic())
		})
	}
}

func TestLocation_Add(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     Location
		other    Location
		expected Location
	}{
		{
			name:     "other is synthetic (line 0) returns base",
			base:     Location{Line: 5, Column: 10, Offset: 50},
			other:    Location{Line: 0, Column: 1, Offset: 0},
			expected: Location{Line: 5, Column: 10, Offset: 50},
		},
		{
			name:     "base is synthetic (line 0) returns other",
			base:     Location{Line: 0, Column: 1, Offset: 0},
			other:    Location{Line: 3, Column: 7, Offset: 30},
			expected: Location{Line: 3, Column: 7, Offset: 30},
		},
		{
			name:     "both synthetic returns base",
			base:     Location{Line: 0, Column: 0, Offset: 0},
			other:    Location{Line: 0, Column: 0, Offset: 0},
			expected: Location{Line: 0, Column: 0, Offset: 0},
		},
		{
			name:     "other on same line adds columns",
			base:     Location{Line: 5, Column: 10, Offset: 50},
			other:    Location{Line: 1, Column: 5, Offset: 0},
			expected: Location{Line: 5, Column: 14, Offset: 0},
		},
		{
			name:     "other on different line uses other column",
			base:     Location{Line: 5, Column: 10, Offset: 50},
			other:    Location{Line: 3, Column: 7, Offset: 0},
			expected: Location{Line: 7, Column: 7, Offset: 0},
		},
		{
			name:     "adds lines correctly",
			base:     Location{Line: 10, Column: 5, Offset: 100},
			other:    Location{Line: 5, Column: 15, Offset: 0},
			expected: Location{Line: 14, Column: 15, Offset: 0},
		},
		{
			name:     "single line addition",
			base:     Location{Line: 1, Column: 1, Offset: 0},
			other:    Location{Line: 1, Column: 10, Offset: 0},
			expected: Location{Line: 1, Column: 10, Offset: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.base.Add(tt.other)
			assert.Equal(t, tt.expected.Line, result.Line, "Line mismatch")
			assert.Equal(t, tt.expected.Column, result.Column, "Column mismatch")
		})
	}
}

func TestLocation_IsBefore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		loc      Location
		other    Location
		expected bool
	}{
		{
			name:     "same location",
			loc:      Location{Line: 5, Column: 10},
			other:    Location{Line: 5, Column: 10},
			expected: false,
		},
		{
			name:     "earlier line",
			loc:      Location{Line: 3, Column: 10},
			other:    Location{Line: 5, Column: 10},
			expected: true,
		},
		{
			name:     "later line",
			loc:      Location{Line: 7, Column: 10},
			other:    Location{Line: 5, Column: 10},
			expected: false,
		},
		{
			name:     "same line earlier column",
			loc:      Location{Line: 5, Column: 5},
			other:    Location{Line: 5, Column: 10},
			expected: true,
		},
		{
			name:     "same line later column",
			loc:      Location{Line: 5, Column: 15},
			other:    Location{Line: 5, Column: 10},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.loc.IsBefore(tt.other))
		})
	}
}

func TestNewDiagnosticWithCode(t *testing.T) {
	t.Parallel()

	t.Run("creates diagnostic with code", func(t *testing.T) {
		t.Parallel()

		diagnostic := NewDiagnosticWithCode(
			Error,
			"Variable is undefined",
			"myVar",
			"undefined-variable",
			Location{Line: 10, Column: 5},
			"test.pkc",
		)

		assert.Equal(t, Error, diagnostic.Severity)
		assert.Equal(t, "Variable is undefined", diagnostic.Message)
		assert.Equal(t, "myVar", diagnostic.Expression)
		assert.Equal(t, "undefined-variable", diagnostic.Code)
		assert.Equal(t, 10, diagnostic.Location.Line)
		assert.Equal(t, 5, diagnostic.Location.Column)
		assert.Equal(t, "test.pkc", diagnostic.SourcePath)
		assert.NotNil(t, diagnostic.Data, "Data map should be initialised")
		assert.Empty(t, diagnostic.Data, "Data map should be empty")
	})

	t.Run("creates diagnostic with warning severity", func(t *testing.T) {
		t.Parallel()

		diagnostic := NewDiagnosticWithCode(
			Warning,
			"Unused variable",
			"unusedVar",
			"unused-variable",
			Location{Line: 5, Column: 1},
			"component.pkc",
		)

		assert.Equal(t, Warning, diagnostic.Severity)
		assert.Equal(t, "unused-variable", diagnostic.Code)
	})
}

func TestNewDiagnosticWithData(t *testing.T) {
	t.Parallel()

	t.Run("creates diagnostic with data", func(t *testing.T) {
		t.Parallel()

		data := map[string]any{
			"suggestedFix": "Add import statement",
			"importPath":   "github.com/example/pkg",
			"lineNumber":   42,
		}

		diagnostic := NewDiagnosticWithData(
			Error,
			"Package not imported",
			"pkg.Function()",
			"missing-import",
			Location{Line: 15, Column: 10},
			"main.pkc",
			data,
		)

		assert.Equal(t, Error, diagnostic.Severity)
		assert.Equal(t, "Package not imported", diagnostic.Message)
		assert.Equal(t, "pkg.Function()", diagnostic.Expression)
		assert.Equal(t, "missing-import", diagnostic.Code)
		assert.Equal(t, 15, diagnostic.Location.Line)
		assert.Equal(t, 10, diagnostic.Location.Column)
		assert.Equal(t, "main.pkc", diagnostic.SourcePath)
		require.NotNil(t, diagnostic.Data)
		assert.Equal(t, "Add import statement", diagnostic.Data["suggestedFix"])
		assert.Equal(t, "github.com/example/pkg", diagnostic.Data["importPath"])
		assert.Equal(t, 42, diagnostic.Data["lineNumber"])
	})

	t.Run("handles nil data", func(t *testing.T) {
		t.Parallel()

		diagnostic := NewDiagnosticWithData(
			Warning,
			"Some warning",
			"expression",
			"warning-code",
			Location{Line: 1, Column: 1},
			"test.pkc",
			nil,
		)

		assert.Nil(t, diagnostic.Data)
	})

	t.Run("handles empty data", func(t *testing.T) {
		t.Parallel()

		diagnostic := NewDiagnosticWithData(
			Info,
			"Some info",
			"expression",
			"info-code",
			Location{Line: 1, Column: 1},
			"test.pkc",
			map[string]any{},
		)

		assert.NotNil(t, diagnostic.Data)
		assert.Empty(t, diagnostic.Data)
	})
}

func TestHasDiagnostics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		diagnostics []*Diagnostic
		expected    bool
	}{
		{
			name:        "nil slice",
			diagnostics: nil,
			expected:    false,
		},
		{
			name:        "empty slice",
			diagnostics: []*Diagnostic{},
			expected:    false,
		},
		{
			name:        "single warning",
			diagnostics: []*Diagnostic{{Severity: Warning}},
			expected:    true,
		},
		{
			name:        "single error",
			diagnostics: []*Diagnostic{{Severity: Error}},
			expected:    true,
		},
		{
			name:        "single info",
			diagnostics: []*Diagnostic{{Severity: Info}},
			expected:    true,
		},
		{
			name:        "single debug",
			diagnostics: []*Diagnostic{{Severity: Debug}},
			expected:    true,
		},
		{
			name: "multiple diagnostics",
			diagnostics: []*Diagnostic{
				{Severity: Warning},
				{Severity: Error},
				{Severity: Info},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, HasDiagnostics(tt.diagnostics))
		})
	}
}

func TestColorForSeverity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		severity Severity
	}{
		{name: "Error", severity: Error},
		{name: "Warning", severity: Warning},
		{name: "Info", severity: Info},
		{name: "Debug", severity: Debug},
		{name: "Unknown", severity: Severity(99)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := colourForSeverity(tt.severity)
			assert.NotNil(t, result, "colourForSeverity should always return a non-nil color")
		})
	}
}

func TestDeduplicateDiagnostics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []*Diagnostic
		expected int
	}{
		{
			name:     "nil slice",
			input:    nil,
			expected: 0,
		},
		{
			name:     "empty slice",
			input:    []*Diagnostic{},
			expected: 0,
		},
		{
			name: "no duplicates",
			input: []*Diagnostic{
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "error 1"},
				{SourcePath: "a.pkc", Location: Location{Line: 2, Column: 1}, Severity: Error, Message: "error 2"},
				{SourcePath: "b.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "error 1"},
			},
			expected: 3,
		},
		{
			name: "exact duplicates",
			input: []*Diagnostic{
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Warning, Message: "warning"},
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Warning, Message: "warning"},
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Warning, Message: "warning"},
			},
			expected: 1,
		},
		{
			name: "same location different messages are not duplicates",
			input: []*Diagnostic{
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "error 1"},
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "error 2"},
			},
			expected: 2,
		},
		{
			name: "same message different locations are not duplicates",
			input: []*Diagnostic{
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "same error"},
				{SourcePath: "a.pkc", Location: Location{Line: 2, Column: 1}, Severity: Error, Message: "same error"},
			},
			expected: 2,
		},
		{
			name: "same location and message different severity are not duplicates",
			input: []*Diagnostic{
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "issue"},
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Warning, Message: "issue"},
			},
			expected: 2,
		},
		{
			name: "same message and location different files are not duplicates",
			input: []*Diagnostic{
				{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "same"},
				{SourcePath: "b.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "same"},
			},
			expected: 2,
		},
		{
			name: "partial duplication scenario - same partial used by multiple pages",
			input: []*Diagnostic{
				{SourcePath: "partials/button.pk", Location: Location{Line: 10, Column: 9}, Severity: Warning, Message: "Missing p-key"},
				{SourcePath: "pages/home.pkc", Location: Location{Line: 5, Column: 1}, Severity: Error, Message: "Type error"},
				{SourcePath: "partials/button.pk", Location: Location{Line: 10, Column: 9}, Severity: Warning, Message: "Missing p-key"},
				{SourcePath: "pages/about.pkc", Location: Location{Line: 3, Column: 1}, Severity: Error, Message: "Type error"},
				{SourcePath: "partials/button.pk", Location: Location{Line: 10, Column: 9}, Severity: Warning, Message: "Missing p-key"},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := DeduplicateDiagnostics(tt.input)
			assert.Len(t, result, tt.expected)
		})
	}

	t.Run("preserves order - first occurrence wins", func(t *testing.T) {
		t.Parallel()

		input := []*Diagnostic{
			{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "first", Expression: "expression1"},
			{SourcePath: "b.pkc", Location: Location{Line: 2, Column: 2}, Severity: Warning, Message: "second", Expression: "expression2"},
			{SourcePath: "a.pkc", Location: Location{Line: 1, Column: 1}, Severity: Error, Message: "first", Expression: "different_expression"},
			{SourcePath: "c.pkc", Location: Location{Line: 3, Column: 3}, Severity: Info, Message: "third", Expression: "expression3"},
		}

		result := DeduplicateDiagnostics(input)

		require.Len(t, result, 3)
		assert.Equal(t, "first", result[0].Message)
		assert.Equal(t, "expression1", result[0].Expression, "First occurrence should be kept")
		assert.Equal(t, "second", result[1].Message)
		assert.Equal(t, "third", result[2].Message)
	})
}
