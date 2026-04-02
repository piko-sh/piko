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

package driven_code_emitter_go_literal

import (
	"context"
	goast "go/ast"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
)

func TestEmitChain_SimpleIf(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{
		emitNodeFunc: func(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {

			return []goast.Stmt{
				&goast.ExprStmt{X: cachedIdent("bodyStmt")},
			}, 1, nil
		},
	}

	ifEmitter := newIfEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	condVarName := "isActive"
	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      &ast_domain.StringLiteral{Value: "if-1"},
		DirIf: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: "isActive",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &condVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
		},
	}

	parentSlice := cachedIdent("nodes")
	statements, nodesConsumed, diagnostics := ifEmitter.emitChain(context.Background(), node, []*ast_domain.TemplateNode{node}, 0, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)
	assert.Equal(t, 1, nodesConsumed, "Should consume only the if node")

	foundIf := false
	for _, statement := range statements {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundIf = true
			assert.NotNil(t, ifStmt.Cond, "If condition should be set")
			assert.NotNil(t, ifStmt.Body, "If body should be set")
			assert.Nil(t, ifStmt.Else, "Simple if should not have else")
			break
		}
	}
	assert.True(t, foundIf, "Should generate an if statement")
}

func TestEmitChain_IfElse(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{
		emitNodeFunc: func(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
			return []goast.Stmt{
				&goast.ExprStmt{X: cachedIdent("bodyStmt")},
			}, 1, nil
		},
	}

	ifEmitter := newIfEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	chainKey := &ast_domain.StringLiteral{Value: "chain-1"}
	condVarName := "hasData"

	ifNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      chainKey,
		DirIf: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: "hasData",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &condVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
		},
	}

	elseNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      &ast_domain.StringLiteral{Value: "else-1"},
		DirElse: &ast_domain.Directive{
			ChainKey: chainKey,
		},
	}

	siblings := []*ast_domain.TemplateNode{ifNode, elseNode}
	parentSlice := cachedIdent("nodes")
	statements, nodesConsumed, diagnostics := ifEmitter.emitChain(context.Background(), ifNode, siblings, 0, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)
	assert.Equal(t, 2, nodesConsumed, "Should consume both if and else nodes")

	foundIf := false
	for _, statement := range statements {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundIf = true
			assert.NotNil(t, ifStmt.Cond)
			assert.NotNil(t, ifStmt.Body)
			assert.NotNil(t, ifStmt.Else, "Should have else block")

			_, isBlock := ifStmt.Else.(*goast.BlockStmt)
			assert.True(t, isBlock, "Else should be a block statement")
			break
		}
	}
	assert.True(t, foundIf, "Should generate an if statement")
}

func TestEmitChain_IfElseIfElse(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{
		emitNodeFunc: func(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
			return []goast.Stmt{
				&goast.ExprStmt{X: cachedIdent("bodyStmt")},
			}, 1, nil
		},
	}

	ifEmitter := newIfEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	chainKey := &ast_domain.StringLiteral{Value: "chain-2"}
	cond1 := "condition1"
	cond2 := "condition2"

	ifNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      chainKey,
		DirIf: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: "condition1",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &cond1,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
		},
	}

	elseIfNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      &ast_domain.StringLiteral{Value: "elseif-1"},
		DirElseIf: &ast_domain.Directive{
			ChainKey: chainKey,
			Expression: &ast_domain.Identifier{
				Name: "condition2",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &cond2,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
		},
	}

	elseNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      &ast_domain.StringLiteral{Value: "else-2"},
		DirElse: &ast_domain.Directive{
			ChainKey: chainKey,
		},
	}

	siblings := []*ast_domain.TemplateNode{ifNode, elseIfNode, elseNode}
	parentSlice := cachedIdent("nodes")
	statements, nodesConsumed, diagnostics := ifEmitter.emitChain(context.Background(), ifNode, siblings, 0, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)
	assert.Equal(t, 3, nodesConsumed, "Should consume all three nodes")

	foundIf := false
	for _, statement := range statements {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundIf = true

			assert.NotNil(t, ifStmt.Cond)
			assert.NotNil(t, ifStmt.Body)
			assert.NotNil(t, ifStmt.Else, "Should have else-if")

			elseIfStmt, isElseIf := ifStmt.Else.(*goast.IfStmt)
			assert.True(t, isElseIf, "First else should be else-if (IfStmt)")

			if elseIfStmt != nil {
				assert.NotNil(t, elseIfStmt.Cond, "Else-if should have condition")
				assert.NotNil(t, elseIfStmt.Body)
				assert.NotNil(t, elseIfStmt.Else, "Should have final else")

				_, isFinalElse := elseIfStmt.Else.(*goast.BlockStmt)
				assert.True(t, isFinalElse, "Final else should be BlockStmt")
			}
			break
		}
	}
	assert.True(t, foundIf, "Should generate an if statement")
}

