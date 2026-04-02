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
	"slices"
	"testing"

	goast "go/ast"
	"go/token"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestEmit_TextNode_StaticContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		textContent  string
		wantEscaped  string
		shouldEscape bool
	}{
		{
			name:         "plain text",
			textContent:  "Hello World",
			wantEscaped:  "Hello World",
			shouldEscape: true,
		},
		{
			name:         "text with special chars",
			textContent:  "Price: $10 & up",
			wantEscaped:  "Price: $10 &amp; up",
			shouldEscape: true,
		},
		{
			name:         "text with HTML entities",
			textContent:  "<script>alert('XSS')</script>",
			wantEscaped:  "&lt;script&gt;alert(&#39;XSS&#39;)&lt;/script&gt;",
			shouldEscape: true,
		},
		{
			name:         "text with quotes",
			textContent:  `He said "Hello"`,
			wantEscaped:  "He said &#34;Hello&#34;",
			shouldEscape: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			mockAttrEmitter := &mockAttributeEmitter{}
			mockExprEmitter := &mockExpressionEmitter{}
			mockAstBld := &mockAstBuilder{}
			ne := newNodeEmitter(mockEmitter, mockExprEmitter, mockAttrEmitter, mockAstBld)

			node := &ast_domain.TemplateNode{
				NodeType:    ast_domain.NodeText,
				TextContent: tc.textContent,
			}

			_, statements, diagnostics := ne.emit(context.Background(), node, "")

			require.NotEmpty(t, statements, "Text node should generate statements")
			assert.Empty(t, diagnostics, "Static text should have no diagnostics")

			var foundAssignment bool
			var assignedValue string
			for _, statement := range statements {
				if assignStmt, ok := statement.(*goast.AssignStmt); ok {
					if len(assignStmt.Lhs) > 0 {
						if selector, ok := assignStmt.Lhs[0].(*goast.SelectorExpr); ok {
							if selector.Sel.Name == "TextContent" {
								foundAssignment = true
								if len(assignStmt.Rhs) > 0 {
									if lit, ok := assignStmt.Rhs[0].(*goast.BasicLit); ok {
										assignedValue = lit.Value
									}
								}
							}
						}
					}
				}
			}

			require.True(t, foundAssignment, "Should assign TextContent")

			if tc.shouldEscape {

				assert.Contains(t, assignedValue, tc.wantEscaped, "Should escape HTML entities")
			}
		})
	}
}

func TestEmit_TextNode_RichText_Escaping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		description    string
		richText       []ast_domain.TextPart
		wantEscapeCall bool
	}{
		{
			name: "XSS attack vector - user input interpolation",
			richText: []ast_domain.TextPart{
				{IsLiteral: true, Literal: "Username: "},
				{
					IsLiteral: false,
					Expression: &ast_domain.Identifier{
						Name: "username",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: new("username"),
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("string"),
							},
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
					},
				},
			},
			wantEscapeCall: true,
			description:    "String interpolation MUST be escaped to prevent XSS",
		},
		{
			name: "numeric interpolation - safe from XSS",
			richText: []ast_domain.TextPart{
				{IsLiteral: true, Literal: "Count: "},
				{
					IsLiteral: false,
					Expression: &ast_domain.Identifier{
						Name: "count",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: new("count"),
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("int"),
							},
						},
					},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("int"),
						},
					},
				},
			},
			wantEscapeCall: true,
			description:    "Even numeric values should be escaped (defensive)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
			mockAttrEmitter := &mockAttributeEmitter{}
			mockAstBld := &mockAstBuilder{}
			ne := newNodeEmitter(mockEmitter, expressionEmitter, mockAttrEmitter, mockAstBld)

			node := &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeText,
				RichText: tc.richText,
			}

			_, statements, diagnostics := ne.emit(context.Background(), node, "")

			require.NotEmpty(t, statements, "Rich text should generate statements")
			assert.Empty(t, diagnostics, "Should have no diagnostics")

			foundEscapeCall := slices.ContainsFunc(statements, containsEscapeStringCall)

			if tc.wantEscapeCall {
				assert.True(t, foundEscapeCall, "SECURITY: %s", tc.description)
			}
		})
	}
}

func TestEmitContentDirectives_PText_AlwaysEscapes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		expression     ast_domain.Expression
		expressionType string
		description    string
	}{
		{
			name: "string expression - MUST escape",
			expression: &ast_domain.Identifier{
				Name: "userInput",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("userInput"),
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
				},
			},
			expressionType: "string",
			description:    "String from p-text MUST be HTML escaped to prevent XSS",
		},
		{
			name: "stringer type - MUST escape",
			expression: &ast_domain.Identifier{
				Name: "obj",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: new("obj"),
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("MyType"),
					},
					Stringability: int(inspector_dto.StringableViaStringer),
				},
			},
			expressionType: "Stringer",
			description:    "String() output from p-text MUST be HTML escaped",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
			mockAttrEmitter := &mockAttributeEmitter{}
			mockAstBld := &mockAstBuilder{}
			ne := newNodeEmitter(mockEmitter, expressionEmitter, mockAttrEmitter, mockAstBld)

			node := &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DirText: &ast_domain.Directive{
					Expression: tc.expression,
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: tc.expression.GetGoAnnotation().ResolvedType,
					},
				},
			}

			_, statements, diagnostics := ne.emit(context.Background(), node, "")

			require.NotEmpty(t, statements, "p-text should generate statements")
			assert.Empty(t, diagnostics)

			foundEscapeCall := slices.ContainsFunc(statements, containsEscapeStringCall)

			assert.True(t, foundEscapeCall, "SECURITY VIOLATION: %s", tc.description)
		})
	}
}

func TestEmitContentDirectives_PHTML_NeverEscapes(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
	mockAttrEmitter := &mockAttributeEmitter{}
	mockAstBld := &mockAstBuilder{}
	ne := newNodeEmitter(mockEmitter, expressionEmitter, mockAttrEmitter, mockAstBld)

	userInput := "userHTML"
	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		DirHTML: &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: "html",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &userInput,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
			},
		},
	}

	_, statements, diagnostics := ne.emit(context.Background(), node, "")

	require.NotEmpty(t, statements, "p-html should generate statements")
	assert.Empty(t, diagnostics)

	foundEscapeCall := slices.ContainsFunc(statements, containsEscapeStringCall)

	assert.False(t, foundEscapeCall, "p-html MUST NOT escape (by design, for trusted HTML)")
}

func TestEmit_ElementNode_BasicProperties(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		tagName  string
		nodeType ast_domain.NodeType
	}{
		{name: "div element", tagName: "div", nodeType: ast_domain.NodeElement},
		{name: "span element", tagName: "span", nodeType: ast_domain.NodeElement},
		{name: "custom element", tagName: "my-component", nodeType: ast_domain.NodeElement},
		{name: "fragment", tagName: "", nodeType: ast_domain.NodeFragment},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			mockAttrEmitter := &mockAttributeEmitter{}
			mockExprEmitter := &mockExpressionEmitter{}
			mockAstBld := &mockAstBuilder{}
			ne := newNodeEmitter(mockEmitter, mockExprEmitter, mockAttrEmitter, mockAstBld)

			node := &ast_domain.TemplateNode{
				NodeType: tc.nodeType,
				TagName:  tc.tagName,
			}

			varName, statements, diagnostics := ne.emit(context.Background(), node, "")

			require.NotEmpty(t, varName, "Should return temp variable name")
			require.NotEmpty(t, statements, "Should generate statements")
			assert.Empty(t, diagnostics)

			foundNodeType := false
			for _, statement := range statements {
				if assignStmt, ok := statement.(*goast.AssignStmt); ok {
					if len(assignStmt.Lhs) > 0 {
						if selector, ok := assignStmt.Lhs[0].(*goast.SelectorExpr); ok {
							if selector.Sel.Name == "NodeType" {
								foundNodeType = true
							}
						}
					}
				}
			}
			assert.True(t, foundNodeType, "Should assign NodeType")

			if tc.tagName != "" {
				foundTagName := false
				for _, statement := range statements {
					if assignStmt, ok := statement.(*goast.AssignStmt); ok {
						if len(assignStmt.Lhs) > 0 {
							if selector, ok := assignStmt.Lhs[0].(*goast.SelectorExpr); ok {
								if selector.Sel.Name == "TagName" {
									foundTagName = true
								}
							}
						}
					}
				}
				assert.True(t, foundTagName, "Should assign TagName")
			}
		})
	}
}

