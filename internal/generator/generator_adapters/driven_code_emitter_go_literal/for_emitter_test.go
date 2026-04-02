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

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

func TestEmit_BasicRangeLoop(t *testing.T) {
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

	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	collectionVar := "items"
	idxVar := "i"
	itemVar := "item"

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirFor: &ast_domain.Directive{
			Expression: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: idxVar},
				ItemVariable:  &ast_domain.Identifier{Name: itemVar},
				Collection: &ast_domain.Identifier{
					Name: "items",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &collectionVar,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: &goast.ArrayType{
								Elt: cachedIdent("string"),
							},
						},
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/file.pp"),
			},
		},
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Hello"},
		},
	}

	parentSlice := cachedIdent("nodes")
	statements, diagnostics := forEmitter.emit(context.Background(), node, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)

	foundRange := false
	var checkStmts func([]goast.Stmt) bool
	checkStmts = func(statements []goast.Stmt) bool {
		for _, statement := range statements {
			if rangeStmt, ok := statement.(*goast.RangeStmt); ok {
				foundRange = true
				assert.Equal(t, token.DEFINE, rangeStmt.Tok)
				assert.NotNil(t, rangeStmt.Key)
				assert.NotNil(t, rangeStmt.Value)
				assert.NotNil(t, rangeStmt.Body)
				return true
			}

			if ifStmt, ok := statement.(*goast.IfStmt); ok {
				if ifStmt.Body != nil && checkStmts(ifStmt.Body.List) {
					return true
				}
			}
		}
		return false
	}
	checkStmts(statements)
	assert.True(t, foundRange, "Should generate a range statement")
}

func TestEmit_UnusedLoopVars(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{
		emitNodeFunc: func(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {

			return []goast.Stmt{
				&goast.ExprStmt{X: &goast.BasicLit{Kind: token.STRING, Value: "\"static\""}},
			}, 1, nil
		},
	}

	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	collectionVar := "items"
	idxVar := "i"
	itemVar := "item"

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirFor: &ast_domain.Directive{
			Expression: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: idxVar},
				ItemVariable:  &ast_domain.Identifier{Name: itemVar},
				Collection: &ast_domain.Identifier{
					Name: "items",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &collectionVar,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: &goast.ArrayType{
								Elt: cachedIdent("string"),
							},
						},
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/file.pp"),
			},
		},
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Static"},
		},
	}

	parentSlice := cachedIdent("nodes")
	statements, _ := forEmitter.emit(context.Background(), node, parentSlice, "", "")

	for _, statement := range statements {
		if rangeStmt, ok := statement.(*goast.RangeStmt); ok {
			keyIdent := requireIdent(t, rangeStmt.Key, "range statement key")
			valueIdent := requireIdent(t, rangeStmt.Value, "range statement value")

			assert.Equal(t, "_", keyIdent.Name, "Unused index should be _")
			assert.Equal(t, "_", valueIdent.Name, "Unused item should be _")
			break
		}
	}
}

func TestEmit_NilCheckForSlice(t *testing.T) {
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

	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	collectionVar := "items"
	idxVar := "i"
	itemVar := "item"

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirFor: &ast_domain.Directive{
			Expression: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: idxVar},
				ItemVariable:  &ast_domain.Identifier{Name: itemVar},
				Collection: &ast_domain.Identifier{
					Name: "items",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &collectionVar,
						ResolvedType: &ast_domain.ResolvedTypeInfo{

							TypeExpression: &goast.StarExpr{
								X: &goast.ArrayType{
									Elt: cachedIdent("string"),
								},
							},
						},
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/file.pp"),
			},
		},
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Hello"},
		},
	}

	parentSlice := cachedIdent("nodes")
	statements, diagnostics := forEmitter.emit(context.Background(), node, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)

	foundNilCheck := false
	for _, statement := range statements {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {

			if binExpr, ok := ifStmt.Cond.(*goast.BinaryExpr); ok {
				if binExpr.Op == token.NEQ {
					foundNilCheck = true

					assert.NotNil(t, ifStmt.Body)
					break
				}
			}
		}
	}
	assert.True(t, foundNilCheck, "Pointer type collection should have nil check")
}

