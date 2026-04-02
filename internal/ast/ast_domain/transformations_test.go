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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDirectiveExpressions(t *testing.T) {
	t.Run("parses valid expressions and populates Expression field", func(t *testing.T) {
		tree := &TemplateAST{
			RootNodes: []*TemplateNode{
				{
					NodeType: NodeElement,
					TagName:  "div",
					Directives: []Directive{
						{Type: DirectiveIf, RawExpression: "user.isActive"},
					},
					DynamicAttributes: []DynamicAttribute{
						{Name: "title", RawExpression: "user.name"},
					},
				},
			},
		}

		applyTemplateTransformations(context.Background(), tree)

		node := tree.RootNodes[0]

		require.NotNil(t, node.DirIf, "DirIf should be populated after transformations")
		require.NotNil(t, node.DirIf.Expression, "Expression inside DirIf should be populated")
		assert.IsType(t, &MemberExpression{}, node.DirIf.Expression)
		assert.Equal(t, "user.isActive", node.DirIf.Expression.String())

		require.Len(t, node.DynamicAttributes, 1)
		dynAttr := node.DynamicAttributes[0]
		require.NotNil(t, dynAttr.RawExpression)
		assert.IsType(t, &MemberExpression{}, dynAttr.Expression)
		assert.Equal(t, "user.name", dynAttr.Expression.String())

		assert.Empty(t, tree.Diagnostics, "Should have no diagnostics for valid expressions")
	})

	t.Run("adds diagnostics for invalid expressions and leaves Expression nil", func(t *testing.T) {
		testCases := []struct {
			checkNode     func(t *testing.T, node *TemplateNode, tree *TemplateAST)
			name          string
			errorContains string
			directive     Directive
			expectedCol   int
		}{
			{
				name: "invalid p-if expression with dangling operator",
				directive: Directive{
					Type:          DirectiveIf,
					RawExpression: "user.isActive &&",
					Location:      Location{Line: 1, Column: 10},
				},
				errorContains: "Expected expression on the right side of the operator",
				expectedCol:   24,
				checkNode: func(t *testing.T, node *TemplateNode, tree *TemplateAST) {
					assert.Nil(t, node.DirIf, "DirIf should be nil due to parsing failure")
				},
			},
			{
				name: "unmatched parenthesis in p-for",
				directive: Directive{
					Type:          DirectiveFor,
					RawExpression: "item in items.filter((i => i > 0)",
					Location:      Location{Line: 5, Column: 15},
				},
				errorContains: "Expected ')'",
				expectedCol:   36,
				checkNode: func(t *testing.T, node *TemplateNode, tree *TemplateAST) {
					assert.Nil(t, node.DirFor, "DirFor should be nil due to parsing failure")
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tree := &TemplateAST{
					RootNodes: []*TemplateNode{
						{
							NodeType:   NodeElement,
							TagName:    "div",
							Location:   Location{Line: tc.directive.Location.Line, Column: 1},
							Directives: []Directive{tc.directive},
						},
					},
				}

				applyTemplateTransformations(context.Background(), tree)

				require.Len(t, tree.RootNodes, 1)
				node := tree.RootNodes[0]
				tc.checkNode(t, node, tree)

				assertHasError(t, tree.Diagnostics, tc.errorContains)
				require.Len(t, tree.Diagnostics, 1)
				diagnostic := tree.Diagnostics[0]
				assert.Equal(t, Error, diagnostic.Severity)
				assert.Equal(t, tc.directive.Location.Line, diagnostic.Location.Line, "Diagnostic line number should match directive's line")
				assert.Equal(t, tc.expectedCol, diagnostic.Location.Column, "Diagnostic column should be adjusted relative to the attribute value's location")
			})
		}
	})
}