func TestEmit_ElementNode_CollectionInitialisation(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockAttrEmitter := &mockAttributeEmitter{}
	mockExprEmitter := &mockExpressionEmitter{}
	mockAstBld := &mockAstBuilder{}
	ne := newNodeEmitter(mockEmitter, mockExprEmitter, mockAttrEmitter, mockAstBld)

	nodeWithContent := &ast_domain.TemplateNode{
		NodeType:   ast_domain.NodeElement,
		TagName:    "div",
		Attributes: []ast_domain.HTMLAttribute{{Name: "class", Value: "test"}},
		Children:   []*ast_domain.TemplateNode{{NodeType: ast_domain.NodeText}},
	}

	_, statements, diagnostics := ne.emit(context.Background(), nodeWithContent, "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)

	requiredPoolCalls := map[string]bool{
		"GetAttrSlice":  false,
		"GetChildSlice": false,
	}

	forbiddenAllocations := map[string]bool{
		"OnEvents":     false,
		"CustomEvents": false,
	}

	for _, statement := range statements {
		if assignStmt, ok := statement.(*goast.AssignStmt); ok {
			if len(assignStmt.Rhs) > 0 {

				if callExpr, ok := assignStmt.Rhs[0].(*goast.CallExpr); ok {
					if funSelector, ok := callExpr.Fun.(*goast.SelectorExpr); ok {
						functionName := funSelector.Sel.Name
						if _, required := requiredPoolCalls[functionName]; required {
							requiredPoolCalls[functionName] = true
						}
					}
				}

				if len(assignStmt.Lhs) > 0 {
					if selector, ok := assignStmt.Lhs[0].(*goast.SelectorExpr); ok {
						fieldName := selector.Sel.Name
						if callExpr, ok := assignStmt.Rhs[0].(*goast.CallExpr); ok {
							if identifier, ok := callExpr.Fun.(*goast.Ident); ok && identifier.Name == "make" {
								if _, isForbidden := forbiddenAllocations[fieldName]; isForbidden {
									forbiddenAllocations[fieldName] = true
								}
							}
						}
					}
				}
			}
		}
	}

	for functionName, found := range requiredPoolCalls {
		assert.True(t, found, "Should call %s pool function when capacity > 0", functionName)
	}
	for fieldName, found := range forbiddenAllocations {
		assert.False(t, found, "Should NOT allocate %s (performance optimisation)", fieldName)
	}

	ne2 := newNodeEmitter(mockEmitter, mockExprEmitter, mockAttrEmitter, mockAstBld)
	nodeEmpty := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "br",
	}

	_, stmtsEmpty, _ := ne2.emit(context.Background(), nodeEmpty, "")

	for _, statement := range stmtsEmpty {
		if assignStmt, ok := statement.(*goast.AssignStmt); ok {
			if len(assignStmt.Rhs) > 0 {
				if callExpr, ok := assignStmt.Rhs[0].(*goast.CallExpr); ok {

					if funSelector, ok := callExpr.Fun.(*goast.SelectorExpr); ok {
						functionName := funSelector.Sel.Name
						if functionName == "GetAttrSlice" || functionName == "GetChildSlice" {
							t.Errorf("Should NOT call %s when capacity is 0", functionName)
						}
					}

					if len(assignStmt.Lhs) > 0 {
						if selector, ok := assignStmt.Lhs[0].(*goast.SelectorExpr); ok {
							fieldName := selector.Sel.Name
							if identifier, ok := callExpr.Fun.(*goast.Ident); ok && identifier.Name == "make" {
								if fieldName == "Attributes" || fieldName == "Children" {
									t.Errorf("Should NOT allocate %s when capacity is 0", fieldName)
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestEmit_ElementNode_WithChildren(t *testing.T) {
	t.Parallel()

	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	stringConv := newStringConverter()
	binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
	expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)

	mockAstBuilder := &mockAstBuilder{
		emitNodeFunc: func(ctx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {

			return []goast.Stmt{
				&goast.ExprStmt{
					X: &goast.CallExpr{
						Fun: cachedIdent("append"),
					},
				},
			}, 1, nil
		},
	}

	mockAttrEmitter := &mockAttributeEmitter{}
	ne := newNodeEmitter(mockEmitter, expressionEmitter, mockAttrEmitter, mockAstBuilder)

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
		Children: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "Child 1"},
			{NodeType: ast_domain.NodeText, TextContent: "Child 2"},
		},
	}

	_, statements, diagnostics := ne.emit(context.Background(), node, "")

	require.NotEmpty(t, statements)
	assert.Empty(t, diagnostics)

	assert.True(t, len(statements) > 5, "Should have statements from parent and children")
}

func TestEmitMiscDirectives(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		dirField      string
		setupNode     func(*ast_domain.TemplateNode)
		wantFieldName string
	}{

		{
			name:     "p-model directive",
			dirField: "DirModel",
			setupNode: func(node *ast_domain.TemplateNode) {
				formData := "formData"
				node.DirModel = &ast_domain.Directive{
					Expression: &ast_domain.Identifier{
						Name: "model",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							BaseCodeGenVarName: &formData,
						},
					},
				}
			},
			wantFieldName: "DirModel",
		},
		{
			name:     "p-scaffold directive",
			dirField: "DirScaffold",
			setupNode: func(node *ast_domain.TemplateNode) {
				node.DirScaffold = &ast_domain.Directive{
					Expression: &ast_domain.BooleanLiteral{Value: true},
				}
			},
			wantFieldName: "DirScaffold",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
			stringConv := newStringConverter()
			binaryEmitter := newBinaryOpEmitter(mockEmitter, nil)
			expressionEmitter := newExpressionEmitter(mockEmitter, binaryEmitter, stringConv)
			mockAttrEmitter := &mockAttributeEmitter{}
			mockAstBld := &mockAstBuilder{}
			ne := newNodeEmitter(mockEmitter, expressionEmitter, mockAttrEmitter, mockAstBld)

			node := &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
			}
			tc.setupNode(node)

			_, statements, diagnostics := ne.emit(context.Background(), node, "")

			require.NotEmpty(t, statements)
			assert.Empty(t, diagnostics)

			foundAssignment := false
			for _, statement := range statements {
				if assignStmt, ok := statement.(*goast.AssignStmt); ok {
					if len(assignStmt.Lhs) > 0 {
						if selector, ok := assignStmt.Lhs[0].(*goast.SelectorExpr); ok {
							if selector.Sel.Name == tc.wantFieldName {
								foundAssignment = true
								break
							}
						}
					}
				}
			}

			assert.True(t, foundAssignment, "Should assign %s", tc.wantFieldName)
		})
	}
}

func containsEscapeStringCall(statement goast.Stmt) bool {
	found := false
	goast.Inspect(statement, func(n goast.Node) bool {
		if call, ok := n.(*goast.CallExpr); ok {
			if selectorExpression, ok := call.Fun.(*goast.SelectorExpr); ok {

				if pkg, ok := selectorExpression.X.(*goast.Ident); ok {
					if pkg.Name == "html" && selectorExpression.Sel.Name == "EscapeString" {
						found = true
						return false
					}
				}

				if selectorExpression.Sel.Name == "AppendEscapeString" {
					found = true
					return false
				}
			}
		}
		return true
	})
	return found
}

func BenchmarkEmit_TextNode_Static(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockAttrEmitter := &mockAttributeEmitter{}
	mockExprEmitter := &mockExpressionEmitter{}
	mockAstBld := &mockAstBuilder{}
	ne := newNodeEmitter(mockEmitter, mockExprEmitter, mockAttrEmitter, mockAstBld)

	node := &ast_domain.TemplateNode{
		NodeType:    ast_domain.NodeText,
		TextContent: "Hello World",
	}

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ne.emit(ctx, node, "")
	}
}

func BenchmarkEmit_ElementNode_Simple(b *testing.B) {
	mockEmitter := &emitter{config: EmitterConfig{}, ctx: NewEmitterContext()}
	mockAttrEmitter := &mockAttributeEmitter{}
	mockExprEmitter := &mockExpressionEmitter{}
	mockAstBld := &mockAstBuilder{}
	ne := newNodeEmitter(mockEmitter, mockExprEmitter, mockAttrEmitter, mockAstBld)

	node := &ast_domain.TemplateNode{
		NodeType: ast_domain.NodeElement,
		TagName:  "div",
	}

	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _, _ = ne.emit(ctx, node, "")
	}
}

