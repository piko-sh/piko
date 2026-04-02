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

package annotator_domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestNewParseDiagnosticError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		sourcePath     string
		templateSource string
		diagnostics    []*ast_domain.Diagnostic
	}{
		{
			name:           "creates error with empty diagnostics",
			diagnostics:    []*ast_domain.Diagnostic{},
			sourcePath:     "/test/file.pk",
			templateSource: "<div>test</div>",
		},
		{
			name: "creates error with single diagnostic",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "test error", Severity: ast_domain.Error},
			},
			sourcePath:     "/test/file.pk",
			templateSource: "<div>test</div>",
		},
		{
			name: "creates error with multiple diagnostics",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "error 1", Severity: ast_domain.Error},
				{Message: "error 2", Severity: ast_domain.Error},
			},
			sourcePath:     "/test/component.pk",
			templateSource: "<template>content</template>",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := NewParseDiagnosticError(tc.diagnostics, tc.sourcePath, tc.templateSource)

			require.NotNil(t, result)
			assert.Equal(t, tc.sourcePath, result.SourcePath)
			assert.Equal(t, tc.templateSource, result.TemplateSource)
			assert.Equal(t, tc.diagnostics, result.Diagnostics)
		})
	}
}

func TestParseDiagnosticError_Error(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		sourcePath  string
		expected    string
		diagnostics []*ast_domain.Diagnostic
	}{
		{
			name:        "empty diagnostics",
			diagnostics: []*ast_domain.Diagnostic{},
			sourcePath:  "/test/file.pk",
			expected:    "found 0 parsing errors in /test/file.pk",
		},
		{
			name: "single diagnostic uses singular form",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "test error"},
			},
			sourcePath: "/test/file.pk",
			expected:   "found 1 parsing error in /test/file.pk",
		},
		{
			name: "multiple diagnostics uses plural form",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "error 1"},
				{Message: "error 2"},
			},
			sourcePath: "/test/file.pk",
			expected:   "found 2 parsing errors in /test/file.pk",
		},
		{
			name: "many diagnostics",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "1"}, {Message: "2"}, {Message: "3"},
				{Message: "4"}, {Message: "5"},
			},
			sourcePath: "/components/button.pk",
			expected:   "found 5 parsing errors in /components/button.pk",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := NewParseDiagnosticError(tc.diagnostics, tc.sourcePath, "")
			result := err.Error()

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewSemanticError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		diagnostics []*ast_domain.Diagnostic
	}{
		{
			name:        "creates error with empty diagnostics",
			diagnostics: []*ast_domain.Diagnostic{},
		},
		{
			name: "creates error with single diagnostic",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "undefined variable", Severity: ast_domain.Error},
			},
		},
		{
			name: "creates error with mixed severity diagnostics",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "error 1", Severity: ast_domain.Error},
				{Message: "warning 1", Severity: ast_domain.Warning},
				{Message: "error 2", Severity: ast_domain.Error},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := NewSemanticError(tc.diagnostics)

			require.NotNil(t, result)
			assert.Equal(t, tc.diagnostics, result.Diagnostics)
		})
	}
}

func TestSemanticError_Error(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		expected    string
		diagnostics []*ast_domain.Diagnostic
	}{
		{
			name:        "empty diagnostics",
			diagnostics: []*ast_domain.Diagnostic{},
			expected:    "found 0 semantic validation errors and 0 semantic validation warnings",
		},
		{
			name: "single error uses singular form",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "test error", Severity: ast_domain.Error},
			},
			expected: "found 1 semantic validation error",
		},
		{
			name: "multiple errors only",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "error 1", Severity: ast_domain.Error},
				{Message: "error 2", Severity: ast_domain.Error},
			},
			expected: "found 2 semantic validation errors and 0 semantic validation warnings",
		},
		{
			name: "errors and warnings",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "error 1", Severity: ast_domain.Error},
				{Message: "warning 1", Severity: ast_domain.Warning},
				{Message: "warning 2", Severity: ast_domain.Warning},
			},
			expected: "found 1 semantic validation errors and 2 semantic validation warnings",
		},
		{
			name: "warnings only",
			diagnostics: []*ast_domain.Diagnostic{
				{Message: "warning 1", Severity: ast_domain.Warning},
				{Message: "warning 2", Severity: ast_domain.Warning},
			},
			expected: "found 0 semantic validation errors and 2 semantic validation warnings",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := NewSemanticError(tc.diagnostics)
			result := err.Error()

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNewCircularDependencyError(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		path []string
	}{
		{
			name: "creates error with empty path",
			path: []string{},
		},
		{
			name: "creates error with single element path",
			path: []string{"A"},
		},
		{
			name: "creates error with two element cycle",
			path: []string{"A", "B"},
		},
		{
			name: "creates error with multi-element cycle",
			path: []string{"A", "B", "C", "A"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := NewCircularDependencyError(tc.path)

			require.NotNil(t, result)
			assert.Equal(t, tc.path, result.Path)
		})
	}
}

