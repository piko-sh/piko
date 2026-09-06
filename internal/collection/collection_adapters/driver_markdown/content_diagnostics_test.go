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

package driver_markdown

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
)

func newTestDiagnostic(severity ast_domain.Severity, message string) *ast_domain.Diagnostic {
	return ast_domain.NewDiagnostic(severity, message, "", ast_domain.Location{Line: 3, Column: 5, Offset: 0}, "/content/blog/post.md")
}

func TestContentDiagnosticsError(t *testing.T) {
	manyErrors := make([]*ast_domain.Diagnostic, 0, maxReportedErrorsPerFile+2)
	for index := range maxReportedErrorsPerFile + 2 {
		manyErrors = append(manyErrors, newTestDiagnostic(ast_domain.Error, fmt.Sprintf("problem %d", index)))
	}

	testCases := []struct {
		name             string
		diagnostics      []*ast_domain.Diagnostic
		expectedContains []string
		expectErr        bool
	}{
		{
			name:        "no diagnostics",
			diagnostics: nil,
			expectErr:   false,
		},
		{
			name: "warnings only are logged, not returned",
			diagnostics: []*ast_domain.Diagnostic{
				newTestDiagnostic(ast_domain.Warning, "minor issue"),
				nil,
			},
			expectErr: false,
		},
		{
			name: "error diagnostics become an error naming the file",
			diagnostics: []*ast_domain.Diagnostic{
				newTestDiagnostic(ast_domain.Warning, "minor issue"),
				newTestDiagnostic(ast_domain.Error, "Invalid piko component syntax"),
			},
			expectErr:        true,
			expectedContains: []string{"blog/post.md", "Invalid piko component syntax", "line 3, col 5"},
		},
		{
			name:             "error diagnostics beyond the cap are summarised",
			diagnostics:      manyErrors,
			expectErr:        true,
			expectedContains: []string{"problem 0", "2 more omitted"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := contentDiagnosticsError(context.Background(), "blog/post.md", tc.diagnostics)

			if !tc.expectErr {
				assert.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, errMarkdownContent)
			for _, expected := range tc.expectedContains {
				assert.Contains(t, err.Error(), expected)
			}
			assert.NotContains(t, err.Error(), "minor issue")
		})
	}
}

func TestJoinBounded(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")
	third := errors.New("third")

	testCases := []struct {
		name          string
		errs          []error
		expected      []string
		limit         int
		expectNil     bool
		expectedLines int
	}{
		{name: "empty input joins to nil", errs: nil, limit: 2, expectNil: true},
		{name: "errors within the limit are all kept", errs: []error{first, second}, limit: 2, expected: []string{"first", "second"}, expectedLines: 2},
		{name: "errors beyond the limit are counted", errs: []error{first, second, third}, limit: 2, expected: []string{"first", "second", "1 more omitted"}, expectedLines: 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := append([]error(nil), tc.errs...)

			joined := joinBounded(tc.errs, tc.limit)

			assert.Equal(t, original, tc.errs)
			if tc.expectNil {
				assert.NoError(t, joined)
				return
			}
			require.Error(t, joined)
			assert.Len(t, strings.Split(joined.Error(), "\n"), tc.expectedLines)
			for _, expected := range tc.expected {
				assert.Contains(t, joined.Error(), expected)
			}
		})
	}
}