func TestNodeHasPartialAttribute(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		attributes []ast_domain.HTMLAttribute
		want       bool
	}{
		{
			name:       "no attributes",
			attributes: nil,
			want:       false,
		},
		{
			name: "attribute named partial",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "partial", Value: "my-partial"},
			},
			want: true,
		},
		{
			name: "attribute named class only",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
			want: false,
		},
		{
			name: "multiple attributes including partial",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "id", Value: "main"},
				{Name: "partial", Value: "header"},
				{Name: "class", Value: "wrapper"},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			node := &ast_domain.TemplateNode{
				Attributes: tc.attributes,
			}

			got := nodeHasPartialAttribute(node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsDynamicAttributeBoolean(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		da   *ast_domain.DynamicAttribute
		name string
		want bool
	}{
		{
			name: "nil GoAnnotations nil Expression",
			da: &ast_domain.DynamicAttribute{
				Name:          "disabled",
				GoAnnotations: nil,
				Expression:    nil,
			},
			want: false,
		},
		{
			name: "GoAnnotations with bool type",
			da: &ast_domain.DynamicAttribute{
				Name: "disabled",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
			want: true,
		},
		{
			name: "GoAnnotations with string type",
			da: &ast_domain.DynamicAttribute{
				Name: "title",
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
				},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isDynamicAttributeBoolean(tc.da)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsBindDirectiveBoolean(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		directive *ast_domain.Directive
		name      string
		want      bool
	}{
		{
			name:      "nil directive",
			directive: nil,
			want:      false,
		},
		{
			name: "directive with bool type GoAnnotations",
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("bool"),
					},
				},
			},
			want: true,
		},
		{
			name: "directive with string type GoAnnotations",
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
				},
			},
			want: false,
		},
		{
			name: "directive with nil GoAnnotations",
			directive: &ast_domain.Directive{
				GoAnnotations: nil,
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isBindDirectiveBoolean(tc.directive)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsCollectionNillable(t *testing.T) {
	t.Parallel()

	ne := requireNodeEmitter(t, requireEmitter(t))

	testCases := []struct {
		ann  *ast_domain.GoGeneratorAnnotation
		name string
		want bool
	}{
		{
			name: "nil annotation",
			ann:  nil,
			want: true,
		},
		{
			name: "nil ResolvedType",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: nil,
			},
			want: true,
		},
		{
			name: "ArrayType",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.ArrayType{Elt: cachedIdent("string")},
				},
			},
			want: true,
		},
		{
			name: "MapType",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.MapType{
						Key:   cachedIdent("string"),
						Value: cachedIdent("int"),
					},
				},
			},
			want: true,
		},
		{
			name: "Ident string",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
			},
			want: false,
		},
		{
			name: "Ident MyType",
			ann: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("MyType"),
				},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ne.isCollectionNillable(tc.ann)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHasWidthDescriptors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		srcset []ast_domain.ResponsiveVariantMetadata
		want   bool
	}{
		{
			name:   "empty list",
			srcset: []ast_domain.ResponsiveVariantMetadata{},
			want:   true,
		},
		{
			name: "all have width greater than zero",
			srcset: []ast_domain.ResponsiveVariantMetadata{
				{Width: 320},
				{Width: 640},
				{Width: 1024},
			},
			want: true,
		},
		{
			name: "one has width zero",
			srcset: []ast_domain.ResponsiveVariantMetadata{
				{Width: 320},
				{Width: 0},
				{Width: 1024},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := hasWidthDescriptors(tc.srcset)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCreateStaticSliceAssignment(t *testing.T) {
	t.Parallel()

	result := createStaticSliceAssignment(cachedIdent("node"), "Children", "staticChildren_1")

	require.NotNil(t, result, "result should not be nil")
	assert.Equal(t, token.ASSIGN, result.Tok, "token should be ASSIGN")

	require.Len(t, result.Lhs, 1, "LHS should have one expression")
	lhsSel := requireSelectorExpr(t, result.Lhs[0], "LHS")
	lhsX := requireIdent(t, lhsSel.X, "LHS selector X")
	assert.Equal(t, "node", lhsX.Name, "LHS X should be 'node'")
	assert.Equal(t, "Children", lhsSel.Sel.Name, "LHS Sel should be 'Children'")

	require.Len(t, result.Rhs, 1, "RHS should have one expression")
	rhsIdent := requireIdent(t, result.Rhs[0], "RHS")
	assert.Equal(t, "staticChildren_1", rhsIdent.Name, "RHS should be 'staticChildren_1'")
}

func TestFindAttributeValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		searchName string
		want       string
		attributes []ast_domain.HTMLAttribute
	}{
		{
			name: "found attribute",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
				{Name: "id", Value: "main"},
			},
			searchName: "id",
			want:       "main",
		},
		{
			name: "not found",
			attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
			searchName: "id",
			want:       "",
		},
		{
			name:       "empty list",
			attributes: []ast_domain.HTMLAttribute{},
			searchName: "class",
			want:       "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := findAttributeValue(tc.attributes, tc.searchName)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEmitPartialScopeAttribute(t *testing.T) {
	t.Parallel()

	em := requireEmitter(t)
	ne := requireNodeEmitter(t, em)

	t.Run("empty partialScopeID returns nil", func(t *testing.T) {
		t.Parallel()

		result := ne.emitPartialScopeAttribute(cachedIdent("node"), "")
		assert.Nil(t, result, "Empty partialScopeID should return nil")
	})

	t.Run("valid partialScopeID returns non-nil statement", func(t *testing.T) {
		t.Parallel()

		result := ne.emitPartialScopeAttribute(cachedIdent("node"), "abc123")
		require.NotNil(t, result, "Valid partialScopeID should return a statement")

		expressionStatement, ok := result.(*goast.AssignStmt)
		require.True(t, ok, "Expected *goast.AssignStmt, got %T", result)

		require.Len(t, expressionStatement.Rhs, 1)
		callExpr, ok := expressionStatement.Rhs[0].(*goast.CallExpr)
		require.True(t, ok, "Expected RHS to contain *goast.CallExpr, got %T", expressionStatement.Rhs[0])

		funIdent, ok := callExpr.Fun.(*goast.Ident)
		require.True(t, ok, "Expected Fun to be *goast.Ident, got %T", callExpr.Fun)
		assert.Equal(t, "append", funIdent.Name)
	})
}

func TestShouldUseWidthDescriptors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		srcset []ast_domain.ResponsiveVariantMetadata
		want   bool
	}{
		{
			name:   "all variants have width returns true",
			srcset: []ast_domain.ResponsiveVariantMetadata{{Width: 100}, {Width: 200}},
			want:   true,
		},
		{
			name:   "one variant missing width returns false",
			srcset: []ast_domain.ResponsiveVariantMetadata{{Width: 100}, {Width: 0}},
			want:   false,
		},
		{
			name:   "empty slice returns true",
			srcset: []ast_domain.ResponsiveVariantMetadata{},
			want:   true,
		},
		{
			name:   "single variant with width returns true",
			srcset: []ast_domain.ResponsiveVariantMetadata{{Width: 320}},
			want:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := shouldUseWidthDescriptors(tc.srcset)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildSrcsetValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		want                string
		srcset              []ast_domain.ResponsiveVariantMetadata
		useWidthDescriptors bool
	}{
		{
			name: "width descriptors",
			srcset: []ast_domain.ResponsiveVariantMetadata{
				{URL: "/img/sm.jpg", Width: 320, Density: "1x"},
				{URL: "/img/lg.jpg", Width: 640, Density: "2x"},
			},
			useWidthDescriptors: true,
			want:                "/img/sm.jpg 320w, /img/lg.jpg 640w",
		},
		{
			name: "density descriptors",
			srcset: []ast_domain.ResponsiveVariantMetadata{
				{URL: "/img/sm.jpg", Width: 0, Density: "1x"},
				{URL: "/img/lg.jpg", Width: 0, Density: "2x"},
			},
			useWidthDescriptors: false,
			want:                "/img/sm.jpg 1x, /img/lg.jpg 2x",
		},
		{
			name:                "empty srcset",
			srcset:              []ast_domain.ResponsiveVariantMetadata{},
			useWidthDescriptors: true,
			want:                "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildSrcsetValue(tc.srcset, tc.useWidthDescriptors)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestBuildSizesAttributeIfNeeded(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when not using width descriptors", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{}
		result := buildSizesAttributeIfNeeded(node, cachedIdent("attrs"), false)
		assert.Nil(t, result)
	})

	t.Run("returns nil when no sizes attribute present", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "foo"},
			},
		}
		result := buildSizesAttributeIfNeeded(node, cachedIdent("attrs"), true)
		assert.Nil(t, result)
	})

	t.Run("returns append statement when sizes attribute present", func(t *testing.T) {
		t.Parallel()
		node := &ast_domain.TemplateNode{
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "sizes", Value: "(max-width: 600px) 100vw, 50vw"},
			},
		}
		result := buildSizesAttributeIfNeeded(node, cachedIdent("attrs"), true)
		require.NotNil(t, result)

		assignStmt, ok := result.(*goast.AssignStmt)
		require.True(t, ok, "Expected *goast.AssignStmt, got %T", result)
		require.Len(t, assignStmt.Rhs, 1)
	})
}