func TestEmitChain_MultipleElseIf(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{
		emitNodeFunc: func(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
			return []goast.Stmt{
				&goast.ExprStmt{X: cachedIdent("bodyStmt")},
			}, 1, nil
		},
	}

	ifEmitter := newIfEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	chainKey := &ast_domain.StringLiteral{Value: "chain-3"}
	cond1 := "cond1"
	cond2 := "cond2"
	cond3 := "cond3"

	nodes := []*ast_domain.TemplateNode{
		{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Key:      chainKey,
			DirIf: &ast_domain.Directive{
				Expression: &ast_domain.Identifier{
					Name: "cond1",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &cond1,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
			},
		},
		{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Key:      &ast_domain.StringLiteral{Value: "elseif-1"},
			DirElseIf: &ast_domain.Directive{
				ChainKey: chainKey,
				Expression: &ast_domain.Identifier{
					Name: "cond2",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &cond2,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
			},
		},
		{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Key:      &ast_domain.StringLiteral{Value: "elseif-2"},
			DirElseIf: &ast_domain.Directive{
				ChainKey: chainKey,
				Expression: &ast_domain.Identifier{
					Name: "cond3",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &cond3,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
			},
		},
	}

	parentSlice := cachedIdent("nodes")
	statements, nodesConsumed, diagnostics := ifEmitter.emitChain(context.Background(), nodes[0], nodes, 0, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)
	assert.Equal(t, 3, nodesConsumed, "Should consume all three conditional nodes")
}

func TestEmitChain_WithWhitespace(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{
		emitNodeFunc: func(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
			return []goast.Stmt{
				&goast.ExprStmt{X: cachedIdent("bodyStmt")},
			}, 1, nil
		},
	}

	ifEmitter := newIfEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	chainKey := &ast_domain.StringLiteral{Value: "chain-ws"}
	condVar := "condition"

	ifNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      chainKey,
		DirIf: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: "condition",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &condVar,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
		},
	}

	whitespaceNode := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "   \n  ",
	}

	commentNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeComment,
	}

	elseNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      &ast_domain.StringLiteral{Value: "else-ws"},
		DirElse: &ast_domain.Directive{
			ChainKey: chainKey,
		},
	}

	siblings := []*ast_domain.TemplateNode{ifNode, whitespaceNode, commentNode, elseNode}
	parentSlice := cachedIdent("nodes")
	statements, nodesConsumed, diagnostics := ifEmitter.emitChain(context.Background(), ifNode, siblings, 0, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)
	assert.Equal(t, 4, nodesConsumed, "Should consume if, whitespace, comment, and else")

	foundIf := false
	for _, statement := range statements {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundIf = true
			assert.NotNil(t, ifStmt.Else, "Should have else despite intervening whitespace")
			break
		}
	}
	assert.True(t, foundIf)
}