func TestEmit_MapIteration(t *testing.T) {
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

	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	mapVar := "data"
	keyVar := "key"
	valueVar := "value"

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirFor: &ast_domain.Directive{
			Expression: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: keyVar},
				ItemVariable:  &ast_domain.Identifier{Name: valueVar},
				Collection: &ast_domain.Identifier{
					Name: "data",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &mapVar,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: &goast.MapType{
								Key:   cachedIdent("string"),
								Value: cachedIdent("int"),
							},
						},
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/file.pp"),
			},
		},
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Value"},
		},
	}

	parentSlice := cachedIdent("nodes")
	statements, diagnostics := forEmitter.emit(context.Background(), node, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)

	foundIfStmt := false
	for _, statement := range statements {
		if ifStmt, ok := statement.(*goast.IfStmt); ok {
			foundIfStmt = true
			assert.NotNil(t, ifStmt.Cond, "Map loop should have nil check")
			assert.NotNil(t, ifStmt.Body, "Map loop should have body")
			break
		}
	}
	assert.True(t, foundIfStmt, "Map iteration should generate if statement wrapper")
}

func TestEmit_LoopWithConditional(t *testing.T) {
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

	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	collectionVar := "items"
	conditionVar := "isVisible"
	idxVar := "i"
	itemVar := "item"

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirFor: &ast_domain.Directive{
			Expression: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: idxVar},
				ItemVariable:  &ast_domain.Identifier{Name: itemVar},
				Collection: &ast_domain.Identifier{
					Name: "items",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &collectionVar,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: &goast.ArrayType{
								Elt: cachedIdent("string"),
							},
						},
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/file.pp"),
			},
		},
		DirIf: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: "isVisible",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &conditionVar,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
		},
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Hello"},
		},
	}

	parentSlice := cachedIdent("nodes")
	statements, diagnostics := forEmitter.emit(context.Background(), node, parentSlice, "", "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)

	foundRange := false
	var checkForRange func([]goast.Stmt) *goast.RangeStmt
	checkForRange = func(statements []goast.Stmt) *goast.RangeStmt {
		for _, statement := range statements {
			if rangeStmt, ok := statement.(*goast.RangeStmt); ok {
				return rangeStmt
			}

			if ifStmt, ok := statement.(*goast.IfStmt); ok {
				if ifStmt.Body != nil {
					if rangeStmt := checkForRange(ifStmt.Body.List); rangeStmt != nil {
						return rangeStmt
					}
				}
			}
		}
		return nil
	}

	rangeStmt := checkForRange(statements)
	if rangeStmt != nil {
		foundRange = true

		if rangeStmt.Body != nil && len(rangeStmt.Body.List) > 0 {
			hasIfInBody := false
			for _, bodyStmt := range rangeStmt.Body.List {
				if _, ok := bodyStmt.(*goast.IfStmt); ok {
					hasIfInBody = true
					break
				}
			}
			assert.True(t, hasIfInBody, "Loop body should contain if statement for p-if directive")
		}
	}
	assert.True(t, foundRange, "Should generate range statement")
}

func TestEmit_InvalidExpression(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{}
	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	varName := "items"
	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirFor: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: "items",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &varName,
				},
			},
			RawExpression: "items",
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/file.pp"),
			},
		},
	}

	parentSlice := cachedIdent("nodes")
	statements, diagnostics := forEmitter.emit(context.Background(), node, parentSlice, "", "")

	assert.Empty(t, statements, "Should not generate statements for invalid expression")
	require.NotEmpty(t, diagnostics, "Should have diagnostic for invalid expression")
	assert.Equal(t, ast_domain.Error, diagnostics[0].Severity)
	assert.Contains(t, diagnostics[0].Message, "ForInExpr")
}