func TestNodeEmitter_DirectiveHasClientScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		sourcePathHasClientScript map[string]bool
		directive                 *ast_domain.Directive
		node                      *ast_domain.TemplateNode
		name                      string
		want                      bool
	}{
		{
			name:                      "nil map returns false",
			sourcePathHasClientScript: nil,
			directive:                 &ast_domain.Directive{},
			node:                      &ast_domain.TemplateNode{},
			want:                      false,
		},
		{
			name:                      "directive source path in map returns map value",
			sourcePathHasClientScript: map[string]bool{"comp.pk": true},
			directive: &ast_domain.Directive{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("comp.pk"),
				},
			},
			node: &ast_domain.TemplateNode{},
			want: true,
		},
		{
			name:                      "node source path in map returns map value",
			sourcePathHasClientScript: map[string]bool{"node.pk": true},
			directive:                 &ast_domain.Directive{},
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					OriginalSourcePath: new("node.pk"),
				},
			},
			want: true,
		},
		{
			name:                      "neither path in map returns false",
			sourcePathHasClientScript: map[string]bool{"other.pk": true},
			directive:                 &ast_domain.Directive{},
			node:                      &ast_domain.TemplateNode{},
			want:                      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			em := requireEmitter(t)
			em.config.SourcePathHasClientScript = tc.sourcePathHasClientScript
			ne := requireNodeEmitter(t, em)
			got := ne.directiveHasClientScript(tc.directive, tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHasDynamicClassContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node *ast_domain.TemplateNode
		name string
		want bool
	}{
		{
			name: "node with DirClass returns true",
			node: &ast_domain.TemplateNode{
				DirClass: &ast_domain.Directive{},
			},
			want: true,
		},
		{
			name: "node with dynamic class attribute returns true",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "class"},
				},
			},
			want: true,
		},
		{
			name: "node with other dynamic attribute returns false",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "id"},
				},
			},
			want: false,
		},
		{
			name: "empty node returns false",
			node: &ast_domain.TemplateNode{},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := hasDynamicClassContent(tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHasDynamicStyleContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node *ast_domain.TemplateNode
		name string
		want bool
	}{
		{
			name: "node with DirStyle returns true",
			node: &ast_domain.TemplateNode{
				DirStyle: &ast_domain.Directive{},
			},
			want: true,
		},
		{
			name: "node with dynamic style attribute returns true",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "style"},
				},
			},
			want: true,
		},
		{
			name: "node with other dynamic attribute returns false",
			node: &ast_domain.TemplateNode{
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "title"},
				},
			},
			want: false,
		},
		{
			name: "empty node returns false",
			node: &ast_domain.TemplateNode{},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := hasDynamicStyleContent(tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCountNonBooleanDynamicAttributes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		dynAttrs []ast_domain.DynamicAttribute
		want     int
	}{
		{
			name:     "empty list returns zero",
			dynAttrs: nil,
			want:     0,
		},
		{
			name: "class and style are skipped",
			dynAttrs: []ast_domain.DynamicAttribute{
				{Name: "class", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
				{Name: "style", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
			},
			want: 0,
		},
		{
			name: "non-boolean non-class attribute is counted",
			dynAttrs: []ast_domain.DynamicAttribute{
				{Name: "title", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
			},
			want: 1,
		},
		{
			name: "boolean attribute is not counted",
			dynAttrs: []ast_domain.DynamicAttribute{
				{Name: "disabled", GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive)},
			},
			want: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := countNonBooleanDynamicAttributes(tc.dynAttrs, "")
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEmitSrcsetAttributes(t *testing.T) {
	t.Parallel()

	t.Run("width descriptors with sizes attribute produces 2 statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "piko:img",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "sizes", Value: "(max-width: 600px) 100vw, 50vw"},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Srcset: []ast_domain.ResponsiveVariantMetadata{
					{URL: "/img/sm.jpg", Width: 320, Density: "1x"},
					{URL: "/img/lg.jpg", Width: 640, Density: "2x"},
				},
			},
		}

		nodeVar := cachedIdent("node")
		statements := ne.emitSrcsetAttributes(nodeVar, node)

		assert.Len(t, statements, 2, "Width descriptors with sizes should produce srcset + sizes statements")
	})

	t.Run("density descriptors without sizes produces 1 statement", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			NodeType:   ast_domain.NodeElement,
			TagName:    "piko:img",
			Attributes: []ast_domain.HTMLAttribute{},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				Srcset: []ast_domain.ResponsiveVariantMetadata{
					{URL: "/img/sm.jpg", Width: 0, Density: "1x"},
					{URL: "/img/lg.jpg", Width: 0, Density: "2x"},
				},
			},
		}

		nodeVar := cachedIdent("node")
		statements := ne.emitSrcsetAttributes(nodeVar, node)

		assert.Len(t, statements, 1, "Density descriptors without sizes should produce only srcset statement")
	})
}

func TestCalculateChildCapacityExpr(t *testing.T) {
	t.Parallel()

	t.Run("no children returns nil", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{},
		}

		result := ne.calculateChildCapacityExpr(node, map[int]*LoopIterableInfo{})
		assert.Nil(t, result)
	})

	t.Run("2 static children no loops returns intLit(2)", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "child1"},
				{NodeType: ast_domain.NodeText, TextContent: "child2"},
			},
		}

		result := ne.calculateChildCapacityExpr(node, map[int]*LoopIterableInfo{})
		require.NotNil(t, result)

		basicLit, ok := result.(*goast.BasicLit)
		require.True(t, ok, "Expected *goast.BasicLit, got %T", result)
		assert.Equal(t, "2", basicLit.Value)
	})

	t.Run("1 static + 1 loop returns BinaryExpr", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "static"},
				{NodeType: ast_domain.NodeElement, TagName: "div"},
			},
		}

		loopInfoByIndex := map[int]*LoopIterableInfo{
			1: {VarName: "items"},
		}

		result := ne.calculateChildCapacityExpr(node, loopInfoByIndex)
		require.NotNil(t, result)

		binExpr, ok := result.(*goast.BinaryExpr)
		require.True(t, ok, "Expected *goast.BinaryExpr, got %T", result)
		assert.Equal(t, token.ADD, binExpr.Op)

		leftLit, ok := binExpr.X.(*goast.BasicLit)
		require.True(t, ok, "Expected left to be *goast.BasicLit, got %T", binExpr.X)
		assert.Equal(t, "1", leftLit.Value)

		rightCall, ok := binExpr.Y.(*goast.CallExpr)
		require.True(t, ok, "Expected right to be *goast.CallExpr, got %T", binExpr.Y)
		lenIdent, ok := rightCall.Fun.(*goast.Ident)
		require.True(t, ok, "Expected Fun to be *goast.Ident")
		assert.Equal(t, "len", lenIdent.Name)
	})

	t.Run("only loop children starts with len call", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
			},
		}

		loopInfoByIndex := map[int]*LoopIterableInfo{
			0: {VarName: "loopVar1"},
		}

		result := ne.calculateChildCapacityExpr(node, loopInfoByIndex)
		require.NotNil(t, result)

		callExpr, ok := result.(*goast.CallExpr)
		require.True(t, ok, "Expected *goast.CallExpr, got %T", result)
		lenIdent, ok := callExpr.Fun.(*goast.Ident)
		require.True(t, ok, "Expected Fun to be *goast.Ident")
		assert.Equal(t, "len", lenIdent.Name)

		require.Len(t, callExpr.Args, 1)
		argIdent, ok := callExpr.Args[0].(*goast.Ident)
		require.True(t, ok, "Expected argument to be *goast.Ident")
		assert.Equal(t, "loopVar1", argIdent.Name)
	})

	t.Run("multiple loops produce sum of len calls", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
				{NodeType: ast_domain.NodeElement, TagName: "span"},
			},
		}

		loopInfoByIndex := map[int]*LoopIterableInfo{
			0: {VarName: "items"},
			1: {VarName: "users"},
		}

		result := ne.calculateChildCapacityExpr(node, loopInfoByIndex)
		require.NotNil(t, result)

		binExpr, ok := result.(*goast.BinaryExpr)
		require.True(t, ok, "Expected *goast.BinaryExpr, got %T", result)
		assert.Equal(t, token.ADD, binExpr.Op)

		leftCall, ok := binExpr.X.(*goast.CallExpr)
		require.True(t, ok, "Expected left to be *goast.CallExpr, got %T", binExpr.X)
		leftLenIdent, ok := leftCall.Fun.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "len", leftLenIdent.Name)

		rightCall, ok := binExpr.Y.(*goast.CallExpr)
		require.True(t, ok, "Expected right to be *goast.CallExpr, got %T", binExpr.Y)
		rightLenIdent, ok := rightCall.Fun.(*goast.Ident)
		require.True(t, ok)
		assert.Equal(t, "len", rightLenIdent.Name)
	})
}