func TestLinkIfElseChains(t *testing.T) {

	t.Run("links a correct p-if -> p-else-if -> p-else chain", func(t *testing.T) {
		source := `
			<div p-if="cond1">A</div>
			<div p-else-if="cond2">B</div>
			<div p-else>C</div>
		`
		tree := parseForValidation(t, source)
		TidyAST(context.Background(), tree)

		require.Empty(t, tree.Diagnostics)
		require.Len(t, tree.RootNodes, 3)

		ifNode := tree.RootNodes[0]
		elseIfNode := tree.RootNodes[1]
		elseNode := tree.RootNodes[2]

		require.NotNil(t, ifNode.DirIf)
		assert.Nil(t, ifNode.DirIf.ChainKey, "The starting p-if should not have a ChainKey")
		require.NotNil(t, ifNode.Key, "The p-if node must have a key")

		require.Nil(t, elseIfNode.DirIf)
		require.NotNil(t, elseIfNode.DirElseIf)
		require.NotNil(t, elseIfNode.DirElseIf.ChainKey, "p-else-if should have a ChainKey")
		assert.Equal(t, ifNode.Key, elseIfNode.DirElseIf.ChainKey, "ChainKey of p-else-if should match the key of p-if")

		require.Nil(t, elseNode.DirIf)
		require.NotNil(t, elseNode.DirElse)
		require.NotNil(t, elseNode.DirElse.ChainKey, "p-else should have a ChainKey")
		assert.Equal(t, ifNode.Key, elseNode.DirElse.ChainKey, "ChainKey of p-else should match the key of p-if")
	})

	t.Run("a node in the middle breaks the chain", func(t *testing.T) {
		source := `
			<div p-if="cond1">A</div>
			<p>Separator</p>
			<div p-else>C</div>
		`
		tree := parseForValidation(t, source)
		TidyAST(context.Background(), tree)

		require.Len(t, tree.RootNodes, 3)
		elseNode := tree.RootNodes[2]

		require.NotNil(t, elseNode.DirElse)
		assert.Nil(t, elseNode.DirElse.ChainKey, "ChainKey should be nil because the chain was broken")

		ValidateAST(tree)
		assertHasValidationError(t, tree, Error, "'p-else' directive must immediately follow")
	})

	t.Run("a comment does NOT break the chain", func(t *testing.T) {
		source := `
			<div p-if="cond1">A</div>
            <!-- A comment is fine -->
			<div p-else>C</div>
		`
		tree := parseForValidation(t, source)
		TidyAST(context.Background(), tree)

		require.Len(t, tree.RootNodes, 3)
		ifNode := tree.RootNodes[0]
		elseNode := tree.RootNodes[2]

		require.NotNil(t, ifNode.Key)
		require.NotNil(t, elseNode.DirElse)
		require.NotNil(t, elseNode.DirElse.ChainKey)
		assert.Equal(t, ifNode.Key, elseNode.DirElse.ChainKey)
	})
}

func TestFullTransformationPipeline(t *testing.T) {
	source := `
		<div>
			<p p-if="user.isActive"></p>
			<p p-else-if="user.isAdmin"></p>
			<p p-else></p>
		</div>
		<span> </span>
	`
	tree := mustParse(t, source)

	require.Len(t, tree.RootNodes, 2, "Should have two root element nodes after whitespace pruning")

	divNode := tree.RootNodes[0]
	require.Equal(t, "div", divNode.TagName)
	require.Len(t, divNode.Children, 3)

	spanNode := tree.RootNodes[1]
	require.Equal(t, "span", spanNode.TagName)
	assert.Len(t, spanNode.Children, 1)

	pIfNode := divNode.Children[0]
	pElseIfNode := divNode.Children[1]
	pElseNode := divNode.Children[2]

	require.NotNil(t, pIfNode.DirIf, "p-if should have DirIf populated")
	assertExprString(t, "user.isActive", pIfNode.DirIf.Expression)
	assert.Equal(t, DirectiveIf, pIfNode.DirIf.Type)

	require.NotNil(t, pElseIfNode.DirElseIf, "p-else-if should have DirElseIf populated")
	assertExprString(t, "user.isAdmin", pElseIfNode.DirElseIf.Expression)
	assert.Equal(t, DirectiveElseIf, pElseIfNode.DirElseIf.Type)
	require.NotNil(t, pElseIfNode.DirElseIf.ChainKey)
	assert.Equal(t, pIfNode.Key, pElseIfNode.DirElseIf.ChainKey)

	require.NotNil(t, pElseNode.DirElse, "p-else should have DirElse populated")
	assert.Nil(t, pElseNode.DirElse.Expression, "p-else directive should have a nil expression")
	require.NotNil(t, pElseNode.DirElse.ChainKey)
	assert.Equal(t, pIfNode.Key, pElseNode.DirElse.ChainKey)
}