func TestIsInvocationLoopDependentRecursive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		invocation       *annotator_dto.PartialInvocation
		invocationsByKey map[string]*annotator_dto.PartialInvocation
		name             string
		want             bool
	}{
		{
			name: "loop-independent (all static props)",
			invocation: &annotator_dto.PartialInvocation{
				InvocationKey: "key1",
				PassedProps: map[string]ast_domain.PropValue{
					"title": {
						Expression:      &ast_domain.StringLiteral{Value: "Static Title"},
						IsLoopDependent: false,
					},
					"count": {
						Expression:      &ast_domain.IntegerLiteral{Value: 42},
						IsLoopDependent: false,
					},
				},
			},
			invocationsByKey: map[string]*annotator_dto.PartialInvocation{},
			want:             false,
		},
		{
			name: "loop-dependent (one prop depends on loop var)",
			invocation: &annotator_dto.PartialInvocation{
				InvocationKey: "key2",
				PassedProps: map[string]ast_domain.PropValue{
					"title": {
						Expression:      &ast_domain.StringLiteral{Value: "Static"},
						IsLoopDependent: false,
					},
					"itemName": {
						Expression:      &ast_domain.Identifier{Name: "item"},
						IsLoopDependent: true,
					},
				},
			},
			invocationsByKey: map[string]*annotator_dto.PartialInvocation{},
			want:             true,
		},
		{
			name: "loop-dependent (multiple loop-dependent props)",
			invocation: &annotator_dto.PartialInvocation{
				InvocationKey: "key3",
				PassedProps: map[string]ast_domain.PropValue{
					"index": {
						Expression:      &ast_domain.Identifier{Name: "i"},
						IsLoopDependent: true,
					},
					"value": {
						Expression:      &ast_domain.Identifier{Name: "item"},
						IsLoopDependent: true,
					},
				},
			},
			invocationsByKey: map[string]*annotator_dto.PartialInvocation{},
			want:             true,
		},
		{
			name: "empty props (loop-independent)",
			invocation: &annotator_dto.PartialInvocation{
				InvocationKey: "key4",
				PassedProps:   map[string]ast_domain.PropValue{},
			},
			invocationsByKey: map[string]*annotator_dto.PartialInvocation{},
			want:             false,
		},
		{
			name: "loop-dependent via dependency chain",
			invocation: &annotator_dto.PartialInvocation{
				InvocationKey: "child_key",
				PassedProps: map[string]ast_domain.PropValue{
					"prop": {
						Expression:      &ast_domain.Identifier{Name: "state"},
						IsLoopDependent: false,
					},
				},
				DependsOn: []string{"parent_key"},
			},
			invocationsByKey: map[string]*annotator_dto.PartialInvocation{
				"parent_key": {
					InvocationKey: "parent_key",
					PassedProps: map[string]ast_domain.PropValue{
						"item_id": {
							Expression:      &ast_domain.Identifier{Name: "item"},
							IsLoopDependent: true,
						},
					},
				},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			visited := make(map[string]bool)
			got := isInvocationLoopDependentRecursive(tc.invocation, tc.invocationsByKey, visited)

			assert.Equal(t, tc.want, got, "Loop dependency detection for %s", tc.name)
		})
	}
}

func TestCollectPartialInvocationsInLoop(t *testing.T) {
	t.Parallel()

	invocationKey1 := "inv_key_1"
	invocationKey2 := "inv_key_2"

	mockEmitter := &emitter{
		config: EmitterConfig{},
		ctx:    NewEmitterContext(),
		AnnotationResult: &annotator_dto.AnnotationResult{
			UniqueInvocations: []*annotator_dto.PartialInvocation{
				{
					InvocationKey: invocationKey1,
					PartialAlias:  "Button",
				},
				{
					InvocationKey: invocationKey2,
					PartialAlias:  "Card",
				},
			},
		},
	}

	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
	mockAstBuilder := &mockAstBuilder{}
	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "button",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: invocationKey1,
					},
				},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: invocationKey2,
					},
				},
			},
		},
	}

	invocations := forEmitter.collectPartialInvocationsInLoop(node)

	assert.Len(t, invocations, 2, "Should collect both partial invocations")
	assert.Equal(t, invocationKey1, invocations[0].InvocationKey)
	assert.Equal(t, invocationKey2, invocations[1].InvocationKey)
}

func TestCollectPartialInvocations_SkipsNestedLoops(t *testing.T) {
	t.Parallel()

	invocationKey1 := "outer_inv"
	invocationKey2 := "nested_inv"

	mockEmitter := &emitter{
		config: EmitterConfig{},
		ctx:    NewEmitterContext(),
		AnnotationResult: &annotator_dto.AnnotationResult{
			UniqueInvocations: []*annotator_dto.PartialInvocation{
				{InvocationKey: invocationKey1},
				{InvocationKey: invocationKey2},
			},
		},
	}

	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
	mockAstBuilder := &mockAstBuilder{}
	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "span",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						InvocationKey: invocationKey1,
					},
				},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "ul",
				DirFor:   &ast_domain.Directive{},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "li",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey: invocationKey2,
							},
						},
					},
				},
			},
		},
	}

	invocations := forEmitter.collectPartialInvocationsInLoop(node)

	assert.Len(t, invocations, 1, "Should skip partials inside nested loops")
	assert.Equal(t, invocationKey1, invocations[0].InvocationKey)
}