func TestCountEmittedEventAttributes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		events                    map[string][]ast_domain.Directive
		sourcePathHasClientScript map[string]bool
		name                      string
		want                      int
	}{
		{
			name:   "empty map returns zero",
			events: map[string][]ast_domain.Directive{},
			want:   0,
		},
		{
			name: "action modifier is counted",
			events: map[string][]ast_domain.Directive{
				"click": {
					{Modifier: "action"},
				},
			},
			want: 1,
		},
		{
			name: "helper modifier is counted",
			events: map[string][]ast_domain.Directive{
				"submit": {
					{Modifier: "helper"},
				},
			},
			want: 1,
		},
		{
			name: "empty modifier with client script is counted",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						Modifier: "",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: new("comp.pk"),
						},
					},
				},
			},
			sourcePathHasClientScript: map[string]bool{
				"comp.pk": true,
			},
			want: 1,
		},
		{
			name: "empty modifier without client script is not counted",
			events: map[string][]ast_domain.Directive{
				"click": {
					{
						Modifier: "",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							OriginalSourcePath: new("no-script.pk"),
						},
					},
				},
			},
			sourcePathHasClientScript: map[string]bool{
				"no-script.pk": false,
			},
			want: 0,
		},
		{
			name: "unknown modifier is not counted",
			events: map[string][]ast_domain.Directive{
				"click": {
					{Modifier: "prevent"},
				},
			},
			want: 0,
		},
		{
			name: "multiple directives across events",
			events: map[string][]ast_domain.Directive{
				"click": {
					{Modifier: "action"},
					{Modifier: "helper"},
				},
				"submit": {
					{Modifier: "action"},
				},
			},
			want: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.config.SourcePathHasClientScript = tc.sourcePathHasClientScript
			ne := requireNodeEmitter(t, em)

			node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
			got := ne.countEmittedEventAttributes(tc.events, node)

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWillEmitPartialInfoAttributes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node             *ast_domain.TemplateNode
		annotationResult *annotator_dto.AnnotationResult
		name             string
		want             bool
	}{
		{
			name: "nil GoAnnotations returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
			},
			annotationResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			},
			want: false,
		},
		{
			name: "nil PartialInfo returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: nil,
				},
			},
			annotationResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			},
			want: false,
		},
		{
			name: "existing partial attribute returns false",
			node: &ast_domain.TemplateNode{
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "partial", Value: "existing"},
				},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "pub_hash",
					},
				},
			},
			annotationResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"pub_hash": {IsPublic: true},
					},
				},
			},
			want: false,
		},
		{
			name: "nil AnnotationResult returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "pub_hash",
					},
				},
			},
			annotationResult: nil,
			want:             false,
		},
		{
			name: "nil VirtualModule returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "pub_hash",
					},
				},
			},
			annotationResult: &annotator_dto.AnnotationResult{
				VirtualModule: nil,
			},
			want: false,
		},
		{
			name: "public partial returns true",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "pub_hash",
					},
				},
			},
			annotationResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"pub_hash": {IsPublic: true},
					},
				},
			},
			want: true,
		},
		{
			name: "private partial returns true",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "priv_hash",
					},
				},
			},
			annotationResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
						"priv_hash": {IsPublic: false},
					},
				},
			},
			want: true,
		},
		{
			name: "component not found returns false",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "nonexistent",
					},
				},
			},
			annotationResult: &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
				},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.AnnotationResult = tc.annotationResult
			ne := requireNodeEmitter(t, em)

			got := ne.willEmitPartialInfoAttributes(tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCountBooleanDynamicAttributes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		dynAttrs []ast_domain.DynamicAttribute
		want     int
	}{
		{
			name:     "empty slice returns zero",
			dynAttrs: []ast_domain.DynamicAttribute{},
			want:     0,
		},
		{
			name: "all non-boolean returns zero",
			dynAttrs: []ast_domain.DynamicAttribute{
				{Name: "title", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
				{Name: "href", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
			},
			want: 0,
		},
		{
			name: "all boolean returns count",
			dynAttrs: []ast_domain.DynamicAttribute{
				{Name: "disabled", GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive)},
				{Name: "readonly", GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive)},
				{Name: "checked", GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive)},
			},
			want: 3,
		},
		{
			name: "mixed boolean and non-boolean returns correct count",
			dynAttrs: []ast_domain.DynamicAttribute{
				{Name: "disabled", GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive)},
				{Name: "title", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
				{Name: "checked", GoAnnotations: createMockAnnotation("bool", inspector_dto.StringablePrimitive)},
				{Name: "href", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
			},
			want: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := countBooleanDynamicAttributes(tc.dynAttrs)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCountBooleanBinds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		binds map[string]*ast_domain.Directive
		name  string
		want  int
	}{
		{
			name:  "empty map returns zero",
			binds: map[string]*ast_domain.Directive{},
			want:  0,
		},
		{
			name: "all non-boolean returns zero",
			binds: map[string]*ast_domain.Directive{
				"title": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
					},
				},
				"href": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "mixed returns correct count",
			binds: map[string]*ast_domain.Directive{
				"disabled": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
				"title": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
					},
				},
				"checked": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "nil directive in map returns zero for that entry",
			binds: map[string]*ast_domain.Directive{
				"disabled": nil,
				"checked": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
			},
			want: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := countBooleanBinds(tc.binds)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCountNonBooleanBinds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		binds map[string]*ast_domain.Directive
		name  string
		want  int
	}{
		{
			name:  "empty map returns zero",
			binds: map[string]*ast_domain.Directive{},
			want:  0,
		},
		{
			name: "all boolean returns zero",
			binds: map[string]*ast_domain.Directive{
				"disabled": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
				"checked": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "mixed returns correct count of non-boolean",
			binds: map[string]*ast_domain.Directive{
				"disabled": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("bool"),
						},
					},
				},
				"title": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
					},
				},
				"value": {
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("int"),
						},
					},
				},
			},
			want: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := countNonBooleanBinds(tc.binds)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHasAllStaticAttributes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node *ast_domain.TemplateNode
		name string
		want bool
	}{
		{
			name: "no dynamic attrs no binds no events no key no ref returns true",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				Attributes:        []ast_domain.HTMLAttribute{{Name: "class", Value: "container"}},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
			},
			want: true,
		},
		{
			name: "has boolean dynamic attributes returns false",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "input",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{
						Name: "disabled",
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("bool"),
							},
						},
					},
				},
				Binds:        map[string]*ast_domain.Directive{},
				OnEvents:     map[string][]ast_domain.Directive{},
				CustomEvents: map[string][]ast_domain.Directive{},
			},
			want: false,
		},
		{
			name: "has boolean bind directives returns false",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "input",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds: map[string]*ast_domain.Directive{
					"disabled": {
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("bool"),
							},
						},
					},
				},
				OnEvents:     map[string][]ast_domain.Directive{},
				CustomEvents: map[string][]ast_domain.Directive{},
			},
			want: false,
		},
		{
			name: "has events with client script returns false",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "button",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{
							Modifier: "action",
						},
					},
				},
				CustomEvents: map[string][]ast_domain.Directive{},
			},
			want: false,
		},
		{
			name: "has non-literal DirKey returns false",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				Key:               &ast_domain.Identifier{Name: "dynamicKey"},
			},
			want: false,
		},
		{
			name: "has string literal DirKey returns true",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				Key:               &ast_domain.StringLiteral{Value: "static-key"},
			},
			want: true,
		},
		{
			name: "text node returns false",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeText,
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
			},
			want: false,
		},
		{
			name: "piko:img with srcset returns false",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "piko:img",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					Srcset: []ast_domain.ResponsiveVariantMetadata{
						{URL: "/img/sm.jpg", Width: 320, Density: "1x"},
					},
				},
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			ne := requireNodeEmitter(t, em)

			got := ne.hasAllStaticAttributes(tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNodeEmitter_CalculateAttributeCapacity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node           *ast_domain.TemplateNode
		setupEmitter   func(*emitter)
		name           string
		partialScopeID string
		want           int
	}{
		{
			name: "no attributes returns zero",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "br",
				Attributes:        []ast_domain.HTMLAttribute{},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
			},
			partialScopeID: "",
			want:           0,
		},
		{
			name: "static attrs only counts them",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "class", Value: "container"},
					{Name: "id", Value: "main"},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
			},
			partialScopeID: "",
			want:           2,
		},
		{
			name: "with string literal Key adds one",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				Attributes:        []ast_domain.HTMLAttribute{{Name: "class", Value: "container"}},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				Key:               &ast_domain.StringLiteral{Value: "key-1"},
			},
			partialScopeID: "",
			want:           2,
		},
		{
			name: "with DirRef adds one",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				Attributes:        []ast_domain.HTMLAttribute{},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				DirRef:            &ast_domain.Directive{RawExpression: "myRef"},
			},
			partialScopeID: "",
			want:           1,
		},
		{
			name: "piko:img with srcset adds extra capacity",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "piko:img",
				Attributes:        []ast_domain.HTMLAttribute{},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					Srcset: []ast_domain.ResponsiveVariantMetadata{
						{URL: "/img/sm.jpg", Width: 320, Density: "1x"},
					},
				},
			},
			partialScopeID: "",
			want:           PikoImgSrcsetAttrCount,
		},
		{
			name: "piko:img with srcset and sizes attr adds srcset plus sizes",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "piko:img",
				Attributes: []ast_domain.HTMLAttribute{
					{Name: "sizes", Value: "(max-width: 600px) 100vw, 50vw"},
				},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					Srcset: []ast_domain.ResponsiveVariantMetadata{
						{URL: "/img/sm.jpg", Width: 320, Density: "1x"},
						{URL: "/img/lg.jpg", Width: 640, Density: "2x"},
					},
				},
			},
			partialScopeID: "",
			want:           1 + PikoImgSrcsetAttrCount + PikoImgSizesAttrCount,
		},
		{
			name: "with events adds event count",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "button",
				Attributes:        []ast_domain.HTMLAttribute{},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents: map[string][]ast_domain.Directive{
					"click": {
						{Modifier: "action"},
						{Modifier: "helper"},
					},
				},
				CustomEvents: map[string][]ast_domain.Directive{},
			},
			partialScopeID: "",
			want:           2,
		},
		{
			name: "with partialScopeID on element adds one",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				Attributes:        []ast_domain.HTMLAttribute{},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
			},
			partialScopeID: "scope123",
			want:           SingleDirectiveAttrCount,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			if tc.setupEmitter != nil {
				tc.setupEmitter(em)
			}
			ne := requireNodeEmitter(t, em)

			got := ne.calculateAttributeCapacity(tc.node, tc.partialScopeID)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCalculateAttributeWriterCapacity(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		node *ast_domain.TemplateNode
		name string
		want int
	}{
		{
			name: "no dynamic content returns zero",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				Attributes:        []ast_domain.HTMLAttribute{{Name: "class", Value: "container"}},
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
			},
			want: 0,
		},
		{
			name: "non-boolean dynamic attrs are counted",
			node: &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{
					{Name: "title", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
					{Name: "href", GoAnnotations: createMockAnnotation("string", inspector_dto.StringablePrimitive)},
				},
				Binds:        map[string]*ast_domain.Directive{},
				OnEvents:     map[string][]ast_domain.Directive{},
				CustomEvents: map[string][]ast_domain.Directive{},
			},
			want: 2,
		},
		{
			name: "non-boolean binds are counted",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds: map[string]*ast_domain.Directive{
					"title": {
						GoAnnotations: &ast_domain.GoGeneratorAnnotation{
							ResolvedType: &ast_domain.ResolvedTypeInfo{
								TypeExpression: cachedIdent("string"),
							},
						},
					},
				},
				OnEvents:     map[string][]ast_domain.Directive{},
				CustomEvents: map[string][]ast_domain.Directive{},
			},
			want: 1,
		},
		{
			name: "with non-literal DirKey adds one",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				Key:               &ast_domain.Identifier{Name: "dynamicKey"},
			},
			want: 1,
		},
		{
			name: "with literal DirKey does not add writer",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				Key:               &ast_domain.StringLiteral{Value: "static-key"},
			},
			want: 0,
		},
		{
			name: "with DirClass adds one",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				DirClass:          &ast_domain.Directive{},
			},
			want: 1,
		},
		{
			name: "with DirStyle adds one",
			node: &ast_domain.TemplateNode{
				NodeType:          ast_domain.NodeElement,
				TagName:           "div",
				DynamicAttributes: []ast_domain.DynamicAttribute{},
				Binds:             map[string]*ast_domain.Directive{},
				OnEvents:          map[string][]ast_domain.Directive{},
				CustomEvents:      map[string][]ast_domain.Directive{},
				DirStyle:          &ast_domain.Directive{},
			},
			want: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			ne := requireNodeEmitter(t, em)

			got := ne.calculateAttributeWriterCapacity(tc.node)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGetChildScopeID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		node          *ast_domain.TemplateNode
		parentScopeID string
		componentHash map[string]*annotator_dto.VirtualComponent
		want          string
	}{
		{
			name: "nil GoAnnotations returns parentScopeID",
			node: &ast_domain.TemplateNode{
				GoAnnotations: nil,
			},
			parentScopeID: "parent",
			componentHash: map[string]*annotator_dto.VirtualComponent{},
			want:          "parent",
		},
		{
			name: "nil PartialInfo returns parentScopeID",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: nil,
				},
			},
			parentScopeID: "parent",
			componentHash: map[string]*annotator_dto.VirtualComponent{},
			want:          "parent",
		},
		{
			name: "component not found returns parentScopeID",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "nonexistent",
					},
				},
			},
			parentScopeID: "parent",
			componentHash: map[string]*annotator_dto.VirtualComponent{},
			want:          "parent",
		},
		{
			name: "found component with different scope combines child and parent",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "child_hash",
					},
				},
			},
			parentScopeID: "parent_scope",
			componentHash: map[string]*annotator_dto.VirtualComponent{
				"child_hash": {HashedName: "child_scope"},
			},
			want: "child_scope parent_scope",
		},
		{
			name: "found component with same scope as parent returns child scope only",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "same_hash",
					},
				},
			},
			parentScopeID: "same_scope",
			componentHash: map[string]*annotator_dto.VirtualComponent{
				"same_hash": {HashedName: "same_scope"},
			},
			want: "same_scope",
		},
		{
			name: "empty parentScopeID returns child scope only",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "child_hash",
					},
				},
			},
			parentScopeID: "",
			componentHash: map[string]*annotator_dto.VirtualComponent{
				"child_hash": {HashedName: "child_scope"},
			},
			want: "child_scope",
		},
		{
			name: "parent already contains child scope returns child scope only",
			node: &ast_domain.TemplateNode{
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					PartialInfo: &ast_domain.PartialInvocationInfo{
						PartialPackageName: "child_hash",
					},
				},
			},
			parentScopeID: "child_scope other_scope",
			componentHash: map[string]*annotator_dto.VirtualComponent{
				"child_hash": {HashedName: "child_scope"},
			},
			want: "child_scope",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			em := requireEmitter(t)
			em.AnnotationResult = &annotator_dto.AnnotationResult{
				VirtualModule: &annotator_dto.VirtualModule{
					ComponentsByHash: tc.componentHash,
				},
			}
			ne := requireNodeEmitter(t, em)

			got := ne.getChildScopeID(tc.node, tc.parentScopeID)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestEmitPartialInfoAttributes(t *testing.T) {
	t.Parallel()

	t.Run("nil GoAnnotations returns nil", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.GoAnnotations = nil

		statements := ne.emitPartialInfoAttributes(cachedIdent("tmpNode"), node)
		assert.Nil(t, statements)
	})

	t.Run("nil PartialInfo returns nil", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			PartialInfo: nil,
		}

		statements := ne.emitPartialInfoAttributes(cachedIdent("tmpNode"), node)
		assert.Nil(t, statements)
	})

	t.Run("existing partial attribute returns nil", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"pub_hash": {
						IsPublic:    true,
						HashedName:  "pub_hash",
						PartialName: "my-partial",
						PartialSrc:  "/_piko/partial/my-partial",
					},
				},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialPackageName: "pub_hash",
			},
		}
		node.Attributes = []ast_domain.HTMLAttribute{
			{Name: "partial", Value: "already-set"},
		}

		statements := ne.emitPartialInfoAttributes(cachedIdent("tmpNode"), node)
		assert.Nil(t, statements)
	})

	t.Run("private partial produces partial and partial_name statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"priv_hash": {
						IsPublic:    false,
						HashedName:  "priv_hash",
						PartialName: "my-private-partial",
					},
				},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialPackageName: "priv_hash",
			},
		}

		statements := ne.emitPartialInfoAttributes(cachedIdent("tmpNode"), node)
		require.NotNil(t, statements, "Expected non-nil statements for private partial")
		assert.Len(t, statements, 2, "Expected 2 statements: partial, partial_name (no partial_src)")
	})

	t.Run("component not found returns nil", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialPackageName: "nonexistent",
			},
		}

		statements := ne.emitPartialInfoAttributes(cachedIdent("tmpNode"), node)
		assert.Nil(t, statements)
	})

	t.Run("public partial produces partial and partial_name and partial_src statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{
					"pub_hash": {
						IsPublic:    true,
						HashedName:  "pub_hash",
						PartialName: "my-partial",
						PartialSrc:  "/_piko/partial/my-partial",
					},
				},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := createMockTemplateNode(ast_domain.NodeElement, "div", "")
		node.GoAnnotations = &ast_domain.GoGeneratorAnnotation{
			PartialInfo: &ast_domain.PartialInvocationInfo{
				PartialPackageName: "pub_hash",
			},
		}

		statements := ne.emitPartialInfoAttributes(cachedIdent("tmpNode"), node)
		require.NotNil(t, statements, "Expected non-nil statements for public partial")
		assert.Len(t, statements, 3, "Expected 3 statements: partial, partial_name, partial_src")
	})
}