func TestEmitChain_BrokenChainKey(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{
		emitNodeFunc: func(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
			return []goast.Stmt{
				&goast.ExprStmt{X: cachedIdent("bodyStmt")},
			}, 1, nil
		},
	}

	ifEmitter := newIfEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	chainKey1 := &ast_domain.StringLiteral{Value: "chain-1"}
	chainKey2 := &ast_domain.StringLiteral{Value: "chain-2"}
	condVar := "condition"

	ifNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      chainKey1,
		DirIf: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: "condition",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &condVar,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
		},
	}

	elseNode := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Key:      &ast_domain.StringLiteral{Value: "else-broken"},
		DirElse: &ast_domain.Directive{
			ChainKey: chainKey2,
		},
	}

	siblings := []*ast_domain.TemplateNode{ifNode, elseNode}
	parentSlice := cachedIdent("nodes")
	statements, nodesConsumed, diagnostics := ifEmitter.emitChain(context.Background(), ifNode, siblings, 0, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)
	assert.Equal(t, 1, nodesConsumed, "Should only consume if node, not else with wrong chain key")

	foundIf := false
	for _, statement := range statements {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundIf = true
			assert.Nil(t, ifStmt.Else, "Should not have else due to mismatched chain key")
			break
		}
	}
	assert.True(t, foundIf)
}

func TestEmitChain_TruthinessWrapping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		conditionType string
		expectWrapped bool
	}{
		{
			name:          "bool type - no wrapping",
			conditionType: "bool",
			expectWrapped: false,
		},
		{
			name:          "int type - needs wrapping",
			conditionType: "int",
			expectWrapped: true,
		},
		{
			name:          "string type - needs wrapping",
			conditionType: "string",
			expectWrapped: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

			mockAstBuilder := &mockAstBuilder{
				emitNodeFunc: func(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
					return []goast.Stmt{
						&goast.ExprStmt{X: cachedIdent("bodyStmt")},
					}, 1, nil
				},
			}

			ifEmitter := newIfEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

			condVar := "condition"
			node := &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Key:      &ast_domain.StringLiteral{Value: "if-truthiness"},
				DirIf: &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name: "condition",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: &condVar,
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent(tc.conditionType),
							},
						},
					},
				},
			}

			parentSlice := cachedIdent("nodes")
			statements, _, _ := ifEmitter.emitChain(context.Background(), node, []*ast_domain.TemplateNode{node}, 0, parentSlice, "", "")

			for _, statement := range statements {
				if ifStmt, ok := statement.(*goast.IfStmt); ok {
					if tc.expectWrapped {

						_, isBinary := ifStmt.Cond.(*goast.BinaryExpr)
						assert.True(t, isBinary, "Non-bool condition should be wrapped in optimised comparison")
					} else {
						_, isIdent := ifStmt.Cond.(*goast.Ident)
						assert.True(t, isIdent, "Bool condition should not be wrapped")
					}
					break
				}
			}
		})
	}
}

func TestBuildIfBlock_NilDirective(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{}
	ifEmitter := newIfEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirIf:    nil,
	}

	parentSlice := cachedIdent("nodes")
	ifStmt, prereqs, nodesInChain, diagnostics := ifEmitter.buildIfBlock(context.Background(), node, parentSlice)

	assert.Nil(t, ifStmt, "Should return nil for node without p-if")
	assert.Empty(t, prereqs)
	assert.Equal(t, 0, nodesInChain)
	assert.Empty(t, diagnostics)
}

func TestNodeContainsForLoops(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockAstBuilder := &mockAstBuilder{}
	ifEmitter := newIfEmitter(mockEmitter, nil, mockAstBuilder)

	testCases := []struct {
		node           *ast_domain.TemplateNode
		name           string
		expectForLoops bool
	}{
		{
			name:           "nil node",
			node:           nil,
			expectForLoops: false,
		},
		{
			name: "node without p-for",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			expectForLoops: false,
		},
		{
			name: "node with p-for",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DirFor:   &ast_domain.Directive{},
			},
			expectForLoops: true,
		},
		{
			name: "child has p-for",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "span",
						DirFor:   &ast_domain.Directive{},
					},
				},
			},
			expectForLoops: true,
		},
		{
			name: "deeply nested p-for",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "ul",
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								TagName:  "li",
								DirFor:   &ast_domain.Directive{},
							},
						},
					},
				},
			},
			expectForLoops: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := ifEmitter.nodeContainsForLoops(tc.node)
			assert.Equal(t, tc.expectForLoops, result, tc.name)
		})
	}
}

