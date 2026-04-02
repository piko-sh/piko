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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParse(t *testing.T, source string) *TemplateAST {
	t.Helper()
	tree, err := ParseAndTransform(context.Background(), source, "test")
	require.NoError(t, err, "ParseAndTransform returned an unexpected error")
	assertNoError(t, tree.Diagnostics, source)
	require.NotNil(t, tree, "ParseAndTransform returned a nil tree without error")
	return tree
}

func mustParseExpr(t *testing.T, expressionString string) Expression {
	t.Helper()
	ctx := context.Background()
	parser := NewExpressionParser(ctx, expressionString, "test")
	expression, diagnostics := parser.ParseExpression(ctx)

	assertNoError(t, diagnostics, expressionString)
	require.NotNil(t, expression, "ParseExpression returned a nil expression without error")

	return expression
}

func formatDiagsForTest(diagnostics []*Diagnostic) string {
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

func assertNoError(t *testing.T, diagnostics []*Diagnostic, sourceContext string) {
	t.Helper()
	if HasErrors(diagnostics) {
		assert.Fail(t, fmt.Sprintf("Expected no errors, but got some for input:\n---\n%s\n---\nDiagnostics:%s", sourceContext, formatDiagsForTest(diagnostics)))
	}
}

func assertHasWarning(t *testing.T, diagnostics []*Diagnostic, msgSubstring string) {
	t.Helper()
	require.NotEmpty(t, diagnostics, "Expected a warning diagnostic, but got none.")

	if msgSubstring != "" {
		found := false
		for _, d := range diagnostics {
			if d.Severity == Warning && strings.Contains(d.Message, msgSubstring) {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected a warning containing substring '%s', but none was found.\nDiagnostics:%s", msgSubstring, formatDiagsForTest(diagnostics))
	}
}

func assertHasError(t *testing.T, diagnostics []*Diagnostic, msgSubstring string) {
	t.Helper()
	require.True(t, HasErrors(diagnostics), "Expected an error diagnostic, but got none.")

	if msgSubstring != "" {
		found := false
		for _, d := range diagnostics {
			if d.Severity == Error && strings.Contains(d.Message, msgSubstring) {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected an error containing substring '%s', but none was found.\nDiagnostics:%s", msgSubstring, formatDiagsForTest(diagnostics))
	}
}

func assertExprString(t *testing.T, expected string, expression Expression) {
	t.Helper()
	require.NotNil(t, expression, "Cannot call String() on a nil expression")
	assert.Equal(t, expected, expression.String())
}

func findNodeByTag(t *testing.T, root *TemplateNode, tagName string) *TemplateNode {
	t.Helper()
	var finder func(*TemplateNode) *TemplateNode
	finder = func(node *TemplateNode) *TemplateNode {
		if node.NodeType == NodeElement && node.TagName == tagName {
			return node
		}
		for _, child := range node.Children {
			if found := finder(child); found != nil {
				return found
			}
		}
		return nil
	}

	foundNode := finder(root)
	require.NotNil(t, foundNode, "Failed to find a node with tag <%s> in the AST", tagName)
	return foundNode
}

func findNodeByTagFromRoots(t *testing.T, roots []*TemplateNode, tagName string) *TemplateNode {
	t.Helper()
	for _, root := range roots {
		var finder func(*TemplateNode) *TemplateNode
		finder = func(node *TemplateNode) *TemplateNode {
			if node.NodeType == NodeElement && node.TagName == tagName {
				return node
			}
			for _, child := range node.Children {
				if found := finder(child); found != nil {
					return found
				}
			}
			return nil
		}
		if found := finder(root); found != nil {
			return found
		}
	}
	require.Fail(t, fmt.Sprintf("Failed to find a node with tag <%s> in any root node", tagName))
	return nil
}

func getDynamicAttribute(t *testing.T, node *TemplateNode, name string) *DynamicAttribute {
	t.Helper()
	for i := range node.DynamicAttributes {
		if node.DynamicAttributes[i].Name == name {
			return &node.DynamicAttributes[i]
		}
	}
	require.Fail(t, fmt.Sprintf("Failed to find dynamic attribute with name ':%s' on node <%s>", name, node.TagName))
	return nil
}

func getAttribute(t *testing.T, node *TemplateNode, name string) *HTMLAttribute {
	t.Helper()
	for i := range node.Attributes {
		if node.Attributes[i].Name == name {
			return &node.Attributes[i]
		}
	}
	require.Fail(t, fmt.Sprintf("Failed to find attribute with name '%s' on node <%s>", name, node.TagName))
	return nil
}