func TestEmitPTextDirective(t *testing.T) {
	t.Parallel()

	t.Run("non-nillable type produces direct statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		textVarName := "myText"
		directive := &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: textVarName,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &textVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
					Stringability: int(inspector_dto.StringablePrimitive),
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
				Stringability: int(inspector_dto.StringablePrimitive),
			},
		}

		handled, statements, diagnostics := ne.emitPTextDirective(cachedIdent("tmpNode"), directive)
		assert.True(t, handled, "Directive should be handled")
		assert.NotEmpty(t, statements, "Expected statements for text writer")
		assert.Empty(t, diagnostics, "Expected no diagnostics")

		for _, statement := range statements {
			_, isIf := statement.(*goast.IfStmt)
			assert.False(t, isIf, "Non-nillable type should not produce an IfStmt")
		}
	})

	t.Run("nillable type produces nil-guarded statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		textVarName := "myText"
		directive := &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: textVarName,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &textVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.StarExpr{X: cachedIdent("string")},
					},
					Stringability: int(inspector_dto.StringablePrimitive),
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.StarExpr{X: cachedIdent("string")},
				},
				Stringability: int(inspector_dto.StringablePrimitive),
			},
		}

		handled, statements, diagnostics := ne.emitPTextDirective(cachedIdent("tmpNode"), directive)
		assert.True(t, handled, "Directive should be handled")
		assert.NotEmpty(t, statements, "Expected statements for nil-guarded text writer")
		assert.Empty(t, diagnostics, "Expected no diagnostics")

		lastStmt := statements[len(statements)-1]
		ifStmt, isIf := lastStmt.(*goast.IfStmt)
		require.True(t, isIf, "Nillable type should produce a nil-guard IfStmt")

		binExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
		require.True(t, ok, "Condition should be a BinaryExpr")
		assert.Equal(t, token.NEQ, binExpr.Op, "Condition should use != operator")
	})
}