func TestCollectPartialInvocations_SiblingAfterNestedLoop(t *testing.T) {
	t.Parallel()

	invocationKeyBeforeLoop := "before_loop_inv"
	invocationKeyNestedLoop := "nested_inv"
	invocationKeyAfterLoop := "after_loop_inv"

	mockEmitter := &emitter{
		config: EmitterConfig{},
		ctx:    NewEmitterContext(),
		AnnotationResult: &annotator_dto.AnnotationResult{
			UniqueInvocations: []*annotator_dto.PartialInvocation{
				{InvocationKey: invocationKeyBeforeLoop},
				{InvocationKey: invocationKeyNestedLoop},
				{InvocationKey: invocationKeyAfterLoop},
			},
		},
	}

	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
	mockAstBuilder := &mockAstBuilder{}
	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "tr",
		Children: []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "td",
				DirFor:   &ast_domain.Directive{},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "span",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey: invocationKeyNestedLoop,
							},
						},
					},
				},
			},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "td",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "div",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							PartialInfo: &ast_domain.PartialInvocationInfo{
								InvocationKey: invocationKeyAfterLoop,
							},
						},
					},
				},
			},
		},
	}

	invocations := forEmitter.collectPartialInvocationsInLoop(node)

	assert.Len(t, invocations, 1, "Should collect sibling partial but skip nested loop partial")
	assert.Equal(t, invocationKeyAfterLoop, invocations[0].InvocationKey)
}

func TestCanHoistLoopBody(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
	mockAstBuilder := &mockAstBuilder{}
	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	testCases := []struct {
		node      *ast_domain.TemplateNode
		forExpr   *ast_domain.ForInExpression
		name      string
		reason    string
		wantHoist bool
	}{
		{
			name: "structurally static, no loop var dependency (CAN HOIST)",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeElement,
				TagName:     "div",
				TextContent: "Static text",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					IsStructurallyStatic: true,
				},
				Children: []*ast_domain.TemplateNode{},
			},
			forExpr: &ast_domain.ForInExpression{
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
				IndexVariable: &ast_domain.Identifier{Name: "i"},
			},
			wantHoist: true,
			reason:    "Static content with no loop dependencies should be hoisted",
		},
		{
			name: "not structurally static (CANNOT HOIST)",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeElement,
				TagName:     "div",
				TextContent: "Dynamic",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					IsStructurallyStatic: false,
				},
			},
			forExpr: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
			},
			wantHoist: false,
			reason:    "Non-static nodes cannot be hoisted",
		},
		{
			name: "depends on loop variable (CANNOT HOIST)",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					IsStructurallyStatic: true,
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name: "data-index",
						Expression: &ast_domain.Identifier{
							Name: "i",
						},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
				IndexVariable: &ast_domain.Identifier{Name: "i"},
			},
			wantHoist: false,
			reason:    "Nodes depending on loop variables cannot be hoisted",
		},
		{
			name: "contains nested for loop (CANNOT HOIST)",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "ul",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					IsStructurallyStatic: true,
				},
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "li",
						DirFor:   &ast_domain.Directive{},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
			},
			wantHoist: false,
			reason:    "Nodes containing nested loops cannot be hoisted",
		},
		{
			name: "no GoAnnotations (CANNOT HOIST)",
			node: &ast_domain.TemplateNode{
				NodeType:      ast_domain.NodeElement,
				TagName:       "div",
				GoAnnotations: nil,
			},
			forExpr: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
			},
			wantHoist: false,
			reason:    "Nodes without annotations cannot be hoisted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := forEmitter.canHoistLoopBody(tc.node, tc.forExpr)

			assert.Equal(t, tc.wantHoist, got, tc.reason)
		})
	}
}