func TestCircularDependencyError_Error(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected string
		path     []string
	}{
		{
			name:     "empty path",
			path:     []string{},
			expected: "circular dependency detected: ",
		},
		{
			name:     "single element",
			path:     []string{"A"},
			expected: "circular dependency detected: A",
		},
		{
			name:     "two elements",
			path:     []string{"A", "B"},
			expected: "circular dependency detected: A -> B",
		},
		{
			name:     "full cycle",
			path:     []string{"components/a", "components/b", "components/c", "components/a"},
			expected: "circular dependency detected: components/a -> components/b -> components/c -> components/a",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := NewCircularDependencyError(tc.path)
			result := err.Error()

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetDiagnosticCounts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		diagnostics   []*ast_domain.Diagnostic
		expectedDebug int
		expectedInfo  int
		expectedWarn  int
		expectedError int
	}{
		{
			name:          "empty diagnostics",
			diagnostics:   []*ast_domain.Diagnostic{},
			expectedDebug: 0,
			expectedInfo:  0,
			expectedWarn:  0,
			expectedError: 0,
		},
		{
			name: "all errors",
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Error},
				{Severity: ast_domain.Error},
			},
			expectedDebug: 0,
			expectedInfo:  0,
			expectedWarn:  0,
			expectedError: 2,
		},
		{
			name: "all warnings",
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Warning},
				{Severity: ast_domain.Warning},
				{Severity: ast_domain.Warning},
			},
			expectedDebug: 0,
			expectedInfo:  0,
			expectedWarn:  3,
			expectedError: 0,
		},
		{
			name: "all info",
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Info},
			},
			expectedDebug: 0,
			expectedInfo:  1,
			expectedWarn:  0,
			expectedError: 0,
		},
		{
			name: "all debug",
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Debug},
				{Severity: ast_domain.Debug},
			},
			expectedDebug: 2,
			expectedInfo:  0,
			expectedWarn:  0,
			expectedError: 0,
		},
		{
			name: "mixed severities",
			diagnostics: []*ast_domain.Diagnostic{
				{Severity: ast_domain.Debug},
				{Severity: ast_domain.Info},
				{Severity: ast_domain.Warning},
				{Severity: ast_domain.Error},
				{Severity: ast_domain.Error},
				{Severity: ast_domain.Warning},
			},
			expectedDebug: 1,
			expectedInfo:  1,
			expectedWarn:  2,
			expectedError: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			debugCount, infoCount, warnCount, errorCount := getDiagnosticCounts(tc.diagnostics)

			assert.Equal(t, tc.expectedDebug, debugCount, "debug count mismatch")
			assert.Equal(t, tc.expectedInfo, infoCount, "info count mismatch")
			assert.Equal(t, tc.expectedWarn, warnCount, "warning count mismatch")
			assert.Equal(t, tc.expectedError, errorCount, "error count mismatch")
		})
	}
}

func TestFormatAllDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("returns empty string for no diagnostics", func(t *testing.T) {
		t.Parallel()

		result := FormatAllDiagnostics(nil, nil)
		assert.Empty(t, result)

		result = FormatAllDiagnostics([]*ast_domain.Diagnostic{}, map[string][]byte{})
		assert.Empty(t, result)
	})

	t.Run("handles diagnostic without source path", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*ast_domain.Diagnostic{
			{
				Message:    "test error",
				SourcePath: "",
				Severity:   ast_domain.Error,
			},
		}

		result := FormatAllDiagnostics(diagnostics, nil)

		assert.Contains(t, result, "Source path was missing for this diagnostic")
	})

	t.Run("handles missing source content", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*ast_domain.Diagnostic{
			{
				Message:    "test error",
				SourcePath: "/missing/file.pk",
				Severity:   ast_domain.Error,
			},
		}

		result := FormatAllDiagnostics(diagnostics, map[string][]byte{})

		assert.Contains(t, result, "source code unavailable for formatting")
		assert.Contains(t, result, "/missing/file.pk")
	})

	t.Run("groups diagnostics by file", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*ast_domain.Diagnostic{
			{Message: "error 1", SourcePath: "/a.pk", Severity: ast_domain.Error},
			{Message: "error 2", SourcePath: "/b.pk", Severity: ast_domain.Error},
			{Message: "error 3", SourcePath: "/a.pk", Severity: ast_domain.Error},
		}

		result := FormatAllDiagnostics(diagnostics, map[string][]byte{})

		assert.Contains(t, result, "/a.pk")
		assert.Contains(t, result, "/b.pk")

		assert.True(t, strings.Contains(result, "2 issue(s) in /a.pk") ||
			strings.Contains(result, "1 issue(s) in /a.pk"))
	})

	t.Run("sorts files alphabetically", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*ast_domain.Diagnostic{
			{Message: "z error", SourcePath: "/z/file.pk", Severity: ast_domain.Error},
			{Message: "a error", SourcePath: "/a/file.pk", Severity: ast_domain.Error},
			{Message: "m error", SourcePath: "/m/file.pk", Severity: ast_domain.Error},
		}

		result := FormatAllDiagnostics(diagnostics, map[string][]byte{})

		aIndex := strings.Index(result, "/a/file.pk")
		mIndex := strings.Index(result, "/m/file.pk")
		zIndex := strings.Index(result, "/z/file.pk")

		assert.True(t, aIndex < mIndex, "a should appear before m")
		assert.True(t, mIndex < zIndex, "m should appear before z")
	})
}