func TestEmitPHTMLDirective(t *testing.T) {
	t.Parallel()

	t.Run("non-nillable type produces assignment statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		htmlVarName := "myHTML"
		directive := &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: htmlVarName,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &htmlVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: cachedIdent("string"),
					},
					Stringability: int(inspector_dto.StringablePrimitive),
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: cachedIdent("string"),
				},
				Stringability: int(inspector_dto.StringablePrimitive),
			},
		}

		handled, statements, diagnostics := ne.emitPHTMLDirective(cachedIdent("tmpNode"), directive)
		assert.True(t, handled, "Directive should be handled")
		assert.NotEmpty(t, statements, "Expected assignment statements")
		assert.Empty(t, diagnostics, "Expected no diagnostics")

		lastStmt := statements[len(statements)-1]
		_, isIf := lastStmt.(*goast.IfStmt)
		assert.False(t, isIf, "Non-nillable type should not produce an IfStmt")

		assignStmt, isAssign := lastStmt.(*goast.AssignStmt)
		require.True(t, isAssign, "Expected an AssignStmt for InnerHTML")
		assert.Equal(t, token.ASSIGN, assignStmt.Tok)
	})

	t.Run("nillable type produces nil-guarded statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		htmlVarName := "myHTML"
		directive := &ast_domain.Directive{
			Expression: &ast_domain.Identifier{
				Name: htmlVarName,
				GoAnnotations: &ast_domain.GoGeneratorAnnotation{
					BaseCodeGenVarName: &htmlVarName,
					ResolvedType: &ast_domain.ResolvedTypeInfo{
						TypeExpression: &goast.StarExpr{X: cachedIdent("string")},
					},
					Stringability: int(inspector_dto.StringablePrimitive),
				},
			},
			GoAnnotations: &ast_domain.GoGeneratorAnnotation{
				ResolvedType: &ast_domain.ResolvedTypeInfo{
					TypeExpression: &goast.StarExpr{X: cachedIdent("string")},
				},
				Stringability: int(inspector_dto.StringablePrimitive),
			},
		}

		handled, statements, diagnostics := ne.emitPHTMLDirective(cachedIdent("tmpNode"), directive)
		assert.True(t, handled, "Directive should be handled")
		assert.NotEmpty(t, statements, "Expected statements for nil-guarded HTML")
		assert.Empty(t, diagnostics, "Expected no diagnostics")

		lastStmt := statements[len(statements)-1]
		ifStmt, isIf := lastStmt.(*goast.IfStmt)
		require.True(t, isIf, "Nillable type should produce a nil-guard IfStmt")

		binExpr, ok := ifStmt.Cond.(*goast.BinaryExpr)
		require.True(t, ok, "Condition should be a BinaryExpr")
		assert.Equal(t, token.NEQ, binExpr.Op, "Condition should use != operator")
	})
}