func TestSubtreeDependsOnLoopVars(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node        *ast_domain.TemplateNode
		forExpr     *ast_domain.ForInExpression
		name        string
		reason      string
		wantDepends bool
	}{
		{
			name: "no loop variable usage",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: "Static text",
			},
			forExpr: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
			},
			wantDepends: false,
			reason:      "Static text has no dependencies",
		},
		{
			name: "direct loop variable in attribute",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name: "data-value",
						Expression: &ast_domain.Identifier{
							Name: "item",
						},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				ItemVariable: &ast_domain.Identifier{Name: "item"},
			},
			wantDepends: true,
			reason:      "Attributes using loop variables create dependencies",
		},
		{
			name: "loop variable in nested child attribute",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Children: []*ast_domain.TemplateNode{
					{
						NodeType: ast_domain.NodeElement,
						TagName:  "span",
						DynamicAttributes: []ast_domain.DynamicAttribute{
							{
								Name: "data-index",
								Expression: &ast_domain.Identifier{
									Name: "i",
								},
							},
						},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
				IndexVariable: &ast_domain.Identifier{Name: "i"},
			},
			wantDepends: true,
			reason:      "Nested children using loop variables create dependencies",
		},
		{
			name: "no loop variables defined",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name: "id",
						Expression: &ast_domain.Identifier{
							Name: "item",
						},
					},
				},
			},
			forExpr:     &ast_domain.ForInExpression{},
			wantDepends: false,
			reason:      "If no loop variables are defined, nothing can depend on them",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := subtreeDependsOnLoopVars(tc.node, tc.forExpr)

			assert.Equal(t, tc.wantDepends, got, tc.reason)
		})
	}
}

func BenchmarkEmit_BasicLoop(b *testing.B) {
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

	forEmitter := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

	collectionVar := "items"
	idxVar := "i"
	itemVar := "item"

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirFor: &ast_domain.Directive{
			Expression: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: idxVar},
				ItemVariable:  &ast_domain.Identifier{Name: itemVar},
				Collection: &ast_domain.Identifier{
					Name: "items",
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						BaseCodeGenVarName: &collectionVar,
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: &goast.ArrayType{
								Elt: cachedIdent("string"),
							},
						},
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				OriginalSourcePath: new("/test/file.pp"),
			},
		},
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Hello"},
		},
	}

	parentSlice := cachedIdent("nodes")
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = forEmitter.emit(ctx, node, parentSlice, "", "")
	}
}

func TestVisitEventsMap(t *testing.T) {
	t.Parallel()

	t.Run("empty map calls visitor zero times", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		visitEventsMap(map[string][]ast_domain.Directive{}, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("single event with non-nil expression calls visitor once", func(t *testing.T) {
		t.Parallel()

		var collected []ast_domain.Expression
		eventsMap := map[string][]ast_domain.Directive{
			"click": {
				{
					Expression: &ast_domain.Identifier{Name: "handleClick"},
				},
			},
		}

		visitEventsMap(eventsMap, func(expression ast_domain.Expression) {
			collected = append(collected, expression)
		})

		require.Len(t, collected, 1)
		identifier, ok := collected[0].(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "handleClick", identifier.Name)
	})

	t.Run("multiple events with some nil expressions", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		eventsMap := map[string][]ast_domain.Directive{
			"click": {
				{Expression: &ast_domain.Identifier{Name: "handler1"}},
				{Expression: nil},
			},
			"submit": {
				{Expression: &ast_domain.Identifier{Name: "handler2"}},
			},
		}

		visitEventsMap(eventsMap, func(_ ast_domain.Expression) {
			callCount++
		})

		assert.Equal(t, 2, callCount, "Should visit only non-nil expressions")
	})
}

func TestVisitBindExpressions(t *testing.T) {
	t.Parallel()

	t.Run("empty binds calls visitor zero times", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{},
		}

		callCount := 0
		visitBindExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount)
	})

	t.Run("single non-nil bind calls visitor once", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"title": {
					Expression: &ast_domain.Identifier{Name: "titleVar"},
				},
			},
		}

		var collected []ast_domain.Expression
		visitBindExpressions(node, func(expression ast_domain.Expression) {
			collected = append(collected, expression)
		})

		require.Len(t, collected, 1)
		identifier, ok := collected[0].(*ast_domain.Identifier)
		require.True(t, ok)
		assert.Equal(t, "titleVar", identifier.Name)
	})

	t.Run("nil directive is skipped", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"title": nil,
				"href": {
					Expression: &ast_domain.Identifier{Name: "hrefVar"},
				},
			},
		}

		callCount := 0
		visitBindExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 1, callCount, "Should skip nil directive")
	})

	t.Run("nil expression inside directive is skipped", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			Binds: map[string]*ast_domain.Directive{
				"title": {
					Expression: nil,
				},
			},
		}

		callCount := 0
		visitBindExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount, "Should skip directive with nil expression")
	})
}

