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

package ast_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

func mustParse(t *testing.T, source string) *ast_domain.TemplateAST {
	t.Helper()
	tree, err := ast_domain.ParseAndTransform(context.Background(), source, "test")
	require.NoError(t, err, "ast_domain.ParseAndTransform returned an unexpected error")
	assertNoError(t, tree.Diagnostics, source)
	require.NotNil(t, tree, "ast_domain.ParseAndTransform returned a nil tree without error")
	return tree
}

func formatDiagsForTest(diagnostics []*ast_domain.Diagnostic) string {
	if len(diagnostics) == 0 {
		return "no diagnostics"
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "\n--- %d Diagnostics ---\n", len(diagnostics))
	for i, d := range diagnostics {
		fmt.Fprintf(&builder, "%d: [%s] at L%d:C%d: %s\n", i+1, d.Severity, d.Location.Line, d.Location.Column, d.Message)
	}
	builder.WriteString("-----------------------\n")
	return builder.String()
}

func assertNoError(t *testing.T, diagnostics []*ast_domain.Diagnostic, sourceContext string) {
	t.Helper()
	if ast_domain.HasErrors(diagnostics) {
		assert.Fail(t, fmt.Sprintf("Expected no errors, but got some for input:\n---\n%s\n---\nDiagnostics:%s", sourceContext, formatDiagsForTest(diagnostics)))
	}
}