func TestEmitCollectionInitialisers(t *testing.T) {
	t.Parallel()

	t.Run("empty node with no attrs or children returns minimal statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "br",
		}

		nodeVar := cachedIdent("tmpNode")
		statements, usedStaticAttrs := ne.emitCollectionInitialisers(nodeVar, node, nil, "")

		assert.Empty(t, statements, "Empty node should produce no collection init statements")
		assert.False(t, usedStaticAttrs, "Empty node should not use static attrs")
	})

	t.Run("node with static attributes produces attribute statements", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "test"},
				{Name: "id", Value: "my-div"},
			},
		}

		nodeVar := cachedIdent("tmpNode")
		statements, usedStaticAttrs := ne.emitCollectionInitialisers(nodeVar, node, nil, "")

		require.NotEmpty(t, statements, "Should produce statements for attribute allocation")

		assert.True(t, usedStaticAttrs, "All-static attributes with static emitter should use static attrs")

		foundAttrAssignment := false
		for _, statement := range statements {
			if assignStmt, ok := statement.(*goast.AssignStmt); ok {
				if len(assignStmt.Lhs) > 0 {
					if selectorExpression, ok := assignStmt.Lhs[0].(*goast.SelectorExpr); ok {
						if selectorExpression.Sel.Name == "Attributes" {
							foundAttrAssignment = true
						}
					}
				}
			}
		}
		assert.True(t, foundAttrAssignment, "Should assign to Attributes field")
	})

	t.Run("node with dynamic attributes allocates writer slice", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			DynamicAttributes: []ast_domain.DynamicAttribute{
				{
					Name:       "title",
					Expression: &ast_domain.Identifier{Name: "titleVal"},
					GoAnnotations: &ast_domain.GoGeneratorAnnotation{
						ResolvedType: &ast_domain.ResolvedTypeInfo{
							TypeExpression: cachedIdent("string"),
						},
					},
				},
			},
		}

		nodeVar := cachedIdent("tmpNode")
		statements, _ := ne.emitCollectionInitialisers(nodeVar, node, nil, "")

		require.NotEmpty(t, statements, "Should produce statements for attribute writer allocation")

		foundWriterPool := false
		for _, statement := range statements {
			if assignStmt, ok := statement.(*goast.AssignStmt); ok {
				if len(assignStmt.Rhs) > 0 {
					if callExpr, ok := assignStmt.Rhs[0].(*goast.CallExpr); ok {
						if selectorExpression, ok := callExpr.Fun.(*goast.SelectorExpr); ok {
							if selectorExpression.Sel.Name == "GetAttrWriterSlice" {
								foundWriterPool = true
							}
						}
					}
				}
			}
		}
		assert.True(t, foundWriterPool, "Should call GetAttrWriterSlice for node with dynamic attributes")
	})

	t.Run("node with children allocates children slice", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Children: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "Hello"},
				{NodeType: ast_domain.NodeText, TextContent: "World"},
			},
		}

		nodeVar := cachedIdent("tmpNode")
		statements, _ := ne.emitCollectionInitialisers(nodeVar, node, nil, "")

		require.NotEmpty(t, statements, "Should produce statements for children allocation")

		foundChildPool := false
		for _, statement := range statements {
			if assignStmt, ok := statement.(*goast.AssignStmt); ok {
				if len(assignStmt.Rhs) > 0 {
					if callExpr, ok := assignStmt.Rhs[0].(*goast.CallExpr); ok {
						if selectorExpression, ok := callExpr.Fun.(*goast.SelectorExpr); ok {
							if selectorExpression.Sel.Name == "GetChildSlice" {
								foundChildPool = true
							}
						}
					}
				}
			}
		}
		assert.True(t, foundChildPool, "Should call GetChildSlice for node with children")
	})

	t.Run("static emitter registers static attrs when all attributes are static", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}

		se := newStaticEmitter(em, "scope1")
		em.staticEmitter = se
		ne := requireNodeEmitter(t, em)

		node := &ast_domain.TemplateNode{
			NodeType: ast_domain.NodeElement,
			TagName:  "div",
			Attributes: []ast_domain.HTMLAttribute{
				{Name: "class", Value: "container"},
			},
		}

		nodeVar := cachedIdent("tmpNode")
		statements, usedStaticAttrs := ne.emitCollectionInitialisers(nodeVar, node, nil, "scope1")

		assert.True(t, usedStaticAttrs, "Should flag that static attrs were used")
		require.NotEmpty(t, statements, "Should produce static assignment statements")

		assignStmt, ok := statements[0].(*goast.AssignStmt)
		require.True(t, ok, "First statement should be AssignStmt")
		assert.Equal(t, token.ASSIGN, assignStmt.Tok)
	})
}

func TestExtractLoopIterables(t *testing.T) {
	t.Parallel()

	t.Run("no children with for directives returns empty result", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		children := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "hello"},
			{NodeType: ast_domain.NodeElement, TagName: "span"},
		}

		extractStmts, loopInfoByIndex, diagnostics := ne.extractLoopIterables(children)

		assert.Empty(t, extractStmts, "Should produce no extract statements")
		assert.Empty(t, loopInfoByIndex, "Should produce no loop info entries")
		assert.Empty(t, diagnostics, "Should produce no diagnostics")
	})

	t.Run("child with non-ForInExpr directive is skipped", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}
		ne := requireNodeEmitter(t, em)

		children := []*ast_domain.TemplateNode{
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "div",
				DirFor: &ast_domain.Directive{
					Expression: &ast_domain.Identifier{Name: "notAForInExpr"},
				},
			},
		}

		extractStmts, loopInfoByIndex, diagnostics := ne.extractLoopIterables(children)

		assert.Empty(t, extractStmts, "Should produce no extract statements for non-ForInExpr")
		assert.Empty(t, loopInfoByIndex, "Should produce no loop info for non-ForInExpr")
		assert.Empty(t, diagnostics)
	})

	t.Run("child with DirFor having ForInExpr extracts iterable", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}

		mockExprEmitter := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("pageData_Items"), nil, nil
			},
		}

		ne := newNodeEmitter(em, mockExprEmitter, &mockAttributeEmitter{}, &mockAstBuilder{})

		collectionAnn := &ast_domain.GoGeneratorAnnotation{
			ResolvedType: &ast_domain.ResolvedTypeInfo{
				TypeExpression: &goast.ArrayType{Elt: cachedIdent("string")},
			},
		}

		children := []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeText, TextContent: "before"},
			{
				NodeType: ast_domain.NodeElement,
				TagName:  "li",
				DirFor: &ast_domain.Directive{
					Expression: &ast_domain.ForInExpression{
						IndexVariable: &ast_domain.Identifier{Name: "i"},
						ItemVariable:  &ast_domain.Identifier{Name: "item"},
						Collection: &ast_domain.Identifier{
							Name:          "items",
							GoAnnotations: collectionAnn,
						},
					},
				},
			},
		}

		extractStmts, loopInfoByIndex, diagnostics := ne.extractLoopIterables(children)

		assert.NotEmpty(t, extractStmts, "Should produce assignment statements for extracted iterable")
		assert.Empty(t, diagnostics, "Should produce no diagnostics")

		require.Len(t, loopInfoByIndex, 1, "Should have one loop info entry")
		loopInfo, exists := loopInfoByIndex[1]
		require.True(t, exists, "Loop info should be at index 1 (second child)")

		assert.NotEmpty(t, loopInfo.VarName, "VarName should be set")
		assert.NotNil(t, loopInfo.CollectionExpression, "CollectionExpr should be set")
		assert.True(t, loopInfo.IsNillable, "Slice type should be nillable")
	})

	t.Run("multiple for-loop children each get separate loop info", func(t *testing.T) {
		t.Parallel()

		em := requireEmitter(t)
		em.ctx = NewEmitterContext()
		em.AnnotationResult = &annotator_dto.AnnotationResult{
			VirtualModule: &annotator_dto.VirtualModule{
				ComponentsByHash: map[string]*annotator_dto.VirtualComponent{},
			},
		}

		mockExprEmitter := &mockExpressionEmitter{
			emitFunc: func(expression ast_domain.Expression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
				return cachedIdent("mockCollection"), nil, nil
			},
		}

		ne := newNodeEmitter(em, mockExprEmitter, &mockAttributeEmitter{}, &mockAstBuilder{})

		makeForChild := func(collectionName string) *ast_domain.TemplateNode {
			return &ast_domain.TemplateNode{
				NodeType: ast_domain.NodeElement,
				TagName:  "li",
				DirFor: &ast_domain.Directive{
					Expression: &ast_domain.ForInExpression{
						ItemVariable: &ast_domain.Identifier{Name: "item"},
						Collection: &ast_domain.Identifier{
							Name: collectionName,
							GoAnnotations: &ast_domain.GoGeneratorAnnotation{
								ResolvedType: &ast_domain.ResolvedTypeInfo{
									TypeExpression: &goast.ArrayType{Elt: cachedIdent("int")},
								},
							},
						},
					},
				},
			}
		}

		children := []*ast_domain.TemplateNode{
			makeForChild("items1"),
			{NodeType: ast_domain.NodeText, TextContent: "between"},
			makeForChild("items2"),
		}

		_, loopInfoByIndex, diagnostics := ne.extractLoopIterables(children)

		assert.Empty(t, diagnostics)
		assert.Len(t, loopInfoByIndex, 2, "Should have two loop info entries")

		_, hasIdx0 := loopInfoByIndex[0]
		_, hasIdx2 := loopInfoByIndex[2]
		assert.True(t, hasIdx0, "Should have loop info for child at index 0")
		assert.True(t, hasIdx2, "Should have loop info for child at index 2")

		assert.NotEqual(t, loopInfoByIndex[0].VarName, loopInfoByIndex[2].VarName,
			"Each loop iterable should have a unique variable name")
	})
}