func TestNodeContainsDynamicContent(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockAstBuilder := &mockAstBuilder{}
	ie := newIfEmitter(mockEmitter, nil, mockAstBuilder)

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name:     "nil node returns false",
			node:     nil,
			expected: false,
		},
		{
			name: "node with RichText",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				RichText: []ast_domain.TextPart{
					{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "x"}},
				},
			},
			expected: true,
		},
		{
			name: "node with DirText",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				DirText:  &ast_domain.Directive{},
			},
			expected: true,
		},
		{
			name: "node with DirHTML",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				DirHTML:  &ast_domain.Directive{},
			},
			expected: true,
		},
		{
			name: "node with DynamicAttributes",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "src"},
				},
			},
			expected: true,
		},
		{
			name: "node with no dynamic content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			expected: false,
		},
		{
			name: "child has dynamic content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						DirText:  &ast_domain.Directive{},
					},
				},
			},
			expected: true,
		},
		{
			name: "deeply nested dynamic content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						Children: []*ast_domain.TemplateNode{
							{
								NodeType: ast_domain.NodeElement,
								DynamicAttributes: []ast_domain.DynamicAttribute{
									{Name: "href"},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := ie.nodeContainsDynamicContent(tc.node)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEmitNotEqualNil(t *testing.T) {
	t.Parallel()

	expression := cachedIdent("myPtr")
	result := emitNotEqualNil(expression)

	require.NotNil(t, result)
	assert.Equal(t, token.NEQ, result.Op)

	xIdent, ok := result.X.(*goast.Ident)
	require.True(t, ok, "X should be an Ident")
	assert.Equal(t, "myPtr", xIdent.Name)

	yIdent, ok := result.Y.(*goast.Ident)
	require.True(t, ok, "Y should be an Ident")
	assert.Equal(t, "nil", yIdent.Name)
}

func TestEmitLenGreaterThanZero(t *testing.T) {
	t.Parallel()

	expression := cachedIdent("mySlice")
	result := emitLenGreaterThanZero(expression)

	require.NotNil(t, result)
	assert.Equal(t, token.GTR, result.Op)

	callExpr, ok := result.X.(*goast.CallExpr)
	require.True(t, ok, "X should be a CallExpr")

	funIdent, ok := callExpr.Fun.(*goast.Ident)
	require.True(t, ok, "Fun should be an Ident")
	assert.Equal(t, "len", funIdent.Name)

	require.Len(t, callExpr.Args, 1)
	argIdent, ok := callExpr.Args[0].(*goast.Ident)
	require.True(t, ok, "Arg should be an Ident")
	assert.Equal(t, "mySlice", argIdent.Name)

	yLit, ok := result.Y.(*goast.BasicLit)
	require.True(t, ok, "Y should be a BasicLit")
	assert.Equal(t, "0", yLit.Value)
}

func TestEmitTruthinessForCompositeType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		typeExpr   goast.Expr
		name       string
		expectKind string
		expectNil  bool
	}{
		{
			name:       "slice type returns len > 0",
			typeExpr:   &goast.ArrayType{Elt: cachedIdent("int")},
			expectKind: "len_check",
		},
		{
			name:       "fixed array type returns true",
			typeExpr:   &goast.ArrayType{Len: intLit(5), Elt: cachedIdent("int")},
			expectKind: "true_ident",
		},
		{
			name:       "map type returns len > 0",
			typeExpr:   &goast.MapType{Key: cachedIdent("string"), Value: cachedIdent("int")},
			expectKind: "len_check",
		},
		{
			name:       "pointer type returns != nil",
			typeExpr:   &goast.StarExpr{X: cachedIdent("MyStruct")},
			expectKind: "nil_check",
		},
		{
			name:       "interface type returns != nil",
			typeExpr:   &goast.InterfaceType{Methods: &goast.FieldList{}},
			expectKind: "nil_check",
		},
		{
			name:       "func type returns != nil",
			typeExpr:   &goast.FuncType{Params: &goast.FieldList{}},
			expectKind: "nil_check",
		},
		{
			name:       "chan type returns != nil",
			typeExpr:   &goast.ChanType{Value: cachedIdent("int")},
			expectKind: "nil_check",
		},
		{
			name:      "ident type returns nil",
			typeExpr:  cachedIdent("int"),
			expectNil: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			expression := cachedIdent("val")
			result := emitTruthinessForCompositeType(expression, tc.typeExpr)

			if tc.expectNil {
				assert.Nil(t, result)
				return
			}

			require.NotNil(t, result)
			switch tc.expectKind {
			case "len_check":
				binExpr, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "should be BinaryExpr")
				assert.Equal(t, token.GTR, binExpr.Op)
				callExpr, ok := binExpr.X.(*goast.CallExpr)
				require.True(t, ok, "X should be CallExpr")
				funIdent, ok := callExpr.Fun.(*goast.Ident)
				require.True(t, ok)
				assert.Equal(t, "len", funIdent.Name)
			case "nil_check":
				binExpr, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "should be BinaryExpr")
				assert.Equal(t, token.NEQ, binExpr.Op)
				yIdent, ok := binExpr.Y.(*goast.Ident)
				require.True(t, ok)
				assert.Equal(t, "nil", yIdent.Name)
			case "true_ident":
				identifier, ok := result.(*goast.Ident)
				require.True(t, ok, "should be Ident")
				assert.Equal(t, "true", identifier.Name)
			}
		})
	}
}