func TestSetSequentialProcessing(t *testing.T) {
	t.Run("sets true and reads back true", func(t *testing.T) {
		setSequentialProcessing(true)
		t.Cleanup(func() {
			setSequentialProcessing(false)
		})
		assert.True(t, forceSequentialProcessing.Load(),
			"forceSequentialProcessing should be true after setting it to true")
	})

	t.Run("sets false and reads back false", func(t *testing.T) {
		setSequentialProcessing(true)
		setSequentialProcessing(false)
		t.Cleanup(func() {
			setSequentialProcessing(false)
		})
		assert.False(t, forceSequentialProcessing.Load(),
			"forceSequentialProcessing should be false after setting it to false")
	})
}

func TestValidateSlotDirective(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		rawExpr        string
		wantTrimmed    string
		wantErrMessage string
		wantDiagLen    int
	}{
		{
			name:           "empty value produces error",
			rawExpr:        "",
			wantDiagLen:    1,
			wantErrMessage: "p-slot value cannot be empty",
		},
		{
			name:           "whitespace-only value produces error",
			rawExpr:        "   ",
			wantDiagLen:    1,
			wantErrMessage: "p-slot value cannot be empty",
		},
		{
			name:        "default value is trimmed and valid",
			rawExpr:     "default",
			wantDiagLen: 0,
			wantTrimmed: "default",
		},
		{
			name:        "named slot with whitespace is trimmed",
			rawExpr:     " header ",
			wantDiagLen: 0,
			wantTrimmed: "header",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			d := &Directive{
				Type:          DirectiveSlot,
				RawExpression: tc.rawExpr,
				Location:      Location{Line: 1, Column: 1},
			}
			diagnostics := validateSlotDirective(d, "test.pk")
			assert.Len(t, diagnostics, tc.wantDiagLen)

			if tc.wantDiagLen > 0 {
				assert.Contains(t, diagnostics[0].Message, tc.wantErrMessage)
				assert.Equal(t, Error, diagnostics[0].Severity)
			} else {
				assert.Equal(t, tc.wantTrimmed, d.RawExpression,
					"RawExpression should be trimmed")
			}
		})
	}
}

func TestAssignKeys(t *testing.T) {
	t.Parallel()

	source := `
		<div>
			<p></p>
			<span></span>
		</div>
		<section></section>
	`
	tree := mustParse(t, source)

	require.Len(t, tree.RootNodes, 2, "Should have two root nodes")

	divNode := tree.RootNodes[0]
	sectionNode := tree.RootNodes[1]

	require.NotNil(t, divNode.Key, "div should have a key")
	require.NotNil(t, sectionNode.Key, "section should have a key")

	divKeyString := divNode.Key.String()
	sectionKeyString := sectionNode.Key.String()

	assert.Contains(t, divKeyString, "r", "div key should contain the root prefix 'r'")
	assert.Contains(t, sectionKeyString, "r", "section key should contain the root prefix 'r'")
	assert.NotEqual(t, divKeyString, sectionKeyString, "Root nodes should have different keys")

	require.Len(t, divNode.Children, 2, "div should have two children")
	pNode := divNode.Children[0]
	spanNode := divNode.Children[1]

	require.NotNil(t, pNode.Key, "p should have a key")
	require.NotNil(t, spanNode.Key, "span should have a key")

	pKeyString := pNode.Key.String()
	spanKeyString := spanNode.Key.String()

	assert.NotEqual(t, pKeyString, spanKeyString, "Sibling children should have different keys")
	assert.Contains(t, pKeyString, ":", "Child key should contain ':' separator")
	assert.Contains(t, spanKeyString, ":", "Child key should contain ':' separator")
}