func TestVisitDynamicContentExpressions(t *testing.T) {
	t.Parallel()

	t.Run("empty node calls visitor zero times", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{},
			RichText:          []ast_domain.TextPart{},
		}

		callCount := 0
		visitDynamicContentExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 0, callCount, "Empty node should not call visitor")
	})

	t.Run("dynamic attributes with non-nil expressions are visited", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", Expression: &ast_domain.Identifier{Name: "titleVal"}},
				{Name: "href", Expression: &ast_domain.Identifier{Name: "hrefVal"}},
				{Name: "empty", Expression: nil},
			},
			RichText: []ast_domain.TextPart{},
		}

		callCount := 0
		visitDynamicContentExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 2, callCount, "Should visit only non-nil dynamic attribute expressions")
	})

	t.Run("rich text parts with non-nil non-literal expressions are visited", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{},
			RichText: []ast_domain.TextPart{
				{IsLiteral: true, Expression: &ast_domain.StringLiteral{Value: "literal"}},
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "dynamicText"}},
				{IsLiteral: false, Expression: nil},
			},
		}

		callCount := 0
		visitDynamicContentExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 1, callCount, "Should visit only non-literal, non-nil rich text expressions")
	})

	t.Run("mixed dynamic attributes and rich text are all visited", func(t *testing.T) {
		t.Parallel()

		node := &ast_domain.TemplateNode{
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{Name: "title", Expression: &ast_domain.Identifier{Name: "titleVal"}},
			},
			RichText: []ast_domain.TextPart{
				{IsLiteral: false, Expression: &ast_domain.Identifier{Name: "dynamicText"}},
			},
		}

		callCount := 0
		visitDynamicContentExpressions(node, func(_ ast_domain.Expression) {
			callCount++
		})
		assert.Equal(t, 2, callCount, "Should visit both dynamic attributes and rich text expressions")
	})
}

func TestDetermineLoopVarNames(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		node        *ast_domain.TemplateNode
		forExpr     *ast_domain.ForInExpression
		wantIdxVar  string
		wantItemVar string
	}{
		{
			name: "both vars unused become blank identifiers",
			node: &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeElement,
				TagName:     "div",
				TextContent: "Static text",
			},
			forExpr: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: "i"},
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
			},
			wantIdxVar:  "_",
			wantItemVar: "_",
		},
		{
			name: "index variable used in dynamic attribute",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "data-index",
						Expression: &ast_domain.Identifier{Name: "i"},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: "i"},
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
			},
			wantIdxVar:  "i",
			wantItemVar: "_",
		},
		{
			name: "item variable used in dynamic attribute",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "data-value",
						Expression: &ast_domain.Identifier{Name: "item"},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: "i"},
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
			},
			wantIdxVar:  "_",
			wantItemVar: "item",
		},
		{
			name: "both vars used in subtree",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "data-index",
						Expression: &ast_domain.Identifier{Name: "i"},
					},
					{
						Name:       "data-value",
						Expression: &ast_domain.Identifier{Name: "item"},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: "i"},
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
			},
			wantIdxVar:  "i",
			wantItemVar: "item",
		},
		{
			name: "nil index variable produces blank identifier",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "data-value",
						Expression: &ast_domain.Identifier{Name: "item"},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				IndexVariable: nil,
				ItemVariable:  &ast_domain.Identifier{Name: "item"},
			},
			wantIdxVar:  "_",
			wantItemVar: "item",
		},
		{
			name: "nil item variable produces blank identifier",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name:       "data-index",
						Expression: &ast_domain.Identifier{Name: "i"},
					},
				},
			},
			forExpr: &ast_domain.ForInExpression{
				IndexVariable: &ast_domain.Identifier{Name: "i"},
				ItemVariable:  nil,
			},
			wantIdxVar:  "i",
			wantItemVar: "_",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
			mockAstBuilder := &mockAstBuilder{}
			fe := newForEmitter(mockEmitter, expressionEmitter, mockAstBuilder)

			idxVar, itemVar := fe.determineLoopVarNames(tc.node, tc.forExpr)

			assert.Equal(t, tc.wantIdxVar, idxVar, "index variable name")
			assert.Equal(t, tc.wantItemVar, itemVar, "item variable name")
		})
	}
}