func TestIsWhitespaceOrComment(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node     *ast_domain.TemplateNode
		name     string
		expected bool
	}{
		{
			name: "comment node",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeComment,
			},
			expected: true,
		},
		{
			name: "whitespace text node",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "   \n  \t  ",
			},
			expected: true,
		},
		{
			name: "empty text node",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "",
			},
			expected: true,
		},
		{
			name: "text node with content",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "Hello",
			},
			expected: false,
		},
		{
			name: "element node",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			},
			expected: false,
		},
		{
			name: "rich text with only whitespace",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: []ast_domain.TextPart{
					{IsLiteral: true, Literal: "   "},
					{IsLiteral: true, Literal: "\n\t"},
				},
			},
			expected: true,
		},
		{
			name: "rich text with content",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: []ast_domain.TextPart{
					{IsLiteral: true, Literal: "  "},
					{IsLiteral: true, Literal: "Hello"},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := isWhitespaceOrComment(tc.node)
			assert.Equal(t, tc.expected, result, tc.name)
		})
	}
}

func TestEmitTruthinessCheck(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		goExpr     goast.Expr
		ann        *ast_domain.GoGeneratorAnnotation
		assertFunc func(t *testing.T, result goast.Expr)
		name       string
	}{
		{
			name:   "nil annotation falls back to runtime truthiness call",
			goExpr: cachedIdent("val"),
			ann:    nil,
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				callExpr := requireCallExpr(t, result, "should be a call expression")
				selectorExpression := requireSelectorExpr(t, callExpr.Fun, "should call pikoruntime.EvaluateTruthiness")
				assert.Equal(t, "EvaluateTruthiness", selectorExpression.Sel.Name)
			},
		},
		{
			name:   "nil ResolvedType falls back to runtime truthiness call",
			goExpr: cachedIdent("val"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: nil,
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				callExpr := requireCallExpr(t, result, "should be a call expression")
				selectorExpression := requireSelectorExpr(t, callExpr.Fun, "should call pikoruntime.EvaluateTruthiness")
				assert.Equal(t, "EvaluateTruthiness", selectorExpression.Sel.Name)
			},
		},
		{
			name:   "nil TypeExpr falls back to runtime truthiness call",
			goExpr: cachedIdent("val"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: nil,
				},
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				callExpr := requireCallExpr(t, result, "should be a call expression")
				selectorExpression := requireSelectorExpr(t, callExpr.Fun, "should call pikoruntime.EvaluateTruthiness")
				assert.Equal(t, "EvaluateTruthiness", selectorExpression.Sel.Name)
			},
		},
		{
			name:   "bool type returns expression directly",
			goExpr: cachedIdent("isActive"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("bool"),
				},
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				identifier := requireIdent(t, result, "bool should return ident directly")
				assert.Equal(t, "isActive", identifier.Name)
			},
		},
		{
			name:   "int type returns != 0 comparison",
			goExpr: cachedIdent("count"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("int"),
				},
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				binExpr, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "should be BinaryExpr")
				assert.Equal(t, token.NEQ, binExpr.Op)
				xIdent := requireIdent(t, binExpr.X, "X should be count ident")
				assert.Equal(t, "count", xIdent.Name)
				yLit := requireBasicLit(t, binExpr.Y, "Y should be 0 literal")
				assert.Equal(t, "0", yLit.Value)
			},
		},
		{
			name:   "float64 type returns != 0 comparison",
			goExpr: cachedIdent("ratio"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("float64"),
				},
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				binExpr, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "should be BinaryExpr")
				assert.Equal(t, token.NEQ, binExpr.Op)
			},
		},
		{
			name:   "string type returns != empty string comparison",
			goExpr: cachedIdent("name"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				binExpr, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "should be BinaryExpr")
				assert.Equal(t, token.NEQ, binExpr.Op)
				xIdent := requireIdent(t, binExpr.X, "X should be name ident")
				assert.Equal(t, "name", xIdent.Name)
				yLit := requireBasicLit(t, binExpr.Y, "Y should be empty string literal")
				assert.Equal(t, `""`, yLit.Value)
			},
		},
		{
			name:   "pointer type returns != nil check",
			goExpr: cachedIdent("myPtr"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.StarExpr{X: cachedIdent("MyStruct")},
				},
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				binExpr, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "should be BinaryExpr")
				assert.Equal(t, token.NEQ, binExpr.Op)
				yIdent := requireIdent(t, binExpr.Y, "Y should be nil")
				assert.Equal(t, "nil", yIdent.Name)
			},
		},
		{
			name:   "unknown ident type falls back to runtime truthiness call",
			goExpr: cachedIdent("unknownVal"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("CustomType"),
				},
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				callExpr := requireCallExpr(t, result, "should be a call expression")
				selectorExpression := requireSelectorExpr(t, callExpr.Fun, "should call pikoruntime.EvaluateTruthiness")
				assert.Equal(t, "EvaluateTruthiness", selectorExpression.Sel.Name)
			},
		},
		{
			name:   "slice type returns len > 0 check",
			goExpr: cachedIdent("items"),
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.ArrayType{Elt: cachedIdent("int")},
				},
			},
			assertFunc: func(t *testing.T, result goast.Expr) {
				t.Helper()
				binExpr, ok := result.(*goast.BinaryExpr)
				require.True(t, ok, "should be BinaryExpr")
				assert.Equal(t, token.GTR, binExpr.Op)
				callExpr := requireCallExpr(t, binExpr.X, "X should be len() call")
				funIdent := requireIdent(t, callExpr.Fun, "Fun should be len ident")
				assert.Equal(t, "len", funIdent.Name)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := emitTruthinessCheck(tc.goExpr, tc.ann)
			require.NotNil(t, result)
			tc.assertFunc(t, result)
		})
	}
}
